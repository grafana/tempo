package blockbuilder

import (
	"testing"
	"time"

	"github.com/go-kit/log"
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

func TestAdjustTimeRangeForSlack(t *testing.T) {
	store, err := getTenantStore(t)
	require.NoError(t, err)

	startCycleTime := time.Now()

	tests := []struct {
		name          string
		start         uint32
		end           uint32
		expectedStart uint32
		expectedEnd   uint32
	}{
		{
			name:          "within slack range",
			start:         uint32(startCycleTime.Add(-2 * time.Minute).Unix()),
			end:           uint32(startCycleTime.Add(2 * time.Minute).Unix()),
			expectedStart: uint32(startCycleTime.Add(-2 * time.Minute).Unix()),
			expectedEnd:   uint32(startCycleTime.Add(2 * time.Minute).Unix()),
		},
		{
			name:          "start before slack range",
			start:         uint32(startCycleTime.Add(-10 * time.Minute).Unix()),
			end:           uint32(startCycleTime.Add(2 * time.Minute).Unix()),
			expectedStart: uint32(startCycleTime.Unix()),
			expectedEnd:   uint32(startCycleTime.Add(2 * time.Minute).Unix()),
		},
		{
			name:          "end after slack range",
			start:         uint32(startCycleTime.Add(-2 * time.Minute).Unix()),
			end:           uint32(startCycleTime.Add(20 * time.Minute).Unix()),
			expectedStart: uint32(startCycleTime.Add(-2 * time.Minute).Unix()),
			expectedEnd:   uint32(startCycleTime.Unix()),
		},
		{
			name:          "end before start",
			start:         uint32(startCycleTime.Add(-2 * time.Minute).Unix()),
			end:           uint32(startCycleTime.Add(-3 * time.Minute).Unix()),
			expectedStart: uint32(startCycleTime.Add(-2 * time.Minute).Unix()),
			expectedEnd:   uint32(startCycleTime.Unix()),
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
