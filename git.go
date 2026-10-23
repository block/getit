package getit

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"strings"

	"github.com/kballard/go-shellquote"
)

// commitHash matches a full SHA-1 or SHA-256 commit hash.
var commitHash = regexp.MustCompile(`^([0-9a-f]{40}|[0-9a-f]{64})$`)

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
// A commit is a full commit hash, checked out detached. It cannot be combined
// with ref, and needs a server that allows fetching a commit by hash, such as
// any server speaking git protocol v2.
type Git struct{}

var _ Resolver = (*Git)(nil)

func NewGit() *Git { return &Git{} }

func (g *Git) Match(source *url.URL) bool {
	return source.Scheme == "git+https" || source.Scheme == "git+ssh" || source.Scheme == "git"
}

func (g *Git) Fetch(ctx context.Context, source Source, dest string) error {
	query := source.URL.Query()
	repoURL := convertGitURL(source.URL)
	if commit := query.Get("commit"); commit != "" {
		return fetchCommit(ctx, repoURL, commit, query, dest)
	}

	args := []string{"clone"}
	if depth := query.Get("depth"); depth != "" {
		args = append(args, "--depth", depth)
	}
	if ref := query.Get("ref"); ref != "" {
		args = append(args, "--branch", ref)
	}
	args = append(args, repoURL, dest)
	return runGit(ctx, "", args...)
}

// fetchCommit fetches commit into a new repository at dest and checks it out.
func fetchCommit(ctx context.Context, repoURL, commit string, query url.Values, dest string) error {
	if query.Get("ref") != "" {
		return errors.New("git source cannot set both ref and commit")
	}
	if !commitHash.MatchString(commit) {
		return fmt.Errorf("git commit %q is not a full commit hash", commit)
	}
	fetch := []string{"fetch", "--quiet"}
	if depth := query.Get("depth"); depth != "" {
		fetch = append(fetch, "--depth", depth)
	}
	fetch = append(fetch, "origin", commit)
	if err := runGit(ctx, "", "init", "--quiet", dest); err != nil {
		return err
	}
	for _, args := range [][]string{
		{"remote", "add", "origin", repoURL},
		fetch,
		{"checkout", "--quiet", "--detach", "FETCH_HEAD"},
	} {
		if err := runGit(ctx, dest, args...); err != nil {
			return err
		}
	}
	return nil
}

// runGit runs a git command in dir, or the current directory if dir is empty.
func runGit(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	killProcessGroup(cmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		argsStr := shellquote.Join(args...)
		return fmt.Errorf("git %s failed: git %s: %w: %s", args[0], argsStr, err, output)
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
