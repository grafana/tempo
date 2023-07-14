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

	db, err := New(&Config{
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
	}, log.NewNopLogger())
	assert.NoError(t, err)

	ctx := context.Background()
	err = db.EnableCompaction(ctx, &CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	db.EnablePolling(ctx, &mockJobSharder{})

	blockID := uuid.New()

	wal := db.WAL()
	assert.NoError(t, err)

	head, err := wal.NewBlock(blockID, testTenantID, model.CurrentEncoding)
	assert.NoError(t, err)

	complete, err := db.CompleteBlock(ctx, head)
	assert.NoError(t, err)
	blockID = complete.BlockMeta().BlockID

	// poll
	checkBlocklists(t, blockID, 1, 0, db)

	// retention should mark it compacted
	db.doRetention(ctx)
	checkBlocklists(t, blockID, 0, 1, db)

	// retention again should clear it
	db.doRetention(ctx)
	checkBlocklists(t, blockID, 0, 0, db)
}

func TestRetentionUpdatesBlocklistImmediately(t *testing.T) {
	// Test that retention updates the in-memory blocklist
	// immediately to reflect affected blocks and doesn't
	// wait for the next polling cycle.

	tempDir := t.TempDir()

	db, err := New(&Config{
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
	}, log.NewNopLogger())
	assert.NoError(t, err)

	ctx := context.Background()
	db.EnablePolling(ctx, &mockJobSharder{})

	err = db.EnableCompaction(ctx, &CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	wal := db.WAL()
	assert.NoError(t, err)

	blockID := uuid.New()

	head, err := wal.NewBlock(blockID, testTenantID, model.CurrentEncoding)
	assert.NoError(t, err)

	complete, err := db.CompleteBlock(ctx, head)
	assert.NoError(t, err)
	blockID = complete.BlockMeta().BlockID

	// We have a block
	db.pollBlocklist(ctx)
	require.Equal(t, blockID, db.blocklist.Metas(testTenantID)[0].BlockID)

	// Mark it compacted
	db.compactorCfg.BlockRetention = 0 // Immediately delete
	db.compactorCfg.CompactedBlockRetention = time.Hour
	db.doRetention(ctx)

	// Immediately compacted
	require.Empty(t, db.blocklist.Metas(testTenantID))
	require.Equal(t, blockID, db.blocklist.CompactedMetas(testTenantID)[0].BlockID)

	// Now delete it permanently
	db.compactorCfg.BlockRetention = time.Hour
	db.compactorCfg.CompactedBlockRetention = 0 // Immediately delete
	db.doRetention(ctx)

	require.Empty(t, db.blocklist.Metas(testTenantID))
	require.Empty(t, db.blocklist.CompactedMetas(testTenantID))
}

func TestBlockRetentionOverride(t *testing.T) {
	tempDir := t.TempDir()

	db, err := New(&Config{
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
	}, log.NewNopLogger())
	require.NoError(t, err)

	overrides := &mockOverrides{}

	ctx := context.Background()
	err = db.EnableCompaction(ctx, &CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          time.Hour,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, overrides)
	require.NoError(t, err)

	db.EnablePolling(ctx, &mockJobSharder{})

	cutTestBlocks(ctx, t, db, testTenantID, 10, 10)

	// The test spans are all 1 second long, so we have to sleep to put all the
	// data in the past
	time.Sleep(time.Second)

	t.Logf("blocklist: %+v", db.blocklist)

	db.pollBlocklist(ctx)
	require.Equal(t, 10, len(db.blocklist.Metas(testTenantID)))

	// Retention = 1 hour, does nothing
	overrides.blockRetention = time.Hour
	db.doRetention(ctx)
	db.pollBlocklist(ctx)
	require.Equal(t, 10, len(db.blocklist.Metas(testTenantID)))

	// Retention = 1 minute, still does nothing
	overrides.blockRetention = time.Minute
	db.doRetention(ctx)
	db.pollBlocklist(ctx)
	require.Equal(t, 10, len(db.blocklist.Metas(testTenantID)))

	// Retention = 1ns, deletes everything
	overrides.blockRetention = time.Nanosecond
	db.doRetention(ctx)
	db.pollBlocklist(ctx)
	require.Equal(t, 0, len(db.blocklist.Metas(testTenantID)))
}
