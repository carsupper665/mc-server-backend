package router

import (
	"go-backend/controller"
	"go-backend/middleware"

	"github.com/gin-gonic/gin"
)

func SetAdminRouter(router *gin.Engine) {
	op := router.Group("/op")
	v1 := op.Group("/v1")

	v1.Use(
		middleware.CORS(),
		middleware.GlobalIPFilter(),
		middleware.UserAgentFilter(),
		middleware.IpRateLimiter(100, 60),
		middleware.ValidateJWTV2(),
		middleware.AdminOnly(),
		middleware.AdminLogger(),
	)
	{
		v1.POST("/register", controller.NewUser)
		v1.POST("/reset", controller.ResetAccount) // v1.POST("/newpassword", controller.SetNewPassword)
		v1.GET("/users", controller.GetAllUser)
		v1.POST("/edit", controller.EditUser)
	}
}
