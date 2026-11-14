package controller

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go-backend/common"
	"go-backend/model"
	"go-backend/service"
)

func TestServerResponsesOmitSystemPath(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, common.RoleMemberUser)
	workDir := t.TempDir()
	backupName := "20261114_120000"
	if err := os.MkdirAll(filepath.Join(workDir, "backup", backupName), 0700); err != nil {
		t.Fatal(err)
	}
	seedOwnedServer(t, user.ID, "path-server", common.GB, 2*common.GB)
	if err := model.DB.Model(&model.UserMinecraftServer{}).Where("server_id = ?", "path-server").Update("system_path", workDir).Error; err != nil {
		t.Fatal(err)
	}

	details := testUIDRouter(t, "/details/:server_id", ServerDetails)
	w := postJSONAs(details, "/details/path-server", user.ID, nil)
	if w.Code != 200 {
		t.Fatalf("details status=%d body=%s", w.Code, w.Body.String())
	}
	assertNoSystemPath(t, w.Body.Bytes(), workDir)
	var detailsBody struct {
		Server map[string]any `json:"server"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &detailsBody); err != nil {
		t.Fatal(err)
	}
	if detailsBody.Server["server_id"] != "path-server" || detailsBody.Server["display_name"] != "s" {
		t.Fatalf("details=%s", w.Body.String())
	}

	mine := testUIDRouter(t, "/myservers", MyServers)
	w = postJSONAs(mine, "/myservers", user.ID, nil)
	if w.Code != 200 {
		t.Fatalf("myservers status=%d body=%s", w.Code, w.Body.String())
	}
	assertNoSystemPath(t, w.Body.Bytes(), workDir)
	var servers []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &servers); err != nil {
		t.Fatal(err)
	}
	if len(servers) != 1 || servers[0]["server_id"] != "path-server" {
		t.Fatalf("myservers=%s", w.Body.String())
	}

	got, err := model.GetServerByID(user.ID, "path-server")
	if err != nil || got.SystemPath != workDir {
		t.Fatalf("stored path=%q err=%v", got.SystemPath, err)
	}
	sc := NewServerController(service.NewServerService(&service.ServerManager{}))
	list := testUIDRouter(t, "/list/:server_id", sc.ListServerBackup)
	w = postJSONAs(list, "/list/path-server", user.ID, nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), backupName) || strings.Contains(w.Body.String(), workDir) {
		t.Fatalf("list status=%d body=%s", w.Code, w.Body.String())
	}
}

func assertNoSystemPath(t *testing.T, body []byte, workDir string) {
	t.Helper()
	if strings.Contains(string(body), "system_path") || strings.Contains(string(body), workDir) {
		t.Fatalf("response leaked host path: %s", body)
	}
}
