package friggdb

import (
	"context"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/frigg/friggdb/backend/local"
	"github.com/grafana/frigg/pkg/friggpb"
	"github.com/grafana/frigg/pkg/util/test"
	"github.com/stretchr/testify/assert"
)

func TestDB(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	r, w, err := New(&Config{
		Backend: "local",
		Local: local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		WALFilepath:              path.Join(tempDir, "wal"),
		IndexDownsample:          17,
		BloomFilterFalsePositive: .01,
	})
	assert.NoError(t, err)

	blockID := uuid.New()
	tenantID := "fake"

	wal, err := w.WAL()
	assert.NoError(t, err)

	head, err := wal.NewBlock(blockID, tenantID)
	assert.NoError(t, err)

	numMsgs := 10
	reqs := make([]*friggpb.PushRequest, 0, numMsgs)
	ids := make([][]byte, 0, numMsgs)
	for i := 0; i < numMsgs; i++ {
		id := make([]byte, 16)
		rand.Read(id)
		req := test.MakeRequest(rand.Int()%1000, id)
		reqs = append(reqs, req)
		ids = append(ids, id)
		err := head.Write(id, req)
		assert.NoError(t, err, "unexpected error writing req")
	}

	complete, err := head.Complete(wal)
	assert.NoError(t, err)

	err = w.WriteBlock(context.Background(), complete)
	assert.NoError(t, err)

	for _, id := range ids {
		out := &friggpb.Trace{}
		_, found, err := r.Find(tenantID, id, out)
		assert.True(t, found)
		assert.NoError(t, err)

		//assert.True(t, proto.Equal(out, reqs[i]))
	}
}
