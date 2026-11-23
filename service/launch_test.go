package service

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"go-backend/common"
)

func TestBuildLaunchArgs(t *testing.T) {
	cases := []struct {
		name   string
		launch Launch
		want   []string
	}{
		{"legacy record defaults to server.jar", Launch{MaxMem: "2048M", MinMem: "1024M"},
			[]string{"-Xms1024M", "-Xmx2048M", "-jar", "server.jar"}},
		{"jar with JVM args", Launch{Mode: LaunchJar, File: "forge-1.16.5-36.2.39.jar", MaxMem: "4G", MinMem: "1G", JVMArgs: []string{"-XX:+UseG1GC"}},
			[]string{"-Xms1G", "-Xmx4G", "-XX:+UseG1GC", "-jar", "forge-1.16.5-36.2.39.jar"}},
		{"args file", Launch{Mode: LaunchArgs, File: "libraries/net/neoforged/neoforge/21.1.77/win_args.txt", MaxMem: "4G", MinMem: "1G"},
			[]string{"-Xms1G", "-Xmx4G", "@libraries/net/neoforged/neoforge/21.1.77/win_args.txt", "nogui"}},
		{"args mode without file falls back to jar", Launch{Mode: LaunchArgs, MaxMem: "1G", MinMem: "1G"},
			[]string{"-Xms1G", "-Xmx1G", "-jar", "server.jar"}},
	}
	for _, c := range cases {
		if got := BuildLaunchArgs(c.launch); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %v", c.name, got)
		}
		if strings.Contains(strings.Join(BuildLaunchArgs(c.launch), " "), "--port") {
			t.Errorf("%s: port must come from server.properties", c.name)
		}
	}
}

func writeFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectLaunch(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "libraries/net/neoforged/neoforge/21.1.77/win_args.txt"), "-cp x")
	writeFile(t, filepath.Join(dir, "libraries/net/neoforged/neoforge/21.1.77/unix_args.txt"), "-cp x")
	for goos, want := range map[string]string{"windows": "win_args.txt", "linux": "unix_args.txt"} {
		mode, file, err := DetectLaunch(dir, goos)
		if err != nil || mode != LaunchArgs || file != "libraries/net/neoforged/neoforge/21.1.77/"+want {
			t.Errorf("%s: %s %s %v", goos, mode, file, err)
		}
	}

	legacy := t.TempDir()
	writeFile(t, filepath.Join(legacy, "forge-1.12.2-14.23.5.2859-installer.jar"), "")
	writeFile(t, filepath.Join(legacy, "forge-1.12.2-14.23.5.2859-universal.jar"), "")
	writeFile(t, filepath.Join(legacy, "forge-1.12.2-14.23.5.2859.jar"), "")
	writeFile(t, filepath.Join(legacy, "minecraft_server.1.12.2.jar"), "")
	mode, file, err := DetectLaunch(legacy, "windows")
	if err != nil || mode != LaunchJar || file != "forge-1.12.2-14.23.5.2859.jar" {
		t.Fatalf("legacy: %s %s %v", mode, file, err)
	}

	if _, _, err := DetectLaunch(t.TempDir(), "windows"); !errors.Is(err, ErrNoLaunchFile) {
		t.Fatalf("empty dir: %v", err)
	}
}

func TestUpdatePropertyWritesPortKeepingOtherSettings(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "server.properties"), "#comment\r\nmotd=Hello\r\nserver-port=25565\r\nmax-players=5\r\n")
	if err := UpdateProperty(dir, "server-port", "25570"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "server.properties"))
	got := string(data)
	if !strings.Contains(got, "server-port=25570\n") || strings.Contains(got, "25565") ||
		!strings.Contains(got, "motd=Hello") || !strings.Contains(got, "max-players=5") || !strings.Contains(got, "#comment") {
		t.Fatalf("server.properties: %q", got)
	}

	missing := t.TempDir()
	if err := UpdateProperty(missing, "server-port", "25571"); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(filepath.Join(missing, "server.properties"))
	if !strings.Contains(string(data), "server-port=25571") {
		t.Fatalf("new server.properties: %q", data)
	}
}

// Start writes the allocated port to server.properties and launches without --port.
func TestServerStartWritesPortToProperties(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a shell script stand-in for java")
	}
	oldDir, oldLogger := *common.LogDir, common.Logger
	*common.LogDir = t.TempDir()
	if err := common.IntiLogger("launch-test"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = common.Logger.Close(); *common.LogDir, common.Logger = oldDir, oldLogger })

	workDir := t.TempDir()
	writeFile(t, filepath.Join(workDir, "server.properties"), "motd=Hi\nserver-port=1\n")
	java := filepath.Join(t.TempDir(), "java")
	// Record arguments, then stay alive until the test process exits.
	writeFile(t, java, "#!/bin/sh\necho \"$@\" > args.txt\nparent=$PPID\nwhile kill -0 \"$parent\" 2>/dev/null; do sleep 0.2; done\n")
	if err := os.Chmod(java, 0o700); err != nil {
		t.Fatal(err)
	}
	mgr := NewServerManager([]int{25599})
	if _, err := mgr.StartServer("sid", "1", workDir, Launch{Java: java, Mode: LaunchArgs, File: "libraries/x/win_args.txt", MaxMem: "1G", MinMem: "1G"}); err != nil {
		t.Fatal(err)
	}
	var args []byte
	for i := 0; i < 50 && len(args) == 0; i++ {
		time.Sleep(20 * time.Millisecond)
		args, _ = os.ReadFile(filepath.Join(workDir, "args.txt"))
	}
	if strings.TrimSpace(string(args)) != "-Xms1G -Xmx1G @libraries/x/win_args.txt nogui" {
		t.Fatalf("args: %q", args)
	}
	props, _ := os.ReadFile(filepath.Join(workDir, "server.properties"))
	if !strings.Contains(string(props), "server-port=25599") || !strings.Contains(string(props), "motd=Hi") {
		t.Fatalf("server.properties: %q", props)
	}
}
