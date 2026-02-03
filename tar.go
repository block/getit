package getit

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
)

// The TAR [Resolver] knows how to unpack tarballs.
type TAR struct{}

var _ Resolver = (*TAR)(nil)

func NewTAR() *TAR { return &TAR{} }

var tarRe = regexp.MustCompile(`(\.tar(\.[a-z]+)?)|(\.tbz|\.tbz2|\.txz|\.tzstd|\.tlz|\.tZ|\.tgz)`)

func (t *TAR) Match(source *url.URL) bool {
	return tarRe.MatchString(source.Path)
}

func (t *TAR) Fetch(ctx context.Context, source Source, dest string) error {
	if err := os.MkdirAll(dest, 0750); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}
	args := []string{"-x", "-C", dest}
	args = append(args, compressionFlag(source.URL.Path))
	return FetchIntoPipe(ctx, source.URL, "tar", args...)
}

func compressionFlag(path string) string {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return "-z"
	case strings.HasSuffix(lower, ".tar.bz2"), strings.HasSuffix(lower, ".tbz"), strings.HasSuffix(lower, ".tbz2"):
		return "-j"
	case strings.HasSuffix(lower, ".tar.xz"), strings.HasSuffix(lower, ".txz"):
		return "-J"
	case strings.HasSuffix(lower, ".tar.zst"), strings.HasSuffix(lower, ".tzstd"):
		return "--zstd"
	case strings.HasSuffix(lower, ".tar.lz"), strings.HasSuffix(lower, ".tlz"):
		return "--lzip"
	case strings.HasSuffix(lower, ".tar.Z"), strings.HasSuffix(lower, ".tZ"):
		return "-Z"
	default:
		return "-a"
	}
}
