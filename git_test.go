package getit //nolint:testpackage

import (
	"context"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
)

func TestGitMatch(t *testing.T) {
	tests := []struct {
		name     string
		scheme   string
		expected bool
	}{
		{name: "GitHTTPS", scheme: "git+https", expected: true},
		{name: "GitSSH", scheme: "git+ssh", expected: true},
		{name: "Git", scheme: "git", expected: true},
		{name: "HTTPS", scheme: "https", expected: false},
		{name: "HTTP", scheme: "http", expected: false},
		{name: "SSH", scheme: "ssh", expected: false},
		{name: "File", scheme: "file", expected: false},
		{name: "Empty", scheme: "", expected: false},
	}

	git := NewGit()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &url.URL{Scheme: tt.scheme, Host: "github.com", Path: "/user/repo"}
			result := git.Match(u)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertGitURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "GitHTTPS",
			input:    "git+https://github.com/user/repo",
			expected: "https://github.com/user/repo",
		},
		{
			name:     "GitSSH",
			input:    "git+ssh://github.com/user/repo",
			expected: "git@github.com:user/repo",
		},
		{
			name:     "Git",
			input:    "git://github.com/user/repo",
			expected: "git://github.com/user/repo",
		},
		{
			name:     "WithQueryParams",
			input:    "git+https://github.com/user/repo?ref=main&depth=1",
			expected: "https://github.com/user/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.input)
			assert.NoError(t, err)
			result := convertGitURL(u)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// createTestRepo creates a git repository with test files and returns its path
// along with a helper function for running git commands in that repo.
func createTestRepo(t *testing.T) (repoDir string, runGit func(args ...string)) {
	t.Helper()

	repoDir = t.TempDir()

	runGit = func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		output, err := cmd.CombinedOutput()
		assert.NoError(t, err, "git %v failed: %s", args, output)
	}

	runGit("init", "--initial-branch=master")
	runGit("config", "user.email", "test@test.com")
	runGit("config", "user.name", "Test")

	err := os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("hello from test\n"), 0o644)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(repoDir, "nested.txt"), []byte("nested content\n"), 0o644)
	assert.NoError(t, err)

	runGit("add", ".")
	runGit("commit", "-m", "Initial commit")

	return repoDir, runGit
}

func TestGitFetch(t *testing.T) {
	repoDir, _ := createTestRepo(t)

	u, err := url.Parse("git+file://" + repoDir)
	assert.NoError(t, err)

	dest := t.TempDir()
	git := NewGit()
	err = git.Fetch(context.Background(), Source{URL: u}, dest)
	assert.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dest, "file.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "hello from test\n", string(content))

	content, err = os.ReadFile(filepath.Join(dest, "nested.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "nested content\n", string(content))

	// Verify it's a git repo
	_, err = os.Stat(filepath.Join(dest, ".git"))
	assert.NoError(t, err)
}

func TestGitFetchWithRef(t *testing.T) {
	repoDir, runGit := createTestRepo(t)

	// Create a branch with different content
	runGit("checkout", "-b", "feature-branch")
	err := os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("feature branch content\n"), 0o644)
	assert.NoError(t, err)
	runGit("add", ".")
	runGit("commit", "-m", "Feature commit")
	runGit("checkout", "master")

	// Clone the feature branch
	u, err := url.Parse("git+file://" + repoDir + "?ref=feature-branch")
	assert.NoError(t, err)

	dest := t.TempDir()
	git := NewGit()
	err = git.Fetch(context.Background(), Source{URL: u}, dest)
	assert.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dest, "file.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "feature branch content\n", string(content))
}

func TestGitFetchWithDepth(t *testing.T) {
	repoDir, runGit := createTestRepo(t)

	// Add more commits
	for i := range 5 {
		err := os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("commit "+string(rune('A'+i))+"\n"), 0o644)
		assert.NoError(t, err)
		runGit("add", ".")
		runGit("commit", "-m", "Commit "+string(rune('A'+i)))
	}

	u, err := url.Parse("git+file://" + repoDir + "?depth=1")
	assert.NoError(t, err)

	dest := t.TempDir()
	git := NewGit()
	err = git.Fetch(context.Background(), Source{URL: u}, dest)
	assert.NoError(t, err)

	// Verify shallow clone by checking commit count
	cmd := exec.Command("git", "rev-list", "--count", "HEAD")
	cmd.Dir = dest
	output, err := cmd.Output()
	assert.NoError(t, err)
	assert.Equal(t, "1\n", string(output))
}

