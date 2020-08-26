package latest

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/go-github/github"
	"github.com/hashicorp/go-version"
)

// FixVersionStrFunc is function to fix version string
// so that it can be interpreted as Semantic Versiongin by
// http://godoc.org/github.com/hashicorp/go-version
type FixVersionStrFunc func(string) string

// TagFilterFunc is fucntion to filter unexpected tags
// from GitHub. Check a given tag as string (before FixVersionStr)
// and return bool. If it's expected, return true. If not return false.
type TagFilterFunc func(string) bool

var (
	defaultFixVersionStrFunc FixVersionStrFunc
	defaultTagFilterFunc     TagFilterFunc
)

func init() {
	defaultFixVersionStrFunc = fixNothing()
	defaultTagFilterFunc = filterNothing()
}

// GithubTag is used to fetch version(tag) information from Github.
type GithubTag struct {
	// Owner and Repository are GitHub owner name and its repository name.
	// e.g., If you want to check https://github.com/tcnksm/ghr version
	// Repository is `ghr`, and Owner is `tcnksm`.
	Owner      string
	Repository string

	// FixVersionStrFunc is function to fix version string (in this case tag
	// name string) on GitHub so that it can be interpreted as Semantic Versioning
	// by hashicorp/go-version. By default, it does nothing.
	FixVersionStrFunc FixVersionStrFunc

	// TagFilterFunc is function to filter tags from GitHub. Some project includes
	// tags you don't want to use for version comparing. It can be used to exclude
	// such tags. By default, it does nothing.
	TagFilterFunc TagFilterFunc

	// URL & Token is used for GitHub Enterprise
	URL   string
	Token string
}

func (g *GithubTag) fixVersionStrFunc() FixVersionStrFunc {
	if g.FixVersionStrFunc == nil {
		return defaultFixVersionStrFunc
	}

	return g.FixVersionStrFunc
}

func (g *GithubTag) tagFilterFunc() TagFilterFunc {
	if g.TagFilterFunc == nil {
		return defaultTagFilterFunc
	}

	return g.TagFilterFunc
}

// fixNothing does nothing. This is a default function of FixVersionStrFunc.
func fixNothing() FixVersionStrFunc {
	return func(s string) string {
		return s
	}
}

func filterNothing() TagFilterFunc {
	return func(s string) bool {
		return true
	}
}

// DeleteFrontV delete first `v` charactor on version string.
// For example version name `v0.1.1` becomes `0.1.1`
func DeleteFrontV() FixVersionStrFunc {
	return func(s string) string {
		return strings.Replace(s, "v", "", 1)
	}
}

func (g *GithubTag) newClient() *github.Client {
	client := github.NewClient(nil)
	if g.URL != "" {
		client.BaseURL, _ = url.Parse(g.URL)
	}
	return client
}

func (g *GithubTag) Validate() error {

	if len(g.Repository) == 0 {
		return fmt.Errorf("GitHub repository name must be set")
	}

	if len(g.Owner) == 0 {
		return fmt.Errorf("GitHub owner name must be set")
	}

	if g.URL != "" {
		if _, err := url.Parse(g.URL); err != nil {
			return fmt.Errorf("GitHub API Url invalid: %s", err)
		}
	}

	return nil
}

func (g *GithubTag) Fetch() (*FetchResponse, error) {

	fr := newFetchResponse()

	// Create a client
	client := g.newClient()
	tags, resp, err := client.Repositories.ListTags(context.Background(), g.Owner, g.Repository, nil)
	if err != nil {
		return fr, err
	}

	if resp.StatusCode != 200 {
		return fr, fmt.Errorf("Unknown status: %d", resp.StatusCode)
	}

	// fixF is FixVersionStrFunc transform tag name string into SemVer string
	// By default, it does nothing.
	fixF := g.fixVersionStrFunc()

	// filterF is TagFilterFunc to filter unexpected tags
	// By default, it filter nothing.
	filterF := g.tagFilterFunc()

	for _, tag := range tags {
		if !filterF(*tag.Name) {
			fr.Malformeds = append(fr.Malformeds, *tag.Name)
			continue
		}
		v, err := version.NewVersion(fixF(*tag.Name))
		if err != nil {
			fr.Malformeds = append(fr.Malformeds, fixF(*tag.Name))
			continue
		}
		fr.Versions = append(fr.Versions, v)
	}

	return fr, nil
}
