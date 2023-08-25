package vparquet2

import (
	"bytes"
	"context"
	"path"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

func TestBackendBlockFindTraceByID(t *testing.T) {
	rawR, rawW, _, err := local.New(&local.Config{
		Path: t.TempDir(),
	})
	require.NoError(t, err)

	r := backend.NewReader(rawR)
	w := backend.NewWriter(rawW)
	ctx := context.Background()

	cfg := &common.BlockConfig{
		BloomFP:             0.01,
		BloomShardSizeBytes: 100 * 1024,
	}

	// Test data - sorted by trace ID
	// Find trace by ID uses the column and page bounds,
	// which by default only stores 16 bytes, which is the first
	// half of the trace ID (which is stored as 32 hex text)
	// Therefore it is important that the test data here has
	// full-length trace IDs.
	var traces []*Trace
	for i := 0; i < 16; i++ {
		bar := "bar"
		traces = append(traces, &Trace{
			TraceID: test.ValidTraceID(nil),
			ResourceSpans: []ResourceSpans{
				{
					Resource: Resource{
						ServiceName: "s",
					},
					ScopeSpans: []ScopeSpans{
						{
							Spans: []Span{
								{
									Name: "hello",
									Attrs: []Attribute{
										{Key: "foo", Value: &bar},
									},
									SpanID:       []byte{},
									ParentSpanID: []byte{},
								},
							},
						},
					},
				},
			},
		})
	}

	// Sort
	sort.Slice(traces, func(i, j int) bool {
		return bytes.Compare(traces[i].TraceID, traces[j].TraceID) == -1
	})

	meta := backend.NewBlockMeta("fake", uuid.New(), VersionString, backend.EncNone, "")
	meta.TotalObjects = len(traces)
	s := newStreamingBlock(ctx, cfg, meta, r, w, tempo_io.NewBufferedWriter)

	// Write test data, occasionally flushing (cutting new row group)
	rowGroupSize := 5
	for _, tr := range traces {
		err := s.Add(tr, 0, 0)
		require.NoError(t, err)
		if s.CurrentBufferedObjects() >= rowGroupSize {
			_, err = s.Flush()
			require.NoError(t, err)
		}
	}
	_, err = s.Complete()
	require.NoError(t, err)

	b := newBackendBlock(s.meta, r)

	// Now find and verify all test traces
	for _, tr := range traces {
		wantProto := ParquetTraceToTempopbTrace(tr)

		gotProto, err := b.FindTraceByID(ctx, tr.TraceID, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.Equal(t, wantProto, gotProto)
	}
}

func TestBackendBlockFindTraceByID_TestData(t *testing.T) {
	rawR, _, _, err := local.New(&local.Config{
		Path: "./test-data",
	})
	require.NoError(t, err)

	r := backend.NewReader(rawR)
	ctx := context.Background()

	blocks, err := r.Blocks(ctx, "single-tenant")
	require.NoError(t, err)
	assert.Len(t, blocks, 1)

	meta, err := r.BlockMeta(ctx, blocks[0], "single-tenant")
	require.NoError(t, err)

	b := newBackendBlock(meta, r)

	iter, err := b.rawIter(context.Background(), newRowPool(10))
	require.NoError(t, err)

	sch := parquet.SchemaOf(new(Trace))
	for {
		_, row, err := iter.Next(context.Background())
		require.NoError(t, err)

		if row == nil {
			break
		}

		tr := &Trace{}
		err = sch.Reconstruct(tr, row)
		require.NoError(t, err)

		protoTr, err := b.FindTraceByID(ctx, tr.TraceID, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.NotNil(t, protoTr)
	}
}

/*func genIndex(b *testing.B, block *backendBlock) *index {
	pf, _, err := block.openForSearch(context.TODO(), common.DefaultSearchOptions())
	require.NoError(b, err)

	i := &index{}

	for j := range pf.RowGroups() {
		iter := parquetquery.NewSyncIterator(context.TODO(), pf.RowGroups()[j:j+1], 0, "", 1000, nil, "TraceID")
		defer iter.Close()

		for {
			v, err := iter.Next()
			require.NoError(b, err)
			if v == nil {
				break
			}

			i.Add(v.Entries[0].Value.ByteArray())
		}
		i.Flush()
	}

	return i
}*/

func BenchmarkFindTraceByID(b *testing.B) {
	ctx := context.TODO()
	tenantID := "1"
	blockID := uuid.MustParse("1f38c153-b798-4bba-b4f4-cc60633e5cab")

	r, _, _, err := local.New(&local.Config{
		Path: path.Join("/Users/marty/src/tmp/"),
	})
	require.NoError(b, err)

	rr := backend.NewReader(r)
	// ww := backend.NewWriter(w)

	meta, err := rr.BlockMeta(ctx, blockID, tenantID)
	require.NoError(b, err)

	traceID := meta.MinID

	block := newBackendBlock(meta, rr)

	// idx := genIndex(b, block)
	// writeBlockMeta(ctx, ww, block.meta, &common.ShardedBloomFilter{}, idx)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tr, err := block.FindTraceByID(ctx, traceID, common.DefaultSearchOptions())
		require.NoError(b, err)
		require.NotNil(b, tr)
	}
}
