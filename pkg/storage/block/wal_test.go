package block

import (
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/grafana/frigg/pkg/friggpb"
	"github.com/grafana/frigg/pkg/util/test"
)

func TestCreateBlock(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	wal, err := New(Config{
		Filepath: tempDir,
	})
	assert.NoError(t, err, "unexpected error creating temp wal")

	blockID := uuid.New()
	instanceID := "fake"

	block, err := wal.NewBlock(blockID, instanceID)
	assert.NoError(t, err, "unexpected error creating block")

	blocks, err := wal.AllBlocks()
	assert.NoError(t, err, "unexpected error getting blocks")
	assert.Len(t, blocks, 1)

	assert.Equal(t, block, blocks[0])
}

func TestReadWrite(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	wal, err := New(Config{
		Filepath: tempDir,
	})
	assert.NoError(t, err, "unexpected error creating temp wal")

	blockID := uuid.New()
	instanceID := "fake"

	block, err := wal.NewBlock(blockID, instanceID)
	assert.NoError(t, err, "unexpected error creating block")

	req := test.MakeRequest(10, []byte{0x00, 0x01})
	err = block.Write([]byte{0x00, 0x01}, req)
	assert.NoError(t, err, "unexpected error creating writing req")

	outReq := &friggpb.PushRequest{}
	found, err := block.Find([]byte{0x00, 0x01}, outReq)
	assert.NoError(t, err, "unexpected error creating reading req")
	assert.True(t, found)
	assert.True(t, proto.Equal(req, outReq))
}

func TestIterator(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	wal, err := New(Config{
		Filepath: tempDir,
	})
	assert.NoError(t, err, "unexpected error creating temp wal")

	blockID := uuid.New()
	instanceID := "fake"

	block, err := wal.NewBlock(blockID, instanceID)
	assert.NoError(t, err, "unexpected error creating block")

	numMsgs := 10
	reqs := make([]*friggpb.PushRequest, 0, numMsgs)
	for i := 0; i < numMsgs; i++ {
		req := test.MakeRequest(rand.Int()%1000, []byte{})
		reqs = append(reqs, req)
		err := block.Write([]byte{}, req)
		assert.NoError(t, err, "unexpected error writing req")
	}

	outReq := &friggpb.PushRequest{}
	i := 0
	err = block.Iterator(outReq, func(msg proto.Message) (bool, error) {
		req := msg.(*friggpb.PushRequest)

		assert.True(t, proto.Equal(req, reqs[i]))
		i++

		return true, nil
	})

	assert.NoError(t, err, "unexpected error iterating")
	assert.Equal(t, numMsgs, i)
}

func BenchmarkWriteRead(b *testing.B) {
	tempDir, _ := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)

	wal, _ := New(Config{
		Filepath: tempDir,
	})

	blockID := uuid.New()
	instanceID := "fake"

	// 1 million requests, 10k spans per request
	block, _ := wal.NewBlock(blockID, instanceID)
	numMsgs := 100
	reqs := make([]*friggpb.PushRequest, 0, numMsgs)
	for i := 0; i < numMsgs; i++ {
		req := test.MakeRequest(100, []byte{})
		reqs = append(reqs, req)
	}

	outReq := &friggpb.PushRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, req := range reqs {
			block.Write(req.Batch.Spans[0].TraceId, req)
			block.Find(req.Batch.Spans[0].TraceId, outReq)
		}
	}
}
