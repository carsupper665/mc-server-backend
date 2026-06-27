package plugins

import (
	"go-backend/plugin"
	"go-backend/plugins/testPlugin"
)

func init() {
	plugin.Register(plugin.BasePlugin{
		Name:      testplugin.Name,
		Version:   testplugin.Version,
		SetRouter: testplugin.SetRouter,
	})
}
