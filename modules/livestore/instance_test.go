package livestore

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
	util_log "github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/pkg/util/test"
)

func instanceWithPushLimits(t *testing.T, maxBytesPerTrace int, maxLiveTraces int) (*instance, *LiveStore) {
	instance, ls := defaultInstance(t)
	limits, err := overrides.NewOverrides(overrides.Config{
		Defaults: overrides.Overrides{
			Global: overrides.GlobalOverrides{
				MaxBytesPerTrace: maxBytesPerTrace,
			},
			Ingestion: overrides.IngestionOverrides{
				MaxLocalTracesPerUser: maxLiveTraces,
			},
		},
	}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)
	instance.overrides = limits

	return instance, ls
}

func pushTrace(ctx context.Context, t *testing.T, instance *instance, tr *tempopb.Trace, id []byte) {
	b, err := tr.Marshal()
	require.NoError(t, err)
	req := &tempopb.PushBytesRequest{
		Traces: []tempopb.PreallocBytes{{Slice: b}},
		Ids:    [][]byte{id},
	}
	instance.pushBytes(ctx, time.Now(), req)
}

// TestInstanceLimits verifies MaxBytesPerTrace and MaxLocalTracesPerUser enforcement in livestore.
func TestInstanceLimits(t *testing.T) {
	const batches = 20
	// Configure limits: allow up to ~1.5x small trace, and max 4 live traces
	maxTraces := 4

	batch1 := test.MakeTrace(batches, test.ValidTraceID(nil))
	batch2 := test.MakeTrace(batches, test.ValidTraceID(nil))
	maxBytes := batch1.Size() + batch2.Size()/2 // set limit between 1 and 2 batches so pushing both batches to a single trace exceeds limit

	// bytes - succeeds: push two different traces under size limit
	t.Run("bytes - succeeds", func(t *testing.T) {
		instance, ls := instanceWithPushLimits(t, maxBytes, maxTraces)
		// two different traces with different ids
		id1 := test.ValidTraceID(nil)
		id2 := test.ValidTraceID(nil)
		pushTrace(t.Context(), t, instance, batch1, id1)
		pushTrace(t.Context(), t, instance, batch2, id2)
		require.Equal(t, uint64(2), instance.liveTraces.Len())

		err := services.StopAndAwaitTerminated(t.Context(), ls)
		require.NoError(t, err)
	})

	// bytes - one fails: second push of the same trace exceeds MaxBytesPerTrace
	t.Run("bytes - one fails", func(t *testing.T) {
		instance, ls := instanceWithPushLimits(t, maxBytes, maxTraces)

		id := test.ValidTraceID(nil)
		// First push fits
		pushTrace(t.Context(), t, instance, batch1, id)
		// Second push with same id will exceed combined size (> maxBytes)
		pushTrace(t.Context(), t, instance, batch2, id)
		// Only one live trace stored, and accumulated size should be <= maxBytes
		require.Equal(t, uint64(1), instance.liveTraces.Len())
		require.LessOrEqual(t, instance.liveTraces.Size(), uint64(maxBytes))

		err := services.StopAndAwaitTerminated(t.Context(), ls)
		require.NoError(t, err)
	})

	// bytes - second push fails even after cutIdleTraces
	t.Run("bytes - second push fails even after cutIdleTraces", func(t *testing.T) {
		instance, ls := instanceWithPushLimits(t, maxBytes, maxTraces)

		id := test.ValidTraceID(nil)
		// First push fits
		pushTrace(t.Context(), t, instance, batch1, id)

		// cut idle traces but we retain the too large trace in traceSizes
		drained, err := instance.cutIdleTraces(t.Context(), true)
		require.NoError(t, err)
		require.True(t, drained, "should drain live traces in one iteration")

		// Second push with same id will fail b/c we are still tracking in traceSizes
		pushTrace(t.Context(), t, instance, batch2, id)
		require.Equal(t, uint64(0), instance.liveTraces.Len())
		require.Equal(t, instance.liveTraces.Size(), uint64(0))

		err = services.StopAndAwaitTerminated(t.Context(), ls)
		require.NoError(t, err)
	})

	// bytes - second push succeeds after cutIdleTraces and 2x cutBlocks
	t.Run("bytes - second push succeeds after cutting head block 2x", func(t *testing.T) {
		instance, ls := instanceWithPushLimits(t, maxBytes, maxTraces)

		id := test.ValidTraceID(nil)
		// First push fits
		pushTrace(t.Context(), t, instance, batch1, id)

		// cut idle traces but we retain the too large trace in traceSizes
		drained, err := instance.cutIdleTraces(t.Context(), true)
		require.NoError(t, err)
		require.True(t, drained, "should drain live traces in one iteration")
		blockID, err := instance.cutBlocks(t.Context(), true) // this won't clear the trace b/c the trace must not be seen for 2 head block cuts to be fully removed from live traces
		require.NoError(t, err)
		_, err = instance.completeBlock(t.Context(), blockID)
		require.NoError(t, err)

		// push a second trace so cutIdle/cutBlocks goes through
		secondID := test.ValidTraceID(nil)
		pushTrace(t.Context(), t, instance, batch1, secondID)

		drained, err = instance.cutIdleTraces(t.Context(), true)
		require.NoError(t, err)
		require.True(t, drained, "should drain live traces in one iteration")
		blockID, err = instance.cutBlocks(t.Context(), true) // this will clear the trace b/c the trace has not been seen for 2 head block cuts
		require.NoError(t, err)
		_, err = instance.completeBlock(t.Context(), blockID)
		require.NoError(t, err)

		// Second push with same id will succeed b/c we have gone through one block flush cycles w/o seeing it
		pushTrace(t.Context(), t, instance, batch1, id)
		require.Equal(t, uint64(1), instance.liveTraces.Len())
		require.LessOrEqual(t, instance.liveTraces.Size(), uint64(maxBytes))

		err = services.StopAndAwaitTerminated(t.Context(), ls)
		require.NoError(t, err)
	})

	// max traces - too many: only first 4 unique traces are accepted
	t.Run("max traces - too many", func(t *testing.T) {
		instance, ls := instanceWithPushLimits(t, maxBytes, maxTraces)

		for range 10 {
			id := test.ValidTraceID(nil)
			ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(1*time.Second)) // Time out after 1s, push should be immediate
			t.Cleanup(cancel)
			pushTrace(ctx, t, instance, test.MakeTrace(1, id), id)
		}
		require.Equal(t, uint64(4), instance.liveTraces.Len())

		err := services.StopAndAwaitTerminated(t.Context(), ls)
		require.NoError(t, err)
	})
}

