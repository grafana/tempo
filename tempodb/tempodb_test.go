package tempodb

import (
	"context"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"

	"github.com/go-kit/kit/log"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testTenantID  = "fake"
	testTenantID2 = "fake2"
)

func TestDB(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	r, w, c, err := New(&Config{
		Backend: "local",
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &encoding.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              .01,
			Encoding:             backend.EncGZIP,
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
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})

	blockID := uuid.New()

	wal := w.WAL()

	head, err := wal.NewBlock(blockID, testTenantID)
	assert.NoError(t, err)

	// write
	numMsgs := 10
	reqs := make([]*tempopb.PushRequest, 0, numMsgs)
	ids := make([][]byte, 0, numMsgs)
	for i := 0; i < numMsgs; i++ {
		id := make([]byte, 16)
		rand.Read(id)
		req := test.MakeRequest(rand.Int()%1000, id)
		reqs = append(reqs, req)
		ids = append(ids, id)

		bReq, err := proto.Marshal(req)
		assert.NoError(t, err)
		err = head.Write(id, bReq)
		assert.NoError(t, err, "unexpected error writing req")
	}

	_, err = w.CompleteBlock(head, &mockSharder{})
	assert.NoError(t, err)

	// poll
	r.(*readerWriter).pollBlocklist()

	// read
	for i, id := range ids {
		bFound, err := r.Find(context.Background(), testTenantID, id, BlockIDMin, BlockIDMax)
		assert.NoError(t, err)

		out := &tempopb.PushRequest{}
		err = proto.Unmarshal(bFound[0], out)
		assert.NoError(t, err)

		assert.True(t, proto.Equal(out, reqs[i]))
	}
}

