package controller

import (
	"go-backend/model"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestUrlVerifyLoginRedirect(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	previous := model.DB
	model.DB = db
	t.Cleanup(func() {
		model.DB = previous
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})
	if err := db.AutoMigrate(&model.LoginAttempt{}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FRONTEND_BASE_URL", "https://frontend.example.test")

	for _, id := range []string{"user-123", "user +&=?#/%中文"} {
		t.Run(id, func(t *testing.T) {
			store := NewChallengeStore()
			email := "user@example.test"
			store.CreateLoginChallenge(id, email)
			router := gin.New()
			router.GET("/verify", store.UrlVerifyLogin)
			query := url.Values{"id": {id}, "code": {email}}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/verify?"+query.Encode(), nil))
			if response.Code != http.StatusFound {
				t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusFound, response.Body)
			}
			location, err := url.Parse(response.Header().Get("Location"))
			if err != nil {
				t.Fatal(err)
			}
			if location.Scheme != "https" || location.Host != "frontend.example.test" || location.Path != "/login/callback" || location.Fragment != "" {
				t.Fatalf("unexpected callback URL: %s", location)
			}
			params, err := url.ParseQuery(location.RawQuery)
			if err != nil {
				t.Fatal(err)
			}
			if len(params) != 2 || len(params["id"]) != 1 || params.Get("id") != id {
				t.Fatalf("unexpected callback parameters: %v", params)
			}
			if len(params["code"]) != 1 || params.Get("code") == "" || params.Get("code") != store.tokenCache[id].AuthToken {
				t.Fatalf("callback code does not match stored auth token: %v", params)
			}
		})
	}
}
