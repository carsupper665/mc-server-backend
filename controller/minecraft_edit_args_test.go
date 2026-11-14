package controller

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"go-backend/common"
	"go-backend/model"
)

func TestEditServerArgsRootCanSet(t *testing.T) {
	setupCreateLimitDB(t)
	root := createLimitUser(t, common.RoleRootUser)
	seedOwnedServer(t, root.ID, "s1", 1*common.GB, 2*common.GB)
	r := testUIDRouter(t, "/args/:server_id", EditServerArgs)

	w := postJSONAs(r, "/args/s1", root.ID, map[string]any{"args": "-XX:+UseG1GC"})
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	got, err := model.GetServerByID(root.ID, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Args != "-XX:+UseG1GC" {
		t.Fatalf("args=%q", got.Args)
	}
	if !strings.Contains(w.Body.String(), "下次啟動生效") {
		t.Fatalf("body=%s", w.Body.String())
	}
}

func TestEditServerArgsNonRootForbidden(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, common.RoleMemberUser)
	seedOwnedServer(t, user.ID, "s1", 1*common.GB, 2*common.GB)
	r := testUIDRouter(t, "/args/:server_id", EditServerArgs)

	w := postJSONAs(r, "/args/s1", user.ID, map[string]any{"args": "-XX:+UseG1GC"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["code"] != "PERM_EDIT_ARGS" {
		t.Fatalf("code=%v", body["code"])
	}
	got, err := model.GetServerByID(user.ID, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Args != "" {
		t.Fatalf("args should stay empty, got %q", got.Args)
	}
}

func TestEditServerArgsRejectsOver1000Runes(t *testing.T) {
	setupCreateLimitDB(t)
	root := createLimitUser(t, common.RoleRootUser)
	seedOwnedServer(t, root.ID, "s1", 1*common.GB, 2*common.GB)
	r := testUIDRouter(t, "/args/:server_id", EditServerArgs)

	args := strings.Repeat("あ", 1001)
	w := postJSONAs(r, "/args/s1", root.ID, map[string]any{"args": args})
	if w.Code != 400 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestEditServerArgsOwnershipSameAsDetails(t *testing.T) {
	setupCreateLimitDB(t)
	root := createLimitUser(t, common.RoleRootUser)
	other := createLimitUser(t, common.RoleMemberUser)
	seedOwnedServer(t, other.ID, "s1", 1*common.GB, 2*common.GB)
	r := testUIDRouter(t, "/args/:server_id", EditServerArgs)

	w := postJSONAs(r, "/args/s1", root.ID, map[string]any{"args": "-Xlog:gc"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
