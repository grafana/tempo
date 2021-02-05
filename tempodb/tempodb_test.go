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
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/stretchr/testify/assert"
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
			IndexDownsample: 17,
			BloomFP:         .01,
			Encoding:        backend.EncGZIP,
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
	}, &mockSharder{})

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

	complete, err := w.CompleteBlock(head, &mockSharder{})
	assert.NoError(t, err)

	err = w.WriteBlock(context.Background(), complete)
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
			IndexDownsample: 17,
			BloomFP:         .01,
			Encoding:        backend.EncLZ4_256k,
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
	complete, err := w.CompleteBlock(head, &mockSharder{})
	assert.NoError(t, err)

	err = w.WriteBlock(context.Background(), complete)
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
			IndexDownsample: 17,
			BloomFP:         .01,
			Encoding:        backend.EncLZ4_256k,
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
			IndexDownsample: 17,
			BloomFP:         .01,
			Encoding:        backend.EncLZ4_256k,
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
	}, &mockSharder{})

	blockID := uuid.New()

	wal := w.WAL()
	assert.NoError(t, err)

	head, err := wal.NewBlock(blockID, testTenantID)
	assert.NoError(t, err)

	complete, err := w.CompleteBlock(head, &mockSharder{})
	assert.NoError(t, err)

	err = w.WriteBlock(context.Background(), complete)
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
					IndexDownsample: 17,
					BloomFP:         .01,
					Encoding:        backend.EncLZ4_256k,
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
			IndexDownsample: 17,
			BloomFP:         .01,
			Encoding:        backend.EncLZ4_256k,
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
			IndexDownsample: 17,
			BloomFP:         .01,
			Encoding:        backend.EncLZ4_256k,
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
