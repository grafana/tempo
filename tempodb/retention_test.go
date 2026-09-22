package tempodb

import (
	"context"
	"os"
	"path"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	prom_testutil "github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/pkg/model"
	testutil "github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	backend_cache "github.com/grafana/tempo/tempodb/backend/cache"
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
			BloomFP:             0.01,
			BloomShardSizeBytes: 100_000,
			Version:             encoding.DefaultEncoding().Version(),
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	assert.NoError(t, err)

	ctx := context.Background()
	err = c.EnableCompaction(ctx, &CompactorConfig{
		MaxCompactionRange:      time.Hour,
		BlockRetention:          time.Hour,
		CompactedBlockRetention: time.Hour,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	r.EnablePolling(ctx, &mockJobSharder{}, false)

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
	checkBlocklists(ctx, t, uuid.UUID(blockID), 1, 0, rw)

	// retention should mark it compacted
	rw.compactorCfg.BlockRetention = 0
	r.(*readerWriter).doRetention(ctx)
	checkBlocklists(ctx, t, uuid.UUID(blockID), 0, 1, rw)

	// retention again should clear it
	rw.compactorCfg.CompactedBlockRetention = 0
	r.(*readerWriter).doRetention(ctx)
	checkBlocklists(ctx, t, uuid.UUID(blockID), 0, 0, rw)
}

func TestRetentionSkipsBlockAlreadyMarkedCompacted(t *testing.T) {
	// A concurrent compaction pass (or an earlier, still-in-flight retention
	// attempt) can mark a block compacted on the backend before this retention
	// pass's own, separately-polled local blocklist snapshot has caught up.
	// That should not be logged/counted as a retention error.
	tempDir := t.TempDir()

	r, w, c, err := New(&Config{
		Backend: backend.Local,
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &common.BlockConfig{
			BloomFP:             0.01,
			BloomShardSizeBytes: 100_000,
			Version:             encoding.DefaultEncoding().Version(),
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	ctx := context.Background()
	err = c.EnableCompaction(ctx, &CompactorConfig{
		MaxCompactionRange:      time.Hour,
		BlockRetention:          time.Hour,
		CompactedBlockRetention: time.Hour,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	r.EnablePolling(ctx, &mockJobSharder{}, false)

	blockID := backend.NewUUID()
	meta := &backend.BlockMeta{BlockID: blockID, TenantID: testTenantID}
	head, err := w.WAL().NewBlock(meta, model.CurrentEncoding)
	require.NoError(t, err)

	complete, err := w.CompleteBlock(ctx, head)
	require.NoError(t, err)
	blockID = complete.BlockMeta().BlockID

	rw := r.(*readerWriter)
	checkBlocklists(ctx, t, uuid.UUID(blockID), 1, 0, rw)

	// Simulate the race directly on the backend, bypassing retention's own
	// in-memory bookkeeping - this leaves rw.blocklist still believing the
	// block is active.
	require.NoError(t, rw.c.MarkBlockCompacted(uuid.UUID(blockID), testTenantID))

	before := prom_testutil.ToFloat64(metricRetentionErrors)

	rw.compactorCfg.BlockRetention = 0
	rw.doRetention(ctx)

	require.Equal(t, before, prom_testutil.ToFloat64(metricRetentionErrors),
		"a block already retired by someone else must not count as a retention error")

	// The next real poll reconciles our stale local view with backend reality.
	rw.pollBlocklist(ctx)
	checkBlocklists(ctx, t, uuid.UUID(blockID), 0, 1, rw)
}

func TestRetentionSkipsBlockAlreadyCleared(t *testing.T) {
	// Symmetric to TestRetentionSkipsBlockAlreadyMarkedCompacted, but for the
	// second retention loop (ClearBlock). Here the goal state - no data, no
	// longer tracked - is unambiguous, so it should be treated the same as a
	// successful clear rather than merely skipped.
	tempDir := t.TempDir()

	r, w, c, err := New(&Config{
		Backend: backend.Local,
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &common.BlockConfig{
			BloomFP:             0.01,
			BloomShardSizeBytes: 100_000,
			Version:             encoding.DefaultEncoding().Version(),
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	ctx := context.Background()
	err = c.EnableCompaction(ctx, &CompactorConfig{
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: time.Hour,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	r.EnablePolling(ctx, &mockJobSharder{}, false)

	blockID := backend.NewUUID()
	meta := &backend.BlockMeta{BlockID: blockID, TenantID: testTenantID}
	head, err := w.WAL().NewBlock(meta, model.CurrentEncoding)
	require.NoError(t, err)

	complete, err := w.CompleteBlock(ctx, head)
	require.NoError(t, err)
	blockID = complete.BlockMeta().BlockID

	rw := r.(*readerWriter)
	checkBlocklists(ctx, t, uuid.UUID(blockID), 1, 0, rw)

	// Mark it compacted for real, updating both the backend and the local view.
	rw.doRetention(ctx)
	checkBlocklists(ctx, t, uuid.UUID(blockID), 0, 1, rw)

	// Simulate a concurrent retention pass clearing the block on the backend
	// first: force ClearBlock to report ErrDoesNotExist for this block,
	// exercising retention.go's tolerance directly rather than relying on any
	// particular backend's incidental delete-on-missing idempotency (the local
	// backend's ClearBlock already no-ops on a missing path via os.RemoveAll,
	// which would make this assertion pass even without the fix).
	realCompactor := rw.c
	rw.c = &backend.MockCompactor{
		ClearBlockFn: func(id uuid.UUID, tenantID string) error {
			if id == uuid.UUID(blockID) {
				// Actually remove it so a later real poll reflects reality, but
				// report the race error retention.go must tolerate.
				_ = realCompactor.ClearBlock(id, tenantID)
				return backend.ErrDoesNotExist
			}
			return realCompactor.ClearBlock(id, tenantID)
		},
	}

	beforeErrors := prom_testutil.ToFloat64(metricRetentionErrors)
	beforeDeleted := prom_testutil.ToFloat64(metricDeleted)

	rw.compactorCfg.CompactedBlockRetention = 0
	rw.doRetention(ctx)

	require.Equal(t, beforeErrors, prom_testutil.ToFloat64(metricRetentionErrors),
		"a block already cleared by someone else must not count as a retention error")
	require.Equal(t, beforeDeleted+1, prom_testutil.ToFloat64(metricDeleted),
		"clearing an already-gone block still reaches the goal state and should count as deleted")
	checkBlocklists(ctx, t, uuid.UUID(blockID), 0, 0, rw)
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
			BloomFP:             0.01,
			BloomShardSizeBytes: 100_000,
			Version:             encoding.DefaultEncoding().Version(),
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	assert.NoError(t, err)

	ctx := context.Background()
	r.EnablePolling(ctx, &mockJobSharder{}, false)

	err = c.EnableCompaction(context.Background(), &CompactorConfig{
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
			BloomFP:             0.01,
			BloomShardSizeBytes: 100_000,
			Version:             encoding.DefaultEncoding().Version(),
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
		MaxCompactionRange:      time.Hour,
		BlockRetention:          time.Hour,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, overrides)
	require.NoError(t, err)

	r.EnablePolling(ctx, &mockJobSharder{}, false)

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
			BloomFP:             0.01,
			BloomShardSizeBytes: 100_000,
			Version:             encoding.DefaultEncoding().Version(),
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
		MaxCompactionRange:      time.Hour,
		BlockRetention:          time.Hour,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, overrides)
	require.NoError(t, err)

	r.EnablePolling(ctx, &mockJobSharder{}, false)

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
	for _, enc := range encoding.AllEncodingsForWrites() {
		t.Run(enc.Version(), func(t *testing.T) {
			testRetainWithConfig(t, enc.Version())
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
				BloomFP:             .01,
				BloomShardSizeBytes: 100_000,
				Version:             targetBlockVersion,
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

	r.EnablePolling(ctx, &mockJobSharder{}, false)

	blocks := cutTestBlocks(t, w, testTenantID, 10, 10)

	metas := make([]*backend.BlockMeta, 0)
	for _, b := range blocks {
		metas = append(metas, b.BlockMeta())
	}

	time.Sleep(time.Second)

	compactorCfg := &CompactorConfig{
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

	c.RetainWithConfig(
		ctx,
		compactorCfg,
		&mockSharder{},
		&mockOverrides{},
	)

	time.Sleep(100 * time.Millisecond)

	require.Empty(t, rw.blocklist.Metas(testTenantID))
	require.Empty(t, rw.blocklist.CompactedMetas(testTenantID))
}

// perRoleMockProvider wraps a separate mock cache per role so tests can verify
// that eviction targets the correct role-specific cache and not a shared one.
type perRoleMockProvider struct {
	services.Service
	caches map[cache.Role]cache.Cache
}

func newPerRoleMockProvider(roles ...cache.Role) *perRoleMockProvider {
	p := &perRoleMockProvider{caches: make(map[cache.Role]cache.Cache, len(roles))}
	for _, r := range roles {
		p.caches[r] = testutil.NewMockClient()
	}
	return p
}

func (p *perRoleMockProvider) CacheFor(role cache.Role) cache.Cache {
	return p.caches[role]
}

func (p *perRoleMockProvider) AddCache(role cache.Role, c cache.Cache) error {
	p.caches[role] = c
	return nil
}

func TestRetentionCacheEviction(t *testing.T) {
	tempDir := t.TempDir()

	provider := newPerRoleMockProvider(cache.RoleBloom, cache.RoleTraceIDIdx)

	r, w, c, err := New(&Config{
		Backend: backend.Local,
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &common.BlockConfig{
			BloomFP:             0.01,
			BloomShardSizeBytes: 100_000,
			Version:             encoding.DefaultEncoding().Version(),
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, provider, log.NewNopLogger())
	require.NoError(t, err)

	ctx := context.Background()
	err = c.EnableCompaction(ctx, &CompactorConfig{
		MaxCompactionRange:      time.Hour,
		BlockRetention:          time.Hour,
		CompactedBlockRetention: time.Hour,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	r.EnablePolling(ctx, &mockJobSharder{}, false)

	// Create and flush a block
	head, err := w.WAL().NewBlock(
		&backend.BlockMeta{BlockID: backend.NewUUID(), TenantID: testTenantID},
		model.CurrentEncoding,
	)
	require.NoError(t, err)

	complete, err := w.CompleteBlock(ctx, head)
	require.NoError(t, err)
	blockMeta := complete.BlockMeta()

	rw := r.(*readerWriter)
	rw.pollBlocklist(ctx)
	require.Len(t, rw.blocklist.Metas(testTenantID), 1)

	// Prime each role's cache separately so a wrong-role eviction is detectable.
	bloomCache := provider.CacheFor(cache.RoleBloom)
	idxCache := provider.CacheFor(cache.RoleTraceIDIdx)
	keyPrefix := backend_cache.BlockKeyPrefix(uuid.UUID(blockMeta.BlockID), testTenantID)
	bloomKey := keyPrefix + common.BloomName(0)
	idxKey := keyPrefix + common.NameIndex
	bloomCache.Store(ctx, []string{bloomKey}, [][]byte{{1}})
	idxCache.Store(ctx, []string{idxKey}, [][]byte{{2}})

	_, found := bloomCache.FetchKey(ctx, bloomKey)
	require.True(t, found, "bloom key should be in bloom cache before retention")
	_, found = idxCache.FetchKey(ctx, idxKey)
	require.True(t, found, "index key should be in trace-id-index cache before retention")

	// Mark compacted — cache should not be touched at this stage
	rw.compactorCfg.BlockRetention = 0
	rw.compactorCfg.CompactedBlockRetention = time.Hour
	rw.doRetention(ctx)
	checkBlocklists(ctx, t, uuid.UUID(blockMeta.BlockID), 0, 1, rw)

	_, found = bloomCache.FetchKey(ctx, bloomKey)
	require.True(t, found, "bloom key should remain cached after mark-compacted")

	// Delete the compacted block — each role's cache entry should be evicted
	rw.compactorCfg.CompactedBlockRetention = 0
	rw.doRetention(ctx)
	checkBlocklists(ctx, t, uuid.UUID(blockMeta.BlockID), 0, 0, rw)

	_, found = bloomCache.FetchKey(ctx, bloomKey)
	require.False(t, found, "bloom key should be evicted from bloom cache after block deletion")
	_, found = idxCache.FetchKey(ctx, idxKey)
	require.False(t, found, "index key should be evicted from trace-id-index cache after block deletion")
}

func TestRetentionClearsEveryBlockConcurrently(t *testing.T) {
	// Every other retention test clears a single block, so nothing covers the
	// concurrent path.
	const numBlocks = 8

	tempDir := t.TempDir()

	r, w, c, err := New(&Config{
		Backend: backend.Local,
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &common.BlockConfig{
			BloomFP:             0.01,
			BloomShardSizeBytes: 100_000,
			Version:             encoding.DefaultEncoding().Version(),
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	ctx := context.Background()
	r.EnablePolling(ctx, &mockJobSharder{}, false)

	require.NoError(t, c.EnableCompaction(ctx, &CompactorConfig{
		MaxCompactionRange:        time.Hour,
		BlockRetention:            0,
		CompactedBlockRetention:   0,
		RetentionBlockConcurrency: 4,
	}, &mockSharder{}, &mockOverrides{}))

	for i := 0; i < numBlocks; i++ {
		head, err := w.WAL().NewBlock(&backend.BlockMeta{BlockID: backend.NewUUID(), TenantID: testTenantID}, model.CurrentEncoding)
		require.NoError(t, err)

		_, err = w.CompleteBlock(ctx, head)
		require.NoError(t, err)
	}

	rw := r.(*readerWriter)
	rw.pollBlocklist(ctx)
	require.Len(t, rw.blocklist.Metas(testTenantID), numBlocks)

	rw.compactorCfg.BlockRetention = 0
	rw.compactorCfg.CompactedBlockRetention = time.Hour
	rw.doRetention(ctx)
	require.Empty(t, rw.blocklist.Metas(testTenantID))
	require.Len(t, rw.blocklist.CompactedMetas(testTenantID), numBlocks)

	rw.compactorCfg.BlockRetention = time.Hour
	rw.compactorCfg.CompactedBlockRetention = 0
	rw.doRetention(ctx)

	require.Empty(t, rw.blocklist.Metas(testTenantID))
	require.Empty(t, rw.blocklist.CompactedMetas(testTenantID), "every compacted block must be cleared")
}