// TestTraceTooLargeLogContainsInsight verifies that the "trace too large" log line contains insight=true
func TestTraceTooLargeLogContainsInsight(t *testing.T) {
	id := test.ValidTraceID(nil)
	trace := test.MakeTrace(1, id)
	instance, ls := instanceWithPushLimits(t, trace.Size(), 4)

	// Replace maxTraceLogger to capture log output
	var logBuf bytes.Buffer
	instance.maxTraceLogger = util_log.NewRateLimitedLogger(maxTraceLogLinesPerSecond, log.NewLogfmtLogger(&logBuf))

	pushTrace(t.Context(), t, instance, trace, id)
	pushTrace(t.Context(), t, instance, trace, id) // second push exceeds limit

	assert.Contains(t, logBuf.String(), "insight=true")

	require.NoError(t, services.StopAndAwaitTerminated(t.Context(), ls))
}

func TestInstanceNoLimits(t *testing.T) {
	instance, ls := instanceWithPushLimits(t, 0, 0) // no limits by default

	for range 100 {
		id := test.ValidTraceID(nil)
		pushTrace(t.Context(), t, instance, test.MakeTrace(1, id), id)
	}

	assert.Equal(t, uint64(100), instance.liveTraces.Len())
	assert.GreaterOrEqual(t, instance.liveTraces.Size(), uint64(1000))

	err := services.StopAndAwaitTerminated(t.Context(), ls)
	require.NoError(t, err)
}

