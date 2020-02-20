package friggdb

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/go-kit/kit/log"

	"github.com/google/uuid"
	"github.com/grafana/frigg/friggdb/backend/local"
	"github.com/grafana/frigg/friggdb/encoding"
	"github.com/grafana/frigg/friggdb/wal"
	"github.com/stretchr/testify/assert"
)

func TestCompaction(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	r, w, err := New(&Config{
		Backend: "local",
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		WAL: &wal.Config{
			Filepath:        path.Join(tempDir, "wal"),
			IndexDownsample: 17,
			BloomFP:         .01,
		},
		MaintenanceCycle:        30 * time.Minute,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, log.NewNopLogger())
	assert.NoError(t, err)

	wal := w.WAL()
	assert.NoError(t, err)

	blockCount := 10
	for i := 0; i < blockCount; i++ {
		blockID := uuid.New()
		head, err := wal.NewBlock(blockID, testTenantID)
		assert.NoError(t, err)

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
	var cursor int
	for {
		var blocks []*encoding.BlockMeta
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
}
