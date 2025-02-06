package blockbuilder

import (
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/stretchr/testify/require"
)

func getTenantStore(t *testing.T, startTime time.Time, cycleDuration, slackDuration time.Duration) (*tenantStore, error) {
	var (
		logger      = log.NewNopLogger()
		blockCfg    = BlockConfig{}
		tmpDir      = t.TempDir()
		partition   = uint64(1)
		startOffset = uint64(1)
	)

	w, err := wal.New(&wal.Config{
		Filepath:       tmpDir,
		Encoding:       backend.EncNone,
		IngestionSlack: 3 * time.Minute,
		Version:        encoding.DefaultEncoding().Version(),
	})
	require.NoError(t, err)
	return newTenantStore("test-tenant", partition, startOffset, startTime, cycleDuration, slackDuration, blockCfg, logger, w, encoding.DefaultEncoding(), &mockOverrides{})
}

func TestAdjustTimeRangeForSlack(t *testing.T) {
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
			expectedStart: startCycleTime,
			expectedEnd:   startCycleTime.Add(2 * time.Minute),
		},
		{
			name:          "end after slack range",
			start:         startCycleTime.Add(-2 * time.Minute),
			end:           startCycleTime.Add(20 * time.Minute),
			expectedStart: startCycleTime.Add(-2 * time.Minute),
			expectedEnd:   startCycleTime,
		},
		{
			name:          "end before start",
			start:         startCycleTime.Add(-2 * time.Minute),
			end:           startCycleTime.Add(-3 * time.Minute),
			expectedStart: startCycleTime.Add(-2 * time.Minute),
			expectedEnd:   startCycleTime,
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
