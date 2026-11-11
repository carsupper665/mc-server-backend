package controller

import (
	"net/http"
	"strings"
	"testing"

	"go-backend/common"
)

func TestMemoryLimitUnderMax(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, common.RoleMemberUser)
	perm, ok := common.PermFor(common.RoleMemberUser)
	if !ok {
		t.Fatal("member permission missing")
	}
	status, body, blocked := memoryLimitResponse(user.ID, perm.ServerMaxMem)
	if blocked {
		t.Fatalf("at limit should allow: status=%d body=%v", status, body)
	}
}

func TestMemoryLimitOverMax(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, common.RoleMemberUser)
	perm, ok := common.PermFor(common.RoleMemberUser)
	if !ok {
		t.Fatal("member permission missing")
	}
	status, body, blocked := memoryLimitResponse(user.ID, perm.ServerMaxMem+1)
	if !blocked || status != http.StatusForbidden {
		t.Fatalf("over limit: status=%d blocked=%v body=%v", status, blocked, body)
	}
	if body["code"] != "PERM_MEMORY_LIMIT" {
		t.Fatalf("code=%v", body["code"])
	}
	errMsg, _ := body["error"].(string)
	if !strings.Contains(errMsg, "請調小記憶體") {
		t.Fatalf("error=%q", errMsg)
	}
}

func TestMemoryLimitRootUnlimited(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, common.RoleRootUser)
	_, body, blocked := memoryLimitResponse(user.ID, 64*common.GB)
	if blocked {
		t.Fatalf("root should allow: body=%v", body)
	}
}

func TestMemoryLimitUndefinedRole(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, 4)
	status, body, blocked := memoryLimitResponse(user.ID, common.MB)
	if !blocked || status != http.StatusForbidden {
		t.Fatalf("undefined role: status=%d blocked=%v body=%v", status, blocked, body)
	}
	if body["code"] == "PERM_MEMORY_LIMIT" {
		t.Fatal("undefined role must not use PERM_MEMORY_LIMIT")
	}
}

func TestDefaultMemoryForUserGuestCaps(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, common.RoleGuestUser)
	perm, ok := common.PermFor(common.RoleGuestUser)
	if !ok {
		t.Fatal("guest permission missing")
	}
	xms, xmx, status, body := defaultMemoryForUser(user.ID)
	if status != 0 {
		t.Fatalf("guest defaults: status=%d body=%v", status, body)
	}
	if xmx != perm.ServerMaxMem {
		t.Fatalf("guest xmx=%d, want %d", xmx, perm.ServerMaxMem)
	}
	if xms > xmx {
		t.Fatalf("guest xms=%d > xmx=%d", xms, xmx)
	}
}
