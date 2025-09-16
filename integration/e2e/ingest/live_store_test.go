package ingest

import (
	"testing"
	"time"

	"github.com/grafana/e2e"
	e2edb "github.com/grafana/e2e/db"
	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/httpclient"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiveStore(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	// copy config template to shared directory and expand template variables
	require.NoError(t, util.CopyFileToSharedDir(s, "config-live-store.yaml", "config.yaml"))

	kafka := e2edb.NewKafka()
	require.NoError(t, s.StartAndWaitReady(kafka))

	liveStore0 := util.NewTempoLiveStore(0)
	liveStore1 := util.NewTempoLiveStore(1)
	require.NoError(t, s.StartAndWaitReady(liveStore0, liveStore1))
	waitUntilJoinedToPartitionRing(t, liveStore0, 2)

	distributor := util.NewTempoDistributor()
	require.NoError(t, s.StartAndWaitReady(distributor))

	// Get port for the Jaeger gRPC receiver endpoint
	c, err := util.NewJaegerToOTLPExporter(distributor.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, c)

	info := tempoUtil.NewTraceInfo(time.Now(), "")
	require.NoError(t, info.EmitAllBatches(c))

	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	// test metrics
	require.NoError(t, distributor.WaitSumMetrics(e2e.Equals(util.SpanCount(expected)), "tempo_distributor_spans_received_total"))

	liveStoreProcessedRecords := liveStore0
	err = liveStoreProcessedRecords.WaitSumMetrics(e2e.Equals(1), "tempo_live_store_traces_created_total")
	if err != nil { // then the trace went to second live store
		require.Error(t, err, "metric not found")
		liveStoreProcessedRecords = liveStore1
		err = liveStoreProcessedRecords.WaitSumMetrics(e2e.Equals(1), "tempo_live_store_traces_created_total")
		require.NoError(t, err)
	}

	// the number of processed records should be reasonable
	assert.NoError(t, liveStoreProcessedRecords.WaitSumMetrics(e2e.Between(1, 25), "tempo_live_store_records_processed_total"))
}

// TestLiveStoreAPISmoke tests the API endpoints that will hit live store.
// It will be deleted after api tests will start using Rhythm.
func TestSmokeLiveStoreAPI(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	// copy config template to shared directory and expand template variables
	require.NoError(t, util.CopyFileToSharedDir(s, "config-live-store.yaml", "config.yaml"))

	frontend, distributor := StartTempoWithLiveStore(t, s)
	client := httpclient.New("http://"+frontend.Endpoint(3200), "")

	// Get port for the Jaeger gRPC receiver endpoint
	c, err := util.NewJaegerToOTLPExporter(distributor.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, c)

	info := tempoUtil.NewTraceInfo(time.Now(), "")
	require.NoError(t, info.EmitAllBatches(c))
	time.Sleep(10 * time.Second) // wait for the trace to be ingested

	t.Run("get trace by id", func(t *testing.T) {
		tr, err := client.QueryTrace(info.HexID())
		require.NoError(t, err)
		require.NotNil(t, tr)
		require.Greater(t, int(util.SpanCount(tr)), 0)
	})

	t.Run("get trace by id v2", func(t *testing.T) {
		resp, err := client.QueryTraceV2(info.HexID())
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Trace)
		require.Greater(t, int(util.SpanCount(resp.Trace)), 0)
	})

	t.Run("search tags v1", func(t *testing.T) {
		resp, err := client.SearchTags()
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Greater(t, len(resp.TagNames), 0)
	})

	t.Run("search tag values v1", func(t *testing.T) {
		resp, err := client.SearchTagValues("service.name")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Greater(t, len(resp.TagValues), 0)
	})

	t.Run("search tags v2", func(t *testing.T) {
		resp, err := client.SearchTagsV2()
		require.NoError(t, err)
		require.NotNil(t, resp)
		total := 0
		for _, sc := range resp.Scopes {
			total += len(sc.Tags)
		}
		require.Greater(t, total, 0)
	})

	t.Run("search tag values v2", func(t *testing.T) {
		resp, err := client.SearchTagValuesV2("resource.service.name", "")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Greater(t, len(resp.TagValues), 0)
	})

	t.Run("metrics query range count_over_time", func(t *testing.T) {
		qr, err := client.MetricsQueryRange("{} | count_over_time()", 0, 0, "", 0)
		require.NoError(t, err)
		require.NotNil(t, qr)
		require.Equal(t, 1, len(qr.Series))
		var count int
		for _, s := range qr.Series[0].Samples {
			count += int(s.Value)
		}
		require.Greater(t, count, 0)
	})
}

func waitUntilJoinedToPartitionRing(t *testing.T, liveStore *e2e.HTTPService, numPartitions float64) {
	matchers := []*labels.Matcher{
		{Type: labels.MatchEqual, Name: "state", Value: "Active"},
		{Type: labels.MatchEqual, Name: "name", Value: "livestore-partitions"},
	}
	require.NoError(t, liveStore.WaitSumMetricsWithOptions(e2e.Equals(numPartitions), []string{"tempo_partition_ring_partitions"}, e2e.WithLabelMatchers(matchers...)))
}

func StartTempoWithLiveStore(t *testing.T, s *e2e.Scenario) (*e2e.HTTPService, *e2e.HTTPService) {
	kafka := e2edb.NewKafka()
	require.NoError(t, s.StartAndWaitReady(kafka))

	liveStore1 := util.NewTempoLiveStore(0)
	liveStore2 := util.NewTempoLiveStore(1)
	require.NoError(t, s.StartAndWaitReady(liveStore1, liveStore2))
	waitUntilJoinedToPartitionRing(t, liveStore1, 2)

	distributor := util.NewTempoDistributor()
	frontend := util.NewTempoQueryFrontend()
	require.NoError(t, s.StartAndWaitReady(distributor, frontend, util.NewTempoQuerier()))

	return frontend, distributor
}
