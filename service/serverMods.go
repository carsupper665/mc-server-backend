package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-backend/common"
	"go-backend/model"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Mod Manager

// ModrinthVersion API 回應結構
type ModrinthVersion struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	VersionNumber string         `json:"version_number"`
	Files         []ModrinthFile `json:"files"`
	GameVersions  []string       `json:"game_versions"`
	Loaders       []string       `json:"loaders"`
}

type ModrinthFile struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Primary  bool   `json:"primary"`
	Size     int64  `json:"size"`
	FileType string `json:"file_type"`
}

var (
	UnSupportModErr = errors.New("no compatible version found")
	NetWorkErr      = errors.New("network error")
	AlreadyInsErr   = errors.New("mod already installed")
)

func AddMod(sid, workDir, modLoader, MCVersion, modID, ver string, autoUpdate bool) error {
	// check mod ver is Support
	if modLoader == Vanilla {
		return errors.New("vanilla can't install mod ")
	}

	if ok, err := model.ModExists(sid, modID); err != nil {
		return err
	} else if ok {
		return AlreadyInsErr
	}

	modInf, err := getLatestOrSpecific(modID, modLoader, MCVersion, ver)
	if err != nil {
		return err
	}

	if !isCompatible(modInf, modLoader, MCVersion) {
		return UnSupportModErr
	}

	// download mod to work dir
	if err := modDownload(*modInf, workDir); err != nil {
		return err
	}

	// write to DB
	if err := model.AddModToServer(sid, modID, modInf.ID, modInf.Files[0].Filename, autoUpdate); err != nil {
		return err
	}

	return nil
}

// Get Latest or Specific
func getLatestOrSpecific(projectID, loader, gameVersion, modeVer string) (*ModrinthVersion, error) {
	base := fmt.Sprintf("https://api.modrinth.com/v2/project/%s/version", projectID)

	// 建 query 參數，注意 value 仍然是 JSON array 字串
	q := url.Values{}
	q.Set("loaders", fmt.Sprintf(`["%s"]`, strings.ToLower(loader)))
	q.Set("game_versions", fmt.Sprintf(`["%s"]`, gameVersion))

	fullURL := base + "?" + q.Encode()

	common.SysDebug(fmt.Sprintf("getLatestOrSpecific url: %s", fullURL))
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "carsupper665/mc-server-backend (contact: carsuooer665@hgmail.com)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var versions []ModrinthVersion
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, err
	}

	if len(versions) == 0 {
		return nil, UnSupportModErr
	}

	// 在版本列表中查找匹配的版本號
	if modeVer != "" {
		for i := range versions {
			if versions[i].VersionNumber == modeVer {
				return &versions[i], nil
			}
		}
	}

	// 返回第一個版本（通常是最新的）
	return &versions[0], nil
}

func isCompatible(version *ModrinthVersion, loader, gameVersion string) bool {
	loaderMatch := true
	gameVersionMatch := false

	// TODO: 更嚴格的 loader 檢查 不應該是強制匹配 應該包含1.0.0以上之類的

	for _, gv := range version.GameVersions {
		if gv == gameVersion {
			gameVersionMatch = true
			break
		}
	}
	common.SysDebug(fmt.Sprintf("Mod %s loaderMatch: %v, gameVersionMatch: %v", version.Name, loaderMatch, gameVersionMatch))

	return loaderMatch && gameVersionMatch
}

func modDownload(modInfo ModrinthVersion, workDir string) error {

	modsDir := filepath.Join(workDir, "mods")
	if err := os.MkdirAll(modsDir, 0755); err != nil {
		return fmt.Errorf("failed to create mods directory: %w", err)
	}

	// download
	dlUrl := modInfo.Files[0].URL
	var (
		resp *http.Response
		err  error
	)
	if resp, err = http.Get(dlUrl); err != nil {
		common.SysError(fmt.Sprintf("Error while mod download: %s", err.Error()))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		common.SysError(fmt.Sprintf("Error while mod download with status: %s", resp.StatusCode))
		return NetWorkErr
	}

	output, outErr := os.Create(modsDir + "/" + modInfo.Files[0].Filename)
	if outErr != nil {
		common.SysError(fmt.Sprintf("Error while outputing mod: %s", outErr.Error()))
		return outErr
	}
	defer output.Close()
	_, err = io.Copy(output, resp.Body)
	return nil
}
