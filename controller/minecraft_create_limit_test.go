package controller

import (
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"go-backend/common"
	"go-backend/model"
	"gorm.io/gorm"
)

func setupCreateLimitDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
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
