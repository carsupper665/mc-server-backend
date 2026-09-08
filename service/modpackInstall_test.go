package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"go-backend/common"
	"go-backend/model"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type packTransport func(*http.Request) (*http.Response, error)

func (f packTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func setupPackInstall(t *testing.T, transport packTransport) {
	t.Helper()
	oldDB, oldPath, oldClient, oldTransport := model.DB, common.MinecraftServerPath, http.DefaultClient, http.DefaultTransport
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	model.DB = db
	if err = db.AutoMigrate(&model.UserMinecraftServer{}, &model.Mod{}, &model.ModVersion{}, &model.ServerMod{}); err != nil {
		t.Fatal(err)
	}
	common.MinecraftServerPath = t.TempDir()
	http.DefaultTransport = transport
	http.DefaultClient = &http.Client{Transport: transport}
	modpackSessions.items = map[string]*ModpackSession{}
	modpackSessions.stopping = false
	t.Cleanup(func() {
		if err := ShutdownModpackSessions(); err != nil {
			t.Error(err)
		}
		model.DB, common.MinecraftServerPath, http.DefaultClient, http.DefaultTransport = oldDB, oldPath, oldClient, oldTransport
		modpackSessions.items = map[string]*ModpackSession{}
		modpackSessions.stopping = false
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})
}

func packResponse(req *http.Request, code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: req}
}
func standardPackTransport(req *http.Request) (*http.Response, error) {
	switch req.URL.Hostname() {
	case "meta.fabricmc.net":
		return packResponse(req, 200, "server"), nil
	case "cdn.modrinth.com":
		if strings.HasSuffix(req.URL.Path, "/bad") {
			return packResponse(req, 200, "bad hash"), nil
		}
		return packResponse(req, 200, "mod"), nil
	case "api.modrinth.com":
		return packResponse(req, 200, `{"id":"ver1","project_id":"mod1","version_number":"1","name":"test","files":[{"filename":"test.jar","url":"https://cdn.modrinth.com/mod","hashes":{}}],"game_versions":["1.21.6"],"loaders":["fabric"]}`), nil
	}
	return nil, fmt.Errorf("unexpected request: %s", req.URL)
}
func startPack(t *testing.T, p MRPack, overrides ...string) *ModpackSession {
	t.Helper()
	data := packBytes(t, p, overrides...)
	file, err := os.CreateTemp(t.TempDir(), "pack-*.mrpack")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.Write(data); err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseMRPack(file, int64(len(data)), nil)
	if err != nil {
		t.Fatal(err)
	}
	s, err := StartModpackInstall(7, parsed, file)
	if err != nil {
		file.Close()
		t.Fatal(err)
	}
	return s
}
func waitPack(t *testing.T, s *ModpackSession) {
	t.Helper()
	select {
	case <-s.done:
	case <-time.After(5 * time.Second):
		t.Fatal("installation did not finish")
	}
}

