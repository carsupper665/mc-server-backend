package controller

import (
	"encoding/json"
	"fmt"
	"testing"

	"go-backend/common"
	"go-backend/model"
	"go-backend/service"
)

func TestServerUsageNotOwnerOrStopped(t *testing.T) {
	setupCreateLimitDB(t)
	owner := createLimitUser(t, common.RoleMemberUser)
	other := createLimitUser(t, common.RoleVipMemberUser)
	workDir := t.TempDir()
	seedOwnedServer(t, owner.ID, "usage-server", 0, 0)
	if err := model.DB.Model(&model.UserMinecraftServer{}).Where("server_id = ?", "usage-server").Update("system_path", workDir).Error; err != nil {
		t.Fatal(err)
	}
	sc := NewServerController(service.NewServerService(&service.ServerManager{}))
	r := testUIDRouter(t, "/usage/:server_id", sc.ServerUsage)
	for _, tc := range []struct {
		name string
		uid  uint
		sid  string
	}{
		{"missing", owner.ID, "missing"},
		{"other owner", other.ID, "usage-server"},
		{"stopped", owner.ID, "usage-server"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := postJSONAs(r, "/usage/"+tc.sid, tc.uid, nil)
			if w.Code != 404 || w.Body.String() != `{"error":"Server not found"}` {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if got, err := model.GetServerByID(owner.ID, "usage-server"); err != nil || got.SystemPath != workDir {
				t.Fatalf("server changed: %+v err=%v", got, err)
			}
		})
	}
}

func TestServerUsageReturnsSummaryWhileRunning(t *testing.T) {
	setupCreateLimitDB(t)
	owner := createLimitUser(t, common.RoleMemberUser)
	other := createLimitUser(t, common.RoleVipMemberUser)
	workDir := t.TempDir()
	seedOwnedServer(t, owner.ID, "usage-server", 0, 0)
	if err := model.DB.Model(&model.UserMinecraftServer{}).Where("server_id = ?", "usage-server").Update("system_path", workDir).Error; err != nil {
		t.Fatal(err)
	}
	sc := NewServerController(startRunningServer(t, "usage-server", fmt.Sprint(owner.ID), workDir))
	r := testUIDRouter(t, "/usage/:server_id", sc.ServerUsage)
	w := postJSONAs(r, "/usage/usage-server", other.ID, nil)
	if w.Code != 404 || w.Body.String() != `{"error":"Server not found"}` {
		t.Fatalf("other: status=%d body=%s", w.Code, w.Body.String())
	}
	w = postJSONAs(r, "/usage/usage-server", owner.ID, nil)
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	if len(raw) != 1 {
		t.Fatalf("body=%s", w.Body.String())
	}
	var usage map[string]json.RawMessage
	if err := json.Unmarshal(raw["usage"], &usage); err != nil {
		t.Fatalf("body=%s err=%v", w.Body.String(), err)
	}
	for _, key := range []string{"cpu_usage", "memory_usage", "threads"} {
		if _, ok := usage[key]; !ok {
			t.Fatalf("missing %s in %s", key, w.Body.String())
		}
	}
	if len(usage) != 3 {
		t.Fatalf("usage=%s", w.Body.String())
	}
}
