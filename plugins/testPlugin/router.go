package testplugin

import (
	"go-backend/middleware"

	"github.com/gin-gonic/gin"
)

const (
	Name    = "TestPlugin"
	Version = "0.0.1"
)

func SetRouter(mainRouter *gin.Engine) {
	h := NewExample()
	g := mainRouter.Group("/" + Name)
	g.Use(middleware.DebugMode())
	g.GET("/testplugin", TestPlugin)
	g.POST("/test/AddUser", h.Add)
	g.POST("/test/RemoveUser", h.Remove)
	g.GET("/test/ListUser", h.List)

}
