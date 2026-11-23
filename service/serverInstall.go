package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-backend/common"
	"go-backend/model"
)

// ServerSource is a resolved, validated description of what to install.
type ServerSource struct {
	Type          string // vanilla / fabric / forge / neoforge
	MCVersion     string
	LoaderVersion string
	URL           string // server jar (vanilla/fabric) or installer jar (forge/neoforge)
	SHA1          string
	Installer     bool
	// Legacy Forge (< 1.17) needs the vanilla jar next to the installer.
	VanillaURL  string
	VanillaSHA1 string
}

// InstallResult is how the installed server must be launched.
type InstallResult struct {
	Mode string
	File string
}

// ResolveServerSource validates the server type and versions and finds the download.
// Only Vanilla needs the network unconditionally; Fabric with an explicit loader does not.
func ResolveServerSource(ctx context.Context, serverType, mc, loaderVer, fabricInstaller string) (*ServerSource, error) {
	t := normalizeServerType(serverType)
	mc = strings.TrimSpace(mc)
	loaderVer = strings.TrimSpace(loaderVer)
	if mc == "" {
		return nil, ErrUnsupportedVersion
	}
	switch t {
	case Vanilla:
		url, sha1, err := vanillaServerDownload(ctx, mc)
		if err != nil {
			return nil, err
		}
		return &ServerSource{Type: t, MCVersion: mc, URL: url, SHA1: sha1}, nil
	case Fabric:
		if loaderVer == "" {
			builds, err := ListFabricLoaders(ctx, mc)
			if err == nil {
				if len(builds) == 0 {
					return nil, ErrUnsupportedVersion
				}
				loaderVer = builds[0].ID
				for _, b := range builds {
					if b.Latest {
						loaderVer = b.ID
						break
					}
				}
			} else {
				loaderVer = common.LatestFabricLoaderVersion
			}
			if loaderVer == "" {
				return nil, fmt.Errorf("fabric loader unavailable: %w", err)
			}
		}
		installer := strings.TrimSpace(fabricInstaller)
		if installer == "" {
			installer = common.LatestFabricInstallerVersion
		}
		url := fmt.Sprintf("%s/versions/loader/%s/%s/%s/server/jar", fabricMetaURL, mc, loaderVer, installer)
		return &ServerSource{Type: t, MCVersion: mc, LoaderVersion: loaderVer, URL: url}, nil
	case Forge, NeoForge:
		return resolveInstallerSource(ctx, t, mc, loaderVer)
	}
	return nil, ErrUnsupportedServerType
}

func resolveInstallerSource(ctx context.Context, t, mc, loaderVer string) (*ServerSource, error) {
	if t == Forge && compareVersions(mc, forgeMinMC) < 0 {
		return nil, ErrUnsupportedVersion
	}
	if t == NeoForge && mc == "1.20.1" {
		loaderVer = strings.TrimPrefix(loaderVer, "1.20.1-")
	}
	var build *LoaderBuild
	index, err := loaderIndex(ctx, t)
	if err == nil {
		builds := index[mc]
		if len(builds) == 0 {
			return nil, ErrUnsupportedVersion
		}
		build = pickBuild(builds, loaderVer)
		if build == nil {
			return nil, ErrUnsupportedVersion
		}
	} else {
		// Without the list only an explicit version can be installed.
		if loaderVer == "" {
			return nil, fmt.Errorf("%s versions unavailable: %w", t, err)
		}
		build = fallbackBuild(t, mc, loaderVer)
	}
	src := &ServerSource{
		Type: t, MCVersion: mc, LoaderVersion: build.ID, Installer: true,
		URL: fmt.Sprintf("%s/%s/%s-%s-installer.jar", build.baseURL, build.full, build.artifact, build.full),
	}
	if t == Forge && compareVersions(mc, "1.17") < 0 {
		// Best effort: old installers download the vanilla jar from retired URLs.
		if url, sha1, err := vanillaServerDownload(ctx, mc); err == nil {
			src.VanillaURL, src.VanillaSHA1 = url, sha1
		}
	}
	return src, nil
}

