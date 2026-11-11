package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go-backend/common"
	"go-backend/model"
	"gorm.io/gorm"
)

func setupAdminUserDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	previous := model.DB
	oldLogDir, oldLogger := *common.LogDir, common.Logger
	*common.LogDir = t.TempDir()
	if err := common.IntiLogger("admin-test"); err != nil {
		t.Fatal(err)
	}
	testLogger := common.Logger
	model.DB = db
	t.Cleanup(func() {
		_ = testLogger.Close()
		model.DB = previous
		*common.LogDir = oldLogDir
		common.Logger = oldLogger
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
}

func postJSON(r http.Handler, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestNewUserRoleValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupAdminUserDB(t)
	r := gin.New()
	r.POST("/register", NewUser)

	guest := 0
	w := postJSON(r, "/register", map[string]any{
		"username": "guest0", "display_name": "Guest", "email": "guest0@example.test",
		"password": "password1", "role": guest,
	})
	if w.Code != 200 {
		t.Fatalf("create guest: %d %s", w.Code, w.Body.String())
	}
	var created model.User
	if err := model.DB.Where("username = ?", "guest0").First(&created).Error; err != nil {
		t.Fatal(err)
	}
	if created.Role != common.RoleGuestUser {
		t.Fatalf("role=%d", created.Role)
	}
	if created.CreatedAt.IsZero() {
		t.Fatal("CreatedAt is zero")
	}

	for _, role := range []int{common.RoleExternalMemberUser, 4, 99} {
		w := postJSON(r, "/register", map[string]any{
			"username": "bad", "display_name": "Bad", "email": "bad@example.test",
			"password": "password1", "role": role,
		})
		if w.Code != 400 {
			t.Fatalf("role %d: want 400, got %d %s", role, w.Code, w.Body.String())
		}
	}
}

func TestEditUserRoleValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupAdminUserDB(t)
	user := model.User{Username: "u1", DisplayName: "U1", Email: "u1@example.test", Password: "h", Salt: "s", Role: common.RoleMemberUser}
	if err := model.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.POST("/edit", EditUser)

	guest := common.RoleGuestUser
	w := postJSON(r, "/edit", map[string]any{"user_id": user.ID, "new_role": guest})
	if w.Code != 200 {
		t.Fatalf("edit guest: %d %s", w.Code, w.Body.String())
	}
	var updated model.User
	if err := model.DB.First(&updated, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.Role != common.RoleGuestUser {
		t.Fatalf("role=%d", updated.Role)
	}

	for _, role := range []int{common.RoleExternalMemberUser, 4, 99} {
		w := postJSON(r, "/edit", map[string]any{"user_id": user.ID, "new_role": role})
		if w.Code != 400 {
			t.Fatalf("role %d: want 400, got %d %s", role, w.Code, w.Body.String())
		}
	}
	if err := model.DB.First(&updated, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.Role != common.RoleGuestUser {
		t.Fatalf("rejected edit changed role to %d", updated.Role)
	}
}