func TestBlockSharding(t *testing.T) {
	// push a req with some traceID
	// cut headblock & write to backend
	// search with different shards and check if its respecting the params

	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	r, w, _, err := New(&Config{
		Backend: "local",
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &encoding.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              .01,
			Encoding:             backend.EncLZ4_256k,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	assert.NoError(t, err)

	// create block with known ID
	blockID := uuid.New()
	wal := w.WAL()

	head, err := wal.NewBlock(blockID, testTenantID)
	assert.NoError(t, err)

	// add a trace to the block
	id := make([]byte, 16)
	rand.Read(id)
	req := test.MakeRequest(rand.Int()%1000, id)

	bReq, err := proto.Marshal(req)
	assert.NoError(t, err)
	err = head.Write(id, bReq)
	assert.NoError(t, err, "unexpected error writing req")

	// write block to backend
	_, err = w.CompleteBlock(head, &mockSharder{})
	assert.NoError(t, err)

	// poll
	r.(*readerWriter).pollBlocklist()

	// get blockID
	r.(*readerWriter).blockListsMtx.Lock()
	blocks := r.(*readerWriter).blockLists[testTenantID]
	r.(*readerWriter).blockListsMtx.Unlock()
	assert.Len(t, blocks, 1)

	// check if it respects the blockstart/blockend params - case1: hit
	blockStart := uuid.MustParse(BlockIDMin).String()
	blockEnd := uuid.MustParse(BlockIDMax).String()
	bFound, err := r.Find(context.Background(), testTenantID, id, blockStart, blockEnd)
	assert.NoError(t, err)
	assert.Greater(t, len(bFound), 0)

	out := &tempopb.PushRequest{}
	err = proto.Unmarshal(bFound[0], out)
	assert.NoError(t, err)
	assert.True(t, proto.Equal(out, req))

	// check if it respects the blockstart/blockend params - case2: miss
	blockStart = uuid.MustParse(BlockIDMin).String()
	blockEnd = uuid.MustParse(BlockIDMin).String()
	bFound, err = r.Find(context.Background(), testTenantID, id, blockStart, blockEnd)
	assert.NoError(t, err)
	assert.Len(t, bFound, 0)
}

func TestNilOnUnknownTenantID(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	r, _, _, err := New(&Config{
		Backend: "local",
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &encoding.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              .01,
			Encoding:             backend.EncLZ4_256k,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	assert.NoError(t, err)

	buff, err := r.Find(context.Background(), "unknown", []byte{0x01}, BlockIDMin, BlockIDMax)
	assert.Nil(t, buff)
	assert.Nil(t, err)
}

func TestBlockCleanup(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	r, w, c, err := New(&Config{
		Backend: "local",
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &encoding.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              .01,
			Encoding:             backend.EncLZ4_256k,
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
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})

	blockID := uuid.New()

	wal := w.WAL()
	assert.NoError(t, err)

	head, err := wal.NewBlock(blockID, testTenantID)
	assert.NoError(t, err)

	_, err = w.CompleteBlock(head, &mockSharder{})
	assert.NoError(t, err)

	rw := r.(*readerWriter)

	// poll
	rw.pollBlocklist()

	assert.Len(t, rw.blockLists[testTenantID], 1)

	os.RemoveAll(tempDir + "/traces/" + testTenantID)

	// poll
	rw.pollBlocklist()

	_, ok := rw.blockLists[testTenantID]
	assert.False(t, ok)
}

func TestCleanMissingTenants(t *testing.T) {
	tests := []struct {
		name      string
		tenants   []string
		blocklist map[string][]*backend.BlockMeta
		expected  map[string][]*backend.BlockMeta
	}{
		{
			name:      "one missing tenant",
			tenants:   []string{"foo"},
			blocklist: map[string][]*backend.BlockMeta{"foo": {{}}, "bar": {{}}},
			expected:  map[string][]*backend.BlockMeta{"foo": {{}}},
		},
		{
			name:      "no missing tenants",
			tenants:   []string{"foo", "bar"},
			blocklist: map[string][]*backend.BlockMeta{"foo": {{}}, "bar": {{}}},
			expected:  map[string][]*backend.BlockMeta{"foo": {{}}, "bar": {{}}},
		},
		{
			name:      "all missing tenants",
			tenants:   []string{},
			blocklist: map[string][]*backend.BlockMeta{"foo": {{}}, "bar": {{}}},
			expected:  map[string][]*backend.BlockMeta{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _, _, err := New(&Config{
				Backend: "local",
				Local: &local.Config{
					Path: path.Join("/tmp", "traces"),
				},
				Block: &encoding.BlockConfig{
					IndexDownsampleBytes: 17,
					BloomFP:              .01,
					Encoding:             backend.EncLZ4_256k,
					IndexPageSizeBytes:   1000,
				},
				WAL: &wal.Config{
					Filepath: path.Join("/tmp", "wal"),
				},
				BlocklistPoll: 0,
			}, log.NewNopLogger())
			assert.NoError(t, err)

			rw := r.(*readerWriter)

			rw.blockLists = tt.blocklist
			rw.cleanMissingTenants(tt.tenants)
			assert.Equal(t, rw.blockLists, tt.expected)
		})
	}
}

func checkBlocklists(t *testing.T, expectedID uuid.UUID, expectedB int, expectedCB int, rw *readerWriter) {
	rw.pollBlocklist()

	blocklist := rw.blockLists[testTenantID]
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

	compactedBlocklist := rw.compactedBlockLists[testTenantID]
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

func TestUpdateBlocklist(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	r, _, _, err := New(&Config{
		Backend: "local",
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &encoding.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              .01,
			Encoding:             backend.EncLZ4_256k,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	assert.NoError(t, err)

	rw := r.(*readerWriter)

	tests := []struct {
		name     string
		existing []*backend.BlockMeta
		add      []*backend.BlockMeta
		remove   []*backend.BlockMeta
		expected []*backend.BlockMeta
	}{
		{
			name:     "all nil",
			existing: nil,
			add:      nil,
			remove:   nil,
			expected: nil,
		},
		{
			name:     "add to nil",
			existing: nil,
			add: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
			remove: nil,
			expected: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
		},
		{
			name: "add to existing",
			existing: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
			add: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
			remove: nil,
			expected: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
		},
		{
			name:     "remove from nil",
			existing: nil,
			add:      nil,
			remove: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
			expected: nil,
		},
		{
			name: "remove nil",
			existing: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
			add:    nil,
			remove: nil,
			expected: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
		},
		{
			name: "remove existing",
			existing: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
			add: nil,
			remove: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
		},
		{
			name: "remove no match",
			existing: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
			add: nil,
			remove: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
		},
		{
			name: "add and remove",
			existing: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
			add: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				},
			},
			remove: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
			expected: []*backend.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rw.blockLists[testTenantID] = tt.existing
			rw.updateBlocklist(testTenantID, tt.add, tt.remove, nil)

			assert.Equal(t, len(tt.expected), len(rw.blockLists[testTenantID]))

			for i := range tt.expected {
				assert.Equal(t, tt.expected[i].BlockID, rw.blockLists[testTenantID][i].BlockID)
			}
		})
	}
}

func TestUpdateBlocklistCompacted(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	r, _, _, err := New(&Config{
		Backend: "local",
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &encoding.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              .01,
			Encoding:             backend.EncLZ4_256k,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	assert.NoError(t, err)

	rw := r.(*readerWriter)

	tests := []struct {
		name     string
		existing []*backend.CompactedBlockMeta
		add      []*backend.CompactedBlockMeta
		expected []*backend.CompactedBlockMeta
	}{
		{
			name:     "all nil",
			existing: nil,
			add:      nil,
			expected: nil,
		},
		{
			name:     "add to nil",
			existing: nil,
			add: []*backend.CompactedBlockMeta{
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					},
				},
			},
			expected: []*backend.CompactedBlockMeta{
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					},
				},
			},
		},
		{
			name: "add to existing",
			existing: []*backend.CompactedBlockMeta{
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					},
				},
			},
			add: []*backend.CompactedBlockMeta{
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					},
				},
			},
			expected: []*backend.CompactedBlockMeta{
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					},
				},
				{
					BlockMeta: backend.BlockMeta{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rw.compactedBlockLists[testTenantID] = tt.existing
			rw.updateBlocklist(testTenantID, nil, nil, tt.add)

			assert.Equal(t, len(tt.expected), len(rw.compactedBlockLists[testTenantID]))

			for i := range tt.expected {
				assert.Equal(t, tt.expected[i].BlockID, rw.compactedBlockLists[testTenantID][i].BlockID)
			}
		})
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
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	r, w, c, err := New(&Config{
		Backend: "local",
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &encoding.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              .01,
			Encoding:             backend.EncLZ4_256k,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: time.Minute,
	}, log.NewNopLogger())
	assert.NoError(t, err)

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})

	wal := w.WAL()
	assert.NoError(t, err)

	head, err := wal.NewBlock(uuid.New(), testTenantID)
	assert.NoError(t, err)

	// write
	numMsgs := 10
	reqs := make([]*tempopb.PushRequest, 0, numMsgs)
	ids := make([][]byte, 0, numMsgs)
	for i := 0; i < numMsgs; i++ {
		id := make([]byte, 16)
		rand.Read(id)
		req := test.MakeRequest(rand.Int()%1000, id)
		reqs = append(reqs, req)
		ids = append(ids, id)

		bReq, err := proto.Marshal(req)
		assert.NoError(t, err)
		err = head.Write(id, bReq)
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
		bFound, err := r.Find(context.Background(), testTenantID, id, blockID, blockID)
		assert.NoError(t, err)

		out := &tempopb.PushRequest{}
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
	compactedBlocks, ok := rw.compactedBlockLists[testTenantID]
	require.True(t, ok)
	require.Len(t, compactedBlocks, 1)
	assert.Equal(t, compactedBlocks[0].BlockID.String(), blockID)
	blocks, ok := rw.blockLists[testTenantID]
	require.True(t, ok)
	require.Len(t, blocks, 1)
	assert.NotEqual(t, blocks[0].BlockID.String(), blockID)

	// find should succeed with old block range
	for i, id := range ids {
		bFound, err := r.Find(context.Background(), testTenantID, id, blockID, blockID)
		assert.NoError(t, err)

		out := &tempopb.PushRequest{}
		err = proto.Unmarshal(bFound[0], out)
		assert.NoError(t, err)

		assert.True(t, proto.Equal(out, reqs[i]))
	}
}