// pickBuild returns the requested build, or recommended → latest → first stable.
func pickBuild(builds []LoaderBuild, want string) *LoaderBuild {
	if want != "" {
		for i := range builds {
			if builds[i].ID == want {
				return &builds[i]
			}
		}
		return nil
	}
	for _, pred := range []func(LoaderBuild) bool{
		func(b LoaderBuild) bool { return b.Recommended },
		func(b LoaderBuild) bool { return b.Latest },
		func(b LoaderBuild) bool { return b.Stable },
	} {
		for i := range builds {
			if pred(builds[i]) {
				return &builds[i]
			}
		}
	}
	return &builds[0]
}

func fallbackBuild(t, mc, v string) *LoaderBuild {
	switch {
	case t == Forge:
		return &LoaderBuild{ID: v, MCVersion: mc, full: mc + "-" + v, baseURL: forgeMavenURL, artifact: "forge"}
	case mc == "1.20.1":
		return &LoaderBuild{ID: v, MCVersion: mc, full: mc + "-" + v, baseURL: neoForgeLegacyMavenURL, artifact: "forge"}
	}
	return &LoaderBuild{ID: v, MCVersion: mc, full: v, baseURL: neoForgeMavenURL, artifact: "neoforge"}
}

// ---------------- installation ----------------

const (
	installerJar    = "installer.jar"
	installerOutLog = "installer-output.log"
)

// installerLogs survive a failed installation so the cause can be inspected.
var installerLogs = []string{installerOutLog, installerJar + ".log", "installer.log"}

func installerTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv("INSTALLER_TIMEOUT"))
	if d, err := time.ParseDuration(raw); err == nil && d > 0 {
		return d
	}
	if n, err := strconv.Atoi(raw); err == nil && n > 0 {
		return time.Duration(n) * time.Second
	}
	return 10 * time.Minute
}

var installerSlots = struct {
	once sync.Once
	ch   chan struct{}
}{}

