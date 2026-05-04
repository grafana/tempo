package blockbuilder

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/util/test"
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
	blockCfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	blockCfg.BlockConfig.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	blockCfg.Version = encoding.DefaultEncoding().Version()

	w, err := wal.New(&wal.Config{
		Filepath:       tmpDir,
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

// TestPartitionWriter_TotalSize verifies TotalSize returns the sum of live trace
// bytes across all tenant stores.
func TestPartitionWriter_TotalSize(t *testing.T) {
	pw := getPartitionWriter(t)

	// Empty writer has zero size.
	require.Equal(t, uint64(0), pw.TotalSize())

	tenant := "test-tenant"
	now := time.Now()
	startTime := uint64(now.UnixNano())
	endTime := uint64(now.Add(time.Second).UnixNano())
	req := test.MakePushBytesRequest(t, 3, nil, startTime, endTime)

	err := pw.pushBytes(now, tenant, req)
	require.NoError(t, err)

	size := pw.TotalSize()
	require.Greater(t, size, uint64(0), "TotalSize must be non-zero after pushing bytes")
}

// TestPartitionWriter_FlushAndReset verifies that FlushAndReset writes blocks and
// then clears liveTraces so TotalSize returns zero afterward.
func TestPartitionWriter_FlushAndReset(t *testing.T) {
	var (
		ctx   = context.Background()
		pw    = getPartitionWriter(t)
		store = newStoreWithLogger(ctx, t, log.NewNopLogger(), false)
	)

	tenant := "test-tenant"
	now := time.Now()
	startTime := uint64(now.UnixNano())
	endTime := uint64(now.Add(time.Second).UnixNano())
	req := test.MakePushBytesRequest(t, 3, nil, startTime, endTime)

	err := pw.pushBytes(now, tenant, req)
	require.NoError(t, err)
	require.Greater(t, pw.TotalSize(), uint64(0))

	// FlushAndReset should write a block and clear liveTraces.
	err = pw.FlushAndReset(ctx, store, store, store)
	require.NoError(t, err)

	require.Equal(t, uint64(0), pw.TotalSize(), "TotalSize must be zero after FlushAndReset")

	store.PollNow(ctx)
	metas := store.BlockMetas(tenant)
	require.Equal(t, 1, len(metas), "one block must have been written")

	// Push more data — writer must still work after reset.
	req2 := test.MakePushBytesRequest(t, 2, nil, startTime, endTime)
	err = pw.pushBytes(now, tenant, req2)
	require.NoError(t, err)
	require.Greater(t, pw.TotalSize(), uint64(0), "TotalSize must be non-zero after pushing to reset writer")
}
