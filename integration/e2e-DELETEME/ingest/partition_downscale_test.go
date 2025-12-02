package ingest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/grafana/e2e"
	e2edb "github.com/grafana/e2e/db"
	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/httpclient"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
)

func TestPartitionDownscale(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	// copy config template to shared directory and expand template variables
	require.NoError(t, util.CopyFileToSharedDir(s, "config-partition-downscale.yaml", "config.yaml"))

	// Start dependencies
	kafka := e2edb.NewKafka()
	require.NoError(t, s.StartAndWaitReady(kafka))

	minio := e2edb.NewMinio(9000, "tempo")
	require.NotNil(t, minio)
	require.NoError(t, s.StartAndWaitReady(minio))

	// Start Tempo components
	distributor := util.NewTempoDistributor()
	ingester := util.NewTempoIngester(0)
	querier := util.NewTempoQuerier()
	queryFrontend := util.NewTempoQueryFrontend()

	require.NoError(t, s.StartAndWaitReady(distributor, ingester, querier, queryFrontend))

	// Wait until ingester and metrics-generator are active
	isServiceActiveMatcher := func(service string) []*labels.Matcher {
		return []*labels.Matcher{
			labels.MustNewMatcher(labels.MatchEqual, "name", service),
			labels.MustNewMatcher(labels.MatchEqual, "state", "ACTIVE"),
		}
	}
	require.NoError(t, distributor.WaitSumMetricsWithOptions(e2e.Equals(1), []string{`tempo_ring_members`}, e2e.WithLabelMatchers(isServiceActiveMatcher("ingester")...), e2e.WaitMissingMetrics))

	// Wait until joined to partition ring
	partitionStateMatchers := func(state string) []*labels.Matcher {
		return []*labels.Matcher{
			{Type: labels.MatchEqual, Name: "state", Value: state},
			{Type: labels.MatchEqual, Name: "name", Value: "ingester-partitions"},
		}
	}
	require.NoError(t, distributor.WaitSumMetricsWithOptions(e2e.Equals(1), []string{"tempo_partition_ring_partitions"}, e2e.WithLabelMatchers(partitionStateMatchers("Active")...)))

	// Get port for the Jaeger gRPC receiver endpoint
	c, err := util.NewJaegerToOTLPExporter(distributor.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, c)

	// Generate and emit initial traces
	info := tempoUtil.NewTraceInfo(time.Now(), "")
	require.NoError(t, info.EmitAllBatches(c))

	// Wait for traces to be received
	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)
	require.NoError(t, distributor.WaitSumMetrics(e2e.Equals(util.SpanCount(expected)), "tempo_distributor_spans_received_total"))

	// Create API client
	apiClient := httpclient.New("http://"+queryFrontend.Endpoint(3200), "")

	// Set ingester's partition to INACTIVE state (prepare for downscale)
	req, err := http.NewRequest("POST", "http://"+ingester.Endpoint(3200)+"/ingester/prepare-partition-downscale", nil)
	require.NoError(t, err)
	httpResp, err := apiClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, httpResp.StatusCode)

	// Verify ingester's partition is INACTIVE by checking the timestamp
	req, err = http.NewRequest("GET", "http://"+ingester.Endpoint(3200)+"/ingester/prepare-partition-downscale", nil)
	require.NoError(t, err)
	httpResp, err = apiClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, httpResp.StatusCode)
	var result map[string]any
	require.NoError(t, json.NewDecoder(httpResp.Body).Decode(&result))
	require.Greater(t, result["timestamp"].(float64), float64(0)) // ts > 0 ==> INACTIVE (when it was marked for downscale)

	require.NoError(t, distributor.WaitSumMetricsWithOptions(e2e.Equals(1), []string{"tempo_partition_ring_partitions"}, e2e.WithLabelMatchers(partitionStateMatchers("Inactive")...)))

	// Start block-builder (it should consume data from the downscaled partition)
	blockbuilder := util.NewTempoBlockBuilder(0)
	require.NoError(t, s.StartAndWaitReady(blockbuilder))

	// Wait for blocks to be flushed from the downscaled partition
	require.NoError(t, blockbuilder.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1), []string{"tempo_block_builder_flushed_blocks"}, e2e.WaitMissingMetrics))
	require.NoError(t, queryFrontend.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1), []string{"tempodb_blocklist_length"},
		e2e.WaitMissingMetrics, e2e.WithLabelMatchers(&labels.Matcher{Type: labels.MatchEqual, Name: "tenant", Value: "single-tenant"})))

	// Verify initial traces can be queried from backend storage
	trace, err := apiClient.QueryTrace(info.HexID())
	require.NoError(t, err)
	require.NotNil(t, trace)

	// Set ingester's partition back to ACTIVE state
	req, err = http.NewRequest("DELETE", "http://"+ingester.Endpoint(3200)+"/ingester/prepare-partition-downscale", nil)
	require.NoError(t, err)
	httpResp, err = apiClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, httpResp.StatusCode)

	// Verify ingester's partition is ACTIVE by checking the timestamp is 0
	req, err = http.NewRequest("GET", "http://"+ingester.Endpoint(3200)+"/ingester/prepare-partition-downscale", nil)
	require.NoError(t, err)
	httpResp, err = apiClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, httpResp.StatusCode)
	require.NoError(t, json.NewDecoder(httpResp.Body).Decode(&result))
	require.Equal(t, float64(0), result["timestamp"].(float64)) // ts == 0 ==> ACTIVE

	require.NoError(t, distributor.WaitSumMetricsWithOptions(e2e.Equals(1), []string{"tempo_partition_ring_partitions"}, e2e.WithLabelMatchers(partitionStateMatchers("Active")...)))

	// Generate and emit more traces after reactivating the partition
	info2 := tempoUtil.NewTraceInfo(time.Now(), "")
	require.NoError(t, info2.EmitAllBatches(c))

	// Wait for new traces to be received
	expected2, err := info2.ConstructTraceFromEpoch()
	require.NoError(t, err)
	require.NoError(t, distributor.WaitSumMetrics(e2e.Equals(util.SpanCount(expected)+util.SpanCount(expected2)), "tempo_distributor_spans_received_total"))

	// Wait for the new traces to be flushed by block-builder
	require.NoError(t, blockbuilder.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(2), []string{"tempo_block_builder_flushed_blocks"}, e2e.WaitMissingMetrics))

	// Verify all traces using trace ID lookup
	trace, err = apiClient.QueryTrace(info.HexID())
	require.NoError(t, err)
	require.NotNil(t, trace)
	require.Equal(t, util.SpanCount(expected), util.SpanCount(trace))

	trace2, err := apiClient.QueryTrace(info2.HexID())
	require.NoError(t, err)
	require.NotNil(t, trace2)
	require.Equal(t, util.SpanCount(expected2), util.SpanCount(trace2))
}

