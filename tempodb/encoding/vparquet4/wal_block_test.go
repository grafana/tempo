package vparquet4

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: Standard wal block functionality (appending, searching, finding, etc.) is tested with all other wal blocks
//  in /tempodb/wal/wal_test.go

func TestFullFilename(t *testing.T) {
	tests := []struct {
		name     string
		b        *walBlock
		expected string
	}{
		{
			name: "basic",
			b: &walBlock{
				meta: backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), VersionString, backend.EncNone, ""),
				path: "/blerg",
			},
			expected: "/blerg/123e4567-e89b-12d3-a456-426614174000+foo+vParquet4",
		},
		{
			name: "no path",
			b: &walBlock{
				meta: backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), VersionString, backend.EncNone, ""),
				path: "",
			},
			expected: "123e4567-e89b-12d3-a456-426614174000+foo+vParquet4",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.b.walPath()
			assert.Equal(t, tc.expected, actual)
		})
	}
}

// TestPartialReplay verifies that we can best-effort replay a partial/corrupted WAL block.
// This test works by flushing a WAL block across a few pages, corrupting one, and then replaying
// it.
func TestPartialReplay(t *testing.T) {
	decoder := model.MustNewSegmentDecoder(model.CurrentEncoding)
	blockID := uuid.New()
	basePath := t.TempDir()

	meta := backend.NewBlockMeta("fake", blockID, VersionString, backend.EncNone, "")
	w, err := createWALBlock(meta, basePath, model.CurrentEncoding, 0)
	require.NoError(t, err)

	// Flush a set of traces across 2 pages
	count := 10
	ids := make([]common.ID, count)
	trs := make([]*tempopb.Trace, count)
	for i := 0; i < count; i++ {
		ids[i] = test.ValidTraceID(nil)
		trs[i] = test.MakeTrace(10, ids[i])
		trace.SortTrace(trs[i])

		b1, err := decoder.PrepareForWrite(trs[i], 0, 0)
		require.NoError(t, err)

		b2, err := decoder.ToObject([][]byte{b1})
		require.NoError(t, err)

		err = w.Append(ids[i], b2, 0, 0)
		require.NoError(t, err)

		if i+1 == count/2 {
			require.NoError(t, w.Flush())
		}
	}
	require.NoError(t, w.Flush())

	// Delete half of page 2
	fpath := w.filepathOf(1)
	info, err := os.Stat(fpath)
	require.NoError(t, err)
	require.NoError(t, os.Truncate(fpath, info.Size()/2))

	// Replay, this has a warning on page 2
	w2, warning, err := openWALBlock(filepath.Base(w.walPath()), filepath.Dir(w.walPath()), 0, 0)
	require.NoError(t, err)
	require.ErrorContains(t, warning, "invalid magic footer of parquet file")

	// Verify we iterate only the records from the first flush
	iter, err := w2.Iterator()
	require.NoError(t, err)

	gotCount := 0
	for ; ; gotCount++ {
		id, tr, err := iter.Next(context.Background())
		require.NoError(t, err)

		if id == nil {
			break
		}

		// Find trace in the input data
		match := 0
		for i := range ids {
			if bytes.Equal(ids[i], id) {
				match = i
				break
			}
		}

		require.Equal(t, ids[match], id)
		require.True(t, proto.Equal(trs[match], tr))
	}
	require.Equal(t, count/2, gotCount)
}

func TestParseFilename(t *testing.T) {
	tests := []struct {
		name            string
		filename        string
		expectUUID      uuid.UUID
		expectTenant    string
		expectedVersion string
		expectError     bool
	}{
		{
			name:            "happy path",
			filename:        "123e4567-e89b-12d3-a456-426614174000+tenant+vParquet4",
			expectUUID:      uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
			expectTenant:    "tenant",
			expectedVersion: "vParquet4",
		},
		{
			name:        "path fails",
			filename:    "/blerg/123e4567-e89b-12d3-a456-426614174000+tenant+vParquet4",
			expectError: true,
		},
		{
			name:        "no +",
			filename:    "123e4567-e89b-12d3-a456-426614174000",
			expectError: true,
		},
		{
			name:        "empty string",
			filename:    "",
			expectError: true,
		},
		{
			name:        "bad uuid",
			filename:    "123e4+tenant+vParquet",
			expectError: true,
		},
		{
			name:        "no tenant",
			filename:    "123e4567-e89b-12d3-a456-426614174000++vParquet4",
			expectError: true,
		},
		{
			name:        "no version",
			filename:    "123e4567-e89b-12d3-a456-426614174000+tenant+",
			expectError: true,
		},
		{
			name:        "wrong version",
			filename:    "123e4567-e89b-12d3-a456-426614174000+tenant+v2",
			expectError: true,
		},
		{
			name:        "wrong splits - 4",
			filename:    "123e4567-e89b-12d3-a456-426614174000+test+test+test",
			expectError: true,
		},
		{
			name:        "wrong splits - 2",
			filename:    "123e4567-e89b-12d3-a456-426614174000+test",
			expectError: true,
		},
		{
			name:        "wrong splits - 1",
			filename:    "123e4567-e89b-12d3-a456-426614174000",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actualUUID, actualTenant, actualVersion, err := parseName(tc.filename)

			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expectUUID, actualUUID)
			require.Equal(t, tc.expectTenant, actualTenant)
			require.Equal(t, tc.expectedVersion, actualVersion)
		})
	}
}

