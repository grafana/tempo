package tempodb

import (
	"context"
	"encoding/binary"
	"path"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v2 "github.com/grafana/tempo/tempodb/encoding/v2"
	"github.com/grafana/tempo/tempodb/encoding/vparquet"
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

func (m *mockSharder) RecordDiscardedSpans(count int, tenantID string, traceID string) {}

type mockJobSharder struct{}

func (m *mockJobSharder) Owns(_ string) bool { return true }

type mockOverrides struct {
	blockRetention   time.Duration
	maxBytesPerTrace int
}

func (m *mockOverrides) BlockRetentionForTenant(_ string) time.Duration {
	return m.blockRetention
}

func (m *mockOverrides) MaxBytesPerTraceForTenant(_ string) int {
	return m.maxBytesPerTrace
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
			Version:              encoding.DefaultEncoding().Version(),
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
			Version:              encoding.DefaultEncoding().Version(),
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
			Version:              encoding.DefaultEncoding().Version(),
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

type testData struct {
	id         common.ID
	t          *tempopb.Trace
	start, end uint32
}

func cutTestBlockWithTraces(t testing.TB, w Writer, tenantID string, data []testData) common.BackendBlock {
	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	wal := w.WAL()

	head, err := wal.NewBlock(uuid.New(), tenantID, model.CurrentEncoding)
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
		head, err := wal.NewBlock(uuid.New(), tenantID, model.CurrentEncoding)
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
	testEncodings := []string{v2.VersionString, vparquet.VersionString}
	for _, enc := range testEncodings {
		b.Run(enc, func(b *testing.B) {
			benchmarkCompaction(b, enc)
		})
	}
}

func benchmarkCompaction(b *testing.B, targetBlockVersion string) {
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
			Version:              targetBlockVersion,
			Encoding:             backend.EncZstd,
			IndexPageSizeBytes:   1000,
			RowGroupSizeBytes:    30_000_000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	require.NoError(b, err)

	rw := c.(*readerWriter)

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:     10_000_000,
		FlushSizeBytes:     10_000_000,
		IteratorBufferSize: DefaultIteratorBufferSize,
	}, &mockSharder{}, &mockOverrides{})

	traceCount := 20_000
	blockCount := 8

	// Cut input blocks
	blocks := cutTestBlocks(b, w, testTenantID, blockCount, traceCount)
	metas := make([]*backend.BlockMeta, 0)
	for _, b := range blocks {
		metas = append(metas, b.BlockMeta())
	}

	b.ResetTimer()

	err = rw.compact(metas, testTenantID)
	require.NoError(b, err)
}
