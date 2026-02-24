package livestore

import (
	"context"
	"errors"
	"flag"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/kv/consul"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/ingest/testkafka"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/prometheus/client_golang/prometheus"
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

func TestLiveStoreShutdownWithPendingCompletions(t *testing.T) {
	tmpDir := t.TempDir()

	liveStore, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)
	require.NotNil(t, liveStore)

	liveStore.cfg.holdAllBackgroundProcesses = false
	liveStore.startAllBackgroundProcesses()

	inst, err := liveStore.getOrCreateInstance(testTenantID)
	require.NoError(t, err)

	// push data
	expectedID, expectedTrace := pushToLiveStore(t, liveStore)

	// in live traces
	requireTraceInLiveStore(t, liveStore, expectedID, expectedTrace)
	requireInstanceState(t, inst, instanceState{liveTraces: 1, walBlocks: 0, completeBlocks: 0})

	require.NoError(t, liveStore.stopping(nil))
}

func TestLiveStoreQueryMethodsBeforeStarted(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", flag.NewFlagSet("", flag.ContinueOnError))
	cfg.WAL.Filepath = tmpDir
	cfg.WAL.Version = encoding.LatestEncoding().Version()
	cfg.ShutdownMarkerDir = tmpDir

	// Set up test Kafka configuration
	const testTopic = "traces"
	_, kafkaAddr := testkafka.CreateCluster(t, 1, testTopic)

	cfg.IngestConfig.Kafka.Address = kafkaAddr
	cfg.IngestConfig.Kafka.Topic = testTopic
	cfg.IngestConfig.Kafka.ConsumerGroup = "test-consumer-group"

	cfg.holdAllBackgroundProcesses = true

	cfg.Ring.RegisterFlagsAndApplyDefaults("", flag.NewFlagSet("", flag.ContinueOnError))
	mockParititionStore, _ := consul.NewInMemoryClient(
		ring.GetPartitionRingCodec(),
		log.NewNopLogger(),
		nil,
	)
	mockStore, _ := consul.NewInMemoryClient(
		ring.GetCodec(),
		log.NewNopLogger(),
		nil,
	)

	cfg.Ring.KVStore.Mock = mockStore
	cfg.Ring.ListenPort = 0
	cfg.Ring.InstanceAddr = "localhost"
	cfg.Ring.InstanceID = "test-1"
	cfg.PartitionRing.KVStore.Mock = mockParititionStore

	// Create overrides
	limits, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	// Create metrics
	reg := prometheus.NewRegistry()

	logger := test.NewTestingLogger(t)

	// Create LiveStore but DO NOT start it
	liveStore, err := New(cfg, limits, logger, reg, true)
	require.NoError(t, err)
	require.NotNil(t, liveStore)

	ctx := user.InjectOrgID(context.Background(), testTenantID)

	testCases := []struct {
		name     string
		callFunc func() (interface{}, error)
	}{
		{
			name: "SearchRecent",
			callFunc: func() (interface{}, error) {
				return liveStore.SearchRecent(ctx, &tempopb.SearchRequest{
					Query: "{}",
				})
			},
		},
		{
			name: "SearchTags",
			callFunc: func() (interface{}, error) {
				return liveStore.SearchTags(ctx, &tempopb.SearchTagsRequest{
					Scope: "span",
				})
			},
		},
		{
			name: "SearchTagsV2",
			callFunc: func() (interface{}, error) {
				return liveStore.SearchTagsV2(ctx, &tempopb.SearchTagsRequest{
					Scope: "span",
				})
			},
		},
		{
			name: "SearchTagValues",
			callFunc: func() (interface{}, error) {
				return liveStore.SearchTagValues(ctx, &tempopb.SearchTagValuesRequest{
					TagName: "foo",
				})
			},
		},
		{
			name: "SearchTagValuesV2",
			callFunc: func() (interface{}, error) {
				return liveStore.SearchTagValuesV2(ctx, &tempopb.SearchTagValuesRequest{
					TagName: "foo",
				})
			},
		},
		{
			name: "QueryRange",
			callFunc: func() (interface{}, error) {
				return liveStore.QueryRange(ctx, &tempopb.QueryRangeRequest{
					Query: "{} | count_over_time()",
					Start: uint64(time.Now().Add(-time.Hour).UnixNano()),
					End:   uint64(time.Now().UnixNano()),
					Step:  uint64(time.Second),
				})
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call the function before livestore has started
			// This should not panic and should return an error indicating not ready
			resp, err := tc.callFunc()

			// Should return ErrStarting error when not ready
			require.Error(t, err)
			require.ErrorIs(t, err, ErrStarting)
			require.NotNil(t, resp)
		})
	}
}

