package tempodb

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"

	"github.com/grafana/tempo/tempodb/encoding/vparquet4"

	"github.com/go-kit/log"
	"github.com/golang/protobuf/proto" //nolint:all
	"github.com/google/uuid"
	v2 "github.com/grafana/tempo/tempodb/encoding/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/cache/memcached"
	"github.com/grafana/tempo/modules/cache/redis"
	"github.com/grafana/tempo/pkg/cache"
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

func testConfig(t *testing.T, enc backend.Encoding, blocklistPoll time.Duration, opts ...testConfigOption) (Reader, Writer, Compactor, string) {
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
		Search: &SearchConfig{
			ChunkSizeBytes:  1_000_000,
			ReadBufferCount: 8, ReadBufferSizeBytes: 4 * 1024 * 1024,
		},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	r, w, c, err := New(cfg, nil, log.NewNopLogger())
	require.NoError(t, err)
	return r, w, c, tempDir
}

func TestDB(t *testing.T) {
	r, w, c, _ := testConfig(t, backend.EncGZIP, 0)

	err := c.EnableCompaction(context.Background(), &CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	r.EnablePolling(context.Background(), &mockJobSharder{})

	blockID := backend.NewUUID()

	wal := w.WAL()

	meta := &backend.BlockMeta{BlockID: blockID, TenantID: testTenantID}
	head, err := wal.NewBlock(meta, model.CurrentEncoding)
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

	_, err = w.CompleteBlock(context.Background(), head)
	assert.NoError(t, err)

	// poll
	r.(*readerWriter).pollBlocklist()

	// read
	for i, id := range ids {
		bFound, failedBlocks, err := r.Find(context.Background(), testTenantID, id, BlockIDMin, BlockIDMax, 0, 0, common.DefaultSearchOptions())
		assert.NoError(t, err)
		assert.Nil(t, failedBlocks)
		assert.True(t, proto.Equal(bFound[0].Trace, reqs[i]))
		require.Greater(t, bFound[0].Metrics.InspectedBytes, uint64(100000))
	}
}

func TestNoCompactionWhenCompactionRange0(t *testing.T) {
	_, _, c, _ := testConfig(t, backend.EncGZIP, 0)

	err := c.EnableCompaction(context.Background(), &CompactorConfig{
		MaxCompactionRange: 0,
	}, &mockSharder{}, &mockOverrides{})
	require.Error(t, err)
}

func TestBlockSharding(t *testing.T) {
	// push a req with some traceID
	// cut headblock & write to backend
	// search with different shards and check if its respecting the params
	r, w, _, _ := testConfig(t, backend.EncLZ4_256k, 0)

	r.EnablePolling(context.Background(), &mockJobSharder{})

	// create block with known ID
	blockID := backend.NewUUID()
	wal := w.WAL()

	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)
	meta := &backend.BlockMeta{BlockID: blockID, TenantID: testTenantID, DataEncoding: model.CurrentEncoding}
	head, err := wal.NewBlock(meta, model.CurrentEncoding)
	assert.NoError(t, err)

	// add a trace to the block
	id := test.ValidTraceID(nil)
	req := test.MakeTrace(1, id)
	writeTraceToWal(t, head, dec, id, req, 0, 0)

	// write block to backend
	_, err = w.CompleteBlock(context.Background(), head)
	assert.NoError(t, err)

	// poll
	r.(*readerWriter).pollBlocklist()

	// get blockID
	blocks := r.(*readerWriter).blocklist.Metas(testTenantID)
	assert.Len(t, blocks, 1)

	// check if it respects the blockstart/blockend params - case1: hit
	blockStart := uuid.MustParse(BlockIDMin).String()
	blockEnd := uuid.MustParse(BlockIDMax).String()
	bFound, failedBlocks, err := r.Find(context.Background(), testTenantID, id, blockStart, blockEnd, 0, 0, common.DefaultSearchOptions())
	assert.NoError(t, err)
	assert.Nil(t, failedBlocks)
	assert.Greater(t, len(bFound), 0)
	assert.True(t, proto.Equal(bFound[0].Trace, req))
	require.Greater(t, bFound[0].Metrics.InspectedBytes, uint64(10000))

	// check if it respects the blockstart/blockend params - case2: miss
	blockStart = uuid.MustParse(BlockIDMin).String()
	blockEnd = uuid.MustParse(BlockIDMin).String()
	bFound, failedBlocks, err = r.Find(context.Background(), testTenantID, id, blockStart, blockEnd, 0, 0, common.DefaultSearchOptions())
	assert.NoError(t, err)
	assert.Nil(t, failedBlocks)
	assert.Len(t, bFound, 0)
}

