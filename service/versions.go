package service

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-backend/common"
)

// 版本來源：Mojang、Fabric meta、Forge／NeoForge maven。
// 所有外部清單都經過 versionCache：TTL 內不重抓、同時請求只打一次外部 API、
// 抓取失敗時回傳上次成功的資料。

var (
	ErrUnsupportedVersion    = errors.New("unsupported version")
	ErrUnsupportedServerType = errors.New("unsupported server type")
)

const (
	Forge    = "forge"
	NeoForge = "neoforge"
)

const (
	mojangManifestURL      = "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json"
	fabricMetaURL          = "https://meta.fabricmc.net/v2"
	forgeMavenURL          = "https://maven.minecraftforge.net/net/minecraftforge/forge"
	forgePromotionsURL     = "https://files.minecraftforge.net/net/minecraftforge/forge/promotions_slim.json"
	neoForgeMavenURL       = "https://maven.neoforged.net/releases/net/neoforged/neoforge"
	neoForgeLegacyMavenURL = "https://maven.neoforged.net/releases/net/neoforged/forge" // MC 1.20.1
	versionFetchTimeout    = 30 * time.Second
	defaultVersionCacheTTL = 6 * time.Hour
)

// forgeMinMC 以前的 Forge 版本啟動方式不同且不支援。
const forgeMinMC = "1.12.2"

// VersionEntry 是遊戲版本清單的一筆資料。
type VersionEntry struct {
	ID     string `json:"id"`
	Type   string `json:"type,omitempty"`
	Stable bool   `json:"stable"`
}

// LoaderBuild 是某個 MC 版本可用的 loader 版本。
type LoaderBuild struct {
	ID          string `json:"id"`
	MCVersion   string `json:"mc_version"`
	Stable      bool   `json:"stable"`
	Latest      bool   `json:"latest"`
	Recommended bool   `json:"recommended"`
	full        string // maven 上的完整版本字串
	baseURL     string // maven artifact 網址
	artifact    string // installer 檔名前綴
}

// VersionCacheTTL reads VERSION_CACHE_TTL (Go duration such as "6h", or seconds).
func VersionCacheTTL() time.Duration {
	raw := strings.TrimSpace(os.Getenv("VERSION_CACHE_TTL"))
	if raw == "" {
		return defaultVersionCacheTTL
	}
	if d, err := time.ParseDuration(raw); err == nil && d > 0 {
		return d
	}
	if n, err := strconv.Atoi(raw); err == nil && n > 0 {
		return time.Duration(n) * time.Second
	}
	return defaultVersionCacheTTL
}

var cacheNow = time.Now

type cacheCall[T any] struct {
	done  chan struct{}
	value T
	err   error
}

type versionCache[T any] struct {
	fetch   func(ctx context.Context) (T, error)
	mu      sync.Mutex
	value   T
	has     bool
	at      time.Time
	pending *cacheCall[T]
}

func newVersionCache[T any](fetch func(ctx context.Context) (T, error)) *versionCache[T] {
	return &versionCache[T]{fetch: fetch}
}

