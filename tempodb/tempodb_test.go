package tempodb

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/golang/protobuf/proto" //nolint:all
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
)

const (
	testTenantID  = "fake"
	testTenantID2 = "fake2"
)

type testConfigOption func(*Config)

func testConfig(
	t *testing.T,
	enc backend.Encoding,
	blocklistPoll time.Duration,
	opts ...testConfigOption,
) (*tempoDB, string) {
	tempDir := t.TempDir()

	cfg := &Config{
		Backend: backend.Local,
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &common.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              .01,
			BloomShardSizeBytes:  100_000,
			Version:              encoding.DefaultEncoding().Version(),
			Encoding:             enc,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: blocklistPoll,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	db, err := New(cfg, log.NewNopLogger())
	require.NoError(t, err)
	return db, tempDir
}

func TestDB(t *testing.T) {
	db, _ := testConfig(t, backend.EncGZIP, 0)

	err := db.EnableCompaction(context.Background(), &CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	db.EnablePolling(context.Background(), &mockJobSharder{})

	blockID := uuid.New()

	wal := db.WAL()

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

	_, err = db.CompleteBlock(context.Background(), head)
	assert.NoError(t, err)

	// poll
	db.pollBlocklist(context.Background())

	// read
	for i, id := range ids {
		bFound, failedBlocks, err := db.Find(context.Background(), testTenantID, id, BlockIDMin, BlockIDMax, 0, 0)
		assert.NoError(t, err)
		assert.Nil(t, failedBlocks)
		assert.True(t, proto.Equal(bFound[0], reqs[i]))
	}
}

func TestNoCompactionWhenCompactionRange0(t *testing.T) {
	db, _ := testConfig(t, backend.EncGZIP, 0)

	err := db.EnableCompaction(context.Background(), &CompactorConfig{
		MaxCompactionRange: 0,
	}, &mockSharder{}, &mockOverrides{})
	require.Error(t, err)
}

func TestBlockSharding(t *testing.T) {
	ctx := context.Background()

	// push a req with some traceID
	// cut headblock & write to backend
	// search with different shards and check if its respecting the params
	db, _ := testConfig(t, backend.EncLZ4_256k, 0)

	db.EnablePolling(ctx, &mockJobSharder{})

	// create block with known ID
	blockID := uuid.New()
	wal := db.WAL()

	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)
	head, err := wal.NewBlock(blockID, testTenantID, model.CurrentEncoding)
	assert.NoError(t, err)

	// add a trace to the block
	id := test.ValidTraceID(nil)
	req := test.MakeTrace(1, id)
	writeTraceToWal(t, head, dec, id, req, 0, 0)

	// write block to backend
	_, err = db.CompleteBlock(ctx, head)
	assert.NoError(t, err)

	// poll
	db.pollBlocklist(ctx)

	// get blockID
	blocks := db.blocklist.Metas(testTenantID)
	assert.Len(t, blocks, 1)

	// check if it respects the blockstart/blockend params - case1: hit
	blockStart := uuid.MustParse(BlockIDMin).String()
	blockEnd := uuid.MustParse(BlockIDMax).String()
	bFound, failedBlocks, err := db.Find(context.Background(), testTenantID, id, blockStart, blockEnd, 0, 0)
	assert.NoError(t, err)
	assert.Nil(t, failedBlocks)
	assert.Greater(t, len(bFound), 0)
	assert.True(t, proto.Equal(bFound[0], req))

	// check if it respects the blockstart/blockend params - case2: miss
	blockStart = uuid.MustParse(BlockIDMin).String()
	blockEnd = uuid.MustParse(BlockIDMin).String()
	bFound, failedBlocks, err = db.Find(context.Background(), testTenantID, id, blockStart, blockEnd, 0, 0)
	assert.NoError(t, err)
	assert.Nil(t, failedBlocks)
	assert.Len(t, bFound, 0)
}

func TestNilOnUnknownTenantID(t *testing.T) {
	db, _ := testConfig(t, backend.EncLZ4_256k, 0)

	buff, failedBlocks, err := db.Find(context.Background(), "unknown", []byte{0x01}, BlockIDMin, BlockIDMax, 0, 0)
	assert.Nil(t, buff)
	assert.Nil(t, err)
	assert.Nil(t, failedBlocks)
}

func TestBlockCleanup(t *testing.T) {
	db, tempDir := testConfig(t, backend.EncLZ4_256k, 0)

	err := db.EnableCompaction(context.Background(), &CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	ctx := context.Background()
	db.EnablePolling(ctx, &mockJobSharder{})

	blockID := uuid.New()

	wal := db.WAL()

	head, err := wal.NewBlock(blockID, testTenantID, model.CurrentEncoding)
	assert.NoError(t, err)

	_, err = db.CompleteBlock(context.Background(), head)
	assert.NoError(t, err)

	// poll
	db.pollBlocklist(ctx)

	assert.Len(t, db.blocklist.Metas(testTenantID), 1)

	os.RemoveAll(tempDir + "/traces/" + testTenantID)

	// poll
	db.pollBlocklist(ctx)

	m := db.blocklist.Metas(testTenantID)
	assert.Equal(t, 0, len(m))
}

func checkBlocklists(t *testing.T, expectedID uuid.UUID, expectedB int, expectedCB int, db *tempoDB) {
	db.pollBlocklist(context.Background())

	blocklist := db.blocklist.Metas(testTenantID)
	require.Len(t, blocklist, expectedB)
	if expectedB > 0 && expectedID != uuid.Nil {
		assert.Equal(t, expectedID, blocklist[0].BlockID)
	}

	// confirm blocklists are in starttime ascending order
	lastTime := time.Time{}
	for _, b := range blocklist {
		assert.True(t, lastTime.Before(b.StartTime) || lastTime.Equal(b.StartTime))
		lastTime = b.StartTime
	}

	compactedBlocklist := db.blocklist.CompactedMetas(testTenantID)
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
		// todo: restore when this is fixed: https://github.com/grafana/tempo/issues/1903
		// {
		// 	name:       "exclude - min id range",
		// 	searchID:   []byte{0x00},
		// 	blockStart: uuid.MustParse(BlockIDMin),
		// 	blockEnd:   uuid.MustParse(BlockIDMax),
		// 	meta: &backend.BlockMeta{
		// 		BlockID: uuid.MustParse("50000000-0000-0000-0000-000000000000"),
		// 		MinID:   []byte{0x01},
		// 		MaxID:   []byte{0x10},
		// 	},
		// },
		// {
		// 	name:       "exclude - max id range",
		// 	searchID:   []byte{0x11},
		// 	blockStart: uuid.MustParse(BlockIDMin),
		// 	blockEnd:   uuid.MustParse(BlockIDMax),
		// 	meta: &backend.BlockMeta{
		// 		BlockID: uuid.MustParse("50000000-0000-0000-0000-000000000000"),
		// 		MinID:   []byte{0x01},
		// 		MaxID:   []byte{0x10},
		// 	},
		// },
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
	db, _ := testConfig(t, backend.EncLZ4_256k, time.Hour)

	err := db.EnableCompaction(context.Background(), &CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	db.EnablePolling(context.Background(), &mockJobSharder{})

	wal := db.WAL()

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

	ctx := context.Background()
	complete, err := db.CompleteBlock(ctx, head)
	require.NoError(t, err)

	blockID := complete.BlockMeta().BlockID.String()

	// poll
	db.pollBlocklist(ctx)

	// read
	for i, id := range ids {
		bFound, failedBlocks, err := db.Find(ctx, testTenantID, id, blockID, blockID, 0, 0)
		require.NoError(t, err)
		require.Nil(t, failedBlocks)
		require.True(t, proto.Equal(bFound[0], reqs[i]))
	}

	// compact
	var blockMetas []*backend.BlockMeta
	blockMetas = append(blockMetas, complete.BlockMeta())
	require.NoError(t, db.compact(ctx, blockMetas, testTenantID))

	// poll
	db.pollBlocklist(ctx)

	// make sure the block is compacted
	compactedBlocks := db.blocklist.CompactedMetas(testTenantID)
	require.Len(t, compactedBlocks, 1)
	require.Equal(t, compactedBlocks[0].BlockID.String(), blockID)
	blocks := db.blocklist.Metas(testTenantID)
	require.Len(t, blocks, 1)
	require.NotEqual(t, blocks[0].BlockID.String(), blockID)

	// find should succeed with old block range
	for i, id := range ids {
		bFound, failedBlocks, err := db.Find(ctx, testTenantID, id, blockID, blockID, 0, 0)
		require.NoError(t, err)
		require.Nil(t, failedBlocks)
		require.True(t, proto.Equal(bFound[0], reqs[i]))
	}
}

func TestCompleteBlock(t *testing.T) {
	for _, from := range encoding.AllEncodings() {
		for _, to := range encoding.AllEncodings() {
			t.Run(fmt.Sprintf("%s->%s", from.Version(), to.Version()), func(t *testing.T) {
				testCompleteBlock(t, from.Version(), to.Version())
			})
		}
	}
}

func testCompleteBlock(t *testing.T, from, to string) {
	db, _ := testConfig(t, backend.EncLZ4_256k, time.Minute, func(c *Config) {
		c.Block.Version = from // temporarily set config to from while we create the wal, so it makes blocks in the "from" format
	})

	wal := db.WAL()
	db.cfg.Block.Version = to // now set it back so we cut blocks in the "to" format

	blockID := uuid.New()

	block, err := wal.NewBlock(blockID, testTenantID, model.CurrentEncoding)
	require.NoError(t, err, "unexpected error creating block")
	require.Equal(t, block.BlockMeta().Version, from)

	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	numMsgs := 100
	reqs := make([]*tempopb.Trace, 0, numMsgs)
	ids := make([][]byte, 0, numMsgs)
	for i := 0; i < numMsgs; i++ {
		id := test.ValidTraceID(nil)
		req := test.MakeTrace(rand.Int()%10, id)
		trace.SortTrace(req)
		writeTraceToWal(t, block, dec, id, req, 0, 0)
		reqs = append(reqs, req)
		ids = append(ids, id)
	}
	require.NoError(t, block.Flush())

	complete, err := db.CompleteBlock(context.Background(), block)
	require.NoError(t, err, "unexpected error completing block")
	require.Equal(t, complete.BlockMeta().Version, to)

	for i, id := range ids {
		found, err := complete.FindTraceByID(context.TODO(), id, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.NotNil(t, found)
		trace.SortTrace(found)
		require.True(t, proto.Equal(found, reqs[i]))
	}
}

func TestCompleteBlockHonorsStartStopTimes(t *testing.T) {
	for _, enc := range encoding.AllEncodings() {
		version := enc.Version()
		t.Run(version, func(t *testing.T) {
			testCompleteBlockHonorsStartStopTimes(t, version)
		})
	}
}

func testCompleteBlockHonorsStartStopTimes(t *testing.T, targetBlockVersion string) {
	tempDir := t.TempDir()

	db, err := New(&Config{
		Backend: backend.Local,
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &common.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              .01,
			BloomShardSizeBytes:  100_000,
			Version:              targetBlockVersion,
			Encoding:             backend.EncNone,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			IngestionSlack: time.Minute,
			Filepath:       path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	require.NoError(t, err)

	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	wal := db.WAL()

	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour).Unix()
	oneHour := now.Add(time.Hour).Unix()

	block, err := wal.NewBlock(uuid.New(), testTenantID, model.CurrentEncoding)
	require.NoError(t, err, "unexpected error creating block")

	// Write a trace from 1 hour ago.
	// The wal slack time will adjust it to 1 minute ago
	id := test.ValidTraceID(nil)
	req := test.MakeTrace(10, id)
	writeTraceToWal(t, block, dec, id, req, uint32(oneHourAgo), uint32(oneHour))

	complete, err := db.CompleteBlock(context.Background(), block)
	require.NoError(t, err, "unexpected error completing block")

	// Verify the block time was constrained to the slack time.
	require.Equal(t, now.Unix(), complete.BlockMeta().StartTime.Unix())
	require.Equal(t, now.Unix(), complete.BlockMeta().EndTime.Unix())
}

func TestShouldCache(t *testing.T) {
	tempDir := t.TempDir()

	db, err := New(&Config{
		Backend: backend.Local,
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &common.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              .01,
			BloomShardSizeBytes:  100_000,
			Version:              encoding.DefaultEncoding().Version(),
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
			assert.Equal(
				t,
				tt.cache,
				db.shouldCache(&backend.BlockMeta{CompactionLevel: tt.compactionLevel, StartTime: tt.startTime}, time.Now()),
			)
		})
	}
}

func writeTraceToWal(
	t require.TestingT,
	b common.WALBlock,
	dec model.SegmentDecoder,
	id common.ID,
	tr *tempopb.Trace,
	start, end uint32,
) {
	b1, err := dec.PrepareForWrite(tr, 0, 0)
	require.NoError(t, err)

	b2, err := dec.ToObject([][]byte{b1})
	require.NoError(t, err)

	err = b.Append(id, b2, start, end)
	require.NoError(t, err, "unexpected error writing req")
}

func BenchmarkCompleteBlock(b *testing.B) {
	for _, enc := range encoding.AllEncodings() {
		b.Run(enc.Version(), func(b *testing.B) {
			benchmarkCompleteBlock(b, enc)
		})
	}
}

func benchmarkCompleteBlock(b *testing.B, e encoding.VersionedEncoding) {
	// Create a WAL block with traces
	traceCount := 10_000
	flushCount := 1000

	tempDir := b.TempDir()
	db, err := New(&Config{
		Backend: backend.Local,
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &common.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              .01,
			BloomShardSizeBytes:  100_000,
			Encoding:             backend.EncNone,
			IndexPageSizeBytes:   1000,
			Version:              e.Version(),
			RowGroupSizeBytes:    30_000_000,
		},
		WAL: &wal.Config{
			IngestionSlack: time.Minute,
			Filepath:       path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	require.NoError(b, err)

	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	wal := db.WAL()
	blk, err := wal.NewBlock(uuid.New(), testTenantID, model.CurrentEncoding)
	require.NoError(b, err)

	for i := 0; i < traceCount; i++ {
		id := test.ValidTraceID(nil)
		req := test.MakeTrace(10, id)
		writeTraceToWal(b, blk, dec, id, req, 0, 0)

		if i%flushCount == 0 {
			require.NoError(b, blk.Flush())
		}
	}

	fmt.Println("Created wal block")

	b.ResetTimer()

	// Complete it
	for i := 0; i < b.N; i++ {
		_, err := db.CompleteBlock(context.Background(), blk)
		require.NoError(b, err)
	}
}
