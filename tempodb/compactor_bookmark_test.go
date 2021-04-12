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
	"github.com/stretchr/testify/require"
)

func TestCurrentClear(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	require.NoError(t, err, "unexpected error creating temp dir")

	_, w, c, err := New(&Config{
		Backend: "local",
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &encoding.BlockConfig{
			IndexDownsampleBytes:  17,
			BloomFilterShardSize:  100_000,
			BloomFilterShardCount: 10,
			Encoding:              backend.EncGZIP,
			IndexPageSizeBytes:    1000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	require.NoError(t, err)

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})

	wal := w.WAL()
	require.NoError(t, err)

	recordCount := 10
	blockID := uuid.New()
	head, err := wal.NewBlock(blockID, testTenantID)
	require.NoError(t, err)

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

	iter, err := complete.Iterator(10)
	assert.NoError(t, err)
	bm := newBookmark(iter)

	i := 0
	for {
		_, _, err = bm.current(context.Background())
		if err == io.EOF {
			break
		}
		assert.NoError(t, err)
		i++
		bm.clear()
	}
	assert.Equal(t, recordCount, i)
}
