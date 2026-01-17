package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type fabricLoaderCompatItem struct {
	Loader struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	} `json:"loader"`
}

type fabricInstallerItem struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	Stable  *bool  `json:"stable,omitempty"` // 有些情況可能沒有這欄，做成 optional
}

func FetchLatestFabricLoaderAndInstaller(mcVersion string) (loaderVer string, installerVer string, err error) {
	client := &http.Client{Timeout: 15 * time.Second}

	// --- 1) Loader (per MC version) ---
	// Fabric Meta: /v2/versions/loader/:game_version
	// game_version 建議 URL encode，且列表 newest first :contentReference[oaicite:1]{index=1}
	mc := url.PathEscape(strings.TrimSpace(mcVersion))
	loaderURL := fmt.Sprintf("https://meta.fabricmc.net/v2/versions/loader/%s", mc)

	{
		req, e := http.NewRequest(http.MethodGet, loaderURL, nil)
		if e != nil {
			return "", "", e
		}
		req.Header.Set("Accept", "application/json")

		resp, e := client.Do(req)
		if e != nil {
			return "", "", e
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", "", fmt.Errorf("fabric meta loader unexpected status: %s", resp.Status)
		}

		var items []fabricLoaderCompatItem
		if e := json.NewDecoder(resp.Body).Decode(&items); e != nil {
			return "", "", e
		}
		if len(items) == 0 {
			return "", "", fmt.Errorf("no fabric loader versions for mc=%s", mcVersion)
		}

		// newest first；stable 優先 :contentReference[oaicite:2]{index=2}
		for _, it := range items {
			if it.Loader.Stable && it.Loader.Version != "" {
				loaderVer = it.Loader.Version
				break
			}
		}
		if loaderVer == "" {
			loaderVer = items[0].Loader.Version
		}
		if loaderVer == "" {
			return "", "", fmt.Errorf("fabric meta returned empty loader version")
		}
	}

	// --- 2) Installer (global latest) ---
	// 常見用法：GET /v2/versions/installer 然後取 .[0].version（最新）:contentReference[oaicite:3]{index=3}
	installerURL := "https://meta.fabricmc.net/v2/versions/installer"

	{
		req, e := http.NewRequest(http.MethodGet, installerURL, nil)
		if e != nil {
			return "", "", e
		}
		req.Header.Set("Accept", "application/json")

		resp, e := client.Do(req)
		if e != nil {
			return "", "", e
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", "", fmt.Errorf("fabric meta installer unexpected status: %s", resp.Status)
		}

		var items []fabricInstallerItem
		if e := json.NewDecoder(resp.Body).Decode(&items); e != nil {
			return "", "", e
		}
		if len(items) == 0 {
			return "", "", fmt.Errorf("no fabric installer versions returned")
		}

		// 若有 stable 欄位就優先 stable；否則直接取第一筆（常見做法）:contentReference[oaicite:4]{index=4}
		for _, it := range items {
			if it.Version == "" {
				continue
			}
			if it.Stable == nil || *it.Stable {
				installerVer = it.Version
				break
			}
		}
		if installerVer == "" {
			installerVer = items[0].Version
		}
		if installerVer == "" {
			return "", "", fmt.Errorf("fabric meta returned empty installer version")
		}
	}

	return loaderVer, installerVer, nil
}
