// router/amongUs.go

package router

import (
	"go-backend/common"
	"go-backend/controller"
	"go-backend/middleware"
	"go-backend/service"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func SetAmongUsIRouter(router *gin.Engine) {
	c := controller.NewAmongUsController(service.NewGameManager())
	router.Use(
		gzip.Gzip(gzip.DefaultCompression),
		middleware.CORS(),
		middleware.IpRateLimiter(common.GlobalApiRateLimitNum, common.GlobalApiRateLimitDuration),
		middleware.UserAgentFilter(),
	)

	auRouter := router.Group("/LOL-AmongUs")

	pr := auRouter.Group("/public")

	pr.Use(
		middleware.GlobalIPFilter(),
	)
	{
		pr.GET("/Join/:id", c.Join)
	}

	web := auRouter.Group("/a")
	web.Use(middleware.ValidateJWTV2())
	{
		web.GET("/end/:id", c.EndGame)
		web.GET("/c", c.Create)
		web.GET("/ls", c.AllGames)
		web.GET("/ls-p/:id", c.ListPlayers)
	}

	admin := auRouter.Group("/appAdmin")
	admin.Use(middleware.AdminOnly())
	{
		admin.GET("/end/:id", c.EndGame)
		admin.GET("/c", c.Create)
		admin.GET("/ls", c.AllGames)
	}

}
