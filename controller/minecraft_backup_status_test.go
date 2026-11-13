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

func TestBackupListAndDeleteStatus(t *testing.T) {
	for _, action := range []string{"list", "delete"} {
		for _, kind := range []string{"missing", "foreign", "owned"} {
			t.Run(action+"/"+kind, func(t *testing.T) {
				setupCreateLimitDB(t)
				user := createLimitUser(t, common.RoleVipMemberUser)
				owner := user
				if kind == "foreign" {
					owner = createLimitUser(t, common.RoleRootUser)
				}
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "sentinel"), []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
				if kind != "missing" {
					seedOwnedServer(t, owner.ID, "server", 0, 0)
					if err := model.DB.Model(&model.UserMinecraftServer{}).Where("server_id = ?", "server").Update("system_path", dir).Error; err != nil {
						t.Fatal(err)
					}
				}
				sc := NewServerController(service.NewServerService(&service.ServerManager{}))
				handler := sc.ListServerBackup
				method := http.MethodGet
				if action == "delete" {
					handler = sc.DeleteServerById
					method = http.MethodDelete
				}
				r := testUIDRouter(t, "/server/:server_id", handler)
				r.Handle(method, "/server/:server_id", handler)
				w := httptest.NewRecorder()
				req := httptest.NewRequest(method, "/server/server", nil)
				req.Header.Set("X-Test-UID", fmt.Sprint(user.ID))
				r.ServeHTTP(w, req)
				wantStatus := 404
				if kind == "owned" {
					wantStatus = 200
				}
				if w.Code != wantStatus {
					t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
				}
				if action == "list" && kind == "owned" {
					var body struct {
						Backups []string `json:"backups"`
					}
					if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
						t.Fatal(err)
					}
					if body.Backups == nil || len(body.Backups) != 0 {
						t.Fatalf("expected empty array: %s", w.Body.String())
					}
				}
				if action == "delete" && kind == "owned" {
					if _, err := os.Stat(dir); !os.IsNotExist(err) {
						t.Fatalf("server files remain: %v", err)
					}
					if _, err := model.GetServerByID(owner.ID, "server"); err == nil {
						t.Fatal("server record remains")
					}
				} else {
					data, err := os.ReadFile(filepath.Join(dir, "sentinel"))
					if err != nil || string(data) != "keep" {
						t.Fatalf("files changed: %q err=%v", data, err)
					}
					if kind == "foreign" {
						if _, err := model.GetServerByID(owner.ID, "server"); err != nil {
							t.Fatalf("foreign server changed: %v", err)
						}
					}
				}
			})
		}
	}
}
