# A Go library for fetching and unpacking archives from multiple sources.

This is a bit like hashicorp/go-getter, but much simpler, and with less supported protocols (so far).

## Features

- **Git repositories**: Clone from git://, git+ssh://, or git+https:// URLs with optional ref, commit and depth parameters
- **TAR archives**: Fetch and extract .tar, .tar.gz, .tar.bz2, .tar.xz, and other compressed tarballs
- **ZIP archives**: Download and unzip .zip files
- **GitHub shortcuts**: Map shorthand URLs like `user/repo` or `github.com/user/repo` to full git URLs
- **Subdirectory support**: Extract specific subdirectories using `//` delimiter (e.g., `https://example.com/archive.tar.gz//subdir`)

## Usage

```go
import "github.com/block/getit"

// Create a fetcher with resolvers and mappers
fetcher := getit.New(
    []getit.Resolver{
        getit.NewGit(),
        getit.NewTAR(),
        getit.NewZIP(),
    },
    []getit.Mapper{
        getit.GitHub,
        getit.GitHubOrgRepo,
        getit.SingleGitHubOrg("myorg"),
    },
)

// Fetch an archive
err := fetcher.Fetch(ctx, "user/repo?ref=main&depth=1", "./destination")

// Fetch an archive and the revision fetched, then fetch that revision again later
revision, err := fetcher.FetchRevision(ctx, "user/repo?ref=main", "./destination")
pinned, err := fetcher.AtRevision("user/repo?ref=main", revision)
err = fetcher.Fetch(ctx, pinned, "./again")
```

Only git sources support revisions, which are commit hashes. Other sources return `ErrNoRevisions`.
