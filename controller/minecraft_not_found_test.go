package controller

import (
	"testing"

	"github.com/gin-gonic/gin"
	"go-backend/common"
)

func TestServerHandlersHideMissingAndUnownedServers(t *testing.T) {
	setupCreateLimitDB(t)
	setupServerHandlerLogger(t)
	root := createLimitUser(t, common.RoleRootUser)
	other := createLimitUser(t, common.RoleMemberUser)
	seedOwnedServer(t, other.ID, "other", common.GB, 2*common.GB)
	sc := &ServerController{}
	for _, tc := range []struct {
		name    string
		handler gin.HandlerFunc
		body    any
	}{
		{"memory", EditServerMemory, gin.H{"xmx": "1G"}},
		{"args", EditServerArgs, gin.H{"args": "-XX:+UseG1GC"}},
		{"details", ServerDetails, nil},
		{"add", sc.AddMod, gin.H{"mod_id": "mod"}},
		{"list", sc.ListMod, nil},
		{"remove", sc.DelMod, nil},
		{"update", sc.UpdateMod, nil},
		{"toggle", sc.ToggleMod, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := testUIDRouter(t, "/server/:server_id", tc.handler)
			for _, sid := range []string{"missing", "other"} {
				w := postJSONAs(r, "/server/"+sid+"?mod_id=mod", root.ID, tc.body)
				if w.Code != 404 || w.Body.String() != `{"error":"Server not found"}` {
					t.Errorf("%s: status=%d body=%s", sid, w.Code, w.Body.String())
				}
			}
		})
	}
}

func setupServerHandlerLogger(t *testing.T) {
	t.Helper()
	oldDir, oldLogger := *common.LogDir, logger
	*common.LogDir = t.TempDir()
	testLogger, err := common.NewSysLogger("server-handler-test", nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	logger = testLogger
	t.Cleanup(func() { testLogger.Close(); *common.LogDir, logger = oldDir, oldLogger })
}
