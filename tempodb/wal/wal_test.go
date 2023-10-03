package wal

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"errors"
	"fmt"
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
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
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
		Version:  encoding.DefaultEncoding().Version(),
	})
	require.NoError(t, err, "unexpected error creating temp wal")

	_, err = os.Stat(path.Join(tempDir, completedDir))
	require.Error(t, err, "completedDir should not exist")
}

func TestAppendBlockStartEnd(t *testing.T) {
	for _, e := range encoding.AllEncodings() {
		t.Run(e.Version(), func(t *testing.T) {
			testAppendBlockStartEnd(t, e)
		})
	}
}

func testAppendBlockStartEnd(t *testing.T, e encoding.VersionedEncoding) {
	wal, err := New(&Config{
		Filepath:       t.TempDir(),
		Encoding:       backend.EncNone,
		IngestionSlack: 3 * time.Minute,
		Version:        encoding.DefaultEncoding().Version(),
	})
	require.NoError(t, err, "unexpected error creating temp wal")

	blockID := uuid.New()
	block, err := wal.newBlock(blockID, testTenantID, model.CurrentEncoding, e.Version(), nil)
	require.NoError(t, err, "unexpected error creating block")

	enc := model.MustNewSegmentDecoder(model.CurrentEncoding)

	// create a new block and confirm start/end times are correct
	blockStart := uint32(time.Now().Add(-time.Minute).Unix())
	blockEnd := uint32(time.Now().Add(time.Minute).Unix())

	for i := 0; i < 10; i++ {
		id := make([]byte, 16)
		_, err := crand.Read(id)
		require.NoError(t, err)
		obj := test.MakeTrace(rand.Int()%10+1, id)

		b1, err := enc.PrepareForWrite(obj, blockStart, blockEnd)
		require.NoError(t, err)

		b2, err := enc.ToObject([][]byte{b1})
		require.NoError(t, err)

		err = block.Append(id, b2, blockStart, blockEnd)
		require.NoError(t, err, "unexpected error writing req")
	}

	require.NoError(t, block.Flush())

	require.Equal(t, blockStart, uint32(block.BlockMeta().StartTime.Unix()))
	require.Equal(t, blockEnd, uint32(block.BlockMeta().EndTime.Unix()))

	// rescan the block and make sure the start/end times are the same
	blocks, err := wal.RescanBlocks(time.Hour, log.NewNopLogger())
	require.NoError(t, err, "unexpected error getting blocks")
	require.Len(t, blocks, 1)

	require.Equal(t, blockStart, uint32(blocks[0].BlockMeta().StartTime.Unix()))
	require.Equal(t, blockEnd, uint32(blocks[0].BlockMeta().EndTime.Unix()))
}

func TestIngestionSlack(t *testing.T) {
	for _, e := range encoding.AllEncodings() {
		t.Run(e.Version(), func(t *testing.T) {
			testIngestionSlack(t, e)
		})
	}
}

func testIngestionSlack(t *testing.T, e encoding.VersionedEncoding) {
	wal, err := New(&Config{
		Filepath:       t.TempDir(),
		Encoding:       backend.EncNone,
		IngestionSlack: time.Minute,
		Version:        encoding.DefaultEncoding().Version(),
	})
	require.NoError(t, err, "unexpected error creating temp wal")

	blockID := uuid.New()
	block, err := wal.newBlock(blockID, testTenantID, model.CurrentEncoding, e.Version(), nil)
	require.NoError(t, err, "unexpected error creating block")

	enc := model.MustNewSegmentDecoder(model.CurrentEncoding)

	traceStart := uint32(time.Now().Add(-2 * time.Minute).Unix()) // Outside of range
	traceEnd := uint32(time.Now().Add(-1 * time.Minute).Unix())   // At end of range

	// Append a trace
	id := make([]byte, 16)
	_, err = crand.Read(id)
	require.NoError(t, err)
	obj := test.MakeTrace(rand.Int()%10+1, id)

	b1, err := enc.PrepareForWrite(obj, traceStart, traceEnd)
	require.NoError(t, err)

	b2, err := enc.ToObject([][]byte{b1})
	require.NoError(t, err)

	appendTime := time.Now()
	err = block.Append(id, b2, traceStart, traceEnd)
	require.NoError(t, err, "unexpected error writing req")

	blockStart := uint32(block.BlockMeta().StartTime.Unix())
	blockEnd := uint32(block.BlockMeta().EndTime.Unix())

	require.Equal(t, uint32(appendTime.Unix()), blockStart)
	require.Equal(t, traceEnd, blockEnd)
}

