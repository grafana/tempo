package kustomize

import (
	"os"
	"os/exec"

	"github.com/grafana/tanka/pkg/kubernetes/manifest"
)

// Kustomize provides high level access to some Kustomize operations
type Kustomize interface {
	// Build returns the individual resources of a Kustomize
	Build(path string) (manifest.List, error)
}

// ExecKustomize is a Kustomize implementation powered by the `kustomize`
// command line utility
type ExecKustomize struct{}

// cmd returns a prepared exec.Cmd to use the `kustomize` binary
func (k ExecKustomize) cmd(action string, args ...string) *exec.Cmd {
	argv := []string{action}
	argv = append(argv, args...)

	cmd := kustomizeCmd(argv...)
	cmd.Stderr = os.Stderr

	return cmd
}

// kustomizeCmd returns a bare exec.Cmd pointed at the local kustomize binary
func kustomizeCmd(args ...string) *exec.Cmd {
	bin := "kustomize"
	if env := os.Getenv("TANKA_KUSTOMIZE_PATH"); env != "" {
		bin = env
	}

	return exec.Command(bin, args...)
}
