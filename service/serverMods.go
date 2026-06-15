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
	"time"
)

// Mod Manager

// ModrinthVersion API 回應結構
type ModrinthVersion struct {
	ID            string               `json:"id"` // version id
	ProjectID     string               `json:"project_id"`
	Name          string               `json:"name"`
	VersionNumber string               `json:"version_number"`
	VersionType   string               `json:"version_type"`
	Changelog     string               `json:"changelog"`
	DatePublished time.Time            `json:"date_published"`
	DateModified  time.Time            `json:"date_modified"`
	Downloads     int64                `json:"downloads"`
	Featured      bool                 `json:"featured"`
	Dependencies  []ModrinthDependency `json:"dependencies"`
	Files         []ModrinthFile       `json:"files"`
	GameVersions  []string             `json:"game_versions"`
	Loaders       []string             `json:"loaders"`
}

type ModrinthProject struct {
	ID                   string    `json:"id"`
	Slug                 string    `json:"slug"`
	Title                string    `json:"title"`
	Description          string    `json:"description"`
	Body                 string    `json:"body"`
	Team                 string    `json:"team"`
	IconURL              string    `json:"icon_url"`
	BannerURL            string    `json:"banner_url"`
	Downloads            int64     `json:"downloads"`
	Categories           []string  `json:"categories"`
	AdditionalCategories []string  `json:"additional_categories"`
	Updated              time.Time `json:"updated"`
}

type ModrinthDependency struct {
	ProjectID      string `json:"project_id"`
	VersionID      string `json:"version_id"`
	DependencyType string `json:"dependency_type"`
}

type ModrinthFile struct {
	URL      string            `json:"url"`
	Filename string            `json:"filename"`
	Primary  bool              `json:"primary"`
	Size     int64             `json:"size"`
	FileType string            `json:"file_type"`
	Hashes   map[string]string `json:"hashes"`
}

var (
	UnSupportModErr = errors.New("no compatible version found")
	NetWorkErr      = errors.New("network error")
	AlreadyInsErr   = errors.New("mod already installed")
)

func AddMod(sid, workDir, modLoader, MCVersion, modID, ver string, useBeta, autoUpdate bool) error {
	// check mod ver is Support
	if modLoader == Vanilla {
		return errors.New("vanilla can't install mod ")
	}

	if ok, err := model.ModExists(sid, modID); err != nil {
		return err
	} else if ok {
		return AlreadyInsErr
	}

	modInf, err := getLatestOrSpecific(modID, modLoader, MCVersion, ver, useBeta)
	if err != nil {
		return err
	}

	if err := syncModCache(modID, modInf); err != nil {
		common.SysError(fmt.Sprintf("Mod metadata sync failed: %v", err))
	}

	if !isCompatible(modInf, modLoader, MCVersion) {
		return UnSupportModErr
	}

	file, err := selectModFile(modInf.Files)
	if err != nil {
		return err
	}

	// download mod to work dir
	modPath, err := modDownload(*file, workDir)
	if err != nil {
		return err
	}

	// write to DB
	if err := model.AddModToServer(sid, modID, modInf.ID, file.Filename, autoUpdate); err != nil {
		_ = os.Remove(modPath)
		return err
	}

	return nil
}

func fetchModVersionByID(versionID string) (*ModrinthVersion, error) {
	if versionID == "" {
		return nil, fmt.Errorf("version id is empty")
	}

	url := fmt.Sprintf("https://api.modrinth.com/v2/version/%s", versionID)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var v ModrinthVersion
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, err
	}
	return &v, nil
}

