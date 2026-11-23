package service

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go-backend/common"
	"go-backend/model"
)

func setupCreateServer(t *testing.T) {
	t.Helper()
	tr := versionTransport(t, nil)
	setupPackInstall(t, tr)
	resetVersionSources(t)
	clearJavaEnv(t)
}

// fakeInstaller replaces the Java installer run.
func fakeInstaller(t *testing.T, fn func(ctx context.Context, java, dir string, out io.Writer) error) {
	t.Helper()
	old, oldGOOS := runInstaller, installGOOS
	runInstaller = fn
	installGOOS = "windows"
	t.Cleanup(func() { runInstaller, installGOOS = old, oldGOOS })
}

func waitSession(t *testing.T, s *ModpackSession) {
	t.Helper()
	select {
	case <-s.done:
	case <-time.After(5 * time.Second):
		t.Fatal("installation did not finish")
	}
}

func TestCreateVanillaServerAsync(t *testing.T) {
	setupCreateServer(t)
	s, err := StartCreateServer(context.Background(), 7, CreateServerRequest{ServerType: "Vanilla", ServerVer: "1.21.4", DisplayName: "v"}, common.GB, 2*common.GB)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(s.ServerID, "mcsvv-1.21.4-") || !strings.HasPrefix(s.SessionID, "create-") || !IsServerInstallSession(s.SessionID) {
		t.Fatalf("ids: %s %s", s.ServerID, s.SessionID)
	}
	// The record exists (and counts) while installing.
	if n, _ := model.GetServerCount(7); n != 1 {
		t.Fatalf("server count while installing: %d", n)
	}
	waitSession(t, s)
	if s.Status != JobCompleted {
		t.Fatalf("status=%s err=%s", s.Status, s.Error)
	}
	server, err := model.GetServerByID(7, s.ServerID)
	if err != nil || server.InstallStatus != "completed" || server.MCVersion != "1.21.4" || server.LaunchMode != LaunchJar || server.LaunchFile != "server.jar" {
		t.Fatalf("record: %+v %v", server, err)
	}
	data, err := os.ReadFile(filepath.Join(server.SystemPath, "server.jar"))
	if err != nil || string(data) != vanillaJar {
		t.Fatalf("server.jar: %q %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(server.SystemPath, "eula.txt")); err != nil {
		t.Fatal("eula missing")
	}
	raw, err := GetModpackSession(s.SessionID, 7)
	if err != nil || !strings.Contains(string(raw), `"status":"completed"`) {
		t.Fatalf("session snapshot: %s %v", raw, err)
	}
}

func TestCreateServerRejectsUnsupportedVersionsSynchronously(t *testing.T) {
	setupCreateServer(t)
	cases := []CreateServerRequest{
		{ServerType: "Vanilla", ServerVer: "b1.7.3"},
		{ServerType: "Vanilla", ServerVer: "0.0.1"},
		{ServerType: "Forge", ServerVer: "1.12.1"},
		{ServerType: "Forge", ServerVer: "1.20.1", LoaderVersion: "1.0.0"},
		{ServerType: "NeoForge", ServerVer: "1.21"},
		{ServerType: "Fabric", ServerVer: "1.0"},
	}
	for _, req := range cases {
		if _, err := StartCreateServer(context.Background(), 7, req, common.GB, 2*common.GB); !errors.Is(err, ErrUnsupportedVersion) {
			t.Errorf("%+v: %v", req, err)
		}
	}
	if _, err := StartCreateServer(context.Background(), 7, CreateServerRequest{ServerType: "Quilt", ServerVer: "1.21.4"}, common.GB, 2*common.GB); !errors.Is(err, ErrUnsupportedServerType) {
		t.Errorf("quilt: %v", err)
	}
	if n, _ := model.GetServerCount(7); n != 0 {
		t.Fatalf("rejected requests created %d records", n)
	}
}

func TestCreateVanillaSha1MismatchFailsAndCleansUp(t *testing.T) {
	setupCreateServer(t)
	s, err := StartCreateServer(context.Background(), 7, CreateServerRequest{ServerType: "Vanilla", ServerVer: "1.20.1"}, common.GB, 2*common.GB)
	if err != nil {
		t.Fatal(err)
	}
	waitSession(t, s)
	if s.Status != JobFailed || !strings.Contains(s.Error, "sha1 mismatch") {
		t.Fatalf("status=%s err=%s", s.Status, s.Error)
	}
	server, err := model.GetServerByID(7, s.ServerID)
	if err != nil || server.InstallStatus != "failed" || !strings.Contains(server.InstallDetails, "sha1 mismatch") {
		t.Fatalf("failed record must be kept with its error: %+v %v", server, err)
	}
	entries, err := os.ReadDir(server.SystemPath)
	if err != nil || len(entries) != 0 {
		t.Fatalf("files not cleared: %v %v", entries, err)
	}
}

func TestCreateFabricUsesLatestStableLoader(t *testing.T) {
	setupCreateServer(t)
	common.LatestFabricInstallerVersion = "1.0.1"
	s, err := StartCreateServer(context.Background(), 7, CreateServerRequest{ServerType: "Fabric", ServerVer: "1.21.4"}, common.GB, 2*common.GB)
	if err != nil {
		t.Fatal(err)
	}
	waitSession(t, s)
	server, _ := model.GetServerByID(7, s.ServerID)
	if s.Status != JobCompleted || server.LoaderVersion != "0.16.9" || !strings.HasPrefix(server.ServerID, "mcsfv-1.21.4-") {
		t.Fatalf("status=%s loader=%s id=%s err=%s", s.Status, server.LoaderVersion, server.ServerID, s.Error)
	}
}

func TestCreateNeoForgeServerWithFakeInstaller(t *testing.T) {
	setupCreateServer(t)
	var gotJava, gotDir string
	fakeInstaller(t, func(ctx context.Context, java, dir string, out io.Writer) error {
		gotJava, gotDir = java, dir
		if _, err := os.Stat(filepath.Join(dir, installerJar)); err != nil {
			t.Error("installer jar not downloaded")
		}
		_, _ = io.WriteString(out, "installing\n")
		writeFile(t, filepath.Join(dir, "libraries/net/neoforged/neoforge/21.1.200/win_args.txt"), "-cp x")
		writeFile(t, filepath.Join(dir, installerJar+".log"), "log")
		return nil
	})
	t.Setenv("JAVA_21", "/opt/java21")
	s, err := StartCreateServer(context.Background(), 7, CreateServerRequest{ServerType: "NeoForge", ServerVer: "1.21.1", DisplayName: "neo"}, common.GB, 2*common.GB)
	if err != nil {
		t.Fatal(err)
	}
	waitSession(t, s)
	if s.Status != JobCompleted {
		t.Fatalf("status=%s err=%s", s.Status, s.Error)
	}
	server, _ := model.GetServerByID(7, s.ServerID)
	if !strings.HasPrefix(server.ServerID, "mcsnv-1.21.1-") || server.LoaderVersion != "21.1.200" || server.ModLoader != "NeoForge" {
		t.Fatalf("record: %+v", server)
	}
	if server.LaunchMode != LaunchArgs || server.LaunchFile != "libraries/net/neoforged/neoforge/21.1.200/win_args.txt" {
		t.Fatalf("launch: %s %s", server.LaunchMode, server.LaunchFile)
	}
	if gotDir != server.SystemPath {
		t.Fatalf("installer dir %s", gotDir)
	}
	if gotJava != "java" && gotJava != "/opt/java21" {
		t.Fatalf("java %s", gotJava)
	}
	for _, name := range []string{installerJar, installerJar + ".log", installerOutLog} {
		if _, err := os.Stat(filepath.Join(server.SystemPath, name)); !os.IsNotExist(err) {
			t.Errorf("%s not removed after success", name)
		}
	}
}

func TestCreateForgeInstallerFailureKeepsLog(t *testing.T) {
	setupCreateServer(t)
	fakeInstaller(t, func(ctx context.Context, java, dir string, out io.Writer) error {
		_, _ = io.WriteString(out, "boom\n")
		writeFile(t, filepath.Join(dir, "libraries/x.jar"), "partial")
		return errors.New("exit status 1")
	})
	s, err := StartCreateServer(context.Background(), 7, CreateServerRequest{ServerType: "Forge", ServerVer: "1.20.1"}, common.GB, 2*common.GB)
	if err != nil {
		t.Fatal(err)
	}
	waitSession(t, s)
	if s.Status != JobFailed || !strings.Contains(s.Error, "installer failed") {
		t.Fatalf("status=%s err=%s", s.Status, s.Error)
	}
	server, _ := model.GetServerByID(7, s.ServerID)
	if !strings.HasPrefix(server.ServerID, "mcsgv-1.20.1-") || server.LoaderVersion != "47.3.0" {
		t.Fatalf("forge must default to the recommended build: %+v", server)
	}
	data, err := os.ReadFile(filepath.Join(server.SystemPath, installerOutLog))
	if err != nil || !strings.Contains(string(data), "boom") {
		t.Fatalf("installer log not kept: %q %v", data, err)
	}
	for _, gone := range []string{"libraries", installerJar} {
		if _, err := os.Stat(filepath.Join(server.SystemPath, gone)); !os.IsNotExist(err) {
			t.Errorf("%s not cleaned up", gone)
		}
	}
}

func TestCreateForgeInstallerTimeout(t *testing.T) {
	setupCreateServer(t)
	t.Setenv("INSTALLER_TIMEOUT", "50ms")
	fakeInstaller(t, func(ctx context.Context, java, dir string, out io.Writer) error {
		<-ctx.Done()
		return ctx.Err()
	})
	s, err := StartCreateServer(context.Background(), 7, CreateServerRequest{ServerType: "Forge", ServerVer: "1.20.1", LoaderVersion: "47.4.10"}, common.GB, 2*common.GB)
	if err != nil {
		t.Fatal(err)
	}
	waitSession(t, s)
	if s.Status != JobFailed || !strings.Contains(s.Error, "timed out") {
		t.Fatalf("status=%s err=%s", s.Status, s.Error)
	}
}

func TestCreateLegacyForgeUsesJarAndVanillaJar(t *testing.T) {
	setupCreateServer(t)
	fakeInstaller(t, func(ctx context.Context, java, dir string, out io.Writer) error {
		data, err := os.ReadFile(filepath.Join(dir, "minecraft_server.1.12.2.jar"))
		if err != nil || string(data) != vanillaJar {
			t.Errorf("vanilla jar not pre-downloaded: %v", err)
		}
		writeFile(t, filepath.Join(dir, "forge-1.12.2-14.23.5.2859.jar"), "forge")
		return nil
	})
	src, err := ResolveServerSource(context.Background(), "forge", "1.12.2", "", "")
	if err != nil || src.LoaderVersion != "14.23.5.2859" ||
		src.URL != forgeMavenURL+"/1.12.2-14.23.5.2859/forge-1.12.2-14.23.5.2859-installer.jar" {
		t.Fatalf("src=%+v err=%v", src, err)
	}
	s, err := StartCreateServer(context.Background(), 7, CreateServerRequest{ServerType: "Forge", ServerVer: "1.12.2"}, common.GB, 2*common.GB)
	if err != nil {
		t.Fatal(err)
	}
	waitSession(t, s)
	server, _ := model.GetServerByID(7, s.ServerID)
	if s.Status != JobCompleted || server.LaunchMode != LaunchJar || server.LaunchFile != "forge-1.12.2-14.23.5.2859.jar" {
		t.Fatalf("status=%s err=%s launch=%s %s", s.Status, s.Error, server.LaunchMode, server.LaunchFile)
	}
}

func TestResolveNeoForgeInstallerURLs(t *testing.T) {
	setupVersionTransport(t, nil)
	src, err := ResolveServerSource(context.Background(), "NeoForge", "1.21.1", "21.1.77", "")
	if err != nil || src.URL != neoForgeMavenURL+"/21.1.77/neoforge-21.1.77-installer.jar" {
		t.Fatalf("src=%+v err=%v", src, err)
	}
	// Modrinth packs give the 1.20.1 version without the MC prefix.
	src, err = ResolveServerSource(context.Background(), "neoforge", "1.20.1", "47.1.106", "")
	if err != nil || src.URL != neoForgeLegacyMavenURL+"/1.20.1-47.1.106/forge-1.20.1-47.1.106-installer.jar" {
		t.Fatalf("1.20.1 src=%+v err=%v", src, err)
	}
	if _, err := ResolveServerSource(context.Background(), "neoforge", "1.21.1", "21.4.10-beta", ""); !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("unstable build: %v", err)
	}
}

