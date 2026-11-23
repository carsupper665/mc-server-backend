package service

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const vanillaJar = "vanilla-server-jar"

func sha1Hex(s string) string { return fmt.Sprintf("%x", sha1.Sum([]byte(s))) }

const testForgeMetadata = `<?xml version="1.0" encoding="UTF-8"?>
<metadata><groupId>net.minecraftforge</groupId><artifactId>forge</artifactId><versioning><versions>
<version>1.7.10-10.13.4.1614-1.7.10</version>
<version>1.12.1-14.22.1.2485</version>
<version>1.12.2-14.23.5.2859</version>
<version>1.12.2-14.23.5.2860</version>
<version>1.16.5-36.2.39</version>
<version>1.20.1-47.2.0</version>
<version>1.20.1-47.3.0</version>
<version>1.20.1-47.4.10</version>
<version>1.21-51.0.33</version>
</versions></versioning></metadata>`

const testForgePromos = `{"homepage":"x","promos":{"1.20.1-latest":"47.4.10","1.20.1-recommended":"47.3.0","1.12.2-recommended":"14.23.5.2859","1.12.2-latest":"14.23.5.2860"}}`

const testNeoForgeMetadata = `<metadata><versioning><versions>
<version>20.2.3-beta</version>
<version>20.4.237</version>
<version>21.0.0-beta</version>
<version>21.1.9</version>
<version>21.1.77</version>
<version>21.1.200</version>
<version>21.4.10-beta</version>
<version>26.1.0.0-alpha.1+snapshot-1</version>
<version>26.1.0.3-beta</version>
<version>26.1.0.7</version>
</versions></versioning></metadata>`

const testNeoForgeLegacyMetadata = `<metadata><versioning><versions>
<version>1.20.1-47.1.3</version>
<version>1.20.1-47.1.106</version>
</versions></versioning></metadata>`

// versionTransport serves every external version source used by the service, in memory.
func versionTransport(t *testing.T, counter *atomic.Int32) packTransport {
	return func(req *http.Request) (*http.Response, error) {
		if counter != nil {
			counter.Add(1)
		}
		u := req.URL.String()
		switch {
		case u == mojangManifestURL:
			return packResponse(req, 200, `{"latest":{"release":"1.21.4"},"versions":[
				{"id":"25w01a","type":"snapshot","url":"https://piston-meta.mojang.com/v1/packages/a/25w01a.json"},
				{"id":"1.21.4","type":"release","url":"https://piston-meta.mojang.com/v1/packages/a/1.21.4.json"},
				{"id":"1.20.1","type":"release","url":"https://piston-meta.mojang.com/v1/packages/a/1.20.1.json"},
				{"id":"1.16.5","type":"release","url":"https://piston-meta.mojang.com/v1/packages/a/1.16.5.json"},
				{"id":"1.12.2","type":"release","url":"https://piston-meta.mojang.com/v1/packages/a/1.12.2.json"},
				{"id":"b1.7.3","type":"old_beta","url":"https://piston-meta.mojang.com/v1/packages/a/b1.7.3.json"},
				{"id":"1.2.5","type":"release","url":"https://piston-meta.mojang.com/v1/packages/a/1.2.5.json"}]}`), nil
		case strings.HasPrefix(u, "https://piston-meta.mojang.com/v1/packages/a/"):
			id := strings.TrimSuffix(strings.TrimPrefix(u, "https://piston-meta.mojang.com/v1/packages/a/"), ".json")
			java := map[string]int{"1.21.4": 21, "1.20.1": 17, "1.16.5": 8, "1.12.2": 8, "25w01a": 21, "1.2.5": 8}[id]
			// Like the real data: b1.7.3 has no downloads.server (1.2.5 does).
			if id == "b1.7.3" {
				return packResponse(req, 200, `{"downloads":{"client":{"url":"https://piston-data.mojang.com/c.jar"}},"javaVersion":{"majorVersion":8}}`), nil
			}
			sum := sha1Hex(vanillaJar)
			if id == "1.20.1" {
				sum = sha1Hex("different") // corrupted download
			}
			return packResponse(req, 200, fmt.Sprintf(`{"downloads":{"server":{"url":"https://piston-data.mojang.com/v1/objects/%s/server.jar","sha1":"%s","size":%d}},"javaVersion":{"majorVersion":%d}}`, id, sum, len(vanillaJar), java)), nil
		case strings.HasPrefix(u, "https://piston-data.mojang.com/"):
			return packResponse(req, 200, vanillaJar), nil
		case u == forgeMavenURL+"/maven-metadata.xml":
			return packResponse(req, 200, testForgeMetadata), nil
		case u == forgePromotionsURL:
			return packResponse(req, 200, testForgePromos), nil
		case u == neoForgeMavenURL+"/maven-metadata.xml":
			return packResponse(req, 200, testNeoForgeMetadata), nil
		case u == neoForgeLegacyMavenURL+"/maven-metadata.xml":
			return packResponse(req, 200, testNeoForgeLegacyMetadata), nil
		case strings.HasSuffix(u, "-installer.jar"):
			return packResponse(req, 200, "installer"), nil
		case u == fabricMetaURL+"/versions/loader/1.21.4":
			return packResponse(req, 200, `[{"loader":{"version":"0.17.0-beta.1","stable":false}},{"loader":{"version":"0.16.9","stable":true}},{"loader":{"version":"0.16.8","stable":true}}]`), nil
		case u == fabricMetaURL+"/versions/loader/1.0":
			return packResponse(req, 200, `[]`), nil
		case strings.HasPrefix(u, fabricMetaURL+"/versions/loader/") && strings.HasSuffix(u, "/server/jar"):
			return packResponse(req, 200, "fabric-server"), nil
		}
		t.Errorf("unexpected request: %s", u)
		return nil, fmt.Errorf("unexpected request: %s", u)
	}
}

