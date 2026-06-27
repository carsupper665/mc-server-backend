// router/auth.go

package router

import (
	"go-backend/controller"
	"go-backend/middleware"

	// "go-backend/middleware"
	"go-backend/common"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func SetAuthRouter(router *gin.Engine) {
	router.Use(middleware.CORS())

	lc := controller.NewChallengeStore()

	auth := router.Group("/Authentication")
	auth.Use(
		gzip.Gzip(gzip.DefaultCompression),
		middleware.IpRateLimiter(common.GlobalApiRateLimitNum, common.GlobalApiRateLimitDuration),
		middleware.UserAgentFilter(),
		middleware.GlobalIPFilter(),
	)
	{
		//auth.POST("/login", controller.Login)
		auth.POST("/login", lc.ChallengeLogin)
		//auth.POST("/verify", controller.VerifyLogin)
		auth.GET("/verify", lc.UrlVerifyLogin)
		auth.GET("/challenge", lc.ExchangeToken)
		//auth.POST("/app/verify", controller.VerifyLogin)// 代刪除
		//auth.POST("/app/login", controller.AppLogin) // 代刪除
		//auth.POST("/oidc/login", controller.OIDCLogin)
	}
}
