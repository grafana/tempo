package blockbuilder

import (
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTenantStore(t *testing.T) (*tenantStore, error) {
	logger := log.NewNopLogger()
	blockCfg := BlockConfig{}
	tmpDir := t.TempDir()
	w, err := wal.New(&wal.Config{
		Filepath:       tmpDir,
		Encoding:       backend.EncNone,
		IngestionSlack: 3 * time.Minute,
		Version:        encoding.DefaultEncoding().Version(),
	})
	require.NoError(t, err)
	return newTenantStore("test-tenant", 1, 1, blockCfg, logger, w, encoding.DefaultEncoding(), &mockOverrides{})
}

func TestAppendTraceHonorCycleTime(t *testing.T) {
	store, err := getTenantStore(t)
	require.NoError(t, err)

	traceID := []byte("test-trace-id")
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now().Add(1 * time.Hour)
	trace := test.MakeTraceWithTimeRange(1, traceID, uint64(start.UnixNano()), uint64(end.UnixNano()))
	startTime := time.Now().Add(-1 * time.Hour)

	err = store.AppendTrace(traceID, trace, startTime)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, store.headBlock.BlockMeta().StartTime.Unix(), start.Unix())
	assert.LessOrEqual(t, store.headBlock.BlockMeta().EndTime.Unix(), end.Unix())
}

func TestAdjustTimeRangeForSlack(t *testing.T) {
	store, err := getTenantStore(t)
	require.NoError(t, err)

	startCycleTime := time.Now()

	tests := []struct {
		name          string
		start         uint64
		end           uint64
		expectedStart uint64
		expectedEnd   uint64
	}{
		{
			name:          "within slack range",
			start:         uint64(startCycleTime.Add(-2 * time.Minute).Unix()),
			end:           uint64(startCycleTime.Add(2 * time.Minute).Unix()),
			expectedStart: uint64(startCycleTime.Add(-2 * time.Minute).Unix()),
			expectedEnd:   uint64(startCycleTime.Add(2 * time.Minute).Unix()),
		},
		{
			name:          "start before slack range",
			start:         uint64(startCycleTime.Add(-10 * time.Minute).Unix()),
			end:           uint64(startCycleTime.Add(2 * time.Minute).Unix()),
			expectedStart: uint64(startCycleTime.Unix()),
			expectedEnd:   uint64(startCycleTime.Add(2 * time.Minute).Unix()),
		},
		{
			name:          "end after slack range",
			start:         uint64(startCycleTime.Add(-2 * time.Minute).Unix()),
			end:           uint64(startCycleTime.Add(20 * time.Minute).Unix()),
			expectedStart: uint64(startCycleTime.Add(-2 * time.Minute).Unix()),
			expectedEnd:   uint64(startCycleTime.Unix()),
		},
		{
			name:          "end before start",
			start:         uint64(startCycleTime.Add(-2 * time.Minute).Unix()),
			end:           uint64(startCycleTime.Add(-3 * time.Minute).Unix()),
			expectedStart: uint64(startCycleTime.Add(-2 * time.Minute).Unix()),
			expectedEnd:   uint64(startCycleTime.Unix()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := store.adjustTimeRangeForSlack(startCycleTime, tt.start, tt.end)
			assert.Equal(t, tt.expectedStart, start)
			assert.Equal(t, tt.expectedEnd, end)
		})
	}
}
