package blockbuilder

import (
	"context"
	"flag"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/stretchr/testify/require"
)

func getTenantStore(t *testing.T, startTime time.Time, cycleDuration, slackDuration time.Duration, publisher addPublisher) (*tenantStore, error) {
	var (
		logger      = log.NewNopLogger()
		tmpDir      = t.TempDir()
		partition   = uint64(1)
		startOffset = uint64(1)
		blockCfg    = BlockConfig{}
	)

	blockCfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	// Simulate modules.go injection of storage.trace.block config
	blockCfg.BlockConfig.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	blockCfg.Version = encoding.DefaultEncoding().Version()

	w, err := wal.New(&wal.Config{
		Filepath:       tmpDir,
		IngestionSlack: 3 * time.Minute,
		Version:        encoding.DefaultEncoding().Version(),
	})
	require.NoError(t, err)
	return newTenantStore("test-tenant", partition, startOffset, startTime, cycleDuration, slackDuration, blockCfg, logger, w, encoding.DefaultEncoding(), &mockOverrides{}, publisher)
}

// recordingPublisher is a test fake for addPublisher: it records every
// PublishAdd call's arguments so tests can assert exactly what Flush
// captured, without constructing a real Kafka-backed Publisher. The zero
// value is disabled, matching addPublisher's real-world default
// (bloomgatewayevents.Config.Enabled defaults to false).
type recordingPublisher struct {
	mu      sync.Mutex
	enabled bool
	adds    []recordedAdd
}

type recordedAdd struct {
	blockID  backend.UUID
	tenantID string
	start    time.Time
	end      time.Time
	traceIDs [][]byte
}

func (p *recordingPublisher) Enabled() bool {
	return p.enabled
}

func (p *recordingPublisher) PublishAdd(_ context.Context, blockID backend.UUID, tenantID string, start, end time.Time, traceIDs [][]byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.adds = append(p.adds, recordedAdd{blockID: blockID, tenantID: tenantID, start: start, end: end, traceIDs: traceIDs})
}

// snapshotAdds returns a copy of the calls recorded so far, safe to
// inspect after Flush returns.
func (p *recordingPublisher) snapshotAdds() []recordedAdd {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]recordedAdd(nil), p.adds...)
}

var _ addPublisher = (*recordingPublisher)(nil)

// pushTraces appends n traces with distinct random IDs to ts and returns
// the IDs pushed.
func pushTraces(t *testing.T, ts *tenantStore, n int, start, end time.Time) [][]byte {
	t.Helper()
	ids := make([][]byte, 0, n)
	for i := 0; i < n; i++ {
		req := test.MakePushBytesRequest(t, 3, nil, uint64(start.UnixNano()), uint64(end.UnixNano()))
		for j := range req.Traces {
			require.NoError(t, ts.AppendTrace(req.Ids[j], req.Traces[j].Slice, start))
			ids = append(ids, req.Ids[j])
		}
	}
	return ids
}

func TestTenantStoreAdjustTimeRangeForSlack(t *testing.T) {
	var (
		startCycleTime = time.Now()
		cycleDuration  = time.Minute
		slackDuration  = 3 * time.Minute
	)

	store, err := getTenantStore(t, startCycleTime, cycleDuration, slackDuration, &recordingPublisher{})
	require.NoError(t, err)

	tests := []struct {
		name          string
		start         time.Time
		end           time.Time
		expectedStart time.Time
		expectedEnd   time.Time
	}{
		{
			name:          "within slack range",
			start:         startCycleTime.Add(-2 * time.Minute),
			end:           startCycleTime.Add(2 * time.Minute),
			expectedStart: startCycleTime.Add(-2 * time.Minute),
			expectedEnd:   startCycleTime.Add(2 * time.Minute),
		},
		{
			name:          "start before slack range",
			start:         startCycleTime.Add(-10 * time.Minute),
			end:           startCycleTime.Add(2 * time.Minute),
			expectedStart: startCycleTime.Add(-slackDuration),
			expectedEnd:   startCycleTime.Add(2 * time.Minute),
		},
		{
			name:          "end after slack range",
			start:         startCycleTime.Add(-2 * time.Minute),
			end:           startCycleTime.Add(20 * time.Minute),
			expectedStart: startCycleTime.Add(-2 * time.Minute),
			expectedEnd:   startCycleTime.Add(slackDuration + cycleDuration),
		},
		{
			name:          "end before start",
			start:         startCycleTime.Add(-10 * time.Minute),
			end:           startCycleTime.Add(-9 * time.Minute),
			expectedStart: startCycleTime.Add(-slackDuration),
			expectedEnd:   startCycleTime.Add(-slackDuration),
		},
		{
			name:          "both start and end after slack range",
			start:         startCycleTime.Add(5 * time.Minute),
			end:           startCycleTime.Add(10 * time.Minute),
			expectedStart: startCycleTime.Add(cycleDuration + slackDuration),
			expectedEnd:   startCycleTime.Add(cycleDuration + slackDuration),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := store.adjustTimeRangeForSlack(tt.start, tt.end)
			require.Equal(t, tt.expectedStart, start)
			require.Equal(t, tt.expectedEnd, end)
		})
	}
}

