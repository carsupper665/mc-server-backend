package controller

import (
	"encoding/json"
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
