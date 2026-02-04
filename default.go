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
