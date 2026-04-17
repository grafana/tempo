package blockbuilder

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/stretchr/testify/require"
)

func getTenantStore(t *testing.T, startTime time.Time, cycleDuration, slackDuration time.Duration) (*tenantStore, error) {
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
	return newTenantStore("test-tenant", partition, startOffset, startTime, cycleDuration, slackDuration, blockCfg, logger, w, encoding.DefaultEncoding(), &mockOverrides{})
}

func TestTenantStoreAdjustTimeRangeForSlack(t *testing.T) {
	var (
		startCycleTime = time.Now()
		cycleDuration  = time.Minute
		slackDuration  = 3 * time.Minute
	)

	store, err := getTenantStore(t, startCycleTime, cycleDuration, slackDuration)
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

	ts, err := getTenantStore(t, startTime, cycleDuration, slackDuration)
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

// TestTenantStoreReset verifies that Reset() clears liveTraces but does NOT
// reset the ID generator, so subsequent Flush() calls produce new unique block IDs.
func TestTenantStoreReset(t *testing.T) {
	startTime := time.Now().Add(-24 * time.Hour)
	store, err := getTenantStore(t, startTime, 5*time.Minute, 5*time.Minute)
	require.NoError(t, err)

	// Append a trace so liveTraces is non-empty.
	req := test.MakePushBytesRequest(t, 1, nil, uint64(startTime.UnixNano()), uint64(startTime.Add(time.Minute).UnixNano()))
	err = store.AppendTrace(req.Ids[0], req.Traces[0].Slice, startTime)
	require.NoError(t, err)
	require.Greater(t, store.liveTraces.Len(), uint64(0), "liveTraces must be non-empty before Reset")

	// Capture an ID before Reset() to verify the generator state advances across the boundary.
	id1 := store.idGenerator.NewID()

	store.Reset()

	require.Equal(t, uint64(0), store.liveTraces.Len(), "liveTraces must be empty after Reset")
	require.Equal(t, uint64(0), store.liveTraces.Size(), "liveTraces size must be zero after Reset")

	// ID generator must still be functional (not nil or panicking) and must
	// produce a different ID than before Reset() — proving it was not re-seeded.
	id2 := store.idGenerator.NewID()
	require.NotEmpty(t, id2.String(), "idGenerator must still work after Reset")
	require.NotEqual(t, id1, id2, "idGenerator must not be re-seeded by Reset")
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

	ts, err := getTenantStore(t, startTime, cycleDuration, slackDuration)
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

// TestTenantStoreAllowCompaction_MultipleBlocks verifies that AllowCompaction
// removes the no-compact flag from ALL blocks written during a cycle, not just the last.
func TestTenantStoreAllowCompaction_MultipleBlocks(t *testing.T) {
	var (
		ctx           = context.Background()
		startTime     = time.Now().Add(-24 * time.Hour)
		cycleDuration = 5 * time.Minute
		slackDuration = 5 * time.Minute
		logger        = log.NewNopLogger()
		store         = newStoreWithLogger(ctx, t, logger, true) // noCompact=true
	)

	ts, err := getTenantStore(t, startTime, cycleDuration, slackDuration)
	require.NoError(t, err)

	// First flush — append traces and flush.
	req1 := test.MakePushBytesRequest(t, 2, nil, uint64(startTime.UnixNano()), uint64(startTime.Add(time.Minute).UnixNano()))
	for j := range req1.Traces {
		require.NoError(t, ts.AppendTrace(req1.Ids[j], req1.Traces[j].Slice, startTime))
	}
	require.NoError(t, ts.Flush(ctx, store, store, store))
	ts.Reset()

	// Second flush — append more traces and flush.
	req2 := test.MakePushBytesRequest(t, 2, nil, uint64(startTime.UnixNano()), uint64(startTime.Add(time.Minute).UnixNano()))
	for j := range req2.Traces {
		require.NoError(t, ts.AppendTrace(req2.Ids[j], req2.Traces[j].Slice, startTime))
	}
	require.NoError(t, ts.Flush(ctx, store, store, store))

	// Before AllowCompaction: 0 blocks visible (all have noCompact flag).
	store.PollNow(ctx)
	require.Equal(t, 0, len(store.BlockMetas(ts.tenantID)), "blocks must not be visible before AllowCompaction")

	// AllowCompaction must clear flags on BOTH blocks.
	require.NoError(t, ts.AllowCompaction(ctx, store))

	store.PollNow(ctx)
	metas := store.BlockMetas(ts.tenantID)
	require.Equal(t, 2, len(metas), "both blocks must be visible after AllowCompaction")
}