func resetVersionSources(t *testing.T) {
	t.Helper()
	old := versionSources
	versionSources = newVersionSources()
	t.Cleanup(func() { versionSources = old })
}

func setupVersionTransport(t *testing.T, counter *atomic.Int32) {
	t.Helper()
	oldClient, oldTransport := http.DefaultClient, http.DefaultTransport
	tr := versionTransport(t, counter)
	http.DefaultTransport = tr
	http.DefaultClient = &http.Client{Transport: tr}
	t.Cleanup(func() { http.DefaultClient, http.DefaultTransport = oldClient, oldTransport })
	resetVersionSources(t)
}

func TestVanillaVersionsReleaseOnlyByDefault(t *testing.T) {
	setupVersionTransport(t, nil)
	got, err := ListVanillaVersions(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, v := range got {
		if v.Type != "release" || !v.Stable {
			t.Fatalf("non-release entry: %+v", v)
		}
		ids = append(ids, v.ID)
	}
	if strings.Join(ids, ",") != "1.21.4,1.20.1,1.16.5,1.12.2,1.2.5" {
		t.Fatalf("ids=%v", ids)
	}
	snap, _ := ListVanillaVersions(context.Background(), "snapshot")
	if len(snap) != 1 || snap[0].ID != "25w01a" || snap[0].Stable {
		t.Fatalf("snapshot filter: %+v", snap)
	}
	all, _ := ListVanillaVersions(context.Background(), "all")
	if len(all) != 7 {
		t.Fatalf("all filter: %d", len(all))
	}
}

func TestVanillaServerDownloadMissingServer(t *testing.T) {
	setupVersionTransport(t, nil)
	url, sum, err := vanillaServerDownload(context.Background(), "1.21.4")
	if err != nil || !strings.Contains(url, "1.21.4/server.jar") || sum != sha1Hex(vanillaJar) {
		t.Fatalf("url=%s sha1=%s err=%v", url, sum, err)
	}
	if _, _, err := vanillaServerDownload(context.Background(), "b1.7.3"); !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("version without server jar: %v", err)
	}
	if _, _, err := vanillaServerDownload(context.Background(), "9.9.9"); !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("unknown version: %v", err)
	}
}

