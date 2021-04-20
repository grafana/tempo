package wal

import (
	"context"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
)

const (
	testTenantID = "fake"
)

type mockCombiner struct {
}

func (m *mockCombiner) Combine(objA []byte, objB []byte) []byte {
	if len(objA) > len(objB) {
		return objA
	}

	return objB
}

func TestCreateBlock(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	wal, err := New(&Config{
		Filepath: tempDir,
	})
	assert.NoError(t, err, "unexpected error creating temp wal")

	blockID := uuid.New()

	block, err := wal.NewBlock(blockID, testTenantID)
	assert.NoError(t, err, "unexpected error creating block")

	blocks, err := wal.AllBlocks()
	assert.NoError(t, err, "unexpected error getting blocks")
	assert.Len(t, blocks, 1)

	assert.Equal(t, block.fullFilename(), blocks[0].fullFilename())
}

func TestReadWrite(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	wal, err := New(&Config{
		Filepath: tempDir,
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

	foundBytes, err := block.Find([]byte{0x00, 0x01}, &mockCombiner{})
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
		Filepath: tempDir,
	})
	assert.NoError(t, err, "unexpected error creating temp wal")

	blockID := uuid.New()

	block, err := wal.NewBlock(blockID, testTenantID)
	assert.NoError(t, err, "unexpected error creating block")

	numMsgs := 100
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

	dataReader, err := block.encoding.NewDataReader(backend.NewContextReaderWithAllReader(file), backend.EncNone)
	assert.NoError(t, err)
	iterator := encoding.NewRecordIterator(records, dataReader, block.encoding.NewObjectReaderWriter())
	defer iterator.Close()
	i := 0

	for {
		_, bytesObject, err := iterator.Next(context.Background())
		if err == io.EOF {
			break
		}
		assert.NoError(t, err)

		req := &tempopb.PushRequest{}
		err = proto.Unmarshal(bytesObject, req)
		assert.NoError(t, err)

		assert.True(t, proto.Equal(req, reqs[i]))
		i++
	}
	assert.Equal(t, numMsgs, i)
}

func TestCompletedDirIsRemoved(t *testing.T) {
	// Create /completed/testfile and verify it is removed.

	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	err = os.MkdirAll(path.Join(tempDir, completedDir), os.ModePerm)
	assert.NoError(t, err, "unexpected error creating completedDir")

	_, err = os.Create(path.Join(tempDir, completedDir, "testfile"))
	assert.NoError(t, err, "unexpected error creating testfile")

	_, err = New(&Config{
		Filepath: tempDir,
	})
	assert.NoError(t, err, "unexpected error creating temp wal")

	_, err = os.Stat(path.Join(tempDir, completedDir))
	assert.Error(t, err, "completedDir should not exist")
}

func TestAppendReplayFind(t *testing.T) {
	for _, e := range backend.SupportedEncoding {
		testAppendReplayFind(t, e)
	}
}

func testAppendReplayFind(t *testing.T, e backend.Encoding) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	require.NoError(t, err, "unexpected error creating temp dir")

	wal, err := New(&Config{
		Filepath: tempDir,
		Encoding: e,
	})
	require.NoError(t, err, "unexpected error creating temp wal")

	blockID := uuid.New()

	block, err := wal.NewBlock(blockID, testTenantID)
	require.NoError(t, err, "unexpected error creating block")

	objects := 1000
	objs := make([][]byte, 0, objects)
	ids := make([][]byte, 0, objects)
	for i := 0; i < objects; i++ {
		id := make([]byte, 16)
		rand.Read(id)
		obj := test.MakeRequest(rand.Int()%10, id)
		ids = append(ids, id)
		bObj, err := proto.Marshal(obj)
		require.NoError(t, err)
		objs = append(objs, bObj)

		err = block.Write(id, bObj)
		require.NoError(t, err, "unexpected error writing req")
	}

	for i, id := range ids {
		obj, err := block.Find(id, &mockCombiner{})
		require.NoError(t, err)
		assert.Equal(t, objs[i], obj)
	}

	blocks, err := wal.AllBlocks()
	require.NoError(t, err, "unexpected error getting blocks")
	require.Len(t, blocks, 1)

	replay := blocks[0]
	iterator, err := replay.Iterator()
	require.NoError(t, err)
	defer iterator.Close()

	i := 0
	for {
		id, obj, err := iterator.Next(context.Background())
		if err == io.EOF {
			break
		} else {
			require.NoError(t, err)
		}
		assert.Equal(t, objs[i], obj)
		assert.Equal(t, ids[i], []byte(id))
		i++
	}

	assert.Equal(t, objects, i)
}

func BenchmarkWALNone(b *testing.B) {
	benchmarkWriteFindReplay(b, backend.EncNone)
}
func BenchmarkWALSnappy(b *testing.B) {
	benchmarkWriteFindReplay(b, backend.EncSnappy)
}
func BenchmarkWALLZ4(b *testing.B) {
	benchmarkWriteFindReplay(b, backend.EncLZ4_1M)
}
func BenchmarkWALGZIP(b *testing.B) {
	benchmarkWriteFindReplay(b, backend.EncGZIP)
}
func BenchmarkWALZSTD(b *testing.B) {
	benchmarkWriteFindReplay(b, backend.EncZstd)
}

func benchmarkWriteFindReplay(b *testing.B, encoding backend.Encoding) {
	objects := 1000
	objs := make([][]byte, 0, objects)
	ids := make([][]byte, 0, objects)
	for i := 0; i < objects; i++ {
		id := make([]byte, 16)
		rand.Read(id)
		obj := test.MakeRequest(rand.Int()%1000, id)
		ids = append(ids, id)
		bObj, err := proto.Marshal(obj)
		require.NoError(b, err)
		objs = append(objs, bObj)
	}
	mockCombiner := &mockCombiner{}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tempDir, _ := ioutil.TempDir("/tmp", "")
		wal, _ := New(&Config{
			Filepath: tempDir,
			Encoding: encoding,
		})

		blockID := uuid.New()
		block, err := wal.NewBlock(blockID, testTenantID)
		require.NoError(b, err)

		// write
		for j, obj := range objs {
			err := block.Write(ids[j], obj)
			require.NoError(b, err)
		}

		// find
		for _, id := range ids {
			_, err := block.Find(id, mockCombiner)
			require.NoError(b, err)
		}

		// replay
		j := 0
		replayBlocks, err := wal.AllBlocks()
		require.NoError(b, err)
		iter, err := replayBlocks[0].Iterator()
		require.NoError(b, err)
		for {
			_, _, err := iter.Next(context.Background())
			if err == io.EOF {
				break
			}
			require.NoError(b, err)
			j++
		}

		err = block.Clear()
		require.NoError(b, err)
		os.RemoveAll(tempDir)
	}
}
