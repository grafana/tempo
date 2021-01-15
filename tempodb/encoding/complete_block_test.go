package encoding

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/index"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
	"github.com/stretchr/testify/assert"
)

type mockCombiner struct {
}

func (m *mockCombiner) Combine(objA []byte, objB []byte) []byte {
	if len(objA) > len(objB) {
		return objA
	}

	return objB
}

func TestCompleteBlock(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	rand.Seed(time.Now().Unix())

	indexDownsample := 13
	buffer := &bytes.Buffer{}
	writer := bufio.NewWriter(buffer)
	assert.NoError(t, err, "unexpected error creating block")
	appender := v0.NewAppender(writer)

	numMsgs := 1000
	reqs := make([]*tempopb.PushRequest, 0, numMsgs)
	ids := make([][]byte, 0, numMsgs)
	var maxID, minID []byte
	for i := 0; i < numMsgs; i++ {
		id := make([]byte, 16)
		rand.Read(id)
		req := test.MakeRequest(rand.Int()%10, id)
		reqs = append(reqs, req)
		ids = append(ids, id)
		bReq, err := proto.Marshal(req)
		assert.NoError(t, err)

		err = appender.Append(id, bReq)
		assert.NoError(t, err, "unexpected error writing req")

		if len(maxID) == 0 || bytes.Compare(id, maxID) == 1 {
			maxID = id
		}
		if len(minID) == 0 || bytes.Compare(id, minID) == -1 {
			minID = id
		}
	}
	appender.Complete()
	err = writer.Flush()
	assert.NoError(t, err, "unexpected error flushing writer")

	originatingMeta := backend.NewBlockMeta(testTenantID, uuid.New())
	originatingMeta.StartTime = time.Now().Add(-5 * time.Minute)
	originatingMeta.EndTime = time.Now().Add(5 * time.Minute)

	iterator := v0.NewRecordIterator(appender.Records(), bytes.NewReader(buffer.Bytes()))
	block, err := NewCompleteBlock(originatingMeta, iterator, .01, numMsgs, indexDownsample, tempDir, "")
	assert.NoError(t, err, "unexpected error completing block")

	// assert no flushed time
	assert.True(t, block.FlushedTime().IsZero())

	// test downsample config
	assert.Equal(t, numMsgs/indexDownsample+1, len(block.records))

	assert.True(t, bytes.Equal(block.meta.MinID, minID))
	assert.True(t, bytes.Equal(block.meta.MaxID, maxID))
	assert.Equal(t, originatingMeta.StartTime, block.meta.StartTime)
	assert.Equal(t, originatingMeta.EndTime, block.meta.EndTime)
	assert.Equal(t, originatingMeta.TenantID, block.meta.TenantID)

	for i, id := range ids {
		out := &tempopb.PushRequest{}
		foundBytes, err := block.Find(id, &mockCombiner{})
		assert.NoError(t, err)

		err = proto.Unmarshal(foundBytes, out)
		assert.NoError(t, err)

		assert.True(t, proto.Equal(out, reqs[i]))
		assert.True(t, block.bloom.Test(id))
	}

	// confirm order
	var prev *index.Record
	for _, r := range block.records {
		if prev != nil {
			assert.Greater(t, r.Start, prev.Start)
		}

		prev = r
	}
}
