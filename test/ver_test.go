package test

import (
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"go-backend/service"
)

func TestFabricVer(t *testing.T) {
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	http.DefaultTransport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://meta.fabricmc.net/v2/versions/game" {
			t.Fatalf("unexpected URL: %s", req.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`[{"version":"1.21.6","stable":true},{"version":"25w20a","stable":false}]`)), Header: make(http.Header)}, nil
	})
	versions, err := service.GetAllFabricVersions()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(versions, []string{"1.21.6", "25w20a"}) {
		t.Fatalf("wrong versions: %v", versions)
	}
}
