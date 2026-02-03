package getit

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

type ZIP struct{}

func NewZIP() *ZIP {
	return &ZIP{}
}

var _ Resolver = (*ZIP)(nil)

func (z *ZIP) Match(source *url.URL) bool {
	return strings.HasSuffix(source.Path, ".zip")
}

func (z *ZIP) Fetch(ctx context.Context, source Source, dest string) error {
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL.String(), nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", source.URL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetching %s: %s", source.URL, resp.Status)
	}

	// Write the zip to a temporary file
	zip, err := os.CreateTemp("", "zip-*.zip")
	if err != nil {
		return fmt.Errorf("creating temporary file: %w", err)
	}
	defer zip.Close()
	defer os.Remove(zip.Name())
	if _, err = io.Copy(zip, resp.Body); err != nil {
		return fmt.Errorf("copying response body to temporary file: %w", err)
	}
	if err = zip.Close(); err != nil {
		return fmt.Errorf("closing temporary file: %w", err)
	}

	// Unzip
	stderr := &bytes.Buffer{}
	cmd := exec.CommandContext(ctx, "unzip", "-d", dest, zip.Name())
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unzip %s: %w: %s", zip.Name(), err, stderr)
	}
	return nil
}
