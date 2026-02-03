package getit

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestGitHub(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected string
		ok       bool
	}{
		{
			name:     "SimpleRepoPath",
			source:   "github.com/user/repo",
			expected: "https://github.com/user/repo",
			ok:       true,
		},
		{
			name:     "RepoWithSubpath",
			source:   "github.com/user/repo/path/to/file",
			expected: "https://github.com/user/repo/path/to/file",
			ok:       true,
		},
		{
			name:     "RepoWithQueryParam",
			source:   "github.com/user/repo?ref=main",
			expected: "https://github.com/user/repo?ref=main",
			ok:       true,
		},
		{
			name:     "RepoWithAnchor",
			source:   "github.com/user/repo#readme",
			expected: "https://github.com/user/repo#readme",
			ok:       true,
		},
		{
			name:     "RepoWithQueryAndAnchor",
			source:   "github.com/user/repo?ref=main#section",
			expected: "https://github.com/user/repo?ref=main#section",
			ok:       true,
		},
		{
			name:   "AlreadyHasScheme",
			source: "https://github.com/user/repo",
		},
		{
			name:   "DifferentDomain",
			source: "gitlab.com/user/repo",
		},
		{
			name:   "OrgRepoOnly",
			source: "user/repo",
		},
		{
			name:   "EmptyString",
			source: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := GitHub(tt.source)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGitHubOrgRepo(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected string
		ok       bool
	}{
		{
			name:     "OrgRepoWithQuery",
			source:   "user/repo?ref=main",
			expected: "https://github.com/user/repo?ref=main",
			ok:       true,
		},
		{
			name:     "OrgRepoWithAnchor",
			source:   "user/repo#readme",
			expected: "https://github.com/user/repo#readme",
			ok:       true,
		},
		{
			name:     "OrgRepoWithQueryAndAnchor",
			source:   "user/repo?ref=main#section",
			expected: "https://github.com/user/repo?ref=main#section",
			ok:       true,
		},
		{
			name:     "OrgRepoWithHyphen",
			source:   "my-org/my-repo?ref=v1",
			expected: "https://github.com/my-org/my-repo?ref=v1",
			ok:       true,
		},
		{
			name:     "OrgRepoWithUnderscore",
			source:   "my_org/my_repo#anchor",
			expected: "https://github.com/my_org/my_repo#anchor",
			ok:       true,
		},
		{
			name:     "OrgRepoWithNumbers",
			source:   "org123/repo456?param=1",
			expected: "https://github.com/org123/repo456?param=1",
			ok:       true,
		},
		{
			name:   "OrgRepoWithoutQueryOrAnchor",
			source: "user/repo",
		},
		{
			name:   "FullGitHubURL",
			source: "github.com/user/repo?ref=main",
		},
		{
			name:   "SingleWord",
			source: "repo?ref=main",
		},
		{
			name:   "EmptyString",
			source: "",
		},
		{
			name:   "ThreeSegments",
			source: "org/repo/path?ref=main",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := GitHubOrgRepo(tt.source)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSingleGitHubOrg(t *testing.T) {
	tests := []struct {
		name     string
		org      string
		source   string
		expected string
		ok       bool
	}{
		{
			name:     "RepoWithQuery",
			org:      "myorg",
			source:   "myrepo?ref=main",
			expected: "https://github.com/myorg/myrepo?ref=main",
			ok:       true,
		},
		{
			name:     "RepoWithAnchor",
			org:      "myorg",
			source:   "myrepo#readme",
			expected: "https://github.com/myorg/myrepo#readme",
			ok:       true,
		},
		{
			name:     "RepoWithQueryAndAnchor",
			org:      "myorg",
			source:   "myrepo?ref=v1#section",
			expected: "https://github.com/myorg/myrepo?ref=v1#section",
			ok:       true,
		},
		{
			name:     "RepoWithHyphen",
			org:      "my-org",
			source:   "my-repo?ref=main",
			expected: "https://github.com/my-org/my-repo?ref=main",
			ok:       true,
		},
		{
			name:     "RepoWithUnderscore",
			org:      "my_org",
			source:   "my_repo#anchor",
			expected: "https://github.com/my_org/my_repo#anchor",
			ok:       true,
		},
		{
			name:     "DifferentOrgs",
			org:      "orgA",
			source:   "repo?ref=main",
			expected: "https://github.com/orgA/repo?ref=main",
			ok:       true,
		},
		{
			name:   "RepoWithoutQueryOrAnchor",
			org:    "myorg",
			source: "myrepo",
		},
		{
			name:   "OrgSlashRepo",
			org:    "myorg",
			source: "other/repo?ref=main",
		},
		{
			name:   "EmptyString",
			org:    "myorg",
			source: "",
		},
		{
			name:   "FullURL",
			org:    "myorg",
			source: "https://github.com/myorg/repo?ref=main",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := SingleGitHubOrg(tt.org)
			result, ok := mapper(tt.source)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.expected, result)
		})
	}
}
