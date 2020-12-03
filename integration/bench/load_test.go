package load

import (
	"fmt"
	"path/filepath"
	"testing"

	util "github.com/grafana/tempo/integration"

	cortex_e2e "github.com/cortexproject/cortex/integration/e2e"
	cortex_e2e_db "github.com/cortexproject/cortex/integration/e2e/db"
	"github.com/stretchr/testify/require"
)

const (
	k6Image = "loadimpact/k6:latest"
)

func TestAllInOne(t *testing.T) {
	s, err := cortex_e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	minio := cortex_e2e_db.NewMinio(9000, "tempo")
	require.NotNil(t, minio)
	require.NoError(t, s.StartAndWaitReady(minio))

	require.NoError(t, util.CopyFileToSharedDir(s, "config.yaml", "config.yaml"))
	require.NoError(t, util.CopyFileToSharedDir(s, "smoke_test.js", "smoke_test.js"))
	require.NoError(t, util.CopyFileToSharedDir(s, "stress_test_write_path.js", "stress_test_write_path.js"))
	require.NoError(t, util.CopyFileToSharedDir(s, "modules/util.js", "modules/util.js"))

	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	k6 := newK6Runner(tempo)
	require.NoError(t, s.StartAndWaitReady(k6))

	require.NoError(t, runK6Test(k6, "smoke_test.js"))
	require.NoError(t, runK6Test(k6, "stress_test_write_path.js"))
}

func runK6Test(k6 *cortex_e2e.ConcreteService, testjs string) error {
	fmt.Println("------ " + testjs + " ------")
	stdout, _, err := k6.Exec(cortex_e2e.NewCommand("k6", "run", "--quiet", filepath.Join(cortex_e2e.ContainerSharedDir, testjs)))
	fmt.Println("------ stdout ------")
	fmt.Println(stdout)
	fmt.Println("------ stderr ------")
	fmt.Println(stdout)

	return err
}

func newK6Runner(tempo *cortex_e2e.HTTPService) *cortex_e2e.ConcreteService {
	s := cortex_e2e.NewConcreteService(
		"k6",
		k6Image,
		cortex_e2e.NewCommandWithoutEntrypoint("sh", "-c", "sleep 3600"),
		cortex_e2e.NewCmdReadinessProbe(cortex_e2e.NewCommand("sh", "-c", "")),
	)

	s.SetUser("0") // required so k6 can read the js files passed in

	tempoHTTP := "http://" + tempo.NetworkEndpoint(3100)
	tempoZipkin := "http://" + tempo.NetworkEndpoint(9411)
	s.SetEnvVars(map[string]string{
		"WRITE_ENDPOINT":       tempoZipkin,
		"DISTRIBUTOR_ENDPOINT": tempoHTTP,
		"INGESTER_ENDPOINT":    tempoHTTP,
		"QUERY_ENDPOINT":       tempoHTTP,
		"QUERIER_ENDPOINT":     tempoHTTP,
	})

	return s
}
