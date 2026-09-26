package tempodb

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"
	"path"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/golang/protobuf/proto" //nolint:all
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/model"
	modeltrace "github.com/grafana/tempo/v3/pkg/model/trace"
	v1 "github.com/grafana/tempo/v3/pkg/model/v1"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/pkg/util/test"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/backend/local"
	"github.com/grafana/tempo/v3/tempodb/blocklist"
	"github.com/grafana/tempo/v3/tempodb/encoding"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
	"github.com/grafana/tempo/v3/tempodb/pool"
	"github.com/grafana/tempo/v3/tempodb/wal"
)

type mockSharder struct{}

func (m *mockSharder) Owns(string) bool {
	return true
}

func (m *mockSharder) RecordDiscardedSpans(int, string, string, string, string) {}

type mockJobSharder struct{}

func (m *mockJobSharder) Owns(string) bool { return true }

type mockOverrides struct {
	blockRetention      time.Duration
	disabled            bool
	maxBytesPerTrace    int
	maxCompactionWindow time.Duration
}

func (m *mockOverrides) BlockRetentionForTenant(_ string) time.Duration {
	return m.blockRetention
}

func (m *mockOverrides) CompactionDisabledForTenant(_ string) bool {
	return m.disabled
}

func (m *mockOverrides) MaxBytesPerTraceForTenant(_ string) int {
	return m.maxBytesPerTrace
}

func (m *mockOverrides) MaxCompactionRangeForTenant(_ string) time.Duration {
	return m.maxCompactionWindow
}

func TestCompactionDropsTraces(t *testing.T) {
	for _, enc := range encoding.AllEncodingsForWrites() {
		t.Run(enc.Version(), func(t *testing.T) {
			t.Parallel()
			testCompactionDropsTraces(t, enc.Version())
		})
	}
}

