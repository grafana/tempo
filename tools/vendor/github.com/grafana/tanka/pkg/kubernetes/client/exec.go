package client

import (
	"fmt"
	"os"
	"os/exec"
)

// kubectlCmd returns command a object that will launch kubectl at an appropriate path.
func kubectlCmd(args ...string) *exec.Cmd {
	binary := "kubectl"
	if env := os.Getenv("TANKA_KUBECTL_PATH"); env != "" {
		binary = env
	}

	return exec.Command(binary, args...)
}

// ctl returns an `exec.Cmd` for `kubectl`. It also forces the correct context
// and injects our patched $KUBECONFIG for the default namespace.
func (k Kubectl) ctl(action string, args ...string) *exec.Cmd {
	// prepare the arguments
	argv := []string{action,
		"--context", k.info.Kubeconfig.Context.Name,
	}
	argv = append(argv, args...)

	// prepare the cmd
	cmd := kubectlCmd(argv...)

	if os.Getenv("TANKA_KUBECTL_TRACE") != "" {
		fmt.Println(cmd.String())
	}

	return cmd
}