func TestWalBlockFindTraceByID(t *testing.T) {
	testWalBlock(t, func(w *walBlock, ids []common.ID, trs []*tempopb.Trace) {
		for i := range ids {
			found, err := w.FindTraceByID(context.Background(), ids[i], common.DefaultSearchOptions())
			require.NoError(t, err)
			require.NotNil(t, found)
			require.True(t, proto.Equal(trs[i], found))
		}
	})
}

func TestWalBlockIterator(t *testing.T) {
	testWalBlock(t, func(w *walBlock, ids []common.ID, trs []*tempopb.Trace) {
		iter, err := w.Iterator()
		require.NoError(t, err)

		count := 0
		for ; ; count++ {
			id, tr, err := iter.Next(context.Background())
			require.NoError(t, err)

			if id == nil {
				break
			}

			// Find trace in the input data
			match := 0
			for i := range ids {
				if bytes.Equal(ids[i], id) {
					match = i
					break
				}
			}

			require.Equal(t, ids[match], id)
			require.True(t, proto.Equal(trs[match], tr))
		}
		require.Equal(t, len(ids), count)
	})
}

// TestRowIterator cheats a bit by testing the rowIterator directly by reaching into the internals
// of walblock. it also ignores the passed in traces and ids and simply asserts that the row iterator
// is internally consistent.
func TestRowIterator(t *testing.T) {
	testWalBlock(t, func(w *walBlock, _ []common.ID, _ []*tempopb.Trace) {
		for _, f := range w.flushed {
			ri, err := f.rowIterator()
			require.NoError(t, err)

			var lastID []byte
			for {
				peekID, err := ri.peekNextID(context.Background())
				require.NoError(t, err)

				peekAgainID, err := ri.peekNextID(context.Background())
				require.NoError(t, err)
				require.Equal(t, peekID, peekAgainID)

				id, _, err := ri.Next(context.Background())
				require.NoError(t, err)
				require.Equal(t, peekID, id)
				if id == nil {
					break
				}

				// make sure ordering is correct
				require.True(t, bytes.Compare(lastID, id) < 0, "ids not in order: %v %v", lastID, id)

				lastID = append([]byte(nil), id...)
			}
		}
	})
}

func testWalBlock(t *testing.T, f func(w *walBlock, ids []common.ID, trs []*tempopb.Trace)) {
	meta := backend.NewBlockMeta("fake", uuid.New(), VersionString, backend.EncNone, "")
	w, err := createWALBlock(meta, t.TempDir(), model.CurrentEncoding, 0)
	require.NoError(t, err)

	decoder := model.MustNewSegmentDecoder(model.CurrentEncoding)

	count := 30
	ids := make([]common.ID, count)
	trs := make([]*tempopb.Trace, count)
	for i := 0; i < count; i++ {
		ids[i] = test.ValidTraceID(nil)
		trs[i] = test.MakeTrace(10, ids[i])
		trace.SortTrace(trs[i])

		b1, err := decoder.PrepareForWrite(trs[i], 0, 0)
		require.NoError(t, err)

		b2, err := decoder.ToObject([][]byte{b1})
		require.NoError(t, err)

		err = w.Append(ids[i], b2, 0, 0)
		require.NoError(t, err)

		if i%10 == 0 {
			require.NoError(t, w.Flush())
		}
	}

	require.NoError(t, w.Flush())

	f(w, ids, trs)
}

func BenchmarkWalTraceQL(b *testing.B) {
	reqs := []string{
		"{ .foo = `bar` }",
		"{ span.foo = `bar` }",
		"{ resource.foo = `bar` }",
	}

	w, warn, err := openWALBlock("15eec7d7-4b9f-4cf7-948d-fb9765ecd9a8+1+vParquet4", "/Users/marty/src/tmp/wal/", 0, 0)
	require.NoError(b, err)
	require.NoError(b, warn)

	for _, q := range reqs {
		req := traceql.MustExtractFetchSpansRequestWithMetadata(q)
		b.Run(q, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				resp, err := w.Fetch(context.TODO(), req, common.DefaultSearchOptions())
				require.NoError(b, err)

				for {
					ss, err := resp.Results.Next(context.TODO())
					require.NoError(b, err)
					if ss == nil {
						break
					}
				}
			}
		})
	}
}

func BenchmarkWalSearchTagValues(b *testing.B) {
	tags := []string{
		"service.name",
		"name",
		"foo",
		"http.url",
		"http.status_code",
		"celery.task_name",
	}

	w, warn, err := openWALBlock("15eec7d7-4b9f-4cf7-948d-fb9765ecd9a8+1+vParquet4", "/Users/marty/src/tmp/wal/", 0, 0)
	require.NoError(b, err)
	require.NoError(b, warn)

	cb := func(v string) bool {
		return true
	}

	for _, t := range tags {
		b.Run(t, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				err := w.SearchTagValues(context.TODO(), t, cb, common.DefaultSearchOptions())
				require.NoError(b, err)
			}
		})
	}
}
