package ingester

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/dskit/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	tempo_util "github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
)

const testTenantID = "fake"

type ringCountMock struct {
	count int
}

func (m *ringCountMock) HealthyInstancesCount() int {
	return m.count
}

func TestInstance(t *testing.T) {
	request := makeRequest([]byte{})

	i, ingester := defaultInstance(t)

	response := i.PushBytesRequest(context.Background(), request)
	require.NotNil(t, response)
	require.Equal(t, int(i.traceCount.Load()), len(i.traces))

	err := i.CutCompleteTraces(0, true)
	require.NoError(t, err)
	require.Equal(t, int(i.traceCount.Load()), len(i.traces))

	blockID, err := i.CutBlockIfReady(0, 0, false)
	require.NoError(t, err, "unexpected error cutting block")
	require.NotEqual(t, blockID, uuid.Nil)

	err = i.CompleteBlock(blockID)
	require.NoError(t, err, "unexpected error completing block")

	block := i.GetBlockToBeFlushed(blockID)
	require.NotNil(t, block)
	require.Len(t, i.completingBlocks, 1)
	require.Len(t, i.completeBlocks, 1)

	err = ingester.store.WriteBlock(context.Background(), block)
	require.NoError(t, err)

	err = i.ClearFlushedBlocks(30 * time.Hour)
	require.NoError(t, err)
	require.Len(t, i.completeBlocks, 1)

	err = i.ClearFlushedBlocks(0)
	require.NoError(t, err)
	require.Len(t, i.completeBlocks, 0)

	err = i.resetHeadBlock()
	require.NoError(t, err, "unexpected error resetting block")

	require.Equal(t, int(i.traceCount.Load()), len(i.traces))
}

func TestInstanceFind(t *testing.T) {
	i, ingester := defaultInstance(t)

	numTraces := 10
	traces, ids := pushTracesToInstance(t, i, numTraces)

	queryAll(t, i, ids, traces)

	err := i.CutCompleteTraces(0, true)
	require.NoError(t, err)
	require.Equal(t, int(i.traceCount.Load()), len(i.traces))

	for j := 0; j < numTraces; j++ {
		traceBytes, err := model.MustNewSegmentDecoder(model.CurrentEncoding).PrepareForWrite(traces[j], 0, 0)
		require.NoError(t, err)

		err = i.PushBytes(context.Background(), ids[j], traceBytes)
		require.NoError(t, err)
	}

	queryAll(t, i, ids, traces)

	blockID, err := i.CutBlockIfReady(0, 0, true)
	require.NoError(t, err)
	require.NotEqual(t, blockID, uuid.Nil)

	queryAll(t, i, ids, traces)

	err = i.CompleteBlock(blockID)
	require.NoError(t, err)

	queryAll(t, i, ids, traces)

	err = i.ClearCompletingBlock(blockID)
	require.NoError(t, err)

	queryAll(t, i, ids, traces)

	localBlock := i.GetBlockToBeFlushed(blockID)
	require.NotNil(t, localBlock)

	err = ingester.store.WriteBlock(context.Background(), localBlock)
	require.NoError(t, err)

	queryAll(t, i, ids, traces)
}

// pushTracesToInstance makes and pushes numTraces in the ingester instance,
// returns traces and trace ids
func pushTracesToInstance(t *testing.T, i *instance, numTraces int) ([]*tempopb.Trace, [][]byte) {
	var ids [][]byte
	var traces []*tempopb.Trace

	for j := 0; j < numTraces; j++ {
		id := make([]byte, 16)
		_, err := crand.Read(id)
		require.NoError(t, err)

		testTrace := test.MakeTrace(10, id)
		trace.SortTrace(testTrace)
		traceBytes, err := model.MustNewSegmentDecoder(model.CurrentEncoding).PrepareForWrite(testTrace, 0, 0)
		require.NoError(t, err)

		err = i.PushBytes(context.Background(), id, traceBytes)
		require.NoError(t, err)
		require.Equal(t, int(i.traceCount.Load()), len(i.traces))

		ids = append(ids, id)
		traces = append(traces, testTrace)
	}
	return traces, ids
}

