package getit

import (
	"regexp"
	"strings"
)

// GitHub is a [Mapper] that supports shorthand GitHub URLs with no scheme or org/repo.
//
// Query parameters and anchors are preserved.
func GitHub(source string) (string, bool) {
	if strings.HasPrefix(source, "github.com/") {
		return "git+https://" + source, true
	}
	if strings.HasPrefix(source, "https://github.com/") {
		return "git+" + source, true
	}
	return "", false
}

var gitHubOrgRe = regexp.MustCompile(`^([a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+)([?#].*)`)

// GitHubOrgRepo is a [Mapper] that supports shorthand GitHub URLs with org/repo.
//
// Query parameters and anchors are preserved.
func GitHubOrgRepo(source string) (string, bool) {
	if gitHubOrgRe.MatchString(source) {
		return gitHubOrgRe.ReplaceAllString(source, `git+https://github.com/$1$2`), true
	}
	return "", false
}

var singleGitHubOrg = regexp.MustCompile(`^([a-zA-Z0-9_-]+)([#?].*)`)

// SingleGitHubOrg is a [Mapper] that supports shorthand GitHub URLs as just repo.
//
// Query parameters and anchors are preserved.
func SingleGitHubOrg(org string) Mapper {
	return func(source string) (string, bool) {
		if singleGitHubOrg.MatchString(source) {
			return singleGitHubOrg.ReplaceAllString(source, `git+https://github.com/`+org+`/$1$2`), true
		}
		return "", false
	}
}
