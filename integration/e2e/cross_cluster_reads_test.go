package e2e

import (
	"testing"
	"time"

	"github.com/grafana/e2e"
	e2edb "github.com/grafana/e2e/db"
	util "github.com/grafana/tempo/integration"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
)

func TestCrossClusterReads(t *testing.T) {
	// start minio
	s, err := e2e.NewScenario("tempo_active_active")
	minio := e2edb.NewMinio(9000, "tempo")
	require.NotNil(t, minio)
	require.NoError(t, s.StartAndWaitReady(minio))

	// setup clusters
	tempoDistributorA, _ := createCluster(t, s, "a")
	_, tempoQueryFrontendB := createCluster(t, s, "b")

	// write to cluster A
	c, err := util.NewJaegerGRPCClient(tempoDistributorA.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, c)

	info := tempoUtil.NewTraceInfo(time.Now(), "")
	require.NoError(t, info.EmitAllBatches(c))

	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	// test metrics
	require.NoError(t, tempoDistributorA.WaitSumMetrics(e2e.Equals(spanCount(expected)), "tempo_distributor_spans_received_total"))

	// read from cluster B
	apiClient := tempoUtil.NewClient("http://"+tempoQueryFrontendB.Endpoint(3200), "")

	// query an in-memory trace
	queryAndAssertTrace(t, apiClient, info)
}

func createCluster(t *testing.T, s *e2e.Scenario, postfix string) (*e2e.HTTPService, *e2e.HTTPService) {
	require.NoError(t, util.CopyFileToSharedDir(s, "config-cross-cluster-"+postfix+".yaml", "config.yaml"))

	tempoIngester1 := util.NewNamedTempoIngester("ingester-"+postfix, 1)
	tempoIngester2 := util.NewNamedTempoIngester("ingester-"+postfix, 2)
	tempoIngester3 := util.NewNamedTempoIngester("ingester-"+postfix, 3)

	tempoDistributor := util.NewNamedTempoDistributor("distributor-" + postfix)
	tempoQueryFrontend := util.NewNamedTempoQueryFrontend("query-frontend-" + postfix)
	tempoQuerier := util.NewNamedTempoQuerier("querier-" + postfix)
	require.NoError(t, s.StartAndWaitReady(tempoIngester1, tempoIngester2, tempoIngester3, tempoDistributor, tempoQueryFrontend, tempoQuerier))

	// wait for active ingesters
	time.Sleep(1 * time.Second)
	matchers := []*labels.Matcher{
		{
			Type:  labels.MatchEqual,
			Name:  "name",
			Value: "ingester",
		},
		{
			Type:  labels.MatchEqual,
			Name:  "state",
			Value: "ACTIVE",
		},
	}
	require.NoError(t, tempoDistributor.WaitSumMetricsWithOptions(e2e.Equals(3), []string{`tempo_ring_members`}, e2e.WithLabelMatchers(matchers...), e2e.WaitMissingMetrics))

	return tempoDistributor, tempoQueryFrontend
}
