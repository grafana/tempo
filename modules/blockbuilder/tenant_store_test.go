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
	cycleEndTs := uint64(time.Now().Unix())
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

	err = store.AppendTrace(traceID, trace, startTime)
	require.NoError(t, err)

	assert.Equal(t, start.Unix(), store.headBlock.BlockMeta().StartTime.Unix())
	assert.Equal(t, end.Unix(), store.headBlock.BlockMeta().EndTime.Unix())
}
