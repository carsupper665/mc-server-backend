// router/api.go

package router

import (
	"go-backend/common"
	"go-backend/controller"
	"go-backend/middleware"
	"go-backend/service"

	// "go-backend/middleware"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func SetAPIRouter(router *gin.Engine) {
	pl := common.GetPortList(30000, 30050)

	mgr := service.NewServerManager(pl)
	svc := service.NewServerService(mgr)
	c := controller.NewServerController(svc)
	router.Use(middleware.CORS())
	// old api delete later
	mcapi := router.Group("/mc-api")
	mcapi.Use(gzip.Gzip(gzip.DefaultCompression),
		middleware.IpRateLimiter(common.GlobalApiRateLimitNum, common.GlobalApiRateLimitDuration),
		middleware.GloabalIPFilter(),
		middleware.UserAgentFilter(),
	)
	{
		mcapi.GET("/finfo", controller.GetAllFabricVersions)
		mcapi.GET("/vinfo", controller.GetAllVanillaVersions)
	}
	amcapi := mcapi.Group("/a")
	amcapi.Use(middleware.ValidateJWT())
	{
		amcapi.POST("/create", controller.CreateServer)
		amcapi.POST("/backup/:server_id", c.Backup)
		amcapi.POST("/status/:server_id", c.GetStatus)
		amcapi.POST("/stop/:server_id", c.Stop)
		amcapi.POST("/start/:server_id", c.Start)
		amcapi.POST("/ls-backup/:server_id", c.ListServerBackup)
		amcapi.POST("/property/:server_id", c.GetServerProperties)
		amcapi.POST("/UploadProperty/:server_id", c.UploadProperty)
		amcapi.POST("/cmd/:server_id", c.SendCommand)
		amcapi.GET("/usage/:server_id", c.ServerUsage)
		amcapi.POST("/recover", c.SaveRollBack)
	}
	sapi := router.Group("/server-api")
	sapi.Use(gzip.Gzip(gzip.DefaultCompression),
		middleware.UserAgentFilter(),
		middleware.GloabalIPFilter(),
	)
	asapi := sapi.Group("/a")
	asapi.Use(middleware.ValidateJWT())
	{
		asapi.GET("/log/:server_id", c.GetServerLog)
	}

	testApi := router.Group("/test-api")
	testApi.Use(gzip.Gzip(gzip.DefaultCompression),
		middleware.DebugMode(),
	)
	{

	}

	client := mcapi.Group("/client")
	client.Use(middleware.ClientAppAuth())
	{
		client.GET("/getUserInfo", controller.GetUserInfo)
	}

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
	serverApi.Use(middleware.ValidateJWT())
	{
		serverApi.POST("/create", controller.CreateServer)
		serverApi.DELETE("/del/:server_id", c.DeleteServerById)

		serverApi.GET("/status/:server_id", c.GetStatus)
		serverApi.GET("/stop/:server_id", c.Stop)
		serverApi.GET("/start/:server_id", c.Start)

		serverApi.GET("/list/backup/:server_id", c.ListServerBackup)
		serverApi.GET("/backup/:server_id", c.Backup)
		serverApi.POST("/recover", c.SaveRollBack)

		serverApi.GET("/property/:server_id", c.GetServerProperties)
		serverApi.POST("/property/upload/:server_id", c.UploadProperty)

		serverApi.POST("/command/:server_id", c.SendCommand)
		serverApi.GET("/usage/:server_id", c.ServerUsage)
		serverApi.GET("/log/:server_id", c.GetServerLog)

	}
	serverMod := serverApi.Group("/mod")
	// New Features TODO list
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
