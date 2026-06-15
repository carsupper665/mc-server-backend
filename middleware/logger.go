// middleware/logger.go

package middleware

import (
	"fmt"
	"go-backend/common"
	"time"

	"github.com/gin-gonic/gin"
)

func SetUpLogger(server *gin.Engine) {
	server.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		var requestID string
		if param.Keys != nil {
			requestID = param.Keys[common.RequestIdKey].(string)
		}
		return fmt.Sprintf("%s | %s | %s %3d | %-50s | %s | %s\n",
			param.TimeStamp.Format("2006/01/02-15:04:05"),
			param.ClientIP,
			param.Method,
			param.StatusCode,
			param.Path,
			param.Latency,
			requestID,
		)
	}))
}

func AdminLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		var requestID string
		if c.Keys != nil {
			requestID = c.Keys[common.RequestIdKey].(string)
		}
		common.Logger.Infof("Admin-Log, %s | %s | %s %3d | %-50s | %s | %s\n",
			time.Now().Format("2006/01/02-15:04:05"),
			c.ClientIP(),
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			requestID,
		)
		c.Next()
	}
}
