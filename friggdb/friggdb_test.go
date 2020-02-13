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
	"github.com/grafana/frigg/friggdb/backend/local"
	"github.com/grafana/frigg/pkg/friggpb"
	"github.com/grafana/frigg/pkg/util/test"
	"github.com/stretchr/testify/assert"
)

const (
	testTenantID = "fake"
)

func TestDB(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	r, w, err := New(&Config{
		Backend: "local",
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		WALFilepath:              path.Join(tempDir, "wal"),
		IndexDownsample:          17,
		BloomFilterFalsePositive: .01,
		BlocklistRefreshRate:     30 * time.Minute,
	}, log.NewNopLogger())
	assert.NoError(t, err)

	blockID := uuid.New()

	wal, err := w.WAL()
	assert.NoError(t, err)

	head, err := wal.NewBlock(blockID, testTenantID)
	assert.NoError(t, err)

	numMsgs := 1
	reqs := make([]*friggpb.PushRequest, 0, numMsgs)
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

	complete, err := head.Complete(wal)
	assert.NoError(t, err)

	err = w.WriteBlock(context.Background(), complete)
	assert.NoError(t, err)

	// force poll the blocklist now that we've written something
	r.(*readerWriter).actuallyPollBlocklist()

	for i, id := range ids {
		bFound, _, err := r.Find(testTenantID, id)
		assert.NoError(t, err)

		out := &friggpb.PushRequest{}
		err = proto.Unmarshal(bFound, out)
		assert.NoError(t, err)

		assert.True(t, proto.Equal(out, reqs[i]))
	}
}
