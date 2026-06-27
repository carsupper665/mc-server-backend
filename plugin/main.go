package plugin

import (
	"flag"
	"go-backend/common"
	"sync"

	"github.com/gin-gonic/gin"
)

var logPath = flag.String("Plugin Log", "plugin_mgr", "specify the log name")

type BasePlugin struct {
	Name      string
	Version   string
	SetRouter func(router *gin.Engine)
}

//type Dependence struct {
//	Name    string
//	Version string
//}

var (
	registeredPlugins []BasePlugin
	pluginMu          sync.RWMutex
)

func Register(p BasePlugin) {
	pluginMu.Lock()
	defer pluginMu.Unlock()

	registeredPlugins = append(registeredPlugins, p)
}

var PluginManagerLogger *common.SysLogger

func PluginInitialize(mainRouter *gin.Engine) error {
	var logger *common.SysLogger
	l, err := common.NewSysLogger("plugin_mgr", logPath, 50000)
	if err != nil {
		return err
	}
	logger = l
	PluginManagerLogger = logger
	PluginManagerLogger.Info("PluginManager Start.")

	plugins := loadAllPlugin()
	for _, p := range plugins {
		logger.Debugf("Loading plugin: %s, Version: %s", p.Name, p.Version)
		p.SetRouter(mainRouter)
	}
	return nil
}

func loadAllPlugin() []BasePlugin {
	logger := PluginManagerLogger
	logger.Infof("Loading registered plugins...")

	pluginMu.RLock()
	defer pluginMu.RUnlock()

	// 複製一份，避免外部直接改 registry
	plugins := make([]BasePlugin, len(registeredPlugins))
	copy(plugins, registeredPlugins)

	logger.Infof(
		"Loaded %d plugins.",
		len(plugins),
	)

	return plugins
}
