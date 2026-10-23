package getit

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	"github.com/kballard/go-shellquote"
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
//	commit=<commit>
//	depth=<depth>
//
// A commit is checked out detached and cannot be combined with ref. It needs
// git 2.49 or later, and a server that allows fetching a commit by hash, such
// as any server speaking git protocol v2.
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
	if commit := source.URL.Query().Get("commit"); commit != "" {
		args = append(args, "--revision", commit)
	}

	repoURL := convertGitURL(source.URL)
	args = append(args, repoURL, dest)

	cmd := exec.CommandContext(ctx, "git", args...)
	killProcessGroup(cmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		argsStr := shellquote.Join(args...)
		return fmt.Errorf("git clone failed: git %s: %w: %s", argsStr, err, output)
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