func TestTenantStoreEndToEndHistoricalData(t *testing.T) {
	var (
		count     = 3
		startTime = time.Now().Add(-24 * time.Hour)
	)

	testCases := []struct {
		name              string
		traceStart        time.Time // All spans will start at this time
		traceEnd          time.Time // All spans will end at this time
		cycleDuration     time.Duration
		slackDuration     time.Duration
		expectedStartTime time.Time
		expectedEndTime   time.Time
	}{
		{
			name:              "all good",
			traceStart:        startTime,
			traceEnd:          startTime.Add(time.Minute),
			cycleDuration:     5 * time.Minute,
			slackDuration:     5 * time.Minute,
			expectedStartTime: startTime,
			expectedEndTime:   startTime.Add(time.Minute),
		},
		{
			name:              "before start",
			traceStart:        startTime.Add(-10 * time.Minute),
			traceEnd:          startTime,
			cycleDuration:     5 * time.Minute,
			slackDuration:     5 * time.Minute,
			expectedStartTime: startTime.Add(-5 * time.Minute), // Only goes as early as -slack
			expectedEndTime:   startTime,
		},
		{
			name:              "after end",
			traceStart:        startTime,
			traceEnd:          startTime.Add(20 * time.Minute),
			cycleDuration:     5 * time.Minute,
			slackDuration:     5 * time.Minute,
			expectedStartTime: startTime,
			expectedEndTime:   startTime.Add(10 * time.Minute), // Only goes as far as cycle + slack
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			meta := writeHistoricalData(t, count, startTime, tc.cycleDuration, tc.slackDuration, tc.traceStart, tc.traceEnd)

			// NOTE - Roundtripped times from the meta json don't include the monotonic clock value,
			//        therefore we do the same on the test values by calling Truncate(0).
			require.Equal(t, tc.expectedStartTime.Truncate(0).UTC(), meta.StartTime.UTC())
			require.Equal(t, tc.expectedEndTime.Truncate(0).UTC(), meta.EndTime.UTC())

			// Verify other properties of the block
			require.EqualValues(t, count, meta.TotalObjects)
			require.EqualValues(t, 1, meta.TotalRecords)
			require.Greater(t, meta.Size_, uint64(0))
		})
	}
}

func writeHistoricalData(t *testing.T, count int, startTime time.Time, cycleDuration, slackDuration time.Duration, traceStart, traceEnd time.Time) *backend.BlockMeta {
	var (
		ctx   = t.Context()
		log   = log.NewNopLogger()
		store = newStoreWithLogger(ctx, t, log, false)
	)

	ts, err := getTenantStore(t, startTime, cycleDuration, slackDuration, &recordingPublisher{})
	require.NoError(t, err)

	for i := 0; i < count; i++ {
		req := test.MakePushBytesRequest(t, 3, nil, uint64(traceStart.UnixNano()), uint64(traceEnd.UnixNano()))
		for j := range req.Traces {
			err = ts.AppendTrace(req.Ids[j], req.Traces[j].Slice, startTime)
			require.NoError(t, err)
		}
	}

	err = ts.Flush(ctx, store, store, store)
	require.NoError(t, err)
	err = ts.AllowCompaction(ctx, store)
	require.NoError(t, err)

	store.PollNow(ctx)
	metas := store.BlockMetas(ts.tenantID)
	require.Equal(t, 1, len(metas))

	return metas[0]
}

func TestTenantStoreNoCompactFlag(t *testing.T) {
	var (
		ctx           = t.Context()
		count         = 3
		startTime     = time.Now().Add(-24 * time.Hour)
		cycleDuration = 5 * time.Minute
		slackDuration = 5 * time.Minute
		traceStart    = startTime
		traceEnd      = startTime.Add(time.Minute)

		log   = log.NewNopLogger()
		store = newStoreWithLogger(ctx, t, log, true)
	)

	ts, err := getTenantStore(t, startTime, cycleDuration, slackDuration, &recordingPublisher{})
	require.NoError(t, err)

	for range count {
		req := test.MakePushBytesRequest(t, 3, nil, uint64(traceStart.UnixNano()), uint64(traceEnd.UnixNano()))
		for j := range req.Traces {
			err = ts.AppendTrace(req.Ids[j], req.Traces[j].Slice, startTime)
			require.NoError(t, err)
		}
	}

	err = ts.Flush(ctx, store, store, store)
	require.NoError(t, err)

	store.PollNow(ctx)
	metas := store.BlockMetas(ts.tenantID)
	require.Equal(t, 0, len(metas))

	// block should be available for polling after compaction is allowed
	err = ts.AllowCompaction(ctx, store)
	require.NoError(t, err)

	store.PollNow(ctx)
	metas = store.BlockMetas(ts.tenantID)
	require.Equal(t, 1, len(metas))

	actualMeta := metas[0]

	// Verify other properties of the block
	require.EqualValues(t, count, actualMeta.TotalObjects)
	require.EqualValues(t, 1, actualMeta.TotalRecords)
	require.Greater(t, actualMeta.Size_, uint64(0))
}