func TestGitFetchWithCommit(t *testing.T) {
	repoDir, runGit := createTestRepo(t)
	gitOutput := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		output, err := cmd.Output()
		assert.NoError(t, err)
		return strings.TrimSpace(string(output))
	}

	for _, content := range []string{"second", "third"} {
		assert.NoError(t, os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte(content+"\n"), 0o644))
		runGit("add", ".")
		runGit("commit", "-m", content)
	}
	commit := gitOutput(repoDir, "rev-parse", "HEAD~1")

	tests := []struct {
		name    string
		query   string
		commits string
	}{
		{name: "FullHistory", query: "?commit=" + commit, commits: "2"},
		{name: "Shallow", query: "?commit=" + commit + "&depth=1", commits: "1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse("git+file://" + repoDir + tt.query)
			assert.NoError(t, err)

			dest := filepath.Join(t.TempDir(), "dest")
			err = NewGit().Fetch(context.Background(), Source{URL: u}, dest)
			assert.NoError(t, err)

			content, err := os.ReadFile(filepath.Join(dest, "file.txt"))
			assert.NoError(t, err)
			assert.Equal(t, "second\n", string(content))
			assert.Equal(t, commit, gitOutput(dest, "rev-parse", "HEAD"))
			assert.Equal(t, tt.commits, gitOutput(dest, "rev-list", "--count", "HEAD"))
		})
	}
}

func TestGitFetchWithCommitAndRef(t *testing.T) {
	repoDir, _ := createTestRepo(t)

	u, err := url.Parse("git+file://" + repoDir + "?commit=" + strings.Repeat("a", 40) + "&ref=master")
	assert.NoError(t, err)

	err = NewGit().Fetch(context.Background(), Source{URL: u}, t.TempDir())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be used together")
}

func TestGitRevisions(t *testing.T) {
	repoDir, runGit := createTestRepo(t)
	gitOutput := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		output, err := cmd.Output()
		assert.NoError(t, err)
		return strings.TrimSpace(string(output))
	}
	first := gitOutput(repoDir, "rev-parse", "HEAD")
	assert.NoError(t, os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("second\n"), 0o644))
	runGit("add", ".")
	runGit("commit", "-m", "second")

	// A shallow clone of the ref does not reach the first commit, and git
	// rejects a ref with a commit.
	u, err := url.Parse("git+file://" + repoDir + "?ref=master&depth=1")
	assert.NoError(t, err)
	source := Source{URL: u}
	git := NewGit()

	latest := t.TempDir()
	revision, err := git.FetchRevision(context.Background(), source, latest)
	assert.NoError(t, err)
	assert.Equal(t, gitOutput(repoDir, "rev-parse", "HEAD"), revision)

	earlier := t.TempDir()
	revision, err = git.FetchRevision(context.Background(), git.AtRevision(source, first), earlier)
	assert.NoError(t, err)
	assert.Equal(t, first, revision)
	content, err := os.ReadFile(filepath.Join(earlier, "file.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "hello from test\n", string(content))
}

func TestGitFetchCancelledContext(t *testing.T) {
	repoDir, _ := createTestRepo(t)

	u, err := url.Parse("git+file://" + repoDir)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dest := t.TempDir()
	git := NewGit()
	err = git.Fetch(ctx, Source{URL: u}, dest)
	assert.Error(t, err)
}

func TestGitFetchCancelKillsProcessGroup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires sh")
	}

	// Fake git that spawns a grandchild holding the output pipe open. If
	// cancellation only kills the direct child, Fetch blocks until the
	// grandchild exits.
	binDir := t.TempDir()
	script := "#!/bin/sh\nsleep 60 &\nwait\n"
	assert.NoError(t, os.WriteFile(filepath.Join(binDir, "git"), []byte(script), 0o755)) //nolint:gosec
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	u, err := url.Parse("git+https://example.com/user/repo")
	assert.NoError(t, err)

	start := time.Now()
	err = NewGit().Fetch(ctx, Source{URL: u}, t.TempDir())
	assert.Error(t, err)
	assert.True(t, time.Since(start) < 5*time.Second, "Fetch blocked on an orphaned grandchild for %s", time.Since(start))
}

func TestGitFetchInvalidRepo(t *testing.T) {
	u, err := url.Parse("git+file:///nonexistent/repo/path")
	assert.NoError(t, err)

	dest := t.TempDir()
	git := NewGit()
	err = git.Fetch(context.Background(), Source{URL: u}, dest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "git clone failed")
}
