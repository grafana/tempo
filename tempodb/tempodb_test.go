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

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
)

const (
	testTenantID     = "fake"
	testTenantID2    = "fake2"
	tmpdir           = "/tmp"
	testDataEncoding = "blerg"
)

func testConfig(t *testing.T, enc backend.Encoding, blocklistPoll time.Duration) (Reader, Writer, Compactor, string) {
	tempDir, err := os.MkdirTemp(tmpdir, "")
	require.NoError(t, err)

	r, w, c, err := New(&Config{
		Backend: "local",
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &encoding.BlockConfig{
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
	r, w, c, tempDir := testConfig(t, backend.EncGZIP, 0)
	defer os.RemoveAll(tempDir)

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})

	r.EnablePolling(&mockJobSharder{})

	blockID := uuid.New()

	wal := w.WAL()

	head, err := wal.NewBlock(blockID, testTenantID, testDataEncoding)
	assert.NoError(t, err)

	// write
	numMsgs := 10
	reqs := make([]*v1.ResourceSpans, 0, numMsgs)
	ids := make([][]byte, 0, numMsgs)
	for i := 0; i < numMsgs; i++ {
		id := make([]byte, 16)
		rand.Read(id)
		req := test.MakeBatch(rand.Int()%1000, id)
		reqs = append(reqs, req)
		ids = append(ids, id)

		bReq, err := proto.Marshal(req)
		assert.NoError(t, err)
		err = head.Append(id, bReq)
		assert.NoError(t, err, "unexpected error writing req")
	}

	_, err = w.CompleteBlock(head, &mockSharder{})
	assert.NoError(t, err)

	// poll
	r.(*readerWriter).pollBlocklist()

	// read
	for i, id := range ids {
		bFound, actualDataEncoding, failedBlocks, err := r.Find(context.Background(), testTenantID, id, BlockIDMin, BlockIDMax)
		assert.NoError(t, err)
		assert.Nil(t, failedBlocks)
		assert.Equal(t, []string{testDataEncoding}, actualDataEncoding)

		out := &v1.ResourceSpans{}
		err = proto.Unmarshal(bFound[0], out)
		assert.NoError(t, err)

		assert.True(t, proto.Equal(out, reqs[i]))
	}
}

func TestBlockSharding(t *testing.T) {
	// push a req with some traceID
	// cut headblock & write to backend
	// search with different shards and check if its respecting the params
	r, w, _, tempDir := testConfig(t, backend.EncLZ4_256k, 0)
	defer os.RemoveAll(tempDir)

	r.EnablePolling(&mockJobSharder{})

	// create block with known ID
	blockID := uuid.New()
	wal := w.WAL()

	head, err := wal.NewBlock(blockID, testTenantID, "")
	assert.NoError(t, err)

	// add a trace to the block
	id := make([]byte, 16)
	rand.Read(id)
	req := test.MakeTrace(rand.Int()%1000, id)

	bReq, err := proto.Marshal(req)
	assert.NoError(t, err)
	err = head.Append(id, bReq)
	assert.NoError(t, err, "unexpected error writing req")

	// write block to backend
	_, err = w.CompleteBlock(head, &mockSharder{})
	assert.NoError(t, err)

	// poll
	r.(*readerWriter).pollBlocklist()

	// get blockID
	blocks := r.(*readerWriter).blocklist.Metas(testTenantID)
	assert.Len(t, blocks, 1)

	// check if it respects the blockstart/blockend params - case1: hit
	blockStart := uuid.MustParse(BlockIDMin).String()
	blockEnd := uuid.MustParse(BlockIDMax).String()
	bFound, _, failedBlocks, err := r.Find(context.Background(), testTenantID, id, blockStart, blockEnd)
	assert.NoError(t, err)
	assert.Nil(t, failedBlocks)
	assert.Greater(t, len(bFound), 0)

	out := &tempopb.Trace{}
	err = proto.Unmarshal(bFound[0], out)
	assert.NoError(t, err)
	assert.True(t, proto.Equal(out, req))

	// check if it respects the blockstart/blockend params - case2: miss
	blockStart = uuid.MustParse(BlockIDMin).String()
	blockEnd = uuid.MustParse(BlockIDMin).String()
	bFound, _, failedBlocks, err = r.Find(context.Background(), testTenantID, id, blockStart, blockEnd)
	assert.NoError(t, err)
	assert.Nil(t, failedBlocks)
	assert.Len(t, bFound, 0)
}

func TestNilOnUnknownTenantID(t *testing.T) {
	r, _, _, tempDir := testConfig(t, backend.EncLZ4_256k, 0)
	defer os.RemoveAll(tempDir)

	buff, _, failedBlocks, err := r.Find(context.Background(), "unknown", []byte{0x01}, BlockIDMin, BlockIDMax)
	assert.Nil(t, buff)
	assert.Nil(t, err)
	assert.Nil(t, failedBlocks)
}

func TestBlockCleanup(t *testing.T) {
	r, w, c, tempDir := testConfig(t, backend.EncLZ4_256k, 0)
	defer os.RemoveAll(tempDir)

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

	_, err = w.CompleteBlock(head, &mockSharder{})
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
			expected: true,
		},
		{
			name:       "include - max block range",
			searchID:   []byte{0x05},
			blockStart: uuid.MustParse(BlockIDMin),
			blockEnd:   uuid.MustParse("50000000-0000-0000-0000-000000000000"),
			meta: &backend.BlockMeta{
				BlockID: uuid.MustParse("50000000-0000-0000-0000-000000000000"),
				MinID:   []byte{0x00},
				MaxID:   []byte{0x10},
			},
			expected: true,
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

			assert.Equal(t, tc.expected, includeBlock(tc.meta, tc.searchID, s, e))
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
		expected   bool
	}{
		{
			name:       "include recent",
			searchID:   []byte{0x05},
			blockStart: uuid.MustParse(BlockIDMin),
			blockEnd:   uuid.MustParse(BlockIDMax),
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

			assert.Equal(t, tc.expected, includeCompactedBlock(tc.meta, tc.searchID, s, e, blocklistPoll))
		})
	}

}