func acquireInstallerSlot(ctx context.Context) (func(), error) {
	installerSlots.once.Do(func() {
		n := common.GetEnvOrDefault("INSTALLER_CONCURRENCY", 2)
		if n < 1 {
			n = 1
		}
		installerSlots.ch = make(chan struct{}, n)
	})
	select {
	case installerSlots.ch <- struct{}{}:
		return func() { <-installerSlots.ch }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// runInstaller executes the Forge/NeoForge installer; tests replace it.
var runInstaller = func(ctx context.Context, java, dir string, out io.Writer) error {
	cmd := exec.CommandContext(ctx, java, "-jar", installerJar, "--installServer")
	cmd.Dir = dir
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.WaitDelay = 5 * time.Second
	return cmd.Run()
}

var installGOOS = runtime.GOOS

type installProgress func(stage string, percent int, message string)

// installServer downloads (and for Forge/NeoForge runs the installer of) a resolved source.
func installServer(ctx context.Context, workDir string, src *ServerSource, progress installProgress) (InstallResult, error) {
	if progress == nil {
		progress = func(string, int, string) {}
	}
	hashes := map[string]string(nil)
	if src.SHA1 != "" {
		hashes = map[string]string{"sha1": src.SHA1}
	}
	if !src.Installer {
		progress("downloading", 10, "downloading "+src.Type+" server")
		if err := common.DownloadFileContext(ctx, http.DefaultClient, filepath.Join(workDir, DefaultLaunchFile), src.URL, hashes, -1); err != nil {
			return InstallResult{}, err
		}
		return InstallResult{Mode: LaunchJar, File: DefaultLaunchFile}, writeEula(workDir)
	}

	if src.VanillaURL != "" {
		progress("downloading", 5, "downloading vanilla server "+src.MCVersion)
		vanilla := filepath.Join(workDir, "minecraft_server."+src.MCVersion+".jar")
		if err := common.DownloadFileContext(ctx, http.DefaultClient, vanilla, src.VanillaURL, map[string]string{"sha1": src.VanillaSHA1}, -1); err != nil {
			return InstallResult{}, err
		}
	}
	progress("downloading", 10, "downloading "+src.Type+" installer "+src.LoaderVersion)
	if err := common.DownloadFileContext(ctx, http.DefaultClient, filepath.Join(workDir, installerJar), src.URL, nil, -1); err != nil {
		return InstallResult{}, err
	}

	progress("queued", 20, "waiting for an installer slot")
	release, err := acquireInstallerSlot(ctx)
	if err != nil {
		return InstallResult{}, err
	}
	defer release()

	progress("installing", 30, "running "+src.Type+" installer")
	logFile, err := os.Create(filepath.Join(workDir, installerOutLog))
	if err != nil {
		return InstallResult{}, err
	}
	timeout := installerTimeout()
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	err = runInstaller(runCtx, JavaForMC(ctx, src.MCVersion), workDir, logFile)
	deadline := runCtx.Err() == context.DeadlineExceeded
	cancel()
	_ = logFile.Close()
	if ctx.Err() != nil {
		return InstallResult{}, ctx.Err()
	}
	if deadline {
		return InstallResult{}, fmt.Errorf("installer timed out after %s (log kept in %s)", timeout, installerOutLog)
	}
	if err != nil {
		return InstallResult{}, fmt.Errorf("installer failed: %v (log kept in %s)", err, installerOutLog)
	}

	mode, file, err := DetectLaunch(workDir, installGOOS)
	if err != nil {
		return InstallResult{}, fmt.Errorf("%w (log kept in %s)", err, installerOutLog)
	}
	for _, name := range append([]string{installerJar}, installerLogs...) {
		_ = os.Remove(filepath.Join(workDir, name))
	}
	return InstallResult{Mode: mode, File: file}, writeEula(workDir)
}

func writeEula(workDir string) error {
	return os.WriteFile(filepath.Join(workDir, "eula.txt"), []byte("eula=true\n"), 0644)
}

// clearFailedInstall removes everything except installer logs; the (empty) directory is
// kept so its server ID is never handed out again.
func clearFailedInstall(workDir string) error {
	entries, err := os.ReadDir(workDir)
	if errors.Is(err, os.ErrNotExist) {
		return os.MkdirAll(workDir, 0755)
	}
	if err != nil {
		return err
	}
	keep := map[string]bool{}
	for _, name := range installerLogs {
		keep[name] = true
	}
	var errs []error
	for _, e := range entries {
		if keep[e.Name()] && !e.IsDir() {
			continue
		}
		if err := os.RemoveAll(filepath.Join(workDir, e.Name())); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// ---------------- asynchronous creation ----------------

const (
	modpackSessionPrefix = "mrpack-"
	createSessionPrefix  = "create-"
)

// IsServerInstallSession reports whether id belongs to a server installation
// (modpack import or server creation) rather than a mod install job.
func IsServerInstallSession(id string) bool {
	return strings.HasPrefix(id, modpackSessionPrefix) || strings.HasPrefix(id, createSessionPrefix)
}

func (r CreateServerRequest) loaderVersion() string {
	if v := strings.TrimSpace(r.LoaderVersion); v != "" {
		return v
	}
	return strings.TrimSpace(r.FabricLoader)
}

// StartCreateServer validates the request, records the server as installing and installs
// it in the background. The returned session tracks progress like a modpack import.
func StartCreateServer(ctx context.Context, owner uint, req CreateServerRequest, xms, xmx int64) (*ModpackSession, error) {
	src, err := ResolveServerSource(ctx, req.ServerType, req.ServerVer, req.loaderVersion(), req.FabricInstaller)
	if err != nil {
		return nil, err
	}
	return startInstallSession(installSessionSpec{
		owner: owner, prefix: createSessionPrefix, displayName: req.DisplayName,
		modLoader: req.ServerType, src: src, xms: xms, xmx: xmx, clearOnFailure: true,
		run: func(ctx context.Context, s *ModpackSession, workDir string) error {
			result, err := installServer(ctx, workDir, src, s.progress)
			if err != nil {
				return err
			}
			return model.SetServerLaunch(s.OwnerID, s.ServerID, result.Mode, result.File)
		},
	})
}

func (s *ModpackSession) progress(stage string, percent int, message string) {
	s.record(stage, "", percent, message, nil)
}
