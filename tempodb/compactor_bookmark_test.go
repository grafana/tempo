package tempodb

import (
	"context"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/stretchr/testify/assert"
)

func TestCurrentClear(t *testing.T) {
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

	complete, err := w.CompleteBlock(head, &mockSharder{})
	assert.NoError(t, err)

	err = w.WriteBlock(context.Background(), complete)
	assert.NoError(t, err)

	rw := r.(*readerWriter)
	block, err := encoding.NewBackendBlock(complete.BlockMeta(), rw.r)
	assert.NoError(t, err)
	iter, err := block.Iterator(10)
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
