package livestore

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/require"
)

func TestLiveStoreBasicConsume(t *testing.T) {
	tmpDir := t.TempDir()

	liveStore, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)
	require.NotNil(t, liveStore)

	// Push 10 traces and store their IDs and expected traces
	expectedTraces := make(map[string]*tempopb.Trace)
	for i := 0; i < 10; i++ {
		expectedID, expectedTrace := pushToLiveStore(t, liveStore)
		expectedTraces[string(expectedID)] = expectedTrace
	}

	// Test that all 10 traces can be found by ID
	for id, expectedTrace := range expectedTraces {
		requireTraceInLiveStore(t, liveStore, []byte(id), expectedTrace)
	}
}

// TestLiveStoreFullBlockLifecycleCheating tests all stages of the trace lifecycle by "cheating". e.g. it
// uses knowledge of the internal state of the livestore and its instances to check the correct blocks exist
func TestLiveStoreFullBlockLifecycleCheating(t *testing.T) {
	tmpDir := t.TempDir()

	liveStore, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)
	require.NotNil(t, liveStore)

	inst, err := liveStore.getOrCreateInstance(testTenantID)
	require.NoError(t, err)

	// push data
	expectedID, expectedTrace := pushToLiveStore(t, liveStore)

	// in live traces
	requireTraceInLiveStore(t, liveStore, expectedID, expectedTrace)
	requireInstanceState(t, inst, instanceState{liveTraces: 1, walBlocks: 0, completeBlocks: 0})

	// cut to head block and test
	err = inst.cutIdleTraces(true)
	require.NoError(t, err)

	requireTraceInLiveStore(t, liveStore, expectedID, expectedTrace)
	requireTraceInBlock(t, inst.headBlock, expectedID, expectedTrace)
	requireInstanceState(t, inst, instanceState{liveTraces: 0, walBlocks: 0, completeBlocks: 0})

	// cut a new head block. old head block is in wal blocks
	walUUID, err := inst.cutBlocks(true)
	require.NoError(t, err)

	requireTraceInLiveStore(t, liveStore, expectedID, expectedTrace)
	requireTraceInBlock(t, inst.walBlocks[walUUID], expectedID, expectedTrace)
	requireInstanceState(t, inst, instanceState{liveTraces: 0, walBlocks: 1, completeBlocks: 0})

	// force complete the wal block
	err = inst.completeBlock(t.Context(), walUUID)
	require.NoError(t, err)

	requireTraceInLiveStore(t, liveStore, expectedID, expectedTrace)
	requireTraceInBlock(t, inst.completeBlocks[walUUID], expectedID, expectedTrace)
	requireInstanceState(t, inst, instanceState{liveTraces: 0, walBlocks: 0, completeBlocks: 1})

	// stop gracefully
	err = services.StopAndAwaitTerminated(t.Context(), liveStore)
	require.NoError(t, err)
}

func TestLiveStoreReplaysTraceInLiveTraces(t *testing.T) {
	tmpDir := t.TempDir()

	liveStore, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)
	require.NotNil(t, liveStore)

	// push data
	expectedID, expectedTrace := pushToLiveStore(t, liveStore)

	// stop the live store and then create a new one to simulate a restart and replay the data on disk
	err = services.StopAndAwaitTerminated(t.Context(), liveStore)
	require.NoError(t, err)

	liveStore, err = defaultLiveStore(t, tmpDir)
	require.NoError(t, err)

	requireTraceInLiveStore(t, liveStore, expectedID, expectedTrace)
	requireInstanceState(t, liveStore.instances[testTenantID], instanceState{liveTraces: 0, walBlocks: 1, completeBlocks: 0})
}

