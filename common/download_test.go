package common

import (
	"context"
	"crypto/sha1"
	"crypto/sha512"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// Run HTTP handlers in memory so integrity tests also work without a TCP listener.
type downloadTestTransport struct{ handler http.Handler }

func (tr downloadTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := req.Context().Err(); err != nil {
		return nil, err
	}
	recorder := httptest.NewRecorder()
	tr.handler.ServeHTTP(recorder, req)
	return recorder.Result(), nil
}

func TestDownloadIntegrityAndCancel(t *testing.T) {
	client := &http.Client{Transport: downloadTestTransport{http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "bad") })}}
	const downloadURL = "http://download.test/mod.jar"
	dest := filepath.Join(t.TempDir(), "mod.jar")
	if err := os.WriteFile(dest, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}
	hashes := map[string]string{"sha1": fmt.Sprintf("%x", sha1.Sum([]byte("mod"))), "sha512": fmt.Sprintf("%x", sha512.Sum512([]byte("mod")))}
	if err := DownloadFileContext(context.Background(), client, dest, downloadURL, hashes, 3); err == nil {
		t.Fatal("accepted corrupt content of equal size")
	}
	data, _ := os.ReadFile(dest)
	if string(data) != "original" {
		t.Fatal("failed download replaced destination")
	}
	entries, _ := os.ReadDir(filepath.Dir(dest))
	if len(entries) != 1 {
		t.Fatal("temporary file leaked")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := DownloadFileContext(ctx, client, dest, downloadURL, nil, -1); err == nil {
		t.Fatal("ignored cancellation")
	}
}
