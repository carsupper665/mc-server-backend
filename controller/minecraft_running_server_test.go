package controller

import (
	"os"
	"path/filepath"
	"testing"

	"go-backend/common"
	"go-backend/service"
)

// startRunningServer registers sid as a running server by launching a stand-in
// java process. The process exits when this test process does.
func startRunningServer(t *testing.T, sid, ownerID, workDir string) *service.ServerService {
	t.Helper()
	setupServerHandlerLogger(t)
	javaDir := t.TempDir()
	script := "#!/bin/sh\nparent=$PPID\nwhile kill -0 \"$parent\" 2>/dev/null; do\n\tsleep 0.2\ndone\n"
	if err := os.WriteFile(filepath.Join(javaDir, "java"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", javaDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	oldLogger := common.Logger
	if err := common.IntiLogger("running-server"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if common.Logger != nil && common.Logger != oldLogger {
			_ = common.Logger.Close()
		}
		common.Logger = oldLogger
	})
	mgr := service.NewServerManager([]int{25565})
	if _, err := mgr.StartServer(sid, ownerID, workDir, "512M", "512M", nil); err != nil {
		t.Fatal(err)
	}
	return service.NewServerService(mgr)
}
