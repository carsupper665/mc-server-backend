package controller

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"go-backend/common"
	"go-backend/model"
	"go-backend/service"
)

func modpackWithOverrides(t *testing.T, jars, others int) []byte {
	t.Helper()
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	f, _ := z.Create("modrinth.index.json")
	f.Write([]byte(`{"formatVersion":1,"game":"minecraft","name":"p","versionId":"1","dependencies":{"minecraft":"1.21.6","fabric-loader":"0.18.1"},"files":[]}`))
	for i := 0; i < jars; i++ {
		f, _ = z.Create(fmt.Sprintf("overrides/mods/m%d.jar", i))
		f.Write([]byte("j"))
	}
	for i := 0; i < others; i++ {
		f, _ = z.Create(fmt.Sprintf("overrides/resourcepacks/r%d.zip", i))
		f.Write([]byte("r"))
	}
	z.Close()
	return buf.Bytes()
}

func postRawAs(h http.Handler, path string, uid uint, body []byte) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("X-Test-UID", fmt.Sprintf("%d", uid))
	h.ServeHTTP(w, req)
	return w
}

func TestImportModpackModLimit(t *testing.T) {
	setupCreateLimitDB(t)
	previousPath := common.MinecraftServerPath
	common.MinecraftServerPath = t.TempDir()
	t.Cleanup(func() {
		service.ShutdownModpackSessions()
		common.MinecraftServerPath = previousPath
	})
	member := createLimitUser(t, common.RoleMemberUser)
	perm, _ := common.PermFor(common.RoleMemberUser)
	max := int(perm.MaxModsCount)
	r := testUIDRouter(t, "/import", ImportModpack)

	post := func(pack []byte) (int, map[string]any) {
		w := postRawAs(r, "/import", member.ID, pack)
		var body map[string]any
		json.Unmarshal(w.Body.Bytes(), &body)
		return w.Code, body
	}

	code, body := post(modpackWithOverrides(t, max+1, 0))
	if code != http.StatusForbidden || body["code"] != "PERM_MOD_LIMIT" {
		t.Fatalf("code=%d body=%v", code, body)
	}
	if body["mod_count"] != float64(max+1) || body["max_mods"] != float64(max) {
		t.Fatalf("counts missing: %v", body)
	}
	entries, _ := os.ReadDir(common.MinecraftServerPath)
	var servers int64
	model.DB.Model(&model.UserMinecraftServer{}).Count(&servers)
	if len(entries) != 0 || servers != 0 {
		t.Fatalf("rejected pack left dirs=%d records=%d", len(entries), servers)
	}

	// Non-jar files do not count toward the limit.
	code, body = post(modpackWithOverrides(t, max, 20))
	if code != http.StatusOK {
		t.Fatalf("code=%d body=%v", code, body)
	}
}

func TestModpackModLimitRootUnlimited(t *testing.T) {
	setupCreateLimitDB(t)
	root := createLimitUser(t, common.RoleRootUser)
	if _, body, blocked := modpackModLimitResponse(root.ID, 100000); blocked {
		t.Fatalf("root should allow: %v", body)
	}
}
