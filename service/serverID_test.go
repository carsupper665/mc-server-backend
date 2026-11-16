package service

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"go-backend/common"
)

// Server ID 格式是使用者指定的（2026-11-16），不要改動；這個測試用來鎖住格式。
func TestServerIDFormatIsFixed(t *testing.T) {
	old := common.MinecraftServerPath
	common.MinecraftServerPath = t.TempDir()
	t.Cleanup(func() { common.MinecraftServerPath = old })

	cases := []struct{ serverType, want string }{
		{"Vanilla", `^mcsvv-1\.21\.4-[0-9]{4}-OID-3$`},
		{"fabric", `^mcsfv-1\.21\.4-[0-9]{4}-OID-3$`},
	}
	for _, c := range cases {
		id, dir, err := prepareServerDirectory(c.serverType, "3", "1.21.4")
		if err != nil {
			t.Fatal(err)
		}
		if !regexp.MustCompile(c.want).MatchString(id) {
			t.Fatalf("%s: server ID %q does not match %s", c.serverType, id, c.want)
		}
		if len(id) > 32 {
			t.Fatalf("server ID %q exceeds the 32-char column", id)
		}
		if dir != filepath.Join(common.MinecraftServerPath, id) {
			t.Fatalf("workDir %q does not match ID %q", dir, id)
		}
		if st, err := os.Stat(dir); err != nil || !st.IsDir() {
			t.Fatalf("workDir not created: %v", err)
		}
	}
}

func TestServerIDRetriesOnCollision(t *testing.T) {
	old := common.MinecraftServerPath
	common.MinecraftServerPath = t.TempDir()
	t.Cleanup(func() { common.MinecraftServerPath = old })

	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		id, _, err := prepareServerDirectory("Vanilla", "3", "1.21.4")
		if err != nil {
			t.Fatal(err)
		}
		if seen[id] {
			t.Fatalf("duplicate server ID %q reused an existing directory", id)
		}
		seen[id] = true
	}
}