func TestInstallerConcurrencyLimit(t *testing.T) {
	setupCreateServer(t)
	t.Setenv("INSTALLER_CONCURRENCY", "1")
	oldSlots := installerSlots.ch
	installerSlots.ch = make(chan struct{}, 1)
	t.Cleanup(func() {
		if oldSlots == nil {
			oldSlots = make(chan struct{}, 2)
		}
		installerSlots.ch = oldSlots
	})
	var running, peak atomic.Int32
	release := make(chan struct{})
	fakeInstaller(t, func(ctx context.Context, java, dir string, out io.Writer) error {
		n := running.Add(1)
		if n > peak.Load() {
			peak.Store(n)
		}
		<-release
		running.Add(-1)
		writeFile(t, filepath.Join(dir, "libraries/net/minecraftforge/forge/1.20.1-47.3.0/win_args.txt"), "")
		return nil
	})
	installerSlots.once.Do(func() {}) // keep the test channel
	var sessions []*ModpackSession
	for i := 0; i < 2; i++ {
		s, err := StartCreateServer(context.Background(), 7, CreateServerRequest{ServerType: "Forge", ServerVer: "1.20.1"}, common.GB, 2*common.GB)
		if err != nil {
			t.Fatal(err)
		}
		sessions = append(sessions, s)
	}
	time.Sleep(100 * time.Millisecond)
	close(release)
	for _, s := range sessions {
		waitSession(t, s)
		if s.Status != JobCompleted {
			t.Fatalf("status=%s err=%s", s.Status, s.Error)
		}
	}
	if peak.Load() != 1 {
		t.Fatalf("%d installers ran at once", peak.Load())
	}
}