// Get Latest or Specific
func getLatestOrSpecific(projectID, loader, gameVersion, modeVer string, useBeta bool) (*ModrinthVersion, error) {
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
		common.Logger.Debugf("getLatestOrSpecific request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var versions []ModrinthVersion
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		common.Logger.Debugf("Failed to fetch latest version: %v", err)
		return nil, err
	}

	if len(versions) == 0 {
		return nil, UnSupportModErr
	}

	filter := make([]ModrinthVersion, 0, len(versions))
	for _, v := range versions {
		vType := strings.ToLower(v.VersionType)

		if vType == "release" || (useBeta && vType == "beta") {
			filter = append(filter, v)
		}

	}

	if len(filter) == 0 {
		return nil, UnSupportModErr
	}

	// 在版本列表中查找匹配的版本號
	if modeVer != "" {
		for i := range versions {
			if versions[i].ID == modeVer || versions[i].VersionNumber == modeVer {
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

	if loader != "" && len(version.Loaders) > 0 {
		loaderMatch = false
		for _, l := range version.Loaders {
			if strings.EqualFold(l, loader) {
				loaderMatch = true
				break
			}
		}
	}

	for _, gv := range version.GameVersions {
		if gv == gameVersion {
			gameVersionMatch = true
			break
		}
	}
	common.SysDebug(fmt.Sprintf("Mod %s loaderMatch: %v, gameVersionMatch: %v", version.Name, loaderMatch, gameVersionMatch))

	return loaderMatch && gameVersionMatch
}

func selectModFile(files []ModrinthFile) (*ModrinthFile, error) {
	if len(files) == 0 {
		return nil, errors.New("mod file list is empty")
	}
	for i := range files {
		if files[i].Primary {
			return &files[i], nil
		}
	}
	return &files[0], nil
}

func syncModCache(modKey string, version *ModrinthVersion) error {
	var errs []string

	project, raw, err := fetchModProject(modKey)
	if err != nil {
		errs = append(errs, fmt.Sprintf("project fetch: %v", err))
	} else if err := upsertModProject(modKey, project, raw); err != nil {
		errs = append(errs, fmt.Sprintf("project upsert: %v", err))
	}

	if version != nil {
		if err := upsertModVersion(modKey, version); err != nil {
			errs = append(errs, fmt.Sprintf("version upsert: %v", err))
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

func fetchModProject(modKey string) (*ModrinthProject, []byte, error) {
	fullURL := fmt.Sprintf("https://api.modrinth.com/v2/project/%s", modKey)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "carsupper665/mc-server-backend (contact: carsuooer665@hgmail.com)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	var project ModrinthProject
	if err := json.Unmarshal(raw, &project); err != nil {
		return nil, nil, err
	}

	return &project, raw, nil
}

func upsertModProject(modKey string, project *ModrinthProject, raw []byte) error {
	if project == nil {
		return errors.New("project is nil")
	}

	categories, err := marshalJSON(project.Categories)
	if err != nil {
		return err
	}
	tags, err := marshalJSON(project.AdditionalCategories)
	if err != nil {
		return err
	}

	rawData := string(raw)
	if rawData == "" {
		if data, err := json.Marshal(project); err == nil {
			rawData = string(data)
		}
	}

	mod := model.Mod{
		ModID:            modKey,
		Slug:             project.Slug,
		Name:             project.Title,
		Summary:          project.Description,
		Description:      project.Body,
		Author:           "",
		AuthorID:         project.Team,
		IconURL:          project.IconURL,
		BannerURL:        project.BannerURL,
		Downloads:        project.Downloads,
		Categories:       categories,
		Tags:             tags,
		ModrinthData:     rawData,
		ProjectUpdatedAt: project.Updated,
		LastSynced:       time.Now(),
		SyncStatus:       "success",
	}

	_, err = model.UpsertMod(&mod)
	return err
}

func upsertModVersion(modKey string, version *ModrinthVersion) error {
	if version == nil {
		return errors.New("version is nil")
	}

	filesJSON, err := marshalJSON(version.Files)
	if err != nil {
		return err
	}
	depsJSON, err := marshalJSON(version.Dependencies)
	if err != nil {
		return err
	}
	gameVersionsJSON, err := marshalJSON(version.GameVersions)
	if err != nil {
		return err
	}

	primaryFile, _ := selectModFile(version.Files)
	modVersion := model.ModVersion{
		VersionID:        version.ID,
		ModID:            modKey,
		VersionNumber:    version.VersionNumber,
		VersionName:      version.Name,
		VersionType:      version.VersionType,
		Changelog:        version.Changelog,
		GameVersions:     gameVersionsJSON,
		Files:            filesJSON,
		Dependencies:     depsJSON,
		Featured:         version.Featured,
		Downloads:        version.Downloads,
		Published:        pickPublishedAt(version),
		VersionUpdatedAt: pickVersionUpdatedAt(version),
	}

	if primaryFile != nil {
		modVersion.PrimaryFile = primaryFile.Filename
		modVersion.DownloadURL = primaryFile.URL
		modVersion.FileSize = primaryFile.Size
		modVersion.FileHash = pickFileHash(primaryFile)
	}

	_, err = model.UpsertModVersion(&modVersion)
	return err
}

func pickPublishedAt(version *ModrinthVersion) time.Time {
	if version == nil {
		return time.Time{}
	}
	if !version.DatePublished.IsZero() {
		return version.DatePublished
	}
	return version.DateModified
}

func pickVersionUpdatedAt(version *ModrinthVersion) time.Time {
	if version == nil {
		return time.Time{}
	}
	if !version.DateModified.IsZero() {
		return version.DateModified
	}
	return version.DatePublished
}

func pickFileHash(file *ModrinthFile) string {
	if file == nil {
		return ""
	}
	if len(file.Hashes) == 0 {
		return ""
	}
	if hash, ok := file.Hashes["sha512"]; ok {
		return hash
	}
	if hash, ok := file.Hashes["sha1"]; ok {
		return hash
	}
	for _, hash := range file.Hashes {
		return hash
	}
	return ""
}

func marshalJSON(value any) (string, error) {
	if value == nil {
		return "[]", nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	if string(data) == "null" {
		return "[]", nil
	}
	return string(data), nil
}

func modDownload(file ModrinthFile, workDir string) (string, error) {

	if file.URL == "" || file.Filename == "" {
		return "", ErrModMetadataMissing
	}

	modsDir := filepath.Join(workDir, "mods")
	if err := os.MkdirAll(modsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create mods directory: %w", err)
	}

	// download
	dlUrl := file.URL
	var (
		resp *http.Response
		err  error
	)
	if resp, err = http.Get(dlUrl); err != nil {
		common.SysError(fmt.Sprintf("Error while mod download: %s", err.Error()))
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		common.SysError(fmt.Sprintf("Error while mod download with status: %d", resp.StatusCode))
		return "", NetWorkErr
	}

	tmpFile, err := os.CreateTemp(modsDir, file.Filename+".tmp-*")
	if err != nil {
		common.SysError(fmt.Sprintf("Error while creating temp mod file: %s", err.Error()))
		return "", err
	}
	tmpName := tmpFile.Name()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
		return "", err
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpName)
		return "", err
	}

	finalPath := filepath.Join(modsDir, file.Filename)
	if err := os.Rename(tmpName, finalPath); err != nil {
		_ = os.Remove(tmpName)
		return "", err
	}

	return finalPath, nil
}
