package e2e

import (
	"os/exec"
	"path/filepath"

	cortex_e2e "github.com/cortexproject/cortex/integration/e2e"
)
var (
	svc_name = "tempo"
	image = "tempo:latest"
)

func NewTempoAllInOne() (*cortex_e2e.HTTPService, error) {

	args := "-config.file="+filepath.Join(cortex_e2e.ContainerSharedDir, "config.yaml")

	exec.Command("docker run tempo:latest ")

	return cortex_e2e.NewHTTPService(
		svc_name,
		image,
		cortex_e2e.NewCommand("/tempo", args),
		cortex_e2e.NewHTTPReadinessProbe(3100, "/ready", 200, 505),
		3100,
		14268,
		), nil
}



