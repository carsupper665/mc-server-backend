package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-backend/common"
	"go-backend/model"
	"go-backend/service"

	"github.com/gin-gonic/gin"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCreateServerUnsupportedVersionReturns400(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, common.RoleMemberUser)
	previous := createMinecraftServer
	t.Cleanup(func() { createMinecraftServer = previous })
	for err, code := range map[error]string{
		service.ErrUnsupportedVersion:                            "UNSUPPORTED_VERSION",
		fmt.Errorf("wrap: %w", service.ErrUnsupportedServerType): "UNSUPPORTED_SERVER_TYPE",
	} {
		createErr := err
		createMinecraftServer = func(context.Context, uint, service.CreateServerRequest, int64, int64) (*service.ModpackSession, error) {
			return nil, createErr
		}
		r := testUIDRouter(t, "/create", CreateServer)
		w := postJSONAs(r, "/create", user.ID, map[string]any{"server_type": "Vanilla", "server_ver": "1.2.5"})
		if w.Code != 400 || !strings.Contains(w.Body.String(), code) {
			t.Fatalf("%v: %d %s", err, w.Code, w.Body.String())
		}
	}
}

func TestStartBlockedWhileInstallingOrFailed(t *testing.T) {
	setupCreateLimitDB(t)
	setupServerHandlerLogger(t)
	user := createLimitUser(t, common.RoleRootUser)
	sc := NewServerController(service.NewServerService(service.NewServerManager([]int{25565})))
	r := testUIDRouter(t, "/start/:server_id", sc.Start)
	for status, code := range map[string]string{"queued": "SERVER_INSTALLING", "running": "SERVER_INSTALLING", "failed": "INSTALL_FAILED", "interrupted": "INSTALL_FAILED"} {
		sid := "sid-" + status
		if err := model.DB.Create(&model.UserMinecraftServer{OwnerID: user.ID, ServerID: sid, DisplayName: "s", SystemPath: t.TempDir(), InstallStatus: status}).Error; err != nil {
			t.Fatal(err)
		}
		w := postJSONAs(r, "/start/"+sid, user.ID, nil)
		if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), code) {
			t.Fatalf("%s: %d %s", status, w.Code, w.Body.String())
		}
	}
}

func TestDeleteBlockedWhileInstalling(t *testing.T) {
	setupCreateLimitDB(t)
	setupServerHandlerLogger(t)
	user := createLimitUser(t, common.RoleMemberUser)
	sc := NewServerController(service.NewServerService(service.NewServerManager([]int{25565})))
	if err := model.DB.Create(&model.UserMinecraftServer{OwnerID: user.ID, ServerID: "sid", DisplayName: "s", SystemPath: t.TempDir(), InstallStatus: "running"}).Error; err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("uintId", user.ID)
		c.Set("stringId", fmt.Sprint(user.ID))
		c.Set("payload", map[string]any{})
	})
	r.DELETE("/del/:server_id", sc.DeleteServerById)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/del/sid", nil))
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "SERVER_INSTALLING") {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}

func TestGetVersionsFabricLoaders(t *testing.T) {
	oldClient, oldTransport := http.DefaultClient, http.DefaultTransport
	tr := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://meta.fabricmc.net/v2/versions/loader/1.21.5" {
			return nil, fmt.Errorf("unexpected request: %s", req.URL)
		}
		body := `[{"loader":{"version":"0.17.0-beta.1","stable":false}},{"loader":{"version":"0.16.14","stable":true}}]`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: req}, nil
	})
	http.DefaultTransport, http.DefaultClient = tr, &http.Client{Transport: tr}
	t.Cleanup(func() { http.DefaultClient, http.DefaultTransport = oldClient, oldTransport })

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/versions/:loader", GetVersions)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/versions/fabric?mc=1.21.5", nil))
	if w.Code != 200 {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	var body struct {
		MC       string `json:"mc_version"`
		Versions []struct {
			ID     string `json:"id"`
			Stable bool   `json:"stable"`
			Latest bool   `json:"latest"`
		} `json:"versions"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.MC != "1.21.5" || len(body.Versions) != 2 || body.Versions[0].Stable || !body.Versions[1].Stable || !body.Versions[1].Latest {
		t.Fatalf("body: %s", w.Body.String())
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/versions/quilt", nil))
	if w.Code != 404 || !strings.Contains(w.Body.String(), "UNSUPPORTED_SERVER_TYPE") {
		t.Fatalf("quilt: %d %s", w.Code, w.Body.String())
	}
}
