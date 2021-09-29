package wal

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/go-kit/kit/log"
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

func (m *mockCombiner) Combine(dataEncoding string, objs ...[]byte) ([]byte, bool) {
	if len(objs) != 2 {
		return nil, false
	}

	if len(objs[0]) > len(objs[1]) {
		return objs[0], true
	}

	return objs[1], true
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

	block, err := wal.NewBlock(blockID, testTenantID, "")
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

func TestErrorConditions(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	require.NoError(t, err, "unexpected error creating temp dir")

	wal, err := New(&Config{
		Filepath: tempDir,
		Encoding: backend.EncGZIP,
	})
	require.NoError(t, err, "unexpected error creating temp wal")

	blockID := uuid.New()

	// create partially corrupt block
	block, err := wal.NewBlock(blockID, testTenantID, "")
	require.NoError(t, err, "unexpected error creating block")

	objects := 10
	for i := 0; i < objects; i++ {
		id := make([]byte, 16)
		rand.Read(id)
		obj := test.MakeRequest(rand.Int()%10, id)
		bObj, err := proto.Marshal(obj)
		require.NoError(t, err)

		err = block.Write(id, bObj)
		require.NoError(t, err, "unexpected error writing req")
	}
	appendFile, err := os.OpenFile(block.fullFilename(), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	require.NoError(t, err)
	_, err = appendFile.Write([]byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01})
	require.NoError(t, err)
	err = appendFile.Close()
	require.NoError(t, err)

	// create unparseable filename
	err = os.WriteFile(filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e:tenant:v2:notanencoding"), []byte{}, 0644)
	require.NoError(t, err)

	// create empty block
	err = os.WriteFile(filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e:blerg:v2:gzip"), []byte{}, 0644)
	require.NoError(t, err)

	blocks, err := wal.RescanBlocks(log.NewNopLogger())
	require.NoError(t, err, "unexpected error getting blocks")
	require.Len(t, blocks, 1)

	assert.Equal(t, objects, blocks[0].appender.Length())

	// confirm block has been removed
	assert.NoFileExists(t, filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e:tenant:v2:gzip"))
	assert.NoFileExists(t, filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e:blerg:v2:gzip"))
}

func TestAppendReplayFind(t *testing.T) {
	for _, e := range backend.SupportedEncoding {
		t.Run(e.String(), func(t *testing.T) {
			testAppendReplayFind(t, backend.EncZstd)
		})
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

	block, err := wal.NewBlock(blockID, testTenantID, "")
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

	// write garbage data at the end to confirm a partial block will load
	appendFile, err := os.OpenFile(block.fullFilename(), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	require.NoError(t, err)
	_, err = appendFile.Write([]byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01})
	require.NoError(t, err)
	err = appendFile.Close()
	require.NoError(t, err)

	blocks, err := wal.RescanBlocks(log.NewNopLogger())
	require.NoError(t, err, "unexpected error getting blocks")
	require.Len(t, blocks, 1)

	iterator, err := blocks[0].GetIterator(&mockCombiner{})
	require.NoError(t, err)
	defer iterator.Close()

	// append block find
	for i, id := range ids {
		obj, err := blocks[0].Find(id, &mockCombiner{})
		require.NoError(t, err)
		assert.Equal(t, objs[i], obj)
	}

	i := 0
	for {
		id, obj, err := iterator.Next(context.Background())
		if err == io.EOF {
			break
		} else {
			require.NoError(t, err)
		}

		found := false
		j := 0
		for ; j < len(ids); j++ {
			if bytes.Equal(ids[j], id) {
				found = true
				break
			}
		}

		require.True(t, found)
		assert.Equal(t, objs[j], obj)
		assert.Equal(t, ids[j], []byte(id))
		i++
	}

	assert.Equal(t, objects, i)

	err = blocks[0].Clear()
	require.NoError(t, err)
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

func BenchmarkWALS2(b *testing.B) {
	benchmarkWriteFindReplay(b, backend.EncS2)
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
		block, err := wal.NewBlock(blockID, testTenantID, "")
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
		_, err = wal.RescanBlocks(log.NewNopLogger())
		require.NoError(b, err)

		os.RemoveAll(tempDir)
	}
}
