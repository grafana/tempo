package ingester

import (
	"context"
	"encoding/binary"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
)

const testTenantID = "fake"

type ringCountMock struct {
	count int
}

func (m *ringCountMock) HealthyInstancesCount() int {
	return m.count
}

func TestInstance(t *testing.T) {
	limits, err := overrides.NewOverrides(overrides.Limits{})
	require.NoError(t, err, "unexpected error creating limits")
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	tempDir, err := os.MkdirTemp("/tmp", "")
	require.NoError(t, err, "unexpected error getting temp dir")
	defer os.RemoveAll(tempDir)

	ingester, _, _ := defaultIngester(t, tempDir)
	request := test.MakeRequest(10, []byte{})

	i, err := newInstance(testTenantID, limiter, ingester.store, ingester.local)
	require.NoError(t, err, "unexpected error creating new instance")
	err = i.PushBytesRequest(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, int(i.traceCount.Load()), len(i.traces))

	err = i.CutCompleteTraces(0, true)
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
	limits, err := overrides.NewOverrides(overrides.Limits{})
	require.NoError(t, err, "unexpected error creating limits")
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	tempDir, err := os.MkdirTemp("/tmp", "")
	require.NoError(t, err, "unexpected error getting temp dir")
	defer os.RemoveAll(tempDir)

	ingester, _, _ := defaultIngester(t, tempDir)
	i, err := newInstance(testTenantID, limiter, ingester.store, ingester.local)
	require.NoError(t, err, "unexpected error creating new instance")

	numTraces := 500
	ids := [][]byte{}
	traces := []*tempopb.Trace{}
	for j := 0; j < numTraces; j++ {
		id := make([]byte, 16)
		rand.Read(id)

		testTrace := test.MakeTrace(10, id)
		trace.SortTrace(testTrace)
		traceBytes, err := testTrace.Marshal()
		require.NoError(t, err)

		err = i.PushBytes(context.Background(), id, traceBytes, nil)
		require.NoError(t, err)
		require.Equal(t, int(i.traceCount.Load()), len(i.traces))

		ids = append(ids, id)
		traces = append(traces, testTrace)
	}

	queryAll(t, i, ids, traces)

	err = i.CutCompleteTraces(0, true)
	require.NoError(t, err)
	require.Equal(t, int(i.traceCount.Load()), len(i.traces))

	for j := 0; j < numTraces; j++ {
		traceBytes, err := traces[j].Marshal()
		require.NoError(t, err)

		err = i.PushBytes(context.Background(), ids[j], traceBytes, nil)
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

func queryAll(t *testing.T, i *instance, ids [][]byte, traces []*tempopb.Trace) {
	for j, id := range ids {
		trace, err := i.FindTraceByID(context.Background(), id)
		require.NoError(t, err)
		require.Equal(t, traces[j], trace)
	}
}

func TestInstanceDoesNotRace(t *testing.T) {
	limits, err := overrides.NewOverrides(overrides.Limits{})
	require.NoError(t, err, "unexpected error creating limits")
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	tempDir, err := os.MkdirTemp("/tmp", "")
	require.NoError(t, err, "unexpected error getting temp dir")
	defer os.RemoveAll(tempDir)

	ingester, _, _ := defaultIngester(t, tempDir)

	i, err := newInstance(testTenantID, limiter, ingester.store, ingester.local)
	require.NoError(t, err, "unexpected error creating new instance")

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
		request := test.MakeRequest(10, []byte{})
		err = i.PushBytesRequest(context.Background(), request)
		require.NoError(t, err, "error pushing traces")
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
	limits, err := overrides.NewOverrides(overrides.Limits{
		MaxBytesPerTrace:      1000,
		MaxLocalTracesPerUser: 4,
	})
	require.NoError(t, err, "unexpected error creating limits")
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	tempDir, err := os.MkdirTemp("/tmp", "")
	require.NoError(t, err, "unexpected error getting temp dir")
	defer os.RemoveAll(tempDir)

	ingester, _, _ := defaultIngester(t, tempDir)

	type push struct {
		req          *tempopb.PushBytesRequest
		expectsError bool
	}

	tests := []struct {
		name   string
		pushes []push
	}{
		{
			name: "bytes - succeeds",
			pushes: []push{
				{
					req: test.MakeRequestWithByteLimit(300, []byte{}),
				},
				{
					req: test.MakeRequestWithByteLimit(500, []byte{}),
				},
				{
					req: test.MakeRequestWithByteLimit(900, []byte{}),
				},
			},
		},
		{
			name: "bytes - one fails",
			pushes: []push{
				{
					req: test.MakeRequestWithByteLimit(300, []byte{}),
				},
				{
					req:          test.MakeRequestWithByteLimit(1500, []byte{}),
					expectsError: true,
				},
				{
					req: test.MakeRequestWithByteLimit(900, []byte{}),
				},
			},
		},
		{
			name: "bytes - multiple pushes same trace",
			pushes: []push{
				{
					req: test.MakeRequestWithByteLimit(500, []byte{0x01}),
				},
				{
					req:          test.MakeRequestWithByteLimit(700, []byte{0x01}),
					expectsError: true,
				},
			},
		},
		{
			name: "max traces - too many",
			pushes: []push{
				{
					req: test.MakeRequestWithByteLimit(100, []byte{}),
				},
				{
					req: test.MakeRequestWithByteLimit(100, []byte{}),
				},
				{
					req: test.MakeRequestWithByteLimit(100, []byte{}),
				},
				{
					req: test.MakeRequestWithByteLimit(100, []byte{}),
				},
				{
					req:          test.MakeRequestWithByteLimit(100, []byte{}),
					expectsError: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i, err := newInstance(testTenantID, limiter, ingester.store, ingester.local)
			require.NoError(t, err, "unexpected error creating new instance")

			for j, push := range tt.pushes {
				err = i.PushBytesRequest(context.Background(), push.req)

				require.Equalf(t, push.expectsError, err != nil, "push %d failed: %w", j, err)
			}
		})
	}
}

func TestInstanceCutCompleteTraces(t *testing.T) {
	tempDir, _ := os.MkdirTemp("/tmp", "")
	defer os.RemoveAll(tempDir)

	id := make([]byte, 16)
	rand.Read(id)
	tracepb := test.MakeTraceBytes(10, id)
	pastTrace := &liveTrace{
		traceID:    id,
		traceBytes: tracepb,
		lastAppend: time.Now().Add(-time.Hour),
	}

	id = make([]byte, 16)
	rand.Read(id)
	nowTrace := &liveTrace{
		traceID:    id,
		traceBytes: tracepb,
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
			instance := defaultInstance(t, tempDir)

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
	tempDir, _ := os.MkdirTemp("/tmp", "")
	defer os.RemoveAll(tempDir)

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
			instance := defaultInstance(t, tempDir)

			for i := 0; i < tc.pushCount; i++ {
				request := test.MakeRequest(10, []byte{})
				err := instance.PushBytesRequest(context.Background(), request)
				require.NoError(t, err)
			}

			// Defaults
			if tc.maxBlockBytes == 0 {
				tc.maxBlockBytes = 1000
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
	limits, err := overrides.NewOverrides(overrides.Limits{})
	require.NoError(t, err, "unexpected error creating limits")
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	tempDir, err := os.MkdirTemp("/tmp", "")
	require.NoError(t, err, "unexpected error getting temp dir")
	defer os.RemoveAll(tempDir)

	ingester, _, _ := defaultIngester(t, tempDir)

	i, err := newInstance(testTenantID, limiter, ingester.store, ingester.local)
	require.NoError(t, err, "unexpected error creating new instance")

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
		request := test.MakeRequest(10, []byte{})
		err = i.PushBytesRequest(context.Background(), request)
		require.NoError(t, err)
	}
	cutAndVerify(count)

	cutAndVerify(0)
}

func TestInstanceFailsLargeTracesEvenAfterFlushing(t *testing.T) {
	ctx := context.Background()
	maxTraceBytes := 100
	id := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

	tempDir, err := os.MkdirTemp("/tmp", "")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	ingester, _, _ := defaultIngester(t, tempDir)

	limits, err := overrides.NewOverrides(overrides.Limits{
		MaxBytesPerTrace: maxTraceBytes,
	})
	require.NoError(t, err)
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	i, err := newInstance(testTenantID, limiter, ingester.store, ingester.local)
	require.NoError(t, err)

	pushFn := func(byteCount int) error {
		return i.PushBytes(ctx, id, make([]byte, byteCount), nil)
	}

	// Fill up trace to max
	err = pushFn(maxTraceBytes)
	require.NoError(t, err)

	// Pushing again fails
	err = pushFn(3)
	require.Contains(t, err.Error(), (newTraceTooLargeError(id, maxTraceBytes, 3)).Error())

	// Pushing still fails after flush
	err = i.CutCompleteTraces(0, true)
	require.NoError(t, err)
	err = pushFn(5)
	require.Contains(t, err.Error(), (newTraceTooLargeError(id, maxTraceBytes, 5)).Error())

	// Cut block and then pushing works again
	_, err = i.CutBlockIfReady(0, 0, true)
	require.NoError(t, err)
	err = pushFn(maxTraceBytes)
	require.NoError(t, err)
}

func defaultInstance(t require.TestingT, tmpDir string) *instance {
	limits, err := overrides.NewOverrides(overrides.Limits{})
	require.NoError(t, err, "unexpected error creating limits")
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	l, err := local.NewBackend(&local.Config{
		Path: tmpDir + "/blocks",
	})
	require.NoError(t, err)

	s, err := storage.NewStore(storage.Config{
		Trace: tempodb.Config{
			Backend: "local",
			Local: &local.Config{
				Path: tmpDir,
			},
			Block: &encoding.BlockConfig{
				IndexDownsampleBytes: 2,
				BloomFP:              0.01,
				BloomShardSizeBytes:  100_000,
				Encoding:             backend.EncLZ4_1M,
				IndexPageSizeBytes:   1000,
			},
			WAL: &wal.Config{
				Filepath:       tmpDir,
				SearchEncoding: backend.EncNone,
			},
		},
	}, log.NewNopLogger())
	require.NoError(t, err, "unexpected error creating store")

	instance, err := newInstance(testTenantID, limiter, s, l)
	require.NoError(t, err, "unexpected error creating new instance")

	return instance
}

func BenchmarkInstancePush(b *testing.B) {
	tempDir, err := os.MkdirTemp("/tmp", "")
	require.NoError(b, err, "unexpected error getting temp dir")
	defer os.RemoveAll(tempDir)

	instance := defaultInstance(b, tempDir)
	request := test.MakeRequest(10, []byte{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Rotate trace ID
		binary.LittleEndian.PutUint32(request.Ids[0].Slice, uint32(i))
		err = instance.PushBytesRequest(context.Background(), request)
		require.NoError(b, err)
	}
}

func BenchmarkInstancePushExistingTrace(b *testing.B) {
	tempDir, err := os.MkdirTemp("/tmp", "")
	require.NoError(b, err, "unexpected error getting temp dir")
	defer os.RemoveAll(tempDir)

	instance := defaultInstance(b, tempDir)
	request := test.MakeRequest(10, []byte{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = instance.PushBytesRequest(context.Background(), request)
		require.NoError(b, err)
	}
}

func BenchmarkInstanceFindTraceByID(b *testing.B) {
	tempDir, err := os.MkdirTemp("/tmp", "")
	require.NoError(b, err, "unexpected error getting temp dir")
	defer os.RemoveAll(tempDir)

	instance := defaultInstance(b, tempDir)
	traceID := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	request := test.MakeRequest(10, traceID)
	err = instance.PushBytesRequest(context.Background(), request)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trace, err := instance.FindTraceByID(context.Background(), traceID)
		require.NotNil(b, trace)
		require.NoError(b, err)
	}
}
