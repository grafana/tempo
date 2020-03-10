package wal

import (
	"bytes"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/dgryski/go-farm"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
)

const (
	testTenantID = "fake"
)

func TestCreateBlock(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	wal, err := New(&Config{
		Filepath:        tempDir,
		IndexDownsample: 2,
		BloomFP:         0.1,
	})
	assert.NoError(t, err, "unexpected error creating temp wal")

	blockID := uuid.New()

	block, err := wal.NewBlock(blockID, testTenantID)
	assert.NoError(t, err, "unexpected error creating block")

	blocks, err := wal.AllBlocks()
	assert.NoError(t, err, "unexpected error getting blocks")
	assert.Len(t, blocks, 1)

	assert.Equal(t, block.fullFilename(), blocks[0].(*CompleteBlock).fullFilename())
}

func TestReadWrite(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	wal, err := New(&Config{
		Filepath:        tempDir,
		IndexDownsample: 2,
		BloomFP:         0.1,
	})
	assert.NoError(t, err, "unexpected error creating temp wal")

	blockID := uuid.New()

	block, err := wal.NewBlock(blockID, testTenantID)
	assert.NoError(t, err, "unexpected error creating block")

	req := test.MakeRequest(10, []byte{0x00, 0x01})
	bReq, err := proto.Marshal(req)
	assert.NoError(t, err)
	err = block.Write([]byte{0x00, 0x01}, bReq)
	assert.NoError(t, err, "unexpected error creating writing req")

	foundBytes, err := block.Find([]byte{0x00, 0x01})
	assert.NoError(t, err, "unexpected error creating reading req")

	outReq := &tempopb.PushRequest{}
	err = proto.Unmarshal(foundBytes, outReq)
	assert.NoError(t, err)
	assert.True(t, proto.Equal(req, outReq))
}

func TestAppend(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	wal, err := New(&Config{
		Filepath:        tempDir,
		IndexDownsample: 2,
		BloomFP:         0.1,
	})
	assert.NoError(t, err, "unexpected error creating temp wal")

	blockID := uuid.New()

	block, err := wal.NewBlock(blockID, testTenantID)
	assert.NoError(t, err, "unexpected error creating block")

	numMsgs := 1
	reqs := make([]*tempopb.PushRequest, 0, numMsgs)
	for i := 0; i < numMsgs; i++ {
		req := test.MakeRequest(rand.Int()%1000, []byte{0x01})
		reqs = append(reqs, req)
		bReq, err := proto.Marshal(req)
		assert.NoError(t, err)
		err = block.Write([]byte{0x01}, bReq)
		assert.NoError(t, err, "unexpected error writing req")
	}

	records := block.appender.Records()
	file, err := block.file()
	assert.NoError(t, err)
	iterator := backend.NewRecordIterator(records, file)
	i := 0

	for {
		bytesID, bytesObject, err := iterator.Next()
		assert.NoError(t, err)
		if bytesID == nil {
			break
		}

		req := &tempopb.PushRequest{}
		err = proto.Unmarshal(bytesObject, req)
		assert.NoError(t, err)

		assert.True(t, proto.Equal(req, reqs[i]))
		i++
	}
	assert.Equal(t, numMsgs, i)
}

func TestIterator(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	wal, err := New(&Config{
		Filepath:        tempDir,
		IndexDownsample: 2,
		BloomFP:         0.1,
	})
	assert.NoError(t, err, "unexpected error creating temp wal")

	blockID := uuid.New()

	block, err := wal.NewBlock(blockID, testTenantID)
	assert.NoError(t, err, "unexpected error creating block")

	numMsgs := 10
	for i := 0; i < numMsgs; i++ {
		req := test.MakeRequest(rand.Int()%1000, []byte{0x01})
		bReq, err := proto.Marshal(req)
		assert.NoError(t, err)
		err = block.Write([]byte{0x01}, bReq)
		assert.NoError(t, err, "unexpected error writing req")
	}

	i := 0
	completeBlock, err := block.Complete(wal)
	assert.NoError(t, err)

	iterator, err := completeBlock.Iterator()
	assert.NoError(t, err)

	for {
		id, msg, err := iterator.Next()
		assert.NoError(t, err)

		if id == nil {
			break
		}

		req := &tempopb.PushRequest{}
		err = proto.Unmarshal(msg, req)
		assert.NoError(t, err)
		i++
	}

	assert.NoError(t, err, "unexpected error iterating")
	assert.Equal(t, numMsgs, i)
}