func queryAll(t *testing.T, i *instance, ids [][]byte, traces []*tempopb.Trace) {
	for j, id := range ids {
		trace, err := i.FindTraceByID(context.Background(), id)
		require.NoError(t, err)
		require.Equal(t, traces[j], trace)
	}
}

func TestInstanceDoesNotRace(t *testing.T) {
	i, ingester := defaultInstance(t)
	end := make(chan struct{})

	concurrent := func(f func()) {
		for {
			select {
			case <-end:
				return
			default:
				f()
			}
		}
	}
	go concurrent(func() {
		request := makeRequest([]byte{})
		response := i.PushBytesRequest(context.Background(), request)
		errored, _, _ := CheckPushBytesError(response)
		require.False(t, errored)
	})

	go concurrent(func() {
		err := i.CutCompleteTraces(0, true)
		require.NoError(t, err, "error cutting complete traces")
	})

	go concurrent(func() {
		blockID, _ := i.CutBlockIfReady(0, 0, false)
		if blockID != uuid.Nil {
			err := i.CompleteBlock(blockID)
			require.NoError(t, err, "unexpected error completing block")
			block := i.GetBlockToBeFlushed(blockID)
			require.NotNil(t, block)
			err = ingester.store.WriteBlock(context.Background(), block)
			require.NoError(t, err, "error writing block")
		}
	})

	go concurrent(func() {
		err := i.ClearFlushedBlocks(0)
		require.NoError(t, err, "error clearing flushed blocks")
	})

	go concurrent(func() {
		_, err := i.FindTraceByID(context.Background(), []byte{0x01})
		require.NoError(t, err, "error finding trace by id")
	})

	time.Sleep(100 * time.Millisecond)
	close(end)
	// Wait for go funcs to quit before
	// exiting and cleaning up
	time.Sleep(2 * time.Second)
}