func (c *versionCache[T]) Get(ctx context.Context) (T, error) {
	c.mu.Lock()
	if c.has && cacheNow().Sub(c.at) < VersionCacheTTL() {
		v := c.value
		c.mu.Unlock()
		return v, nil
	}
	call := c.pending
	if call == nil {
		call = &cacheCall[T]{done: make(chan struct{})}
		c.pending = call
		go c.run(call)
	}
	c.mu.Unlock()
	select {
	case <-call.done:
		return call.value, call.err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

// The fetch is detached from the caller so one canceled request does not fail the others.
func (c *versionCache[T]) run(call *cacheCall[T]) {
	ctx, cancel := context.WithTimeout(context.Background(), versionFetchTimeout)
	defer cancel()
	value, err := c.fetch(ctx)
	c.mu.Lock()
	if err == nil {
		c.value, c.has, c.at = value, true, cacheNow()
	} else if c.has {
		versionWarn("version source failed, using cached data: " + err.Error())
		value, err = c.value, nil
	}
	call.value, call.err = value, err
	c.pending = nil
	c.mu.Unlock()
	close(call.done)
}

type keyedVersionCache[T any] struct {
	fetch func(ctx context.Context, key string) (T, error)
	mu    sync.Mutex
	items map[string]*versionCache[T]
}

func newKeyedVersionCache[T any](fetch func(ctx context.Context, key string) (T, error)) *keyedVersionCache[T] {
	return &keyedVersionCache[T]{fetch: fetch, items: map[string]*versionCache[T]{}}
}

func (k *keyedVersionCache[T]) Get(ctx context.Context, key string) (T, error) {
	k.mu.Lock()
	c := k.items[key]
	if c == nil {
		c = newVersionCache(func(ctx context.Context) (T, error) { return k.fetch(ctx, key) })
		k.items[key] = c
	}
	k.mu.Unlock()
	return c.Get(ctx)
}

func versionWarn(msg string) {
	if common.Logger != nil {
		common.Logger.Errorf("%s", msg)
		return
	}
	log.Println(msg)
}

// ---------------- HTTP helpers ----------------

func fetchBody(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "mc-server-backend/versions")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned HTTP %d", url, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 32<<20))
}

func fetchJSON(ctx context.Context, url string, out any) error {
	data, err := fetchBody(ctx, url)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

// ---------------- version comparison ----------------

func versionParts(v string) []string {
	return strings.FieldsFunc(v, func(r rune) bool { return r == '.' || r == '-' || r == '+' || r == '_' })
}

// compareVersions compares dotted versions numerically ("1.21" < "1.21.1" < "26.1").
func compareVersions(a, b string) int {
	pa, pb := versionParts(a), versionParts(b)
	for i := 0; i < len(pa) && i < len(pb); i++ {
		na, ea := strconv.Atoi(pa[i])
		nb, eb := strconv.Atoi(pb[i])
		switch {
		case ea == nil && eb == nil:
			if na != nb {
				if na < nb {
					return -1
				}
				return 1
			}
		case ea == nil: // number > text (e.g. "1" > "beta")
			return 1
		case eb == nil:
			return -1
		default:
			if c := strings.Compare(pa[i], pb[i]); c != 0 {
				return c
			}
		}
	}
	switch {
	case len(pa) < len(pb):
		return -1
	case len(pa) > len(pb):
		return 1
	}
	return 0
}

func isReleaseMC(v string) bool {
	for _, part := range strings.Split(v, ".") {
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}
	return v != ""
}

func isUnstableBuild(v string) bool {
	l := strings.ToLower(v)
	return strings.Contains(l, "beta") || strings.Contains(l, "alpha") || strings.Contains(l, "snapshot") || strings.Contains(l, "-rc") || strings.Contains(l, "pre")
}

// ---------------- Vanilla (Mojang) ----------------

type mojangManifest struct {
	Versions []struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		URL  string `json:"url"`
		SHA1 string `json:"sha1"`
	} `json:"versions"`
}

type mojangVersionDetail struct {
	Downloads struct {
		Server *struct {
			URL  string `json:"url"`
			SHA1 string `json:"sha1"`
			Size int64  `json:"size"`
		} `json:"server"`
	} `json:"downloads"`
	JavaVersion struct {
		MajorVersion int `json:"majorVersion"`
	} `json:"javaVersion"`
}

var versionSources = newVersionSources()

type versionSourceSet struct {
	mojang       *versionCache[*mojangManifest]
	mojangDetail *keyedVersionCache[*mojangVersionDetail]
	fabricLoader *keyedVersionCache[[]LoaderBuild]
	forge        *versionCache[map[string][]LoaderBuild]
	neoForge     *versionCache[map[string][]LoaderBuild]
}

func newVersionSources() *versionSourceSet {
	s := &versionSourceSet{}
	s.mojang = newVersionCache(func(ctx context.Context) (*mojangManifest, error) {
		var m mojangManifest
		if err := fetchJSON(ctx, mojangManifestURL, &m); err != nil {
			return nil, err
		}
		if len(m.Versions) == 0 {
			return nil, errors.New("empty Mojang version manifest")
		}
		return &m, nil
	})
	s.mojangDetail = newKeyedVersionCache(func(ctx context.Context, id string) (*mojangVersionDetail, error) {
		m, err := s.mojang.Get(ctx)
		if err != nil {
			return nil, err
		}
		for _, v := range m.Versions {
			if v.ID == id {
				var d mojangVersionDetail
				if err := fetchJSON(ctx, v.URL, &d); err != nil {
					return nil, err
				}
				return &d, nil
			}
		}
		return nil, ErrUnsupportedVersion
	})
	s.fabricLoader = newKeyedVersionCache(fetchFabricLoaders)
	s.forge = newVersionCache(fetchForgeBuilds)
	s.neoForge = newVersionCache(fetchNeoForgeBuilds)
	return s
}

// FilterVanillaVersions keeps "release" (default), "snapshot", or every type for "all".
func FilterVanillaVersions(m *mojangManifest, typ string) []VersionEntry {
	typ = strings.ToLower(strings.TrimSpace(typ))
	if typ == "" {
		typ = "release"
	}
	out := []VersionEntry{}
	for _, v := range m.Versions {
		if typ != "all" && v.Type != typ {
			continue
		}
		out = append(out, VersionEntry{ID: v.ID, Type: v.Type, Stable: v.Type == "release"})
	}
	return out
}

func ListVanillaVersions(ctx context.Context, typ string) ([]VersionEntry, error) {
	m, err := versionSources.mojang.Get(ctx)
	if err != nil {
		return nil, err
	}
	return FilterVanillaVersions(m, typ), nil
}

func vanillaVersionDetail(ctx context.Context, mc string) (*mojangVersionDetail, error) {
	return versionSources.mojangDetail.Get(ctx, mc)
}

// vanillaServerDownload returns the server jar URL and sha1; versions without one are unsupported.
func vanillaServerDownload(ctx context.Context, mc string) (string, string, error) {
	d, err := vanillaVersionDetail(ctx, mc)
	if err != nil {
		return "", "", err
	}
	if d.Downloads.Server == nil || d.Downloads.Server.URL == "" {
		return "", "", ErrUnsupportedVersion
	}
	return d.Downloads.Server.URL, d.Downloads.Server.SHA1, nil
}

// ---------------- Fabric ----------------

func fetchFabricLoaders(ctx context.Context, mc string) ([]LoaderBuild, error) {
	var raw []struct {
		Loader struct {
			Version string `json:"version"`
			Stable  bool   `json:"stable"`
		} `json:"loader"`
	}
	if err := fetchJSON(ctx, fabricMetaURL+"/versions/loader/"+mc, &raw); err != nil {
		return nil, err
	}
	builds := make([]LoaderBuild, 0, len(raw))
	latest := false
	for _, r := range raw {
		b := LoaderBuild{ID: r.Loader.Version, MCVersion: mc, Stable: r.Loader.Stable}
		if b.Stable && !latest {
			b.Latest, latest = true, true
		}
		builds = append(builds, b)
	}
	return builds, nil
}

// ListFabricLoaders returns Fabric loader versions for mc; the newest stable one is marked latest.
func ListFabricLoaders(ctx context.Context, mc string) ([]LoaderBuild, error) {
	return versionSources.fabricLoader.Get(ctx, mc)
}

func fabricGameVersions(ctx context.Context) ([]VersionEntry, error) {
	var gv []GameVersion
	if err := fetchJSON(ctx, fabricMetaURL+"/versions/game", &gv); err != nil {
		return nil, err
	}
	out := make([]VersionEntry, 0, len(gv))
	for _, v := range gv {
		out = append(out, VersionEntry{ID: v.Version, Stable: v.Stable})
	}
	return out, nil
}

// ---------------- Forge ----------------

type mavenMetadata struct {
	Versions []string `xml:"versioning>versions>version"`
}

func fetchMavenVersions(ctx context.Context, base string) ([]string, error) {
	data, err := fetchBody(ctx, base+"/maven-metadata.xml")
	if err != nil {
		return nil, err
	}
	var m mavenMetadata
	if err := xml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m.Versions, nil
}

// ParseForgeVersion splits a Forge maven version ("1.20.1-47.2.0") into MC and Forge versions.
// Old branch suffixes ("1.7.10-10.13.4.1614-1.7.10") are removed.
func ParseForgeVersion(raw string) (mc, forge string, ok bool) {
	mc, forge, found := strings.Cut(strings.TrimSpace(raw), "-")
	if !found || mc == "" || forge == "" {
		return "", "", false
	}
	forge = strings.TrimSuffix(forge, "-"+mc)
	if !isReleaseMC(mc) {
		return "", "", false
	}
	return mc, forge, true
}

func buildForgeIndex(versions []string, promos map[string]string) map[string][]LoaderBuild {
	index := map[string][]LoaderBuild{}
	for _, raw := range versions {
		mc, forge, ok := ParseForgeVersion(raw)
		if !ok || compareVersions(mc, forgeMinMC) < 0 {
			continue
		}
		index[mc] = append(index[mc], LoaderBuild{
			ID: forge, MCVersion: mc, Stable: !isUnstableBuild(forge),
			full: raw, baseURL: forgeMavenURL, artifact: "forge",
		})
	}
	for mc, builds := range index {
		sortBuildsDesc(builds)
		latest, recommended := promos[mc+"-latest"], promos[mc+"-recommended"]
		markedLatest := false
		for i := range builds {
			if recommended != "" && builds[i].ID == recommended {
				builds[i].Recommended = true
			}
			if latest != "" && builds[i].ID == latest {
				builds[i].Latest, markedLatest = true, true
			}
		}
		if !markedLatest {
			for i := range builds {
				if builds[i].Stable {
					builds[i].Latest = true
					break
				}
			}
		}
		index[mc] = builds
	}
	return index
}

func fetchForgeBuilds(ctx context.Context) (map[string][]LoaderBuild, error) {
	versions, err := fetchMavenVersions(ctx, forgeMavenURL)
	if err != nil {
		return nil, err
	}
	var promos struct {
		Promos map[string]string `json:"promos"`
	}
	if err := fetchJSON(ctx, forgePromotionsURL, &promos); err != nil {
		versionWarn("forge promotions unavailable: " + err.Error())
	}
	return buildForgeIndex(versions, promos.Promos), nil
}

// ---------------- NeoForge ----------------

// NeoForgeMCVersion maps a NeoForge version to its MC version.
//   - 3-part scheme: "20.4.237" → "1.20.4", "21.0.5-beta" → "1.21", "21.1.77" → "1.21.1"
//   - year scheme (MC 26.x+): "26.1.0.5" → "26.1", "26.1.2.7-beta" → "26.1.2"
func NeoForgeMCVersion(v string) (string, bool) {
	base, _, _ := strings.Cut(strings.TrimSpace(v), "-")
	base, _, _ = strings.Cut(base, "+")
	parts := strings.Split(base, ".")
	nums := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return "", false
		}
		nums[i] = n
	}
	switch {
	case len(nums) == 3 && nums[0] >= 20 && nums[0] < 26:
		if nums[1] == 0 {
			return fmt.Sprintf("1.%d", nums[0]), true
		}
		return fmt.Sprintf("1.%d.%d", nums[0], nums[1]), true
	case len(nums) == 4 && nums[0] >= 26:
		if nums[2] == 0 {
			return fmt.Sprintf("%d.%d", nums[0], nums[1]), true
		}
		return fmt.Sprintf("%d.%d.%d", nums[0], nums[1], nums[2]), true
	}
	return "", false
}

