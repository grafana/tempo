package tempodb

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type rowSizeTraceIterator struct {
	traces []*tempopb.Trace
	next   int
}

func (i *rowSizeTraceIterator) Next(context.Context) (common.ID, *tempopb.Trace, error) {
	if i.next == len(i.traces) {
		return nil, nil, io.EOF
	}
	tr := i.traces[i.next]
	i.next++
	return tr.ResourceSpans[0].ScopeSpans[0].Spans[0].TraceId, tr, nil
}
func (*rowSizeTraceIterator) Close() {}

type rowSizeCompactionFixture struct {
	enc     encoding.VersionedEncoding
	cfg     common.BlockConfig
	r       backend.Reader
	w       backend.Writer
	clearer backend.Compactor
	inputs  []*backend.BlockMeta
	traces  map[string]*tempopb.Trace
}

func newRowSizeCompactionFixture(t testing.TB, enc encoding.VersionedEncoding, traceCount, spanCount, outlierSpans int, duplicate bool) *rowSizeCompactionFixture {
	t.Helper()
	rr, rw, clearer, err := local.New(&local.Config{Path: t.TempDir()})
	require.NoError(t, err)
	f := &rowSizeCompactionFixture{
		enc: enc, r: backend.NewReader(rr), w: backend.NewWriter(rw), clearer: clearer,
		cfg: common.BlockConfig{Version: enc.Version(), BloomFP: 0.01, BloomShardSizeBytes: 100 * 1024, RowGroupSizeBytes: 20_000_000}, traces: map[string]*tempopb.Trace{},
	}
	for block := 0; block < 2; block++ {
		traces := make([]*tempopb.Trace, 0, traceCount)
		for i := 0; i < traceCount; i++ {
			id := make([]byte, 16)
			n := i*2 + block + 1
			if duplicate {
				n = i + 1
			}
			binary.BigEndian.PutUint64(id[8:], uint64(n))
			count := spanCount
			if outlierSpans > 0 && i == traceCount/2 {
				count = outlierSpans
			}
			spans := make([]*tracev1.Span, count)
			rootID := make([]byte, 8)
			binary.BigEndian.PutUint64(rootID, 1)
			for j := range spans {
				spanID := make([]byte, 8)
				binary.BigEndian.PutUint64(spanID, uint64(j+1))
				start := uint64(1_700_000_000_000_000_000) + uint64(j)*1000
				spans[j] = &tracev1.Span{TraceId: id, SpanId: spanID, Name: fmt.Sprintf("operation-%d", j%11), Status: &tracev1.Status{}, StartTimeUnixNano: start, EndTimeUnixNano: start + 1000, Attributes: []*commonv1.KeyValue{{Key: "http.route", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: fmt.Sprintf("/api/%d", j%7)}}}}}
				if j > 0 {
					spans[j].ParentSpanId = rootID
				}
			}
			tr := &tempopb.Trace{ResourceSpans: []*tracev1.ResourceSpans{{Resource: &resourcev1.Resource{Attributes: []*commonv1.KeyValue{{Key: "service.name", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "benchmark"}}}}}, ScopeSpans: []*tracev1.ScopeSpans{{Scope: &commonv1.InstrumentationScope{}, Spans: spans}}}}}
			traces = append(traces, tr)
			f.traces[string(id)] = tr
		}
		meta := backend.NewBlockMeta("row-size-benchmark", uuid.New(), enc.Version())
		meta.TotalObjects = int64(traceCount)
		meta.StartTime = time.Unix(1_700_000_000, 0)
		meta.EndTime = meta.StartTime.Add(time.Second)
		out, err := enc.CreateBlock(context.Background(), &f.cfg, meta, &rowSizeTraceIterator{traces: traces}, f.r, f.w)
		require.NoError(t, err)
		f.inputs = append(f.inputs, out)
	}
	return f
}

