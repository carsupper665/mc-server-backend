package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"go-backend/common"
	"go-backend/model"
	"go-backend/service"
)

func TestBackupPermission(t *testing.T) {
	for _, role := range []int{common.RoleGuestUser, common.RoleMemberUser, common.RoleExternalMemberUser, common.RoleVipMemberUser, common.RoleRootUser, 99} {
		t.Run(fmt.Sprintf("role_%d", role), func(t *testing.T) {
			setupCreateLimitDB(t)
			user := createLimitUser(t, role)
			seedOwnedServer(t, user.ID, "backup-server", 0, 0)
			workDir := t.TempDir()
			if err := model.DB.Model(&model.UserMinecraftServer{}).Where("server_id = ?", "backup-server").Update("system_path", workDir).Error; err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(filepath.Join(workDir, "world"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(workDir, "world", "level.dat"), []byte("world data"), 0600); err != nil {
				t.Fatal(err)
			}
			sc := NewServerController(service.NewServerService(&service.ServerManager{}))
			r := testUIDRouter(t, "/backup/:server_id", sc.Backup)
			w := postJSONAs(r, "/backup/backup-server", user.ID, nil)
			perm, defined := common.PermFor(role)
			if !common.IsRoot(role) && (!defined || !perm.CanBackUpServer) {
				var body map[string]any
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				code := "PERM_BACKUP_DENIED"
				if !defined {
					code = "PERM_ROLE_UNDEFINED"
				}
				if w.Code != http.StatusForbidden || body["code"] != code {
					t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
				}
				if _, err := os.Stat(filepath.Join(workDir, "backup")); !os.IsNotExist(err) {
					t.Fatalf("denied request created backup directory: %v", err)
				}
				return
			}
			if w.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			files, err := filepath.Glob(filepath.Join(workDir, "backup", "*", "level.dat"))
			if err != nil || len(files) != 1 {
				t.Fatalf("backup files=%v err=%v", files, err)
			}
			data, err := os.ReadFile(files[0])
			if err != nil || string(data) != "world data" {
				t.Fatalf("backup data=%q err=%v", data, err)
			}
		})
	}
}