func TestLiveStorePartitionDownscale(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, "config-live-store.yaml", "config.yaml"))

	// Start dependencies
	kafka := e2edb.NewKafka()
	require.NoError(t, s.StartAndWaitReady(kafka))

	// Start live-store instances
	liveStore0 := util.NewTempoLiveStore(0)
	liveStore1 := util.NewTempoLiveStore(1)
	require.NoError(t, s.StartAndWaitReady(liveStore0, liveStore1))
	waitUntilJoinedToPartitionRing(t, liveStore0, 2) // wait for both to join

	// Start other Tempo components
	distributor := util.NewTempoDistributor()
	querier := util.NewTempoQuerier()
	queryFrontend := util.NewTempoQueryFrontend()
	require.NoError(t, s.StartAndWaitReady(distributor, querier, queryFrontend))

	// Generate and emit initial traces
	c, err := util.NewJaegerToOTLPExporter(distributor.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, c)

	tracePool := newTracePool()
	firstTrace, err := tracePool.Generate()
	require.NoError(t, err)
	require.NoError(t, firstTrace.EmitAllBatches(c))

	// Wait for traces to be received
	expected, err := firstTrace.ConstructTraceFromEpoch()
	require.NoError(t, err)
	distributorSpanCount := util.SpanCount(expected)
	require.NoError(t, distributor.WaitSumMetrics(e2e.Equals(distributorSpanCount), "tempo_distributor_spans_received_total"))

	// Verify traces are processed by one of the live-stores
	// First try to find which live-store processed the records
	// We will shutdown this instance later
	var liveStoreInactive, liveStoreActive *e2e.HTTPService

	ls := waitForTraceInLiveStore(t, 1, liveStore0, liveStore1)
	if ls == liveStore0 {
		liveStoreInactive = liveStore0
		liveStoreActive = liveStore1
	} else {
		liveStoreInactive = liveStore1
		liveStoreActive = liveStore0
	}

	apiClient := httpclient.New("http://"+queryFrontend.Endpoint(3200), "")

	t.Run("verify partition is ACTIVE", func(t *testing.T) {
		for _, liveStore := range []*e2e.HTTPService{liveStoreInactive, liveStoreActive} {
			res := preparePartitionDownscale(t, http.MethodGet, liveStore)
			require.Equal(t, "PartitionActive", res.State)
		}
	})

	t.Run("prepare partition for downscale", func(t *testing.T) {
		// Set live-store's partition to INACTIVE state (prepare for downscale)
		for range 2 { // repeated action will be noop
			res := preparePartitionDownscale(t, http.MethodPost, liveStoreInactive)
			require.Greater(t, res.Timestamp, int64(0)) // ts > 0 ==> INACTIVE
			require.Equal(t, "PartitionInactive", res.State)
		}

		// Test GET method
		res := preparePartitionDownscale(t, http.MethodGet, liveStoreInactive)
		require.Greater(t, res.Timestamp, int64(0)) // ts > 0 ==> INACTIVE
		require.Equal(t, "PartitionInactive", res.State)
		res = preparePartitionDownscale(t, http.MethodGet, liveStoreActive)
		require.Equal(t, "PartitionActive", res.State) // still active

		for _, component := range []*e2e.HTTPService{liveStoreInactive, liveStoreActive, distributor} {
			verifyPartitionState(t, component, "Inactive", 1)
			verifyPartitionState(t, component, "Active", 1)
		}
	})

	t.Run("verify data is still accessible during downscale", func(t *testing.T) {
		// Even with partition INACTIVE, existing data should still be queryable
		// from the live-store during the grace period
		trace, err := apiClient.QueryTrace(firstTrace.HexID())
		require.NoError(t, err)
		require.NotNil(t, trace)
		require.Equal(t, util.SpanCount(expected), util.SpanCount(trace))
	})

	t.Run("generate new traces during downscale", func(t *testing.T) {
		inactiveCount, err := liveStoreInactive.SumMetrics([]string{"tempo_live_store_records_processed_total"}, e2e.SkipMissingMetrics)
		require.NoError(t, err)

		// Send 10 traces. In case of a bug when it sends to both live-stores,
		// possibility of false pass is 1/1024.
		const numTraces = 10
		traces := make([]string, 0, numTraces)
		spanCounts := make([]float64, 0, numTraces)

		// Generate new traces - these should be processed by the other live-store
		// since the first one is in INACTIVE state
		for range numTraces {
			trace, err := tracePool.Generate()
			require.NoError(t, err)
			require.NoError(t, trace.EmitAllBatches(c))
			expected, err := trace.ConstructTraceFromEpoch()
			require.NoError(t, err)
			traces = append(traces, trace.HexID())
			spanCounts = append(spanCounts, util.SpanCount(expected))
		}

		for _, spanCount := range spanCounts {
			distributorSpanCount += spanCount
		}
		// Wait for new traces to be received
		require.NoError(t, distributor.WaitSumMetrics(e2e.Equals(distributorSpanCount), "tempo_distributor_spans_received_total"))
		require.NoError(t, liveStoreActive.WaitSumMetrics(e2e.GreaterOrEqual(float64(1)), "tempo_live_store_records_processed_total"))

		// Verify inactive live-store still has the same number of processed records
		inactiveCountAfter, err := liveStoreInactive.SumMetrics([]string{"tempo_live_store_records_processed_total"}, e2e.SkipMissingMetrics)
		require.NoError(t, err)
		require.Equal(t, inactiveCount, inactiveCountAfter, "inactive count should be the same")

		// The other live-store (not the one being downscaled) should process new records
		for i, traceID := range traces {
			actualTrace, err := apiClient.QueryTrace(traceID)
			require.NoError(t, err)
			require.Equal(t, spanCounts[i], util.SpanCount(actualTrace))
		}
	})

	t.Run("cancel downscale preparation", func(t *testing.T) {
		// Cancel partition downscale
		res := preparePartitionDownscale(t, http.MethodDelete, liveStoreInactive)
		require.Equal(t, int64(0), res.Timestamp) // ts == 0 ==> ACTIVE
		require.Equal(t, "PartitionActive", res.State)

		// Verify partition is back to ACTIVE
		res = preparePartitionDownscale(t, http.MethodGet, liveStoreInactive)
		require.Equal(t, int64(0), res.Timestamp) // ts == 0 ==> ACTIVE
		require.Equal(t, "PartitionActive", res.State)

		for _, component := range []*e2e.HTTPService{liveStoreInactive, liveStoreActive, distributor} {
			verifyPartitionState(t, component, "Active", 2)
		}
		for _, liveStore := range []*e2e.HTTPService{liveStoreInactive, liveStoreActive} {
			res := preparePartitionDownscale(t, http.MethodGet, liveStore)
			require.Equal(t, "PartitionActive", res.State)
		}
	})

	t.Run("verify normal operation after cancellation", func(t *testing.T) {
		inactiveCount, err := liveStoreInactive.SumMetrics([]string{"tempo_live_store_records_processed_total"}, e2e.SkipMissingMetrics)
		require.NoError(t, err)
		activeCount, err := liveStoreActive.SumMetrics([]string{"tempo_live_store_records_processed_total"}, e2e.SkipMissingMetrics)
		require.NoError(t, err)

		// Send 10 traces. Possibility of false positive is 1/1024.
		const numTraces = 10
		traces := make([]string, 0, numTraces)
		spanCounts := make([]float64, 0, numTraces)

		for range numTraces {
			trace, err := tracePool.Generate()
			require.NoError(t, err)
			require.NoError(t, trace.EmitAllBatches(c))
			expected, err := trace.ConstructTraceFromEpoch()
			require.NoError(t, err)
			traces = append(traces, trace.HexID())
			spanCounts = append(spanCounts, util.SpanCount(expected))
		}

		// Wait for new traces to be received
		for _, spanCount := range spanCounts {
			distributorSpanCount += spanCount
		}
		// Verify distributor and both live-stores have processed new records
		require.NoError(t, distributor.WaitSumMetrics(e2e.Equals(distributorSpanCount), "tempo_distributor_spans_received_total"))
		require.NoError(t, liveStoreActive.WaitSumMetrics(e2e.Greater(activeCount[0]), "tempo_live_store_records_processed_total"))
		require.NoError(t, liveStoreInactive.WaitSumMetrics(e2e.Greater(inactiveCount[0]), "tempo_live_store_records_processed_total"))

		// The other live-store (not the one being downscaled) should process new records
		for i, traceID := range traces {
			actualTrace, err := apiClient.QueryTrace(traceID)
			require.NoError(t, err)
			require.Equal(t, spanCounts[i], util.SpanCount(actualTrace))
		}
	})
}

