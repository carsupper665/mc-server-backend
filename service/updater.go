package service

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go-backend/common"
	"go-backend/model"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	tmpName = "main_new.exe"
	curr    = "main.exe"
	backup  = "main.exe.bk"
	repo    = "carsupper665/mc-server-backend"
)

var logPath = flag.String("update-log-name", "update", "specify the log name")
var checkerLog = flag.String("updater-log-name", "update_checker", "specify the log name")
var (
	ErrAlreadyLatest  = errors.New("already updated")
	ErrUpdateDisabled = errors.New("auto update is disabled")
	ErrAlreadyStarted = errors.New("update checker already started")
	ErrHashMismatch   = errors.New("downloaded file hash mismatch")
)

type status struct {
	running bool
	mu      sync.RWMutex
}

func (s *status) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *status) SetRunning(r bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = r
}

var UpdaterStatus = &status{
	running: false,
	mu:      sync.RWMutex{},
}

type release struct {
	TagName     string    `json:"tag_name"`
	HTMLURL     string    `json:"html_url"`
	Prerelease  bool      `json:"prerelease"`
	Draft       bool      `json:"draft"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []struct {
		Name               string `json:"name"`
		Digest             string `json:"digest"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

type UpdateLogger struct {
	console io.Writer
	file    io.Writer
	strip   *regexp.Regexp
}

func (u *UpdateLogger) Write(p []byte) (n int, err error) {
	if _, err = u.console.Write(p); err != nil {
		return len(p), err
	}
	clean := u.strip.ReplaceAll(p, []byte(""))
	if _, err = u.file.Write(clean); err != nil {
		return len(p), err
	}
	return len(p), nil
}

func (u *UpdateLogger) Info(msg string) {
	_, _ = fmt.Fprintf(u, "%s%s|%s \n", common.ColorBrightGreen+"[UPDT][INFO]", common.ColorReset, msg)
}

func (u *UpdateLogger) Error(msg string) {
	_, _ = fmt.Fprintf(u, "%s%s|%s \n", common.ColorRed+"[UPDT][ERROR]", common.ColorReset, msg)
	_ = model.WriteUpdateErrorLog(msg)
}

func (u *UpdateLogger) Close() error {
	if fileCloser, ok := u.file.(io.Closer); ok {
		return fileCloser.Close()
	}
	return nil
}

func newLogger(console, file io.Writer) *UpdateLogger {
	return &UpdateLogger{
		console: console,
		file:    file,
		strip:   regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`),
	}
}

//func getLogger(logFile string) *UpdateLogger {
//	if *common.LogDir != "" {
//		logFilePath := fmt.Sprintf("%s/%s", *common.LogDir, logFile)
//		file, err := os.OpenFile(logFilePath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
//		if err != nil {
//			log.Fatal("failed to open log file:", err)
//		}
//		stdout := colorable.NewColorableStdout()
//		return newLogger(stdout, file)
//	}
//	return nil
//}

// performUpdate 執行更新檢查和下載
func performUpdate(logger *common.SysLogger) error {
	_ = model.SetLastCheck(time.Now())

	if UpdaterStatus.IsRunning() {
		logger.Info("update already running, skipping...")
		return ErrAlreadyStarted
	}

	UpdaterStatus.SetRunning(true)
	defer UpdaterStatus.SetRunning(false)

	if !common.GetEnvOrDefaultBool("AUTO_UPDATE", false) {
		_ = model.SetStatus("auto update disabled")
		return ErrUpdateDisabled
	}

	// 獲取最新版本
	var r *release
	var err error

	if !common.UseBetaVersion {
		r, err = fetchLatestRelease(logger)
		if err != nil {
			return err
		}
	} else {
		r, err = fetchLatestBeta(logger)
		if err != nil {
			return err
		}
	}

	if strings.EqualFold(r.TagName, common.Version+common.Build) {
		_ = model.SetStatus("latest version")
		return ErrAlreadyLatest
	}

	// 下載並應用更新
	logger.Info("New version available: " + r.TagName)
	dlUrl := r.Assets[0].BrowserDownloadURL
	hashString := strings.TrimPrefix(r.Assets[0].Digest, "sha256:")

	if err := downloadAndApplyUpdate(dlUrl, hashString, logger); err != nil {
		_ = model.SetStatus("update failed")
		return err
	}

	_ = model.SetStatus("updated to " + r.TagName)
	_ = model.UpdateTime()
	_ = model.ClearUpdateError()

	return nil
}

// CheckForUpdates 初次啟動時的更新檢查
func CheckForUpdates() error {
	logger, err := common.NewSysLogger("updater", logPath, 50000)
	if logger == nil || err != nil {
		_ = model.WriteUpdateErrorLog("startup logger failed")
		return errors.New("startup logger failed")
	}

	err = performUpdate(logger)
	return err
}

// fetchLatestRelease 獲取最新版本資訊
func fetchLatestRelease(logger *common.SysLogger) (*release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		logger.Error("failed to fetch release info: " + err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("unexpected status: %s", resp.Status)
		logger.Error(err.Error())
		return nil, err
	}

	var r release
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		logger.Error("failed to decode response: " + err.Error())
		return nil, err
	}

	return &r, nil
}

// fetchLatestBeta 獲取最新 beta 版本資訊
func fetchLatestBeta(logger *common.SysLogger) (*release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases", repo)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		logger.Error("failed to fetch beta release info: " + err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("unexpected status: %s", resp.Status)
		logger.Error(err.Error())
		return nil, err
	}

	var releases []release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		logger.Error("failed to decode response: " + err.Error())
		return nil, err
	}
	if len(releases) == 0 {
		return nil, errors.New("no releases found")
	}

	var latest *release
	for i := range releases {
		r := &releases[i]
		if r.Draft {
			continue
		}
		if !r.Prerelease && !strings.Contains(strings.ToLower(r.TagName), "beta") {
			continue
		}
		if latest == nil || r.PublishedAt.After(latest.PublishedAt) {
			latest = r
		}
	}

	if latest == nil {
		return nil, errors.New("no beta release found")
	}

	return latest, nil
}

func downloadAndApplyUpdate(url, expectedHash string, logger *common.SysLogger) error {
	logger.Info("Downloading update...")

	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		logger.Error("failed to download: " + err.Error())
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("unexpected status: %s", resp.Status)
		logger.Error(err.Error())
		return err
	}

	tmpFile, err := os.Create(tmpName)
	if err != nil {
		logger.Error("failed to create temp file: " + err.Error())
		return err
	}

	var downloadErr error
	defer func() {
		tmpFile.Close()
		if downloadErr != nil {
			os.Remove(tmpName)
		}
	}()

	hasher := sha256.New()
	writer := io.MultiWriter(tmpFile, hasher)

	written, err := io.Copy(writer, resp.Body)
	if err != nil {
		downloadErr = err
		logger.Error("failed to save update: " + err.Error())
		return err
	}

	logger.Info(fmt.Sprintf("Downloaded %d bytes", written))

	// 同步到磁碟
	if err := tmpFile.Sync(); err != nil {
		downloadErr = err
		logger.Error("failed to sync: " + err.Error())
		return err
	}

	// 驗證 hash
	hashSum := fmt.Sprintf("%x", hasher.Sum(nil))
	if hashSum != expectedHash {
		downloadErr = ErrHashMismatch
		logger.Error(fmt.Sprintf("hash mismatch: expected %s got %s", expectedHash, hashSum))
		return ErrHashMismatch
	}

	logger.Info("Hash verified successfully")

	// 備份舊版本
	if _, err := os.Stat(curr); err == nil {
		logger.Info("Backing up old executable")
		os.Remove(backup)
		if err := os.Rename(curr, backup); err != nil {
			logger.Error("failed to backup: " + err.Error())
			return err
		}
	}

	// 關閉檔案
	if err := tmpFile.Close(); err != nil {
		logger.Error("failed to close temp file: " + err.Error())
		return err
	}

	// 替換執行檔
	logger.Info("Replacing executable")
	if err := os.Rename(tmpName, curr); err != nil {
		logger.Error("failed to replace exe: " + err.Error())
		return err
	}

	return nil
}

func BuildUpdateChecker(onUpdateSuccess func()) func() error {
	return func() error {
		logger, err := common.NewSysLogger("checker", checkerLog, 50000)
		if logger == nil || err != nil {
			common.SysError("failed to start update checker logger")
			return errors.New("failed to start update checker logger")
		}

		logger.Info("Running scheduled update check...")

		if err := performUpdate(logger); err != nil {
			if errors.Is(err, ErrAlreadyLatest) {
				logger.Info("Already at latest version")
			} else if errors.Is(err, ErrUpdateDisabled) {
				logger.Info("Auto update is disabled")
			} else {
				logger.Error("Update check failed: " + err.Error())
			}
		} else {
			// 更新成功
			logger.Info("Update successful! Triggering restart...")
			if onUpdateSuccess != nil {
				onUpdateSuccess()
			}
			return nil
		}
		return nil
	}
}
