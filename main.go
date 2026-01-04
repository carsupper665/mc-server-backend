// main.go
package main

import (
	"errors"
	"fmt"
	"go-backend/common"
	"go-backend/middleware"
	"go-backend/model"
	"go-backend/router"
	"go-backend/service"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// 用於控制 server 關閉的 channel
var shutdownChan = make(chan struct{}, 1)

func main() {
	// .env config load
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println(err)
	}
	common.LoadEnv()

	// Setup logger
	common.SetupLogger()
	common.SysLog("Backend Server Engine | " + common.Version + common.ColorBuild + " started")

	if os.Getenv("DEBUG") != "true" {
		common.SysLog(common.ColorGreen + "Running in Release Mode" + common.ColorReset)
		gin.SetMode(gin.ReleaseMode)
	} else {
		common.SysLog(common.ColorBrightCyan + "Debug mode is enabled, running in Debug Mode" + common.ColorReset)
	}

	// init DB
	err = model.InitDB()
	if err != nil {
		common.FatalLog("failed to init DB: " + err.Error())
	}
	errUpLog := model.InitUpdateLogTable()
	if errUpLog != nil {
		common.FatalLog("failed to init update log table: " + errUpLog.Error())
	} else {

		updateErr := service.CheckForUpdates()
		if updateErr != nil {
			if !errors.Is(updateErr, service.ErrAlreadyLatest) && !errors.Is(updateErr, service.ErrUpdateDisabled) {
				common.SysError("Update check failed: " + updateErr.Error())
			}
		} else {
			common.SysLog("Application updated successfully, please restart the application.")
			return
		}

	}

	// check root user
	err = model.CheckRootUser()
	if err != nil {
		common.SysError("failed to create root user: " + err.Error())
	}

	// init HTTP server
	server := gin.New()
	server.Use(gin.CustomRecovery(func(c *gin.Context, err any) {
		common.SysError(fmt.Sprintf("panic detected: %v", err))
		err = common.SendErrorToDc(fmt.Sprintf("Panic detected: %v", err))
		if err != nil {
			common.SysError(fmt.Sprintf("Failed to send error to Discord: %v", err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": fmt.Sprintf("Unknown Error: %v", err),
				"type":    "unknown_panic",
			},
		})
	}))

	server.Use(middleware.RequestId())
	middleware.SetUpLogger(server)

	// init session store
	store := cookie.NewStore([]byte(common.SessionSecret))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   2592000, // 30 days
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})
	server.Use(sessions.Sessions("session", store))

	// set router
	router.SetRouter(server)

	// get port
	var port = os.Getenv("PORT")
	if port == "" {
		port = strconv.Itoa(*common.Port)
	}

	service.StartUpdateChecker(3*24*time.Hour, func() {
		err := restartApplication()
		if err != nil {
			common.FatalLog("failed to restart application: " + err.Error())
		}
		os.Exit(0)
	})
	common.SysLog(fmt.Sprintf("HTTP server listening on :%s", port))
	err = server.Run(":" + port)
	if err != nil {
		common.FatalLog("failed to start HTTP server: " + err.Error())
	}

	common.SysLog("Application stopped")
}

// restartApplication 重啟應用程式
func restartApplication() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	args := os.Args[1:]
	cmd := exec.Command(executable, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start new process: %w", err)
	}

	common.SysLog(fmt.Sprintf("New process started with PID: %d", cmd.Process.Pid))

	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("failed to release new process: %w", err)
	}

	return nil
}
