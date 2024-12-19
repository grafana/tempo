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
	cycleEndTs := time.Now().Unix()
	blockCfg := BlockConfig{}
	tmpDir := t.TempDir()
	w, err := wal.New(&wal.Config{
		Filepath:       tmpDir,
		Encoding:       backend.EncNone,
		IngestionSlack: 3 * time.Minute,
		Version:        encoding.DefaultEncoding().Version(),
	})
	require.NoError(t, err)
	return newTenantStore("test-tenant", 1, cycleEndTs, blockCfg, logger, w, encoding.DefaultEncoding(), &mockOverrides{})
}

func TestAppendTraceHonorCycleTime(t *testing.T) {
	store, err := getTenantStore(t)
	require.NoError(t, err)

	traceID := []byte("test-trace-id")
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now().Add(-1 * time.Hour)
	trace := test.MakeTraceWithTimeRange(1, traceID, uint64(start.UnixNano()), uint64(end.UnixNano()))
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := startTime.Add(5 * time.Minute)

	err = store.AppendTrace(traceID, trace, startTime, endTime)
	require.NoError(t, err)

	assert.Equal(t, start.Unix(), store.headBlock.BlockMeta().StartTime.Unix())
	assert.Equal(t, end.Unix(), store.headBlock.BlockMeta().EndTime.Unix())
}

func TestAdjustTimeRangeForSlack(t *testing.T) {
	store, err := getTenantStore(t)
	require.NoError(t, err)

	startCycleTime := time.Now()
	endCycleTime := startCycleTime.Add(10 * time.Minute)

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
			end:           uint64(endCycleTime.Add(2 * time.Minute).Unix()),
			expectedStart: uint64(startCycleTime.Add(-2 * time.Minute).Unix()),
			expectedEnd:   uint64(endCycleTime.Add(2 * time.Minute).Unix()),
		},
		{
			name:          "start before slack range",
			start:         uint64(startCycleTime.Add(-10 * time.Minute).Unix()),
			end:           uint64(endCycleTime.Add(2 * time.Minute).Unix()),
			expectedStart: uint64(startCycleTime.Unix()),
			expectedEnd:   uint64(endCycleTime.Add(2 * time.Minute).Unix()),
		},
		{
			name:          "end after slack range",
			start:         uint64(startCycleTime.Add(-2 * time.Minute).Unix()),
			end:           uint64(endCycleTime.Add(20 * time.Minute).Unix()),
			expectedStart: uint64(startCycleTime.Add(-2 * time.Minute).Unix()),
			expectedEnd:   uint64(endCycleTime.Unix()),
		},
		{
			name:          "end before start",
			start:         uint64(startCycleTime.Add(-2 * time.Minute).Unix()),
			end:           uint64(startCycleTime.Add(-3 * time.Minute).Unix()),
			expectedStart: uint64(startCycleTime.Add(-2 * time.Minute).Unix()),
			expectedEnd:   uint64(endCycleTime.Unix()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := store.adjustTimeRangeForSlack(startCycleTime, endCycleTime, tt.start, tt.end)
			assert.Equal(t, tt.expectedStart, start)
			assert.Equal(t, tt.expectedEnd, end)
		})
	}
}