func TestLiveStoreDownscaleHappyPath(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, "config-live-store.yaml", "config.yaml"))

	// Start dependencies
	kafka := e2edb.NewKafka()
	require.NoError(t, s.StartAndWaitReady(kafka))

	liveStoreActive := util.NewTempoLiveStore(0)
	liveStoreInactive := util.NewTempoLiveStore(1)
	require.NoError(t, s.StartAndWaitReady(liveStoreActive, liveStoreInactive))
	waitUntilJoinedToPartitionRing(t, liveStoreActive, 2) // wait for both to join

	distributor := util.NewTempoDistributor()
	require.NoError(t, s.StartAndWaitReady(distributor))

	// Wait until both live-stores are active in the ring
	isServiceActiveMatcher := func(service, state string) []*labels.Matcher {
		return []*labels.Matcher{
			labels.MustNewMatcher(labels.MatchEqual, "name", service),
			labels.MustNewMatcher(labels.MatchEqual, "state", state),
		}
	}
	require.NoError(t, distributor.WaitSumMetricsWithOptions(
		e2e.Equals(2),
		[]string{`tempo_ring_members`},
		e2e.WithLabelMatchers(isServiceActiveMatcher("live-store", "ACTIVE")...),
		e2e.WaitMissingMetrics,
	))

	// Change partition state to INACTIVE
	preparePartitionDownscale(t, http.MethodPost, liveStoreInactive)

	// Prepare downscale, right before shutdown
	req, err := http.NewRequest("POST", "http://"+liveStoreInactive.Endpoint(3200)+"/live-store/prepare-downscale", nil)
	require.NoError(t, err)
	httpResp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, 204, httpResp.StatusCode) // PrepareDownscaleHandler returns StatusNoContent (204)

	// Shutdown inactive live-store
	require.NoError(t, liveStoreInactive.Stop())
	time.Sleep(5 * time.Second)

	// Only one ACTIVE live-store remains
	partitions := getRingStatus(t, distributor).Partitions
	require.Equal(t, 1, len(partitions), "only one ACTIVE live-store should remain, actual result: %v", partitions)
	partition := partitions[0]

	require.Equal(t, partitionData{ID: 0, Corrupted: false, State: 2, OwnerIDs: []string{"live-store-0"}}, partition)
}