func TestFlush_PublishesCapturedIDs(t *testing.T) {
	var (
		ctx           = t.Context()
		startTime     = time.Now().Add(-24 * time.Hour)
		cycleDuration = 5 * time.Minute
		slackDuration = 5 * time.Minute
		publisher     = &recordingPublisher{enabled: true}
		store         = newStoreWithLogger(ctx, t, log.NewNopLogger(), false)
	)

	ts, err := getTenantStore(t, startTime, cycleDuration, slackDuration, publisher)
	require.NoError(t, err)

	wantIDs := pushTraces(t, ts, 5, startTime, startTime.Add(time.Minute))

	require.NoError(t, ts.Flush(ctx, store, store, store))

	adds := publisher.snapshotAdds()
	require.Len(t, adds, 1)
	require.ElementsMatch(t, wantIDs, adds[0].traceIDs)
	require.Equal(t, ts.tenantID, adds[0].tenantID)

	// Compare against the block Flush actually produced.
	require.NoError(t, ts.AllowCompaction(ctx, store))
	store.PollNow(ctx)
	metas := store.BlockMetas(ts.tenantID)
	require.Len(t, metas, 1)
	meta := metas[0]

	require.Equal(t, meta.BlockID, adds[0].blockID)
	require.True(t, meta.StartTime.Equal(adds[0].start))
	require.True(t, meta.EndTime.Equal(adds[0].end))
}

func TestFlush_NoLiveTraces_NoPublish(t *testing.T) {
	var (
		ctx           = t.Context()
		startTime     = time.Now()
		cycleDuration = 5 * time.Minute
		slackDuration = 5 * time.Minute
		publisher     = &recordingPublisher{enabled: true}
	)

	ts, err := getTenantStore(t, startTime, cycleDuration, slackDuration, publisher)
	require.NoError(t, err)

	// Nothing appended -- Flush must take its early-return path (no live
	// traces) without ever touching the store, let alone publishing.
	require.NoError(t, ts.Flush(ctx, nil, nil, nil))
	require.Empty(t, publisher.snapshotAdds())
}

func TestFlush_DisabledPublisher_NoCaptureNoPublish(t *testing.T) {
	var (
		ctx           = t.Context()
		startTime     = time.Now().Add(-24 * time.Hour)
		cycleDuration = 5 * time.Minute
		slackDuration = 5 * time.Minute
		publisher     = &recordingPublisher{enabled: false}
		store         = newStoreWithLogger(ctx, t, log.NewNopLogger(), false)
	)

	ts, err := getTenantStore(t, startTime, cycleDuration, slackDuration, publisher)
	require.NoError(t, err)

	pushTraces(t, ts, 3, startTime, startTime.Add(time.Minute))

	require.NoError(t, ts.Flush(ctx, store, store, store))

	// Enabled() == false means Flush never wraps the iterator in a tee: the
	// only externally observable evidence, since the tee is internal to
	// Flush, is that no PublishAdd call happens at all, at zero cost.
	require.Empty(t, publisher.snapshotAdds())

	// Flush's own behavior must be unaffected by the disabled publisher.
	require.NoError(t, ts.AllowCompaction(ctx, store))
	store.PollNow(ctx)
	metas := store.BlockMetas(ts.tenantID)
	require.Len(t, metas, 1)
	require.EqualValues(t, 3, metas[0].TotalObjects)
}

func TestFlush_MultipleFlushCycles_NoCrossContamination(t *testing.T) {
	var (
		ctx           = t.Context()
		startTime     = time.Now().Add(-24 * time.Hour)
		cycleDuration = 5 * time.Minute
		slackDuration = 5 * time.Minute
		publisher     = &recordingPublisher{enabled: true}
		store         = newStoreWithLogger(ctx, t, log.NewNopLogger(), false)
	)

	ts, err := getTenantStore(t, startTime, cycleDuration, slackDuration, publisher)
	require.NoError(t, err)

	firstIDs := pushTraces(t, ts, 4, startTime, startTime.Add(time.Minute))
	require.NoError(t, ts.Flush(ctx, store, store, store))

	secondIDs := pushTraces(t, ts, 2, startTime, startTime.Add(time.Minute))
	require.NoError(t, ts.Flush(ctx, store, store, store))

	adds := publisher.snapshotAdds()
	require.Len(t, adds, 2)
	require.ElementsMatch(t, firstIDs, adds[0].traceIDs)
	require.ElementsMatch(t, secondIDs, adds[1].traceIDs)
	require.NotEqual(t, adds[0].blockID, adds[1].blockID)
}
