package e2e

import (
	cortex_e2e "github.com/cortexproject/cortex/integration/e2e"
	cortex_e2e_db "github.com/cortexproject/cortex/integration/e2e/db"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

var (
	svc_name = "tempo"
	image = "tempo:latest"
)

func NewTempoAllInOne() (*cortex_e2e.HTTPService, error) {
	args := "-config.file="+filepath.Join(cortex_e2e.ContainerSharedDir, "config.yaml")

	return cortex_e2e.NewHTTPService(
		svc_name,
		image,
		cortex_e2e.NewCommandWithoutEntrypoint("/tempo", args),
		cortex_e2e.NewHTTPReadinessProbe(3100, "/ready", 200, 505),
		3100,
	), nil
}

func TestIngest(t *testing.T) {
	s, err := cortex_e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	minio := cortex_e2e_db.NewMinio(9000, "tempo")
	require.NotNil(t, minio)
	require.NoError(t, s.StartAndWaitReady(minio))

	require.NoError(t, copyFileToSharedDir(s, "./config.yaml", "config.yaml"))
	tempo, err := NewTempoAllInOne()
	require.NoError(t, err)
	require.NoError(t, s.StartAndWaitReady(tempo))
}
