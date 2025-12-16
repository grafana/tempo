package deployments

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/integration/util"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
)

// TestLiveStorePartitionDownscale tests the complete partition downscale workflow:
// - Marking a partition as INACTIVE
// - Verifying new traffic goes to active partitions
// - Cancelling the downscale
// - Verifying normal operation resumes
func TestLiveStorePartitionDownscale(t *testing.T) {
	util.WithTempoHarness(t, util.TestHarnessConfig{
		Components: util.ComponentsRecentDataQuerying,
	}, func(h *util.TempoHarness) {
		// harness creates 2 live stores that own a first partition, shutdown B we don't want it for this test
		require.NoError(t, h.Services[util.ServiceLiveStoreZoneB].Stop())

		// start a new livestorea that own a second partition. -1 postfix is important to start a new partition!
		liveStoreParition1 := newLiveStore("live-store-zone-a-1")
		require.NoError(t, h.TestScenario.StartAndWaitReady(liveStoreParition1))

		// wait for 2 active partitions
		waitActivePartitions(t, h.Services[util.ServiceDistributor], 2)

		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, h.WriteTraceInfo(info, ""))

		liveStorePartition0 := h.Services[util.ServiceLiveStoreZoneA]
		liveStoreActive := waitForTraceInLiveStore(t, 1, liveStorePartition0, liveStoreParition1)
		var liveStoreInactive *e2e.HTTPService
		if liveStoreActive == liveStorePartition0 {
			liveStoreInactive = liveStorePartition0
		} else {
			liveStoreInactive = liveStoreParition1
		}

		apiClient := h.APIClientHTTP("")
		distributor := h.Services[util.ServiceDistributor]

		t.Run("verify partition is ACTIVE", func(t *testing.T) {
			for _, liveStore := range []*e2e.HTTPService{liveStoreInactive, liveStoreActive} {
				res := preparePartitionDownscale(t, http.MethodGet, liveStore)
				require.Equal(t, "PartitionActive", res.State)
			}
		})

		t.Run("prepare partition for downscale", func(t *testing.T) {
			// Set live-store's partition to INACTIVE
			res := preparePartitionDownscale(t, http.MethodPost, liveStoreInactive)
			require.Greater(t, res.Timestamp, int64(0))
			require.Equal(t, "PartitionInactive", res.State)

			// Verify state
			verifyPartitionState(t, distributor, "Inactive", 1)
			verifyPartitionState(t, distributor, "Active", 1)
		})

		t.Run("verify data is still accessible during downscale", func(t *testing.T) {
			util.QueryAndAssertTrace(t, apiClient, info)
		})

		t.Run("generate new trace during downscale", func(t *testing.T) {
			inactiveCount, err := liveStoreInactive.SumMetrics([]string{"tempo_live_store_records_processed_total"}, e2e.SkipMissingMetrics)
			require.NoError(t, err)

			info := tempoUtil.NewTraceInfo(time.Now(), "")
			require.NoError(t, h.WriteTraceInfo(info, ""))

			require.NoError(t, liveStoreActive.WaitSumMetrics(e2e.GreaterOrEqual(float64(1)), "tempo_live_store_records_processed_total"))

			// Verify inactive live-store didn't process new records
			inactiveCountAfter, err := liveStoreInactive.SumMetrics([]string{"tempo_live_store_records_processed_total"}, e2e.SkipMissingMetrics)
			require.NoError(t, err)
			require.Equal(t, inactiveCount, inactiveCountAfter)

			util.QueryAndAssertTrace(t, apiClient, info)
		})

		t.Run("cancel downscale preparation", func(t *testing.T) {
			res := preparePartitionDownscale(t, http.MethodDelete, liveStoreInactive)
			require.Equal(t, int64(0), res.Timestamp)
			require.Equal(t, "PartitionActive", res.State)

			verifyPartitionState(t, distributor, "Active", 2)
		})

		t.Run("verify normal operation after cancellation", func(t *testing.T) {
			inactiveCount, err := liveStoreInactive.SumMetrics([]string{"tempo_live_store_records_processed_total"}, e2e.SkipMissingMetrics)
			require.NoError(t, err)
			activeCount, err := liveStoreActive.SumMetrics([]string{"tempo_live_store_records_processed_total"}, e2e.SkipMissingMetrics)
			require.NoError(t, err)

			info := tempoUtil.NewTraceInfo(time.Now(), "")
			require.NoError(t, h.WriteTraceInfo(info, ""))

			util.QueryAndAssertTrace(t, apiClient, info)
			require.NoError(t, liveStoreActive.WaitSumMetrics(e2e.Greater(activeCount[0]), "tempo_live_store_records_processed_total"))
			require.NoError(t, liveStoreInactive.WaitSumMetrics(e2e.Greater(inactiveCount[0]), "tempo_live_store_records_processed_total"))
		})
	})
}

