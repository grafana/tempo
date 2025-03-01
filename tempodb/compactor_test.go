package tempodb

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"math/rand"
	"path"
	"testing"
	"time"

	"github.com/go-kit/log"
	proto "github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/model/trace"
	v1 "github.com/grafana/tempo/pkg/model/v1"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/blocklist"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/pool"
	"github.com/grafana/tempo/tempodb/wal"
)

type mockSharder struct{}

func (m *mockSharder) Owns(string) bool {
	return true
}

func (m *mockSharder) Combine(dataEncoding string, _ string, objs ...[]byte) ([]byte, bool, error) {
	return model.StaticCombiner.Combine(dataEncoding, objs...)
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

func TestCompactionRoundtrip(t *testing.T) {
	for _, enc := range encoding.AllEncodings() {
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
			IndexDownsampleBytes: 11,
			BloomFP:              .01,
			BloomShardSizeBytes:  100_000,
			Version:              targetBlockVersion,
			Encoding:             backend.EncLZ4_4M,
			IndexPageSizeBytes:   1000,
			RowGroupSizeBytes:    30_000_000,
			DedicatedColumns:     backend.DedicatedColumns{{Scope: "span", Name: "key", Type: "string"}},
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	ctx := context.Background()
	err = c.EnableCompaction(ctx, &CompactorConfig{
		ChunkSizeBytes:          10_000_000,
		FlushSizeBytes:          10_000_000,
		MaxCompactionRange:      24 * time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	r.EnablePolling(ctx, &mockJobSharder{})

	wal := w.WAL()
	require.NoError(t, err)

	blockCount := 4
	recordCount := 100

	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	allReqs := make([]*tempopb.Trace, 0, blockCount*recordCount)
	allIds := make([]common.ID, 0, blockCount*recordCount)

	for i := 0; i < blockCount; i++ {
		blockID := backend.NewUUID()
		meta := &backend.BlockMeta{BlockID: blockID, TenantID: testTenantID, DataEncoding: model.CurrentEncoding}
		head, err := wal.NewBlock(meta, model.CurrentEncoding)
		require.NoError(t, err)

		for j := 0; j < recordCount; j++ {
			id := test.ValidTraceID(nil)
			req := test.MakeTrace(10, id)

			writeTraceToWal(t, head, dec, id, req, 0, 0)

			allReqs = append(allReqs, req)
			allIds = append(allIds, id)
		}

		_, err = w.CompleteBlock(ctx, head)
		require.NoError(t, err)
	}

	rw := r.(*readerWriter)

	expectedBlockCount := blockCount
	expectedCompactedCount := 0
	checkBlocklists(t, uuid.Nil, expectedBlockCount, expectedCompactedCount, rw)

	blocksPerCompaction := (inputBlocks - outputBlocks)

	rw.pollBlocklist()

	blocklist := rw.blocklist.Metas(testTenantID)
	blockSelector := newTimeWindowBlockSelector(blocklist, rw.compactorCfg.MaxCompactionRange, 10000, 1024*1024*1024, defaultMinInputBlocks, 2)

	expectedCompactions := len(blocklist) / inputBlocks
	compactions := 0
	for {
		blocks, _ := blockSelector.BlocksToCompact()
		if len(blocks) == 0 {
			break
		}
		require.Len(t, blocks, inputBlocks)

		compactions++
		err := rw.Compact(context.Background(), blocks, testTenantID)
		require.NoError(t, err)

		expectedBlockCount -= blocksPerCompaction
		expectedCompactedCount += inputBlocks
		checkBlocklists(t, uuid.Nil, expectedBlockCount, expectedCompactedCount, rw)
	}

	require.Equal(t, expectedCompactions, compactions)

	// do we have the right number of records
	var records int
	for _, meta := range rw.blocklist.Metas(testTenantID) {
		records += int(meta.TotalObjects)
	}
	require.Equal(t, blockCount*recordCount, records)

	// now see if we can find our ids
	for i, id := range allIds {
		trs, failedBlocks, err := rw.Find(context.Background(), testTenantID, id, BlockIDMin, BlockIDMax, 0, 0, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.Nil(t, failedBlocks)
		require.NotNil(t, trs)

		c := trace.NewCombiner(0, false)
		for _, tr := range trs {
			require.NotNil(t, tr)
			require.NotNil(t, tr.Trace)
			_, err = c.Consume(tr.Trace)
			require.NoError(t, err)
		}
		tr, _ := c.Result()

		// Sort all traces to check equality consistently.
		trace.SortTrace(allReqs[i])
		trace.SortTrace(tr)

		if !proto.Equal(allReqs[i], tr) {
			wantJSON, _ := json.MarshalIndent(allReqs[i], "", "  ")
			gotJSON, _ := json.MarshalIndent(tr, "", "  ")
			require.Equal(t, wantJSON, gotJSON)
		}
	}
}

func TestSameIDCompaction(t *testing.T) {
	for _, enc := range encoding.AllEncodings() {
		version := enc.Version()
		t.Run(version, func(t *testing.T) {
			testSameIDCompaction(t, version)
		})
	}
}

// TestSameIDCompaction is a bit gross in that it has a bad dependency with on the /pkg/model
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
			IndexDownsampleBytes: 11,
			BloomFP:              .01,
			BloomShardSizeBytes:  100_000,
			Version:              targetBlockVersion,
			Encoding:             backend.EncSnappy,
			IndexPageSizeBytes:   1000,
			RowGroupSizeBytes:    30_000_000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	ctx := context.Background()
	err = c.EnableCompaction(ctx, &CompactorConfig{
		ChunkSizeBytes:          10_000_000,
		MaxCompactionRange:      24 * time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
		FlushSizeBytes:          10_000_000,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	r.EnablePolling(ctx, &mockJobSharder{})

	wal := w.WAL()
	require.NoError(t, err)

	dec := model.MustNewSegmentDecoder(v1.Encoding)

	blockCount := 5
	recordCount := 100

	// make a bunch of sharded requests
	allReqs := make([][][]byte, 0, recordCount)
	allIds := make([][]byte, 0, recordCount)
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
		allIds = append(allIds, id)
	}

	// and write them to different blocks
	for i := 0; i < blockCount; i++ {
		blockID := backend.NewUUID()
		meta := &backend.BlockMeta{BlockID: blockID, TenantID: testTenantID, DataEncoding: v1.Encoding}
		head, err := wal.NewBlock(meta, v1.Encoding)
		require.NoError(t, err)

		for j := 0; j < recordCount; j++ {
			req := allReqs[j]
			id := allIds[j]

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
	checkBlocklists(t, uuid.Nil, blockCount, 0, rw)

	var blocks []*backend.BlockMeta
	list := rw.blocklist.Metas(testTenantID)
	blockSelector := newTimeWindowBlockSelector(list, rw.compactorCfg.MaxCompactionRange, 10000, 1024*1024*1024, defaultMinInputBlocks, blockCount)
	blocks, _ = blockSelector.BlocksToCompact()
	require.Len(t, blocks, blockCount)

	combinedStart, err := test.GetCounterVecValue(metricCompactionObjectsCombined, "0")
	require.NoError(t, err)

	err = rw.Compact(ctx, blocks, testTenantID)
	require.NoError(t, err)

	checkBlocklists(t, uuid.Nil, 1, blockCount, rw)

	// force clear compacted blocks to guarantee that we're only querying the new blocks that went through the combiner
	metas := rw.blocklist.Metas(testTenantID)
	rw.blocklist.ApplyPollResults(blocklist.PerTenant{testTenantID: metas}, blocklist.PerTenantCompacted{})

	// search for all ids
	for i, id := range allIds {
		trs, failedBlocks, err := rw.Find(context.Background(), testTenantID, id, BlockIDMin, BlockIDMax, 0, 0, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.Nil(t, failedBlocks)

		c := trace.NewCombiner(0, false)
		for _, tr := range trs {
			_, err = c.Consume(tr.Trace)
			require.NoError(t, err)
		}
		tr, _ := c.Result()
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

func TestCompactionUpdatesBlocklist(t *testing.T) {
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
			IndexDownsampleBytes: 11,
			BloomFP:              .01,
			BloomShardSizeBytes:  100_000,
			Version:              encoding.DefaultEncoding().Version(),
			Encoding:             backend.EncNone,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	ctx := context.Background()
	err = c.EnableCompaction(ctx, &CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      24 * time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	r.EnablePolling(ctx, &mockJobSharder{})

	// Cut x blocks with y records each
	blockCount := 5
	recordCount := 1
	cutTestBlocks(t, w, testTenantID, blockCount, recordCount)

	rw := r.(*readerWriter)
	rw.pollBlocklist()

	// compact everything
	err = rw.Compact(ctx, rw.blocklist.Metas(testTenantID), testTenantID)
	require.NoError(t, err)

	// New blocklist contains 1 compacted block with everything
	blocks := rw.blocklist.Metas(testTenantID)
	require.Equal(t, 1, len(blocks))
	require.Equal(t, uint32(1), blocks[0].CompactionLevel)
	require.Equal(t, int64(blockCount*recordCount), blocks[0].TotalObjects)

	// Compacted list contains all old blocks
	require.Equal(t, blockCount, len(rw.blocklist.CompactedMetas(testTenantID)))

	// Make sure all expected traces are found.
	for i := 0; i < blockCount; i++ {
		for j := 0; j < recordCount; j++ {
			trace, failedBlocks, err := rw.Find(context.TODO(), testTenantID, makeTraceID(i, j), BlockIDMin, BlockIDMax, 0, 0, common.DefaultSearchOptions())
			require.NotNil(t, trace)
			require.Greater(t, len(trace), 0)
			require.NoError(t, err)
			require.Nil(t, failedBlocks)
		}
	}
}

func TestCompactionMetrics(t *testing.T) {
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
			IndexDownsampleBytes: 11,
			BloomFP:              .01,
			BloomShardSizeBytes:  100_000,
			Version:              encoding.DefaultEncoding().Version(),
			Encoding:             backend.EncNone,
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
		MaxCompactionRange:      24 * time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	r.EnablePolling(ctx, &mockJobSharder{})

	// Cut x blocks with y records each
	blockCount := 5
	recordCount := 10
	cutTestBlocks(t, w, testTenantID, blockCount, recordCount)

	rw := r.(*readerWriter)
	rw.pollBlocklist()

	// Get starting metrics
	processedStart, err := test.GetCounterVecValue(metricCompactionObjectsWritten, "0")
	assert.NoError(t, err)

	blocksStart, err := test.GetCounterVecValue(metricCompactionBlocks, "0")
	assert.NoError(t, err)

	bytesStart, err := test.GetCounterVecValue(metricCompactionBytesWritten, "0")
	assert.NoError(t, err)

	// compact everything
	err = rw.Compact(ctx, rw.blocklist.Metas(testTenantID), testTenantID)
	assert.NoError(t, err)

	// Check metric
	processedEnd, err := test.GetCounterVecValue(metricCompactionObjectsWritten, "0")
	assert.NoError(t, err)
	assert.Equal(t, float64(blockCount*recordCount), processedEnd-processedStart)

	blocksEnd, err := test.GetCounterVecValue(metricCompactionBlocks, "0")
	assert.NoError(t, err)
	assert.Equal(t, float64(blockCount), blocksEnd-blocksStart)

	bytesEnd, err := test.GetCounterVecValue(metricCompactionBytesWritten, "0")
	assert.NoError(t, err)
	assert.Greater(t, bytesEnd, bytesStart) // calculating the exact bytes requires knowledge of the bytes as written in the blocks.  just make sure it goes up
}

func TestCompactionIteratesThroughTenants(t *testing.T) {
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
			IndexDownsampleBytes: 11,
			BloomFP:              .01,
			BloomShardSizeBytes:  100_000,
			Version:              encoding.DefaultEncoding().Version(),
			Encoding:             backend.EncLZ4_64k,
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
		MaxCompactionRange:      24 * time.Hour,
		MaxCompactionObjects:    1000,
		MaxBlockBytes:           1024 * 1024 * 1024,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	r.EnablePolling(ctx, &mockJobSharder{})

	// Cut blocks for multiple tenants
	cutTestBlocks(t, w, testTenantID, 2, 2)
	cutTestBlocks(t, w, testTenantID2, 2, 2)

	rw := r.(*readerWriter)
	rw.pollBlocklist()

	assert.Equal(t, 2, len(rw.blocklist.Metas(testTenantID)))
	assert.Equal(t, 2, len(rw.blocklist.Metas(testTenantID2)))

	// Verify that tenant 2 compacted, tenant 1 is not
	// Compaction starts at index 1 for simplicity
	rw.compactOneTenant(ctx)
	assert.Equal(t, 2, len(rw.blocklist.Metas(testTenantID)))
	assert.Equal(t, 1, len(rw.blocklist.Metas(testTenantID2)))

	// Verify both tenants compacted after second run
	rw.compactOneTenant(ctx)
	assert.Equal(t, 1, len(rw.blocklist.Metas(testTenantID)))
	assert.Equal(t, 1, len(rw.blocklist.Metas(testTenantID2)))
}

func TestCompactionHonorsBlockStartEndTimes(t *testing.T) {
	for _, enc := range encoding.AllEncodings() {
		version := enc.Version()
		t.Run(version, func(t *testing.T) {
			testCompactionHonorsBlockStartEndTimes(t, version)
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
			IndexDownsampleBytes: 11,
			BloomFP:              .01,
			BloomShardSizeBytes:  100_000,
			Version:              targetBlockVersion,
			Encoding:             backend.EncNone,
			IndexPageSizeBytes:   1000,
			RowGroupSizeBytes:    30_000_000,
		},
		WAL: &wal.Config{
			Filepath:       path.Join(tempDir, "wal"),
			IngestionSlack: time.Since(time.Unix(0, 0)), // Let us use obvious start/end times below
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	ctx := context.Background()
	err = c.EnableCompaction(ctx, &CompactorConfig{
		ChunkSizeBytes:          10_000_000,
		FlushSizeBytes:          10_000_000,
		MaxCompactionRange:      24 * time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	r.EnablePolling(ctx, &mockJobSharder{})

	cutTestBlockWithTraces(t, w, []testData{
		{test.ValidTraceID(nil), test.MakeTrace(10, nil), 100, 101},
		{test.ValidTraceID(nil), test.MakeTrace(10, nil), 102, 103},
	})
	cutTestBlockWithTraces(t, w, []testData{
		{test.ValidTraceID(nil), test.MakeTrace(10, nil), 104, 105},
		{test.ValidTraceID(nil), test.MakeTrace(10, nil), 106, 107},
	})

	rw := r.(*readerWriter)
	rw.pollBlocklist()

	// compact everything
	err = rw.Compact(ctx, rw.blocklist.Metas(testTenantID), testTenantID)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// New blocklist contains 1 compacted block with min start and max end
	blocks := rw.blocklist.Metas(testTenantID)
	require.Equal(t, 1, len(blocks))
	require.Equal(t, uint32(1), blocks[0].CompactionLevel)
	require.Equal(t, 100, int(blocks[0].StartTime.Unix()))
	require.Equal(t, 107, int(blocks[0].EndTime.Unix()))
}

func TestCompactionDropsTraces(t *testing.T) {
	for _, enc := range encoding.AllEncodings() {
		version := enc.Version()
		t.Run(version, func(t *testing.T) {
			testCompactionDropsTraces(t, version)
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
			IndexDownsampleBytes: 11,
			BloomFP:              .01,
			BloomShardSizeBytes:  100_000,
			Version:              targetBlockVersion,
			Encoding:             backend.EncSnappy,
			IndexPageSizeBytes:   1000,
			RowGroupSizeBytes:    30_000_000,
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
	meta := &backend.BlockMeta{BlockID: blockID, TenantID: testTenantID, DataEncoding: v1.Encoding}
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
		BlockConfig:        *rw.cfg.Block,
		ChunkSizeBytes:     DefaultChunkSizeBytes,
		FlushSizeBytes:     DefaultFlushSizeBytes,
		IteratorBufferSize: DefaultIteratorBufferSize,
		OutputBlocks:       1,
		Combiner:           model.StaticCombiner,
		MaxBytesPerTrace:   0,

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

func TestDoForAtLeast(t *testing.T) {
	// test that it runs for at least the duration
	start := time.Now()
	doForAtLeast(context.Background(), time.Second, func() { time.Sleep(time.Millisecond) })
	require.WithinDuration(t, time.Now(), start.Add(time.Second), 100*time.Millisecond)

	// test that it allows func to overrun
	start = time.Now()
	doForAtLeast(context.Background(), time.Second, func() { time.Sleep(2 * time.Second) })
	require.WithinDuration(t, time.Now(), start.Add(2*time.Second), 100*time.Millisecond)

	// make sure cancelling the context stops the function if the function is complete and we're
	// just waiting. it is presumed, but not enforced that the function responds to a cancelled contxt
	ctx, cancel := context.WithCancel(context.Background())
	start = time.Now()
	go func() {
		time.Sleep(time.Second)
		cancel()
	}()
	doForAtLeast(ctx, 2*time.Second, func() {})
	require.WithinDuration(t, time.Now(), start.Add(time.Second), 100*time.Millisecond)
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

func BenchmarkCompaction(b *testing.B) {
	for _, enc := range encoding.AllEncodings() {
		version := enc.Version()
		b.Run(version, func(b *testing.B) {
			benchmarkCompaction(b, version)
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
			IndexDownsampleBytes: 11,
			BloomFP:              .01,
			BloomShardSizeBytes:  100_000,
			Version:              targetBlockVersion,
			Encoding:             backend.EncZstd,
			IndexPageSizeBytes:   1000,
			RowGroupSizeBytes:    30_000_000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	require.NoError(b, err)

	rw := c.(*readerWriter)

	ctx := context.Background()
	err = c.EnableCompaction(ctx, &CompactorConfig{
		ChunkSizeBytes:     10_000_000,
		FlushSizeBytes:     10_000_000,
		IteratorBufferSize: DefaultIteratorBufferSize,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(b, err)

	traceCount := 20_000
	blockCount := 8

	// Cut input blocks
	blocks := cutTestBlocks(b, w, testTenantID, blockCount, traceCount)
	metas := make([]*backend.BlockMeta, 0)
	for _, b := range blocks {
		metas = append(metas, b.BlockMeta())
	}

	b.ResetTimer()

	err = rw.Compact(ctx, metas, testTenantID)
	require.NoError(b, err)
}