func buildNeoForgeIndex(versions, legacy []string) map[string][]LoaderBuild {
	index := map[string][]LoaderBuild{}
	for _, v := range versions {
		mc, ok := NeoForgeMCVersion(v)
		if !ok || isUnstableBuild(v) {
			continue
		}
		index[mc] = append(index[mc], LoaderBuild{ID: v, MCVersion: mc, Stable: true, full: v, baseURL: neoForgeMavenURL, artifact: "neoforge"})
	}
	// MC 1.20.1 lives in the old net.neoforged:forge artifact ("1.20.1-47.1.106").
	for _, raw := range legacy {
		mc, build, ok := ParseForgeVersion(raw)
		if !ok || mc != "1.20.1" || isUnstableBuild(build) {
			continue
		}
		index[mc] = append(index[mc], LoaderBuild{ID: build, MCVersion: mc, Stable: true, full: raw, baseURL: neoForgeLegacyMavenURL, artifact: "forge"})
	}
	for mc, builds := range index {
		sortBuildsDesc(builds)
		builds[0].Latest = true
		index[mc] = builds
	}
	return index
}

func fetchNeoForgeBuilds(ctx context.Context) (map[string][]LoaderBuild, error) {
	versions, err := fetchMavenVersions(ctx, neoForgeMavenURL)
	if err != nil {
		return nil, err
	}
	legacy, err := fetchMavenVersions(ctx, neoForgeLegacyMavenURL)
	if err != nil {
		versionWarn("neoforge 1.20.1 artifact unavailable: " + err.Error())
	}
	return buildNeoForgeIndex(versions, legacy), nil
}