// erroredEnc is a wrapper around a VersionedEncoding that returns given error on CreateBlock
// if error is not nil. Otherwise, it calls the original CreateBlock method.
type erroredEnc struct {
	encoding.VersionedEncoding
	err error
	mx  sync.Mutex // to make race detection happy
}

func (e *erroredEnc) CreateBlock(ctx context.Context, cfg *common.BlockConfig, meta *backend.BlockMeta, i common.Iterator, r backend.Reader, to backend.Writer) (*backend.BlockMeta, error) {
	e.mx.Lock()
	defer e.mx.Unlock()
	if e.err != nil {
		return nil, e.err
	}
	return e.VersionedEncoding.CreateBlock(ctx, cfg, meta, i, r, to)
}

func (e *erroredEnc) SetError(err error) {
	e.mx.Lock()
	defer e.mx.Unlock()
	e.err = err
}

func TestRequeueOnError(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := defaultConfig(t, tmpDir)
	initialBackoff := 100 * time.Millisecond
	cfg.initialBackoff = initialBackoff
	cfg.maxBackoff = 3 * initialBackoff
	cfg.CompleteBlockConcurrency = 1 // to simplify the test
	cfg.holdAllBackgroundProcesses = false
	liveStore, err := liveStoreWithConfig(t, cfg)
	require.NoError(t, err)
	require.NotNil(t, liveStore)

	inst, err := liveStore.getOrCreateInstance(testTenantID)
	require.NoError(t, err)
	enc := erroredEnc{
		VersionedEncoding: inst.completeBlockEncoding,
		mx:                sync.Mutex{},
	}
	enc.SetError(errors.New("forced error"))
	inst.completeBlockEncoding = &enc

	// push data
	expectedID, expectedTrace := pushToLiveStore(t, liveStore)
	requireTraceInLiveStore(t, liveStore, expectedID, expectedTrace)
	requireInstanceState(t, inst, instanceState{liveTraces: 1, walBlocks: 0, completeBlocks: 0})

	// cut to wal and enqueue complete operation
	liveStore.cutAllInstancesToWal()
	requireInstanceState(t, inst, instanceState{liveTraces: 0, walBlocks: 1, completeBlocks: 0})

	// wait for the first backoff that should not be successful
	time.Sleep(initialBackoff * 2)
	requireInstanceState(t, inst, instanceState{liveTraces: 0, walBlocks: 1, completeBlocks: 0})
	// now completeBlockEncoding does not error and block should be flushed successfully
	enc.SetError(nil)
	time.Sleep(initialBackoff * 8)
	requireInstanceState(t, inst, instanceState{liveTraces: 0, walBlocks: 0, completeBlocks: 1})
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
	require.Equal(t, uint64(state.liveTraces), inst.liveTraces.Len(), "live traces count mismatch")
	require.Len(t, inst.walBlocks, state.walBlocks, "wal blocks count mismatch")
	require.Len(t, inst.completeBlocks, state.completeBlocks, "complete blocks count mismatch")
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

func TestIsLagged(t *testing.T) {
	now := time.Now()
	testCases := []struct {
		name           string
		failOnHighLag  bool
		readerLag      int64
		lastRecordNano int64
		end            time.Time
		expectedLagged bool
		description    string
	}{
		{
			name:           "config disabled - never lagged",
			failOnHighLag:  false,
			readerLag:      50000000,                             // high lag
			lastRecordNano: now.Add(-100 * time.Hour).UnixNano(), // high lag
			end:            now.Add(-1 * time.Second),
			expectedLagged: false,
			description:    "When FailOnHighLag is disabled, isLagged should always return false",
		},
		{
			name:           "lag unknown - should be lagged",
			failOnHighLag:  true,
			readerLag:      -1,                                   // lag unknown
			lastRecordNano: now.Add(-100 * time.Hour).UnixNano(), // high lag
			end:            now,
			expectedLagged: true,
			description:    "When lag is unknown (nil), prefer error over potentially incomplete results",
		},
		{
			name:           "no last record - should be lagged",
			failOnHighLag:  true,
			readerLag:      10, // low lag
			lastRecordNano: -1, // no last record yet
			end:            now,
			expectedLagged: true,
			description:    "When no last record yet, should not be lagged",
		},
		{
			name:           "no lag - recent request - not lagged",
			failOnHighLag:  true,
			readerLag:      100,            // low lag
			lastRecordNano: now.UnixNano(), // no lag
			end:            now.Add(-1 * time.Second),
			expectedLagged: false,
			description:    "When lag is low (near zero), recent requests should not be lagged",
		},
		{
			name:           "high lag - recent request - should be lagged",
			failOnHighLag:  true,
			readerLag:      5000,                                  // high lag
			lastRecordNano: now.Add(-10 * time.Second).UnixNano(), // 10 seconds ago
			end:            now.Add(-5 * time.Second),             // 5 seconds ago
			expectedLagged: true,
			description:    "When lag is high and request is within the lag period, should be lagged",
		},
		{
			name:           "high lag - old request - not lagged",
			failOnHighLag:  true,
			readerLag:      5000,                                  // high lag
			lastRecordNano: now.Add(-10 * time.Second).UnixNano(), // 10 seconds ago
			end:            now.Add(-100 * time.Second),           // 100 seconds ago (well before lag)
			expectedLagged: false,
			description:    "When lag is high but request is old (outside lag period), should not be lagged",
		},
		{
			name:           "high lag - request at boundary",
			failOnHighLag:  true,
			readerLag:      5000,                                  // high lag
			lastRecordNano: now.Add(-10 * time.Second).UnixNano(), // last record was 10s ago
			end:            now.Add(-10 * time.Second),            // request end is 10s ago
			expectedLagged: false,
			description:    "When request end time is equals the calculated lag, should not be lagged",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			ls, err := defaultLiveStore(t, tmpDir)
			require.NoError(t, err)

			ls.cfg.FailOnHighLag = tc.failOnHighLag
			ls.reader.lag.Store(tc.readerLag)
			ls.lastRecordTimeNanos.Store(tc.lastRecordNano)

			t.Run("isLagged", func(t *testing.T) {
				result := ls.isLagged(tc.end.UnixNano())
				require.Equal(t, tc.expectedLagged, result, tc.description)
			})

			t.Run("SearchRecent", func(t *testing.T) {
				ctx := user.InjectOrgID(t.Context(), testTenantID)
				resp, err := ls.SearchRecent(ctx, &tempopb.SearchRequest{
					Query: "{}",
					Start: uint32(now.Add(-5 * time.Hour).Second()),
					End:   uint32(tc.end.Unix()),
				})

				if tc.expectedLagged {
					require.ErrorIs(t, err, errLagged)
					require.Nil(t, resp)
				} else {
					require.NoError(t, err)
					require.NotNil(t, resp)
				}
			})

			t.Run("QueryRange", func(t *testing.T) {
				ctx := user.InjectOrgID(t.Context(), testTenantID)
				resp, err := ls.QueryRange(ctx, &tempopb.QueryRangeRequest{
					Query: "{} | rate()",
					Start: uint64(now.Add(-5 * time.Hour).UnixNano()),
					End:   uint64(tc.end.UnixNano()),
				})
				if tc.expectedLagged {
					require.ErrorIs(t, err, errLagged)
					require.Nil(t, resp)
				} else {
					require.NoError(t, err)
					require.NotNil(t, resp)
				}
			})
		})
	}
}

