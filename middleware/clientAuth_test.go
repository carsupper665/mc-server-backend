package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go-backend/common"
	"go-backend/model"
	"gorm.io/gorm"
)

func TestAdminOnlyWithJWT(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	previousRegistry := common.RevokedTokens
	common.RevokedTokens = common.NewRevokedTokenRegistry()
	t.Cleanup(func() { common.RevokedTokens = previousRegistry })
	previous := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previous; sqlDB, _ := db.DB(); _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	root := model.User{Username: "root", Email: "root@example.test", Role: common.RoleRootUser}
	member := model.User{Username: "member", Email: "member@example.test", Role: common.RoleMemberUser}
	for _, user := range []*model.User{&root, &member} {
		if err := db.Create(user).Error; err != nil {
			t.Fatal(err)
		}
	}
	tokenFor := func(id uint) string {
		token, err := common.GenerateJWTToken(map[string]interface{}{"user_id": fmt.Sprint(id)})
		if err != nil {
			t.Fatal(err)
		}
		return token
	}
	tests := []struct {
		name, token, adminToken string
		want                    int
	}{
		{"root without admin token", tokenFor(root.ID), "", http.StatusNoContent},
		{"root with obsolete admin token", tokenFor(root.ID), "unused", http.StatusNoContent},
		{"non root", tokenFor(member.ID), "", http.StatusForbidden},
		{"not logged in", "", "", http.StatusUnauthorized},
		{"admin token alone", "", tokenFor(root.ID), http.StatusUnauthorized},
	}
	r := gin.New()
	r.GET("/admin", ValidateJWTV2(), AdminOnly(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			req.Header.Set(common.DeviceHeader, "test-device")
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			if tt.adminToken != "" {
				req.Header.Set("A-OP-Token", tt.adminToken)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tt.want {
				t.Fatalf("want %d, got %d: %s", tt.want, w.Code, w.Body.String())
			}
		})
	}
}

func TestAdminOnlyWithoutAuthenticatedUser(t *testing.T) {
	r := gin.New()
	r.GET("/admin", AdminOnly(), func(c *gin.Context) { t.Error("unauthenticated request reached handler") })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}