func TestVersionCacheTTLStaleAndErrors(t *testing.T) {
	now := time.Unix(0, 0)
	oldNow := cacheNow
	cacheNow = func() time.Time { return now }
	t.Cleanup(func() { cacheNow = oldNow })
	t.Setenv("VERSION_CACHE_TTL", "1h")

	calls, fail := 0, false
	c := newVersionCache(func(ctx context.Context) (int, error) {
		calls++
		if fail {
			return 0, errors.New("remote down")
		}
		return calls, nil
	})
	if v, err := c.Get(context.Background()); err != nil || v != 1 {
		t.Fatalf("first: %d %v", v, err)
	}
	now = now.Add(30 * time.Minute)
	if v, _ := c.Get(context.Background()); v != 1 || calls != 1 {
		t.Fatalf("refetched inside TTL: v=%d calls=%d", v, calls)
	}
	now = now.Add(time.Hour)
	fail = true
	if v, err := c.Get(context.Background()); err != nil || v != 1 || calls != 2 {
		t.Fatalf("stale fallback: v=%d err=%v calls=%d", v, err, calls)
	}
	fail = false
	if v, _ := c.Get(context.Background()); v != 3 {
		t.Fatalf("refresh after failure: %d", v)
	}

	empty := newVersionCache(func(ctx context.Context) (int, error) { return 0, errors.New("remote down") })
	if _, err := empty.Get(context.Background()); err == nil {
		t.Fatal("no cache and remote failure must return an error")
	}
}

func TestVersionCacheCoalescesConcurrentRequests(t *testing.T) {
	var calls atomic.Int32
	release := make(chan struct{})
	c := newVersionCache(func(ctx context.Context) (string, error) {
		calls.Add(1)
		<-release
		return "ok", nil
	})
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if v, err := c.Get(context.Background()); err != nil || v != "ok" {
				t.Errorf("got %q %v", v, err)
			}
		}()
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("external API called %d times", calls.Load())
	}
}

func TestVersionCacheTTLEnv(t *testing.T) {
	for raw, want := range map[string]time.Duration{"": 6 * time.Hour, "30m": 30 * time.Minute, "120": 2 * time.Minute, "bad": 6 * time.Hour, "-1h": 6 * time.Hour} {
		t.Setenv("VERSION_CACHE_TTL", raw)
		if got := VersionCacheTTL(); got != want {
			t.Errorf("%q: %s, want %s", raw, got, want)
		}
	}
}

func TestParseForgeVersion(t *testing.T) {
	cases := []struct{ raw, mc, forge string }{
		{"1.20.1-47.2.0", "1.20.1", "47.2.0"},
		{"1.12.2-14.23.5.2859", "1.12.2", "14.23.5.2859"},
		{"1.21-51.0.33", "1.21", "51.0.33"},
		{"1.7.10-10.13.4.1614-1.7.10", "1.7.10", "10.13.4.1614"},
	}
	for _, c := range cases {
		mc, forge, ok := ParseForgeVersion(c.raw)
		if !ok || mc != c.mc || forge != c.forge {
			t.Errorf("%s → %s %s %v", c.raw, mc, forge, ok)
		}
	}
	for _, bad := range []string{"", "47.2.0", "1.7.10_pre4-10.12.0.1024"} {
		if _, _, ok := ParseForgeVersion(bad); ok {
			t.Errorf("%q should not parse", bad)
		}
	}
}

