package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go-backend/common"
	"go-backend/model"
	"go-backend/service"

	"github.com/gin-gonic/gin"
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

func TestRollingBackup(t *testing.T) {
	perm, ok := common.PermFor(common.RoleVipMemberUser)
	if !ok || !perm.CanBackUpServer || perm.MaxBackUpCount <= 0 {
		t.Fatal("VIP backup permission missing")
	}
	limit := int(perm.MaxBackUpCount)
	for _, tt := range []struct {
		name      string
		role      int
		count     int
		fail      bool
		wantCount int
		removed   int
	}{
		{"below limit", common.RoleVipMemberUser, limit - 1, false, limit, 0},
		{"at limit", common.RoleVipMemberUser, limit, false, limit, 1},
		{"above limit", common.RoleVipMemberUser, limit + 1, false, limit, 2},
		{"failed backup", common.RoleVipMemberUser, limit, true, 0, 0},
		{"root unlimited", common.RoleRootUser, limit + 1, false, limit + 2, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			setupCreateLimitDB(t)
			user := createLimitUser(t, tt.role)
			seedOwnedServer(t, user.ID, "backup-server", 0, 0)
			workDir := t.TempDir()
			if err := model.DB.Model(&model.UserMinecraftServer{}).Where("server_id = ?", "backup-server").Update("system_path", workDir).Error; err != nil {
				t.Fatal(err)
			}
			oldNames := make([]string, tt.count)
			// Create in reverse order so creation order does not define the oldest backup.
			for i := tt.count - 1; i >= 0; i-- {
				oldNames[i] = time.Date(2020, 1, 1, 0, 0, i, 0, time.UTC).Format("20060102_150405")
				dir := filepath.Join(workDir, "backup", oldNames[i])
				if err := os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "level.dat"), []byte(oldNames[i]), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if !tt.fail {
				if err := os.Mkdir(filepath.Join(workDir, "world"), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(workDir, "world", "level.dat"), []byte("new world data"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			sc := NewServerController(service.NewServerService(&service.ServerManager{}))
			r := testUIDRouter(t, "/backup/:server_id", func(c *gin.Context) {
				c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), common.RequestIdKey, "rolling-backup-test"))
				sc.Backup(c)
			})
			w := postJSONAs(r, "/backup/backup-server", user.ID, nil)
			wantStatus := http.StatusOK
			if tt.fail {
				wantStatus = http.StatusConflict
			}
			if w.Code != wantStatus {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			for i, name := range oldNames {
				data, err := os.ReadFile(filepath.Join(workDir, "backup", name, "level.dat"))
				if i < tt.removed {
					if !os.IsNotExist(err) {
						t.Fatalf("oldest backup %s was not deleted: %v", name, err)
					}
				} else if err != nil || string(data) != name {
					t.Fatalf("old backup %s changed: data=%q err=%v", name, data, err)
				}
			}
			if tt.fail {
				return
			}
			backups, err := sc.svc.ListBackups(workDir)
			if err != nil || len(backups) != tt.wantCount {
				t.Fatalf("backups=%v want count=%d err=%v", backups, tt.wantCount, err)
			}
			data, err := os.ReadFile(filepath.Join(workDir, "backup", backups[len(backups)-1], "level.dat"))
			if err != nil || string(data) != "new world data" {
				t.Fatalf("new backup data=%q err=%v", data, err)
			}
		})
	}
}

func TestRecoverBackupName(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, common.RoleGuestUser)
	seedOwnedServer(t, user.ID, "recover-server", 0, 0)
	workDir := t.TempDir()
	if err := model.DB.Model(&model.UserMinecraftServer{}).Where("server_id = ?", "recover-server").Update("system_path", workDir).Error; err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(workDir, "backup", "20261114_120000")
	if err := os.MkdirAll(backup, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backup, "level.dat"), []byte("saved world"), 0600); err != nil {
		t.Fatal(err)
	}
	sc := NewServerController(service.NewServerService(&service.ServerManager{}))
	r := testUIDRouter(t, "/recover", sc.SaveRollBack)
	for _, name := range []string{"../x", "../../other/world", "/tmp/world", "a/b", "", ".", "..", `a\b`, `C:\world`, "not-a-backup"} {
		t.Run(name, func(t *testing.T) {
			w := postJSONAs(r, "/recover", user.ID, map[string]string{"server_id": "recover-server", "file_name": name})
			var body map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if w.Code != 400 || body["code"] != "INVALID_BACKUP_NAME" {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
	w := postJSONAs(r, "/recover", user.ID, map[string]string{"server_id": "recover-server", "file_name": "20261114_120001"})
	if w.Code != 404 {
		t.Fatalf("missing: status=%d body=%s", w.Code, w.Body.String())
	}
	w = postJSONAs(r, "/recover", user.ID, map[string]string{"server_id": "recover-server", "file_name": "20261114_120000"})
	if w.Code != 200 {
		t.Fatalf("valid: status=%d body=%s", w.Code, w.Body.String())
	}
	data, err := os.ReadFile(filepath.Join(workDir, "world", "level.dat"))
	if err != nil || string(data) != "saved world" {
		t.Fatalf("data=%q err=%v", data, err)
	}
}

func TestBackupAndRecoverInvalidWorldResponses(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, common.RoleVipMemberUser)
	seedOwnedServer(t, user.ID, "backup-server", 0, 0)
	dir := t.TempDir()
	if err := model.DB.Model(&model.UserMinecraftServer{}).Where("server_id = ?", "backup-server").Update("system_path", dir).Error; err != nil {
		t.Fatal(err)
	}
	sc := NewServerController(service.NewServerService(&service.ServerManager{}))
	r := testUIDRouter(t, "/backup/:server_id", sc.Backup)
	w := postJSONAs(r, "/backup/backup-server", user.ID, nil)
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if w.Code != 409 || body["code"] != "NO_WORLD" || body["error"] != "伺服器尚未產生世界，請先啟動一次" {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "backup")); !os.IsNotExist(err) {
		t.Fatalf("backup directory created: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "backup", "20261114_120000"), 0700); err != nil {
		t.Fatal(err)
	}
	r = testUIDRouter(t, "/recover", sc.SaveRollBack)
	w = postJSONAs(r, "/recover", user.ID, map[string]string{"server_id": "backup-server", "file_name": "20261114_120000"})
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if w.Code != 400 || body["code"] != "INVALID_BACKUP" {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
