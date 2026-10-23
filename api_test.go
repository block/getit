package getit_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/alecthomas/assert/v2"

	"github.com/block/getit"
)

func TestFetchIntoPipe(t *testing.T) {
	tests := []struct {
		name          string
		serverHandler http.HandlerFunc
		cmd           string
		args          []string
		expectedErr   string
		cancelContext bool
	}{
		{
			name: "SuccessfulFetch",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("hello world"))
			},
			cmd:  "cat",
			args: nil,
		},
		{
			name: "SuccessfulFetchWithArgs",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("line1\nline2\nline3"))
			},
			cmd:  "wc",
			args: []string{"-l"},
		},
		{
			name: "HTTPNotFound",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			cmd:         "cat",
			expectedErr: "404 Not Found",
		},
		{
			name: "HTTPInternalServerError",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			cmd:         "cat",
			expectedErr: "500 Internal Server Error",
		},
		{
			name: "CommandNotFound",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("data"))
			},
			cmd:         "nonexistent-command-that-does-not-exist",
			expectedErr: "nonexistent-command-that-does-not-exist failed:",
		},
		{
			name: "CommandFails",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("data"))
			},
			cmd:         "false",
			expectedErr: "false failed:",
		},
		{
			name: "CancelledContext",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("data"))
			},
			cmd:           "cat",
			cancelContext: true,
			expectedErr:   "context canceled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			u, err := url.Parse(server.URL)
			assert.NoError(t, err)

			ctx := context.Background()
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			err = getit.FetchIntoPipe(ctx, u, tt.cmd, tt.args...)

			if tt.expectedErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFetchIntoPipeInvalidURL(t *testing.T) {
	ctx := context.Background()
	u := &url.URL{Scheme: "http", Host: "localhost:99999"}
	err := getit.FetchIntoPipe(ctx, u, "cat")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fetching")
}

func TestAtRevision(t *testing.T) {
	const commit = "9f3c1e7f6b3a5c0d2e4f6a8b0c2d4e6f8a0b2c4d"
	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{name: "Git", source: "git+ssh://github.com/org/repo", expected: "git+ssh://github.com/org/repo?commit=" + commit + "&depth=1"},
		{name: "SubDirAndRef", source: "git+https://github.com/org/repo//app?ref=main&depth=5", expected: "git+https://github.com/org/repo//app?commit=" + commit + "&depth=1"},
		{name: "Shorthand", source: "org/repo?ref=main", expected: "git+https://github.com/org/repo?commit=" + commit + "&depth=1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pinned, err := getit.AtRevision(tt.source, commit)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, pinned)

			_, source, err := getit.Resolve(pinned)
			assert.NoError(t, err)
			assert.Equal(t, commit, source.URL.Query().Get("commit"))
		})
	}
}

func TestRevisionsUnsupported(t *testing.T) {
	_, err := getit.AtRevision("https://example.com/archive.tar.gz", "abc")
	assert.IsError(t, err, getit.ErrNoRevisions)
	_, err = getit.FetchRevision(context.Background(), t.TempDir(), t.TempDir())
	assert.IsError(t, err, getit.ErrNoRevisions)
}