func (f *rowSizeCompactionFixture) compact(t testing.TB) []*backend.BlockMeta {
	t.Helper()
	c := f.enc.NewCompactor(common.CompactionOptions{
		BlockConfig: f.cfg, OutputBlocks: 1,
		MaxBytesPerTrace: 50_000_000,
		ObjectsCombined:  func(int, int) {}, SpansDiscarded: func(string, string, string, int) {}, DisconnectedTrace: func() {}, RootlessTrace: func() {}, DedupedSpans: func(int, int) {},
	})
	out, err := c.Compact(context.Background(), log.NewNopLogger(), f.r, f.w, f.inputs)
	require.NoError(t, err)
	return out
}

func (f *rowSizeCompactionFixture) remove(t testing.TB, metas []*backend.BlockMeta) {
	t.Helper()
	for _, m := range metas {
		require.NoError(t, f.clearer.ClearBlock(uuid.UUID(m.BlockID), m.TenantID))
	}
}

func TestCompactionRowSize(t *testing.T) {
	for _, enc := range encoding.AllEncodings() {
		t.Run(enc.Version(), func(t *testing.T) {
			for _, duplicate := range []bool{false, true} {
				t.Run(fmt.Sprintf("duplicate=%t", duplicate), func(t *testing.T) {
					f := newRowSizeCompactionFixture(t, enc, 8, 4, 0, duplicate)
					for _, groupSize := range []int{1, 1024, 20_000_000} {
						t.Run(fmt.Sprintf("row-group=%d", groupSize), func(t *testing.T) {
							f.cfg.RowGroupSizeBytes = groupSize
							out := f.compact(t)
							defer f.remove(t, out)
							require.Len(t, out, 1)
							require.Equal(t, int64(len(f.traces)), out[0].TotalObjects)
							if groupSize == 1024 {
								groups := uint32(4)
								if enc.Version() == "vParquet4" {
									groups = 6
								}
								if duplicate {
									groups /= 2
								}
								require.Equal(t, groups, out[0].TotalRecords)
							}
							if groupSize == 1 {
								require.Equal(t, uint32(len(f.traces)+1), out[0].TotalRecords)
							}
							if groupSize == 20_000_000 {
								require.Equal(t, uint32(1), out[0].TotalRecords)
							}
							block, err := enc.OpenBlock(out[0], f.r)
							require.NoError(t, err)
							for id, want := range f.traces {
								got, err := block.FindTraceByID(context.Background(), []byte(id), common.DefaultSearchOptions())
								require.NoError(t, err)
								require.NotNil(t, got)
								if !proto.Equal(want, got.Trace) {
									require.Equal(t, want, got.Trace)
								}
							}
						})
					}
				})
			}
		})
	}
}

// BenchmarkCompactionRowSize includes reading, merging, writing and completing
// local Parquet blocks. Input creation and output cleanup are outside the timer.
func BenchmarkCompactionRowSize(b *testing.B) {
	for _, enc := range encoding.AllEncodings() {
		b.Run(enc.Version(), func(b *testing.B) {
			for _, tc := range []struct {
				name                              string
				traces, spans, outlier, groupSize int
				duplicate                         bool
			}{
				{"small", 128, 40, 0, 20_000_000, false},
				{"large", 2, 30_000, 0, 20_000_000, false},
				{"mixed", 16, 40, 30_000, 20_000_000, false},
				{"duplicates", 64, 40, 0, 20_000_000, true},
				{"frequent-flush", 16, 40, 0, 8192, false},
				{"single-row", 1, 40, 0, 1, false},
			} {
				b.Run(tc.name, func(b *testing.B) {
					f := newRowSizeCompactionFixture(b, enc, tc.traces, tc.spans, tc.outlier, tc.duplicate)
					f.cfg.RowGroupSizeBytes = tc.groupSize
					expected := int64(len(f.traces))
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						out := f.compact(b)
						b.StopTimer()
						require.Len(b, out, 1)
						require.Equal(b, expected, out[0].TotalObjects)
						f.remove(b, out)
						b.StartTimer()
					}
					b.StopTimer()
				})
			}
		})
	}
}
