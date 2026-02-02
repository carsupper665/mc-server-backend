// main.go
package main

import (
	"errors"
	"fmt"
	"go-backend/common"
	"go-backend/controller"
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

var logger *common.SysLogger

func main() {
	// .env config load
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println(err)
	}
	common.LoadEnv()

	// Setup logger
	common.SetupLogger()
	err = common.IntiLogger("system")
	if err != nil {
		fmt.Println(err)
		return
	}

	logger = common.Logger

	//common.SysLog("Backend Server Engine | " + common.Version + common.ColorBuild + " started")
	logger.Infof("Backend Server Engine | %s%s started", common.Version, common.ColorBuild)

	if os.Getenv("DEBUG") != "true" {
		logger.Info(common.ColorGreen + "Running in Release Mode" + common.ColorReset)
		gin.SetMode(gin.ReleaseMode)
	} else {
		logger.Info(common.ColorBrightCyan + "Debug mode is enabled, running in Debug Mode" + common.ColorReset)
	}

	err = controller.InitLogger()
	if err != nil {
		fmt.Println(err)
		return
	}

	// init DB
	err = model.InitDB()
	if err != nil {
		logger.Fatal("failed to init DB: %s", err.Error())
	}
	errUpLog := model.InitUpdateLogTable()
	if errUpLog != nil {
		logger.Fatal("failed to init update log table: %s", errUpLog.Error())
	} else {

		updateErr := service.CheckForUpdates()
		if updateErr != nil {
			if !errors.Is(updateErr, service.ErrAlreadyLatest) && !errors.Is(updateErr, service.ErrUpdateDisabled) {
				logger.Errorf("Update check failed: %s", updateErr.Error())
			}
		} else {
			logger.Info("Application updated successfully, please restart the application.")
			return
		}

	}

	// check root user
	err = model.CheckRootUser()
	if err != nil {
		logger.Errorf("failed to create root user: %s", err.Error())
	}

	// init HTTP server
	server := gin.New()
	server.Use(gin.CustomRecovery(func(c *gin.Context, err any) {
		logger.Errorf("panic detected: %v", err)
		err = common.SendErrorToDc(fmt.Sprintf("Panic detected: %v", err))
		if err != nil {
			logger.Errorf("Failed to send error to Discord: %v", err)
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
			logger.Fatal("failed to restart application: " + err.Error())
		}
		os.Exit(0)
	})

	time.Sleep(500 * time.Millisecond)
	logger.Infof("HTTP server listening on :%s", port)

	err = server.Run(":" + port)
	if err != nil {
		logger.Fatal("failed to start HTTP server: " + err.Error())
	}
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

	logger.Infof("New process started with PID: %d", cmd.Process.Pid)

	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("failed to release new process: %w", err)
	}

	return nil
}
