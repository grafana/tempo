package tempodb

import (
	"context"
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
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
)

const (
	testTenantID  = "fake"
	testTenantID2 = "fake2"
)

func testConfig(t *testing.T, enc backend.Encoding, blocklistPoll time.Duration) (Reader, Writer, Compactor, string) {
	tempDir := t.TempDir()

	r, w, c, err := New(&Config{
		Backend: "local",
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &common.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              .01,
			BloomShardSizeBytes:  100_000,
			Encoding:             enc,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: blocklistPoll,
	}, log.NewNopLogger())
	require.NoError(t, err)
	return r, w, c, tempDir
}

func TestDB(t *testing.T) {
	r, w, c, _ := testConfig(t, backend.EncGZIP, 0)

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})

	r.EnablePolling(&mockJobSharder{})

	blockID := uuid.New()

	wal := w.WAL()

	head, err := wal.NewBlock(blockID, testTenantID, model.CurrentEncoding)
	assert.NoError(t, err)

	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	// write
	numMsgs := 10
	reqs := make([]*tempopb.Trace, numMsgs)
	ids := make([]common.ID, numMsgs)
	for i := 0; i < numMsgs; i++ {
		ids[i] = test.ValidTraceID(nil)
		reqs[i] = test.MakeTrace(10, ids[i])
		writeTraceToWal(t, head, dec, ids[i], reqs[i], 0, 0)
	}

	_, err = w.CompleteBlock(head, &mockCombiner{})
	assert.NoError(t, err)

	// poll
	r.(*readerWriter).pollBlocklist()

	// read
	for i, id := range ids {
		bFound, failedBlocks, err := r.Find(context.Background(), testTenantID, id, BlockIDMin, BlockIDMax, 0, 0)
		assert.NoError(t, err)
		assert.Nil(t, failedBlocks)
		assert.True(t, proto.Equal(bFound[0], reqs[i]))
	}
}

func TestBlockSharding(t *testing.T) {
	// push a req with some traceID
	// cut headblock & write to backend
	// search with different shards and check if its respecting the params
	r, w, _, _ := testConfig(t, backend.EncLZ4_256k, 0)

	r.EnablePolling(&mockJobSharder{})

	// create block with known ID
	blockID := uuid.New()
	wal := w.WAL()

	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)
	head, err := wal.NewBlock(blockID, testTenantID, model.CurrentEncoding)
	assert.NoError(t, err)

	// add a trace to the block
	id := test.ValidTraceID(nil)
	req := test.MakeTrace(1, id)
	writeTraceToWal(t, head, dec, id, req, 0, 0)

	// write block to backend
	_, err = w.CompleteBlock(head, &mockCombiner{})
	assert.NoError(t, err)

	// poll
	r.(*readerWriter).pollBlocklist()

	// get blockID
	blocks := r.(*readerWriter).blocklist.Metas(testTenantID)
	assert.Len(t, blocks, 1)

	// check if it respects the blockstart/blockend params - case1: hit
	blockStart := uuid.MustParse(BlockIDMin).String()
	blockEnd := uuid.MustParse(BlockIDMax).String()
	bFound, failedBlocks, err := r.Find(context.Background(), testTenantID, id, blockStart, blockEnd, 0, 0)
	assert.NoError(t, err)
	assert.Nil(t, failedBlocks)
	assert.Greater(t, len(bFound), 0)
	assert.True(t, proto.Equal(bFound[0], req))

	// check if it respects the blockstart/blockend params - case2: miss
	blockStart = uuid.MustParse(BlockIDMin).String()
	blockEnd = uuid.MustParse(BlockIDMin).String()
	bFound, failedBlocks, err = r.Find(context.Background(), testTenantID, id, blockStart, blockEnd, 0, 0)
	assert.NoError(t, err)
	assert.Nil(t, failedBlocks)
	assert.Len(t, bFound, 0)
}

func TestNilOnUnknownTenantID(t *testing.T) {
	r, _, _, _ := testConfig(t, backend.EncLZ4_256k, 0)

	buff, failedBlocks, err := r.Find(context.Background(), "unknown", []byte{0x01}, BlockIDMin, BlockIDMax, 0, 0)
	assert.Nil(t, buff)
	assert.Nil(t, err)
	assert.Nil(t, failedBlocks)
}

