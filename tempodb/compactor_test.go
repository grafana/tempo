package tempodb

import (
	"context"
	"encoding/binary"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/model"
	v1 "github.com/grafana/tempo/pkg/model/v1"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/blocklist"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/pool"
	"github.com/grafana/tempo/tempodb/wal"
)

type mockSharder struct {
}

func (m *mockSharder) Owns(hash string) bool {
	return true
}

func (m *mockSharder) Combine(dataEncoding string, objs ...[]byte) ([]byte, bool, error) {
	return model.ObjectCombiner.Combine(dataEncoding, objs...)
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
	tempDir, err := os.MkdirTemp("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	r, w, c, err := New(&Config{
		Backend: "local",
		Pool: &pool.Config{
			MaxWorkers: 10,
			QueueDepth: 100,
		},
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &encoding.BlockConfig{
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

	allReqs := make([]*tempopb.Trace, 0, blockCount*recordCount)
	allIds := make([][]byte, 0, blockCount*recordCount)

	for i := 0; i < blockCount; i++ {
		blockID := uuid.New()
		head, err := wal.NewBlock(blockID, testTenantID, "")
		require.NoError(t, err)

		reqs := make([]*tempopb.Trace, 0, recordCount)
		ids := make([][]byte, 0, recordCount)
		for j := 0; j < recordCount; j++ {
			id := make([]byte, 16)
			_, err = rand.Read(id)
			require.NoError(t, err, "unexpected creating random id")

			req := test.MakeTrace(10, id)
			reqs = append(reqs, req)
			ids = append(ids, id)

			bReq, err := proto.Marshal(req)
			require.NoError(t, err)
			err = head.Append(id, bReq)
			require.NoError(t, err, "unexpected error writing req")
		}
		allReqs = append(allReqs, reqs...)
		allIds = append(allIds, ids...)

		_, err = w.CompleteBlock(head, &mockSharder{})
		require.NoError(t, err)

		//err = w.WriteBlock(context.Background(), complete)
		//assert.NoError(t, err)
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
		b, _, failedBlocks, err := rw.Find(context.Background(), testTenantID, id, BlockIDMin, BlockIDMax)
		assert.NoError(t, err)
		assert.Nil(t, failedBlocks)

		out := &tempopb.Trace{}
		err = proto.Unmarshal(b[0], out)
		assert.NoError(t, err)

		assert.True(t, proto.Equal(allReqs[i], out))
	}
}

// TestSameIDCompaction is a bit gross in that it has a bad dependency with on the /pkg/model
// module to do a full e2e compaction/combination test.
func TestSameIDCompaction(t *testing.T) {
	tempDir, err := os.MkdirTemp("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	r, w, c, err := New(&Config{
		Backend: "local",
		Pool: &pool.Config{
			MaxWorkers: 10,
			QueueDepth: 100,
		},
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &encoding.BlockConfig{
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
	assert.NoError(t, err)

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      24 * time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})

	r.EnablePolling(&mockJobSharder{})

	wal := w.WAL()
	assert.NoError(t, err)

	blockCount := 5
	recordCount := 100

	// make a bunch of sharded requests
	allReqs := make([][][]byte, 0, recordCount)
	allIds := make([][]byte, 0, recordCount)
	for i := 0; i < recordCount; i++ {
		id := make([]byte, 16)
		_, err = rand.Read(id)
		require.NoError(t, err, "unexpected creating random id")

		requestShards := rand.Intn(blockCount) + 1

		reqs := make([][]byte, 0, requestShards)
		for j := 0; j < requestShards; j++ {
			buff, err := proto.Marshal(test.MakeTrace(1, id))
			require.NoError(t, err)
			reqs = append(reqs, buff)
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
				err = head.Append(id, req[i])
				require.NoError(t, err, "unexpected error writing req")
			}
		}

		_, err = w.CompleteBlock(head, &mockSharder{})
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
		b, _, failedBlocks, err := rw.Find(context.Background(), testTenantID, id, BlockIDMin, BlockIDMax)
		assert.NoError(t, err)
		assert.Nil(t, failedBlocks)

		actualBytes, _, err := model.ObjectCombiner.Combine(v1.Encoding, b...)
		require.NoError(t, err)

		expectedBytes, _, err := model.ObjectCombiner.Combine(v1.Encoding, allReqs[i]...)
		require.NoError(t, err)

		assert.Equal(t, expectedBytes, actualBytes)
	}
}

func TestCompactionUpdatesBlocklist(t *testing.T) {
	tempDir, err := os.MkdirTemp("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	r, w, c, err := New(&Config{
		Backend: "local",
		Pool: &pool.Config{
			MaxWorkers: 10,
			QueueDepth: 100,
		},
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &encoding.BlockConfig{
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

	// compact everything
	err = rw.compact(rw.blocklist.Metas(testTenantID), testTenantID)
	assert.NoError(t, err)

	// New blocklist contains 1 compacted block with everything
	blocks := rw.blocklist.Metas(testTenantID)
	assert.Equal(t, 1, len(blocks))
	assert.Equal(t, uint8(1), blocks[0].CompactionLevel)
	assert.Equal(t, blockCount*recordCount, blocks[0].TotalObjects)

	// Compacted list contains all old blocks
	assert.Equal(t, blockCount, len(rw.blocklist.CompactedMetas(testTenantID)))

	// Make sure all expected traces are found.
	for i := 0; i < blockCount; i++ {
		for j := 0; j < recordCount; j++ {
			trace, _, failedBlocks, err := rw.Find(context.TODO(), testTenantID, makeTraceID(i, j), BlockIDMin, BlockIDMax)
			assert.NotNil(t, trace)
			assert.Greater(t, len(trace), 0)
			assert.NoError(t, err)
			assert.Nil(t, failedBlocks)
		}
	}
}

func TestCompactionMetrics(t *testing.T) {
	tempDir, err := os.MkdirTemp("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	r, w, c, err := New(&Config{
		Backend: "local",
		Pool: &pool.Config{
			MaxWorkers: 10,
			QueueDepth: 100,
		},
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &encoding.BlockConfig{
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
	processedStart, err := test.GetCounterVecValue(metricCompactionObjectsWritten, "0")
	assert.NoError(t, err)

	blocksStart, err := test.GetCounterVecValue(metricCompactionBlocks, "0")
	assert.NoError(t, err)

	bytesStart, err := test.GetCounterVecValue(metricCompactionBytesWritten, "0")
	assert.NoError(t, err)

	// compact everything
	err = rw.compact(rw.blocklist.Metas(testTenantID), testTenantID)
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
	tempDir, err := os.MkdirTemp("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	r, w, c, err := New(&Config{
		Backend: "local",
		Pool: &pool.Config{
			MaxWorkers: 10,
			QueueDepth: 100,
		},
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &encoding.BlockConfig{
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

func cutTestBlocks(t testing.TB, w Writer, tenantID string, blockCount int, recordCount int) []*encoding.BackendBlock {
	blocks := make([]*encoding.BackendBlock, 0)

	wal := w.WAL()
	for i := 0; i < blockCount; i++ {
		head, err := wal.NewBlock(uuid.New(), tenantID, "")
		require.NoError(t, err)

		for j := 0; j < recordCount; j++ {
			// Use i and j to ensure unique ids
			body := make([]byte, 1024)
			rand.Read(body)

			err = head.Append(
				makeTraceID(i, j),
				body)
			//[]byte{0x01, 0x02, 0x03})
			require.NoError(t, err, "unexpected error writing rec")
		}

		b, err := w.CompleteBlock(head, &mockSharder{})
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
		Block: &encoding.BlockConfig{
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