func TestInstanceLimits(t *testing.T) {
	limits, err := overrides.NewOverrides(overrides.Config{
		Defaults: overrides.Overrides{
			Global: overrides.GlobalOverrides{
				MaxBytesPerTrace: 1000,
			},
			Ingestion: overrides.IngestionOverrides{
				MaxLocalTracesPerUser: 4,
			},
		},
	})
	require.NoError(t, err, "unexpected error creating limits")
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	ingester, _, _ := defaultIngester(t, t.TempDir())
	ingester.limiter = limiter

	type push struct {
		req          *tempopb.PushBytesRequest
		expectsError bool
		errorReason  string
	}

	traceTooLarge := "trace_too_large"
	maxLiveTraces := "max_live_traces"

	tests := []struct {
		name   string
		pushes []push
	}{
		{
			name: "bytes - succeeds",
			pushes: []push{
				{
					req: makeRequestWithByteLimit(300, []byte{}),
				},
				{
					req: makeRequestWithByteLimit(500, []byte{}),
				},
				{
					req: makeRequestWithByteLimit(100, []byte{}),
				},
			},
		},
		{
			name: "bytes - one fails",
			pushes: []push{
				{
					req: makeRequestWithByteLimit(300, []byte{}),
				},
				{
					req:          makeRequestWithByteLimit(1500, []byte{}),
					expectsError: true,
					errorReason:  traceTooLarge,
				},
				{
					req: makeRequestWithByteLimit(600, []byte{}),
				},
			},
		},
		{
			name: "bytes - multiple pushes same trace",
			pushes: []push{
				{
					req: makeRequestWithByteLimit(500, []byte{0x01}),
				},
				{
					req:          makeRequestWithByteLimit(700, []byte{0x01}),
					expectsError: true,
					errorReason:  traceTooLarge,
				},
			},
		},
		{
			name: "max traces - too many",
			pushes: []push{
				{
					req: makeRequestWithByteLimit(100, []byte{}),
				},
				{
					req: makeRequestWithByteLimit(100, []byte{}),
				},
				{
					req: makeRequestWithByteLimit(100, []byte{}),
				},
				{
					req: makeRequestWithByteLimit(100, []byte{}),
				},
				{
					req:          makeRequestWithByteLimit(100, []byte{}),
					expectsError: true,
					errorReason:  maxLiveTraces,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delete(ingester.instances, testTenantID) // force recreate instance to reset limits
			i, err := ingester.getOrCreateInstance(testTenantID)
			require.NoError(t, err, "unexpected error creating new instance")

			for j, push := range tt.pushes {
				response := i.PushBytesRequest(context.Background(), push.req)
				if push.expectsError && push.errorReason == traceTooLarge {
					errored, maxLiveCount, traceTooLargeCount := CheckPushBytesError(response)
					require.True(t, errored)
					require.Zero(t, maxLiveCount, "push %d failed: %w", j, err)
					require.NotZero(t, traceTooLargeCount, "push %d failed: %w", j, err)
				} else if push.expectsError && push.errorReason == maxLiveTraces {
					errored, maxLiveCount, traceTooLargeCount := CheckPushBytesError(response)
					require.True(t, errored)
					require.NotZero(t, maxLiveCount, "push %d failed: %w", j, err)
					require.Zero(t, traceTooLargeCount, "push %d failed: %w", j, err)
				} else {
					errored, _, _ := CheckPushBytesError(response)
					require.False(t, errored, "push %d failed: %w", j, err)
				}

			}
		})
	}
}

func TestInstanceCutCompleteTraces(t *testing.T) {
	id := make([]byte, 16)
	_, err := crand.Read(id)
	require.NoError(t, err)

	pastTrace := &liveTrace{
		traceID:    id,
		lastAppend: time.Now().Add(-time.Hour),
	}

	id = make([]byte, 16)
	_, err = crand.Read(id)
	require.NoError(t, err)

	nowTrace := &liveTrace{
		traceID:    id,
		lastAppend: time.Now().Add(time.Hour),
	}

	tt := []struct {
		name             string
		cutoff           time.Duration
		immediate        bool
		input            []*liveTrace
		expectedExist    []*liveTrace
		expectedNotExist []*liveTrace
	}{
		{
			name:      "empty",
			cutoff:    0,
			immediate: false,
		},
		{
			name:             "cut immediate",
			cutoff:           0,
			immediate:        true,
			input:            []*liveTrace{pastTrace, nowTrace},
			expectedNotExist: []*liveTrace{pastTrace, nowTrace},
		},
		{
			name:             "cut recent",
			cutoff:           0,
			immediate:        false,
			input:            []*liveTrace{pastTrace, nowTrace},
			expectedExist:    []*liveTrace{nowTrace},
			expectedNotExist: []*liveTrace{pastTrace},
		},
		{
			name:             "cut all time",
			cutoff:           2 * time.Hour,
			immediate:        false,
			input:            []*liveTrace{pastTrace, nowTrace},
			expectedNotExist: []*liveTrace{pastTrace, nowTrace},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			instance, _ := defaultInstance(t)

			for _, trace := range tc.input {
				fp := instance.tokenForTraceID(trace.traceID)
				instance.traces[fp] = trace
			}

			err := instance.CutCompleteTraces(tc.cutoff, tc.immediate)
			require.NoError(t, err)

			require.Equal(t, len(tc.expectedExist), len(instance.traces))
			for _, expectedExist := range tc.expectedExist {
				_, ok := instance.traces[instance.tokenForTraceID(expectedExist.traceID)]
				require.True(t, ok)
			}

			for _, expectedNotExist := range tc.expectedNotExist {
				_, ok := instance.traces[instance.tokenForTraceID(expectedNotExist.traceID)]
				require.False(t, ok)
			}
		})
	}
}

