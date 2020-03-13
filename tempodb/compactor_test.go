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
	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/pool"
	"github.com/grafana/tempo/tempodb/wal"
)

func TestCompaction(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
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
		WAL: &wal.Config{
			Filepath:        path.Join(tempDir, "wal"),
			IndexDownsample: rand.Int()%20 + 1,
			BloomFP:         .01,
		},
		MaintenanceCycle: 0,
	}, log.NewNopLogger())
	assert.NoError(t, err)

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	})

	wal := w.WAL()
	assert.NoError(t, err)

	blockCount := rand.Int()%20 + 1
	recordCount := rand.Int()%20 + 1

	allReqs := make([]*tempopb.PushRequest, 0, blockCount*recordCount)
	allIds := make([][]byte, 0, blockCount*recordCount)

	for i := 0; i < blockCount; i++ {
		blockID := uuid.New()
		head, err := wal.NewBlock(blockID, testTenantID)
		assert.NoError(t, err)

		reqs := make([]*tempopb.PushRequest, 0, recordCount)
		ids := make([][]byte, 0, recordCount)
		for j := 0; j < recordCount; j++ {
			id := make([]byte, 16)
			_, err = rand.Read(id)
			assert.NoError(t, err, "unexpected creating random id")

			req := test.MakeRequest(rand.Int()%1000, id)
			reqs = append(reqs, req)
			ids = append(ids, id)

			bReq, err := proto.Marshal(req)
			assert.NoError(t, err)
			err = head.Write(id, bReq)
			assert.NoError(t, err, "unexpected error writing req")
		}
		allReqs = append(allReqs, reqs...)
		allIds = append(allIds, ids...)

		complete, err := head.Complete(wal)
		assert.NoError(t, err)

		err = w.WriteBlock(context.Background(), complete)
		assert.NoError(t, err)
	}

	rw := r.(*readerWriter)

	// poll
	expectedBlockCount := blockCount
	expectedCompactedCount := 0
	checkBlocklists(t, uuid.Nil, expectedBlockCount, expectedCompactedCount, rw)

	blocksPerCompaction := (inputBlocks - outputBlocks)

	rw.blockSelector.ResetCursor()
	for {
		var blocks []*backend.BlockMeta
		blocklist := rw.blocklist(testTenantID)

		blocksPerLevel := make([][]*backend.BlockMeta, maxNumLevels)
		for k := 0; k < maxNumLevels; k++ {
			blocksPerLevel[k] = make([]*backend.BlockMeta, 0)
		}

		for _, block := range blocklist {
			blocksPerLevel[block.CompactionLevel] = append(blocksPerLevel[block.CompactionLevel], block)
		}

		pos := rw.blockSelector.BlocksToCompactInSameLevel(blocksPerLevel[0])
		if pos == -1 {
			break
		}
		blocks = blocksPerLevel[0][pos : pos+inputBlocks]
		assert.Len(t, blocks, inputBlocks)

		err := rw.compact(blocks, testTenantID)
		assert.NoError(t, err)

		expectedBlockCount -= blocksPerCompaction
		expectedCompactedCount += inputBlocks
		checkBlocklists(t, uuid.Nil, expectedBlockCount, expectedCompactedCount, rw)
	}

	// do we have the right number of records
	var records int
	for _, meta := range rw.blockLists[testTenantID] {
		records += meta.TotalObjects
	}
	assert.Equal(t, blockCount*recordCount, records)

	// now see if we can find our ids
	for i, id := range allIds {
		b, _, err := rw.Find(testTenantID, id)
		assert.NoError(t, err)

		out := &tempopb.PushRequest{}
		err = proto.Unmarshal(b, out)
		assert.NoError(t, err)

		assert.True(t, proto.Equal(allReqs[i], out))
	}
}

func TestSameIDCompaction(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
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
		WAL: &wal.Config{
			Filepath:        path.Join(tempDir, "wal"),
			IndexDownsample: rand.Int()%20 + 1,
			BloomFP:         .01,
		},
		MaintenanceCycle: 0,
	}, log.NewNopLogger())
	assert.NoError(t, err)

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	})

	wal := w.WAL()
	assert.NoError(t, err)

	blockCount := 5

	for i := 0; i < blockCount; i++ {
		blockID := uuid.New()
		head, err := wal.NewBlock(blockID, testTenantID)
		assert.NoError(t, err)
		id := []byte{0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02}
		rec := []byte{0x01, 0x02, 0x03}

		err = head.Write(id, rec)
		assert.NoError(t, err, "unexpected error writing req")

		complete, err := head.Complete(wal)
		assert.NoError(t, err)

		err = w.WriteBlock(context.Background(), complete)
		assert.NoError(t, err)
	}

	rw := r.(*readerWriter)

	// poll
	checkBlocklists(t, uuid.Nil, 5, 0, rw)

	var blocks []*backend.BlockMeta
	rw.blockSelector.ResetCursor()
	blocklist := rw.blocklist(testTenantID)
	pos := rw.blockSelector.BlocksToCompactInSameLevel(blocklist)
	assert.NotEqual(t, -1, pos)
	blocks = blocklist[pos : pos+inputBlocks]
	assert.Len(t, blocks, inputBlocks)

	err = rw.compact(blocks, testTenantID)
	assert.NoError(t, err)

	checkBlocklists(t, uuid.Nil, 2, 4, rw)

	// do we have the right number of records
	var records int
	for _, meta := range rw.blockLists[testTenantID] {
		records += meta.TotalObjects
	}
	assert.Equal(t, 2, records)
}
