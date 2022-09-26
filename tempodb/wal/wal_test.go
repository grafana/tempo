package wal

import (
	"bytes"
	"context"
	"io"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-kit/log" //nolint:all
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
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
	block, err := wal.NewBlock(blockID, testTenantID, model.CurrentEncoding)
	require.NoError(t, err, "unexpected error creating block")

	// create a new block and confirm start/end times are correct
	blockStart := uint32(time.Now().Add(-time.Minute).Unix())
	blockEnd := uint32(time.Now().Add(time.Minute).Unix())

	for i := 0; i < 10; i++ {
		id := make([]byte, 16)
		rand.Read(id)

		tr := test.MakeTrace(10, id)
		b, err := model.MustNewSegmentDecoder(model.CurrentEncoding).PrepareForWrite(tr, blockStart, blockEnd)
		require.NoError(t, err, "unexpected error writing req")

		err = block.Append(id, b, blockStart, blockEnd)
		require.NoError(t, err, "unexpected error writing req")
	}

	require.Equal(t, blockStart, uint32(block.BlockMeta().StartTime.Unix()))
	require.Equal(t, blockEnd, uint32(block.BlockMeta().EndTime.Unix()))

	// rescan the block and make sure the start/end times are the same
	blocks, err := wal.RescanBlocks(time.Hour, log.NewNopLogger())
	require.NoError(t, err, "unexpected error getting blocks")
	require.Len(t, blocks, 1)

	require.Equal(t, blockStart, uint32(blocks[0].BlockMeta().StartTime.Unix()))
	require.Equal(t, blockEnd, uint32(blocks[0].BlockMeta().EndTime.Unix()))
}

// jpe extend this test to all wal formats
// - add similar tests for search and spanset fetcher
// - test with mixed wals

func TestFindByTraceID(t *testing.T) {
	for _, e := range encoding.AllEncodings() {
		t.Run(e.Version(), func(t *testing.T) {
			testFindByTraceID(t, e)
		})
	}
}

func testFindByTraceID(t *testing.T, e encoding.VersionedEncoding) {
	runWALTest(t, e.Version(), func(ids [][]byte, objs []*tempopb.Trace, block common.WALBlock) {
		// find all traces pushed
		ctx := context.Background()
		for i, id := range ids {
			obj, err := block.FindTraceByID(ctx, id, common.DefaultSearchOptions())
			require.NoError(t, err)

			require.Equal(t, objs[i], obj)
		}
	})
}

func TestIterator(t *testing.T) {
	for _, e := range encoding.AllEncodings() {
		t.Run(e.Version(), func(t *testing.T) {
			testIterator(t, e)
		})
	}
}

