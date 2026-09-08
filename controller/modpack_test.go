package controller

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go-backend/common"
	"go-backend/model"
	"go-backend/service"
	"gorm.io/gorm"
)

type importTransport func(*http.Request) (*http.Response, error)

func (f importTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestImportEndpointAndOwner(t *testing.T) {
	oldDB, oldRoot, oldClient := model.DB, common.MinecraftServerPath, http.DefaultClient
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "api.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	model.DB = db
	if err = db.AutoMigrate(&model.UserMinecraftServer{}); err != nil {
		t.Fatal(err)
	}
	common.MinecraftServerPath = t.TempDir()
	http.DefaultClient = &http.Client{Transport: importTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("server")), Header: make(http.Header)}, nil
	})}
	t.Cleanup(func() {
		service.ShutdownModpackSessions()
		model.DB, common.MinecraftServerPath, http.DefaultClient = oldDB, oldRoot, oldClient
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Authentication is supplied by the existing router middleware in production.
	r.Use(func(c *gin.Context) {
		owner := uint(1)
		if c.GetHeader("Test-Owner") == "2" {
			owner = 2
		}
		c.Set("uintId", owner)
		c.Set("stringId", "1")
		c.Set("payload", map[string]any{})
	})
	r.POST("/import", ImportModpack)
	c := &ServerController{}
	r.GET("/job/:job_id", c.GetModInstallJob)
	r.GET("/subscribe/:job_id", c.SubscribeModInstall)
	r.GET("/config", ModpackConfig)
	var body bytes.Buffer
	z := zip.NewWriter(&body)
	f, _ := z.Create("modrinth.index.json")
	io.WriteString(f, `{"formatVersion":1,"game":"minecraft","name":"API pack","versionId":"1","dependencies":{"minecraft":"1.21.6","fabric-loader":"0.18.1"},"files":[]}`)
	z.Close()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/import", &body))
	if w.Code != 200 {
		t.Fatalf("import HTTP %d: %s", w.Code, w.Body)
	}
	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	id := result["ins_ses_id"]
	if id == "" || result["server_id"] == "" {
		t.Fatal("missing session ID")
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		w = httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/job/"+id, nil))
		if w.Code != 200 {
			t.Fatalf("get job: %s", w.Body)
		}
		var snapshot service.InstallJobSnapshot
		json.Unmarshal(w.Body.Bytes(), &snapshot)
		if snapshot.Status == service.JobCompleted {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("session did not finish")
		}
		time.Sleep(10 * time.Millisecond)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/subscribe/"+id, nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"stage":"completed"`) {
		t.Fatalf("missing SSE terminal event: %s", w.Body)
	}
	for _, route := range []string{"/job/", "/subscribe/"} {
		req := httptest.NewRequest("GET", route+id, nil)
		req.Header.Set("Test-Owner", "2")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 404 {
			t.Fatalf("owner isolation HTTP %d", w.Code)
		}
	}
	t.Setenv("MRPACK_MAX_SIZE_MB", "1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/import", strings.NewReader(strings.Repeat("x", (1<<20)+1))))
	if w.Code != 413 {
		t.Fatalf("expected size limit 413, got %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/import", strings.NewReader("bad ZIP")))
	if w.Code != 400 {
		t.Fatalf("expected invalid ZIP 400, got %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/config", nil))
	if !strings.Contains(w.Body.String(), `"max_size":1048576`) {
		t.Fatal("config differs from enforced limit")
	}
}
