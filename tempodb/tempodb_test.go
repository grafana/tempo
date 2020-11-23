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
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/stretchr/testify/assert"
)

const (
	testTenantID = "fake"
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
		WAL: &wal.Config{
			Filepath:        path.Join(tempDir, "wal"),
			IndexDownsample: 17,
			BloomFP:         .01,
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

	complete, err := head.Complete(wal, &mockSharder{})
	assert.NoError(t, err)

	err = w.WriteBlock(context.Background(), complete)
	assert.NoError(t, err)

	// poll
	r.(*readerWriter).pollBlocklist()

	// read
	for i, id := range ids {
		bFound, _, err := r.Find(context.Background(), testTenantID, id)
		assert.NoError(t, err)

		out := &tempopb.PushRequest{}
		err = proto.Unmarshal(bFound, out)
		assert.NoError(t, err)

		assert.True(t, proto.Equal(out, reqs[i]))
	}
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
		WAL: &wal.Config{
			Filepath:        path.Join(tempDir, "wal"),
			IndexDownsample: 17,
			BloomFP:         .01,
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	assert.NoError(t, err)

	buff, _, err := r.Find(context.Background(), "unknown", []byte{0x01}, nil, nil)
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
		WAL: &wal.Config{
			Filepath:        path.Join(tempDir, "wal"),
			IndexDownsample: 17,
			BloomFP:         .01,
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

	complete, err := head.Complete(wal, &mockSharder{})
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
		blocklist map[string][]*encoding.BlockMeta
		expected  map[string][]*encoding.BlockMeta
	}{
		{
			name:      "one missing tenant",
			tenants:   []string{"foo"},
			blocklist: map[string][]*encoding.BlockMeta{"foo": {{}}, "bar": {{}}},
			expected:  map[string][]*encoding.BlockMeta{"foo": {{}}},
		},
		{
			name:      "no missing tenants",
			tenants:   []string{"foo", "bar"},
			blocklist: map[string][]*encoding.BlockMeta{"foo": {{}}, "bar": {{}}},
			expected:  map[string][]*encoding.BlockMeta{"foo": {{}}, "bar": {{}}},
		},
		{
			name:      "all missing tenants",
			tenants:   []string{},
			blocklist: map[string][]*encoding.BlockMeta{"foo": {{}}, "bar": {{}}},
			expected:  map[string][]*encoding.BlockMeta{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _, _, err := New(&Config{
				Backend: "local",
				Local: &local.Config{
					Path: path.Join("/tmp", "traces"),
				},
				WAL: &wal.Config{
					Filepath:        path.Join("/tmp", "wal"),
					IndexDownsample: 17,
					BloomFP:         .01,
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
		WAL: &wal.Config{
			Filepath:        path.Join(tempDir, "wal"),
			IndexDownsample: 17,
			BloomFP:         .01,
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	assert.NoError(t, err)

	rw := r.(*readerWriter)

	tests := []struct {
		name     string
		existing []*encoding.BlockMeta
		add      []*encoding.BlockMeta
		remove   []*encoding.BlockMeta
		expected []*encoding.BlockMeta
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
			add: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
			remove: nil,
			expected: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
		},
		{
			name: "add to existing",
			existing: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
			add: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
			remove: nil,
			expected: []*encoding.BlockMeta{
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
			remove: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
			expected: nil,
		},
		{
			name: "remove nil",
			existing: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
			add:    nil,
			remove: nil,
			expected: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
		},
		{
			name: "remove existing",
			existing: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
			add: nil,
			remove: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
			expected: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
		},
		{
			name: "remove no match",
			existing: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
			add: nil,
			remove: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
			expected: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
		},
		{
			name: "add and remove",
			existing: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
			add: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				},
			},
			remove: []*encoding.BlockMeta{
				{
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},
			expected: []*encoding.BlockMeta{
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
		WAL: &wal.Config{
			Filepath:        path.Join(tempDir, "wal"),
			IndexDownsample: 17,
			BloomFP:         .01,
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	assert.NoError(t, err)

	rw := r.(*readerWriter)

	tests := []struct {
		name     string
		existing []*encoding.CompactedBlockMeta
		add      []*encoding.CompactedBlockMeta
		expected []*encoding.CompactedBlockMeta
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
			add: []*encoding.CompactedBlockMeta{
				{
					BlockMeta: encoding.BlockMeta{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					},
				},
			},
			expected: []*encoding.CompactedBlockMeta{
				{
					BlockMeta: encoding.BlockMeta{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					},
				},
			},
		},
		{
			name: "add to existing",
			existing: []*encoding.CompactedBlockMeta{
				{
					BlockMeta: encoding.BlockMeta{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					},
				},
			},
			add: []*encoding.CompactedBlockMeta{
				{
					BlockMeta: encoding.BlockMeta{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					},
				},
			},
			expected: []*encoding.CompactedBlockMeta{
				{
					BlockMeta: encoding.BlockMeta{
						BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					},
				},
				{
					BlockMeta: encoding.BlockMeta{
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
