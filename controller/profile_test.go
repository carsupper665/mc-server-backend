package controller

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go-backend/common"
	"go-backend/model"
	"gorm.io/gorm"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestCurrentUserDisplayNameAndCredentialBoundary(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	previous := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previous; sqlDB, _ := db.DB(); sqlDB.Close() })
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	secret := "private-access-token"
	user := model.User{Username: "operator", DisplayName: "顯示名稱", Email: "operator@example.test", Password: "private-hash", Salt: "private-salt", AccessToken: &secret}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	for _, id := range []uint{0, user.ID, user.ID + 1} {
		r := gin.New()
		r.GET("/me", func(c *gin.Context) {
			if id != 0 {
				c.Set("uintId", id)
			}
			CurrentUser(c)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/me?user_id=1", nil))
		if id != user.ID {
			if w.Code != 401 {
				t.Fatalf("missing identity accepted: %d", w.Code)
			}
			continue
		}
		if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal(w.Code, w.Header())
		}
		var profile map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &profile); err != nil {
			t.Fatal(err)
		}
		if profile["display_name"] != "顯示名稱" || profile["username"] != "operator" {
			t.Fatal(profile)
		}
		for _, field := range []string{"password", "salt", "access_token", "private-"} {
			if strings.Contains(w.Body.String(), field) {
				t.Fatal("credential leaked")
			}
		}
	}
}

func TestCurrentUserPermissions(t *testing.T) {
	for _, tt := range []struct {
		name                                             string
		role                                             int
		maxMem, maxServer, maxOnline, maxMods, maxBackup int64
		create, backup, unlimited                        bool
	}{
		{"guest", common.RoleGuestUser, 512 * common.MB, 1, 1, 10, 0, true, false, false},
		{"member", common.RoleMemberUser, 3 * common.GB, 5, 1, 50, 0, true, false, false},
		{"external member", common.RoleExternalMemberUser, 3 * common.GB, 5, 1, 50, 0, true, false, false},
		{"vip", common.RoleVipMemberUser, 8 * common.GB, 10, 2, 256, 5, true, true, false},
		{"root", common.RoleRootUser, 0, 0, 0, 0, 0, true, true, true},
		{"undefined", 4, 0, 0, 0, 0, 0, false, false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			setupCreateLimitDB(t)
			user := createLimitUser(t, tt.role)
			r := testUIDRouter(t, "/me", CurrentUser)
			w := postJSONAs(r, "/me", user.ID, nil)
			if w.Code != 200 {
				t.Fatalf("status=%d body=%s", w.Code, w.Body)
			}
			var profile struct {
				Permissions map[string]any `json:"permissions"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &profile); err != nil {
				t.Fatal(err)
			}
			want := map[string]any{
				"unlimited":            tt.unlimited,
				"server_max_mem":       float64(tt.maxMem),
				"max_server_count":     float64(tt.maxServer),
				"max_online_count":     float64(tt.maxOnline),
				"max_mods_count":       float64(tt.maxMods),
				"max_back_up_count":    float64(tt.maxBackup),
				"can_add_user":         tt.unlimited,
				"can_create_server":    tt.create,
				"can_edit_server_args": tt.unlimited,
				"can_back_up_server":   tt.backup,
			}
			if !reflect.DeepEqual(profile.Permissions, want) {
				t.Fatalf("permissions=%v, want=%v", profile.Permissions, want)
			}
		})
	}
}