func TestBlockCleanup(t *testing.T) {
	r, w, c, tempDir := testConfig(t, backend.EncLZ4_256k, 0)

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})

	r.EnablePolling(&mockJobSharder{})

	blockID := uuid.New()

	wal := w.WAL()

	head, err := wal.NewBlock(blockID, testTenantID, "")
	assert.NoError(t, err)

	_, err = w.CompleteBlock(head, &mockCombiner{})
	assert.NoError(t, err)

	rw := r.(*readerWriter)

	// poll
	rw.pollBlocklist()

	assert.Len(t, rw.blocklist.Metas(testTenantID), 1)

	os.RemoveAll(tempDir + "/traces/" + testTenantID)

	// poll
	rw.pollBlocklist()

	m := rw.blocklist.Metas(testTenantID)
	assert.Equal(t, 0, len(m))
}

func checkBlocklists(t *testing.T, expectedID uuid.UUID, expectedB int, expectedCB int, rw *readerWriter) {
	rw.pollBlocklist()

	blocklist := rw.blocklist.Metas(testTenantID)
	assert.Len(t, blocklist, expectedB)
	if expectedB > 0 && expectedID != uuid.Nil {
		assert.Equal(t, expectedID, blocklist[0].BlockID)
	}

	//confirm blocklists are in starttime ascending order
	lastTime := time.Time{}
	for _, b := range blocklist {
		assert.True(t, lastTime.Before(b.StartTime) || lastTime.Equal(b.StartTime))
		lastTime = b.StartTime
	}

	compactedBlocklist := rw.blocklist.CompactedMetas(testTenantID)
	assert.Len(t, compactedBlocklist, expectedCB)
	if expectedCB > 0 && expectedID != uuid.Nil {
		assert.Equal(t, expectedID, compactedBlocklist[0].BlockID)
	}

	lastTime = time.Time{}
	for _, b := range compactedBlocklist {
		assert.True(t, lastTime.Before(b.StartTime) || lastTime.Equal(b.StartTime))
		lastTime = b.StartTime
	}
}

