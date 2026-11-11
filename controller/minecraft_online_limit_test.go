package controller

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"go-backend/common"
	"go-backend/service"
)

func TestOnlineLimitUnderMax(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, common.RoleMemberUser)
	status, body, blocked := onlineLimitResponse(user.ID, 0)
	if blocked {
		t.Fatalf("under limit should allow: status=%d body=%v", status, body)
	}
}

func TestOnlineLimitAtMax(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, common.RoleMemberUser)
	perm, ok := common.PermFor(common.RoleMemberUser)
	if !ok {
		t.Fatal("member permission missing")
	}
	status, body, blocked := onlineLimitResponse(user.ID, int(perm.MaxOnlineCount))
	if !blocked || status != http.StatusForbidden {
		t.Fatalf("at limit: status=%d blocked=%v body=%v", status, blocked, body)
	}
	if body["code"] != "PERM_ONLINE_LIMIT" {
		t.Fatalf("code=%v", body["code"])
	}
	errMsg, _ := body["error"].(string)
	if !strings.Contains(errMsg, "已達同時運行上限") {
		t.Fatalf("error=%q", errMsg)
	}
}

func TestOnlineLimitRootUnlimited(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, common.RoleRootUser)
	_, body, blocked := onlineLimitResponse(user.ID, 50)
	if blocked {
		t.Fatalf("root should allow: body=%v", body)
	}
}

func TestOnlineLimitUndefinedRole(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, 4)
	status, body, blocked := onlineLimitResponse(user.ID, 0)
	if !blocked || status != http.StatusForbidden {
		t.Fatalf("undefined role: status=%d blocked=%v body=%v", status, blocked, body)
	}
	if body["code"] == "PERM_ONLINE_LIMIT" {
		t.Fatal("undefined role must not use PERM_ONLINE_LIMIT")
	}
}

func TestStartFailureResponseCodes(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{service.ErrAlreadyRunning, http.StatusConflict, ""},
		{service.ErrNotFound, http.StatusNotFound, ""},
		{errors.New("boom"), http.StatusInternalServerError, ""},
	}
	for _, tc := range cases {
		status, body := startFailureResponse(tc.err)
		if status != tc.status {
			t.Fatalf("err=%v status=%d want %d", tc.err, status, tc.status)
		}
		if tc.code != "" && body["code"] != tc.code {
			t.Fatalf("err=%v code=%v want %s", tc.err, body["code"], tc.code)
		}
	}
}
