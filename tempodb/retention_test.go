package tempodb

import (
	"context"
	"os"
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
	"github.com/grafana/tempo/tempodb/pool"
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
		BlockRetention:          time.Hour,
		CompactedBlockRetention: time.Hour,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	r.EnablePolling(ctx, &mockJobSharder{})

	blockID := backend.NewUUID()

	wal := w.WAL()
	assert.NoError(t, err)

	meta := &backend.BlockMeta{BlockID: blockID, TenantID: testTenantID}
	head, err := wal.NewBlock(meta, model.CurrentEncoding)
	assert.NoError(t, err)

	complete, err := w.CompleteBlock(context.Background(), head)
	assert.NoError(t, err)
	blockID = complete.BlockMeta().BlockID

	rw := r.(*readerWriter)
	// poll
	checkBlocklists(ctx, t, (uuid.UUID)(blockID), 1, 0, rw)

	// retention should mark it compacted
	rw.compactorCfg.BlockRetention = 0
	r.(*readerWriter).doRetention(ctx)
	checkBlocklists(ctx, t, (uuid.UUID)(blockID), 0, 1, rw)

	// retention again should clear it
	rw.compactorCfg.CompactedBlockRetention = 0
	r.(*readerWriter).doRetention(ctx)
	checkBlocklists(ctx, t, (uuid.UUID)(blockID), 0, 0, rw)
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

	blockID := backend.NewUUID()

	meta := &backend.BlockMeta{BlockID: blockID, TenantID: testTenantID}
	head, err := wal.NewBlock(meta, model.CurrentEncoding)
	assert.NoError(t, err)

	complete, err := w.CompleteBlock(ctx, head)
	assert.NoError(t, err)
	blockID = complete.BlockMeta().BlockID

	// We have a block
	rw := r.(*readerWriter)
	rw.pollBlocklist(ctx)
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
	rw.pollBlocklist(ctx)
	require.Equal(t, 10, len(rw.blocklist.Metas(testTenantID)))

	// Retention = 1 hour, does nothing
	overrides.blockRetention = time.Hour
	r.(*readerWriter).doRetention(ctx)
	rw.pollBlocklist(ctx)
	require.Equal(t, 10, len(rw.blocklist.Metas(testTenantID)))

	// Retention = 1 minute, still does nothing
	overrides.blockRetention = time.Minute
	r.(*readerWriter).doRetention(ctx)
	rw.pollBlocklist(ctx)
	require.Equal(t, 10, len(rw.blocklist.Metas(testTenantID)))

	// Retention = 1ns, deletes everything
	overrides.blockRetention = time.Nanosecond
	r.(*readerWriter).doRetention(ctx)
	rw.pollBlocklist(ctx)
	require.Equal(t, 0, len(rw.blocklist.Metas(testTenantID)))
}

func TestBlockRetentionOverrideDisabled(t *testing.T) {
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

	overrides := &mockOverrides{
		disabled: true,
	}

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
	rw.pollBlocklist(ctx)
	require.Equal(t, 10, len(rw.blocklist.Metas(testTenantID)))

	time.Sleep(10 * time.Millisecond)

	// Retention = 1ns, deletes everything
	overrides.blockRetention = time.Nanosecond
	r.(*readerWriter).doRetention(ctx)
	rw.pollBlocklist(ctx)
	require.Equal(t, 10, len(rw.blocklist.Metas(testTenantID)))
}

func TestRetainWithConfig(t *testing.T) {
	for _, enc := range encoding.AllEncodings() {
		version := enc.Version()
		t.Run(version, func(t *testing.T) {
			testRetainWithConfig(t, version)
		})
	}
}

func testRetainWithConfig(t *testing.T, targetBlockVersion string) {
	tempDir := t.TempDir()

	logger := log.NewLogfmtLogger(os.Stderr)

	r, w, c, err := New(
		&Config{
			Backend: backend.Local,
			Pool: &pool.Config{
				MaxWorkers: 10,
				QueueDepth: 100,
			},
			Local: &local.Config{
				Path: path.Join(tempDir, "traces"),
			},
			Block: &common.BlockConfig{
				IndexDownsampleBytes: 11,
				BloomFP:              .01,
				BloomShardSizeBytes:  100_000,
				Version:              targetBlockVersion,
				Encoding:             backend.EncNone,
				IndexPageSizeBytes:   1000,
			},
			WAL: &wal.Config{
				Filepath: path.Join(tempDir, "wal"),
			},
			BlocklistPoll: 100 * time.Millisecond,
		}, nil,
		// log.NewNopLogger()
		logger,
	)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r.EnablePolling(ctx, &mockJobSharder{})

	blocks := cutTestBlocks(t, w, testTenantID, 10, 10)

	metas := make([]*backend.BlockMeta, 0)
	for _, b := range blocks {
		metas = append(metas, b.BlockMeta())
	}

	time.Sleep(time.Second)

	compactorCfg := &CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          time.Nanosecond,
		CompactedBlockRetention: time.Nanosecond,
		MaxCompactionObjects:    1000,
		MaxBlockBytes:           100_000_000, // Needs to be sized appropriately for the test data
		RetentionConcurrency:    1,
	}

	var (
		rw    = r.(*readerWriter)
		preM  = rw.blocklist.Metas(testTenantID)
		preCm = rw.blocklist.CompactedMetas(testTenantID)
	)

	require.Len(t, preM, 10)
	require.Len(t, preCm, 0)

	_, err = c.CompactWithConfig(ctx, metas, testTenantID, compactorCfg, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	var (
		postM  = rw.blocklist.Metas(testTenantID)
		postCm = rw.blocklist.CompactedMetas(testTenantID)
	)

	require.Less(t, len(postM), len(preM))
	require.Greater(t, len(postCm), len(preCm))

	require.Len(t, rw.blocklist.Metas(testTenantID), 1)
	require.Len(t, rw.blocklist.CompactedMetas(testTenantID), 10)

	c.RetainWithConfig(ctx,
		compactorCfg,
		&mockSharder{},
		&mockOverrides{},
	)

	time.Sleep(100 * time.Millisecond)

	require.Empty(t, rw.blocklist.Metas(testTenantID))
	require.Empty(t, rw.blocklist.CompactedMetas(testTenantID))
}