func TestLiveStoreKeepsPartitionOwnerOnShutdown(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := defaultConfig(t, tmpDir)
	cfg.RemoveOwnerOnShutdown = false
	partitionKV := cfg.PartitionRing.KVStore.Mock

	liveStore, err := liveStoreWithConfig(t, cfg)
	require.NoError(t, err)

	requirePartitionOwnerEventually(t, partitionKV, cfg.Ring.InstanceID, true, "owner should be registered after startup")

	// Stop the live store
	_ = services.StopAndAwaitTerminated(t.Context(), liveStore)

	requirePartitionOwnerEventually(t, partitionKV, cfg.Ring.InstanceID, true, "owner should remain registered after shutdown when RemoveOwnerOnShutdown is false")
}

func TestLiveStoreDownscaleOverridesConfig(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := defaultConfig(t, tmpDir)
	cfg.RemoveOwnerOnShutdown = false
	partitionKV := cfg.PartitionRing.KVStore.Mock

	liveStore, err := liveStoreWithConfig(t, cfg)
	require.NoError(t, err)

	requirePartitionOwnerEventually(t, partitionKV, cfg.Ring.InstanceID, true, "owner should be registered after startup")

	// Simulate downscale API call which explicitly sets remove owner on shutdown
	liveStore.setPrepareShutdown()

	// Stop the live store
	_ = services.StopAndAwaitTerminated(t.Context(), liveStore)

	requirePartitionOwnerEventually(t, partitionKV, cfg.Ring.InstanceID, false, "owner should be removed after shutdown when downscale was triggered")
}

func requirePartitionOwnerEventually(t *testing.T, partitionKV kv.Client, instanceID string, expected bool, msg string) {
	t.Helper()

	require.Eventually(t, func() bool {
		val, err := partitionKV.Get(t.Context(), PartitionRingKey)
		if err != nil {
			return false
		}

		desc := ring.GetOrCreatePartitionRingDesc(val)
		return desc.HasOwner(instanceID) == expected
	}, 5*time.Second, 10*time.Millisecond, msg)
}
