// Package getit provides a simple API for fetching archives.
package getit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
)

// Mapper maps one form of a source to another.
//
// eg.
//
//	github.com/user/repo -> https://github.com/user/repo.git
//	user/repo -> https://github.com/user/repo.git
type Mapper func(source string) (string, bool)

type Resolver interface {
	// Match returns true if this Resolver can handle the given source URL.
	Match(source *url.URL) bool
	// Fetch an archive from a source and unpacks it to a destination.
	Fetch(ctx context.Context, source Source, dest string) error
}

// ErrNoRevisions is returned for a source whose resolver is not a
// [RevisionResolver].
var ErrNoRevisions = errors.New("source does not support revisions")

// RevisionResolver is a [Resolver] that can fetch a source again exactly as
// it was.
type RevisionResolver interface {
	Resolver
	// FetchRevision fetches like Fetch and returns the revision fetched.
	FetchRevision(ctx context.Context, source Source, dest string) (string, error)
	// AtRevision returns source pinned to a revision returned by FetchRevision.
	AtRevision(source Source, revision string) Source
}

// Source is a resolved source with optional sub-directory.
type Source struct {
	URL    *url.URL
	SubDir string
}

// Fetcher retrieves archives from a pluggable source.
//
// All sources support an optional subdirectory, specified via appending a //<subdir> to the URL path:
//
//	git+ssh://host/path/to/repo.git//path/to/subdir
//	https://host/path/to/archive.tgz//path/to/subdir
type Fetcher struct {
	mappers   []Mapper
	resolvers []Resolver
}

func New(resolvers []Resolver, mappers []Mapper) *Fetcher {
	return &Fetcher{
		mappers:   mappers,
		resolvers: resolvers,
	}
}

// Resolve a source string to a Source and URL.
func (f *Fetcher) Resolve(source string) (Resolver, Source, error) {
	for _, mapper := range f.mappers {
		if mapped, ok := mapper(source); ok {
			source = mapped
			if _, err := url.Parse(source); err != nil {
				panic("mapper did not produce a valid URL: " + source)
			}
			break
		}
	}
	u, err := url.Parse(source)
	if err != nil {
		return nil, Source{}, fmt.Errorf("invalid source %q", source)
	}
	for _, resolver := range f.resolvers {
		if !resolver.Match(u) {
			continue
		}
		base, subdir, ok := strings.Cut(u.Path, "//")
		if ok {
			// Strip subdir, if any
			nu := *u
			nu.Path = base
			u = &nu
		}
		return resolver, Source{
			URL:    u,
			SubDir: subdir,
		}, nil
	}
	return nil, Source{}, fmt.Errorf("unsupported source: %s", u)
}

// Fetch fetches an archive from a source and unpacks it to a destination.
func (f *Fetcher) Fetch(ctx context.Context, source, dest string) error {
	src, u, err := f.Resolve(source)
	if err != nil {
		return err
	}
	if err := src.Fetch(ctx, u, dest); err != nil {
		return fmt.Errorf("fetching %s: %w", source, err)
	}
	return nil
}

// FetchRevision fetches like Fetch and returns the revision fetched, to pass
// to AtRevision later.
//
// It returns [ErrNoRevisions] if the source's resolver is not a
// [RevisionResolver].
func (f *Fetcher) FetchRevision(ctx context.Context, source, dest string) (string, error) {
	resolver, src, err := f.resolveRevisions(source)
	if err != nil {
		return "", err
	}
	revision, err := resolver.FetchRevision(ctx, src, dest)
	if err != nil {
		return "", fmt.Errorf("fetching %s: %w", source, err)
	}
	return revision, nil
}

// AtRevision returns source pinned to a revision returned by FetchRevision.
//
// It returns [ErrNoRevisions] if the source's resolver is not a
// [RevisionResolver].
func (f *Fetcher) AtRevision(source, revision string) (string, error) {
	resolver, src, err := f.resolveRevisions(source)
	if err != nil {
		return "", err
	}
	return sourceString(resolver.AtRevision(src, revision)), nil
}

// sourceString returns source in the form Resolve accepts.
func sourceString(source Source) string {
	u := *source.URL
	if source.SubDir != "" {
		u.Path += "//" + source.SubDir
	}
	return u.String()
}

func (f *Fetcher) resolveRevisions(source string) (RevisionResolver, Source, error) {
	resolver, src, err := f.Resolve(source)
	if err != nil {
		return nil, Source{}, err
	}
	revisions, ok := resolver.(RevisionResolver)
	if !ok {
		return nil, Source{}, fmt.Errorf("%s: %w", source, ErrNoRevisions)
	}
	return revisions, src, nil
}

// FetchIntoPipe retrieves the given URL using Go's HTTP library then pipes it into the input of the given command.
func FetchIntoPipe(ctx context.Context, u *url.URL, cmd string, args ...string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", u, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetching %s: %s", u, resp.Status)
	}

	stderr := &bytes.Buffer{}
	c := exec.CommandContext(ctx, cmd, args...)
	c.Stdin = resp.Body
	c.Stderr = stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("%s failed: %w: %s", cmd, err, stderr.String())
	}
	return nil
}
