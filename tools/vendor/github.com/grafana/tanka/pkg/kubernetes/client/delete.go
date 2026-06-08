package client

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
)

func buildFullType(group, version, kind string) string {
	output := strings.Builder{}
	output.WriteString(kind)
	// Unfortunately, kubectl does not support `Type.Version` for things like
	// `Service` in v1. In this case, we cannot provide anything but the kind
	// name:
	if version != "" && group != "" {
		output.WriteString(".")
		output.WriteString(version)
		output.WriteString(".")
		output.WriteString(group)
	}
	return output.String()
}

// Test-ability: isolate deleteCtl to build and return exec.Cmd from DeleteOpts
func (k Kubectl) deleteCtl(namespace, group, version, kind, name string, opts DeleteOpts) *exec.Cmd {
	fullType := buildFullType(group, version, kind)
	argv := []string{
		"-n", namespace,
		fullType, name,
	}
	log.Debug().Str("name", name).Str("group", group).Str("version", version).Str("kind", kind).Str("namespace", namespace).Msg("Preparing to delete")
	if opts.Force {
		argv = append(argv, "--force")
	}

	if opts.DryRun != "" {
		dryRun := fmt.Sprintf("--dry-run=%s", opts.DryRun)
		argv = append(argv, dryRun)
	}

	return k.ctl("delete", argv...)
}

// Delete deletes the given Kubernetes resource from the cluster
func (k Kubectl) Delete(namespace, apiVersion, kind, name string, opts DeleteOpts) error {
	apiVersionElements := strings.SplitN(apiVersion, "/", 2)
	if len(apiVersionElements) < 1 {
		return fmt.Errorf("apiVersion does not follow the group/version or version format: %s", apiVersion)
	}
	var group string
	var version string
	if len(apiVersionElements) == 1 {
		group = ""
		version = apiVersionElements[0]
	} else {
		group = apiVersionElements[0]
		version = apiVersionElements[1]
	}

	cmd := k.deleteCtl(namespace, group, version, kind, name, opts)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if strings.Contains(stderr.String(), "Error from server (NotFound):") {
			print("Delete failed: " + stderr.String())
			return nil
		}
		log.Trace().Msgf("Delete failed: %s", stderr.String())
		return err
	}
	if opts.DryRun != "" {
		print(stdout.String())
	}

	return nil
}