func TestInstanceCutBlockIfReady(t *testing.T) {
	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	tt := []struct {
		name               string
		maxBlockLifetime   time.Duration
		maxBlockBytes      uint64
		immediate          bool
		pushCount          int
		expectedToCutBlock bool
	}{
		{
			name:               "empty",
			expectedToCutBlock: false,
		},
		{
			name:               "doesnt cut anything",
			pushCount:          1,
			expectedToCutBlock: false,
		},
		{
			name:               "cut immediate",
			immediate:          true,
			pushCount:          1,
			expectedToCutBlock: true,
		},
		{
			name:               "cut based on block lifetime",
			maxBlockLifetime:   time.Microsecond,
			pushCount:          1,
			expectedToCutBlock: true,
		},
		{
			name:               "cut based on block size",
			maxBlockBytes:      10,
			pushCount:          10,
			expectedToCutBlock: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			instance, _ := defaultInstance(t)

			for i := 0; i < tc.pushCount; i++ {
				tr := test.MakeTrace(1, uuid.Nil[:])
				bytes, err := dec.PrepareForWrite(tr, 0, 0)
				require.NoError(t, err)
				err = instance.PushBytes(context.Background(), uuid.Nil[:], bytes)
				require.NoError(t, err)
			}

			// Defaults
			if tc.maxBlockBytes == 0 {
				tc.maxBlockBytes = 100000
			}
			if tc.maxBlockLifetime == 0 {
				tc.maxBlockLifetime = time.Hour
			}

			lastCutTime := instance.lastBlockCut

			// Cut all traces to headblock for testing
			err := instance.CutCompleteTraces(0, true)
			require.NoError(t, err)

			blockID, err := instance.CutBlockIfReady(tc.maxBlockLifetime, tc.maxBlockBytes, tc.immediate)
			require.NoError(t, err)

			err = instance.CompleteBlock(blockID)
			if tc.expectedToCutBlock {
				require.NoError(t, err, "unexpected error completing block")
			}

			// Wait for goroutine to finish flushing to avoid test flakiness
			if tc.expectedToCutBlock {
				time.Sleep(time.Millisecond * 250)
			}

			require.Equal(t, tc.expectedToCutBlock, instance.lastBlockCut.After(lastCutTime))
		})
	}
}

func TestInstanceMetrics(t *testing.T) {
	i, _ := defaultInstance(t)
	cutAndVerify := func(v int) {
		err := i.CutCompleteTraces(0, true)
		require.NoError(t, err)

		liveTraces, err := test.GetGaugeVecValue(metricLiveTraces, testTenantID)
		require.NoError(t, err)
		require.Equal(t, v, int(liveTraces))
	}

	cutAndVerify(0)

	// Push some traces
	count := 100
	for j := 0; j < count; j++ {
		request := makeRequest([]byte{})
		response := i.PushBytesRequest(context.Background(), request)
		errored, _, _ := CheckPushBytesError(response)
		require.False(t, errored, "push %d failed: %w", j, response.ErrorsByTrace)
	}
	cutAndVerify(count)
	cutAndVerify(0)
}

