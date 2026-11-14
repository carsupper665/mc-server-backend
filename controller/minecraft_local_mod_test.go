package controller

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"go-backend/common"
	"go-backend/model"
	"go-backend/service"
)

func TestLocalOverrideModListToggleRemoveAndUpdate(t *testing.T) {
	setupCreateLimitDB(t)
	setupServerHandlerLogger(t)
	if err := model.DB.AutoMigrate(&model.Mod{}, &model.ModVersion{}, &model.ServerMod{}); err != nil {
		t.Fatal(err)
	}
	user := createLimitUser(t, common.RoleGuestUser)
	workDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workDir, "mods", "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	server := model.UserMinecraftServer{OwnerID: user.ID, ServerID: "s1", DisplayName: "local", SystemPath: workDir}
	if err := model.DB.Create(&server).Error; err != nil {
		t.Fatal(err)
	}
	filename := "nested/unknown.jar"
	modID := model.LocalModID(filename)
	for _, file := range []string{filename, "another.jar"} {
		if err := model.AddModToServer("s1", model.LocalModID(file), "", file, false); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(workDir, "mods", file), []byte("jar"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	sc := NewServerController(service.NewServerService(&service.ServerManager{}))
	r := testUIDRouter(t, "/list/:server_id", sc.ListMod)
	r.POST("/toggle/:server_id", sc.ToggleMod)
	r.POST("/remove/:server_id", sc.DelMod)
	r.POST("/update/:server_id", sc.UpdateMod)
	query := "?mod_id=" + url.QueryEscape(modID)
	w := postJSONAs(r, "/list/s1", user.ID, nil)
	var list struct {
		Data []struct {
			Name  string `json:"name"`
			ModID string `json:"mod_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil || w.Code != 200 || len(list.Data) != 2 {
		t.Fatalf("list: %d %s err=%v", w.Code, w.Body.String(), err)
	}
	found := false
	for _, item := range list.Data {
		if item.ModID == modID && item.Name == filename {
			found = true
		}
	}
	if !found {
		t.Fatalf("list did not show filename: %s", w.Body.String())
	}
	if w := postJSONAs(r, "/update/s1"+query, user.ID, nil); w.Code != 400 {
		t.Fatalf("update: %d %s", w.Code, w.Body.String())
	}
	jarPath := filepath.Join(workDir, "mods", filename)
	if _, err := os.Stat(jarPath); err != nil {
		t.Fatalf("update changed file: %v", err)
	}
	for _, enabled := range []bool{false, true, false} {
		w := postJSONAs(r, "/toggle/s1"+query, user.ID, gin.H{})
		if w.Code != 200 {
			t.Fatalf("toggle: %d %s", w.Code, w.Body.String())
		}
		mod, err := model.GetServerMod("s1", modID)
		expectedFile := filename
		if !enabled {
			expectedFile += ".disable"
		}
		if err != nil || mod.Enabled != enabled || mod.Filename != expectedFile {
			t.Fatalf("toggle record=%+v err=%v", mod, err)
		}
		if _, err := os.Stat(filepath.Join(workDir, "mods", expectedFile)); err != nil {
			t.Fatal(err)
		}
		if count, err := model.GetModCount("s1"); err != nil || count != 2 {
			t.Fatalf("disabled mod quota: %d %v", count, err)
		}
	}
	if w := postJSONAs(r, "/remove/s1"+query, user.ID, nil); w.Code != 200 {
		t.Fatalf("remove: %d %s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(jarPath + ".disable"); !os.IsNotExist(err) {
		t.Fatalf("disabled jar not removed: %v", err)
	}
	if count, err := model.GetModCount("s1"); err != nil || count != 1 {
		t.Fatalf("remove quota: %d %v", count, err)
	}
	if _, err := os.Stat(filepath.Join(workDir, "mods", "another.jar")); err != nil {
		t.Fatalf("other jar affected: %v", err)
	}
}
