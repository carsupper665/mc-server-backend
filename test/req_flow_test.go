package test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-backend/common"
	"go-backend/model"
	"go-backend/router"
)

// Exercise the real HTTP routes, JWT middleware and database in an isolated server.
// Only external artifact downloads are mocked; no local running server or email is used.
func requestFixture(t *testing.T) func(string, string, any) (int, []byte) {
	t.Helper()
	r := setupTestRouter(t)
	router.SetUserRouter(r)
	t.Cleanup(setupMockModrinth(t, "ver-123", "1.2.3", "mod.jar", "fabric", "1.20.1"))
	server := httptest.NewServer(r)
	t.Cleanup(server.Close)
	token, err := common.GenerateJWTToken(map[string]any{"user_id": "1", "Login_IP": "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	return func(method, path string, payload any) (int, []byte) {
		t.Helper()
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		req, err := http.NewRequest(method, server.URL+path, bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Mozilla/5.0 Chrome/120.0")
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set(common.DeviceHeader, "test-device")
		resp, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		return resp.StatusCode, data
	}
}

func createRequestServer(t *testing.T, request func(string, string, any) (int, []byte)) string {
	t.Helper()
	status, body := request(http.MethodPost, "/api/v1/server/create", map[string]string{"server_type": "Fabric", "server_ver": "1.20.1", "fabric_loader": "0.16.0", "fabric_installer": "1.0.1", "display_name": "req-test-server"})
	if status != 200 {
		t.Fatalf("create server HTTP %d: %s", status, body)
	}
	var result struct {
		ServerID string `json:"server_id"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.ServerID == "" {
		t.Fatalf("missing server ID: %s", body)
	}
	return result.ServerID
}

func TestReqCreateServer(t *testing.T) {
	id := createRequestServer(t, requestFixture(t))
	server, err := model.GetServerByID(1, id)
	if err != nil || server.DisplayName != "req-test-server" {
		t.Fatalf("missing created server record: %v", err)
	}
}

func TestReqQueryServer(t *testing.T) {
	request := requestFixture(t)
	id := createRequestServer(t, request)
	status, body := request(http.MethodGet, "/user/myservers", nil)
	if status != 200 {
		t.Fatalf("query servers HTTP %d: %s", status, body)
	}
	var servers []model.UserMinecraftServer
	if err := json.Unmarshal(body, &servers); err != nil {
		t.Fatal(err)
	}
	if len(servers) != 1 || servers[0].ServerID != id {
		t.Fatalf("created server not returned: %s", body)
	}
}

func TestReqAddMod(t *testing.T) {
	request := requestFixture(t)
	id := createRequestServer(t, request)
	status, body := request(http.MethodPost, "/api/v1/server/mod/add/"+id, map[string]any{"mod_id": "sodium", "version_id": "ver-123", "auto_update": true})
	if status != 200 {
		t.Fatalf("add mod HTTP %d: %s", status, body)
	}
	mod, err := model.GetServerMod(id, "sodium")
	if err != nil || mod.VersionID != "ver-123" || !mod.AutoUpdate {
		t.Fatalf("incorrect installed mod: %+v %v", mod, err)
	}
}
