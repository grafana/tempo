package blockbuilder

import (
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/stretchr/testify/require"
)

func getPartitionWriter(t *testing.T) *writer {
	var (
		logger        = log.NewNopLogger()
		blockCfg      = BlockConfig{}
		tmpDir        = t.TempDir()
		partition     = uint64(1)
		startOffset   = uint64(1)
		startTime     = time.Now()
		cycleDuration = time.Minute
		slackDuration = time.Minute
	)
	w, err := wal.New(&wal.Config{
		Filepath:       tmpDir,
		Encoding:       backend.EncNone,
		IngestionSlack: 3 * time.Minute,
		Version:        encoding.DefaultEncoding().Version(),
	})
	require.NoError(t, err)

	return newPartitionSectionWriter(logger, partition, startOffset, startTime, cycleDuration, slackDuration, blockCfg, &mockOverrides{}, w, encoding.DefaultEncoding())
}

func TestPushBytes(t *testing.T) {
	pw := getPartitionWriter(t)

	tenant := "test-tenant"
	traceID := generateTraceID(t)
	now := time.Now()
	startTime := uint64(now.UnixNano())
	endTime := uint64(now.Add(time.Second).UnixNano())
	req := test.MakePushBytesRequest(t, 1, traceID, startTime, endTime)

	err := pw.pushBytes(now, tenant, req)
	require.NoError(t, err)
}
