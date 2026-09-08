package common

import (
	"context"
	"crypto/sha1"
	"crypto/sha512"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DownloadFileContext also serves imports: cancellation, optional integrity checks,
// and a temporary file so failed downloads never replace a destination.
func DownloadFileContext(ctx context.Context, client *http.Client, dest, url string, hashes map[string]string, size int64) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(dest), ".download-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	h1, h512 := sha1.New(), sha512.New()
	var reader io.Reader = resp.Body
	if size >= 0 {
		reader = io.LimitReader(reader, size+1)
	}
	n, err := io.Copy(io.MultiWriter(file, h1, h512), reader)
	if err != nil {
		return err
	}
	if size >= 0 && n != size {
		return fmt.Errorf("size mismatch: got %d, expected %d", n, size)
	}
	for algorithm, actual := range map[string]string{"sha1": fmt.Sprintf("%x", h1.Sum(nil)), "sha512": fmt.Sprintf("%x", h512.Sum(nil))} {
		if expected := hashes[algorithm]; expected != "" && !strings.EqualFold(actual, expected) {
			return fmt.Errorf("%s mismatch", algorithm)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), dest)
}
