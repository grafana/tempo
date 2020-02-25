package friggdb

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
	"github.com/grafana/frigg/friggdb/backend"
	"github.com/grafana/frigg/friggdb/backend/local"
	"github.com/grafana/frigg/friggdb/pool"
	"github.com/grafana/frigg/friggdb/wal"
	"github.com/grafana/frigg/pkg/friggpb"
	"github.com/grafana/frigg/pkg/util/test"
	"github.com/stretchr/testify/assert"
)

func TestCompaction(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	r, w, err := New(&Config{
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
			IndexDownsample: 17,
			BloomFP:         .01,
		},
		Compactor: &compactorConfig{
			ChunkSizeBytes:     10,
			MaxCompactionRange: time.Hour,
		},
		MaintenanceCycle:        0,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, log.NewNopLogger())
	assert.NoError(t, err)

	wal := w.WAL()
	assert.NoError(t, err)

	blockCount := 10
	recordCount := 10

	allReqs := make([]*friggpb.PushRequest, 0, blockCount*recordCount)
	allIds := make([][]byte, 0, blockCount*recordCount)

	for i := 0; i < blockCount; i++ {
		blockID := uuid.New()
		head, err := wal.NewBlock(blockID, testTenantID)
		assert.NoError(t, err)

		reqs := make([]*friggpb.PushRequest, 0, recordCount)
		ids := make([][]byte, 0, recordCount)
		for j := 0; j < recordCount; j++ {
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
	for {
		cursor := 0
		var blocks []*backend.BlockMeta
		blocks, cursor = rw.blocksToCompact(testTenantID, cursor)
		if cursor == cursorDone {
			break
		}
		if blocks == nil {
			continue
		}
		assert.Len(t, blocks, inputBlocks)

		err := rw.compact(blocks, testTenantID)
		assert.NoError(t, err)

		expectedBlockCount -= blocksPerCompaction
		expectedCompactedCount += inputBlocks
		checkBlocklists(t, uuid.Nil, expectedBlockCount, expectedCompactedCount, rw)
	}

	// now see if we can find our ids
	for i, id := range allIds {
		b, _, err := rw.Find(testTenantID, id)
		assert.NoError(t, err)

		out := &friggpb.PushRequest{}
		err = proto.Unmarshal(b, out)
		assert.NoError(t, err)

		assert.True(t, proto.Equal(allReqs[i], out))
	}
}
