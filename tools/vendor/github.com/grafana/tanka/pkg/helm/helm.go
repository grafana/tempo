package helm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/grafana/tanka/pkg/kubernetes/manifest"
	"github.com/rs/zerolog/log"
)

// Helm provides high level access to some Helm operations
type Helm interface {
	// Pull downloads a Helm Chart from a remote
	Pull(chart, version string, opts PullOpts) error

	// RepoUpdate fetches the latest remote index
	RepoUpdate(opts Opts) error

	// Template returns the individual resources of a Helm Chart
	Template(name, chart string, opts TemplateOpts) (manifest.List, error)

	// ChartExists checks if a chart exists in the provided calledFromPath
	ChartExists(chart string, opts *JsonnetOpts) (string, error)

	// SearchRepo searches the repository for an updated chart version
	SearchRepo(chart, currVersion string, opts Opts) (ChartSearchVersions, error)
}

// PullOpts are additional, non-required options for Helm.Pull
type PullOpts struct {
	Opts

	// Directory to put the resulting .tgz into
	Destination string

	// Where to extract the chart to, defaults to the name of the chart
	ExtractDirectory string
}

// ChartSearchVersion represents a single chart version returned from the helm search repo command.
type ChartSearchVersion struct {
	// Name of the chart in the form of repo/chartName
	Name string `json:"name,omitempty"`

	// Version of the Helm chart
	Version string `json:"version,omitempty"`

	// Version of the application being deployed by the Helm chart
	AppVersion string `json:"app_version,omitempty"`

	// Description of the Helm chart
	Description string `json:"description,omitempty"`
}

type ChartSearchVersions []ChartSearchVersion

// RequiresVersionInfo represents a specific required chart and the information around the current
// version and any upgrade information.
type RequiresVersionInfo struct {
	// Name of the required chart in the form of repo/chartName
	Name string `json:"name,omitempty"`

	// Directory information for the chart.
	Directory string `json:"directory,omitempty"`

	// The current version information of the required helm chart.
	CurrentVersion string `json:"current_version,omitempty"`

	// Boolean representing if the required chart is already up to date.
	UsingLatestVersion bool `json:"using_latest_version"`

	// The most up-to-date version information of the required helm chart.
	LatestVersion ChartSearchVersion `json:"latest_version,omitempty"`

	// The latest version information of the required helm chart that matches the current major version.
	LatestMatchingMajorVersion ChartSearchVersion `json:"latest_matching_major_version,omitempty"`

	// The latest version information of the required helm chart that matches the current minor version.
	LatestMatchingMinorVersion ChartSearchVersion `json:"latest_matching_minor_version,omitempty"`
}

// Opts are additional, non-required options that all Helm operations accept
type Opts struct {
	Repositories []Repo
}

// ExecHelm is a Helm implementation powered by the `helm` command line utility
type ExecHelm struct{}

// Pull implements Helm.Pull
func (e ExecHelm) Pull(chart, version string, opts PullOpts) error {
	repoFile, err := writeRepoTmpFile(opts.Repositories)
	if err != nil {
		return err
	}
	defer os.Remove(repoFile)

	// Pull to a temp dir within the destination directory (not /tmp) to avoid possible cross-device issues when renaming
	tempDir, err := os.MkdirTemp(opts.Destination, ".pull-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	chartPullPath := chart
	chartRepoName, chartName := parseReqRepo(chart), parseReqName(chart)
	for _, configuredRepo := range opts.Repositories {
		if configuredRepo.Name == chartRepoName {
			// OCI images are pulled with their full path
			if strings.HasPrefix(configuredRepo.URL, "oci://") {
				chartPullPath = fmt.Sprintf("%s/%s", configuredRepo.URL, chartName)
			}
		}
	}

	cmd := e.cmd("pull", chartPullPath,
		"--version", version,
		"--repository-config", repoFile,
		"--destination", tempDir,
		"--untar",
	)

	if err = cmd.Run(); err != nil {
		return err
	}

	if opts.ExtractDirectory == "" {
		opts.ExtractDirectory = chartName
	}

	// It is not possible to tell `helm pull` to extract to a specific directory
	// so we extract to a temp dir and then move the files to the destination
	return os.Rename(
		filepath.Join(tempDir, chartName),
		filepath.Join(opts.Destination, opts.ExtractDirectory),
	)
}

// RepoUpdate implements Helm.RepoUpdate
func (e ExecHelm) RepoUpdate(opts Opts) error {
	repoFile, err := writeRepoTmpFile(opts.Repositories)
	if err != nil {
		return err
	}
	defer os.Remove(repoFile)

	cmd := e.cmd("repo", "update",
		"--repository-config", repoFile,
	)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s\n%s", errBuf.String(), err)
	}

	return nil
}

