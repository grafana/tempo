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

	w, err := wal.New(&wal.Config{
		Filepath:       tmpDir,
		Encoding:       backend.EncNone,
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
		ctx   = context.Background()
		log   = log.NewNopLogger()
		store = newStoreWithLogger(ctx, t, log)
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

	// This allows a polling cycle
	time.Sleep(100 * time.Millisecond)

	metas := store.BlockMetas(ts.tenantID)
	require.Equal(t, 1, len(metas))

	return metas[0]
}
