package controller

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go-backend/common"
	"go-backend/model"
)

func TestImportModpackRejectsAtServerLimitBeforeUpload(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, common.RoleMemberUser)
	perm, ok := common.PermFor(common.RoleMemberUser)
	if !ok {
		t.Fatal("member permission missing")
	}
	addOwnedServers(t, user.ID, int(perm.MaxServerCount))
	previousPath := common.MinecraftServerPath
	common.MinecraftServerPath = t.TempDir()
	t.Cleanup(func() { common.MinecraftServerPath = previousPath })

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("uintId", user.ID)
		c.Set("stringId", "1")
		c.Set("payload", map[string]any{})
	})
	r.POST("/import", ImportModpack)

	w := httptest.NewRecorder()
	// Invalid body would be 400 if processed; 403 means we reject before upload handling.
	r.ServeHTTP(w, httptest.NewRequest("POST", "/import", strings.NewReader("not-a-zip")))
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", w.Code, w.Body)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["code"] != "PERM_SERVER_LIMIT" {
		t.Fatalf("code=%v body=%v", body["code"], body)
	}
	errMsg, _ := body["error"].(string)
	if !strings.Contains(errMsg, "已達伺服器數量上限") {
		t.Fatalf("error=%q", errMsg)
	}

	var count int64
	if err := model.DB.Model(&model.UserMinecraftServer{}).Where("owner_id = ?", user.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != perm.MaxServerCount {
		t.Fatalf("DB records changed: got %d want %d", count, perm.MaxServerCount)
	}
}

// Consume the final slot after ImportModpack's initial check, during body upload.
type reserveDuringUpload struct {
	io.Reader
	reserve func()
}

func (r *reserveDuringUpload) Read(p []byte) (int, error) {
	if r.reserve != nil {
		r.reserve()
		r.reserve = nil
	}
	return r.Reader.Read(p)
}

func TestImportModpackRechecksReservationsAfterUpload(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, common.RoleGuestUser)
	var body bytes.Buffer
	archive := zip.NewWriter(&body)
	index, err := archive.Create("modrinth.index.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(index, `{"formatVersion":1,"game":"minecraft","name":"race","versionId":"1","dependencies":{"minecraft":"1.21.6","fabric-loader":"0.18.1"},"files":[]}`); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	var release func()
	defer func() {
		if release != nil {
			release()
		}
	}()
	upload := &reserveDuringUpload{Reader: &body, reserve: func() {
		var status int
		var response gin.H
		release, status, response = reserveServerCreation(user.ID)
		if status != 0 {
			t.Fatalf("reserve during upload: %d %v", status, response)
		}
	}}
	r := testUIDRouter(t, "/import", ImportModpack)
	req := httptest.NewRequest(http.MethodPost, "/import", upload)
	req.Header.Set("X-Test-UID", fmt.Sprint(user.ID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 403 || !strings.Contains(w.Body.String(), "PERM_SERVER_LIMIT") {
		t.Fatalf("import: %d %s", w.Code, w.Body.String())
	}
	if count, err := model.GetServerCount(user.ID); err != nil || count != 0 {
		t.Fatalf("import persisted an extra server: %d %v", count, err)
	}
}