func TestIncludeBlock(t *testing.T) {
	tests := []struct {
		name       string
		searchID   common.ID
		blockStart uuid.UUID
		blockEnd   uuid.UUID
		start      int64
		end        int64
		meta       *backend.BlockMeta
		expected   bool
	}{
		// includes
		{
			name:       "include - duh",
			searchID:   []byte{0x05},
			blockStart: uuid.MustParse(BlockIDMin),
			blockEnd:   uuid.MustParse(BlockIDMax),
			meta: &backend.BlockMeta{
				BlockID: uuid.MustParse("50000000-0000-0000-0000-000000000000"),
				MinID:   []byte{0x00},
				MaxID:   []byte{0x10},
			},
			start:    0,
			end:      0,
			expected: true,
		},
		{
			name:       "include - min id range",
			searchID:   []byte{0x00},
			blockStart: uuid.MustParse(BlockIDMin),
			blockEnd:   uuid.MustParse(BlockIDMax),
			meta: &backend.BlockMeta{
				BlockID: uuid.MustParse("50000000-0000-0000-0000-000000000000"),
				MinID:   []byte{0x00},
				MaxID:   []byte{0x10},
			},
			start:    0,
			end:      0,
			expected: true,
		},
		{
			name:       "include - max id range",
			searchID:   []byte{0x10},
			blockStart: uuid.MustParse(BlockIDMin),
			blockEnd:   uuid.MustParse(BlockIDMax),
			meta: &backend.BlockMeta{
				BlockID: uuid.MustParse("50000000-0000-0000-0000-000000000000"),
				MinID:   []byte{0x00},
				MaxID:   []byte{0x10},
			},
			start:    0,
			end:      0,
			expected: true,
		},
		{
			name:       "include - min block range",
			searchID:   []byte{0x05},
			blockStart: uuid.MustParse("50000000-0000-0000-0000-000000000000"),
			blockEnd:   uuid.MustParse(BlockIDMax),
			meta: &backend.BlockMeta{
				BlockID: uuid.MustParse("50000000-0000-0000-0000-000000000000"),
				MinID:   []byte{0x00},
				MaxID:   []byte{0x10},
			},
			start:    0,
			end:      0,
			expected: true,
		},
		{
			name:       "include - max block range",
			searchID:   []byte{0x05},
			blockStart: uuid.MustParse(BlockIDMin),
			blockEnd:   uuid.MustParse("50000000-0000-0000-0000-000000000000"),
			meta: &backend.BlockMeta{
				BlockID:   uuid.MustParse("50000000-0000-0000-0000-000000000000"),
				MinID:     []byte{0x00},
				MaxID:     []byte{0x10},
				StartTime: time.Unix(10000, 0),
				EndTime:   time.Unix(20000, 0),
			},
			start:    10000,
			end:      20000,
			expected: true,
		},
		{
			name:       "include - max block range",
			searchID:   []byte{0x05},
			blockStart: uuid.MustParse(BlockIDMin),
			blockEnd:   uuid.MustParse("50000000-0000-0000-0000-000000000000"),
			meta: &backend.BlockMeta{
				BlockID:   uuid.MustParse("50000000-0000-0000-0000-000000000000"),
				MinID:     []byte{0x00},
				MaxID:     []byte{0x10},
				StartTime: time.Unix(1650285326, 0),
				EndTime:   time.Unix(1650288990, 0),
			},
			start:    10000,
			end:      20000,
			expected: false,
		},
		{
			name:       "include - exact hit",
			searchID:   []byte{0x05},
			blockStart: uuid.MustParse("50000000-0000-0000-0000-000000000000"),
			blockEnd:   uuid.MustParse("50000000-0000-0000-0000-000000000000"),
			meta: &backend.BlockMeta{
				BlockID: uuid.MustParse("50000000-0000-0000-0000-000000000000"),
				MinID:   []byte{0x05},
				MaxID:   []byte{0x05},
			},
			start:    0,
			end:      0,
			expected: true,
		},
		// excludes
		{
			name:       "exclude - duh",
			searchID:   []byte{0x20},
			blockStart: uuid.MustParse("50000000-0000-0000-0000-000000000000"),
			blockEnd:   uuid.MustParse("51000000-0000-0000-0000-000000000000"),
			meta: &backend.BlockMeta{
				BlockID: uuid.MustParse("52000000-0000-0000-0000-000000000000"),
				MinID:   []byte{0x00},
				MaxID:   []byte{0x10},
			},
		},
		{
			name:       "exclude - min id range",
			searchID:   []byte{0x00},
			blockStart: uuid.MustParse(BlockIDMin),
			blockEnd:   uuid.MustParse(BlockIDMax),
			meta: &backend.BlockMeta{
				BlockID: uuid.MustParse("50000000-0000-0000-0000-000000000000"),
				MinID:   []byte{0x01},
				MaxID:   []byte{0x10},
			},
		},
		{
			name:       "exclude - max id range",
			searchID:   []byte{0x11},
			blockStart: uuid.MustParse(BlockIDMin),
			blockEnd:   uuid.MustParse(BlockIDMax),
			meta: &backend.BlockMeta{
				BlockID: uuid.MustParse("50000000-0000-0000-0000-000000000000"),
				MinID:   []byte{0x01},
				MaxID:   []byte{0x10},
			},
		},
		{
			name:       "exclude - min block range",
			searchID:   []byte{0x05},
			blockStart: uuid.MustParse("50000000-0000-0000-0000-000000000000"),
			blockEnd:   uuid.MustParse("51000000-0000-0000-0000-000000000000"),
			meta: &backend.BlockMeta{
				BlockID: uuid.MustParse("4FFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF"),
				MinID:   []byte{0x00},
				MaxID:   []byte{0x10},
			},
		},
		{
			name:       "exclude - max block range",
			searchID:   []byte{0x05},
			blockStart: uuid.MustParse("50000000-0000-0000-0000-000000000000"),
			blockEnd:   uuid.MustParse("51000000-0000-0000-0000-000000000000"),
			meta: &backend.BlockMeta{
				BlockID: uuid.MustParse("51000000-0000-0000-0000-000000000001"),
				MinID:   []byte{0x00},
				MaxID:   []byte{0x10},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, err := tc.blockStart.MarshalBinary()
			require.NoError(t, err)
			e, err := tc.blockEnd.MarshalBinary()
			require.NoError(t, err)

			assert.Equal(t, tc.expected, includeBlock(tc.meta, tc.searchID, s, e, tc.start, tc.end))
		})
	}
}

