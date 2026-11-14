package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go-backend/common"
	"go-backend/model"
	"go-backend/service"
)

func TestDeleteRunningServerConflictsAndKeepsData(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, common.RoleMemberUser)
	workDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workDir, "keep.txt"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	seedOwnedServer(t, user.ID, "run-server", common.GB, 2*common.GB)
	if err := model.DB.Model(&model.UserMinecraftServer{}).Where("server_id = ?", "run-server").Update("system_path", workDir).Error; err != nil {
		t.Fatal(err)
	}
	if err := model.DB.Create(&model.ServerMod{
		ServerID: "run-server", ModID: "keep-mod", VersionID: "v1", Filename: "keep.jar", Status: "installed",
	}).Error; err != nil {
		t.Fatal(err)
	}
	svc := startRunningServer(t, "run-server", fmt.Sprint(user.ID), workDir)
	if status, err := svc.Status("run-server"); err != nil || status != "running" {
		t.Fatalf("status=%s err=%v", status, err)
	}
	sc := NewServerController(svc)
	r := testUIDRouter(t, "/server/:server_id", sc.DeleteServerById)
	r.DELETE("/server/:server_id", sc.DeleteServerById)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/server/run-server", nil)
	req.Header.Set("X-Test-UID", fmt.Sprint(user.ID))
	r.ServeHTTP(w, req)
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusConflict || body["code"] != "SERVER_RUNNING" {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	data, err := os.ReadFile(filepath.Join(workDir, "keep.txt"))
	if err != nil || string(data) != "keep" {
		t.Fatalf("files changed: %q err=%v", data, err)
	}
	if got, err := model.GetServerByID(user.ID, "run-server"); err != nil || got.SystemPath != workDir {
		t.Fatalf("server record changed: %+v err=%v", got, err)
	}
	mod, err := model.GetServerMod("run-server", "keep-mod")
	if err != nil || mod.Filename != "keep.jar" {
		t.Fatalf("mod changed: %+v err=%v", mod, err)
	}
	if status, err := svc.Status("run-server"); err != nil || status != "running" {
		t.Fatalf("server no longer running: status=%s err=%v", status, err)
	}
}

func TestDeleteServerRemovesItsModsOnly(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, common.RoleMemberUser)
	keepDir := t.TempDir()
	goneDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(keepDir, "keep.txt"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goneDir, "gone.txt"), []byte("gone"), 0600); err != nil {
		t.Fatal(err)
	}
	seedOwnedServer(t, user.ID, "keep-server", 0, 0)
	seedOwnedServer(t, user.ID, "gone-server", 0, 0)
	for id, dir := range map[string]string{"keep-server": keepDir, "gone-server": goneDir} {
		if err := model.DB.Model(&model.UserMinecraftServer{}).Where("server_id = ?", id).Update("system_path", dir).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, mod := range []model.ServerMod{
		{ServerID: "keep-server", ModID: "keep-mod", VersionID: "v1", Filename: "keep.jar", Status: "installed"},
		{ServerID: "gone-server", ModID: "gone-mod", VersionID: "v1", Filename: "gone.jar", Status: "installed"},
	} {
		if err := model.DB.Create(&mod).Error; err != nil {
			t.Fatal(err)
		}
	}
	sc := NewServerController(service.NewServerService(&service.ServerManager{}))
	r := testUIDRouter(t, "/server/:server_id", sc.DeleteServerById)
	r.DELETE("/server/:server_id", sc.DeleteServerById)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/server/gone-server", nil)
	req.Header.Set("X-Test-UID", fmt.Sprint(user.ID))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(goneDir); !os.IsNotExist(err) {
		t.Fatalf("deleted server files remain: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(keepDir, "keep.txt"))
	if err != nil || string(data) != "keep" {
		t.Fatalf("other server files changed: %q err=%v", data, err)
	}
	if _, err := model.GetServerByID(user.ID, "gone-server"); err == nil {
		t.Fatal("deleted server record remains")
	}
	if got, err := model.GetServerByID(user.ID, "keep-server"); err != nil || got.SystemPath != keepDir {
		t.Fatalf("other server record changed: %+v err=%v", got, err)
	}
	if _, err := model.GetServerMod("gone-server", "gone-mod"); err == nil {
		t.Fatal("deleted server mod remains")
	}
	mod, err := model.GetServerMod("keep-server", "keep-mod")
	if err != nil || mod.Filename != "keep.jar" {
		t.Fatalf("other server mod changed: %+v err=%v", mod, err)
	}
}