func TestInstanceFailsLargeTracesEvenAfterFlushing(t *testing.T) {
	ctx := context.Background()
	maxTraceBytes := 1000
	id := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

	limits, err := overrides.NewOverrides(overrides.Config{
		Defaults: overrides.Overrides{
			Global: overrides.GlobalOverrides{
				MaxBytesPerTrace: maxTraceBytes,
			},
		},
	})
	require.NoError(t, err)
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	ingester, _, _ := defaultIngester(t, t.TempDir())
	ingester.limiter = limiter
	i, err := ingester.getOrCreateInstance(testTenantID)
	require.NoError(t, err)

	req := makeRequestWithByteLimit(maxTraceBytes-300, id)
	reqSize := 0
	for _, b := range req.Traces {
		reqSize += len(b.Slice)
	}

	// Fill up trace to max
	response := i.PushBytesRequest(ctx, req)
	errored, _, _ := CheckPushBytesError(response)
	require.False(t, errored, "push failed: %w", response.ErrorsByTrace)

	// Pushing again fails
	response = i.PushBytesRequest(ctx, req)
	_, _, traceTooLargeCount := CheckPushBytesError(response)
	assert.Equal(t, true, traceTooLargeCount > 0)

	// Pushing still fails after flush
	err = i.CutCompleteTraces(0, true)
	require.NoError(t, err)
	response = i.PushBytesRequest(ctx, req)
	_, _, traceTooLargeCount = CheckPushBytesError(response)
	assert.Equal(t, true, traceTooLargeCount > 0)

	// Cut block and then pushing works again
	_, err = i.CutBlockIfReady(0, 0, true)
	require.NoError(t, err)
	response = i.PushBytesRequest(ctx, req)
	errored, _, _ = CheckPushBytesError(response)
	require.False(t, errored, "push failed: %w", response.ErrorsByTrace)
}

func TestInstancePartialSuccess(t *testing.T) {
	ctx := context.Background()
	maxTraceBytes := 1000

	limits, err := overrides.NewOverrides(overrides.Config{
		Defaults: overrides.Overrides{
			Global: overrides.GlobalOverrides{
				MaxBytesPerTrace: maxTraceBytes,
			},
			Ingestion: overrides.IngestionOverrides{
				MaxLocalTracesPerUser: 2,
			},
		},
	})
	require.NoError(t, err)
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	ingester, _, _ := defaultIngester(t, t.TempDir())
	ingester.limiter = limiter

	delete(ingester.instances, testTenantID) // force recreate instance to reset limits
	i, err := ingester.getOrCreateInstance(testTenantID)
	require.NoError(t, err, "unexpected error creating new instance")

	count := 5
	ids := make([][]byte, 0, count)
	for j := 0; j < count; j++ {
		id := make([]byte, 16)
		_, err := crand.Read(id)
		require.NoError(t, err)
		ids = append(ids, id)
	}

	// one with no error [0], two with trace_too_large [1,2], one with no error [3], one should trigger live_traces_exceeded [4]
	multiMaxBytes := []int{maxTraceBytes - 300, maxTraceBytes + 200, maxTraceBytes + 200, maxTraceBytes - 300, maxTraceBytes - 200}
	req := makePushBytesRequestMultiTraces(ids, multiMaxBytes)

	// Pushing pass
	// response should contain errors for both LIVE_TRACES_EXCEEDED and TRACE_TOO_LARGE
	response := i.PushBytesRequest(ctx, req)
	errored, maxLiveCount, traceTooLargeCount := CheckPushBytesError(response)

	assert.True(t, errored)
	assert.Equal(t, true, maxLiveCount > 0)
	assert.Equal(t, true, traceTooLargeCount > 0)

	// check that the two good ones actually made it
	result, err := i.FindTraceByID(ctx, ids[0])
	require.NoError(t, err, "error finding trace by id")
	assert.Equal(t, 1, len(result.Batches))

	result, err = i.FindTraceByID(ctx, ids[3])
	require.NoError(t, err, "error finding trace by id")
	assert.Equal(t, 1, len(result.Batches))

	// check that the three traces that had errors did not actually make it
	var expected *tempopb.Trace
	result, err = i.FindTraceByID(ctx, ids[1])
	require.NoError(t, err, "error finding trace by id")
	assert.Equal(t, expected, result)

	result, err = i.FindTraceByID(ctx, ids[2])
	require.NoError(t, err, "error finding trace by id")
	assert.Equal(t, expected, result)

	result, err = i.FindTraceByID(ctx, ids[4])
	require.NoError(t, err, "error finding trace by id")
	assert.Equal(t, expected, result)
}