func TestInstanceBackpressure(t *testing.T) {
	instance, ls := defaultInstance(t)

	id1 := test.ValidTraceID(nil)
	pushTrace(t.Context(), t, instance, test.MakeTrace(1, id1), id1)

	instance.Cfg.MaxLiveTracesBytes = instance.liveTraces.Size() // Set max size to current live-traces size

	id2 := test.ValidTraceID(nil)

	// Use a channel to coordinate the blocking push operation
	pushComplete := make(chan struct{})
	go func() {
		defer close(pushComplete)
		// Second write will block waiting for the live traces to have room
		pushTrace(t.Context(), t, instance, test.MakeTrace(1, id2), id2)
	}()

	// Give goroutine time to start and block
	time.Sleep(10 * time.Millisecond)

	// First trace is found
	res, err := instance.FindByTraceID(t.Context(), id1, true)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Trace)
	require.Greater(t, res.Trace.Size(), 0)

	// Second is not (should be blocked)
	res, err = instance.FindByTraceID(t.Context(), id2, true)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Nil(t, res.Trace)

	// Free up space for the blocked push
	drained, cutErr := instance.cutIdleTraces(t.Context(), true)
	require.NoError(t, cutErr)
	require.True(t, drained, "should drain live traces in one iteration")

	// Wait for push to complete with timeout
	select {
	case <-pushComplete:
		// Push completed successfully
	case <-time.After(1 * time.Second):
		t.Fatal("push operation did not complete within timeout")
	}

	// After cut, second trace is pushed to instance and can be found
	res, err = instance.FindByTraceID(t.Context(), id2, true)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Trace)
	require.Greater(t, res.Trace.Size(), 0)

	require.NoError(t, services.StopAndAwaitTerminated(t.Context(), ls))
}

func TestInstanceWALBackpressure(t *testing.T) {
	inst, ls := defaultInstance(t)
	// Disable live traces backpressure so we only test WAL backpressure.
	inst.Cfg.MaxLiveTracesBytes = 0

	// Build up WAL blocks: push a trace, flush to head, cut to WAL.
	createWALBlock := func() {
		id := test.ValidTraceID(nil)
		pushTrace(t.Context(), t, inst, test.MakeTrace(1, id), id)
		drained, cutErr := inst.cutIdleTraces(t.Context(), true)
		require.NoError(t, cutErr)
		require.True(t, drained, "should drain live traces in one iteration")
		walID, err := inst.cutBlocks(t.Context(), true)
		require.NoError(t, err)
		require.NotEqual(t, walID, [16]byte{})
	}

	// At the limit, no backpressure.
	for range walBackpressureLimit {
		createWALBlock()
	}
	require.False(t, inst.backpressure(t.Context()), "expected no backpressure at %d WAL blocks", walBackpressureLimit)

	// One more WAL block should trigger backpressure.
	createWALBlock()
	require.True(t, inst.backpressure(t.Context()), "expected backpressure at %d WAL blocks", walBackpressureLimit+1)

	require.NoError(t, services.StopAndAwaitTerminated(t.Context(), ls))
}

func TestCutIdleTracesRespectsMaxBlockBytes(t *testing.T) {
	inst, ls := defaultInstance(t)

	inst.Cfg.MaxBlockBytes = 5 * 1024 * 1024 // 5 Mb

	// Push enough traces to require multiple blocks.
	traceCount := 100 // that will be around 119Mb
	traceIDs := make([][]byte, 0, traceCount)
	traces := make([]*tempopb.Trace, 0, traceCount)

	for range traceCount {
		id := test.ValidTraceID(nil)
		tr := test.MakeTraceWithSpanCount(50, 100, id)
		pushTrace(t.Context(), t, inst, tr, id)
		traceIDs = append(traceIDs, id)
		traces = append(traces, tr)
	}

	ls.cutOneInstanceToWal(t.Context(), inst, true)
	for i := 0; i < traceCount; i += 5 {
		requireTraceInLiveStore(t, ls, traceIDs[i], traces[i])
	}

	// Verify no WAL block exceeds MaxBlockBytes.
	walBlocksNum := len(inst.walBlocks)
	assert.Greater(t, walBlocksNum, 2, "expected multiple WAL blocks")
	for id, blk := range inst.walBlocks {
		// block size estimation can be x5 off, so we check that that block size at least makes sense
		assert.LessOrEqual(t, blk.DataLength(), inst.Cfg.MaxBlockBytes*5,
			"WAL block %s exceeds MaxBlockBytes: %d > %d", id, blk.DataLength(), inst.Cfg.MaxBlockBytes)
	}

	// with no new traces, number of WAL blocks should not increase after another cut to WAL
	ls.cutOneInstanceToWal(t.Context(), inst, true)
	assert.Equal(t, walBlocksNum, len(inst.walBlocks), "expected no new WAL blocks after cut to WAL with no new traces")

	require.NoError(t, services.StopAndAwaitTerminated(t.Context(), ls))
}
