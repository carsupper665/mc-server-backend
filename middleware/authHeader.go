package middleware

import (
	"fmt"
	"go-backend/common"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func ValidateJWTHeader() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := strings.TrimSpace(c.GetHeader("Authorization"))
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		token := strings.TrimSpace(parts[1])
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		if !common.ValidateUser(token, c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		payload, err := common.GetJWTPayload(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		rawUID, _ := payload["user_id"]
		rawUIDStr, ok := rawUID.(string)
		if !ok || rawUIDStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		uid, parseErr := strconv.ParseUint(rawUIDStr, 10, 32)
		if parseErr != nil {
			common.LogError(c.Request.Context(), fmt.Sprintf("Parse user_id error: %v", parseErr))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		c.Set("uintId", uint(uid))
		c.Set("stringId", rawUIDStr)
		c.Set("payload", payload)

		c.Next()
	}
}