func TestSearchCompactedBlocks(t *testing.T) {
	r, w, c, tempDir := testConfig(t, backend.EncLZ4_256k, time.Minute)
	defer os.RemoveAll(tempDir)

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})

	r.EnablePolling(&mockJobSharder{})

	wal := w.WAL()

	head, err := wal.NewBlock(uuid.New(), testTenantID, "")
	assert.NoError(t, err)

	// write
	numMsgs := 10
	reqs := make([]*tempopb.Trace, 0, numMsgs)
	ids := make([][]byte, 0, numMsgs)
	for i := 0; i < numMsgs; i++ {
		id := make([]byte, 16)
		rand.Read(id)
		req := test.MakeTrace(rand.Int()%1000, id)
		reqs = append(reqs, req)
		ids = append(ids, id)

		bReq, err := proto.Marshal(req)
		assert.NoError(t, err)
		err = head.Append(id, bReq)
		assert.NoError(t, err, "unexpected error writing req")
	}

	complete, err := w.CompleteBlock(head, &mockSharder{})
	assert.NoError(t, err)

	blockID := complete.BlockMeta().BlockID.String()

	rw := r.(*readerWriter)

	// poll
	rw.pollBlocklist()

	// read
	for i, id := range ids {
		bFound, _, failedBlocks, err := r.Find(context.Background(), testTenantID, id, blockID, blockID)
		assert.NoError(t, err)
		assert.Nil(t, failedBlocks)

		out := &tempopb.Trace{}
		err = proto.Unmarshal(bFound[0], out)
		assert.NoError(t, err)

		assert.True(t, proto.Equal(out, reqs[i]))
	}

	// compact
	var blockMetas []*backend.BlockMeta
	blockMetas = append(blockMetas, complete.BlockMeta())
	assert.NoError(t, rw.compact(blockMetas, testTenantID))

	// poll
	rw.pollBlocklist()

	// make sure the block is compacted
	compactedBlocks := rw.blocklist.CompactedMetas(testTenantID)
	require.Len(t, compactedBlocks, 1)
	assert.Equal(t, compactedBlocks[0].BlockID.String(), blockID)
	blocks := rw.blocklist.Metas(testTenantID)
	require.Len(t, blocks, 1)
	assert.NotEqual(t, blocks[0].BlockID.String(), blockID)

	// find should succeed with old block range
	for i, id := range ids {
		bFound, _, failedBlocks, err := r.Find(context.Background(), testTenantID, id, blockID, blockID)
		assert.NoError(t, err)
		assert.Nil(t, failedBlocks)

		out := &tempopb.Trace{}
		err = proto.Unmarshal(bFound[0], out)
		assert.NoError(t, err)

		assert.True(t, proto.Equal(out, reqs[i]))
	}
}

func TestCompleteBlock(t *testing.T) {
	_, w, _, tempDir := testConfig(t, backend.EncLZ4_256k, time.Minute)
	defer os.RemoveAll(tempDir)

	wal := w.WAL()

	blockID := uuid.New()

	block, err := wal.NewBlock(blockID, testTenantID, "")
	assert.NoError(t, err, "unexpected error creating block")

	numMsgs := 100
	reqs := make([]*tempopb.Trace, 0, numMsgs)
	ids := make([][]byte, 0, numMsgs)
	for i := 0; i < numMsgs; i++ {
		id := make([]byte, 16)
		rand.Read(id)
		req := test.MakeTrace(rand.Int()%1000, id)
		reqs = append(reqs, req)
		ids = append(ids, id)
		bReq, err := proto.Marshal(req)
		assert.NoError(t, err)
		err = block.Append(id, bReq)
		assert.NoError(t, err, "unexpected error writing req")
	}

	complete, err := w.CompleteBlock(block, &mockSharder{})
	require.NoError(t, err, "unexpected error completing block")

	for i, id := range ids {
		out := &tempopb.Trace{}
		foundBytes, err := complete.Find(context.TODO(), id)
		assert.NoError(t, err)

		err = proto.Unmarshal(foundBytes, out)
		assert.NoError(t, err)

		assert.True(t, proto.Equal(out, reqs[i]))
	}
}

func TestShouldCache(t *testing.T) {
	tempDir, err := os.MkdirTemp(tmpdir, "")
	defer os.RemoveAll(tempDir)
	require.NoError(t, err)

	r, _, _, err := New(&Config{
		Backend: "local",
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &encoding.BlockConfig{
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
