package getit_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"

	"github.com/block/getit"
)

func TestZIPMatch(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{name: "ZipFile", path: "/archive.zip", expected: true},
		{name: "NestedPath", path: "/some/deep/path/archive.zip", expected: true},
		{name: "WithQueryParams", path: "/archive.zip?token=abc", expected: false},
		{name: "UppercaseZip", path: "/archive.ZIP", expected: false},
		{name: "TarGz", path: "/archive.tar.gz", expected: false},
		{name: "PlainFile", path: "/file.txt", expected: false},
		{name: "ZipInName", path: "/zipfile.tar", expected: false},
		{name: "NoExtension", path: "/archive", expected: false},
		{name: "EmptyPath", path: "", expected: false},
	}

	zip := getit.NewZIP()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &url.URL{Path: tt.path}
			result := zip.Match(u)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestZIPFetch(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "archive.zip"))
	assert.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL + "/archive.zip")
	assert.NoError(t, err)

	dest := t.TempDir()
	zip := getit.NewZIP()
	err = zip.Fetch(context.Background(), getit.Source{URL: u}, dest)
	assert.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dest, "file.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "hello from test\n", string(content))

	content, err = os.ReadFile(filepath.Join(dest, "nested.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "nested content\n", string(content))
}

func TestZIPFetchHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL + "/archive.zip")
	assert.NoError(t, err)

	dest := t.TempDir()
	zip := getit.NewZIP()
	err = zip.Fetch(context.Background(), getit.Source{URL: u}, dest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestZIPFetchInvalidZip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not a valid zip file"))
	}))
	defer server.Close()

	u, err := url.Parse(server.URL + "/archive.zip")
	assert.NoError(t, err)

	dest := t.TempDir()
	zip := getit.NewZIP()
	err = zip.Fetch(context.Background(), getit.Source{URL: u}, dest)
	assert.Error(t, err)
}

func TestZIPFetchCancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data"))
	}))
	defer server.Close()

	u, err := url.Parse(server.URL + "/archive.zip")
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dest := t.TempDir()
	zip := getit.NewZIP()
	err = zip.Fetch(ctx, getit.Source{URL: u}, dest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}