func testIterator(t *testing.T, e encoding.VersionedEncoding) {
	runWALTest(t, e.Version(), func(ids [][]byte, objs []*tempopb.Trace, block common.WALBlock) {
		ctx := context.Background()

		iterator, err := block.Iterator()
		require.NoError(t, err)
		defer iterator.Close()

		i := 0
		for {
			id, obj, err := iterator.Next(ctx)
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

		require.Equal(t, len(objs), i)
	})
}

func TestSearch(t *testing.T) {
	for _, e := range encoding.AllEncodings() {
		t.Run(e.Version(), func(t *testing.T) {
			testSearch(t, e)
		})
	}
}

func testSearch(t *testing.T, e encoding.VersionedEncoding) {
	findFirstAttribute := func(obj *tempopb.Trace) (string, string) {
		for _, b := range obj.Batches {
			for _, s := range b.InstrumentationLibrarySpans {
				for _, span := range s.Spans {
					for _, a := range span.Attributes {
						return a.Key, a.Value.GetStringValue()
					}
				}
			}
		}

		return "", ""
	}

	runWALTest(t, e.Version(), func(ids [][]byte, objs []*tempopb.Trace, block common.WALBlock) {
		ctx := context.Background()

		for i, o := range objs {
			k, v := findFirstAttribute(o)
			if k == "" || v == "" {
				continue
			}

			resp, err := block.Search(ctx, &tempopb.SearchRequest{
				Tags: map[string]string{
					k: v,
				},
			}, common.DefaultSearchOptions())
			if err == common.ErrUnsupported {
				return
			}
			require.NoError(t, err)
			require.Equal(t, 1, len(resp.Traces))
			require.Equal(t, util.TraceIDToHexString(ids[i]), resp.Traces[0].TraceID)
		}
	})
}

func TestInvalidFilesAndFoldersAreHandled(t *testing.T) {
	tempDir := t.TempDir()
	wal, err := New(&Config{
		Filepath: tempDir,
		Encoding: backend.EncGZIP,
	})
	require.NoError(t, err, "unexpected error creating temp wal")

	// create all valid blocks
	for _, e := range encoding.AllEncodings() {
		block, err := wal.newBlock(uuid.New(), testTenantID, model.CurrentEncoding, e.Version())
		require.NoError(t, err)

		id := make([]byte, 16)
		rand.Read(id)
		tr := test.MakeTrace(10, id)
		b1, err := model.MustNewSegmentDecoder(model.CurrentEncoding).PrepareForWrite(tr, 0, 0)
		require.NoError(t, err)
		b2, err := model.MustNewSegmentDecoder(model.CurrentEncoding).ToObject([][]byte{b1})
		require.NoError(t, err)
		err = block.Append(id, b2, 0, 0)
		require.NoError(t, err)
		err = block.Flush()
		require.NoError(t, err)
	}

	// create unparseable filename
	err = os.WriteFile(filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e:tenant:v2:notanencoding"), []byte{}, 0644)
	require.NoError(t, err)

	// create empty block
	err = os.WriteFile(filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e:blerg:v2:gzip"), []byte{}, 0644)
	require.NoError(t, err)

	// create unparseable block
	os.MkdirAll(filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e+tenant+vOther"), os.ModePerm)

	blocks, err := wal.RescanBlocks(0, log.NewNopLogger())
	require.NoError(t, err, "unexpected error getting blocks")
	require.Len(t, blocks, len(encoding.AllEncodings())) // valid blocks created above

	// empty file should have been removed
	require.NoFileExists(t, filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e:blerg:v2:gzip"))

	// unparseable files/folder should have been ignored
	require.FileExists(t, filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e:tenant:v2:notanencoding"))
	require.DirExists(t, filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e+tenant+vOther"))
}

func runWALTest(t testing.TB, dbEncoding string, runner func([][]byte, []*tempopb.Trace, common.WALBlock)) {
	wal, err := New(&Config{
		Filepath: t.TempDir(),
		Encoding: backend.EncNone,
	})
	require.NoError(t, err, "unexpected error creating temp wal")

	blockID := uuid.New()

	block, err := wal.newBlock(blockID, testTenantID, model.CurrentEncoding, dbEncoding)
	require.NoError(t, err, "unexpected error creating block")

	enc := model.MustNewSegmentDecoder(model.CurrentEncoding)

	objects := 1000
	objs := make([]*tempopb.Trace, 0, objects)
	ids := make([][]byte, 0, objects)
	for i := 0; i < objects; i++ {
		id := make([]byte, 16)
		rand.Read(id)
		obj := test.MakeTrace(rand.Int()%10+1, id)
		ids = append(ids, id)

		b1, err := enc.PrepareForWrite(obj, 0, 0)
		require.NoError(t, err)

		b2, err := enc.ToObject([][]byte{b1})
		require.NoError(t, err)

		objs = append(objs, obj)

		err = block.Append(id, b2, 0, 0)
		require.NoError(t, err)

		if i%100 == 0 {
			err = block.Flush()
			require.NoError(t, err)
		}
	}
	err = block.Flush()
	require.NoError(t, err)

	runner(ids, objs, block)

	// rescan blocks
	blocks, err := wal.RescanBlocks(0, log.NewNopLogger())
	require.NoError(t, err, "unexpected error getting blocks")
	require.Len(t, blocks, 1)

	runner(ids, objs, blocks[0])

	err = block.Clear()
	require.NoError(t, err)
}

<<<<<<< HEAD
func TestInvalidFiles(t *testing.T) {
	tempDir := t.TempDir()
	wal, err := New(&Config{
		Filepath: tempDir,
		Encoding: backend.EncGZIP,
	})
	require.NoError(t, err, "unexpected error creating temp wal")

	// create one valid block
	block, err := wal.NewBlock(uuid.New(), testTenantID, model.CurrentEncoding)
	require.NoError(t, err)

	id := make([]byte, 16)
	rand.Read(id)
	tr := test.MakeTrace(10, id)
	b, err := model.MustNewSegmentDecoder(model.CurrentEncoding).PrepareForWrite(tr, 0, 0)
	require.NoError(t, err)
	err = block.Append(id, b, 0, 0)
	require.NoError(t, err)

	// create unparseable filename
	err = os.WriteFile(filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e+tenant+v2+notanencoding"), []byte{}, 0644)
	require.NoError(t, err)

	// create empty block
	err = os.WriteFile(filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e+blerg+v2+gzip"), []byte{}, 0644)
	require.NoError(t, err)

	blocks, err := wal.RescanBlocks(0, log.NewNopLogger())
	require.NoError(t, err, "unexpected error getting blocks")
	require.Len(t, blocks, 1) // this is our 1 valid block from above

	// confirm invalid blocks have been cleaned up
	require.NoFileExists(t, filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e+tenant+v2+notanencoding"))
	require.NoFileExists(t, filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e+blerg+v2+gzip"))
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

=======
// jpe benchmark v2 vs vparquet
func BenchmarkV2(b *testing.B) {
>>>>>>> 4f8eb179b (tests)
	for i := 0; i < b.N; i++ {
		runWALTest(b, "v2", func(ids [][]byte, objs []*tempopb.Trace, block common.WALBlock) {})
	}
}

func BenchmarkVParquet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runWALTest(b, "vParquet", func(ids [][]byte, objs []*tempopb.Trace, block common.WALBlock) {})
	}
}