func TestLiveStoreReplaysTraceInHeadBlock(t *testing.T) {
	tmpDir := t.TempDir()

	liveStore, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)
	require.NotNil(t, liveStore)

	// push data
	expectedID, expectedTrace := pushToLiveStore(t, liveStore)

	inst, err := liveStore.getOrCreateInstance(testTenantID)
	require.NoError(t, err)

	// cut to head block
	err = inst.cutIdleTraces(true)
	require.NoError(t, err)

	// stop the live store and then create a new one to simulate a restart and replay the data on disk
	err = services.StopAndAwaitTerminated(t.Context(), liveStore)
	require.NoError(t, err)

	liveStore, err = defaultLiveStore(t, tmpDir)
	require.NoError(t, err)

	requireTraceInLiveStore(t, liveStore, expectedID, expectedTrace)
	requireInstanceState(t, liveStore.instances[testTenantID], instanceState{liveTraces: 0, walBlocks: 1, completeBlocks: 0})
}

func TestLiveStoreReplaysTraceInWalBlocks(t *testing.T) {
	tmpDir := t.TempDir()

	liveStore, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)
	require.NotNil(t, liveStore)

	// push data
	expectedID, expectedTrace := pushToLiveStore(t, liveStore)

	inst, err := liveStore.getOrCreateInstance(testTenantID)
	require.NoError(t, err)

	// cut to head block
	err = inst.cutIdleTraces(true)
	require.NoError(t, err)

	// cut head to wal blocks
	_, err = inst.cutBlocks(true)
	require.NoError(t, err)

	// stop the live store and then create a new one to simulate a restart and replay the data on disk
	err = services.StopAndAwaitTerminated(t.Context(), liveStore)
	require.NoError(t, err)

	liveStore, err = defaultLiveStore(t, tmpDir)
	require.NoError(t, err)

	requireTraceInLiveStore(t, liveStore, expectedID, expectedTrace)
	requireInstanceState(t, liveStore.instances[testTenantID], instanceState{liveTraces: 0, walBlocks: 1, completeBlocks: 0})
}

func TestLiveStoreReplaysTraceInCompleteBlocks(t *testing.T) {
	tmpDir := t.TempDir()

	liveStore, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)
	require.NotNil(t, liveStore)

	// push data
	expectedID, expectedTrace := pushToLiveStore(t, liveStore)

	inst, err := liveStore.getOrCreateInstance(testTenantID)
	require.NoError(t, err)

	// cut to head block
	err = inst.cutIdleTraces(true)
	require.NoError(t, err)

	// cut head to wal blocks
	walUUID, err := inst.cutBlocks(true)
	require.NoError(t, err)

	// complete the wal blocks
	err = inst.completeBlock(t.Context(), walUUID)
	require.NoError(t, err)

	// stop the live store and then create a new one to simulate a restart and replay the data on disk
	err = services.StopAndAwaitTerminated(t.Context(), liveStore)
	require.NoError(t, err)

	liveStore, err = defaultLiveStore(t, tmpDir)
	require.NoError(t, err)

	requireTraceInLiveStore(t, liveStore, expectedID, expectedTrace)
	requireInstanceState(t, liveStore.instances[testTenantID], instanceState{liveTraces: 0, walBlocks: 0, completeBlocks: 1})
}

