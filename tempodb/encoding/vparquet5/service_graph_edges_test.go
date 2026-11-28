package vparquet5

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/go-kit/log"
	"github.com/parquet-go/parquet-go"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	tempo_io "github.com/grafana/tempo/v3/pkg/io"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/v3/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/backend/local"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
)

// Reopening the backend uses only disk: the service graph matcher and its TTL
// are deliberately absent. No synthetic spans or collapsed edges are persisted.
func TestServiceGraphEdgesPersistAcrossReopen(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "disabled", true: "enabled"}[enabled], func(t *testing.T) {
			path := t.TempDir()
			rawR, rawW, _, err := local.New(&local.Config{Path: path})
			require.NoError(t, err)
			reader, writer := backend.NewReader(rawR), backend.NewWriter(rawW)
			traceID := make([]byte, 16)
			traceID[15] = 1
			input := &tempopb.Trace{}
			for i, service := range []string{"A", "gateway", "proxy", "B"} {
				id := make([]byte, 8)
				id[7] = byte(i + 1)
				var parent []byte
				if i > 0 {
					parent = make([]byte, 8)
					parent[7] = byte(i)
				}
				input.ResourceSpans = append(input.ResourceSpans, &tracev1.ResourceSpans{
					Resource:   &resourcev1.Resource{Attributes: []*commonv1.KeyValue{tempopb.MakeKeyValueStringPtr("service.name", service)}},
					ScopeSpans: []*tracev1.ScopeSpans{{Spans: []*tracev1.Span{{TraceId: traceID, SpanId: id, ParentSpanId: parent, Kind: tracev1.Span_SPAN_KIND_INTERNAL, StartTimeUnixNano: 1, EndTimeUnixNano: 2}}}},
				})
			}
			iter := &serviceGraphEdgeIterator{testIterator: newTestIterator(), id: traceID}
			iter.Add(input, 0, 0)
			meta := backend.NewBlockMeta("tenant", uuid.New(), VersionString)
			meta.TotalObjects = 1
			cfg := &common.BlockConfig{BloomFP: 0.01, BloomShardSizeBytes: 100 * 1024, RowGroupSizeBytes: 1 << 20, PersistServiceGraphEdges: enabled}
			stored, err := CreateBlock(context.Background(), cfg, meta, iter, reader, writer)
			require.NoError(t, err)
			// A new backend instance has no access to the producer's in-memory objects.
			reopened, _, _, err := local.New(&local.Config{Path: path})
			require.NoError(t, err)
			block := newBackendBlock(stored, backend.NewReader(reopened))
			found, err := block.FindTraceByID(context.Background(), traceID, common.DefaultSearchOptions())
			require.NoError(t, err)
			require.NotNil(t, found.Trace)
			if !enabled {
				require.Empty(t, found.Trace.ServiceGraphEdges)
				return
			}
			require.Len(t, found.Trace.ServiceGraphEdges, 3)
			var pairs []string
			for _, edge := range found.Trace.ServiceGraphEdges {
				pairs = append(pairs, edge.ParentService+"->"+edge.ChildService)
				require.False(t, edge.ParentMissing)
			}
			require.ElementsMatch(t, []string{"A->gateway", "gateway->proxy", "proxy->B"}, pairs)
			spans := 0
			for _, rs := range found.Trace.ResourceSpans {
				for _, ss := range rs.ScopeSpans {
					spans += len(ss.Spans)
				}
			}
			require.Equal(t, 4, spans, "edge records must not replace or add spans")
		})
	}
}

type serviceGraphEdgeIterator struct {
	*testIterator
	id common.ID
}

func (i *serviceGraphEdgeIterator) Next(ctx context.Context) (common.ID, *tempopb.Trace, error) {
	_, trace, err := i.testIterator.Next(ctx)
	return i.id, trace, err
}

