// middleware/cors.go

package middleware

import (
	"go-backend/common"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS allows cross-origin requests only from FRONTEND_BASE_URL and CORS_ALLOW_ORIGINS.
// When the frontend is served from the same origin (reverse proxy), no CORS headers are needed.
// DEBUG mode allows every origin so the Vite dev server works from localhost, 127.0.0.1 or a LAN IP.
func CORS() gin.HandlerFunc {
	config := cors.DefaultConfig()
	if common.DebugMode {
		config.AllowAllOrigins = true
	} else {
		origins := AllowedOrigins(os.Getenv("FRONTEND_BASE_URL"), os.Getenv("CORS_ALLOW_ORIGINS"))
		if len(origins) == 0 {
			return func(c *gin.Context) { c.Next() }
		}
		config.AllowOrigins = origins
	}
	config.AllowCredentials = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"*"}
	return cors.New(config)
}

// AllowedOrigins merges the frontend URL with the extra comma-separated origins.
// Entries that are not http(s) origins (including "*") are dropped.
func AllowedOrigins(frontend, extra string) []string {
	seen := map[string]bool{}
	var out []string
	for _, o := range append([]string{frontend}, common.SplitList(extra)...) {
		o = strings.TrimSuffix(strings.Trim(strings.TrimSpace(o), `"`), "/")
		if !strings.HasPrefix(o, "http://") && !strings.HasPrefix(o, "https://") {
			continue
		}
		if !seen[o] {
			seen[o] = true
			out = append(out, o)
		}
	}
	return out
}
