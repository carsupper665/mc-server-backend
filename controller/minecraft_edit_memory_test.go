package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go-backend/common"
	"go-backend/model"
)

func postJSONAs(r http.Handler, path string, uid uint, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-UID", fmt.Sprintf("%d", uid))
	r.ServeHTTP(w, req)
	return w
}

func testUIDRouter(t *testing.T, path string, h gin.HandlerFunc) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), common.RequestIdKey, "test-request"))
		uid := uint(1)
		if raw := c.GetHeader("X-Test-UID"); raw != "" {
			var parsed uint64
			fmt.Sscanf(raw, "%d", &parsed)
			uid = uint(parsed)
		}
		c.Set("uintId", uid)
		c.Set("stringId", fmt.Sprintf("%d", uid))
		c.Set("payload", map[string]any{})
	})
	r.POST(path, h)
	return r
}

func seedOwnedServer(t *testing.T, ownerID uint, serverID string, xms, xmx int64) {
	t.Helper()
	srv := model.UserMinecraftServer{
		OwnerID: ownerID, DisplayName: "s", ServerID: serverID, SystemPath: "/tmp",
		Xms: xms, Xmx: xmx,
	}
	if err := model.DB.Create(&srv).Error; err != nil {
		t.Fatal(err)
	}
}

func TestEditServerMemoryAPIOmitsXms(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, common.RoleMemberUser)
	seedOwnedServer(t, user.ID, "s1", 1*common.GB, 2*common.GB)
	r := testUIDRouter(t, "/memory/:server_id", EditServerMemory)

	w := postJSONAs(r, "/memory/s1", user.ID, map[string]any{"xmx": "3G"})
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	got, err := model.GetServerByID(user.ID, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Xmx != 3*common.GB {
		t.Fatalf("xmx=%d", got.Xmx)
	}
	if got.Xms != 1*common.GB {
		t.Fatalf("xms should stay %d, got %d", 1*common.GB, got.Xms)
	}
	if !strings.Contains(w.Body.String(), "下次啟動生效") {
		t.Fatalf("body=%s", w.Body.String())
	}

	w = postJSONAs(r, "/memory/s1", user.ID, map[string]any{"xmx": "3G", "xms": "512M"})
	if w.Code != 200 {
		t.Fatalf("with xms: status=%d body=%s", w.Code, w.Body.String())
	}
	got, err = model.GetServerByID(user.ID, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Xms != 512*common.MB {
		t.Fatalf("xms=%d, want %d", got.Xms, 512*common.MB)
	}
}

func TestEditServerMemoryAPIRejectsZeroXmx(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, common.RoleMemberUser)
	seedOwnedServer(t, user.ID, "s1", 1*common.GB, 2*common.GB)
	r := testUIDRouter(t, "/memory/:server_id", EditServerMemory)

	w := postJSONAs(r, "/memory/s1", user.ID, map[string]any{"xmx": "0"})
	if w.Code != 400 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Xmx 格式無效") {
		t.Fatalf("body=%s", w.Body.String())
	}
}

func TestEditServerMemoryAPIRejectsXmsGreaterThanXmx(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, common.RoleMemberUser)
	seedOwnedServer(t, user.ID, "s1", 1*common.GB, 2*common.GB)
	r := testUIDRouter(t, "/memory/:server_id", EditServerMemory)

	w := postJSONAs(r, "/memory/s1", user.ID, map[string]any{"xmx": "1G", "xms": "2G"})
	if w.Code != 400 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	// omitted xms: existing Xms > new Xmx
	w = postJSONAs(r, "/memory/s1", user.ID, map[string]any{"xmx": "512M"})
	if w.Code != 400 {
		t.Fatalf("omit xms over new xmx: status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestEditServerMemoryAPIMemoryLimitAndOwnership(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, common.RoleGuestUser)
	other := createLimitUser(t, common.RoleMemberUser)
	perm, ok := common.PermFor(common.RoleGuestUser)
	if !ok {
		t.Fatal("guest permission missing")
	}
	seedOwnedServer(t, user.ID, "s1", 256*common.MB, 512*common.MB)
	r := testUIDRouter(t, "/memory/:server_id", EditServerMemory)

	over := fmt.Sprintf("%dM", perm.ServerMaxMem/common.MB+1)
	w := postJSONAs(r, "/memory/s1", user.ID, map[string]any{"xmx": over})
	if w.Code != http.StatusForbidden {
		t.Fatalf("over limit: status=%d body=%s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["code"] != "PERM_MEMORY_LIMIT" {
		t.Fatalf("code=%v", body["code"])
	}

	w = postJSONAs(r, "/memory/s1", other.ID, map[string]any{"xmx": "512M"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("other owner: status=%d body=%s", w.Code, w.Body.String())
	}
}
