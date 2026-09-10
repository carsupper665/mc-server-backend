package controller

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go-backend/model"
	"gorm.io/gorm"
	"net/http/httptest"
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