func TestCompleteBlock(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	indexDownsample := 13
	wal, err := New(&Config{
		Filepath:        tempDir,
		IndexDownsample: indexDownsample,
		BloomFP:         .01,
	})
	assert.NoError(t, err, "unexpected error creating temp wal")

	blockID := uuid.New()

	block, err := wal.NewBlock(blockID, testTenantID)
	assert.NoError(t, err, "unexpected error creating block")

	numMsgs := 100
	reqs := make([]*tempopb.PushRequest, 0, numMsgs)
	ids := make([][]byte, 0, numMsgs)
	for i := 0; i < numMsgs; i++ {
		id := make([]byte, 16)
		rand.Read(id)
		req := test.MakeRequest(rand.Int()%1000, id)
		reqs = append(reqs, req)
		ids = append(ids, id)
		bReq, err := proto.Marshal(req)
		assert.NoError(t, err)
		err = block.Write(id, bReq)
		assert.NoError(t, err, "unexpected error writing req")
	}

	complete, err := block.Complete(wal)
	assert.NoError(t, err, "unexpected error completing block")
	// test downsample config
	assert.Equal(t, numMsgs/indexDownsample+1, len(complete.records))

	assert.True(t, bytes.Equal(complete.meta.MinID, block.meta.MinID))
	assert.True(t, bytes.Equal(complete.meta.MaxID, block.meta.MaxID))

	for i, id := range ids {
		out := &tempopb.PushRequest{}
		foundBytes, err := complete.Find(id)
		assert.NoError(t, err)

		err = proto.Unmarshal(foundBytes, out)
		assert.NoError(t, err)

		assert.True(t, proto.Equal(out, reqs[i]))
		assert.True(t, complete.BloomFilter().Has(farm.Fingerprint64(id)))
	}

	// confirm order
	var prev *backend.Record
	for _, r := range complete.records {
		if prev != nil {
			assert.Greater(t, r.Start, prev.Start)
		}

		prev = r
	}
}

func TestWorkDir(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	err = os.MkdirAll(path.Join(tempDir, workDir), os.ModePerm)
	assert.NoError(t, err, "unexpected error creating workdir")

	_, err = os.Create(path.Join(tempDir, workDir, "testfile"))
	assert.NoError(t, err, "unexpected error creating testfile")

	_, err = New(&Config{
		Filepath:        tempDir,
		IndexDownsample: 2,
		BloomFP:         0.1,
	})
	assert.NoError(t, err, "unexpected error creating temp wal")

	_, err = os.Stat(path.Join(tempDir, workDir))
	assert.NoError(t, err, "work folder should exist")

	files, err := ioutil.ReadDir(path.Join(tempDir, workDir))
	assert.NoError(t, err, "unexpected reading work dir")

	assert.Len(t, files, 0, "work dir should be empty")
}

func BenchmarkWriteRead(b *testing.B) {
	tempDir, _ := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)

	wal, _ := New(&Config{
		Filepath:        tempDir,
		IndexDownsample: 2,
		BloomFP:         0.1,
	})

	blockID := uuid.New()

	// 1 million requests, 10k spans per request
	block, _ := wal.NewBlock(blockID, testTenantID)
	numMsgs := 100
	reqs := make([]*tempopb.PushRequest, 0, numMsgs)
	for i := 0; i < numMsgs; i++ {
		req := test.MakeRequest(100, []byte{})
		reqs = append(reqs, req)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, req := range reqs {
			bytes, _ := proto.Marshal(req)
			_ = block.Write(req.Batch.Spans[0].TraceId, bytes)
			_, _ = block.Find(req.Batch.Spans[0].TraceId)
		}
	}
}