func TestModpackInstallAndArchive(t *testing.T) {
	setupPackInstall(t, standardPackTransport)
	p := testPack()
	p.Files[0].Downloads = []string{"https://cdn.modrinth.com/bad", "https://cdn.modrinth.com/mod"}
	s := startPack(t, p, "server-overrides/config/test.txt", "overrides/config/test.txt", "client-overrides/client.txt")
	waitPack(t, s)
	if s.Status != JobCompleted || s.Progress != 100 || s.Error != "" {
		t.Fatalf("failed install: %+v", s.InstallJobSnapshot)
	}
	server, err := model.GetServerByID(7, s.ServerID)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(server.SystemPath, "config/test.txt"))
	if err != nil || string(data) != "server-overrides/config/test.txt" {
		t.Fatalf("wrong override: %s %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(server.SystemPath, "client.txt")); !os.IsNotExist(err) {
		t.Fatal("client override was installed")
	}
	var mod model.ServerMod
	if err := model.DB.First(&mod).Error; err != nil || mod.VersionID != "ver1" || mod.AutoUpdate {
		t.Fatalf("missing pinned mod record: %+v %v", mod, err)
	}
	if _, err := GetModpackSession(s.ID, 8); err == nil {
		t.Fatal("other owner can read session")
	}
	raw, err := GetModpackSession(s.ID, 7)
	if err != nil || !bytes.Contains(raw, []byte("retrying")) {
		t.Fatal("missing complete history")
	}
	_, history, unsub, err := SubscribeInstallEvents(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	unsub()
	if len(history) == 0 || history[len(history)-1].Stage != "completed" {
		t.Fatal("missing terminal SSE event")
	}
	s.EndedAt = time.Now().Add(-2 * time.Hour)
	if err := PruneModpackSessions(); err != nil {
		t.Fatal(err)
	}
	if modpackSessions.items[s.ID] != nil {
		t.Fatal("expired session retained")
	}
	saved, err := GetModpackSession(s.ID, 7)
	if err != nil {
		t.Fatal(err)
	}
	var restored ModpackSession
	if err = json.Unmarshal(saved, &restored); err != nil || restored.Status != JobCompleted || len(restored.Details) != len(s.Details) {
		t.Fatalf("history not persisted: %v", err)
	}
}

func TestModpackDownloadFailure(t *testing.T) {
	setupPackInstall(t, standardPackTransport)
	p := testPack()
	p.Files[0].Downloads = []string{"https://cdn.modrinth.com/bad"}
	s := startPack(t, p)
	waitPack(t, s)
	if s.Status != JobFailed || !strings.Contains(s.Error, "size mismatch") {
		t.Fatalf("wrong failure: %+v", s.InstallJobSnapshot)
	}
	server, _ := model.GetServerByID(7, s.ServerID)
	if _, err := os.Stat(filepath.Join(server.SystemPath, "mods/test.jar")); !os.IsNotExist(err) {
		t.Fatal("failed download left final file")
	}
	if server.InstallStatus != "failed" || !strings.Contains(server.InstallDetails, "test.jar") {
		t.Fatal("failure details not saved")
	}
}

func TestModpackShutdownCancelsAndSaves(t *testing.T) {
	started := make(chan struct{})
	setupPackInstall(t, func(req *http.Request) (*http.Response, error) {
		close(started)
		<-req.Context().Done()
		return nil, req.Context().Err()
	})
	s := startPack(t, testPack())
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("download not started")
	}
	if err := ShutdownModpackSessions(); err != nil {
		t.Fatal(err)
	}
	if s.Status != "interrupted" {
		t.Fatalf("shutdown status: %s", s.Status)
	}
	server, err := model.GetServerByID(7, s.ServerID)
	if err != nil {
		t.Fatal(err)
	}
	if server.InstallStatus != "interrupted" || !strings.Contains(server.InstallDetails, "backend shutdown") {
		t.Fatal("interruption not recorded")
	}
	if _, err := StartModpackInstall(7, &MRPack{Dependencies: map[string]string{"minecraft": "1.21.6", "fabric-loader": "0.18.1"}}, nil); err == nil {
		t.Fatal("accepted session after shutdown")
	}
}

func TestModpackRetainsAllDetailsAndEmbeddedMods(t *testing.T) {
	setupPackInstall(t, standardPackTransport)
	p := testPack()
	p.Files = nil
	for i := 0; i < 110; i++ {
		p.Files = append(p.Files, testPackFile(fmt.Sprintf("config/file-%d.txt", i), ""))
	}
	s := startPack(t, p, "overrides/mods/test.jar")
	waitPack(t, s)
	if s.Status != JobCompleted {
		t.Fatalf("install: %s", s.Error)
	}
	if len(s.Details) <= installEventBufferSize {
		t.Fatal("test did not exceed SSE buffer")
	}
	_, history, unsub, err := SubscribeInstallEvents(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	unsub()
	if len(history) != installEventBufferSize {
		t.Fatal("SSE history is not bounded")
	}
	var mod model.ServerMod
	if err := model.DB.Where("server_id = ?", s.ServerID).First(&mod).Error; err != nil || mod.Filename != "test.jar" {
		t.Fatalf("embedded mod not recorded: %v", err)
	}
	server, _ := model.GetServerByID(7, s.ServerID)
	var saved ModpackSession
	if err := json.Unmarshal([]byte(server.InstallDetails), &saved); err != nil || len(saved.Details) != len(s.Details) {
		t.Fatal("full details were truncated")
	}
}