func TestLiveStoreConsumeDropsOldRecords(t *testing.T) {
	// default live store uses the default complete block timeout
	ls, _ := defaultLiveStore(t, t.TempDir())

	// Reset metrics
	metricRecordsProcessed.Reset()
	metricRecordsDropped.Reset()

	now := time.Now()
	older := now.Add(-1 * (defaultCompleteBlockTimeout + time.Second))
	newer := now.Add(-1 * (defaultCompleteBlockTimeout - time.Second))

	// Create test records - some old, some new
	records := []record{
		{
			tenantID:  "tenant1",
			timestamp: older, // Too old (older than CompleteBlockTimeout)
			content:   createValidPushRequest(t),
		},
		{
			tenantID:  "tenant1",
			timestamp: newer, // Valid (newer than CompleteBlockTimeout)
			content:   createValidPushRequest(t),
		},
		{
			tenantID:  "tenant2",
			timestamp: older, // Too old
			content:   createValidPushRequest(t),
		},
		{
			tenantID:  "tenant2",
			timestamp: newer, // Valid
			content:   createValidPushRequest(t),
		},
	}

	// Call consume
	err := ls.consume(context.Background(), records, now)
	require.NoError(t, err)

	// Verify metrics
	// Should have processed 2 valid records (1 per tenant)
	require.Equal(t, float64(1), test.MustGetCounterValue(metricRecordsProcessed.WithLabelValues("tenant1")))
	require.Equal(t, float64(1), test.MustGetCounterValue(metricRecordsProcessed.WithLabelValues("tenant2")))

	// Should have dropped 2 old records (1 per tenant)
	require.Equal(t, float64(1), test.MustGetCounterValue(metricRecordsDropped.WithLabelValues("tenant1", "too_old")))
	require.Equal(t, float64(1), test.MustGetCounterValue(metricRecordsDropped.WithLabelValues("tenant2", "too_old")))
}

type instanceState struct {
	liveTraces     int
	walBlocks      int
	completeBlocks int
}

func requireInstanceState(t *testing.T, inst *instance, state instanceState) {
	require.Equal(t, uint64(state.liveTraces), inst.liveTraces.Len())
	require.Len(t, inst.walBlocks, state.walBlocks)
	require.Len(t, inst.completeBlocks, state.completeBlocks)
}

func requireTraceInLiveStore(t *testing.T, liveStore *LiveStore, traceID []byte, expectedTrace *tempopb.Trace) {
	ctx := user.InjectOrgID(t.Context(), testTenantID)
	resp, err := liveStore.FindTraceByID(ctx, &tempopb.TraceByIDRequest{
		TraceID: traceID,
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Trace)
	require.Equal(t, expectedTrace, resp.Trace)
}

func requireTraceInBlock(t *testing.T, block common.BackendBlock, traceID []byte, expectedTrace *tempopb.Trace) {
	ctx := user.InjectOrgID(t.Context(), testTenantID)
	actualTrace, err := block.FindTraceByID(ctx, traceID, common.DefaultSearchOptions())
	require.NoError(t, err)
	require.NotNil(t, actualTrace)
	require.Equal(t, expectedTrace, actualTrace.Trace)
}

func createValidPushRequest(t *testing.T) []byte {
	id := test.ValidTraceID(nil)
	expectedTrace := test.MakeTrace(5, id)
	traceBytes, err := proto.Marshal(expectedTrace)
	require.NoError(t, err)

	req := &tempopb.PushBytesRequest{
		Traces: []tempopb.PreallocBytes{{Slice: traceBytes}},
		Ids:    [][]byte{id},
	}

	records, err := ingest.Encode(0, testTenantID, req, 1_000_000)
	require.NoError(t, err)

	return records[0].Value
}

func pushToLiveStore(t *testing.T, liveStore *LiveStore) ([]byte, *tempopb.Trace) {
	// create trace
	id := test.ValidTraceID(nil)
	expectedTrace := test.MakeTrace(5, id)
	traceBytes, err := proto.Marshal(expectedTrace)
	require.NoError(t, err)

	// create push bytes request
	request := &tempopb.PushBytesRequest{
		Traces: []tempopb.PreallocBytes{{Slice: traceBytes}},
		Ids:    [][]byte{id},
	}
	requestRecords, err := ingest.Encode(0, testTenantID, request, 1_000_000)
	require.NoError(t, err)

	// set timestamp so they are accepted
	now := time.Now()
	for _, kgoRec := range requestRecords {
		kgoRec.Timestamp = now
	}

	records := make([]record, 0, len(requestRecords))
	for _, kgoRec := range requestRecords {
		records = append(records, fromKGORecord(kgoRec))
	}

	err = liveStore.consume(t.Context(), records, now)
	require.NoError(t, err)

	return id, expectedTrace
}
