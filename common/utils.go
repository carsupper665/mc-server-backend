// common/utils.go

package common

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func GetRandomString(length int) string {
	key := make([]byte, length)
	for i := 0; i < length; i++ {
		key[i] = keyChars[rand.Intn(len(keyChars))]
	}
	return string(key)
}

func GetRandomIntString(length int) string {
	key := make([]byte, length)
	for i := 0; i < length; i++ {
		key[i] = NumberChars[rand.Intn(10)] // 只使用數字
	}
	return string(key)
}

func GetTimeString() string {
	now := time.Now()
	return fmt.Sprintf("%s%d", now.Format("20060102150405"), now.UnixNano()%1e9)
}

func DownloadFile(dest, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("http get %s error: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("bad status downloading %s: %s", url, resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create file %s error: %w", dest, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("writing to %s error: %w", dest, err)
	}
	return nil
}

func SendErrorToDc(msg string) error {
	url := DCWebHookUrl
	if url == "" {
		return fmt.Errorf("Discord webhook URL is not set")
	}

	payload := map[string]string{
		"content":  msg,
		"username": "ServerControllerNotify",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook send failed: status %d, body: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func GetPortList(start int, end int) []int {
	ports := make([]int, 0, end-start+1)
	for i := start; i <= end; i++ {
		ports = append(ports, i)
	}
	return ports
}

func Copy(src, dst string) error {
	err := os.MkdirAll(dst, os.ModePerm) //0777 = os.ModePerm
	if err != nil {
		return err
	}

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		} else {
			return copyFile(path, targetPath)
		}
	})

}

func copyFile(src, dst string) error {
	file, err := os.Open(src)

	if err != nil {
		return err
	}
	defer file.Close()

	dstFile, err := os.Create(dst)

	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, file)

	if err != nil {
		return err
	}
	// 複製檔案權限
	info, err := os.Stat(src)
	if err == nil {
		err = os.Chmod(dst, info.Mode())
	}
	return err
}

type ModrinthVersion struct {
	ID            string               `json:"id"`
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

func FetchLatestModrinthVersion(projectID, loader, gameVersion string) (*ModrinthVersion, error) {
	base := fmt.Sprintf("https://api.modrinth.com/v2/project/%s/version", projectID)

	q := url.Values{}
	if strings.TrimSpace(loader) != "" {
		q.Set("loaders", fmt.Sprintf(`["%s"]`, strings.ToLower(loader)))
	}
	if strings.TrimSpace(gameVersion) != "" {
		q.Set("game_versions", fmt.Sprintf(`["%s"]`, gameVersion))
	}

	fullURL := base
	if len(q) > 0 {
		fullURL = base + "?" + q.Encode()
	}

	SysDebug(fmt.Sprintf("fetchLatestModrinthVersion url: %s", fullURL))
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
		return nil, errors.New("no compatible version found")
	}

	return &versions[0], nil
}

func FetchModrinthProject(modKey string) (*ModrinthProject, []byte, error) {
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

type CleanupErrorItem struct {
	SID string
	Err error
}

type ErrCleanUpStack struct {
	Items []CleanupErrorItem
}

func (e *ErrCleanUpStack) Add(sid string, err error) {
	if err == nil {
		return
	}

	e.Items = append(e.Items, CleanupErrorItem{
		SID: sid,
		Err: err,
	})
}

func (e *ErrCleanUpStack) HasError() bool {
	return len(e.Items) > 0
}

// Error 會在 return error 時自動被呼叫。
func (e *ErrCleanUpStack) Error() string {
	if len(e.Items) == 0 {
		return ""
	}

	// map 走訪順序不固定；排序後 log / 測試結果比較穩定。
	items := append([]CleanupErrorItem(nil), e.Items...)
	sort.Slice(items, func(i, j int) bool {
		return items[i].SID < items[j].SID
	})

	var b strings.Builder
	fmt.Fprintf(&b, "cleanup failed for %d server(s):", len(items))

	for _, item := range items {
		fmt.Fprintf(&b, "\n  SID=%s: %v", item.SID, item.Err)
	}

	return b.String()
}

// 讓 errors.Is / errors.As 可以檢查裡面的原始錯誤。
// Go 1.20+ 支援 Unwrap() []error。
func (e *ErrCleanUpStack) Unwrap() []error {
	errs := make([]error, 0, len(e.Items))

	for _, item := range e.Items {
		errs = append(errs, item.Err)
	}

	return errs
}

// 可選：方便拿到標準 errors.Join 的結果。
func (e *ErrCleanUpStack) JoinedError() error {
	if !e.HasError() {
		return nil
	}
	return errors.Join(e.Unwrap()...)
}
