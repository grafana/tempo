package e2e

import (
	"github.com/stretchr/testify/require"
	"testing"

	cortex_e2e "github.com/cortexproject/cortex/integration/e2e"
	cortex_e2e_db "github.com/cortexproject/cortex/integration/e2e/db"
)

func TestIngest(t *testing.T) {
	s, err := cortex_e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	minio := cortex_e2e_db.NewMinio(9000, "tempo")
	require.NotNil(t, minio)
	require.NoError(t, s.StartAndWaitReady(minio))

	require.NoError(t, copyFileToSharedDir(s, "./config.yaml","config.yaml"))
	tempo, err := NewTempoAllInOne()
	require.NoError(t, err)
	require.NoError(t, s.StartAndWaitReady(tempo))
}
