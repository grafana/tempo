package wal

import (
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/joe-elliott/frigg/pkg/friggpb"
	"github.com/joe-elliott/frigg/pkg/util/test"
)

func TestCreateBlock(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	wal := New(&Config{
		filepath: tempDir,
	})

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

	wal := New(&Config{
		filepath: tempDir,
	})

	blockID := uuid.New()
	instanceID := "fake"

	block, err := wal.NewBlock(blockID, instanceID)
	assert.NoError(t, err, "unexpected error creating block")

	req := test.MakeRequest(10)
	start, length, err := block.Write(req)
	assert.NoError(t, err, "unexpected error creating writing req")

	outReq := &friggpb.PushRequest{}
	err = block.Read(start, length, outReq)
	assert.NoError(t, err, "unexpected error creating reading req")
	assert.Equal(t, req, outReq)
}

func TestIterator(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	wal := New(&Config{
		filepath: tempDir,
	})

	blockID := uuid.New()
	instanceID := "fake"

	block, err := wal.NewBlock(blockID, instanceID)
	assert.NoError(t, err, "unexpected error creating block")

	numMsgs := 10
	reqs := make([]*friggpb.PushRequest, 0, numMsgs)
	for i := 0; i < numMsgs; i++ {
		req := test.MakeRequest(rand.Int() % 1000)
		reqs = append(reqs, req)
		_, _, err := block.Write(req)
		assert.NoError(t, err, "unexpected error writing req")
	}

	iterator, err := block.Iterator()
	assert.NoError(t, err, "unexpected error getting iterator")

	outReq := &friggpb.PushRequest{}
	i := 0
	for {
		more, err := iterator(outReq)
		assert.NoError(t, err, "unexpected error creating reading req")

		if !more {
			break
		}
		assert.Equal(t, outReq, reqs[i])
		i++
	}

	assert.Equal(t, numMsgs, i)
}
