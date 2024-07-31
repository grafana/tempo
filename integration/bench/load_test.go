package load

import (
	"fmt"
	"path/filepath"
	"testing"

	util "github.com/grafana/tempo/v2/integration"

	"github.com/grafana/e2e"
	e2e_db "github.com/grafana/e2e/db"
	"github.com/stretchr/testify/require"
)

const (
	k6Image = "loadimpact/k6:latest"
)

func TestAllInOne(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	minio := e2e_db.NewMinio(9000, "tempo")
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

func runK6Test(k6 *e2e.ConcreteService, testjs string) error {
	fmt.Println("------ " + testjs + " ------")
	stdout, stderr, err := k6.Exec(e2e.NewCommand("k6", "run", "--quiet", "--log-output", "none", filepath.Join(e2e.ContainerSharedDir, testjs)))
	fmt.Println("------ stdout ------")
	fmt.Println(stdout)

	if err != nil {
		fmt.Println("------ stderr ------")
		fmt.Println(stderr)
	}

	return err
}

func newK6Runner(tempo *e2e.HTTPService) *e2e.ConcreteService {
	s := e2e.NewConcreteService(
		"k6",
		k6Image,
		e2e.NewCommandWithoutEntrypoint("sh", "-c", "sleep 3600"),
		e2e.NewCmdReadinessProbe(e2e.NewCommand("sh", "-c", "")),
	)

	s.SetUser("0") // required so k6 can read the js files passed in

	tempoHTTP := "http://" + tempo.NetworkEndpoint(3200)
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
