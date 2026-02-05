package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go-backend/common"
	"go-backend/model"
	"go-backend/router"
)

func setupTestRouter(t *testing.T) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.GlobalApiRateLimitNum = 10
	common.GlobalApiRateLimitDuration = 60

	common.SQLitePath = fmt.Sprintf("file:mc_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := model.InitSqliteDB(false)
	if err != nil {
		t.Fatalf("init sqlite: %v", err)
	}
	model.DB = db
	if err := model.DB.AutoMigrate(
		&model.BlockedIP{},
		&model.LoginAttempt{},
		&model.UserMinecraftServer{},
		&model.ServerMod{},
		&model.Mod{},
		&model.ModVersion{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	r := gin.New()
	router.SetAPIRouter(r)
	return r
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type mockModrinthVersion struct {
	ID            string             `json:"id"`
	VersionNumber string             `json:"version_number"`
	Files         []mockModrinthFile `json:"files"`
	GameVersions  []string           `json:"game_versions"`
	Loaders       []string           `json:"loaders"`
}

type mockModrinthFile struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Primary  bool   `json:"primary"`
}

func setupMockModrinth(t *testing.T, versionID, versionNumber, filename, loader, gameVersion string) func() {
	t.Helper()

	fileURL := "https://mocked.download/" + filename
	payload := []mockModrinthVersion{
		{
			ID:            versionID,
			VersionNumber: versionNumber,
			Files: []mockModrinthFile{
				{
					URL:      fileURL,
					Filename: filename,
					Primary:  true,
				},
			},
			GameVersions: []string{gameVersion},
			Loaders:      []string{loader},
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal modrinth payload: %v", err)
	}

	origTransport := http.DefaultTransport
	http.DefaultTransport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Host {
		case "api.modrinth.com":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewReader(data)),
				Request:    req,
			}, nil
		case "mocked.download":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/java-archive"}},
				Body:       io.NopCloser(bytes.NewReader([]byte("jar-bytes"))),
				Request:    req,
			}, nil
		default:
			return nil, fmt.Errorf("unexpected request: %s", req.URL.String())
		}
	})

	return func() {
		http.DefaultTransport = origTransport
	}
}

func insertServer(t *testing.T, ownerID uint, serverID, systemPath, modLoader, mcVersion string) {
	t.Helper()

	server := model.UserMinecraftServer{
		OwnerID:     ownerID,
		ServerID:    serverID,
		DisplayName: "test-server",
		MCVersion:   mcVersion,
		ModLoader:   modLoader,
		SystemPath:  systemPath,
	}
	if err := model.DB.Create(&server).Error; err != nil {
		t.Fatalf("insert server: %v", err)
	}
}

func TestAddModRouteRequiresAuth(t *testing.T) {
	r := setupTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/server/mod/add/server1", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.RemoteAddr = "127.0.0.1:12345"

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d: %s", http.StatusUnauthorized, w.Code, w.Body.String())
	}
}

func TestAddModRouteInvalidBody(t *testing.T) {
	r := setupTestRouter(t)

	common.CryptoSecret = "test-secret"
	token, err := common.GenerateJWTToken(map[string]interface{}{
		"user_id":  "1",
		"Login_IP": "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("generate jwt: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/server/mod/add/server1", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.RemoteAddr = "127.0.0.1:12345"
	req.AddCookie(&http.Cookie{Name: common.JwtCookieName, Value: token})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Invalid request") {
		t.Fatalf("expected invalid request error, got: %s", w.Body.String())
	}
}

func TestAddModRouteSuccess(t *testing.T) {
	r := setupTestRouter(t)
	restore := setupMockModrinth(t, "ver-123", "1.2.3", "mod.jar", "fabric", "1.20.1")
	t.Cleanup(restore)

	common.CryptoSecret = "test-secret"
	token, err := common.GenerateJWTToken(map[string]interface{}{
		"user_id":  "1",
		"Login_IP": "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("generate jwt: %v", err)
	}

	workDir := t.TempDir()
	insertServer(t, 1, "server1", workDir, "Fabric", "1.20.1")

	body := `{"mod_id":"sodium","version_id":"ver-123","auto_update":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/server/mod/add/server1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.RemoteAddr = "127.0.0.1:12345"
	req.AddCookie(&http.Cookie{Name: common.JwtCookieName, Value: token})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var serverMod model.ServerMod
	if err := model.DB.Where("server_id = ? AND mod_id = ?", "server1", "sodium").First(&serverMod).Error; err != nil {
		t.Fatalf("expected server mod record: %v", err)
	}
	if serverMod.VersionID != "ver-123" {
		t.Fatalf("expected version_id ver-123, got %s", serverMod.VersionID)
	}
	if serverMod.Filename != "mod.jar" {
		t.Fatalf("expected filename mod.jar, got %s", serverMod.Filename)
	}
	if !serverMod.AutoUpdate {
		t.Fatalf("expected auto_update true")
	}

	jarPath := filepath.Join(workDir, "mods", "mod.jar")
	if _, err := os.Stat(jarPath); err != nil {
		t.Fatalf("expected mod file at %s: %v", jarPath, err)
	}
}

func TestAddModSameModDifferentServers(t *testing.T) {
	r := setupTestRouter(t)
	restore := setupMockModrinth(t, "ver-123", "1.2.3", "mod.jar", "fabric", "1.20.1")
	t.Cleanup(restore)

	common.CryptoSecret = "test-secret"
	token, err := common.GenerateJWTToken(map[string]interface{}{
		"user_id":  "1",
		"Login_IP": "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("generate jwt: %v", err)
	}

	workDir1 := t.TempDir()
	workDir2 := t.TempDir()
	insertServer(t, 1, "server1", workDir1, "Fabric", "1.20.1")
	insertServer(t, 1, "server2", workDir2, "Fabric", "1.20.1")

	body := `{"mod_id":"sodium","version_id":"ver-123","auto_update":false}`
	for _, serverID := range []string{"server1", "server2"} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/server/mod/add/"+serverID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Mozilla/5.0")
		req.RemoteAddr = "127.0.0.1:12345"
		req.AddCookie(&http.Cookie{Name: common.JwtCookieName, Value: token})

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected %d for %s, got %d: %s", http.StatusOK, serverID, w.Code, w.Body.String())
		}
	}

	var count int64
	if err := model.DB.Model(&model.ServerMod{}).Where("mod_id = ?", "sodium").Count(&count).Error; err != nil {
		t.Fatalf("count server mods: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 records for mod_id sodium, got %d", count)
	}
}
