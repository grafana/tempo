package tempodb

import (
	"context"
	"encoding/json"
	"math/rand"
	"path"
	"testing"
	"time"

	"github.com/go-kit/log"
	proto "github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
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
	"github.com/stretchr/testify/require"
)

func TestIntegrationCompactionRoundtrip(t *testing.T) {
	for _, enc := range encoding.AllEncodings() {
		t.Run(enc.Version(), func(t *testing.T) {
			testCompactionRoundtrip(t, enc.Version())
		})
	}
}

func testCompactionRoundtrip(t *testing.T, targetBlockVersion string) {
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
			Version:              targetBlockVersion,
			Encoding:             backend.EncLZ4_4M,
			IndexPageSizeBytes:   1000,
			RowGroupSizeBytes:    30_000_000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	require.NoError(t, err)

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:          10_000_000,
		FlushSizeBytes:          10_000_000,
		MaxCompactionRange:      24 * time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})

	r.EnablePolling(&mockJobSharder{})

	wal := w.WAL()
	require.NoError(t, err)

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

		_, err = w.CompleteBlock(context.Background(), head)
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
		err := rw.compact(blocks, testTenantID)
		require.NoError(t, err)

		expectedBlockCount -= blocksPerCompaction
		expectedCompactedCount += inputBlocks
		checkBlocklists(t, uuid.Nil, expectedBlockCount, expectedCompactedCount, rw)
	}

	require.Equal(t, expectedCompactions, compactions)

	// do we have the right number of records
	var records int
	for _, meta := range rw.blocklist.Metas(testTenantID) {
		records += meta.TotalObjects
	}
	require.Equal(t, blockCount*recordCount, records)

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

func TestIntegrationSameIDCompaction(t *testing.T) {
	for _, enc := range encoding.AllEncodings() {
		t.Run(enc.Version(), func(t *testing.T) {
			testSameIDCompaction(t, enc.Version())
		})
	}
}

// TestSameIDCompaction is a bit gross in that it has a bad dependency with on the /pkg/model
// module to do a full e2e compaction/combination test.
func testSameIDCompaction(t *testing.T, targetBlockVersion string) {
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
			Version:              targetBlockVersion,
			Encoding:             backend.EncSnappy,
			IndexPageSizeBytes:   1000,
			RowGroupSizeBytes:    30_000_000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	require.NoError(t, err)

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:          10_000_000,
		MaxCompactionRange:      24 * time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
		FlushSizeBytes:          10_000_000,
	}, &mockSharder{}, &mockOverrides{})

	r.EnablePolling(&mockJobSharder{})

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

	err = rw.compact(blocks, testTenantID)
	require.NoError(t, err)

	checkBlocklists(t, uuid.Nil, 1, blockCount, rw)

	// force clear compacted blocks to guarantee that we're only querying the new blocks that went through the combiner
	metas := rw.blocklist.Metas(testTenantID)
	rw.blocklist.ApplyPollResults(blocklist.PerTenant{testTenantID: metas}, blocklist.PerTenantCompacted{})

	// search for all ids
	for i, id := range allIds {
		trs, failedBlocks, err := rw.Find(context.Background(), testTenantID, id, BlockIDMin, BlockIDMax, 0, 0)
		require.NoError(t, err)
		require.Nil(t, failedBlocks)

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

	combinedEnd, err := test.GetCounterVecValue(metricCompactionObjectsCombined, "0")
	require.NoError(t, err)
	require.Equal(t, float64(sharded), combinedEnd-combinedStart)
}

func TestIntegrationCompactionHonorsBlockStartEndTimes(t *testing.T) {
	for _, enc := range encoding.AllEncodings() {
		t.Run(enc.Version(), func(t *testing.T) {
			testCompactionHonorsBlockStartEndTimes(t, enc.Version())
		})
	}
}

func testCompactionHonorsBlockStartEndTimes(t *testing.T, targetBlockVersion string) {
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
	}, log.NewNopLogger())
	require.NoError(t, err)

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:          10_000_000,
		FlushSizeBytes:          10_000_000,
		MaxCompactionRange:      24 * time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})

	r.EnablePolling(&mockJobSharder{})

	cutTestBlockWithTraces(t, w, testTenantID, []testData{
		{test.ValidTraceID(nil), test.MakeTrace(10, nil), 100, 101},
		{test.ValidTraceID(nil), test.MakeTrace(10, nil), 102, 103},
	})
	cutTestBlockWithTraces(t, w, testTenantID, []testData{
		{test.ValidTraceID(nil), test.MakeTrace(10, nil), 104, 105},
		{test.ValidTraceID(nil), test.MakeTrace(10, nil), 106, 107},
	})

	rw := r.(*readerWriter)
	rw.pollBlocklist()

	// compact everything
	err = rw.compact(rw.blocklist.Metas(testTenantID), testTenantID)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// New blocklist contains 1 compacted block with min start and max end
	blocks := rw.blocklist.Metas(testTenantID)
	require.Equal(t, 1, len(blocks))
	require.Equal(t, uint8(1), blocks[0].CompactionLevel)
	require.Equal(t, 100, int(blocks[0].StartTime.Unix()))
	require.Equal(t, 107, int(blocks[0].EndTime.Unix()))
}
