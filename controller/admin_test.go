package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go-backend/common"
	"go-backend/model"
	"gorm.io/gorm"
)

func setupAdminUserDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	previous := model.DB
	oldLogDir, oldLogger := *common.LogDir, common.Logger
	*common.LogDir = t.TempDir()
	if err := common.IntiLogger("admin-test"); err != nil {
		t.Fatal(err)
	}
	testLogger := common.Logger
	model.DB = db
	t.Cleanup(func() {
		_ = testLogger.Close()
		model.DB = previous
		*common.LogDir = oldLogDir
		common.Logger = oldLogger
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
}

func postJSON(r http.Handler, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestNewUserRoleValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupAdminUserDB(t)
	r := gin.New()
	r.POST("/register", NewUser)

	guest := 0
	w := postJSON(r, "/register", map[string]any{
		"username": "guest0", "display_name": "Guest", "email": "guest0@example.test",
		"password": "password1", "role": guest,
	})
	if w.Code != 200 {
		t.Fatalf("create guest: %d %s", w.Code, w.Body.String())
	}
	var created model.User
	if err := model.DB.Where("username = ?", "guest0").First(&created).Error; err != nil {
		t.Fatal(err)
	}
	if created.Role != common.RoleGuestUser {
		t.Fatalf("role=%d", created.Role)
	}
	if created.CreatedAt.IsZero() {
		t.Fatal("CreatedAt is zero")
	}

	for _, role := range []int{common.RoleExternalMemberUser, 4, 99} {
		w := postJSON(r, "/register", map[string]any{
			"username": "bad", "display_name": "Bad", "email": "bad@example.test",
			"password": "password1", "role": role,
		})
		if w.Code != 400 {
			t.Fatalf("role %d: want 400, got %d %s", role, w.Code, w.Body.String())
		}
	}
}

func TestEditUserRoleValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupAdminUserDB(t)
	user := model.User{Username: "u1", DisplayName: "U1", Email: "u1@example.test", Password: "h", Salt: "s", Role: common.RoleMemberUser}
	if err := model.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.POST("/edit", EditUser)

	guest := common.RoleGuestUser
	w := postJSON(r, "/edit", map[string]any{"user_id": user.ID, "new_role": guest})
	if w.Code != 200 {
		t.Fatalf("edit guest: %d %s", w.Code, w.Body.String())
	}
	var updated model.User
	if err := model.DB.First(&updated, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.Role != common.RoleGuestUser {
		t.Fatalf("role=%d", updated.Role)
	}

	for _, role := range []int{common.RoleExternalMemberUser, 4, 99} {
		w := postJSON(r, "/edit", map[string]any{"user_id": user.ID, "new_role": role})
		if w.Code != 400 {
			t.Fatalf("role %d: want 400, got %d %s", role, w.Code, w.Body.String())
		}
	}
	if err := model.DB.First(&updated, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.Role != common.RoleGuestUser {
		t.Fatalf("rejected edit changed role to %d", updated.Role)
	}
}

func TestGetAllUserHidesSecretsAndMarksFGF(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupAdminUserDB(t)
	token, subject := "0123456789abcdef0123456789abcdef", "fgf-subject"
	local := model.User{Username: "local", DisplayName: "L", Email: "l@example.test", Password: "h", Salt: "s", VerificationCode: "123456", AccessToken: &token, Role: common.RoleMemberUser}
	fgf := model.User{Username: "fgf", DisplayName: "F", Email: "f@example.test", Password: "h", Salt: "s", FGFSubject: &subject, Role: common.RoleMemberUser}
	for _, u := range []*model.User{&local, &fgf} {
		if err := model.DB.Create(u).Error; err != nil {
			t.Fatal(err)
		}
	}
	r := gin.New()
	r.GET("/users", GetAllUser)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users", nil))
	if w.Code != 200 {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	var body struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || len(body.Data) != 2 {
		t.Fatalf("bad body: %v %s", err, w.Body.String())
	}
	for _, u := range body.Data {
		for _, f := range []string{"verification_code", "access_token", "password", "salt", "FGFSubject"} {
			if _, ok := u[f]; ok {
				t.Fatalf("field %q leaked: %v", f, u)
			}
		}
		want := u["username"] == "fgf"
		if u["is_fgf"] != want {
			t.Fatalf("is_fgf for %v = %v", u["username"], u["is_fgf"])
		}
	}
}

func TestEditUserPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupAdminUserDB(t)
	if err := model.AddUser("pwuser", "pw@example.test", "PW", "oldpassword", common.RoleMemberUser); err != nil {
		t.Fatal(err)
	}
	var u model.User
	if err := model.DB.Where("username = ?", "pwuser").First(&u).Error; err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.POST("/edit", EditUser)
	valid := func(pw string) bool {
		var cur model.User
		if err := model.DB.First(&cur, u.ID).Error; err != nil {
			t.Fatal(err)
		}
		return common.ValidatePasswordAndHash(pw+cur.Salt, cur.Password)
	}

	if w := postJSON(r, "/edit", map[string]any{"user_id": u.ID, "new_email": "pw2@example.test"}); w.Code != 200 || !valid("oldpassword") {
		t.Fatalf("no new_password must keep password: %d %s", w.Code, w.Body.String())
	}
	if w := postJSON(r, "/edit", map[string]any{"user_id": u.ID, "new_password": "short"}); w.Code != 400 || !valid("oldpassword") {
		t.Fatalf("short password: %d %s", w.Code, w.Body.String())
	}
	if w := postJSON(r, "/edit", map[string]any{"user_id": u.ID, "new_password": "newpassword"}); w.Code != 200 {
		t.Fatalf("change password: %d %s", w.Code, w.Body.String())
	}
	if !valid("newpassword") || valid("oldpassword") {
		t.Fatal("password was not replaced")
	}
}

func TestEditUserRejectsFGFManagedFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupAdminUserDB(t)
	subject := "fgf-subject"
	fgf := model.User{Username: "fgf", DisplayName: "Old", Email: "f@example.test", Password: "h", Salt: "s", FGFSubject: &subject, Role: common.RoleMemberUser}
	local := model.User{Username: "local", DisplayName: "Old", Email: "l@example.test", Password: "h", Salt: "s", Role: common.RoleMemberUser}
	for _, u := range []*model.User{&fgf, &local} {
		if err := model.DB.Create(u).Error; err != nil {
			t.Fatal(err)
		}
	}
	r := gin.New()
	r.POST("/edit", EditUser)

	for _, body := range []map[string]any{
		{"user_id": fgf.ID, "new_role": common.RoleGuestUser},
		{"user_id": fgf.ID, "new_display_name": "New"},
	} {
		w := postJSON(r, "/edit", body)
		var resp map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if w.Code != 400 || resp["code"] != "ADMIN_FGF_MANAGED" {
			t.Fatalf("%v: %d %s", body, w.Code, w.Body.String())
		}
	}
	var got model.User
	if err := model.DB.First(&got, fgf.ID).Error; err != nil || got.Role != common.RoleMemberUser || got.DisplayName != "Old" {
		t.Fatalf("fgf data changed: %+v %v", got, err)
	}
	if w := postJSON(r, "/edit", map[string]any{"user_id": fgf.ID, "new_username": "fgf2"}); w.Code != 200 {
		t.Fatalf("fgf username edit: %d %s", w.Code, w.Body.String())
	}
	if w := postJSON(r, "/edit", map[string]any{"user_id": local.ID, "new_role": common.RoleGuestUser, "new_display_name": "New"}); w.Code != 200 {
		t.Fatalf("local edit: %d %s", w.Code, w.Body.String())
	}
	var gotLocal model.User
	if err := model.DB.First(&gotLocal, local.ID).Error; err != nil || gotLocal.Role != common.RoleGuestUser || gotLocal.DisplayName != "New" {
		t.Fatalf("local not updated: %+v %v", gotLocal, err)
	}
}
