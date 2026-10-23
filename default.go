package getit

import "context"

// Default Fetcher with built-in resolvers and mappers.
var Default = New(
	[]Resolver{
		NewFile(),
		NewGit(),
		NewTAR(),
		NewZIP(),
	},
	[]Mapper{
		GitHub,
		GitHubOrgRepo,
		FilePath,
	},
)

// Resolve a source string to a Source and URL.
func Resolve(source string) (Resolver, Source, error) { return Default.Resolve(source) }

// Fetch fetches an archive from a source and unpacks it to a destination.
func Fetch(ctx context.Context, source, dest string) error { return Default.Fetch(ctx, source, dest) }

// FetchRevision fetches like Fetch and returns the revision fetched.
func FetchRevision(ctx context.Context, source, dest string) (string, error) {
	return Default.FetchRevision(ctx, source, dest)
}

// AtRevision returns source pinned to a revision returned by FetchRevision.
func AtRevision(source, revision string) (string, error) { return Default.AtRevision(source, revision) }