func (e ExecHelm) ChartExists(chart string, opts *JsonnetOpts) (string, error) {
	// resolve the Chart relative to the caller
	callerDir := filepath.Dir(opts.CalledFrom)
	chart = filepath.Join(callerDir, chart)
	if _, err := os.Stat(chart); err != nil {
		return "", fmt.Errorf("helmTemplate: Failed to find a chart at '%s': %s. See https://tanka.dev/helm#failed-to-find-chart", chart, err)
	}

	return chart, nil
}

// Searches the helm repositories for the latest, the latest matching major, and the latest
// matching minor versions for the given chart.
func (e ExecHelm) SearchRepo(chart, currVersion string, opts Opts) (ChartSearchVersions, error) {
	searchVersions := []string{
		fmt.Sprintf(">=%s", currVersion), // Latest version X.X.X
		fmt.Sprintf("^%s", currVersion),  // Latest matching major version 1.X.X
		fmt.Sprintf("~%s", currVersion),  // Latest matching minor version 1.1.X
	}

	repoFile, err := writeRepoTmpFile(opts.Repositories)
	if err != nil {
		return nil, err
	}
	defer os.Remove(repoFile)

	// Helm repo update in order to let "helm search repo" function
	updateCmd := e.cmd("repo", "update",
		"--repository-config", repoFile,
	)
	var updateErrBuf bytes.Buffer
	var updateOutBuf bytes.Buffer
	updateCmd.Stderr = &updateErrBuf
	updateCmd.Stdout = &updateOutBuf
	if err := updateCmd.Run(); err != nil {
		return nil, fmt.Errorf("%s\n%s", updateErrBuf.String(), err)
	}

	var chartVersions ChartSearchVersions
	for _, versionRegex := range searchVersions {
		var chartVersion ChartSearchVersions

		// Vertical tabs are used as deliminators in table so \v is used to match exactly the chart.
		// Helm search by default only returns the latest version matching the given version regex.
		cmd := e.cmd("search", "repo",
			"--repository-config", repoFile,
			"--regexp", fmt.Sprintf("\v%s\v", chart),
			"--version", versionRegex,
			"-o", "json",
		)
		var errBuf bytes.Buffer
		var outBuf bytes.Buffer
		cmd.Stderr = &errBuf
		cmd.Stdout = &outBuf

		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("%s\n%s", errBuf.String(), err)
		}

		err = json.Unmarshal(outBuf.Bytes(), &chartVersion)
		if err != nil {
			return nil, err
		}

		if len(chartVersion) != 1 {
			log.Debug().Msgf("helm search repo for %s did not return 1 version : %+v", chart, chartVersion)
			chartVersions = append(chartVersions, ChartSearchVersion{
				Name:        chart,
				Version:     currVersion,
				AppVersion:  "",
				Description: "search did not return 1 version",
			})
		} else {
			chartVersions = append(chartVersions, chartVersion...)
		}
	}

	return chartVersions, nil
}

// cmd returns a prepared exec.Cmd to use the `helm` binary
func (e ExecHelm) cmd(action string, args ...string) *exec.Cmd {
	argv := []string{action}
	argv = append(argv, args...)
	log.Debug().Strs("argv", argv).Msg("running helm")

	cmd := helmCmd(argv...)
	cmd.Stderr = os.Stderr

	return cmd
}

// helmCmd returns a bare exec.Cmd pointed at the local helm binary
func helmCmd(args ...string) *exec.Cmd {
	bin := "helm"
	if env := os.Getenv("TANKA_HELM_PATH"); env != "" {
		bin = env
	}

	return exec.Command(bin, args...)
}

// writeRepoTmpFile creates a temporary repositories.yaml from the passed Repo
// slice to be used by the helm binary
func writeRepoTmpFile(r []Repo) (string, error) {
	m := map[string]interface{}{
		"repositories": r,
	}

	f, err := os.CreateTemp("", "charts-repos")
	if err != nil {
		return "", err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	if err := enc.Encode(m); err != nil {
		return "", err
	}

	return f.Name(), nil
}