func TestForgeIndexFiltersOldAndMarksPromotions(t *testing.T) {
	setupVersionTransport(t, nil)
	mcs, err := ListGameVersions(context.Background(), "Forge", "")
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, v := range mcs {
		ids = append(ids, v.ID)
	}
	if strings.Join(ids, ",") != "1.21,1.20.1,1.16.5,1.12.2" {
		t.Fatalf("forge MC versions: %v", ids)
	}
	builds, err := ListLoaderBuilds(context.Background(), "forge", "1.20.1")
	if err != nil || len(builds) != 3 {
		t.Fatalf("builds=%+v err=%v", builds, err)
	}
	if builds[0].ID != "47.4.10" || !builds[0].Latest || builds[0].Recommended {
		t.Fatalf("latest: %+v", builds[0])
	}
	if builds[1].ID != "47.3.0" || !builds[1].Recommended {
		t.Fatalf("recommended: %+v", builds[1])
	}
	// No promotions: newest stable is latest.
	b121, _ := ListLoaderBuilds(context.Background(), "forge", "1.21")
	if len(b121) != 1 || !b121[0].Latest {
		t.Fatalf("1.21: %+v", b121)
	}
	if old, _ := ListLoaderBuilds(context.Background(), "forge", "1.12.1"); len(old) != 0 {
		t.Fatalf("pre-1.12.2 builds listed: %+v", old)
	}
}

func TestNeoForgeMCVersion(t *testing.T) {
	cases := map[string]string{
		"20.4.237": "1.20.4", "20.2.3-beta": "1.20.2", "21.0.0-beta": "1.21", "21.1.77": "1.21.1",
		"21.10.5": "1.21.10", "26.1.0.7": "26.1", "26.1.2.3-beta": "26.1.2",
	}
	for v, want := range cases {
		if got, ok := NeoForgeMCVersion(v); !ok || got != want {
			t.Errorf("%s → %s %v, want %s", v, got, ok, want)
		}
	}
	for _, bad := range []string{"", "1.20.1-47.1.106", "abc", "19.1.0"} {
		if _, ok := NeoForgeMCVersion(bad); ok {
			t.Errorf("%q should not map", bad)
		}
	}
}

func TestNeoForgeIndexStableOnlyWith1201(t *testing.T) {
	setupVersionTransport(t, nil)
	mcs, err := ListGameVersions(context.Background(), "NeoForge", "")
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, v := range mcs {
		ids = append(ids, v.ID)
	}
	// 1.20.2 / 1.21 / 1.21.4 only have betas.
	if strings.Join(ids, ",") != "26.1,1.21.1,1.20.4,1.20.1" {
		t.Fatalf("neoforge MC versions: %v", ids)
	}
	builds, _ := ListLoaderBuilds(context.Background(), "neoforge", "1.21.1")
	if len(builds) != 3 || builds[0].ID != "21.1.200" || !builds[0].Latest || builds[1].Latest {
		t.Fatalf("1.21.1 builds: %+v", builds)
	}
	legacy, _ := ListLoaderBuilds(context.Background(), "neoforge", "1.20.1")
	if len(legacy) != 2 || legacy[0].ID != "47.1.106" || legacy[0].full != "1.20.1-47.1.106" || legacy[0].artifact != "forge" {
		t.Fatalf("1.20.1 builds: %+v", legacy)
	}
}

func TestFabricLoaderList(t *testing.T) {
	var calls atomic.Int32
	setupVersionTransport(t, &calls)
	builds, err := ListLoaderBuilds(context.Background(), "Fabric", "1.21.4")
	if err != nil || len(builds) != 3 {
		t.Fatalf("builds=%+v err=%v", builds, err)
	}
	if builds[0].Stable || builds[0].Latest || !builds[1].Stable || !builds[1].Latest || builds[2].Latest {
		t.Fatalf("stable/latest flags: %+v", builds)
	}
	if _, err := ListLoaderBuilds(context.Background(), "fabric", "1.21.4"); err != nil || calls.Load() != 1 {
		t.Fatalf("loader list not cached: calls=%d err=%v", calls.Load(), err)
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.21", "1.21.1", -1}, {"1.21.10", "1.21.9", 1}, {"26.1", "1.21.11", 1}, {"1.12.2", "1.12.2", 0}, {"21.1.200", "21.1.77", 1},
	}
	for _, c := range cases {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("%s vs %s = %d", c.a, c.b, got)
		}
	}
}
