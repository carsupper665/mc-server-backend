// main.go
package main

import (
	"context"
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
	"os/signal"
	"strconv"
	"sync"
	"syscall"
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
	logger.Infof("Backend Server Engine | %s%s started", common.Version, common.ColorBuild)

	if os.Getenv("DEBUG") != "true" {
		logger.Info(common.ColorGreen + "Running in Release Mode" + common.ColorReset)
		gin.SetMode(gin.ReleaseMode)
	} else {
		logger.Info(common.ColorBrightCyan + "Debug mode is enabled, running in Debug Mode" + common.ColorReset)
	}

	// 初始化棄用 token 表
	common.InitTokenRegister()

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
		//var updateErr error = service.ErrAlreadyLatest
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

	// unified manual shutdown trigger for non-signal paths.
	manualCtx, manualCancel := context.WithCancel(context.Background())
	defer manualCancel()
	var shutdownOnce sync.Once
	triggerShutdown := func(reason string) {
		shutdownOnce.Do(func() {
			logger.Infof("Shutdown requested: %s", reason)
			manualCancel()
		})
	}

	// 背景事件 初始化
	common.InitEventLoop()
	common.EL.Start()
	if err := common.EL.RegisterEvent("Clear-Tokens", common.RevokedTokens.ClearEvent, 7*time.Hour, -1); err != nil {
		logger.Fatal("failed to register Clear-Tokens event: " + err.Error())
		return
	}
	if err := eventRegister(triggerShutdown); err != nil {
		logger.Fatal("failed to register events: " + err.Error())
		return
	}

	time.Sleep(500 * time.Millisecond)

	// get port and start server
	var port = os.Getenv("PORT")
	if port == "" {
		port = strconv.Itoa(*common.Port)
	}
	logger.Infof("HTTP server listening on :%s, Ctrl+C To close.", port)

	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: server,
	}

	serverErrCh := make(chan error, 1)
	go func() {
		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
			return
		}
		serverErrCh <- nil
	}()

	//  system signals are routed to the same shutdown path.
	sigCtx, stopSig := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSig()

	select {
	case <-sigCtx.Done():
		triggerShutdown("signal")
	case <-manualCtx.Done():
		// triggered by non-signal shutdown sources.
	case err := <-serverErrCh:
		if err != nil {
			logger.Fatal("failed to start HTTP server: " + err.Error())
		}
	}

	common.EL.StopEventLoop()

	// Shutdown HTTP server and wait
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Errorf("HTTP server shutdown error: %s", err.Error())
	}
}

func eventRegister(triggerShutdown func(reason string)) error {
	upf := service.BuildUpdateChecker(func() {
		err := restartApplication()
		if err != nil {
			logger.Errorf("failed to restart application: %s", err.Error())
			return
		}
		triggerShutdown("auto-update")
	})

	if err := common.EL.RegisterEvent("Auto-Update", upf, 24*3*time.Hour, -1); err != nil {
		logger.Errorf("failed to register update checker: %s", err.Error())
	}

	if err := common.EL.RegisterEvent("Mod-Version-AutoSync", service.SyncInstalledModVersions, 24*time.Hour, -1); err != nil {
		logger.Errorf("failed to register mod version auto-sync: %s", err.Error())
	}

	return nil
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
