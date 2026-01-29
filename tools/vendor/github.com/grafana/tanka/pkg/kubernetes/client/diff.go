package client

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/Masterminds/semver"

	"github.com/grafana/tanka/pkg/kubernetes/manifest"
)

// Provides just the parts of `*exec.ExitError` that we need, so we can swap for a fake in the tests
type exitError interface {
	error
	ExitCode() int
}

// DiffServerSide takes the desired state and computes the differences server-side, returning them in `diff(1)` format
func (k Kubectl) DiffServerSide(data manifest.List) (*string, error) {
	return k.diff(data, true)
}

// ValidateServerSide takes the desired state and computes the differences, returning them in `diff(1)` format
// It also validates that manifests are valid server-side, but still returns the client-side diff
func (k Kubectl) ValidateServerSide(data manifest.List) (*string, error) {
	if _, diffErr := k.diff(data, true); diffErr != nil {
		return nil, diffErr
	}
	return k.diff(data, false)
}

// DiffClientSide takes the desired state and computes the differences, returning them in `diff(1)` format
func (k Kubectl) DiffClientSide(data manifest.List) (*string, error) {
	return k.diff(data, false)
}

func (k Kubectl) diff(data manifest.List, serverSide bool) (*string, error) {
	fw := FilterWriter{filters: []*regexp.Regexp{regexp.MustCompile(`exit status \d`)}}

	args := []string{"-f", "-"}
	if serverSide {
		args = append(args, "--server-side", "--force-conflicts")
		if k.info.ClientVersion.GreaterThan(semver.MustParse("1.19.0")) {
			args = append(args, "--field-manager=tanka")
		}
	}
	cmd := k.ctl("diff", args...)

	raw := bytes.Buffer{}
	// If using an external diff tool, let it keep the parent's stdout
	if os.Getenv("KUBECTL_INTERACTIVE_DIFF") != "" {
		cmd.Stdout = os.Stdout
	} else {
		cmd.Stdout = &raw
	}
	cmd.Stderr = &fw
	cmd.Stdin = strings.NewReader(data.String())
	err := cmd.Run()
	if diffErr := parseDiffErr(err, fw.buf, k.Info().ClientVersion); diffErr != nil {
		return nil, diffErr
	}

	s := raw.String()
	if s == "" {
		return nil, nil
	}

	return &s, nil
}

// parseDiffErr handles the exit status code of `kubectl diff`. It returns err
// when an error happened, nil otherwise.
// "Differences found (exit status 1)" is not an error.
//
// kubectl >= 1.18:
// 0: no error, no differences
// 1: differences found
// >1: error
//
// kubectl < 1.18:
// 0: no error, no differences
// 1: error OR differences found
func parseDiffErr(err error, stderr string, version *semver.Version) error {
	exitErr, ok := err.(exitError)
	if !ok {
		// this error is not kubectl related
		return err
	}

	// internal kubectl error
	if exitErr.ExitCode() != 1 {
		return err
	}

	// before 1.18 "exit status 1" meant error as well ... so we need to check stderr
	if version.LessThan(semver.MustParse("1.18.0")) && stderr != "" {
		return fmt.Errorf("diff failed: %w (%s)", err, stderr)
	}

	// differences found is not an error
	return nil
}
