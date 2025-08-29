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
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
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
	records := []*kgo.Record{
		{
			Key:       []byte("tenant1"),
			Timestamp: older, // Too old (older than CompleteBlockTimeout)
			Value:     createValidPushRequest(t),
		},
		{
			Key:       []byte("tenant1"),
			Timestamp: newer, // Valid (newer than CompleteBlockTimeout)
			Value:     createValidPushRequest(t),
		},
		{
			Key:       []byte("tenant2"),
			Timestamp: older, // Too old
			Value:     createValidPushRequest(t),
		},
		{
			Key:       []byte("tenant2"),
			Timestamp: newer, // Valid
			Value:     createValidPushRequest(t),
		},
	}

	// Call consume
	_, err := ls.consume(context.Background(), createRecordIter(records), now)
	require.NoError(t, err)

	// Verify metrics
	// Should have processed 2 valid records (1 per tenant)
	require.Equal(t, float64(1), test.MustGetCounterValue(metricRecordsProcessed.WithLabelValues("tenant1")))
	require.Equal(t, float64(1), test.MustGetCounterValue(metricRecordsProcessed.WithLabelValues("tenant2")))

	// Should have dropped 2 old records (1 per tenant)
	require.Equal(t, float64(1), test.MustGetCounterValue(metricRecordsDropped.WithLabelValues("tenant1", "too_old")))
	require.Equal(t, float64(1), test.MustGetCounterValue(metricRecordsDropped.WithLabelValues("tenant2", "too_old")))

	err = services.StopAndAwaitTerminated(t.Context(), ls)
	require.NoError(t, err)
}

func TestLiveStoreUsesRecordTimestampForBlockStartAndEnd(t *testing.T) {
	// default ingestion slack is 2 minutes. create some convenient times to help the test below
	now := time.Unix(1000000, 0)
	oneMinuteAgo := now.Add(-time.Minute)
	oneMinuteLater := now.Add(time.Minute)
	twoMinutesAgo := now.Add(-2 * time.Minute)
	twoMinutesLater := now.Add(2 * time.Minute)
	threeMinutesAgo := now.Add(-3 * time.Minute)

	tcs := []struct {
		records       []*kgo.Record
		expectedStart time.Time
		expectedEnd   time.Time
	}{
		{ // records where the timestamp exactly matches the span timings
			records: []*kgo.Record{{
				Key:       []byte(testTenantID),
				Timestamp: oneMinuteAgo,
				Value:     createValidPushRequestStartEnd(t, oneMinuteAgo, oneMinuteAgo),
			}, {
				Key:       []byte(testTenantID),
				Timestamp: now,
				Value:     createValidPushRequestStartEnd(t, now, now),
			}},
			expectedStart: oneMinuteAgo,
			expectedEnd:   now,
		},
		{ // records where the timestamp doesn't match the span timings, but within the ingestion slack
			records: []*kgo.Record{{
				Key:       []byte(testTenantID),
				Timestamp: now,
				Value:     createValidPushRequestStartEnd(t, oneMinuteAgo, oneMinuteLater),
			}, {
				Key:       []byte(testTenantID),
				Timestamp: now,
				Value:     createValidPushRequestStartEnd(t, twoMinutesAgo, twoMinutesLater),
			}},
			expectedStart: twoMinutesAgo,
			expectedEnd:   twoMinutesLater,
		},
		{ // records where the timestamp doesn't match the span timings and is outside the ingestion slack
			records: []*kgo.Record{{
				Key:       []byte(testTenantID),
				Timestamp: now,
				Value:     createValidPushRequestStartEnd(t, threeMinutesAgo, now),
			}, {
				Key:       []byte(testTenantID),
				Timestamp: now,
				Value:     createValidPushRequestStartEnd(t, threeMinutesAgo, oneMinuteLater),
			}},
			expectedStart: twoMinutesAgo, // default ingestion slack is 2 minutes
			expectedEnd:   oneMinuteLater,
		},
	}

	for _, tc := range tcs {
		ls, err := defaultLiveStore(t, t.TempDir())
		require.NoError(t, err)

		_, err = ls.consume(t.Context(), createRecordIter(tc.records), now)
		require.NoError(t, err)

		inst, err := ls.getOrCreateInstance(testTenantID)
		require.NoError(t, err)

		// force just pushed traces to the head block
		err = inst.cutIdleTraces(true)
		require.NoError(t, err)

		meta := inst.headBlock.BlockMeta()
		require.Equal(t, tc.expectedStart, meta.StartTime)
		require.Equal(t, tc.expectedEnd, meta.EndTime)

		// cut to complete block and test again
		uuid, err := inst.cutBlocks(true)
		require.NoError(t, err)
		err = inst.completeBlock(t.Context(), uuid)
		require.NoError(t, err)

		meta = inst.completeBlocks[uuid].BlockMeta()
		require.Equal(t, tc.expectedStart, meta.StartTime)
		require.Equal(t, tc.expectedEnd, meta.EndTime)

		err = services.StopAndAwaitTerminated(t.Context(), ls)
		require.NoError(t, err)
	}
}

type instanceState struct {
	liveTraces     int
	walBlocks      int
	completeBlocks int
}

// testRecordIter is a simple recordIter implementation for tests
type testRecordIter struct {
	records []*kgo.Record
	index   int
}

func (t *testRecordIter) Next() *kgo.Record {
	if t.index >= len(t.records) {
		return nil
	}
	record := t.records[t.index]
	t.index++
	return record
}

func (t *testRecordIter) Done() bool {
	return t.index >= len(t.records)
}

// createRecordIter creates a recordIter from a slice of *kgo.Record for testing
func createRecordIter(records []*kgo.Record) recordIter {
	return &testRecordIter{
		records: records,
		index:   0,
	}
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

func createValidPushRequestStartEnd(t *testing.T, start, end time.Time) []byte {
	id := test.ValidTraceID(nil)
	tr := &tempopb.Trace{
		ResourceSpans: []*v1_trace.ResourceSpans{
			{
				Resource: &v1_resource.Resource{},
				ScopeSpans: []*v1_trace.ScopeSpans{
					{
						Spans: []*v1_trace.Span{
							{
								TraceId:           id,
								StartTimeUnixNano: uint64(start.UnixNano()),
								EndTimeUnixNano:   uint64(end.UnixNano()),
							},
						},
					},
				},
			},
		},
	}
	traceBytes, err := proto.Marshal(tr)
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

	_, err = liveStore.consume(t.Context(), createRecordIter(requestRecords), now)
	require.NoError(t, err)

	return id, expectedTrace
}
