package helm

import (
	"fmt"
	"strings"
)

const (
	// Version of the current Chartfile implementation
	Version = 1

	// Filename of the Chartfile
	Filename = "chartfile.yaml"

	// DefaultDir is the directory used for storing Charts if not specified
	// otherwise
	DefaultDir = "charts"
)

// Chartfile is the schema used to declaratively define locally required Helm
// Charts
type Chartfile struct {
	// Version of the Chartfile schema (for future use)
	Version uint `json:"version"`

	// Repositories to source from
	Repositories Repos `json:"repositories"`

	// Requires lists Charts expected to be present in the charts folder
	Requires Requirements `json:"requires"`

	// Folder to use for storing Charts. Defaults to 'charts'
	Directory string `json:"directory,omitempty"`
}

// ConfigFile represents the default Helm config structure to be used in place of the chartfile
// Repositories if supplied.
type ConfigFile struct {
	// Version of the Helm repo config schema
	APIVersion string `json:"apiVersion"`

	// The datetime of when this repo config was generated
	Generated string `json:"generated"`

	// Repositories to source from
	Repositories Repos `json:"repositories"`
}

// Repo describes a single Helm repository
type Repo struct {
	Name     string `json:"name,omitempty"`
	URL      string `json:"url,omitempty"`
	CAFile   string `json:"caFile,omitempty"`
	CertFile string `json:"certFile,omitempty"`
	KeyFile  string `json:"keyFile,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type Repos []Repo

// Has reports whether 'repo' is already part of the repositories
func (r Repos) Has(repo Repo) bool {
	for _, x := range r {
		if x == repo {
			return true
		}
	}

	return false
}

// Has reports whether one of the repos has the given name
func (r Repos) HasName(repoName string) bool {
	for _, x := range r {
		if x.Name == repoName {
			return true
		}
	}

	return false
}

// Requirement describes a single required Helm Chart.
// Both, Chart and Version are required
type Requirement struct {
	Chart     string `json:"chart"`
	Version   string `json:"version"`
	Directory string `json:"directory,omitempty"`
}

func (r Requirement) String() string {
	dir := r.Directory
	if dir == "" {
		dir = parseReqName(r.Chart)
	}
	return fmt.Sprintf("%s@%s (dir: %s)", r.Chart, r.Version, dir)
}

// Requirements is an aggregate of all required Charts
type Requirements []Requirement

// Has reports whether 'req' is already part of the requirements
func (r Requirements) Has(req Requirement) bool {
	for _, x := range r {
		if x == req {
			return true
		}
	}

	return false
}

func (r Requirements) Validate() error {
	outputDirs := make(map[string]Requirement)
	errs := make([]string, 0)

	for _, req := range r {
		if !strings.Contains(req.Chart, "/") {
			errs = append(errs, fmt.Sprintf("Chart name %q is not valid. Expecting a repo/name format.", req.Chart))
			continue
		}

		dir := req.Directory
		if dir == "" {
			dir = parseReqName(req.Chart)
		}
		if previous, ok := outputDirs[dir]; ok {
			errs = append(errs, fmt.Sprintf(`output directory %q is used twice, by charts "%s@%s" and "%s@%s"`, dir, previous.Chart, previous.Version, req.Chart, req.Version))
		}
		outputDirs[dir] = req
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation errors:\n - %s", strings.Join(errs, "\n - "))
	}

	return nil
}