func TestServiceGraphEdgesSurviveCompaction(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir()
	rawR, rawW, _, err := local.New(&local.Config{Path: path})
	require.NoError(t, err)
	reader, writer := backend.NewReader(rawR), backend.NewWriter(rawW)
	id := make([]byte, 16)
	id[15] = 9
	makeFragment := func(ids []byte) *tempopb.Trace {
		tr := &tempopb.Trace{}
		for _, spanID := range ids {
			service := map[byte]string{1: "A", 2: "gateway", 3: "proxy", 4: "B"}[spanID]
			var parent []byte
			if spanID > 1 {
				parent = []byte{spanID - 1}
			}
			tr.ResourceSpans = append(tr.ResourceSpans, &tracev1.ResourceSpans{
				Resource:   &resourcev1.Resource{Attributes: []*commonv1.KeyValue{tempopb.MakeKeyValueStringPtr("service.name", service)}},
				ScopeSpans: []*tracev1.ScopeSpans{{Spans: []*tracev1.Span{{TraceId: id, SpanId: []byte{spanID}, ParentSpanId: parent, Kind: tracev1.Span_SPAN_KIND_INTERNAL, StartTimeUnixNano: 1, EndTimeUnixNano: 2}}}},
			})
		}
		return tr
	}
	cfg := common.BlockConfig{BloomFP: .01, BloomShardSizeBytes: 100 * 1024, RowGroupSizeBytes: 1 << 20, PersistServiceGraphEdges: true}
	var metas []*backend.BlockMeta
	for _, ids := range [][]byte{{1, 2}, {3, 4}} {
		iter := &serviceGraphEdgeIterator{testIterator: newTestIterator(), id: id}
		iter.Add(makeFragment(ids), 0, 0)
		meta := backend.NewBlockMeta("tenant", uuid.New(), VersionString)
		meta.TotalObjects = 1
		written, err := CreateBlock(ctx, &cfg, meta, iter, reader, writer)
		require.NoError(t, err)
		metas = append(metas, written)
	}
	// Disabling the feature later must preserve records already on disk.
	cfg.PersistServiceGraphEdges = false
	compactor := NewCompactor(common.CompactionOptions{
		BlockConfig: cfg, OutputBlocks: 1,
		DedupedSpans: func(int, int) {}, DisconnectedTrace: func() {}, RootlessTrace: func() {},
		ObjectsCombined: func(int, int) {}, ObjectsWritten: func(int, int) {}, BytesWritten: func(int, int) {}, TraceWritten: func(int, int) {},
	})
	compacted, err := compactor.Compact(ctx, log.NewNopLogger(), reader, writer, metas)
	require.NoError(t, err)
	require.Len(t, compacted, 1)
	reopened, _, _, err := local.New(&local.Config{Path: path})
	require.NoError(t, err)
	found, err := newBackendBlock(compacted[0], backend.NewReader(reopened)).FindTraceByID(ctx, id, common.DefaultSearchOptions())
	require.NoError(t, err)
	require.Len(t, found.Trace.ServiceGraphEdges, 3)
	var pairs []string
	for _, edge := range found.Trace.ServiceGraphEdges {
		require.False(t, edge.ParentMissing, "compaction must resolve parents from other fragments")
		pairs = append(pairs, edge.ParentService+"->"+edge.ChildService)
	}
	require.ElementsMatch(t, []string{"A->gateway", "gateway->proxy", "proxy->B"}, pairs)
}

func TestServiceGraphEdgesDoNotBackfillWhenDisabled(t *testing.T) {
	tr := &Trace{ResourceSpans: []ResourceSpans{{Resource: Resource{ServiceName: "A"}, ScopeSpans: []ScopeSpans{{Spans: []Span{
		{SpanID: []byte{1}}, {SpanID: []byte{2}, ParentSpanID: []byte{1}}, {SpanID: []byte{3}, ParentSpanID: []byte{2}},
	}}}}}}
	tr.ServiceGraphEdges = []ServiceGraphEdge{{ParentSpanID: []byte{1}, ChildSpanID: []byte{2}}}
	reconcileStoredServiceGraphEdges(tr)
	require.Len(t, tr.ServiceGraphEdges, 1, "disabled compaction must not create records for older spans")
}

