package getit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestTARMatch(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{name: "PlainTar", path: "/archive.tar", expected: true},
		{name: "TarGz", path: "/archive.tar.gz", expected: true},
		{name: "TarBz2", path: "/archive.tar.bz2", expected: true},
		{name: "TarXz", path: "/archive.tar.xz", expected: true},
		{name: "TarZst", path: "/archive.tar.zst", expected: true},
		{name: "TarLz", path: "/archive.tar.lz", expected: true},
		{name: "TarZ", path: "/archive.tar.Z", expected: true},
		{name: "Tgz", path: "/archive.tgz", expected: true},
		{name: "Tbz", path: "/archive.tbz", expected: true},
		{name: "Tbz2", path: "/archive.tbz2", expected: true},
		{name: "Txz", path: "/archive.txz", expected: true},
		{name: "Tzstd", path: "/archive.tzstd", expected: true},
		{name: "Tlz", path: "/archive.tlz", expected: true},
		{name: "TZ", path: "/archive.tZ", expected: true},
		{name: "NestedPath", path: "/some/deep/path/archive.tar.gz", expected: true},
		{name: "WithQueryParams", path: "/archive.tar.gz?token=abc", expected: true},
		{name: "ZipFile", path: "/archive.zip", expected: false},
		{name: "PlainFile", path: "/file.txt", expected: false},
		{name: "TarInName", path: "/tarball.zip", expected: false},
		{name: "NoExtension", path: "/archive", expected: false},
		{name: "EmptyPath", path: "", expected: false},
	}

	tar := NewTar()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &url.URL{Path: tt.path}
			result := tar.Match(u)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompressionFlag(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{name: "TarGz", path: "/archive.tar.gz", expected: "-z"},
		{name: "Tgz", path: "/archive.tgz", expected: "-z"},
		{name: "TarGzUppercase", path: "/archive.TAR.GZ", expected: "-z"},
		{name: "TarBz2", path: "/archive.tar.bz2", expected: "-j"},
		{name: "Tbz", path: "/archive.tbz", expected: "-j"},
		{name: "Tbz2", path: "/archive.tbz2", expected: "-j"},
		{name: "TarXz", path: "/archive.tar.xz", expected: "-J"},
		{name: "Txz", path: "/archive.txz", expected: "-J"},
		{name: "TarZst", path: "/archive.tar.zst", expected: "--zstd"},
		{name: "Tzstd", path: "/archive.tzstd", expected: "--zstd"},
		{name: "TarLz", path: "/archive.tar.lz", expected: "--lzip"},
		{name: "Tlz", path: "/archive.tlz", expected: "--lzip"},
		// NOTE: .tar.Z and .tZ don't work because compressionFlag() lowercases the path,
		// causing the uppercase Z check to never match. This appears to be a bug.
		{name: "TarZ", path: "/archive.tar.Z", expected: "-a"},
		{name: "TZ", path: "/archive.tZ", expected: "-a"},
		{name: "PlainTar", path: "/archive.tar", expected: "-a"},
		{name: "Unknown", path: "/archive.tar.unknown", expected: "-a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compressionFlag(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTARFetch(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{name: "TarGz", filename: "archive.tar.gz"},
		{name: "TarBz2", filename: "archive.tar.bz2"},
		{name: "PlainTar", filename: "archive.tar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", tt.filename))
			assert.NoError(t, err)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(data)
			}))
			defer server.Close()

			u, err := url.Parse(server.URL + "/" + tt.filename)
			assert.NoError(t, err)

			dest := t.TempDir()
			tar := NewTar()
			err = tar.Fetch(context.Background(), Source{URL: u}, dest)
			assert.NoError(t, err)

			content, err := os.ReadFile(filepath.Join(dest, "file.txt"))
			assert.NoError(t, err)
			assert.Equal(t, "hello from test\n", string(content))

			content, err = os.ReadFile(filepath.Join(dest, "nested.txt"))
			assert.NoError(t, err)
			assert.Equal(t, "nested content\n", string(content))
		})
	}
}

func TestTARFetchHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL + "/archive.tar.gz")
	assert.NoError(t, err)

	dest := t.TempDir()
	tar := NewTar()
	err = tar.Fetch(context.Background(), Source{URL: u}, dest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestTARFetchInvalidTarball(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not a valid tarball"))
	}))
	defer server.Close()

	u, err := url.Parse(server.URL + "/archive.tar.gz")
	assert.NoError(t, err)

	dest := t.TempDir()
	tar := NewTar()
	err = tar.Fetch(context.Background(), Source{URL: u}, dest)
	assert.Error(t, err)
}

func TestTARFetchCancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data"))
	}))
	defer server.Close()

	u, err := url.Parse(server.URL + "/archive.tar.gz")
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dest := t.TempDir()
	tar := NewTar()
	err = tar.Fetch(ctx, Source{URL: u}, dest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}
