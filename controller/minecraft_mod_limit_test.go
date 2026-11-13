package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"go-backend/common"
	"go-backend/model"
)

func migrateServerMods(t *testing.T) {
	t.Helper()
	if err := model.DB.AutoMigrate(&model.ServerMod{}); err != nil {
		t.Fatal(err)
	}
}

func seedServerMods(t *testing.T, serverID string, enabledN, disabledN int) {
	t.Helper()
	now := time.Now()
	idx := 0
	for i := 0; i < enabledN; i++ {
		sm := model.ServerMod{
			ServerID: serverID, ModID: fmt.Sprintf("mod-%d", idx), VersionID: "v1",
			Filename: fmt.Sprintf("mod-%d.jar", idx), Status: "installed", Enabled: true,
			InstalledAt: now, UpdatedAt: now,
		}
		if err := model.DB.Create(&sm).Error; err != nil {
			t.Fatal(err)
		}
		idx++
	}
	for i := 0; i < disabledN; i++ {
		sm := model.ServerMod{
			ServerID: serverID, ModID: fmt.Sprintf("mod-%d", idx), VersionID: "v1",
			Filename: fmt.Sprintf("mod-%d.jar", idx), Status: "disabled", Enabled: false,
			InstalledAt: now, UpdatedAt: now,
		}
		if err := model.DB.Create(&sm).Error; err != nil {
			t.Fatal(err)
		}
		idx++
	}
}

func TestModLimitUnderMax(t *testing.T) {
	setupCreateLimitDB(t)
	migrateServerMods(t)
	user := createLimitUser(t, common.RoleGuestUser)
	seedOwnedServer(t, user.ID, "s1", 256*common.MB, 512*common.MB)
	perm, ok := common.PermFor(common.RoleGuestUser)
	if !ok {
		t.Fatal("guest permission missing")
	}
	seedServerMods(t, "s1", int(perm.MaxModsCount)-1, 0)
	status, body, blocked := modLimitResponse(user.ID, "s1")
	if blocked {
		t.Fatalf("under limit should allow: status=%d body=%v", status, body)
	}
}

func TestAddModLimitAtMaxIncludesDisabled(t *testing.T) {
	setupCreateLimitDB(t)
	migrateServerMods(t)
	user := createLimitUser(t, common.RoleGuestUser)
	seedOwnedServer(t, user.ID, "s1", 256*common.MB, 512*common.MB)
	perm, ok := common.PermFor(common.RoleGuestUser)
	if !ok {
		t.Fatal("guest permission missing")
	}
	// disabled mods count toward the limit
	seedServerMods(t, "s1", int(perm.MaxModsCount)-1, 1)

	sc := &ServerController{}
	r := testUIDRouter(t, "/mod/add/:server_id", sc.AddMod)
	w := postJSONAs(r, "/mod/add/s1", user.ID, map[string]any{"mod_id": "extra-mod"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["code"] != "PERM_MOD_LIMIT" {
		t.Fatalf("code=%v", body["code"])
	}
	errMsg, _ := body["error"].(string)
	if !strings.Contains(errMsg, "已達 mod 數量上限") {
		t.Fatalf("error=%q", errMsg)
	}
}

func TestModLimitRootUnlimited(t *testing.T) {
	setupCreateLimitDB(t)
	migrateServerMods(t)
	user := createLimitUser(t, common.RoleRootUser)
	seedOwnedServer(t, user.ID, "s1", 1*common.GB, 2*common.GB)
	seedServerMods(t, "s1", 50, 50)
	_, body, blocked := modLimitResponse(user.ID, "s1")
	if blocked {
		t.Fatalf("root should allow: body=%v", body)
	}
}
