package helm

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"sigs.k8s.io/yaml"
)

var (
	// https://regex101.com/r/9m42pQ/1
	chartExp = regexp.MustCompile(`^(?P<chart>[\w+-\/.]+)@(?P<version>[^:\n\s]+)(?:\:(?P<path>[\w-. ]+))?$`)
	// https://regex101.com/r/xoAx8c/1
	repoExp = regexp.MustCompile(`^[\w-]+$`)
)

// LoadChartfile opens a Chartfile tree
func LoadChartfile(projectRoot string) (*Charts, error) {
	// make sure project root is valid
	abs, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, err
	}

	// open chartfile
	chartfile := filepath.Join(abs, Filename)
	data, err := os.ReadFile(chartfile)
	if err != nil {
		return nil, err
	}

	// parse it
	c := Chartfile{
		Version:   Version,
		Directory: DefaultDir,
	}
	if err := yaml.UnmarshalStrict(data, &c); err != nil {
		return nil, err
	}

	for i, r := range c.Requires {
		if r.Chart == "" {
			return nil, fmt.Errorf("requirements[%v]: 'chart' must be set", i)
		}
	}

	// return Charts handle
	charts := &Charts{
		Manifest:    c,
		projectRoot: abs,

		// default to ExecHelm, but allow injecting from the outside
		Helm: ExecHelm{},
	}
	return charts, nil
}

// LoadHelmRepoConfig reads in a helm config file
func LoadHelmRepoConfig(repoConfigPath string) (*ConfigFile, error) {
	// make sure path is valid
	repoPath, err := filepath.Abs(repoConfigPath)
	if err != nil {
		return nil, err
	}

	// open repo config file
	data, err := os.ReadFile(repoPath)
	if err != nil {
		return nil, err
	}

	// parse the file non-strictly to account for any minor config changes
	rc := &ConfigFile{}
	if err := yaml.Unmarshal(data, rc); err != nil {
		return nil, err
	}

	return rc, nil
}

// Charts exposes the central Chartfile management functions
type Charts struct {
	// Manifest are the chartfile.yaml contents. It holds data about the developers intentions
	Manifest Chartfile

	// projectRoot is the enclosing directory of chartfile.yaml
	projectRoot string

	// Helm is the helm implementation underneath. ExecHelm is the default, but
	// any implementation of the Helm interface may be used
	Helm Helm
}

// chartManifest represents a Helm chart's Chart.yaml
type chartManifest struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

// ChartDir returns the directory pulled charts are saved in
func (c Charts) ChartDir() string {
	return filepath.Join(c.projectRoot, c.Manifest.Directory)
}

// ManifestFile returns the full path to the chartfile.yaml
func (c Charts) ManifestFile() string {
	return filepath.Join(c.projectRoot, Filename)
}