func TestIncludeCompactedBlock(t *testing.T) {
	blocklistPoll := 5 * time.Minute
	tests := []struct {
		name       string
		searchID   common.ID
		blockStart uuid.UUID
		blockEnd   uuid.UUID
		meta       *backend.CompactedBlockMeta
		start      int64
		end        int64
		expected   bool
	}{
		{
			name:       "include recent",
			searchID:   []byte{0x05},
			blockStart: uuid.MustParse(BlockIDMin),
			blockEnd:   uuid.MustParse(BlockIDMax),
			start:      0,
			end:        0,
			meta: &backend.CompactedBlockMeta{
				BlockMeta: backend.BlockMeta{
					BlockID: uuid.MustParse("50000000-0000-0000-0000-000000000000"),
					MinID:   []byte{0x00},
					MaxID:   []byte{0x10},
				},
				CompactedTime: time.Now().Add(-(1 * blocklistPoll)),
			},
			expected: true,
		},
		{
			name:       "skip old",
			searchID:   []byte{0x05},
			blockStart: uuid.MustParse(BlockIDMin),
			blockEnd:   uuid.MustParse(BlockIDMax),
			start:      0,
			end:        0,
			meta: &backend.CompactedBlockMeta{
				BlockMeta: backend.BlockMeta{
					BlockID: uuid.MustParse("50000000-0000-0000-0000-000000000000"),
					MinID:   []byte{0x00},
					MaxID:   []byte{0x10},
				},
				CompactedTime: time.Now().Add(-(3 * blocklistPoll)),
			},
			expected: false,
		},
		{
			name:       "skip recent but out of range",
			searchID:   []byte{0x05},
			blockStart: uuid.MustParse("40000000-0000-0000-0000-000000000000"),
			blockEnd:   uuid.MustParse("50000000-0000-0000-0000-000000000000"),
			start:      0,
			end:        0,
			meta: &backend.CompactedBlockMeta{
				BlockMeta: backend.BlockMeta{
					BlockID: uuid.MustParse("51000000-0000-0000-0000-000000000000"),
					MinID:   []byte{0x00},
					MaxID:   []byte{0x10},
				},
				CompactedTime: time.Now().Add(-(1 * blocklistPoll)),
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, err := tc.blockStart.MarshalBinary()
			require.NoError(t, err)
			e, err := tc.blockEnd.MarshalBinary()
			require.NoError(t, err)

			assert.Equal(t, tc.expected, includeCompactedBlock(tc.meta, tc.searchID, s, e, blocklistPoll, tc.start, tc.end))
		})
	}

}

