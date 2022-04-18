package tempodb

import (
	"context"
	"encoding/binary"
	"math/rand"
	"path"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/golang/protobuf/proto"
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
	"github.com/grafana/tempo/tempodb/encoding/common"
	v2 "github.com/grafana/tempo/tempodb/encoding/v2"
	"github.com/grafana/tempo/tempodb/metrics"
	"github.com/grafana/tempo/tempodb/pool"
	"github.com/grafana/tempo/tempodb/wal"
)

type mockSharder struct {
}

func (m *mockSharder) Owns(hash string) bool {
	return true
}

func (m *mockSharder) Combine(dataEncoding string, tenantID string, objs ...[]byte) ([]byte, bool, error) {
	return model.StaticCombiner.Combine(dataEncoding, objs...)
}

type mockCombiner struct {
}

func (m *mockCombiner) Combine(dataEncoding string, objs ...[]byte) ([]byte, bool, error) {
	return model.StaticCombiner.Combine(dataEncoding, objs...)
}

type mockJobSharder struct{}

func (m *mockJobSharder) Owns(_ string) bool { return true }

type mockOverrides struct {
	blockRetention time.Duration
}

func (m *mockOverrides) BlockRetentionForTenant(_ string) time.Duration {
	return m.blockRetention
}

func TestCompaction(t *testing.T) {
	tempDir := t.TempDir()

	r, w, c, err := New(&Config{
		Backend: "local",
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
			Encoding:             backend.EncLZ4_4M,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	require.NoError(t, err)

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      24 * time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})

	r.EnablePolling(&mockJobSharder{})

	wal := w.WAL()
	assert.NoError(t, err)

	blockCount := 4
	recordCount := 100

	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	allReqs := make([]*tempopb.Trace, 0, blockCount*recordCount)
	allIds := make([]common.ID, 0, blockCount*recordCount)

	for i := 0; i < blockCount; i++ {
		blockID := uuid.New()
		head, err := wal.NewBlock(blockID, testTenantID, model.CurrentEncoding)
		require.NoError(t, err)

		for j := 0; j < recordCount; j++ {
			id := test.ValidTraceID(nil)
			req := test.MakeTrace(10, id)
			writeTraceToWal(t, head, dec, id, req, 0, 0)

			allReqs = append(allReqs, req)
			allIds = append(allIds, id)
		}

		_, err = w.CompleteBlock(head, &mockCombiner{})
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
		assert.Len(t, blocks, inputBlocks)

		compactions++
		err := rw.compact(blocks, testTenantID)
		assert.NoError(t, err)

		expectedBlockCount -= blocksPerCompaction
		expectedCompactedCount += inputBlocks
		checkBlocklists(t, uuid.Nil, expectedBlockCount, expectedCompactedCount, rw)
	}

	assert.Equal(t, expectedCompactions, compactions)

	// do we have the right number of records
	var records int
	for _, meta := range rw.blocklist.Metas(testTenantID) {
		records += meta.TotalObjects
	}
	assert.Equal(t, blockCount*recordCount, records)

	// now see if we can find our ids
	for i, id := range allIds {
		trs, failedBlocks, err := rw.Find(context.Background(), testTenantID, id, BlockIDMin, BlockIDMax, 0, 0)
		require.NoError(t, err)
		require.Nil(t, failedBlocks)
		require.NotNil(t, trs)

		c := trace.NewCombiner()
		for _, tr := range trs {
			c.Consume(tr)
		}
		tr, _ := c.Result()

		// Traces come out of the combiner sorted,
		// so do the same here.
		trace.SortTrace(allReqs[i])
		require.True(t, proto.Equal(allReqs[i], tr))
	}
}