// Vendor pulls all Charts specified in the manifest into the local charts
// directory. It fetches the repository index before doing so.
func (c Charts) Vendor(prune bool, repoConfigPath string) error {
	dir := c.ChartDir()
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	repositories, err := c.getRepositories(repoConfigPath)
	if err != nil {
		return err
	}

	// Check that there are no output conflicts before vendoring
	if err := c.Manifest.Requires.Validate(); err != nil {
		return err
	}

	expectedDirs := make(map[string]bool)

	repositoriesUpdated := false
	log.Info().Msg("Vendoring...")
	for _, r := range c.Manifest.Requires {
		chartSubDir := parseReqName(r.Chart)
		if r.Directory != "" {
			chartSubDir = r.Directory
		}
		chartPath := filepath.Join(dir, chartSubDir)
		chartManifestPath := filepath.Join(chartPath, "Chart.yaml")
		expectedDirs[chartSubDir] = true

		chartDirExists, chartManifestExists := false, false
		if _, err := os.Stat(chartPath); err == nil {
			chartDirExists = true
			if _, err := os.Stat(chartManifestPath); err == nil {
				chartManifestExists = true
			} else if !os.IsNotExist(err) {
				return err
			}
		} else if !os.IsNotExist(err) {
			return err
		}

		if chartManifestExists {
			chartManifestBytes, err := os.ReadFile(chartManifestPath)
			if err != nil {
				return fmt.Errorf("reading chart manifest: %w", err)
			}
			var chartYAML chartManifest
			if err := yaml.Unmarshal(chartManifestBytes, &chartYAML); err != nil {
				return fmt.Errorf("unmarshalling chart manifest: %w", err)
			}

			if chartYAML.Version == r.Version {
				log.Info().Msgf("%s exists", r)
				continue
			}

			log.Info().Msgf("Removing %s", r)
			if err := os.RemoveAll(chartPath); err != nil {
				return err
			}
		} else if chartDirExists {
			// If the chart dir exists but the manifest doesn't, we'll clear it out and re-download the chart
			log.Info().Msgf("Removing %s", r)
			if err := os.RemoveAll(chartPath); err != nil {
				return err
			}
		}

		if !repositoriesUpdated {
			log.Info().Msg("Syncing Repositories ...")
			if err := c.Helm.RepoUpdate(Opts{Repositories: repositories}); err != nil {
				return err
			}
			repositoriesUpdated = true
		}
		log.Info().Msg("Pulling Charts ...")
		if repoName := parseReqRepo(r.Chart); !repositories.HasName(repoName) {
			return fmt.Errorf("repository %q not found for chart %q", repoName, r.Chart)
		}
		err := c.Helm.Pull(r.Chart, r.Version, PullOpts{
			Destination:      dir,
			ExtractDirectory: r.Directory,
			Opts:             Opts{Repositories: repositories},
		})
		if err != nil {
			return err
		}

		log.Info().Msgf("%s@%s downloaded", r.Chart, r.Version)
	}

	if prune {
		items, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("error listing the content of the charts dir: %w", err)
		}
		for _, i := range items {
			if !expectedDirs[i.Name()] {
				itemType := "file"
				if i.IsDir() {
					itemType = "directory"
				}
				log.Info().Msgf("Pruning %s: %s", itemType, i.Name())
				if err := os.RemoveAll(filepath.Join(dir, i.Name())); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// Add adds every Chart in reqs to the Manifest after validation, and runs
// Vendor afterwards
func (c *Charts) Add(reqs []string, repoConfigPath string) error {
	log.Info().Msgf("Adding %v Charts ...", len(reqs))

	// parse new charts, append in memory
	requirements := c.Manifest.Requires
	for _, s := range reqs {
		r, err := parseReq(s)
		if err != nil {
			skip(s, err)
			continue
		}

		if requirements.Has(*r) {
			skip(s, fmt.Errorf("already exists"))
			continue
		}

		requirements = append(requirements, *r)
		log.Info().Msgf("OK: %s", s)
	}

	if err := requirements.Validate(); err != nil {
		return err
	}

	// write out
	added := len(requirements) - len(c.Manifest.Requires)
	c.Manifest.Requires = requirements
	if err := write(c.Manifest, c.ManifestFile()); err != nil {
		return err
	}

	// skipped some? fail then
	if added != len(reqs) {
		return fmt.Errorf("%v Chart(s) were skipped. Please check above logs for details", len(reqs)-added)
	}

	// worked fine? vendor it
	log.Info().Msgf("Added %v Charts to helmfile.yaml. Vendoring ...", added)
	return c.Vendor(false, repoConfigPath)
}

func (c *Charts) AddRepos(repos ...Repo) error {
	added := 0
	for _, r := range repos {
		if c.Manifest.Repositories.Has(r) {
			skip(r.Name, fmt.Errorf("already exists"))
			continue
		}

		if !repoExp.MatchString(r.Name) {
			skip(r.Name, fmt.Errorf("invalid name. cannot contain any special characters"))
			continue
		}

		c.Manifest.Repositories = append(c.Manifest.Repositories, r)
		added++
		log.Info().Msgf("OK: %s", r.Name)
	}

	// write out
	if err := write(c.Manifest, c.ManifestFile()); err != nil {
		return err
	}

	if added != len(repos) {
		return fmt.Errorf("%v Repo(s) were skipped. Please check above logs for details", len(repos)-added)
	}

	return nil
}

// VersionCheck checks each of the charts in the requires section and returns information regarding
// related to version upgrades. This includes if the current version is latest as well as the
// latest matching versions of the major and minor version the chart is currently on.
func (c *Charts) VersionCheck(repoConfigPath string) (map[string]RequiresVersionInfo, error) {
	requiresVersionInfo := make(map[string]RequiresVersionInfo)
	repositories, err := c.getRepositories(repoConfigPath)
	if err != nil {
		return nil, err
	}

	for _, r := range c.Manifest.Requires {
		searchVersions, err := c.Helm.SearchRepo(r.Chart, r.Version, Opts{Repositories: repositories})
		if err != nil {
			return nil, err
		}
		usingLatestVersion := r.Version == searchVersions[0].Version

		requiresVersionInfo[fmt.Sprintf("%s@%s", r.Chart, r.Version)] = RequiresVersionInfo{
			Name:                       r.Chart,
			Directory:                  r.Directory,
			CurrentVersion:             r.Version,
			UsingLatestVersion:         usingLatestVersion,
			LatestVersion:              searchVersions[0],
			LatestMatchingMajorVersion: searchVersions[1],
			LatestMatchingMinorVersion: searchVersions[2],
		}
	}

	return requiresVersionInfo, nil
}

// getRepositories will dynamically return the repositores either loaded from the given
// repoConfigPath file or from the existing manifest.
func (c *Charts) getRepositories(repoConfigPath string) (Repos, error) {
	if repoConfigPath != "" {
		repoConfig, err := LoadHelmRepoConfig(repoConfigPath)
		if err != nil {
			return nil, err
		}
		return repoConfig.Repositories, nil
	}
	return c.Manifest.Repositories, nil
}

func InitChartfile(path string) (*Charts, error) {
	c := Chartfile{
		Version: Version,
		Repositories: []Repo{{
			Name: "stable",
			URL:  "https://charts.helm.sh/stable",
		}},
		Requires: make(Requirements, 0),
	}

	if err := write(c, path); err != nil {
		return nil, err
	}

	return LoadChartfile(filepath.Dir(path))
}

// write saves a Chartfile to dest
func write(c Chartfile, dest string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(dest, data, 0644)
}

// parseReq parses a requirement from a string of the format `repo/name@version`
func parseReq(s string) (*Requirement, error) {
	matches := chartExp.FindStringSubmatch(s)
	if matches == nil {
		return nil, fmt.Errorf("not of form 'repo/chart@version(:path)' where repo contains no special characters")
	}

	chart, ver := matches[1], matches[2]

	directory := ""
	if len(matches) == 4 {
		directory = matches[3]
	}

	return &Requirement{
		Chart:     chart,
		Version:   ver,
		Directory: directory,
	}, nil
}

// parseReqRepo parses a repo from a string of the format `repo/name`
func parseReqRepo(s string) string {
	elems := strings.SplitN(s, "/", 2)
	repo := elems[0]
	return repo
}

// parseReqName parses a name from a string of the format `repo/name`
func parseReqName(s string) string {
	elems := strings.SplitN(s, "/", 2)
	if len(elems) == 1 {
		return ""
	}
	name := elems[1]
	return name
}

func skip(s string, err error) {
	log.Info().Msgf("Skipping %s: %s.", s, err)
}