func TestLiveStoreRestart(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, "config-live-store.yaml", "config.yaml"))

	// Start dependencies
	kafka := e2edb.NewKafka()
	require.NoError(t, s.StartAndWaitReady(kafka))

	liveStore := util.NewTempoLiveStore(0)
	distributor := util.NewTempoDistributor()
	require.NoError(t, s.StartAndWaitReady(liveStore, distributor))
	waitUntilJoinedToPartitionRing(t, liveStore, 1)

	// Change partition state to INACTIVE
	preparePartitionDownscale(t, http.MethodPost, liveStore)

	// Check that after stop (restart), it is not removed from the ring
	require.NoError(t, liveStore.Stop())
	time.Sleep(5 * time.Second)
	rs := getRingStatus(t, distributor)
	require.Equal(t, 1, len(rs.Partitions))
}

type tracePool struct {
	pool map[string]*tempoUtil.TraceInfo
	ts   time.Time
	idx  int
}

func newTracePool() *tracePool {
	return &tracePool{
		pool: make(map[string]*tempoUtil.TraceInfo),
		ts:   time.Now(),
		idx:  0,
	}
}

func (p *tracePool) Generate() (*tempoUtil.TraceInfo, error) {
	p.idx++
	// We add a second to make sure the trace ID is different
	// as seed of the rand is based on the timestamp
	info := tempoUtil.NewTraceInfo(p.ts.Add(time.Second*time.Duration(p.idx)), "")
	if p.pool[info.HexID()] != nil {
		return nil, fmt.Errorf("test is invalid, generated the same trace ID: %s", info.HexID())
	}
	p.pool[info.HexID()] = info
	return info, nil
}

