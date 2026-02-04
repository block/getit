package getit

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// File is a [Resolver] that copies local directories.
//
// The URL format supported is:
//
//	file:///absolute/path/to/dir
//	file://relative/path/to/dir
type File struct{}

var _ Resolver = (*File)(nil)

func NewFile() *File { return &File{} }

func (f *File) Match(source *url.URL) bool {
	return source.Scheme == "file"
}

func (f *File) Fetch(ctx context.Context, source Source, dest string) error {
	srcPath := source.URL.Path
	if source.URL.Host != "" {
		srcPath = filepath.Join(source.URL.Host, srcPath)
	}

	info, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("stat %s: %w", srcPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", srcPath)
	}

	if err := copyDir(ctx, srcPath, dest); err != nil {
		return fmt.Errorf("copying %s: %w", srcPath, err)
	}
	return nil
}

func copyDir(ctx context.Context, src, dest string) error {
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", path, err)
		}
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("context: %w", err)
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("rel path %s: %w", path, err)
		}
		destPath := filepath.Join(dest, relPath)

		if d.Type()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("readlink %s: %w", path, err)
			}
			return os.Symlink(target, destPath)
		}

		if d.IsDir() {
			return os.MkdirAll(destPath, 0750)
		}

		return copyFile(path, destPath)
	})
	if err != nil {
		return fmt.Errorf("walk %s: %w", src, err)
	}
	return nil
}

func copyFile(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %s: %w", src, err)
	}
	defer srcFile.Close()

	info, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("stat %s: %w", src, err)
	}

	destFile, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return fmt.Errorf("create %s: %w", dest, err)
	}
	defer destFile.Close()

	if _, err = io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("copy to %s: %w", dest, err)
	}
	return nil
}

// FilePath is a [Mapper] that maps filesystem paths to file:// URLs.
//
// It handles absolute paths, relative paths (./..., ../...), home-relative paths (~/...),
// and bare directory names. The path must exist and be a directory.
func FilePath(source string) (string, bool) {
	if source == "" {
		return "", false
	}

	if strings.HasPrefix(source, "file://") {
		return source, true
	}

	path := source

	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", false
		}
		path = filepath.Join(home, path[2:])
	}

	if !filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", false
		}
		path = abs
	}

	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", false
	}

	// Resolve symlinks for canonical path
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", false
	}

	return "file://" + path, true
}