// TestLiveStoreDownscaleHappyPath tests the complete downscale flow where
// an inactive live-store is properly shut down
func TestLiveStoreDownscaleHappyPath(t *testing.T) {
	util.WithTempoHarness(t, util.TestHarnessConfig{
		Components: util.ComponentsRecentDataQuerying,
	}, func(h *util.TempoHarness) {
		// harness creates 2 live stores that own a first partition, shutdown B we don't want it for this test
		require.NoError(t, h.Services[util.ServiceLiveStoreZoneB].Stop())

		// start a new livestore that own a second partition. -1 postfix is important to start a new partition!
		liveStorePartition1 := newLiveStore("live-store-zone-a-1")
		require.NoError(t, h.TestScenario.StartAndWaitReady(liveStorePartition1))

		// wait for 2 active partitions
		distributor := h.Services[util.ServiceDistributor]
		waitActivePartitions(t, distributor, 2)

		// Mark partition as INACTIVE
		preparePartitionDownscale(t, http.MethodPost, liveStorePartition1)

		// Prepare for shutdown
		req, err := http.NewRequest("POST", "http://"+liveStorePartition1.Endpoint(3200)+"/live-store/prepare-downscale", nil)
		require.NoError(t, err)
		httpResp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, 204, httpResp.StatusCode)

		// Stop inactive live-store
		require.NoError(t, liveStorePartition1.Stop())

		waitActivePartitions(t, distributor, 1)

		// Verify only one active partition remains
		partitions := getRingStatus(t, distributor).Partitions

		activeCount := 0
		var activePartition *partitionData

		for _, partition := range partitions {
			if partition.State == activePartitionState {
				activePartition = &partition
				activeCount++
			}
		}
		require.Equal(t, 1, activeCount)
		require.NotNil(t, activePartition)
		require.Equal(t, int32(0), activePartition.ID)
		require.False(t, activePartition.Corrupted)
		require.Equal(t, activePartitionState, activePartition.State)
		require.Contains(t, activePartition.OwnerIDs, "live-store-zone-a-0")
	})
}

const activePartitionState = 2 // apparently state == 2 is active

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

func verifyPartitionState(t *testing.T, service *e2e.HTTPService, expectedState string, expectedCount int) {
	partitionStateMatchers := []*labels.Matcher{
		{Type: labels.MatchEqual, Name: "state", Value: expectedState},
		{Type: labels.MatchEqual, Name: "name", Value: "livestore-partitions"},
	}
	require.NoError(t, service.WaitSumMetricsWithOptions(
		e2e.Equals(float64(expectedCount)),
		[]string{"tempo_partition_ring_partitions"},
		e2e.WithLabelMatchers(partitionStateMatchers...),
	))
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
		close(ch)
		return liveStore
	case <-time.After(timeout):
		t.Fatalf("timeout waiting for trace in live-store after %s", timeout)
	}
	return nil
}

func waitActivePartitions(t *testing.T, service *e2e.HTTPService, count int) {
	matchers := []*labels.Matcher{
		{Type: labels.MatchEqual, Name: "state", Value: "Active"},
		{Type: labels.MatchEqual, Name: "name", Value: "livestore-partitions"},
	}
	require.NoError(t, service.WaitSumMetricsWithOptions(
		e2e.Equals(float64(count)),
		[]string{"tempo_partition_ring_partitions"},
		e2e.WithLabelMatchers(matchers...)), "distributor failed to see the partition ring")
}

func newLiveStore(name string) *e2e.HTTPService {
	return util.NewTempoService(name, "live-store", e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299))
}