func TestSearchCompactedBlocks(t *testing.T) {
	r, w, c, _ := testConfig(t, backend.EncLZ4_256k, time.Minute)

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})

	r.EnablePolling(&mockJobSharder{})

	wal := w.WAL()

	head, err := wal.NewBlock(uuid.New(), testTenantID, model.CurrentEncoding)
	assert.NoError(t, err)

	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	// write
	numMsgs := 10
	reqs := make([]*tempopb.Trace, 0, numMsgs)
	ids := make([][]byte, 0, numMsgs)
	for i := 0; i < numMsgs; i++ {
		id := test.ValidTraceID(nil)
		req := test.MakeTrace(rand.Int()%1000, id)
		writeTraceToWal(t, head, dec, id, req, 0, 0)
		reqs = append(reqs, req)
		ids = append(ids, id)
	}

	complete, err := w.CompleteBlock(head, &mockCombiner{})
	require.NoError(t, err)

	blockID := complete.BlockMeta().BlockID.String()

	rw := r.(*readerWriter)

	// poll
	rw.pollBlocklist()

	// read
	for i, id := range ids {
		bFound, failedBlocks, err := r.Find(context.Background(), testTenantID, id, blockID, blockID, 0, 0)
		require.NoError(t, err)
		require.Nil(t, failedBlocks)
		require.True(t, proto.Equal(bFound[0], reqs[i]))
	}

	// compact
	var blockMetas []*backend.BlockMeta
	blockMetas = append(blockMetas, complete.BlockMeta())
	require.NoError(t, rw.compact(blockMetas, testTenantID))

	// poll
	rw.pollBlocklist()

	// make sure the block is compacted
	compactedBlocks := rw.blocklist.CompactedMetas(testTenantID)
	require.Len(t, compactedBlocks, 1)
	require.Equal(t, compactedBlocks[0].BlockID.String(), blockID)
	blocks := rw.blocklist.Metas(testTenantID)
	require.Len(t, blocks, 1)
	require.NotEqual(t, blocks[0].BlockID.String(), blockID)

	// find should succeed with old block range
	for i, id := range ids {
		bFound, failedBlocks, err := r.Find(context.Background(), testTenantID, id, blockID, blockID, 0, 0)
		require.NoError(t, err)
		require.Nil(t, failedBlocks)
		require.True(t, proto.Equal(bFound[0], reqs[i]))
	}
}

func TestCompleteBlock(t *testing.T) {
	_, w, _, _ := testConfig(t, backend.EncLZ4_256k, time.Minute)

	wal := w.WAL()

	blockID := uuid.New()

	block, err := wal.NewBlock(blockID, testTenantID, model.CurrentEncoding)
	require.NoError(t, err, "unexpected error creating block")

	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	numMsgs := 100
	reqs := make([]*tempopb.Trace, 0, numMsgs)
	ids := make([][]byte, 0, numMsgs)
	for i := 0; i < numMsgs; i++ {
		id := test.ValidTraceID(nil)
		req := test.MakeTrace(rand.Int()%1000, id)
		writeTraceToWal(t, block, dec, id, req, 0, 0)
		reqs = append(reqs, req)
		ids = append(ids, id)
	}

	complete, err := w.CompleteBlock(block, &mockCombiner{})
	require.NoError(t, err, "unexpected error completing block")

	for i, id := range ids {
		found, err := complete.FindTraceByID(context.TODO(), id)
		require.NoError(t, err)
		require.True(t, proto.Equal(found, reqs[i]))
	}
}

func TestShouldCache(t *testing.T) {
	tempDir := t.TempDir()

	r, _, _, err := New(&Config{
		Backend: "local",
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &common.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              .01,
			BloomShardSizeBytes:  100_000,
			Encoding:             backend.EncLZ4_256k,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll:           0,
		CacheMaxBlockAge:        time.Hour,
		CacheMinCompactionLevel: 1,
	}, log.NewNopLogger())
	require.NoError(t, err)

	rw := r.(*readerWriter)

	testCases := []struct {
		name            string
		compactionLevel uint8
		startTime       time.Time
		cache           bool
	}{
		{
			name:            "both pass",
			compactionLevel: 1,
			startTime:       time.Now(),
			cache:           true,
		},
		{
			name:            "startTime fail",
			compactionLevel: 2,
			startTime:       time.Now().Add(-2 * time.Hour),
			cache:           false,
		},
		{
			name:            "compactionLevel fail",
			compactionLevel: 0,
			startTime:       time.Now(),
			cache:           false,
		},
		{
			name:            "both fail",
			compactionLevel: 0,
			startTime:       time.Now(),
			cache:           false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.cache, rw.shouldCache(&backend.BlockMeta{CompactionLevel: tt.compactionLevel, StartTime: tt.startTime}, time.Now()))
		})
	}
}

func writeTraceToWal(t require.TestingT, b *wal.AppendBlock, dec model.SegmentDecoder, id common.ID, tr *tempopb.Trace, start, end uint32) {
	b1, err := dec.PrepareForWrite(tr, 0, 0)
	require.NoError(t, err)

	b2, err := dec.ToObject([][]byte{b1})
	require.NoError(t, err)

	err = b.Append(id, b2, start, end)
	require.NoError(t, err, "unexpected error writing req")
}