func TestServiceGraphEdgesReadOldSchema(t *testing.T) {
	// Existing vParquet5 files have no edge group. New readers must treat it as
	// absent rather than requiring a rewritten historical block.
	schema, writerOptions, readerOptions := SchemaWithDynamicChanges(TraceSchemaConfig{})
	fields := parquet.Group{}
	for _, field := range schema.Fields() {
		if field.Name() != "ServiceGraphEdges" {
			fields[field.Name()] = field
		}
	}
	oldSchema := parquet.NewSchema("Trace", fields)
	var data bytes.Buffer
	writer := parquet.NewGenericWriter[*Trace](&data, append(writerOptions, oldSchema)...)
	_, err := writer.Write([]*Trace{{TraceID: []byte{1}, RootServiceName: "A"}})
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	reader := parquet.NewGenericReader[*Trace](bytes.NewReader(data.Bytes()), readerOptions...)
	defer reader.Close()
	tr := &Trace{}
	n, err := reader.Read([]*Trace{tr})
	require.Equal(t, 1, n)
	if !errors.Is(err, io.EOF) {
		require.NoError(t, err)
	}
	require.Equal(t, "A", tr.RootServiceName)
	require.Empty(t, tr.ServiceGraphEdges)
}

func BenchmarkServiceGraphEdgeStorage(b *testing.B) {
	for _, count := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			spans := make([]Span, count)
			for i := range spans {
				spans[i].SpanID = make([]byte, 8)
				binary.BigEndian.PutUint64(spans[i].SpanID, uint64(i+1))
				if i > 0 {
					spans[i].ParentSpanID = spans[i-1].SpanID
				}
			}
			tr := &Trace{ResourceSpans: []ResourceSpans{{Resource: Resource{ServiceName: "service"}, ScopeSpans: []ScopeSpans{{Spans: spans}}}}}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tr.ServiceGraphEdges = buildServiceGraphEdges(tr)
			}
		})
	}
}

// Edge generation from raw rows must use the block's optional-column schema,
// preserving upstream similarity data while adding the complete edge records.
func TestServiceGraphEdgesWithTraceSimilarity(t *testing.T) {
	ctx := context.Background()
	rawR, rawW, _, err := local.New(&local.Config{Path: t.TempDir()})
	require.NoError(t, err)
	r, w := backend.NewReader(rawR), backend.NewWriter(rawW)
	meta := backend.NewBlockMeta("tenant", uuid.New(), VersionString)
	meta.TotalObjects = 1
	meta.Options = map[string]string{TraceSignatureVersionOption: TraceSignatureVersionV1}
	cfg := &common.BlockConfig{BloomFP: .01, BloomShardSizeBytes: 100 * 1024, RowGroupSizeBytes: 1 << 20, PersistServiceGraphEdges: true}
	sb, stored := newStreamingBlock(ctx, cfg, meta, r, w, tempo_io.NewBufferedWriter)
	id := make([]byte, 16)
	id[15] = 1
	similarity := TraceSimilarity{Signature: [64]byte{42}, ShingleCount: 7}
	tr := &Trace{TraceID: id, Similarity: similarity, ResourceSpans: []ResourceSpans{
		{Resource: Resource{ServiceName: "A"}, ScopeSpans: []ScopeSpans{{Spans: []Span{{SpanID: []byte{1}}}}}},
		{Resource: Resource{ServiceName: "B"}, ScopeSpans: []ScopeSpans{{Spans: []Span{{SpanID: []byte{2}, ParentSpanID: []byte{1}}}}}},
	}}
	schema, _, _ := SchemaWithDynamicChanges(TraceSchemaConfig{Options: meta.Options})
	require.NoError(t, sb.AddRaw(id, schema.Deconstruct(nil, tr), 1, 2))
	_, err = sb.Complete()
	require.NoError(t, err)
	require.Equal(t, similarity, readSimilarityLifecycleTraces(ctx, t, stored, r)[string(id)])
	found, err := newBackendBlock(stored, r).FindTraceByID(ctx, id, common.DefaultSearchOptions())
	require.NoError(t, err)
	require.Len(t, found.Trace.ServiceGraphEdges, 1)
	require.Equal(t, "A", found.Trace.ServiceGraphEdges[0].ParentService)
	require.Equal(t, "B", found.Trace.ServiceGraphEdges[0].ChildService)
}
