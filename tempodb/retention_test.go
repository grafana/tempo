package tempodb

import (
	"context"
	"path"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
)

func TestRetention(t *testing.T) {
	tempDir := t.TempDir()

	r, w, c, err := New(&Config{
		Backend: backend.Local,
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &common.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              0.01,
			BloomShardSizeBytes:  100_000,
			Version:              encoding.DefaultEncoding().Version(),
			Encoding:             backend.EncLZ4_256k,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	assert.NoError(t, err)

	ctx := context.Background()
	err = c.EnableCompaction(ctx, &CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	r.EnablePolling(ctx, &mockJobSharder{})

	blockID := uuid.New()

	wal := w.WAL()
	assert.NoError(t, err)

	head, err := wal.NewBlock(blockID, testTenantID, model.CurrentEncoding)
	assert.NoError(t, err)

	complete, err := w.CompleteBlock(context.Background(), head)
	assert.NoError(t, err)
	blockID = complete.BlockMeta().BlockID

	rw := r.(*readerWriter)
	// poll
	checkBlocklists(t, blockID, 1, 0, rw)

	// retention should mark it compacted
	r.(*readerWriter).doRetention(ctx)
	checkBlocklists(t, blockID, 0, 1, rw)

	// retention again should clear it
	r.(*readerWriter).doRetention(ctx)
	checkBlocklists(t, blockID, 0, 0, rw)
}

func TestRetentionUpdatesBlocklistImmediately(t *testing.T) {
	// Test that retention updates the in-memory blocklist
	// immediately to reflect affected blocks and doesn't
	// wait for the next polling cycle.

	tempDir := t.TempDir()

	r, w, c, err := New(&Config{
		Backend: backend.Local,
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &common.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              0.01,
			BloomShardSizeBytes:  100_000,
			Version:              encoding.DefaultEncoding().Version(),
			Encoding:             backend.EncLZ4_256k,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	assert.NoError(t, err)

	ctx := context.Background()
	r.EnablePolling(ctx, &mockJobSharder{})

	err = c.EnableCompaction(context.Background(), &CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	wal := w.WAL()
	assert.NoError(t, err)

	blockID := uuid.New()

	head, err := wal.NewBlock(blockID, testTenantID, model.CurrentEncoding)
	assert.NoError(t, err)

	complete, err := w.CompleteBlock(ctx, head)
	assert.NoError(t, err)
	blockID = complete.BlockMeta().BlockID

	// We have a block
	rw := r.(*readerWriter)
	rw.pollBlocklist()
	require.Equal(t, blockID, rw.blocklist.Metas(testTenantID)[0].BlockID)

	// Mark it compacted
	r.(*readerWriter).compactorCfg.BlockRetention = 0 // Immediately delete
	r.(*readerWriter).compactorCfg.CompactedBlockRetention = time.Hour
	r.(*readerWriter).doRetention(ctx)

	// Immediately compacted
	require.Empty(t, rw.blocklist.Metas(testTenantID))
	require.Equal(t, blockID, rw.blocklist.CompactedMetas(testTenantID)[0].BlockID)

	// Now delete it permanently
	r.(*readerWriter).compactorCfg.BlockRetention = time.Hour
	r.(*readerWriter).compactorCfg.CompactedBlockRetention = 0 // Immediately delete
	r.(*readerWriter).doRetention(ctx)

	require.Empty(t, rw.blocklist.Metas(testTenantID))
	require.Empty(t, rw.blocklist.CompactedMetas(testTenantID))
}

func TestBlockRetentionOverride(t *testing.T) {
	tempDir := t.TempDir()

	r, w, c, err := New(&Config{
		Backend: backend.Local,
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &common.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              0.01,
			BloomShardSizeBytes:  100_000,
			Version:              encoding.DefaultEncoding().Version(),
			Encoding:             backend.EncLZ4_256k,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	overrides := &mockOverrides{}

	ctx := context.Background()
	err = c.EnableCompaction(ctx, &CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          time.Hour,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, overrides)
	require.NoError(t, err)

	r.EnablePolling(ctx, &mockJobSharder{})

	cutTestBlocks(t, w, testTenantID, 10, 10)

	// The test spans are all 1 second long, so we have to sleep to put all the
	// data in the past
	time.Sleep(time.Second)

	rw := r.(*readerWriter)
	rw.pollBlocklist()
	require.Equal(t, 10, len(rw.blocklist.Metas(testTenantID)))

	// Retention = 1 hour, does nothing
	overrides.blockRetention = time.Hour
	r.(*readerWriter).doRetention(ctx)
	rw.pollBlocklist()
	require.Equal(t, 10, len(rw.blocklist.Metas(testTenantID)))

	// Retention = 1 minute, still does nothing
	overrides.blockRetention = time.Minute
	r.(*readerWriter).doRetention(ctx)
	rw.pollBlocklist()
	require.Equal(t, 10, len(rw.blocklist.Metas(testTenantID)))

	// Retention = 1ns, deletes everything
	overrides.blockRetention = time.Nanosecond
	r.(*readerWriter).doRetention(ctx)
	rw.pollBlocklist()
	require.Equal(t, 0, len(rw.blocklist.Metas(testTenantID)))
}