func testCompactionDropsTraces(t *testing.T, targetBlockVersion string) {
	tempDir := t.TempDir()

	r, w, _, err := New(&Config{
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
			RowGroupSizeBytes:   30_000_000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	wal := w.WAL()
	require.NoError(t, err)

	dec := model.MustNewSegmentDecoder(v1.Encoding)

	recordCount := 100
	allIDs := make([]common.ID, 0, recordCount)

	// write a bunch of dummy data
	blockID := backend.NewUUID()
	meta := &backend.BlockMeta{BlockID: blockID, TenantID: testTenantID}
	head, err := wal.NewBlock(meta, v1.Encoding)
	require.NoError(t, err)

	for j := 0; j < recordCount; j++ {
		id := test.ValidTraceID(nil)
		allIDs = append(allIDs, id)

		obj, err := dec.PrepareForWrite(test.MakeTrace(1, id), 0, 0)
		require.NoError(t, err)

		obj2, err := dec.ToObject([][]byte{obj})
		require.NoError(t, err)

		err = head.Append(id, obj2, 0, 0, true)
		require.NoError(t, err, "unexpected error writing req")
	}

	firstBlock, err := w.CompleteBlock(context.Background(), head)
	require.NoError(t, err)

	// choose a random id to drop
	dropID := allIDs[rand.Intn(len(allIDs))]

	rw := r.(*readerWriter)
	// force compact to a new block
	opts := common.CompactionOptions{
		BlockConfig:      *rw.cfg.Block,
		OutputBlocks:     1,
		MaxBytesPerTrace: 0,

		// hook to drop the trace
		DropObject: func(id common.ID) bool {
			return bytes.Equal(id, dropID)
		},

		// setting to prevent panics.
		BytesWritten:      func(_, _ int) {},
		ObjectsCombined:   func(_, _ int) {},
		ObjectsWritten:    func(_, _ int) {},
		SpansDiscarded:    func(_, _, _ string, _ int) {},
		DisconnectedTrace: func() {},
		RootlessTrace:     func() {},
	}

	enc, err := encoding.FromVersion(targetBlockVersion)
	require.NoError(t, err)

	compactor := enc.NewCompactor(opts)
	newMetas, err := compactor.Compact(context.Background(), log.NewNopLogger(), rw.r, rw.w, []*backend.BlockMeta{firstBlock.BlockMeta()})
	require.NoError(t, err)

	// require new meta has len 1
	require.Len(t, newMetas, 1)

	secondBlock, err := enc.OpenBlock(newMetas[0], rw.r)
	require.NoError(t, err)

	// search for all ids. confirm they all return except the dropped one
	for _, id := range allIDs {
		tr, err := secondBlock.FindTraceByID(context.Background(), id, common.DefaultSearchOptions())
		require.NoError(t, err)

		if bytes.Equal(id, dropID) {
			require.Nil(t, tr)
		} else {
			require.NotNil(t, tr)
		}
	}
}

func TestCompactWithConfig(t *testing.T) {
	for _, enc := range encoding.AllEncodingsForWrites() {
		t.Run(enc.Version(), func(t *testing.T) {
			t.Parallel()
			testCompactWithConfig(t, enc.Version())
		})
	}
}

func testCompactWithConfig(t *testing.T, targetBlockVersion string) {
	tempDir := t.TempDir()

	_, w, c, err := New(&Config{
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
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	ctx := context.Background()

	blocks := cutTestBlocks(t, w, testTenantID, 10, 10)
	metas := make([]*backend.BlockMeta, 0)
	for _, b := range blocks {
		metas = append(metas, b.BlockMeta())
	}

	_, err = c.CompactWithConfig(
		ctx,
		metas,
		testTenantID,
		&CompactorConfig{
			MaxCompactionRange:      24 * time.Hour,
			BlockRetention:          0,
			CompactedBlockRetention: 0,
			MaxCompactionObjects:    1000,
			MaxBlockBytes:           100_000_000, // Needs to be sized appropriately for the test data
		},
		&mockSharder{},
		&mockOverrides{},
	)
	require.NoError(t, err)
}

func TestCompactWithConfigSkipsMissingBlocks(t *testing.T) {
	for _, enc := range encoding.AllEncodingsForWrites() {
		t.Run(enc.Version(), func(t *testing.T) {
			t.Parallel()
			testCompactWithConfigSkipsMissingBlocks(t, enc.Version())
		})
	}
}

func testCompactWithConfigSkipsMissingBlocks(t *testing.T, targetBlockVersion string) {
	tenantID := "missing-blocks-" + targetBlockVersion

	_, w, c, _ := testConfig(t, 0, func(cfg *Config) {
		cfg.Block.Version = targetBlockVersion
	})

	ctx := context.Background()
	compactorCfg := &CompactorConfig{
		MaxCompactionRange:   24 * time.Hour,
		MaxCompactionObjects: 1000,
		MaxBlockBytes:        100_000_000,
	}

	// A meta the block list still carries but whose block is gone from the backend:
	// the race a caller hits when it selects blocks from a stale block list. It is
	// placed first so the version lookup cannot depend on it.
	missing := &backend.BlockMeta{
		BlockID:  backend.NewUUID(),
		TenantID: tenantID,
		Version:  targetBlockVersion,
	}

	blocks := cutTestBlocks(t, w, tenantID, 3, 10)
	metas := []*backend.BlockMeta{missing}
	for _, b := range blocks {
		metas = append(metas, b.BlockMeta())
	}

	before := testutil.ToFloat64(metricCompactionBlocksMissing.WithLabelValues(tenantID))

	compacted, err := c.CompactWithConfig(ctx, metas, tenantID, compactorCfg, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)
	require.NotEmpty(t, compacted, "blocks that still exist should be compacted")
	require.Equal(t, before+1, testutil.ToFloat64(metricCompactionBlocksMissing.WithLabelValues(tenantID)))

	// A lone survivor has nothing to combine with: a no-op success, not a failure.
	single := cutTestBlocks(t, w, tenantID, 1, 10)
	compacted, err = c.CompactWithConfig(ctx,
		[]*backend.BlockMeta{missing, single[0].BlockMeta()},
		tenantID, compactorCfg, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)
	require.Empty(t, compacted)
}

func TestCompactWithConfigFailsOnUnreadableMeta(t *testing.T) {
	tenantID := "unreadable-meta"

	_, w, c, tempDir := testConfig(t, 0)

	ctx := context.Background()
	compactorCfg := &CompactorConfig{
		MaxCompactionRange:   24 * time.Hour,
		MaxCompactionObjects: 1000,
		MaxBlockBytes:        100_000_000,
	}

	// A meta that is present but cannot be read is not the already-compacted race, so
	// it must still fail the compaction instead of being skipped.
	_, rawW, _, err := local.New(&local.Config{Path: path.Join(tempDir, "traces")})
	require.NoError(t, err)

	unreadable := backend.NewUUID()
	garbage := []byte("not a block meta")
	err = backend.NewWriter(rawW).Write(ctx, backend.MetaName, uuid.UUID(unreadable), tenantID, garbage, nil)
	require.NoError(t, err)

	blocks := cutTestBlocks(t, w, tenantID, 2, 10)
	metas := []*backend.BlockMeta{{
		BlockID:  unreadable,
		TenantID: tenantID,
		Version:  encoding.DefaultEncoding().Version(),
	}}
	for _, b := range blocks {
		metas = append(metas, b.BlockMeta())
	}

	before := testutil.ToFloat64(metricCompactionBlocksMissing.WithLabelValues(tenantID))

	compacted, err := c.CompactWithConfig(ctx, metas, tenantID, compactorCfg, &mockSharder{}, &mockOverrides{})
	require.Error(t, err)
	require.NotErrorIs(t, err, backend.ErrDoesNotExist)
	require.Empty(t, compacted)
	require.Equal(t, before, testutil.ToFloat64(metricCompactionBlocksMissing.WithLabelValues(tenantID)),
		"an unreadable meta must not be counted as a missing block")
}

func TestMarkCompactedSkipsAlreadyRetiredBlock(t *testing.T) {
	// markCompacted retires each old block via MarkBlockCompacted after a
	// successful merge. retention's own, independent age-based sweep can reach
	// the same block first (or a duplicate compaction job can be dispatched
	// against a stale block list). Either way, finding the block already
	// retired is not a real compaction error.
	tenantID := "already-retired"

	_, w, c, _ := testConfig(t, 0)
	rw := c.(*readerWriter)

	blocks := cutTestBlocks(t, w, tenantID, 1, 10)
	oldMeta := blocks[0].BlockMeta()

	// Simulate a concurrent retention pass that already retired this block on
	// the backend, bypassing markCompacted's own in-memory bookkeeping.
	require.NoError(t, rw.c.MarkBlockCompacted(uuid.UUID(oldMeta.BlockID), tenantID))

	before := testutil.ToFloat64(metricCompactionErrors)

	newMeta := &backend.BlockMeta{BlockID: backend.NewUUID(), TenantID: tenantID}
	err := markCompacted(rw, tenantID, []*backend.BlockMeta{oldMeta}, []*backend.BlockMeta{newMeta})
	require.NoError(t, err)
	require.Equal(t, before, testutil.ToFloat64(metricCompactionErrors),
		"a block already retired by someone else must not count as a compaction error")
}

type testData struct {
	id         common.ID
	t          *tempopb.Trace
	start, end uint32
}

func cutTestBlockWithTraces(t testing.TB, w Writer, data []testData) common.BackendBlock {
	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	wal := w.WAL()

	meta := &backend.BlockMeta{BlockID: backend.NewUUID(), TenantID: testTenantID}
	head, err := wal.NewBlock(meta, model.CurrentEncoding)
	require.NoError(t, err)

	for _, d := range data {
		writeTraceToWal(t, head, dec, d.id, d.t, d.start, d.end)
	}

	b, err := w.CompleteBlock(context.Background(), head)
	require.NoError(t, err)

	return b
}

func cutTestBlocks(t testing.TB, w Writer, tenantID string, blockCount int, recordCount int) []common.BackendBlock {
	blocks := make([]common.BackendBlock, 0)
	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	wal := w.WAL()
	for i := 0; i < blockCount; i++ {
		meta := &backend.BlockMeta{BlockID: backend.NewUUID(), TenantID: tenantID}
		head, err := wal.NewBlock(meta, model.CurrentEncoding)
		require.NoError(t, err)

		for j := 0; j < recordCount; j++ {
			id := makeTraceID(i, j)
			tr := test.MakeTrace(1, id)
			now := uint32(time.Now().Unix())
			writeTraceToWal(t, head, dec, id, tr, now, now)
		}

		b, err := w.CompleteBlock(context.Background(), head)
		require.NoError(t, err)
		blocks = append(blocks, b)
	}

	return blocks
}

func makeTraceID(i int, j int) []byte {
	id := make([]byte, 16)
	binary.LittleEndian.PutUint64(id, uint64(i))
	binary.LittleEndian.PutUint64(id[8:], uint64(j))
	return id
}

func TestCompactionRoundtrip(t *testing.T) {
	for _, enc := range encoding.AllEncodingsForWrites() {
		version := enc.Version()
		t.Run(version, func(t *testing.T) {
			t.Parallel()
			testCompactionRoundtrip(t, version)
		})
	}
}

func testCompactionRoundtrip(t *testing.T, targetBlockVersion string) {
	tempDir := t.TempDir()

	r, w, c, err := New(&Config{
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
			RowGroupSizeBytes:   30_000_000,
			DedicatedColumns:    backend.DedicatedColumns{{Scope: "span", Name: "key", Type: "string"}},
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	ctx := context.Background()
	r.EnablePolling(ctx, &mockJobSharder{}, false)

	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	blockCount := 4
	recordCount := 50

	allReqs := make([]*tempopb.Trace, 0, blockCount*recordCount)
	allIDs := make([]common.ID, 0, blockCount*recordCount)

	for i := 0; i < blockCount; i++ {
		blockID := backend.NewUUID()
		meta := &backend.BlockMeta{BlockID: blockID, TenantID: testTenantID}
		head, err := w.WAL().NewBlock(meta, model.CurrentEncoding)
		require.NoError(t, err)

		for j := 0; j < recordCount; j++ {
			id := test.ValidTraceID(nil)
			req := test.MakeTrace(10, id)
			now := uint32(time.Now().Unix())
			writeTraceToWal(t, head, dec, id, req, now, now)
			allReqs = append(allReqs, req)
			allIDs = append(allIDs, id)
		}

		_, err = w.CompleteBlock(ctx, head)
		require.NoError(t, err)
	}

	rw := r.(*readerWriter)
	checkBlocklists(ctx, t, uuid.Nil, blockCount, 0, rw)

	metas := rw.blocklist.Metas(testTenantID)
	cfg := &CompactorConfig{
		MaxCompactionRange:   24 * time.Hour,
		MaxCompactionObjects: 1000,
		MaxBlockBytes:        100_000_000,
	}

	_, err = c.CompactWithConfig(ctx, metas, testTenantID, cfg, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	checkBlocklists(ctx, t, uuid.Nil, 1, blockCount, rw)

	// verify total object count is preserved
	var totalObjects int64
	for _, m := range rw.blocklist.Metas(testTenantID) {
		totalObjects += m.TotalObjects
	}
	require.Equal(t, int64(blockCount*recordCount), totalObjects)

	// verify all traces are findable with correct content
	for i, id := range allIDs {
		t.Run(fmt.Sprintf("trace-%d", i), func(t *testing.T) {
			trs, failedBlocks, err := r.Find(ctx, testTenantID, id, BlockIDMin, BlockIDMax, time.Time{}, time.Time{}, common.DefaultSearchOptions())
			require.NoError(t, err)
			require.Nil(t, failedBlocks)
			require.NotEmpty(t, trs)

			combiner := modeltrace.NewCombiner(0, false)
			for _, tr := range trs {
				require.NotNil(t, tr.Trace)
				_, err = combiner.Consume(tr.Trace)
				require.NoError(t, err)
			}
			result, _ := combiner.Result()

			modeltrace.SortTrace(allReqs[i])
			modeltrace.SortTrace(result)
			require.True(t, proto.Equal(allReqs[i], result))
		})
	}
}

func TestSameIDCompaction(t *testing.T) {
	for _, enc := range encoding.AllEncodingsForWrites() {
		t.Run(enc.Version(), func(t *testing.T) {
			testSameIDCompaction(t, enc.Version())
		})
	}
}

// testSameIDCompaction is a bit gross in that it has a bad dependency on the /pkg/model
// module to do a full e2e compaction/combination test.
func testSameIDCompaction(t *testing.T, targetBlockVersion string) {
	tempDir := t.TempDir()

	r, w, c, err := New(&Config{
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
			RowGroupSizeBytes:   30_000_000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	ctx := context.Background()
	r.EnablePolling(ctx, &mockJobSharder{}, false)

	wal := w.WAL()
	require.NoError(t, err)

	dec := model.MustNewSegmentDecoder(v1.Encoding)

	blockCount := 5
	recordCount := 100

	// make a bunch of sharded requests
	allReqs := make([][][]byte, 0, recordCount)
	allIDs := make([][]byte, 0, recordCount)
	sharded := 0
	for i := 0; i < recordCount; i++ {
		id := test.ValidTraceID(nil)

		requestShards := rand.Intn(blockCount) + 1
		reqs := make([][]byte, 0, requestShards)
		for j := 0; j < requestShards; j++ {
			buff, err := dec.PrepareForWrite(test.MakeTrace(1, id), 0, 0)
			require.NoError(t, err)

			buff2, err := dec.ToObject([][]byte{buff})
			require.NoError(t, err)

			reqs = append(reqs, buff2)
		}

		if requestShards > 1 {
			sharded++
		}
		allReqs = append(allReqs, reqs)
		allIDs = append(allIDs, id)
	}

	// and write them to different blocks
	for i := 0; i < blockCount; i++ {
		blockID := backend.NewUUID()
		meta := &backend.BlockMeta{BlockID: blockID, TenantID: testTenantID}
		head, err := wal.NewBlock(meta, v1.Encoding)
		require.NoError(t, err)

		for j := 0; j < recordCount; j++ {
			req := allReqs[j]
			id := allIDs[j]

			if i < len(req) {
				err = head.Append(id, req[i], 0, 0, true)
				require.NoError(t, err, "unexpected error writing req")
			}
		}

		_, err = w.CompleteBlock(context.Background(), head)
		require.NoError(t, err)
	}

	rw := r.(*readerWriter)

	// check blocklists, force compaction and check again
	checkBlocklists(ctx, t, uuid.Nil, blockCount, 0, rw)

	metas := rw.blocklist.Metas(testTenantID)
	require.Len(t, metas, blockCount)

	combinedStart, err := test.GetCounterVecValue(metricCompactionObjectsCombined, "0")
	require.NoError(t, err)

	cfg := &CompactorConfig{
		MaxCompactionRange:   24 * time.Hour,
		MaxCompactionObjects: 10000,
		MaxBlockBytes:        1024 * 1024 * 1024,
	}
	_, err = c.CompactWithConfig(ctx, metas, testTenantID, cfg, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	checkBlocklists(ctx, t, uuid.Nil, 1, blockCount, rw)

	// force clear compacted blocks to guarantee that we're only querying the new blocks that went through the combiner
	remaining := rw.blocklist.Metas(testTenantID)
	rw.blocklist.ApplyPollResults(blocklist.PerTenant{testTenantID: remaining}, blocklist.PerTenantCompacted{})

	// search for all ids
	for i, id := range allIDs {
		trs, failedBlocks, err := rw.Find(context.Background(), testTenantID, id, BlockIDMin, BlockIDMax, time.Time{}, time.Time{}, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.Nil(t, failedBlocks)

		combiner := modeltrace.NewCombiner(0, false)
		for _, tr := range trs {
			_, err = combiner.Consume(tr.Trace)
			require.NoError(t, err)
		}
		tr, _ := combiner.Result()
		b1, err := dec.PrepareForWrite(tr, 0, 0)
		require.NoError(t, err)

		b2, err := dec.ToObject([][]byte{b1})
		require.NoError(t, err)

		expectedBytes, _, err := model.StaticCombiner.Combine(v1.Encoding, allReqs[i]...)
		require.NoError(t, err)
		require.Equal(t, expectedBytes, b2)
	}

	combinedEnd, err := test.GetCounterVecValue(metricCompactionObjectsCombined, "0")
	require.NoError(t, err)
	require.Equal(t, float64(sharded), combinedEnd-combinedStart)
}

func TestCompactionHonorsBlockStartEndTimes(t *testing.T) {
	for _, enc := range encoding.AllEncodingsForWrites() {
		t.Run(enc.Version(), func(t *testing.T) {
			t.Parallel()
			testCompactionHonorsBlockStartEndTimes(t, enc.Version())
		})
	}
}

func testCompactionHonorsBlockStartEndTimes(t *testing.T, targetBlockVersion string) {
	tempDir := t.TempDir()

	r, w, c, err := New(&Config{
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
			RowGroupSizeBytes:   30_000_000,
		},
		WAL: &wal.Config{
			Filepath:       path.Join(tempDir, "wal"),
			IngestionSlack: time.Since(time.Unix(0, 0)), // allow explicit start/end times below
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	ctx := context.Background()
	r.EnablePolling(ctx, &mockJobSharder{}, false)

	cutTestBlockWithTraces(t, w, []testData{
		{test.ValidTraceID(nil), test.MakeTrace(10, nil), 100, 101},
		{test.ValidTraceID(nil), test.MakeTrace(10, nil), 102, 103},
	})
	cutTestBlockWithTraces(t, w, []testData{
		{test.ValidTraceID(nil), test.MakeTrace(10, nil), 104, 105},
		{test.ValidTraceID(nil), test.MakeTrace(10, nil), 106, 107},
	})

	rw := r.(*readerWriter)
	rw.pollBlocklist(ctx)

	cfg := &CompactorConfig{
		MaxCompactionRange:   24 * time.Hour,
		MaxCompactionObjects: 1000,
		MaxBlockBytes:        100_000_000,
	}
	_, err = c.CompactWithConfig(ctx, rw.blocklist.Metas(testTenantID), testTenantID, cfg, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	blocks := rw.blocklist.Metas(testTenantID)
	require.Equal(t, 1, len(blocks))
	require.Equal(t, uint32(1), blocks[0].CompactionLevel)
	require.Equal(t, 100, int(blocks[0].StartTime.Unix()))
	require.Equal(t, 107, int(blocks[0].EndTime.Unix()))
}

func BenchmarkCompaction(b *testing.B) {
	for _, enc := range encoding.AllEncodingsForWrites() {
		b.Run(enc.Version(), func(b *testing.B) {
			benchmarkCompaction(b, enc.Version())
		})
	}
}

func benchmarkCompaction(b *testing.B, targetBlockVersion string) {
	tempDir := b.TempDir()

	_, w, c, err := New(&Config{
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
			RowGroupSizeBytes:   30_000_000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	require.NoError(b, err)

	ctx := context.Background()

	traceCount := 20_000
	blockCount := 8

	blocks := cutTestBlocks(b, w, testTenantID, blockCount, traceCount)
	metas := make([]*backend.BlockMeta, 0, len(blocks))
	for _, blk := range blocks {
		metas = append(metas, blk.BlockMeta())
	}

	cfg := &CompactorConfig{
		MaxCompactionRange:   24 * time.Hour,
		MaxCompactionObjects: 1000,
		MaxBlockBytes:        100_000_000,
	}

	b.ResetTimer()
	_, err = c.CompactWithConfig(ctx, metas, testTenantID, cfg, &mockSharder{}, &mockOverrides{})
	require.NoError(b, err)
}

func TestCompactWithConfigUnsupportedVersion(t *testing.T) {
	tempDir := t.TempDir()

	// Create backend reader/writer directly to write the block meta
	localCfg := &local.Config{Path: path.Join(tempDir, "traces")}
	_, rawW, _, err := local.New(localCfg)
	require.NoError(t, err)
	backendW := backend.NewWriter(rawW)

	_, _, c, err := New(&Config{
		Backend: backend.Local,
		Pool: &pool.Config{
			MaxWorkers: 10,
			QueueDepth: 100,
		},
		Local: localCfg,
		Block: &common.BlockConfig{
			BloomFP:             .01,
			BloomShardSizeBytes: 100_000,
			Version:             "vParquet4",
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	ctx := context.Background()

	// Create a block meta with an unsupported preview version and write it to storage
	meta := &backend.BlockMeta{
		BlockID:  backend.NewUUID(),
		TenantID: testTenantID,
		Version:  "vParquet5-preview6",
	}
	err = backendW.WriteBlockMeta(ctx, meta)
	require.NoError(t, err)

	// Try to compact
	_, err = c.CompactWithConfig(
		ctx,
		[]*backend.BlockMeta{meta},
		testTenantID,
		&CompactorConfig{
			MaxCompactionRange: 24 * time.Hour,
		},
		&mockSharder{},
		&mockOverrides{},
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "compaction not supported for block version vParquet5-preview6")
}
