package service

import (
	"errors"
	"testing"
)

func TestPickUpdateAsset(t *testing.T) {
	type asset = struct {
		Name               string `json:"name"`
		Digest             string `json:"digest"`
		BrowserDownloadURL string `json:"browser_download_url"`
	}
	ok := asset{Name: "main.exe", Digest: "sha256:abc", BrowserDownloadURL: "https://x/main.exe"}

	r := &release{TagName: "v", Assets: []asset{{Name: "notes.txt", Digest: "sha256:zz", BrowserDownloadURL: "https://x/n"}, ok}}
	url, hash, err := pickUpdateAsset(r)
	if err != nil || url != ok.BrowserDownloadURL || hash != "abc" {
		t.Fatalf("got %q %q %v", url, hash, err)
	}

	for name, assets := range map[string][]asset{
		"no assets":  nil,
		"wrong name": {{Name: "main_linux", Digest: "sha256:abc", BrowserDownloadURL: "https://x"}},
		"no digest":  {{Name: "main.exe", BrowserDownloadURL: "https://x"}},
		"no url":     {{Name: "main.exe", Digest: "sha256:abc"}},
	} {
		if _, _, err := pickUpdateAsset(&release{TagName: "v", Assets: assets}); !errors.Is(err, ErrNoUpdateAsset) {
			t.Errorf("%s: want ErrNoUpdateAsset, got %v", name, err)
		}
	}
}
