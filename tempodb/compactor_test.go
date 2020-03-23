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
		/*GCS: &gcs.Config{
			BucketName:      "temp-jelliott",
			ChunkBufferSize: 10 * 1024 * 1024,
		},*/
		WAL: &wal.Config{
			Filepath:        path.Join(tempDir, "wal"),
			IndexDownsample: 11,
			BloomFP:         .01,
		},
		MaintenanceCycle: 0,
	}, log.NewNopLogger())

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Second,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	})

	wal := w.WAL()
	assert.NoError(t, err)

	blockCount := 4
	recordCount := 100

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

			req := test.MakeRequest(i*10, id)
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

	for l := 0; l < maxNumLevels-1; l++ {
		rw.pollBlocklist()

		blocklist := rw.blocklist(testTenantID)
		blocksPerLevel := blocklistPerLevel(blocklist)
		blockSelector := newTimeWindowBlockSelector(blocksPerLevel[l], rw.compactorCfg.MaxCompactionRange)

		expectedCompactions := len(blocksPerLevel[l]) / inputBlocks
		compactions := 0
		for {
			blocks := blockSelector.BlocksToCompact()
			if len(blocks) == 0 {
				break
			}
			assert.Len(t, blocks, inputBlocks)

			compactions++
			err := rw.compact(blocks, testTenantID)
			assert.NoError(t, err)

			expectedBlockCount -= blocksPerCompaction
			expectedCompactedCount += inputBlocks
			checkBlocklists(t, uuid.Nil, expectedBlockCount, expectedCompactedCount, rw)
		}

		assert.Equal(t, expectedCompactions, compactions)
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
		MaxCompactionRange:      24 * time.Hour,
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
	blocklist := rw.blocklist(testTenantID)
	blockSelector := newTimeWindowBlockSelector(blocklist, rw.compactorCfg.MaxCompactionRange)
	blocks = blockSelector.BlocksToCompact()
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
