package getit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/alecthomas/assert/v2"
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
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("hello world"))
			},
			cmd:  "cat",
			args: nil,
		},
		{
			name: "SuccessfulFetchWithArgs",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("line1\nline2\nline3"))
			},
			cmd:  "wc",
			args: []string{"-l"},
		},
		{
			name: "HTTPNotFound",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			cmd:         "cat",
			expectedErr: "404 Not Found",
		},
		{
			name: "HTTPInternalServerError",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			cmd:         "cat",
			expectedErr: "500 Internal Server Error",
		},
		{
			name: "CommandNotFound",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("data"))
			},
			cmd:         "nonexistent-command-that-does-not-exist",
			expectedErr: "nonexistent-command-that-does-not-exist failed:",
		},
		{
			name: "CommandFails",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("data"))
			},
			cmd:         "false",
			expectedErr: "false failed:",
		},
		{
			name: "CancelledContext",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
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

			err = FetchIntoPipe(ctx, u, tt.cmd, tt.args...)

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
	err := FetchIntoPipe(ctx, u, "cat")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fetching")
}
