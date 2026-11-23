package middleware

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"go-backend/common"

	"github.com/gin-gonic/gin"
)

func setDebug(t *testing.T, on bool) {
	t.Helper()
	old := common.DebugMode
	common.DebugMode = on
	t.Cleanup(func() { common.DebugMode = old })
}

func TestAllowedOrigins(t *testing.T) {
	got := AllowedOrigins(`"https://panel.example.com/"`, " https://a.example.com , *, ftp://x, https://panel.example.com,,http://localhost:5173")
	want := []string{"https://panel.example.com", "https://a.example.com", "http://localhost:5173"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("AllowedOrigins = %v, want %v", got, want)
	}
	if got := AllowedOrigins("", ""); len(got) != 0 {
		t.Fatalf("empty config should allow no origins, got %v", got)
	}
}

func corsResponse(t *testing.T, origin string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", origin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCORSAllowsOnlyConfiguredOrigins(t *testing.T) {
	setDebug(t, false)
	t.Setenv("FRONTEND_BASE_URL", "https://panel.example.com")
	t.Setenv("CORS_ALLOW_ORIGINS", "https://tools.example.com")

	for _, origin := range []string{"https://panel.example.com", "https://tools.example.com"} {
		w := corsResponse(t, origin)
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Fatalf("origin %s: Allow-Origin = %q", origin, got)
		}
		if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Fatalf("origin %s: credentials not allowed", origin)
		}
	}

	w := corsResponse(t, "https://evil.example.com")
	if w.Code != http.StatusForbidden || w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("unknown origin: code=%d allow-origin=%q", w.Code, w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSWithoutConfigSendsNoHeaders(t *testing.T) {
	setDebug(t, false)
	t.Setenv("FRONTEND_BASE_URL", "")
	t.Setenv("CORS_ALLOW_ORIGINS", "")
	w := corsResponse(t, "https://evil.example.com")
	if w.Code != http.StatusOK || w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("code=%d allow-origin=%q", w.Code, w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSDebugModeAllowsAnyOrigin(t *testing.T) {
	setDebug(t, true)
	t.Setenv("FRONTEND_BASE_URL", "http://localhost:5173")
	w := corsResponse(t, "http://127.0.0.1:5173")
	if w.Code != http.StatusOK || w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatalf("code=%d allow-origin=%q", w.Code, w.Header().Get("Access-Control-Allow-Origin"))
	}
}