func TestTracePool(t *testing.T) {
	pool := newTracePool()
	for i := range 50 {
		trace, err := pool.Generate()
		require.NoError(t, err, "error generating trace %d", i)
		require.NotNil(t, trace, "trace is nil for %d", i)
	}
}

func waitForTraceInLiveStore(t *testing.T, expectedRecords int, liveStores ...*e2e.HTTPService) *e2e.HTTPService {
	ch := make(chan *e2e.HTTPService)
	timeout := 30 * time.Second

	for _, liveStore := range liveStores {
		go func(liveStore *e2e.HTTPService) {
			err := liveStore.WaitSumMetrics(e2e.GreaterOrEqual(float64(expectedRecords)), "tempo_live_store_records_processed_total")
			if err == nil {
				ch <- liveStore
			}
		}(liveStore)
	}

	select {
	case liveStore := <-ch:
		{
			close(ch)
			return liveStore
		}
	case <-time.After(timeout):
		t.Fatalf("timeout waiting for trace to appear in any live-store after %s", timeout)
	}
	return nil
}

func verifyPartitionState(t *testing.T, liveStore *e2e.HTTPService, expectedState string, expectedCount int) {
	partitionStateMatchers := []*labels.Matcher{
		{Type: labels.MatchEqual, Name: "state", Value: expectedState},
		{Type: labels.MatchEqual, Name: "name", Value: "livestore-partitions"},
	}
	require.NoError(t, liveStore.WaitSumMetricsWithOptions(e2e.Equals(float64(expectedCount)), []string{"tempo_partition_ring_partitions"}, e2e.WithLabelMatchers(partitionStateMatchers...)))
}

type preparePartitionDownscaleResponse struct {
	Timestamp int64  `json:"timestamp"`
	State     string `json:"state"`
}

type ringStatus struct {
	Partitions []partitionData `json:"partitions"`
}

type partitionData struct {
	ID        int32    `json:"id"`
	Corrupted bool     `json:"corrupted"`
	State     int      `json:"state"`
	OwnerIDs  []string `json:"owner_ids"`
	// skip the fields below
	// StateTimestamp string   `json:"state_timestamp"`
	// Tokens         []uint32  `json:"tokens"`
}

func getRingStatus(t *testing.T, service *e2e.HTTPService) ringStatus {
	req, err := http.NewRequest("GET", "http://"+service.Endpoint(3200)+"/partition-ring", nil)
	req.Header.Set("Accept", "application/json")
	require.NoError(t, err)
	httpResp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, httpResp.StatusCode)

	var result ringStatus
	require.NoError(t, json.NewDecoder(httpResp.Body).Decode(&result))
	return result
}

func preparePartitionDownscale(t *testing.T, method string, liveStore *e2e.HTTPService) preparePartitionDownscaleResponse {
	req, err := http.NewRequest(method, "http://"+liveStore.Endpoint(3200)+"/live-store/prepare-partition-downscale", nil)
	require.NoError(t, err)
	httpResp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, httpResp.StatusCode)

	var result preparePartitionDownscaleResponse
	require.NoError(t, json.NewDecoder(httpResp.Body).Decode(&result))
	return result
}