func sortBuildsDesc(builds []LoaderBuild) {
	sort.SliceStable(builds, func(i, j int) bool { return compareVersions(builds[i].ID, builds[j].ID) > 0 })
}

func loaderIndex(ctx context.Context, loader string) (map[string][]LoaderBuild, error) {
	switch loader {
	case Forge:
		return versionSources.forge.Get(ctx)
	case NeoForge:
		return versionSources.neoForge.Get(ctx)
	}
	return nil, ErrUnsupportedServerType
}

// ---------------- public listing ----------------

func normalizeServerType(t string) string { return strings.ToLower(strings.TrimSpace(t)) }

// ListGameVersions lists the MC versions a server type supports (newest first for Forge/NeoForge).
func ListGameVersions(ctx context.Context, serverType, typ string) ([]VersionEntry, error) {
	switch normalizeServerType(serverType) {
	case Vanilla:
		return ListVanillaVersions(ctx, typ)
	case Fabric:
		return fabricGameVersions(ctx)
	case Forge, NeoForge:
		index, err := loaderIndex(ctx, normalizeServerType(serverType))
		if err != nil {
			return nil, err
		}
		out := make([]VersionEntry, 0, len(index))
		for mc := range index {
			out = append(out, VersionEntry{ID: mc, Type: "release", Stable: true})
		}
		sort.Slice(out, func(i, j int) bool { return compareVersions(out[i].ID, out[j].ID) > 0 })
		return out, nil
	}
	return nil, ErrUnsupportedServerType
}

// ListLoaderBuilds lists loader versions for one MC version (newest first).
func ListLoaderBuilds(ctx context.Context, serverType, mc string) ([]LoaderBuild, error) {
	switch t := normalizeServerType(serverType); t {
	case Fabric:
		return ListFabricLoaders(ctx, mc)
	case Forge, NeoForge:
		index, err := loaderIndex(ctx, t)
		if err != nil {
			return nil, err
		}
		builds := index[mc]
		if builds == nil {
			builds = []LoaderBuild{}
		}
		return builds, nil
	}
	return nil, ErrUnsupportedServerType
}
