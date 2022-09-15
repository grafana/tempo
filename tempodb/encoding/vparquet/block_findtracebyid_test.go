package vparquet

import (
	"bytes"
	"context"
	"path"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/pkg/util"
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
					InstrumentationLibrarySpans: []ILS{
						{
							Spans: []Span{
								{
									Name: "hello",
									Attrs: []Attribute{
										{Key: "foo", Value: &bar},
									},
									ID:           []byte{},
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
		s.Add(tr, 0, 0)
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
		wantProto := parquetTraceToTempopbTrace(tr)

		gotProto, err := b.FindTraceByID(ctx, tr.TraceID, common.SearchOptions{})
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

	iter, err := b.Iterator(context.Background())
	require.NoError(t, err)

	for {
		tr, err := iter.Next(context.Background())
		require.NoError(t, err)

		if tr == nil {
			break
		}

		// fmt.Println(tr)
		// fmt.Println("going to search for traceID", util.TraceIDToHexString(tr.TraceID))

		protoTr, err := b.FindTraceByID(ctx, tr.TraceID, common.SearchOptions{})
		require.NoError(t, err)
		require.NotNil(t, protoTr)
	}
}

func BenchmarkFindTraceByID(b *testing.B) {
	ctx := context.TODO()
	tenantID := "1"
	//blockID := uuid.MustParse("3685ee3d-cbbf-4f36-bf28-93447a19dea6")
	blockID := uuid.MustParse("1a2d50d7-f10e-41f0-850d-158b19ead23d")

	r, _, _, err := local.New(&local.Config{
		Path: path.Join("/Users/marty/src/tmp/"),
	})
	require.NoError(b, err)

	rr := backend.NewReader(r)

	meta, err := rr.BlockMeta(ctx, blockID, tenantID)
	require.NoError(b, err)

	// traceID := meta.minID
	traceID, err := util.HexStringToTraceID("1a029f7ace79c7f2")
	require.NoError(b, err)

	block := newBackendBlock(meta, rr)

	for i := 0; i < b.N; i++ {
		tr, err := block.FindTraceByID2(ctx, traceID, defaultSearchOptions())
		require.NoError(b, err)
		require.NotNil(b, tr)
	}
}