// TestSameIDCompaction is a bit gross in that it has a bad dependency with on the /pkg/model
// module to do a full e2e compaction/combination test.
func TestSameIDCompaction(t *testing.T) {
	tempDir := t.TempDir()

	r, w, c, err := New(&Config{
		Backend: "local",
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
			Encoding:             backend.EncSnappy,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	require.NoError(t, err)

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      24 * time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})

	r.EnablePolling(&mockJobSharder{})

	wal := w.WAL()
	assert.NoError(t, err)

	dec := model.MustNewSegmentDecoder(v1.Encoding)

	blockCount := 5
	recordCount := 100

	// make a bunch of sharded requests
	allReqs := make([][][]byte, 0, recordCount)
	allIds := make([][]byte, 0, recordCount)
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

		allReqs = append(allReqs, reqs)
		allIds = append(allIds, id)
	}

	// and write them to different blocks
	for i := 0; i < blockCount; i++ {
		blockID := uuid.New()
		head, err := wal.NewBlock(blockID, testTenantID, v1.Encoding)
		require.NoError(t, err)

		for j := 0; j < recordCount; j++ {
			req := allReqs[j]
			id := allIds[j]

			if i < len(req) {
				err = head.Append(id, req[i], 0, 0)
				require.NoError(t, err, "unexpected error writing req")
			}
		}

		_, err = w.CompleteBlock(head, &mockCombiner{})
		require.NoError(t, err)
	}

	rw := r.(*readerWriter)

	// check blocklists, force compaction and check again
	checkBlocklists(t, uuid.Nil, blockCount, 0, rw)

	var blocks []*backend.BlockMeta
	list := rw.blocklist.Metas(testTenantID)
	blockSelector := newTimeWindowBlockSelector(list, rw.compactorCfg.MaxCompactionRange, 10000, 1024*1024*1024, defaultMinInputBlocks, blockCount)
	blocks, _ = blockSelector.BlocksToCompact()
	assert.Len(t, blocks, blockCount)

	err = rw.compact(blocks, testTenantID)
	require.NoError(t, err)

	checkBlocklists(t, uuid.Nil, 1, blockCount, rw)

	// force clear compacted blocks to guarantee that we're only querying the new blocks that went through the combiner
	metas := rw.blocklist.Metas(testTenantID)
	rw.blocklist.ApplyPollResults(blocklist.PerTenant{testTenantID: metas}, blocklist.PerTenantCompacted{})

	// search for all ids
	for i, id := range allIds {
		trs, failedBlocks, err := rw.Find(context.Background(), testTenantID, id, BlockIDMin, BlockIDMax, 0, 0)
		assert.NoError(t, err)
		assert.Nil(t, failedBlocks)

		c := trace.NewCombiner()
		for _, tr := range trs {
			c.Consume(tr)
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
}

func TestCompactionUpdatesBlocklist(t *testing.T) {
	tempDir := t.TempDir()

	r, w, c, err := New(&Config{
		Backend: "local",
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
			Encoding:             backend.EncNone,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	require.NoError(t, err)

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      24 * time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})

	r.EnablePolling(&mockJobSharder{})

	// Cut x blocks with y records each
	blockCount := 5
	recordCount := 1
	cutTestBlocks(t, w, testTenantID, blockCount, recordCount)

	rw := r.(*readerWriter)
	rw.pollBlocklist()

	// compact everything
	err = rw.compact(rw.blocklist.Metas(testTenantID), testTenantID)
	require.NoError(t, err)

	// New blocklist contains 1 compacted block with everything
	blocks := rw.blocklist.Metas(testTenantID)
	require.Equal(t, 1, len(blocks))
	require.Equal(t, uint8(1), blocks[0].CompactionLevel)
	require.Equal(t, blockCount*recordCount, blocks[0].TotalObjects)

	// Compacted list contains all old blocks
	require.Equal(t, blockCount, len(rw.blocklist.CompactedMetas(testTenantID)))

	// Make sure all expected traces are found.
	for i := 0; i < blockCount; i++ {
		for j := 0; j < recordCount; j++ {
			trace, failedBlocks, err := rw.Find(context.TODO(), testTenantID, makeTraceID(i, j), BlockIDMin, BlockIDMax, 0, 0)
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
		Backend: "local",
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
			Encoding:             backend.EncNone,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	assert.NoError(t, err)

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      24 * time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})

	r.EnablePolling(&mockJobSharder{})

	// Cut x blocks with y records each
	blockCount := 5
	recordCount := 1
	cutTestBlocks(t, w, testTenantID, blockCount, recordCount)

	rw := r.(*readerWriter)
	rw.pollBlocklist()

	// Get starting metrics
	processedStart, err := test.GetCounterVecValue(metrics.MetricCompactionObjectsWritten, "0")
	assert.NoError(t, err)

	blocksStart, err := test.GetCounterVecValue(metrics.MetricCompactionBlocks, "0")
	assert.NoError(t, err)

	bytesStart, err := test.GetCounterVecValue(metrics.MetricCompactionBytesWritten, "0")
	assert.NoError(t, err)

	// compact everything
	err = rw.compact(rw.blocklist.Metas(testTenantID), testTenantID)
	assert.NoError(t, err)

	// Check metric
	processedEnd, err := test.GetCounterVecValue(metrics.MetricCompactionObjectsWritten, "0")
	assert.NoError(t, err)
	assert.Equal(t, float64(blockCount*recordCount), processedEnd-processedStart)

	blocksEnd, err := test.GetCounterVecValue(metrics.MetricCompactionBlocks, "0")
	assert.NoError(t, err)
	assert.Equal(t, float64(blockCount), blocksEnd-blocksStart)

	bytesEnd, err := test.GetCounterVecValue(metrics.MetricCompactionBytesWritten, "0")
	assert.NoError(t, err)
	assert.Greater(t, bytesEnd, bytesStart) // calculating the exact bytes requires knowledge of the bytes as written in the blocks.  just make sure it goes up
}

func TestCompactionIteratesThroughTenants(t *testing.T) {
	tempDir := t.TempDir()

	r, w, c, err := New(&Config{
		Backend: "local",
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
			Encoding:             backend.EncLZ4_64k,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	assert.NoError(t, err)

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      24 * time.Hour,
		MaxCompactionObjects:    1000,
		MaxBlockBytes:           1024 * 1024 * 1024,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})

	r.EnablePolling(&mockJobSharder{})

	// Cut blocks for multiple tenants
	cutTestBlocks(t, w, testTenantID, 2, 2)
	cutTestBlocks(t, w, testTenantID2, 2, 2)

	rw := r.(*readerWriter)
	rw.pollBlocklist()

	assert.Equal(t, 2, len(rw.blocklist.Metas(testTenantID)))
	assert.Equal(t, 2, len(rw.blocklist.Metas(testTenantID2)))

	// Verify that tenant 2 compacted, tenant 1 is not
	// Compaction starts at index 1 for simplicity
	rw.doCompaction()
	assert.Equal(t, 2, len(rw.blocklist.Metas(testTenantID)))
	assert.Equal(t, 1, len(rw.blocklist.Metas(testTenantID2)))

	// Verify both tenants compacted after second run
	rw.doCompaction()
	assert.Equal(t, 1, len(rw.blocklist.Metas(testTenantID)))
	assert.Equal(t, 1, len(rw.blocklist.Metas(testTenantID2)))
}

func cutTestBlocks(t testing.TB, w Writer, tenantID string, blockCount int, recordCount int) []*v2.BackendBlock {
	blocks := make([]*v2.BackendBlock, 0)
	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	wal := w.WAL()
	for i := 0; i < blockCount; i++ {
		head, err := wal.NewBlock(uuid.New(), tenantID, model.CurrentEncoding)
		require.NoError(t, err)

		for j := 0; j < recordCount; j++ {
			id := makeTraceID(i, j)
			tr := test.MakeTrace(1, id)
			now := uint32(time.Now().Unix())
			writeTraceToWal(t, head, dec, id, tr, now, now)
		}

		b, err := w.CompleteBlock(head, &mockCombiner{})
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
	tempDir := b.TempDir()

	_, w, c, err := New(&Config{
		Backend: "local",
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
			Encoding:             backend.EncZstd,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	require.NoError(b, err)

	rw := c.(*readerWriter)

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:     1_000_000,
		FlushSizeBytes:     1_000_000,
		IteratorBufferSize: DefaultIteratorBufferSize,
	}, &mockSharder{}, &mockOverrides{})

	n := b.N

	// Cut input blocks
	blocks := cutTestBlocks(b, w, testTenantID, 8, n)
	metas := make([]*backend.BlockMeta, 0)
	for _, b := range blocks {
		metas = append(metas, b.BlockMeta())
	}

	b.ResetTimer()

	err = rw.compact(metas, testTenantID)
	require.NoError(b, err)
}
