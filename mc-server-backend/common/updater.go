package common

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const repo = "carsupper665/mc-server-backend"
const logName = "update.log"

var ErrAlreadyLatest = fmt.Errorf("Already updated.")
var ErrUpdateDisabled = fmt.Errorf("Auto update is disabled.")

type release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Assets  []struct {
		Name               string `json:"name"`
		Digest             string `json:"digest"` // SHA256 checksum
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

type UpdateLogger struct {
	console io.Writer
	file    io.Writer
	strip   *regexp.Regexp
}

func (u *UpdateLogger) Write(p []byte) (n int, err error) {
	// 無修改寫到 console
	if _, err = u.console.Write(p); err != nil {
		return len(p), err
	}
	// 去掉 ANSI code，再寫到 file
	clean := u.strip.ReplaceAll(p, []byte(""))
	if _, err = u.file.Write(clean); err != nil {
		return len(p), err
	}

	return len(p), nil
}

func (u *UpdateLogger) Debug(msg string) {
	if !DebugMode {
		return
	}
	_, _ = fmt.Fprintf(u, "%s%s|%s\n", ColorBrightBlue+"[UPDT][DEBUG]", ColorReset, msg)
}

func (u *UpdateLogger) Info(msg string) {
	_, _ = fmt.Fprintf(u, "%s%s|%s\n", ColorBrightGreen+"[UPDT][INFO]", ColorReset, msg)
}

func (u *UpdateLogger) Warn(msg string) {
	_, _ = fmt.Fprintf(u, "%s%s|%s\n", ColorYellow+"[UPDT][WARN]", ColorReset, msg)
}

func (u *UpdateLogger) Error(msg string) {
	_, _ = fmt.Fprintf(u, "%s%s|%s\n", ColorRed+"[UPDT][ERROR]", ColorReset, msg)
}

func (u *UpdateLogger) Close() error {
	consoleCloser, ok := u.console.(io.Closer)
	fileCloser, ok2 := consoleCloser.(io.Closer)
	if ok {
		err := consoleCloser.Close()
		if err != nil {
			return err
		}
	}
	if ok2 {
		err := fileCloser.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func newLogger(console, file io.Writer) *UpdateLogger {
	return &UpdateLogger{
		console: console,
		file:    file,
		// 這個正則會匹配所有 ANSI Escape Code
		strip: regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`),
	}
}

func getLogger() *UpdateLogger {
	if *LogDir != "" {
		logFilePath := fmt.Sprintf("%s/%s", *LogDir, logName)
		file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal("failed to open log file:", err)
		}

		return newLogger(os.Stdout, file)
	}
	return nil
}

func CheckForUpdates() error {
	var logger = getLogger()
	if logger == nil {
		return errors.New("startup logger failed.")
	}
	if GetEnvOrDefaultBool("AUTO_UPDATE", false) == false {
		logger.Info("Auto update is disabled, skipping update check.")
		return ErrUpdateDisabled
	}
	logger.Info("Checking for updates...")
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error(fmt.Sprintf("Failed to get latest version of %s. Status: %s", repo, resp.Status))
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var r release
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return err
	}

	if r.TagName == Version+Build {
		logger.Info("You are running the latest version: " + Version + Build)
		return ErrAlreadyLatest
	}

	logger.Info("updating....")
	dlUrl := r.Assets[0].BrowserDownloadURL
	hashString := strings.TrimPrefix(r.Assets[0].Digest, "sha256:")
	err = downloadAndApplyUpdate(dlUrl, hashString, logger)
	if err != nil {
		return err
	}

	return nil
}

func downloadAndApplyUpdate(url, expectedHash string, logger *UpdateLogger) error {
	const (
		tmpName = "main_new.exe"
		curr    = "main.exe"
		backup  = "main.exe.bk"
	)

	logger.Info(fmt.Sprintf("Downloading update from %s", url))

	resp, err := http.Get(url)
	if err != nil {
		logger.Error("failed to download update: " + err.Error())
		return err
	}
	defer resp.Body.Close()

	tmpFile, err := os.Create(tmpName)
	if err != nil {
		logger.Error("failed to create temp file: " + err.Error())
		return err
	}
	defer tmpFile.Close()

	// 下載 + 同時計算 hash
	hasher := sha256.New()
	writer := io.MultiWriter(tmpFile, hasher)

	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		logger.Error("failed to save update: " + err.Error())
		return err
	}

	hashSum := fmt.Sprintf("%x", hasher.Sum(nil))
	if hashSum != expectedHash {
		logger.Error(fmt.Sprintf("hash mismatch: expected %s got %s", expectedHash, hashSum))
		os.Remove(tmpName)
		return fmt.Errorf("hash mismatch")
	}

	logger.Info("hash verified successfully")

	if _, err := os.Stat(curr); err == nil {
		logger.Info("backing up old executable")
		os.Remove(backup) // 如果已經有舊的 .bk
		if err := os.Rename(curr, backup); err != nil {
			logger.Error("failed to backup old exe: " + err.Error())
			return err
		}
	}
	_ = tmpFile.Close()
	logger.Info("replacing executable with updated version")
	if err := os.Rename(tmpName, curr); err != nil {
		logger.Error("failed to replace exe: " + err.Error())
		return err
	}

	logger.Info("update applied successfully 🎉")
	return nil
}
