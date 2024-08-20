package deployments

import (
	"sync"
	"testing"
	"time"

	"github.com/grafana/e2e"
	e2edb "github.com/grafana/e2e/db"
	"github.com/grafana/tempo/v2/integration/util"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v2/pkg/httpclient"
	tempoUtil "github.com/grafana/tempo/v2/pkg/util"
)

const (
	configHA = "config-scalable-single-binary.yaml"
)

func TestScalableSingleBinary(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	minio := e2edb.NewMinio(9000, "tempo")
	require.NotNil(t, minio)
	require.NoError(t, s.StartAndWaitReady(minio))

	// copy configuration file over to shared dir
	require.NoError(t, util.CopyFileToSharedDir(s, configHA, "config.yaml"))

	// start three scalable single binary tempos in parallel
	var wg sync.WaitGroup
	var tempo1, tempo2, tempo3 *e2e.HTTPService
	wg.Add(3)
	go func() {
		tempo1 = util.NewTempoScalableSingleBinary(1)
		wg.Done()
	}()
	go func() {
		tempo2 = util.NewTempoScalableSingleBinary(2)
		wg.Done()
	}()
	go func() {
		tempo3 = util.NewTempoScalableSingleBinary(3)
		wg.Done()
	}()
	wg.Wait()
	require.NoError(t, s.StartAndWaitReady(tempo1, tempo2, tempo3))

	// wait for 2 active ingesters
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

	t.Logf("tempo1.Endpoint(): %+v", tempo1.Endpoint(3200))

	require.NoError(t, tempo1.WaitSumMetricsWithOptions(e2e.Equals(3), []string{`tempo_ring_members`}, e2e.WithLabelMatchers(matchers...), e2e.WaitMissingMetrics))
	require.NoError(t, tempo2.WaitSumMetricsWithOptions(e2e.Equals(3), []string{`tempo_ring_members`}, e2e.WithLabelMatchers(matchers...), e2e.WaitMissingMetrics))
	require.NoError(t, tempo3.WaitSumMetricsWithOptions(e2e.Equals(3), []string{`tempo_ring_members`}, e2e.WithLabelMatchers(matchers...), e2e.WaitMissingMetrics))

	c1, err := util.NewJaegerGRPCClient(tempo1.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, c1)

	c2, err := util.NewJaegerGRPCClient(tempo2.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, c2)

	c3, err := util.NewJaegerGRPCClient(tempo3.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, c3)

	info := tempoUtil.NewTraceInfo(time.Unix(1632169410, 0), "")
	require.NoError(t, info.EmitBatches(c1))

	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	// test metrics
	require.NoError(t, tempo1.WaitSumMetrics(e2e.Equals(util.SpanCount(expected)), "tempo_distributor_spans_received_total"))

	// wait trace_idle_time and ensure trace is created in ingester
	time.Sleep(1 * time.Second)
	require.NoError(t, tempo1.WaitSumMetricsWithOptions(e2e.Less(3), []string{"tempo_ingester_traces_created_total"}, e2e.WaitMissingMetrics))

	for _, i := range []*e2e.HTTPService{tempo1, tempo2, tempo3} {
		util.CallFlush(t, i)
		require.NoError(t, i.WaitSumMetrics(e2e.Equals(1), "tempo_ingester_blocks_flushed_total"))
		util.CallIngesterRing(t, i)
		util.CallCompactorRing(t, i)
		util.CallStatus(t, i)
		util.CallBuildinfo(t, i)
	}

	apiClient1 := httpclient.New("http://"+tempo1.Endpoint(3200), "")

	util.QueryAndAssertTrace(t, apiClient1, info)

	err = tempo1.Kill()
	require.NoError(t, err)

	// Push to one of the instances that are still running.
	require.NoError(t, info.EmitBatches(c2))

	err = tempo2.Kill()
	require.NoError(t, err)

	err = tempo3.Kill()
	require.NoError(t, err)
}