func TestSortByteSlices(t *testing.T) {
	numTraces := 100

	// create first trace
	traceBytes := &tempopb.TraceBytes{
		Traces: make([][]byte, numTraces),
	}
	for i := range traceBytes.Traces {
		traceBytes.Traces[i] = make([]byte, rand.Intn(10))
		_, err := crand.Read(traceBytes.Traces[i])
		require.NoError(t, err)
	}

	// dupe
	traceBytes2 := &tempopb.TraceBytes{
		Traces: make([][]byte, numTraces),
	}
	for i := range traceBytes.Traces {
		traceBytes2.Traces[i] = make([]byte, len(traceBytes.Traces[i]))
		copy(traceBytes2.Traces[i], traceBytes.Traces[i])
	}

	// randomize dupe
	rand.Shuffle(len(traceBytes2.Traces), func(i, j int) {
		traceBytes2.Traces[i], traceBytes2.Traces[j] = traceBytes2.Traces[j], traceBytes2.Traces[i]
	})

	assert.NotEqual(t, traceBytes, traceBytes2)

	// sort and compare
	sortByteSlices(traceBytes.Traces)
	sortByteSlices(traceBytes2.Traces)

	assert.Equal(t, traceBytes, traceBytes2)
}

func defaultInstance(t testing.TB) (*instance, *Ingester) {
	instance, ingester, _ := defaultInstanceAndTmpDir(t)
	return instance, ingester
}

func defaultInstanceAndTmpDir(t testing.TB) (*instance, *Ingester, string) {
	tmpDir := t.TempDir()

	ingester, _, _ := defaultIngester(t, tmpDir)
	instance, err := ingester.getOrCreateInstance(testTenantID)
	require.NoError(t, err, "unexpected error creating new instance")

	return instance, ingester, tmpDir
}

