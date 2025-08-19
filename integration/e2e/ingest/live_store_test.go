package ingest

import (
	"testing"
	"time"

	"github.com/grafana/e2e"
	e2edb "github.com/grafana/e2e/db"
	"github.com/grafana/tempo/integration/util"
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

	// Wait until joined to partition ring
	matchers := []*labels.Matcher{
		{Type: labels.MatchEqual, Name: "state", Value: "Active"},
		{Type: labels.MatchEqual, Name: "name", Value: "ingester-partitions"},
	}
	require.NoError(t, liveStore0.WaitSumMetricsWithOptions(e2e.Equals(2), []string{"tempo_partition_ring_partitions"}, e2e.WithLabelMatchers(matchers...)))

	ingester0 := util.NewTempoIngester(0)
	ingester1 := util.NewTempoIngester(1)
	require.NoError(t, s.StartAndWaitReady(ingester0, ingester1))

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
