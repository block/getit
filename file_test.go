package getit_test

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"

	"github.com/block/getit"
)

func TestFileMatch(t *testing.T) {
	tests := []struct {
		name     string
		scheme   string
		expected bool
	}{
		{name: "File", scheme: "file", expected: true},
		{name: "HTTPS", scheme: "https", expected: false},
		{name: "Git", scheme: "git", expected: false},
		{name: "Empty", scheme: "", expected: false},
	}

	f := getit.NewFile()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &url.URL{Scheme: tt.scheme, Path: "/some/path"}
			result := f.Match(u)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFileFetch(t *testing.T) {
	srcDir := t.TempDir()
	err := os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("hello\n"), 0o644)
	assert.NoError(t, err)
	err = os.MkdirAll(filepath.Join(srcDir, "subdir"), 0o755)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(srcDir, "subdir", "nested.txt"), []byte("nested\n"), 0o644)
	assert.NoError(t, err)

	u, err := url.Parse("file://" + srcDir)
	assert.NoError(t, err)

	dest := t.TempDir()
	f := getit.NewFile()
	err = f.Fetch(context.Background(), getit.Source{URL: u}, dest)
	assert.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dest, "file.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "hello\n", string(content))

	content, err = os.ReadFile(filepath.Join(dest, "subdir", "nested.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "nested\n", string(content))
}

func TestFileFetchNonExistent(t *testing.T) {
	u, err := url.Parse("file:///nonexistent/path/to/dir")
	assert.NoError(t, err)

	dest := t.TempDir()
	f := getit.NewFile()
	err = f.Fetch(context.Background(), getit.Source{URL: u}, dest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stat")
}

func TestFileFetchNotDirectory(t *testing.T) {
	srcDir := t.TempDir()
	filePath := filepath.Join(srcDir, "file.txt")
	err := os.WriteFile(filePath, []byte("hello\n"), 0o644)
	assert.NoError(t, err)

	u, err := url.Parse("file://" + filePath)
	assert.NoError(t, err)

	dest := t.TempDir()
	f := getit.NewFile()
	err = f.Fetch(context.Background(), getit.Source{URL: u}, dest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a directory")
}

func TestFileFetchCancelledContext(t *testing.T) {
	srcDir := t.TempDir()
	for i := range 100 {
		err := os.WriteFile(filepath.Join(srcDir, "file"+string(rune('0'+i))+".txt"), []byte("content\n"), 0o644)
		assert.NoError(t, err)
	}

	u, err := url.Parse("file://" + srcDir)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dest := t.TempDir()
	f := getit.NewFile()
	err = f.Fetch(ctx, getit.Source{URL: u}, dest)
	assert.Error(t, err)
}

func TestFilePath(t *testing.T) {
	existingDir := t.TempDir()
	existingDirResolved, err := filepath.EvalSymlinks(existingDir)
	assert.NoError(t, err)

	cwd, err := os.Getwd()
	assert.NoError(t, err)
	cwdResolved, err := filepath.EvalSymlinks(cwd)
	assert.NoError(t, err)

	tests := []struct {
		name     string
		source   string
		expected string
		ok       bool
	}{
		{
			name:     "AbsolutePath",
			source:   existingDir,
			expected: "file://" + existingDirResolved,
			ok:       true,
		},
		{
			name:     "FileURLPassthrough",
			source:   "file://" + existingDir,
			expected: "file://" + existingDir,
			ok:       true,
		},
		{
			name:     "CurrentDir",
			source:   ".",
			expected: "file://" + cwdResolved,
			ok:       true,
		},
		{
			name:   "NonExistentPath",
			source: "/nonexistent/path/to/dir",
		},
		{
			name:   "GitHubURL",
			source: "github.com/user/repo",
		},
		{
			name:   "OrgRepo",
			source: "user/repo",
		},
		{
			name:   "HTTPSUrl",
			source: "https://example.com/file.tar.gz",
		},
		{
			name:   "EmptyString",
			source: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := getit.FilePath(tt.source)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFilePathRelative(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.MkdirAll(subDir, 0o755)
	assert.NoError(t, err)

	subDirResolved, err := filepath.EvalSymlinks(subDir)
	assert.NoError(t, err)

	t.Chdir(tmpDir)

	result, ok := getit.FilePath("./subdir")
	assert.True(t, ok)
	assert.Equal(t, "file://"+subDirResolved, result)
}

func TestFileFetchSymlinks(t *testing.T) {
	srcDir := t.TempDir()

	err := os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("hello\n"), 0o644)
	assert.NoError(t, err)

	err = os.Symlink("file.txt", filepath.Join(srcDir, "link.txt"))
	assert.NoError(t, err)

	err = os.MkdirAll(filepath.Join(srcDir, "subdir"), 0o755)
	assert.NoError(t, err)
	err = os.Symlink("subdir", filepath.Join(srcDir, "linkdir"))
	assert.NoError(t, err)

	u, err := url.Parse("file://" + srcDir)
	assert.NoError(t, err)

	dest := t.TempDir()
	f := getit.NewFile()
	err = f.Fetch(context.Background(), getit.Source{URL: u}, dest)
	assert.NoError(t, err)

	// Verify the symlink to file is preserved
	linkTarget, err := os.Readlink(filepath.Join(dest, "link.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "file.txt", linkTarget)

	// Verify the symlink to directory is preserved
	linkTarget, err = os.Readlink(filepath.Join(dest, "linkdir"))
	assert.NoError(t, err)
	assert.Equal(t, "subdir", linkTarget)

	// Verify we can still read through the symlink
	content, err := os.ReadFile(filepath.Join(dest, "link.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "hello\n", string(content))
}
