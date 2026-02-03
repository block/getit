package getit

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

// The Git [Resolver] uses Git repositories as archive sources, cloning directly.
//
// The URL format supported is:
//
//	git://host/path/to/repo
//	git+ssh://host/path/to/repo
//	git+https://host/path/to/repo
//
// All forms support the following query parameters that control cloning behaviour:
//
//	ref=<ref>
//	depth=<depth>
type Git struct{}

var _ Resolver = (*Git)(nil)

func NewGit() *Git { return &Git{} }

func (g *Git) Match(source *url.URL) bool {
	return source.Scheme == "git+https" || source.Scheme == "git+ssh" || source.Scheme == "git"
}

func (g *Git) Fetch(ctx context.Context, source Source, dest string) error {
	args := []string{"clone"}
	if depth := source.URL.Query().Get("depth"); depth != "" {
		args = append(args, "--depth", depth)
	}
	if ref := source.URL.Query().Get("ref"); ref != "" {
		args = append(args, "--branch", ref)
	}

	repoURL := convertGitURL(source.URL)
	args = append(args, repoURL, dest)

	stderr := &bytes.Buffer{}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w: %s", err, stderr)
	}
	return nil
}

// convertGitURL converts a getit git URL to a standard git URL.
// git+https://host/path -> https://host/path
// git+ssh://host/path -> git@host:path (SCP-style)
// git://host/path -> git://host/path
func convertGitURL(u *url.URL) string {
	clone := *u
	clone.RawQuery = ""

	if clone.Scheme == "git+ssh" {
		path := strings.TrimPrefix(clone.Path, "/")
		return "git@" + clone.Host + ":" + path
	}

	clone.Scheme = strings.TrimPrefix(clone.Scheme, "git+")
	return clone.String()
}