func TestNilOnUnknownTenantID(t *testing.T) {
	r, _, _, _ := testConfig(t, backend.EncLZ4_256k, 0)

	buff, failedBlocks, err := r.Find(context.Background(), "unknown", []byte{0x01}, BlockIDMin, BlockIDMax, 0, 0, common.DefaultSearchOptions())
	assert.Nil(t, buff)
	assert.Nil(t, err)
	assert.Nil(t, failedBlocks)
}

func TestBlockCleanup(t *testing.T) {
	r, w, c, tempDir := testConfig(t, backend.EncLZ4_256k, 0)

	err := c.EnableCompaction(context.Background(), &CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	r.EnablePolling(context.Background(), &mockJobSharder{})

	blockID := backend.NewUUID()

	wal := w.WAL()

	meta := &backend.BlockMeta{BlockID: blockID, TenantID: testTenantID}
	head, err := wal.NewBlock(meta, model.CurrentEncoding)
	assert.NoError(t, err)

	_, err = w.CompleteBlock(context.Background(), head)
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
	require.Len(t, blocklist, expectedB)
	if expectedB > 0 && expectedID != uuid.Nil {
		require.Equal(t, expectedID, (uuid.UUID)(blocklist[0].BlockID))
	}

	compactedBlocklist := rw.blocklist.CompactedMetas(testTenantID)
	require.Len(t, compactedBlocklist, expectedCB)
	if expectedCB > 0 && expectedID != uuid.Nil {
		require.Equal(t, expectedID, (uuid.UUID)(compactedBlocklist[0].BlockID))
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
				BlockID: backend.MustParse("50000000-0000-0000-0000-000000000000"),
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
				BlockID: backend.MustParse("50000000-0000-0000-0000-000000000000"),
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
				BlockID: backend.MustParse("50000000-0000-0000-0000-000000000000"),
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
				BlockID: backend.MustParse("50000000-0000-0000-0000-000000000000"),
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
				BlockID:   backend.MustParse("50000000-0000-0000-0000-000000000000"),
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
				BlockID:   backend.MustParse("50000000-0000-0000-0000-000000000000"),
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
				BlockID: backend.MustParse("50000000-0000-0000-0000-000000000000"),
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
				BlockID: backend.MustParse("52000000-0000-0000-0000-000000000000"),
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
		// 	},
		// },
		// {
		// 	name:       "exclude - max id range",
		// 	searchID:   []byte{0x11},
		// 	blockStart: uuid.MustParse(BlockIDMin),
		// 	blockEnd:   uuid.MustParse(BlockIDMax),
		// 	meta: &backend.BlockMeta{
		// 		BlockID: uuid.MustParse("50000000-0000-0000-0000-000000000000"),
		// 	},
		// },
		{
			name:       "exclude - min block range",
			searchID:   []byte{0x05},
			blockStart: uuid.MustParse("50000000-0000-0000-0000-000000000000"),
			blockEnd:   uuid.MustParse("51000000-0000-0000-0000-000000000000"),
			meta: &backend.BlockMeta{
				BlockID: backend.MustParse("4FFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF"),
			},
		},
		{
			name:       "exclude - max block range",
			searchID:   []byte{0x05},
			blockStart: uuid.MustParse("50000000-0000-0000-0000-000000000000"),
			blockEnd:   uuid.MustParse("51000000-0000-0000-0000-000000000000"),
			meta: &backend.BlockMeta{
				BlockID: backend.MustParse("51000000-0000-0000-0000-000000000001"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, err := tc.blockStart.MarshalBinary()
			require.NoError(t, err)
			e, err := tc.blockEnd.MarshalBinary()
			require.NoError(t, err)

			assert.Equal(t, tc.expected, includeBlock(tc.meta, tc.searchID, s, e, tc.start, tc.end, 0))
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
					BlockID: backend.MustParse("50000000-0000-0000-0000-000000000000"),
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
					BlockID: backend.MustParse("50000000-0000-0000-0000-000000000000"),
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
					BlockID: backend.MustParse("51000000-0000-0000-0000-000000000000"),
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

			assert.Equal(t, tc.expected, includeCompactedBlock(tc.meta, tc.searchID, s, e, blocklistPoll, tc.start, tc.end, 0))
		})
	}
}

func TestSearchCompactedBlocks(t *testing.T) {
	t.Parallel()
	r, w, c, _ := testConfig(t, backend.EncLZ4_256k, time.Hour)

	err := c.EnableCompaction(context.Background(), &CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	r.EnablePolling(context.Background(), &mockJobSharder{})

	wal := w.WAL()

	meta := &backend.BlockMeta{BlockID: backend.NewUUID(), TenantID: testTenantID}
	head, err := wal.NewBlock(meta, model.CurrentEncoding)
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
	complete, err := w.CompleteBlock(ctx, head)
	require.NoError(t, err)

	blockID := complete.BlockMeta().BlockID.String()

	rw := r.(*readerWriter)

	// poll
	rw.pollBlocklist()

	// read
	for i, id := range ids {
		bFound, failedBlocks, err := r.Find(ctx, testTenantID, id, blockID, blockID, 0, 0, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.Nil(t, failedBlocks)
		require.True(t, proto.Equal(bFound[0].Trace, reqs[i]))
		require.Greater(t, bFound[0].Metrics.InspectedBytes, uint64(100000))
	}

	// compact
	var blockMetas []*backend.BlockMeta
	blockMetas = append(blockMetas, complete.BlockMeta())
	require.NoError(t, rw.Compact(ctx, blockMetas, testTenantID))

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
		bFound, failedBlocks, err := r.Find(ctx, testTenantID, id, blockID, blockID, 0, 0, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.Nil(t, failedBlocks)
		require.True(t, proto.Equal(bFound[0].Trace, reqs[i]))
		require.Greater(t, bFound[0].Metrics.InspectedBytes, uint64(100000))
	}
}

func TestCompleteBlock(t *testing.T) {
	for _, from := range encoding.AllEncodings() {
		for _, to := range encoding.AllEncodings() {
			t.Run(fmt.Sprintf("%s->%s", from.Version(), to.Version()), func(t *testing.T) {
				t.Parallel()
				testCompleteBlock(t, from.Version(), to.Version())
			})
		}
	}
}

func testCompleteBlock(t *testing.T, from, to string) {
	_, w, _, _ := testConfig(t, backend.EncLZ4_256k, time.Minute, func(c *Config) {
		c.Block.Version = from // temporarily set config to from while we create the wal, so it makes blocks in the "from" format
	})

	wal := w.WAL()
	rw := w.(*readerWriter)
	rw.cfg.Block.Version = to // now set it back so we cut blocks in the "to" format

	blockID := uuid.New()

	var dataEncoding string
	if from == v2.VersionString {
		dataEncoding = model.CurrentEncoding
	}
	meta := backend.NewBlockMeta(testTenantID, blockID, from, backend.EncNone, dataEncoding)
	block, err := wal.NewBlock(meta, model.CurrentEncoding)
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

	complete, err := w.CompleteBlock(context.Background(), block)
	require.NoError(t, err, "unexpected error completing block")
	require.Equal(t, complete.BlockMeta().Version, to)

	for i, id := range ids {
		found, err := complete.FindTraceByID(context.TODO(), id, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.NotNil(t, found)
		trace.SortTrace(found.Trace)
		require.True(t, proto.Equal(found.Trace, reqs[i]))
		vparquet4 := vparquet4.Encoding{}.Version()
		if to == vparquet4 {
			require.Greater(t, found.Metrics.InspectedBytes, uint64(100000))
		}
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

	_, w, _, err := New(&Config{
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
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	wal := w.WAL()

	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour).Unix()
	oneHour := now.Add(time.Hour).Unix()

	meta := &backend.BlockMeta{BlockID: backend.NewUUID(), TenantID: testTenantID}
	block, err := wal.NewBlock(meta, model.CurrentEncoding)
	require.NoError(t, err, "unexpected error creating block")

	// Write a trace from 1 hour ago.
	// The wal slack time will adjust it to 1 minute ago
	id := test.ValidTraceID(nil)
	req := test.MakeTrace(10, id)
	writeTraceToWal(t, block, dec, id, req, uint32(oneHourAgo), uint32(oneHour))

	complete, err := w.CompleteBlock(context.Background(), block)
	require.NoError(t, err, "unexpected error completing block")

	// Verify the block time was constrained to the slack time.
	// Accept a couple seconds of slack time to ensure test reliability.
	require.Less(t, complete.BlockMeta().StartTime.Sub(now).Seconds(), 2.0)
	require.Less(t, complete.BlockMeta().EndTime.Sub(now).Seconds(), 2.0)
}

func writeTraceToWal(t require.TestingT, b common.WALBlock, dec model.SegmentDecoder, id common.ID, tr *tempopb.Trace, start, end uint32) {
	b1, err := dec.PrepareForWrite(tr, 0, 0)
	require.NoError(t, err)

	b2, err := dec.ToObject([][]byte{b1})
	require.NoError(t, err)

	err = b.Append(id, b2, start, end, true)
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
	_, w, _, err := New(&Config{
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
	}, nil, log.NewNopLogger())
	require.NoError(b, err)

	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	wal := w.WAL()
	meta := &backend.BlockMeta{BlockID: backend.NewUUID(), TenantID: testTenantID}
	blk, err := wal.NewBlock(meta, model.CurrentEncoding)
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
		_, err := w.CompleteBlock(context.Background(), blk)
		require.NoError(b, err)
	}
}

func TestCreateLegacyCache(t *testing.T) {
	tcs := []struct {
		name          string
		cfg           *Config
		expectedErr   error
		expectedRoles []cache.Role
		expectedCache bool
	}{
		{
			name: "no caches",
			cfg:  &Config{},
		},
		{
			name: "redis",
			cfg: &Config{
				Cache:           "redis",
				Redis:           &redis.Config{},
				BackgroundCache: &cache.BackgroundConfig{},
			},
			expectedRoles: []cache.Role{cache.RoleBloom, cache.RoleTraceIDIdx},
			expectedCache: true,
		},
		{
			name: "memcached",
			cfg: &Config{
				Cache:           "memcached",
				Memcached:       &memcached.Config{},
				BackgroundCache: &cache.BackgroundConfig{},
			},
			expectedRoles: []cache.Role{cache.RoleBloom, cache.RoleTraceIDIdx},
			expectedCache: true,
		},
		{
			name: "no cache but cache control",
			cfg: &Config{
				Search: &SearchConfig{
					CacheControl: CacheControlConfig{
						Footer: true,
					},
				},
			},
			expectedErr: errors.New("no legacy cache configured, but cache_control is enabled. Please use the new top level cache configuration."),
		},
		{
			name: "memcached + cache control",
			cfg: &Config{
				Cache:           "memcached",
				Memcached:       &memcached.Config{},
				BackgroundCache: &cache.BackgroundConfig{},
				Search: &SearchConfig{
					CacheControl: CacheControlConfig{
						Footer:      true,
						ColumnIndex: true,
						OffsetIndex: true,
					},
				},
			},
			expectedRoles: []cache.Role{cache.RoleBloom, cache.RoleTraceIDIdx, cache.RoleParquetColumnIdx, cache.RoleParquetFooter, cache.RoleParquetOffsetIdx},
			expectedCache: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			prometheus.DefaultRegisterer = prometheus.NewRegistry() // prevent duplicate registration

			cache, actualRoles, actualErr := createLegacyCache(tc.cfg, log.NewNopLogger())
			require.Equal(t, tc.expectedErr, actualErr)
			require.Equal(t, tc.expectedRoles, actualRoles)
			if tc.expectedCache {
				require.NotNil(t, cache)
			} else {
				require.Nil(t, cache)
			}
		})
	}
}
