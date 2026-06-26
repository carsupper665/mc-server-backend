// router/api.go

package router

import (
	"go-backend/common"
	"go-backend/controller"
	"go-backend/middleware"
	// "go-backend/middleware"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func SetAPIRouter(router *gin.Engine, serverController *controller.ServerController) {

	c := serverController
	router.Use(middleware.CORS())

	//testApi := router.Group("/test-api")
	//testApi.Use(gzip.Gzip(gzip.DefaultCompression),
	//	middleware.DebugMode(),
	//)
	//{
	//
	//}

	//client := mcapi.Group("/client")
	//client.Use(middleware.ClientAppAuth())
	//{
	//	client.GET("/getUserInfo", controller.GetUserInfo)
	//}

	// New Api Router
	api := router.Group("/api")
	api.Use(gzip.Gzip(gzip.DefaultCompression),
		middleware.IpRateLimiter(common.GlobalApiRateLimitNum, common.GlobalApiRateLimitDuration),
		middleware.GloabalIPFilter(),
		middleware.UserAgentFilter(),
	)
	v1 := api.Group("/v1")
	apiPublic := v1.Group("/public")
	{
		apiPublic.GET("/finfo", controller.GetAllFabricVersions)
		apiPublic.GET("/vinfo", controller.GetAllVanillaVersions)
	}
	serverApi := v1.Group("/server")
	//serverApi.Use(middleware.ValidateJWTHeader())
	serverApi.Use(middleware.ValidateJWTV2())
	{
		serverApi.POST("/create", controller.CreateServer)
		serverApi.DELETE("/del/:server_id", c.DeleteServerById)

		serverApi.GET("/status/:server_id", c.GetStatus)
		serverApi.GET("/stop/:server_id", c.Stop)
		serverApi.GET("/start/:server_id", c.Start)

		serverApi.GET("/list/backup/:server_id", c.ListServerBackup)
		serverApi.GET("/backup/:server_id", c.Backup)
		serverApi.POST("/recover", c.SaveRollBack)

		serverApi.POST("/property/:server_id", c.GetServerProperties)
		serverApi.POST("/property/upload/:server_id", c.UploadProperty)

		serverApi.POST("/command/:server_id", c.SendCommand)
		serverApi.GET("/usage/:server_id", c.ServerUsage)
		serverApi.GET("/log/:server_id", c.GetServerLog)
		serverApi.GET("/details/:server_id", controller.ServerDetails)

	}
	serverMod := serverApi.Group("/mod")
	{
		serverMod.POST("/add/:server_id", c.AddMod)
		serverMod.GET("/remove/:server_id", c.DelMod)
		serverMod.GET("/update/:server_id", c.UpdateMod)
		serverMod.GET("/toggle/:server_id", c.ToggleMod) // Enable or disable a mod on the server
		serverMod.GET("/list/:server_id", c.ListMod)     // Query the list of mods installed on the server
		serverMod.GET("/job/:job_id", c.GetModInstallJob)
		serverMod.GET("/subscribe/:job_id", c.SubscribeModInstall)
	}

}
