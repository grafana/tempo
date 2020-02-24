package friggdb

import (
	"context"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/grafana/frigg/friggdb/backend"
	"github.com/grafana/frigg/friggdb/backend/local"
	"github.com/grafana/frigg/friggdb/wal"
	"github.com/grafana/frigg/pkg/util/test"
	"github.com/stretchr/testify/assert"
)

func TestCurrentClear(t *testing.T) {
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
		Compactor: &compactorConfig{
			ChunkSizeBytes: 10,
		},
		MaintenanceCycle:        0,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, log.NewNopLogger())
	assert.NoError(t, err)

	wal := w.WAL()
	assert.NoError(t, err)

	recordCount := 10
	blockID := uuid.New()
	head, err := wal.NewBlock(blockID, testTenantID)
	assert.NoError(t, err)

	for i := 0; i < recordCount; i++ {
		id := make([]byte, 16)
		rand.Read(id)
		req := test.MakeRequest(rand.Int()%1000, id)
		bReq, err := proto.Marshal(req)
		assert.NoError(t, err)
		err = head.Write(id, bReq)
		assert.NoError(t, err, "unexpected error writing req")
	}

	complete, err := head.Complete(wal)
	assert.NoError(t, err)

	err = w.WriteBlock(context.Background(), complete)
	assert.NoError(t, err)

	blockID = complete.BlockMeta().BlockID
	rw := r.(*readerWriter)

	iter, err := backend.NewLazyIterator(testTenantID, blockID, 10, rw.r)
	assert.NoError(t, err)
	bm := newBookmark(iter)

	i := 0
	for {
		_, _, err = bm.current()
		if err == io.EOF {
			break
		}
		assert.NoError(t, err)
		i++
		bm.clear()
	}
	assert.Equal(t, recordCount, i)
}
