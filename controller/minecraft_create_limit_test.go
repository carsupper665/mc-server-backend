package controller

import (
	"errors"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"go-backend/common"
	"go-backend/model"
	"gorm.io/gorm"
)

func setupCreateLimitDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	previous := model.DB
	model.DB = db
	t.Cleanup(func() {
		model.DB = previous
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})
	if err := db.AutoMigrate(&model.User{}, &model.UserMinecraftServer{}); err != nil {
		t.Fatal(err)
	}
}

func createLimitUser(t *testing.T, role int) model.User {
	t.Helper()
	user := model.User{
		Username:    fmt.Sprintf("u%d-%d", role, role+1),
		DisplayName: "U",
		Email:       fmt.Sprintf("u%d@example.test", role),
		Password:    "h",
		Salt:        "s",
		Role:        role,
	}
	if err := model.CreateUser(&user); err != nil {
		t.Fatal(err)
	}
	return user
}

func addOwnedServers(t *testing.T, ownerID uint, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		srv := model.UserMinecraftServer{
			OwnerID:     ownerID,
			DisplayName: "s",
			ServerID:    fmt.Sprintf("sid%d", i),
			SystemPath:  "/tmp/s",
		}
		if err := model.DB.Create(&srv).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func TestCreateServerLimitUnderMax(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, common.RoleMemberUser)
	status, body, blocked := createServerLimitResponse(user.ID)
	if blocked {
		t.Fatalf("under limit should allow: status=%d body=%v", status, body)
	}
}

func TestCreateServerLimitAtMax(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, common.RoleMemberUser)
	perm, ok := common.PermFor(common.RoleMemberUser)
	if !ok {
		t.Fatal("member permission missing")
	}
	addOwnedServers(t, user.ID, int(perm.MaxServerCount))
	status, body, blocked := createServerLimitResponse(user.ID)
	if !blocked || status != 403 {
		t.Fatalf("at limit: status=%d blocked=%v body=%v", status, blocked, body)
	}
	if body["code"] != "PERM_SERVER_LIMIT" {
		t.Fatalf("code=%v", body["code"])
	}
	errMsg, _ := body["error"].(string)
	if !strings.Contains(errMsg, "已達伺服器數量上限") {
		t.Fatalf("error=%q", errMsg)
	}
}

func TestCreateServerLimitRootUnlimited(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, common.RoleRootUser)
	addOwnedServers(t, user.ID, 50)
	_, body, blocked := createServerLimitResponse(user.ID)
	if blocked {
		t.Fatalf("root should allow: body=%v", body)
	}
}

func TestCreateServerLimitUndefinedRole(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, 4)
	status, body, blocked := createServerLimitResponse(user.ID)
	if !blocked || status != 403 {
		t.Fatalf("undefined role: status=%d blocked=%v body=%v", status, blocked, body)
	}
	if body["code"] == "PERM_SERVER_LIMIT" {
		t.Fatal("undefined role must not use PERM_SERVER_LIMIT")
	}
}

func TestCreateServerConcurrentReservation(t *testing.T) {
	setupCreateLimitDB(t)
	sqlDB, _ := model.DB.DB()
	sqlDB.SetMaxOpenConns(1)
	user := createLimitUser(t, common.RoleGuestUser)
	started, finish := make(chan struct{}, 2), make(chan struct{})
	previous := createMinecraftServer
	createMinecraftServer = func(_, _, _, _, _ string) (string, error) {
		started <- struct{}{}
		<-finish
		return "created", nil
	}
	t.Cleanup(func() { createMinecraftServer = previous })
	r := testUIDRouter(t, "/create", CreateServer)
	first, second := make(chan *httptest.ResponseRecorder, 1), make(chan *httptest.ResponseRecorder, 1)
	go func() { first <- postJSONAs(r, "/create", user.ID, map[string]any{}) }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("first create did not start")
	}
	go func() { second <- postJSONAs(r, "/create", user.ID, map[string]any{}) }()
	select {
	case w := <-second:
		second = nil
		if w.Code != 403 || !strings.Contains(w.Body.String(), "PERM_SERVER_LIMIT") {
			t.Errorf("status=%d body=%s", w.Code, w.Body.String())
		}
	case <-started:
		t.Error("second request passed the limit while the first was creating")
	case <-time.After(5 * time.Second):
		t.Error("second request blocked behind download")
	}
	close(finish)
	if w := <-first; w.Code != 200 {
		t.Fatalf("first: status=%d body=%s", w.Code, w.Body.String())
	}
	// Drain the second request before restoring globals and closing the DB.
	if second != nil {
		<-second
	}
}

func TestCreateServerFailureReleasesReservation(t *testing.T) {
	setupCreateLimitDB(t)
	user := createLimitUser(t, common.RoleGuestUser)
	previous := createMinecraftServer
	defer func() { createMinecraftServer = previous }()
	attempts := 0
	createMinecraftServer = func(_, _, _, _, _ string) (string, error) {
		attempts++
		if attempts == 1 {
			return "", errors.New("download failed")
		}
		return "retry", nil
	}
	r := testUIDRouter(t, "/create", CreateServer)
	if w := postJSONAs(r, "/create", user.ID, map[string]any{}); w.Code != 500 {
		t.Fatalf("first: %d %s", w.Code, w.Body.String())
	}
	if w := postJSONAs(r, "/create", user.ID, map[string]any{}); w.Code != 200 {
		t.Fatalf("retry: %d %s", w.Code, w.Body.String())
	}
	if _, _, blocked := createServerLimitResponse(user.ID); !blocked {
		t.Fatal("persisted server must count after reservation released")
	}
}

func TestServerCreationReservationsArePerUserAndRootBypasses(t *testing.T) {
	setupCreateLimitDB(t)
	guest := createLimitUser(t, common.RoleGuestUser)
	member := createLimitUser(t, common.RoleMemberUser)
	root := createLimitUser(t, common.RoleRootUser)
	release, status, body := reserveServerCreation(guest.ID)
	if status != 0 {
		t.Fatalf("guest reservation: %d %v", status, body)
	}
	defer release()
	for _, uid := range []uint{member.ID, root.ID, root.ID} {
		releaseOther, status, body := reserveServerCreation(uid)
		if status != 0 {
			t.Fatalf("other user %d: %d %v", uid, status, body)
		}
		defer releaseOther()
	}
	serverCreationMu.Lock()
	rootPending := pendingServerCreations[root.ID]
	serverCreationMu.Unlock()
	if rootPending != 0 {
		t.Fatalf("root reserved %d slots", rootPending)
	}
}