// The real installer runner passes --installServer in the server directory.
func TestRunInstallerExecutesJava(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a shell script stand-in for java")
	}
	dir := t.TempDir()
	java := filepath.Join(t.TempDir(), "java")
	writeFile(t, java, "#!/bin/sh\necho \"$@\" > args.txt\npwd >> args.txt\n")
	if err := os.Chmod(java, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := runInstaller(context.Background(), java, dir, io.Discard); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "args.txt"))
	if !strings.HasPrefix(string(data), "-jar installer.jar --installServer\n") {
		t.Fatalf("args: %q", data)
	}
	slow := filepath.Join(t.TempDir(), "java")
	writeFile(t, slow, "#!/bin/sh\nexec sleep 10\n")
	_ = os.Chmod(slow, 0o700)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := runInstaller(ctx, slow, dir, io.Discard); err == nil || time.Since(start) > 5*time.Second {
		t.Fatalf("timeout not enforced: %v after %s", err, time.Since(start))
	}
}

func TestForgeModpackImport(t *testing.T) {
	setupCreateServer(t)
	fakeInstaller(t, func(ctx context.Context, java, dir string, out io.Writer) error {
		writeFile(t, filepath.Join(dir, "libraries/net/minecraftforge/forge/1.20.1-47.2.0/win_args.txt"), "")
		return nil
	})
	for _, c := range []struct{ dep, ver, prefix, launch string }{
		{"forge", "47.2.0", "mcsgv-1.20.1-", "libraries/net/minecraftforge/forge/1.20.1-47.2.0/win_args.txt"},
		{"neoforge", "47.1.106", "mcsnv-1.20.1-", "libraries/net/minecraftforge/forge/1.20.1-47.2.0/win_args.txt"},
	} {
		p := testPack()
		p.Files = nil
		p.Dependencies = map[string]string{"minecraft": "1.20.1", c.dep: c.ver}
		s := startPack(t, p, "overrides/config/a.txt")
		waitPack(t, s)
		if s.Status != JobCompleted {
			t.Fatalf("%s: status=%s err=%s", c.dep, s.Status, s.Error)
		}
		server, _ := model.GetServerByID(7, s.ServerID)
		if !strings.HasPrefix(server.ServerID, c.prefix) || server.LaunchMode != LaunchArgs || server.LaunchFile != c.launch || server.LoaderVersion != c.ver {
			t.Fatalf("%s: %+v", c.dep, server)
		}
		if _, err := os.Stat(filepath.Join(server.SystemPath, "config/a.txt")); err != nil {
			t.Fatalf("%s override missing", c.dep)
		}
	}
	bad := testPack()
	bad.Dependencies = map[string]string{"minecraft": "1.20.1", "forge": "1.0.0"}
	data := packBytes(t, bad)
	file, _ := os.CreateTemp(t.TempDir(), "pack-*.mrpack")
	_, _ = file.Write(data)
	defer file.Close()
	parsed, err := ParseMRPack(file, int64(len(data)), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := StartModpackInstall(7, parsed, file, common.GB, 2*common.GB); !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("unknown forge version must fail precheck: %v", err)
	}
}