func TestFindByTraceID(t *testing.T) {
	for _, e := range encoding.AllEncodings() {
		t.Run(e.Version(), func(t *testing.T) {
			testFindByTraceID(t, e)
		})
	}
}

func testFindByTraceID(t *testing.T, e encoding.VersionedEncoding) {
	f := func(ids [][]byte, objs []*tempopb.Trace, block common.WALBlock) {
		// find all traces pushed
		ctx := context.Background()
		for i, id := range ids {
			obj, err := block.FindTraceByID(ctx, id, common.DefaultSearchOptions())
			require.NoError(t, err)
			require.Equal(t, objs[i], obj)
		}
	}

	// Test with both append methods
	t.Run("Append", func(t *testing.T) {
		runWALTestWithAppendMode(t, e.Version(), false, f)
	})
	t.Run("AppendTrace", func(t *testing.T) {
		runWALTestWithAppendMode(t, e.Version(), true, f)
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
			if errors.Is(err, io.EOF) || id == nil {
				break
			}
			require.NoError(t, err)

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
	runWALTest(t, e.Version(), func(ids [][]byte, objs []*tempopb.Trace, block common.WALBlock) {
		ctx := context.Background()

		for i, o := range objs {
			k, v := findFirstAttribute(o)
			require.NotEmpty(t, k)
			require.NotEmpty(t, v)

			resp, err := block.Search(ctx, &tempopb.SearchRequest{
				Tags: map[string]string{
					k: v,
				},
				Limit: 10,
			}, common.DefaultSearchOptions())
			if errors.Is(err, common.ErrUnsupported) {
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp.Metrics.InspectedBytes)
			require.NotZero(t, resp.Metrics.InspectedBytes)
			require.LessOrEqual(t, resp.Metrics.InspectedBytes, block.DataLength())

			require.Equal(t, 1, len(resp.Traces))
			require.Equal(t, util.TraceIDToHexString(ids[i]), resp.Traces[0].TraceID)
		}
	})
}

func TestFetch(t *testing.T) {
	for _, e := range encoding.AllEncodings() {
		t.Run(e.Version(), func(t *testing.T) {
			testFetch(t, e)
		})
	}
}

func testFetch(t *testing.T, e encoding.VersionedEncoding) {
	runWALTest(t, e.Version(), func(ids [][]byte, objs []*tempopb.Trace, block common.WALBlock) {
		ctx := context.Background()

		for i, o := range objs {
			k, v := findFirstAttribute(o)
			require.NotEmpty(t, k)
			require.NotEmpty(t, v)

			query := fmt.Sprintf("{ .%s = \"%s\" }", k, v)
			resp, err := block.Fetch(ctx, traceql.MustExtractFetchSpansRequestWithMetadata(query), common.DefaultSearchOptions())
			// not all blocks support fetch
			if errors.Is(err, common.ErrUnsupported) {
				return
			}
			require.NoError(t, err)

			// grab the first result
			ss, err := resp.Results.Next(ctx)
			require.NoError(t, err)
			require.NotNil(t, ss)

			// confirm traceid matches
			expectedID := ids[i]
			require.NotNil(t, ss)
			require.Equal(t, ss.TraceID, expectedID)

			// ensure Bytes callback is set
			require.NotNil(t, resp.Bytes())
			require.NotZero(t, resp.Bytes())
			require.LessOrEqual(t, resp.Bytes(), block.DataLength())

			// confirm no more matches
			ss, err = resp.Results.Next(ctx)
			require.NoError(t, err)
			require.Nil(t, ss)
		}
	})
}

func findFirstAttribute(obj *tempopb.Trace) (string, string) {
	for _, b := range obj.Batches {
		for _, s := range b.ScopeSpans {
			for _, span := range s.Spans {
				for _, a := range span.Attributes {
					return a.Key, a.Value.GetStringValue()
				}
			}
		}
	}

	return "", ""
}

func TestInvalidFilesAndFoldersAreHandled(t *testing.T) {
	tempDir := t.TempDir()
	wal, err := New(&Config{
		Filepath: tempDir,
		Encoding: backend.EncGZIP,
		Version:  encoding.DefaultEncoding().Version(),
	})
	require.NoError(t, err, "unexpected error creating temp wal")

	// create all valid blocks
	for _, e := range encoding.AllEncodings() {
		block, err := wal.newBlock(uuid.New(), testTenantID, model.CurrentEncoding, e.Version(), nil)
		require.NoError(t, err)

		id := make([]byte, 16)
		_, err = crand.Read(id)
		require.NoError(t, err)
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
	err = os.WriteFile(filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e:tenant:v2:notanencoding"), []byte{}, 0o644)
	require.NoError(t, err)

	// create empty block
	err = os.WriteFile(filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e:blerg:v2:gzip"), []byte{}, 0o644)
	require.NoError(t, err)

	// create unparseable block
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e+tenant+vOther"), os.ModePerm))

	blocks, err := wal.RescanBlocks(0, log.NewNopLogger())
	require.NoError(t, err, "unexpected error getting blocks")
	require.Len(t, blocks, len(encoding.AllEncodings())) // valid blocks created above

	// empty file should have been removed
	require.NoFileExists(t, filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e:blerg:v2:gzip"))

	// unparseable files/folder should have been ignored
	require.FileExists(t, filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e:tenant:v2:notanencoding"))
	require.DirExists(t, filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e+tenant+vOther"))
}

func runWALTest(t testing.TB, encoding string, runner func([][]byte, []*tempopb.Trace, common.WALBlock)) {
	runWALTestWithAppendMode(t, encoding, false, runner)
}

func runWALTestWithAppendMode(t testing.TB, encoding string, appendTrace bool, runner func([][]byte, []*tempopb.Trace, common.WALBlock)) {
	wal, err := New(&Config{
		Filepath: t.TempDir(),
		Encoding: backend.EncNone,
		Version:  encoding,
	})
	require.NoError(t, err, "unexpected error creating temp wal")

	blockID := uuid.New()

	block, err := wal.newBlock(blockID, testTenantID, model.CurrentEncoding, encoding, nil)
	require.NoError(t, err, "unexpected error creating block")

	enc := model.MustNewSegmentDecoder(model.CurrentEncoding)

	objects := 250
	objs := make([]*tempopb.Trace, 0, objects)
	ids := make([][]byte, 0, objects)
	for i := 0; i < objects; i++ {
		id := make([]byte, 16)
		_, err = crand.Read(id)
		require.NoError(t, err)
		obj := test.MakeTrace(rand.Int()%10+1, id)

		trace.SortTrace(obj)

		ids = append(ids, id)
		objs = append(objs, obj)

		if appendTrace {
			err = block.AppendTrace(id, obj, 0, 0)
			require.NoError(t, err)
		} else {
			b1, err := enc.PrepareForWrite(obj, 0, 0)
			require.NoError(t, err)

			b2, err := enc.ToObject([][]byte{b1})
			require.NoError(t, err)

			err = block.Append(id, b2, 0, 0)
			require.NoError(t, err)
		}

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

func BenchmarkAppendFlush(b *testing.B) {
	for _, enc := range encoding.AllEncodings() {
		version := enc.Version()
		b.Run(version, func(b *testing.B) {
			b.Run("Append", func(b *testing.B) {
				runWALBenchmarkWithAppendMode(b, version, b.N, false, nil)
			})
			b.Run("AppendTrace", func(b *testing.B) {
				runWALBenchmarkWithAppendMode(b, version, b.N, true, nil)
			})
		})
	}
}

func BenchmarkFindTraceByID(b *testing.B) {
	for _, enc := range encoding.AllEncodings() {
		version := enc.Version()
		b.Run(version, func(b *testing.B) {
			runWALBenchmark(b, version, 1, func(ids [][]byte, objs []*tempopb.Trace, block common.WALBlock) {
				ctx := context.Background()
				for i := 0; i < b.N; i++ {
					j := i % len(ids)

					obj, err := block.FindTraceByID(ctx, ids[j], common.DefaultSearchOptions())
					require.NoError(b, err)
					require.Equal(b, objs[j], obj)
				}
			})
		})
	}
}

func BenchmarkFindUnknownTraceID(b *testing.B) {
	for _, enc := range encoding.AllEncodings() {
		version := enc.Version()
		b.Run(version, func(b *testing.B) {
			runWALBenchmark(b, version, 1, func(ids [][]byte, objs []*tempopb.Trace, block common.WALBlock) {
				for i := 0; i < b.N; i++ {
					_, err := block.FindTraceByID(context.Background(), common.ID{}, common.DefaultSearchOptions())
					require.NoError(b, err)
				}
			})
		})
	}
}

func BenchmarkSearch(b *testing.B) {
	for _, enc := range encoding.AllEncodings() {
		version := enc.Version()
		b.Run(version, func(b *testing.B) {
			runWALBenchmark(b, version, 1, func(ids [][]byte, objs []*tempopb.Trace, block common.WALBlock) {
				ctx := context.Background()

				for i := 0; i < b.N; i++ {
					j := i % len(ids)
					id, o := ids[j], objs[j]

					k, v := findFirstAttribute(o)
					require.NotEmpty(b, k)
					require.NotEmpty(b, v)

					resp, err := block.Search(ctx, &tempopb.SearchRequest{
						Tags: map[string]string{
							k: v,
						},
						Limit: 10,
					}, common.DefaultSearchOptions())
					if errors.Is(err, common.ErrUnsupported) {
						return
					}
					require.NoError(b, err)
					require.Equal(b, 1, len(resp.Traces))
					require.Equal(b, util.TraceIDToHexString(id), resp.Traces[0].TraceID)
				}
			})
		})
	}
}

func runWALBenchmark(b *testing.B, encoding string, flushCount int, runner func([][]byte, []*tempopb.Trace, common.WALBlock)) {
	runWALBenchmarkWithAppendMode(b, encoding, flushCount, false, runner)
}

func runWALBenchmarkWithAppendMode(b *testing.B, encoding string, flushCount int, appendTrace bool, runner func([][]byte, []*tempopb.Trace, common.WALBlock)) {
	wal, err := New(&Config{
		Filepath: b.TempDir(),
		Encoding: backend.EncNone,
		Version:  encoding,
	})
	require.NoError(b, err, "unexpected error creating temp wal")

	blockID := uuid.New()

	block, err := wal.newBlock(blockID, testTenantID, model.CurrentEncoding, encoding, nil)
	require.NoError(b, err, "unexpected error creating block")

	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	objects := 250
	traces := make([]*tempopb.Trace, 0, objects)
	objs := make([][]byte, 0, objects)
	ids := make([][]byte, 0, objects)
	for i := 0; i < objects; i++ {
		id := make([]byte, 16)
		_, err = crand.Read(id)
		require.NoError(b, err)
		obj := test.MakeTrace(rand.Int()%10+1, id)

		trace.SortTrace(obj)

		ids = append(ids, id)
		traces = append(traces, obj)

		b1, err := dec.PrepareForWrite(obj, 0, 0)
		require.NoError(b, err)

		b2, err := dec.ToObject([][]byte{b1})
		require.NoError(b, err)

		objs = append(objs, b2)
	}

	b.ResetTimer()

	for flush := 0; flush < flushCount; flush++ {

		for i := range traces {
			if appendTrace {
				require.NoError(b, block.AppendTrace(ids[i], traces[i], 0, 0))
			} else {
				require.NoError(b, block.Append(ids[i], objs[i], 0, 0))
			}
		}

		err = block.Flush()
		require.NoError(b, err)
	}

	if runner != nil {
		runner(ids, traces, block)
	}
}
