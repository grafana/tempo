/*
go-latest is pacakge to check a provided version is latest from various sources.

http://github.com/tcnksm/go-latest

  package main

  import (
      "github.com/tcnksm/go-latest"
  )

  githubTag := &latest.GithubTag{
      Owner: "tcnksm",
      Repository: "ghr"
  }

  res, _ := latest.Check("0.1.0",githubTag)
  if res.Outdated {
      fmt.Printf("version 0.1.0 is out of date, you can upgrade to %s", res.Current)
  }

*/
package latest

import (
	"fmt"
	"os"
	"sort"

	"github.com/hashicorp/go-version"
)

// EnvGoLatestDisable is environmental variable to disable go-latest
// execution.
const EnvGoLatestDisable = "GOLATEST_DISABLE"

// Source is the interface that every version information source must implement.
type Source interface {
	// Validate is called before Fetch in Check.
	// Source may need to have various information like URL or product name,
	// so it is used for check each variables are correctly set.
	// If it is failed, Fetch() will not executed.
	Validate() error

	// Fetch is called in Check to fetch information from remote sources.
	// After fetching, it will convert it into common expression (FetchResponse)
	Fetch() (*FetchResponse, error)
}

// FetchResponse the commom response of Fetch request.
type FetchResponse struct {
	Versions   []*version.Version
	Malformeds []string
	Meta       *Meta
}

// Meta is meta information from Fetch request.
type Meta struct {
	Message string
	URL     string
}

// CheckResponse is a response for a Check request.
type CheckResponse struct {
	// Current is current latest version on source.
	Current string

	// Outdate is true when target version is less than Curernt on source.
	Outdated bool

	// Latest is true when target version is equal to Current on source.
	Latest bool

	// New is true when target version is greater than Current on source.
	New bool

	// Malformed store versions or tags which can not be parsed as
	// Semantic versioning (not compared with target).
	Malformeds []string

	// Meta is meta information from source.
	Meta *Meta
}

// Check fetches last version information from its source
// and compares with target and return result (CheckResponse).
func Check(s Source, target string) (*CheckResponse, error) {

	if os.Getenv(EnvGoLatestDisable) != "" {
		return &CheckResponse{}, nil
	}

	// Convert target to *version.Version
	targetV, err := version.NewVersion(target)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s, %s", target, err.Error())
	}

	// Validate source
	if err = s.Validate(); err != nil {
		return nil, err
	}

	fr, err := s.Fetch()
	if err != nil {
		return nil, err
	}

	// Source must has at leaset one version information
	versions := fr.Versions
	if len(fr.Versions) == 0 {
		return nil, fmt.Errorf("no version to compare")
	}
	sort.Sort(version.Collection(versions))
	currentV := versions[len(versions)-1]

	var outdated, latest, new bool
	if targetV.LessThan(currentV) {
		outdated = true
	}

	// If target = current, target is `latest`
	if targetV.Equal(currentV) {
		latest = true
	}

	// If target > current, target is `latest` and `new`
	if targetV.GreaterThan(currentV) {
		latest, new = true, true
	}

	return &CheckResponse{
		Current:    currentV.String(),
		Outdated:   outdated,
		Latest:     latest,
		New:        new,
		Malformeds: fr.Malformeds,
		Meta:       fr.Meta,
	}, nil
}

// newFetchResponse is constructor of FetchResponse. This is only for
// implement your own Source
func newFetchResponse() *FetchResponse {
	var versions []*version.Version
	var malformeds []string
	return &FetchResponse{
		Versions:   versions,
		Malformeds: malformeds,
		Meta:       &Meta{},
	}
}