func BenchmarkInstancePush(b *testing.B) {
	instance, _ := defaultInstance(b)
	request := makeRequest([]byte{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Rotate trace ID
		binary.LittleEndian.PutUint32(request.Ids[0].Slice, uint32(i))
		response := instance.PushBytesRequest(context.Background(), request)
		errored, _, _ := CheckPushBytesError(response)
		require.False(b, errored, "push failed: %w", response.ErrorsByTrace)
	}
}

func BenchmarkInstancePushExistingTrace(b *testing.B) {
	instance, _ := defaultInstance(b)
	request := makeRequest([]byte{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		response := instance.PushBytesRequest(context.Background(), request)
		errored, _, _ := CheckPushBytesError(response)
		require.False(b, errored, "push failed: %w", response.ErrorsByTrace)
	}
}

func BenchmarkInstanceFindTraceByIDFromCompleteBlock(b *testing.B) {
	instance, _ := defaultInstance(b)
	traceID := test.ValidTraceID([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	request := makeRequest(traceID)
	response := instance.PushBytesRequest(context.Background(), request)
	errored, _, _ := CheckPushBytesError(response)
	require.False(b, errored, "push failed: %w", response.ErrorsByTrace)

	// force the trace to be in a complete block
	err := instance.CutCompleteTraces(0, true)
	require.NoError(b, err)
	id, err := instance.CutBlockIfReady(0, 0, true)
	require.NoError(b, err)
	err = instance.CompleteBlock(id)
	require.NoError(b, err)

	require.Equal(b, 1, len(instance.completeBlocks))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trace, err := instance.FindTraceByID(context.Background(), traceID)
		require.NotNil(b, trace)
		require.NoError(b, err)
	}
}

func BenchmarkInstanceSearchCompleteParquet(b *testing.B) {
	benchmarkInstanceSearch(b)
}

func TestInstanceSearchCompleteParquet(t *testing.T) {
	benchmarkInstanceSearch(t)
}

func benchmarkInstanceSearch(b testing.TB) {
	instance, _ := defaultInstance(b)
	for i := 0; i < 1000; i++ {
		request := makeRequest(nil)
		response := instance.PushBytesRequest(context.Background(), request)
		errored, _, _ := CheckPushBytesError(response)
		require.False(b, errored, "push failed: %w", response.ErrorsByTrace)

		if i%100 == 0 {
			err := instance.CutCompleteTraces(0, true)
			require.NoError(b, err)
		}
	}

	// force the traces to be in a complete block
	id, err := instance.CutBlockIfReady(0, 0, true)
	require.NoError(b, err)
	err = instance.CompleteBlock(id)
	require.NoError(b, err)

	require.Equal(b, 1, len(instance.completeBlocks))

	ctx := context.Background()
	ctx = user.InjectOrgID(ctx, testTenantID)

	if rt, ok := b.(*testing.B); ok {
		rt.ResetTimer()
		for i := 0; i < rt.N; i++ {
			resp, err := instance.SearchTags(ctx, "")
			require.NoError(b, err)
			require.NotNil(b, resp)
		}
		return
	}

	for i := 0; i < 100; i++ {
		resp, err := instance.SearchTags(ctx, "")
		require.NoError(b, err)
		require.NotNil(b, resp)
	}
}

func makeRequest(traceID []byte) *tempopb.PushBytesRequest {
	const spans = 10

	traceID = test.ValidTraceID(traceID)
	return makePushBytesRequest(traceID, test.MakeBatch(spans, traceID))
}

// Note that this fn will generate a request with size **close to** maxBytes
func makeRequestWithByteLimit(maxBytes int, traceID []byte) *tempopb.PushBytesRequest {
	traceID = test.ValidTraceID(traceID)
	batch := makeBatchWithMaxBytes(maxBytes, traceID)

	return makePushBytesRequest(traceID, batch)
}

func makePushBytesRequest(traceID []byte, batch *v1_trace.ResourceSpans) *tempopb.PushBytesRequest {
	trace := &tempopb.Trace{Batches: []*v1_trace.ResourceSpans{batch}}

	buffer, err := model.MustNewSegmentDecoder(model.CurrentEncoding).PrepareForWrite(trace, 0, 0)
	if err != nil {
		panic(err)
	}

	return &tempopb.PushBytesRequest{
		Ids: []tempopb.PreallocBytes{{
			Slice: traceID,
		}},
		Traces: []tempopb.PreallocBytes{{
			Slice: buffer,
		}},
	}
}

func BenchmarkInstanceContention(t *testing.B) {
	var (
		ctx          = user.InjectOrgID(context.Background(), testTenantID)
		end          = make(chan struct{})
		wg           = sync.WaitGroup{}
		pushes       = 0
		traceFlushes = 0
		blockFlushes = 0
		retentions   = 0
		finds        = 0
		searches     = 0
		searchBytes  = 0
		searchTags   = 0
	)

	i, ingester := defaultInstance(t)

	concurrent := func(f func()) {
		wg.Add(1)
		defer wg.Done()
		for {
			select {
			case <-end:
				return
			default:
				f()
			}
		}
	}
	go concurrent(func() {
		request := makeRequestWithByteLimit(10_000, nil)
		response := i.PushBytesRequest(ctx, request)
		errored, _, _ := CheckPushBytesError(response)
		require.False(t, errored, "push failed: %w", response.ErrorsByTrace)
		pushes++
	})

	go concurrent(func() {
		err := i.CutCompleteTraces(0, true)
		require.NoError(t, err, "error cutting complete traces")
		traceFlushes++
	})

	go concurrent(func() {
		blockID, _ := i.CutBlockIfReady(0, 0, false)
		if blockID != uuid.Nil {
			err := i.CompleteBlock(blockID)
			require.NoError(t, err, "unexpected error completing block")
			err = i.ClearCompletingBlock(blockID)
			require.NoError(t, err, "unexpected error clearing wal block")
			block := i.GetBlockToBeFlushed(blockID)
			require.NotNil(t, block)
			err = ingester.store.WriteBlock(ctx, block)
			require.NoError(t, err, "error writing block")
		}
		blockFlushes++
	})

	go concurrent(func() {
		err := i.ClearFlushedBlocks(0)
		require.NoError(t, err, "error clearing flushed blocks")
		retentions++
	})

	go concurrent(func() {
		_, err := i.FindTraceByID(ctx, []byte{0x01})
		require.NoError(t, err, "error finding trace by id")
		finds++
	})

	go concurrent(func() {
		x, err := i.Search(ctx, &tempopb.SearchRequest{
			Query: "{ .foo=`bar` }",
		})
		require.NoError(t, err, "error searching traceql")
		searchBytes += int(x.Metrics.InspectedBytes)
		searches++
	})

	go concurrent(func() {
		_, err := i.SearchTags(ctx, "")
		require.NoError(t, err, "error searching tags")
		searchTags++
	})

	time.Sleep(2 * time.Second)
	close(end)
	wg.Wait()

	report := func(n int, u string) {
		t.ReportMetric(float64(n)/t.Elapsed().Seconds(), u+"/sec")
	}

	report(pushes, "pushes")
	report(traceFlushes, "traceflushes")
	report(blockFlushes, "blockflushes")
	report(retentions, "retentions")
	report(finds, "finds")
	report(searches, "searches")
	report(searchBytes, "searchedBytes")
	report(searchTags, "searchTags")
}

func makeBatchWithMaxBytes(maxBytes int, traceID []byte) *v1_trace.ResourceSpans {
	traceID = test.ValidTraceID(traceID)
	batch := test.MakeBatch(1, traceID)

	for batch.Size() < maxBytes {
		batch.ScopeSpans[0].Spans = append(batch.ScopeSpans[0].Spans, test.MakeSpanWithAttributeCount(traceID, 0))
	}

	return batch
}

func makeTraces(batches []*v1_trace.ResourceSpans) []*tempopb.Trace {
	traces := make([]*tempopb.Trace, 0, len(batches))
	for _, batch := range batches {
		traces = append(traces, &tempopb.Trace{Batches: []*v1_trace.ResourceSpans{batch}})
	}

	return traces
}

func makePushBytesRequestMultiTraces(traceIDs [][]byte, maxBytes []int) *tempopb.PushBytesRequest {
	batches := make([]*v1_trace.ResourceSpans, 0, len(traceIDs))
	for index, id := range traceIDs {
		batch := makeBatchWithMaxBytes(maxBytes[index], id)
		batches = append(batches, batch)
	}
	traces := makeTraces(batches)

	byteIDs := make([]tempopb.PreallocBytes, 0, len(traceIDs))
	byteTraces := make([]tempopb.PreallocBytes, 0, len(traceIDs))
	byteSpanCounts := make([]tempopb.PreallocBytes, 0, len(traceIDs))

	for index, id := range traceIDs {
		buffer, err := model.MustNewSegmentDecoder(model.CurrentEncoding).PrepareForWrite(traces[index], 0, 0)
		if err != nil {
			panic(err)
		}

		spanCount, _ := tempo_util.HexStringToTraceID("1")

		byteIDs = append(byteIDs, tempopb.PreallocBytes{
			Slice: id,
		})
		byteTraces = append(byteTraces, tempopb.PreallocBytes{
			Slice: buffer,
		})
		byteSpanCounts = append(byteSpanCounts, tempopb.PreallocBytes{
			Slice: spanCount,
		})
	}

	return &tempopb.PushBytesRequest{
		Ids:    byteIDs,
		Traces: byteTraces,
	}
}

func CheckPushBytesError(response *tempopb.PushResponse) (errored bool, maxLiveTracesCount int, traceTooLargeCount int) {
	for _, result := range response.ErrorsByTrace {
		switch result {
		case tempopb.PushErrorReason_MAX_LIVE_TRACES:
			maxLiveTracesCount++
		case tempopb.PushErrorReason_TRACE_TOO_LARGE:
			traceTooLargeCount++
		}
	}

	if (maxLiveTracesCount + traceTooLargeCount) > 0 {
		errored = true
	}

	return errored, maxLiveTracesCount, traceTooLargeCount
}
