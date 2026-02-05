package test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	defaultBasePort  = "8080"
	defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

var (
	reqClient     *http.Client
	baseURL       string
	clientOnce    sync.Once
	clientInitErr error

	loginOnce sync.Once
	loginErr  error

	serverOnce         sync.Once
	serverID           string
	serverCreateStatus int
	serverCreateBody   string
	serverErr          error
)

type serverConfig struct {
	ServerType      string
	ServerVersion   string
	FabricLoader    string
	FabricInstaller string
	DisplayName     string
}

func TestReqCreateServer(t *testing.T) {
	_ = ensureServer(t)
}

func TestReqQueryServer(t *testing.T) {
	sid := ensureServer(t)

	status, body, err := doJSON(http.MethodGet, "/user/myservers", nil)
	if err != nil {
		t.Fatalf("query servers request failed: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, status, string(body))
	}

	var servers []struct {
		ServerID string `json:"server_id"`
	}
	if err := json.Unmarshal(body, &servers); err != nil {
		t.Fatalf("failed to parse myservers response: %v (%s)", err, string(body))
	}

	found := false
	for _, s := range servers {
		if s.ServerID == sid {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("created server_id %s not found in myservers response", sid)
	}
}

func TestReqAddMod(t *testing.T) {
	sid := ensureServer(t)

	cfg := getServerConfig()
	if !strings.EqualFold(cfg.ServerType, "fabric") {
		t.Fatalf("add mod requires a Fabric server, got %q (set TEST_SERVER_TYPE=Fabric)", cfg.ServerType)
	}

	modID := getEnv("TEST_MOD_ID", "sodium")
	versionID := getEnv("TEST_MOD_VERSION", "")
	autoUpdate := getEnvBool("TEST_MOD_AUTO_UPDATE", true)

	payload := map[string]any{
		"mod_id":      modID,
		"version_id":  versionID,
		"auto_update": autoUpdate,
	}
	status, body, err := doJSON(http.MethodPost, "/api/v1/server/mod/add/"+sid, payload)
	if err != nil {
		t.Fatalf("add mod request failed: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, status, string(body))
	}
}

func ensureServer(t *testing.T) string {
	t.Helper()

	ensureLogin(t)

	serverOnce.Do(func() {
		serverID, serverCreateStatus, serverCreateBody, serverErr = createServer()
	})

	if serverErr != nil {
		t.Fatalf("create server failed: %v", serverErr)
	}
	if serverCreateStatus != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, serverCreateStatus, serverCreateBody)
	}
	if serverID == "" {
		t.Fatalf("create server returned empty server_id")
	}

	return serverID
}

func ensureLogin(t *testing.T) {
	t.Helper()

	loginOnce.Do(func() {
		loginErr = loginAndVerify()
	})

	if loginErr != nil {
		t.Fatalf("login failed: %v", loginErr)
	}
}

func loginAndVerify() error {
	if err := ensureClient(); err != nil {
		return err
	}

	username := getEnv("TEST_USERNAME", "root")
	password := getEnv("TEST_PASSWORD", "123")
	email := getEnv("TEST_EMAIL", "")

	loginPayload := map[string]string{
		"password": password,
	}
	if email != "" {
		loginPayload["email"] = email
	} else {
		loginPayload["username"] = username
	}

	status, body, err := doJSON(http.MethodPost, "/Authentication/login", loginPayload)
	if err != nil {
		return err
	}

	switch status {
	case http.StatusOK:
		return nil
	case http.StatusAccepted:
		code := getEnv("TEST_EMAIL_CODE", "")
		if code == "" {
			code, err = promptVerificationCode()
			if err != nil {
				return err
			}
		}
		verifyPayload := map[string]string{"code": code}
		verifyStatus, verifyBody, verifyErr := doJSON(http.MethodPost, "/Authentication/verify", verifyPayload)
		if verifyErr != nil {
			return verifyErr
		}
		if verifyStatus != http.StatusOK {
			return fmt.Errorf("verify failed with status %d: %s", verifyStatus, string(verifyBody))
		}
		return nil
	default:
		return fmt.Errorf("login failed with status %d: %s", status, string(body))
	}
}

func createServer() (string, int, string, error) {
	cfg := getServerConfig()

	payload := map[string]string{
		"server_type":  cfg.ServerType,
		"server_ver":   cfg.ServerVersion,
		"display_name": cfg.DisplayName,
	}
	if cfg.FabricLoader != "" {
		payload["fabric_loader"] = cfg.FabricLoader
	}
	if cfg.FabricInstaller != "" {
		payload["fabric_installer"] = cfg.FabricInstaller
	}

	status, body, err := doJSON(http.MethodPost, "/api/v1/server/create", payload)
	if err != nil {
		return "", status, string(body), err
	}
	if status != http.StatusOK {
		return "", status, string(body), nil
	}

	var resp struct {
		ServerID string `json:"server_id"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", status, string(body), fmt.Errorf("failed to parse create server response: %w", err)
	}

	return resp.ServerID, status, string(body), nil
}

func doJSON(method, path string, payload any) (int, []byte, error) {
	if err := ensureClient(); err != nil {
		return 0, nil, err
	}

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, baseURL+path, body)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := reqClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	return resp.StatusCode, respBody, nil
}

func ensureClient() error {
	clientOnce.Do(func() {
		baseURL = resolveBaseURL()
		if baseURL == "" {
			clientInitErr = fmt.Errorf("base URL is empty")
			return
		}

		jar, err := cookiejar.New(nil)
		if err != nil {
			clientInitErr = err
			return
		}

		reqClient = &http.Client{
			Jar:     jar,
			Timeout: requestTimeout(),
		}
	})

	return clientInitErr
}

func resolveBaseURL() string {
	if v := getEnv("TEST_BASE_URL", ""); v != "" {
		return strings.TrimRight(v, "/")
	}

	if v := getEnv("BASE_URL", ""); v != "" {
		return strings.TrimRight(v, "/")
	}

	host := getEnv("TEST_HOST", "http://localhost")
	port := getEnv("PORT", defaultBasePort)
	host = strings.TrimRight(host, "/")

	return fmt.Sprintf("%s:%s", host, port)
}

func requestTimeout() time.Duration {
	if v := getEnv("REQ_TEST_TIMEOUT_SECONDS", ""); v != "" {
		if seconds, err := strconv.Atoi(v); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return 10 * time.Minute
}

func getServerConfig() serverConfig {
	serverType := getEnv("TEST_SERVER_TYPE", "Fabric")
	serverVer := getEnv("TEST_SERVER_VER", "")
	if serverVer == "" {
		serverVer = getEnv("TEST_MC_VERSION", "1.20.1")
	}

	return serverConfig{
		ServerType:      serverType,
		ServerVersion:   serverVer,
		FabricLoader:    getEnv("TEST_FABRIC_LOADER", getEnv("LATEST_FABRIC_LOADER_VERSION", "")),
		FabricInstaller: getEnv("TEST_FABRIC_INSTALLER", getEnv("LATEST_FABRIC_INSTALLER_VERSION", "")),
		DisplayName:     getEnv("TEST_SERVER_NAME", "req-test-server"),
	}
}

func promptVerificationCode() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter email verification code: ")
	code, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(code), nil
}

func getEnvBool(key string, fallback bool) bool {
	if v := getEnv(key, ""); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func getEnv(key, fallback string) string {
	val := strings.TrimSpace(os.Getenv(key))
	val = strings.Trim(val, "\"")
	val = strings.Trim(val, "'")
	if val == "" {
		return fallback
	}
	return val
}