func TestCompleteBlock(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	_, w, _, err := New(&Config{
		Backend: "local",
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &encoding.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              .01,
			Encoding:             backend.EncLZ4_256k,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: time.Minute,
	}, log.NewNopLogger())
	assert.NoError(t, err)

	wal := w.WAL()

	blockID := uuid.New()

	block, err := wal.NewBlock(blockID, testTenantID)
	assert.NoError(t, err, "unexpected error creating block")

	numMsgs := 100
	reqs := make([]*tempopb.PushRequest, 0, numMsgs)
	ids := make([][]byte, 0, numMsgs)
	for i := 0; i < numMsgs; i++ {
		id := make([]byte, 16)
		rand.Read(id)
		req := test.MakeRequest(rand.Int()%1000, id)
		reqs = append(reqs, req)
		ids = append(ids, id)
		bReq, err := proto.Marshal(req)
		assert.NoError(t, err)
		err = block.Write(id, bReq)
		assert.NoError(t, err, "unexpected error writing req")
	}

	complete, err := w.CompleteBlock(block, &mockSharder{})
	require.NoError(t, err, "unexpected error completing block")

	for i, id := range ids {
		out := &tempopb.PushRequest{}
		foundBytes, err := complete.Find(context.TODO(), id)
		assert.NoError(t, err)

		err = proto.Unmarshal(foundBytes, out)
		assert.NoError(t, err)

		assert.True(t, proto.Equal(out, reqs[i]))
	}
}
