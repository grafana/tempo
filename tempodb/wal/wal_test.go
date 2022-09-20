package wal

import (
	"bytes"
	"context"
	"io"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"

	"github.com/go-kit/log" //nolint:all
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
)

const (
	testTenantID = "fake"
)

func TestCompletedDirIsRemoved(t *testing.T) {
	// Create /completed/testfile and verify it is removed.
	tempDir := t.TempDir()

	err := os.MkdirAll(path.Join(tempDir, completedDir), os.ModePerm)
	require.NoError(t, err, "unexpected error creating completedDir")

	_, err = os.Create(path.Join(tempDir, completedDir, "testfile"))
	require.NoError(t, err, "unexpected error creating testfile")

	_, err = New(&Config{
		Filepath: tempDir,
	})
	require.NoError(t, err, "unexpected error creating temp wal")

	_, err = os.Stat(path.Join(tempDir, completedDir))
	require.Error(t, err, "completedDir should not exist")
}

func TestAppendBlockStartEnd(t *testing.T) {
	wal, err := New(&Config{
		Filepath:       t.TempDir(),
		Encoding:       backend.EncNone,
		IngestionSlack: 2 * time.Minute,
	})
	require.NoError(t, err, "unexpected error creating temp wal")

	blockID := uuid.New()
	block, err := wal.NewBlock(blockID, testTenantID, "")
	require.NoError(t, err, "unexpected error creating block")

	// create a new block and confirm start/end times are correct
	blockStart := uint32(time.Now().Unix())
	blockEnd := uint32(time.Now().Add(time.Minute).Unix())

	for i := 0; i < 10; i++ {
		bytes := make([]byte, 16)
		rand.Read(bytes)

		err = block.Append(bytes, bytes, blockStart, blockEnd)
		require.NoError(t, err, "unexpected error writing req")
	}

	require.Equal(t, blockStart, uint32(block.Meta().StartTime.Unix()))
	require.Equal(t, blockEnd, uint32(block.Meta().EndTime.Unix()))

	// rescan the block and make sure that start/end times are correct
	blockStart = uint32(time.Now().Add(-time.Hour).Unix())
	blockEnd = uint32(time.Now().Unix())

	blocks, err := wal.RescanBlocks(func([]byte, string) (uint32, uint32, error) {
		return blockStart, blockEnd, nil
	}, time.Hour, log.NewNopLogger())
	require.NoError(t, err, "unexpected error getting blocks")
	require.Len(t, blocks, 1)

	require.Equal(t, blockStart, uint32(blocks[0].Meta().StartTime.Unix()))
	require.Equal(t, blockEnd, uint32(blocks[0].Meta().EndTime.Unix()))
}

func TestAppendReplayFind(t *testing.T) {
	for _, e := range backend.SupportedEncoding {
		t.Run(e.String(), func(t *testing.T) {
			testAppendReplayFind(t, backend.EncZstd)
		})
	}
}

func testAppendReplayFind(t *testing.T, e backend.Encoding) {
	wal, err := New(&Config{
		Filepath: t.TempDir(),
		Encoding: e,
	})
	require.NoError(t, err, "unexpected error creating temp wal")

	blockID := uuid.New()

	block, err := wal.NewBlock(blockID, testTenantID, model.CurrentEncoding)
	require.NoError(t, err, "unexpected error creating block")

	enc := model.MustNewSegmentDecoder(model.CurrentEncoding)

	objects := 1000
	objs := make([][]byte, 0, objects)
	ids := make([][]byte, 0, objects)
	for i := 0; i < objects; i++ {
		id := make([]byte, 16)
		rand.Read(id)
		obj := test.MakeTrace(rand.Int()%10, id)
		ids = append(ids, id)

		b1, err := enc.PrepareForWrite(obj, 0, 0)
		require.NoError(t, err)

		b2, err := enc.ToObject([][]byte{b1})
		require.NoError(t, err)

		objs = append(objs, b2)

		err = block.Append(id, b2, 0, 0)
		require.NoError(t, err, "unexpected error writing req")
	}

	for i, id := range ids {
		obj, err := block.Find(id)
		require.NoError(t, err)
		require.Equal(t, objs[i], obj)
	}

	blocks, err := wal.RescanBlocks(func([]byte, string) (uint32, uint32, error) {
		return 0, 0, nil
	}, 0, log.NewNopLogger())
	require.NoError(t, err, "unexpected error getting blocks")
	require.Len(t, blocks, 1)

	iterator, err := blocks[0].Iterator()
	require.NoError(t, err)
	defer iterator.Close()

	// append block find
	for i, id := range ids {
		obj, err := blocks[0].Find(id)
		require.NoError(t, err)
		require.Equal(t, objs[i], obj)
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
		require.Equal(t, objs[j], obj)
		require.Equal(t, ids[j], []byte(id))
		i++
	}

	require.Equal(t, objects, i)

	err = blocks[0].Clear()
	require.NoError(t, err)
}

func TestInvalidFiles(t *testing.T) { // jpe - make this something
	// create unparseable filename
	// err = os.WriteFile(filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e:tenant:v2:notanencoding"), []byte{}, 0644)
	// require.NoError(t, err)

	// // create empty block
	// err = os.WriteFile(filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e:blerg:v2:gzip"), []byte{}, 0644)
	// require.NoError(t, err)

	// blocks, err := wal.RescanBlocks(func([]byte, string) (uint32, uint32, error) {
	// 	return 0, 0, nil
	// }, 0, log.NewNopLogger())
	// require.NoError(t, err, "unexpected error getting blocks")
	// require.Len(t, blocks, 1)

	// // confirm block has been removed
	// require.NoFileExists(t, filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e:tenant:v2:gzip"))
	// require.NoFileExists(t, filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e:blerg:v2:gzip"))

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
		obj := make([]byte, rand.Intn(100)+1)
		rand.Read(obj)
		ids = append(ids, id)
		objs = append(objs, obj)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		wal, _ := New(&Config{
			Filepath: b.TempDir(),
			Encoding: encoding,
		})

		blockID := uuid.New()
		block, err := wal.NewBlock(blockID, testTenantID, "")
		require.NoError(b, err)

		// write
		for j, obj := range objs {
			err := block.Append(ids[j], obj, 0, 0)
			require.NoError(b, err)
		}

		// find
		for _, id := range ids {
			_, err := block.Find(id)
			require.NoError(b, err)
		}

		// replay
		_, err = wal.RescanBlocks(func([]byte, string) (uint32, uint32, error) {
			return 0, 0, nil
		}, 0, log.NewNopLogger())
		require.NoError(b, err)
	}
}
