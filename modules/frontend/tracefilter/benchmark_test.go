package tracefilter

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
)

// matrixSizesMB are the trace sizes (MB) for the benchmark matrix.
var matrixSizesMB = []int{10, 50, 100, 200, 500}

// benchQuery is a plain span-attribute filter, so the filter and the vp5 search return comparable outputs.
const benchQuery = `{ .http.status_code = 500 }`

// BenchmarkTraceFilterMatrix benchmarks the chosen filter (perElement) against the vp5-search ground
// truth (a real in-memory vp5 search) on the same span-attribute query across matrixSizesMB. Sizes are nested
// sub-benchmarks so each can run in an isolated process (-bench Matrix/500MB). Run with -benchtime=1x:
// a single Process is representative and the vp5 block is slow to build.
func BenchmarkTraceFilterMatrix(b *testing.B) {
	for _, mb := range matrixSizesMB {
		b.Run(fmt.Sprintf("%dMB", mb), func(b *testing.B) {
			trace, spanCount := makeLargeTrace(b, mb*1024*1024)
			b.Logf("target=%dMB actualBytes=%d (%.1f MB) spans=%d", mb, trace.Size(), float64(trace.Size())/(1024*1024), spanCount)

			b.Run("perElement", func(b *testing.B) {
				filter, err := Options{Query: benchQuery}.Compile()
				require.NoError(b, err)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if _, err := filter.Process(trace); err != nil {
						b.Fatal(err)
					}
				}
			})

			// inmemVP5 is the vp5-search ground truth (a real vp5 search) - benchmarked to show the cost of exact parity.
			b.Run("inmemVP5", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = vp5Search(b, trace, benchQuery)
				}
			})
		})
	}
}

// makeLargeTrace builds a single trace of ~targetBytes made of flat root->child spans, where every
// 10th span carries http.status_code=500. Returns the trace and its span count.
func makeLargeTrace(tb testing.TB, targetBytes int) (*tempopb.Trace, int) {
	tb.Helper()

	traceID := make([]byte, 16)
	binary.BigEndian.PutUint64(traceID[8:], 0x1122334455667788)

	// a longer attribute value pushes each span toward a realistic size so ~100MB is not an absurd span count.
	filler := strings.Repeat("x", 128)

	trace := &tempopb.Trace{}
	var (
		spanCounter uint64
		spanCount   int
		rootID      []byte
		accumulated int
	)

	const spansPerBatch = 500
	for accumulated < targetBytes {
		ss := &tracev1.ScopeSpans{Scope: &commonv1.InstrumentationScope{Name: "bench-scope"}}
		for j := 0; j < spansPerBatch; j++ {
			spanCounter++
			spanCount++

			id := make([]byte, 8)
			binary.BigEndian.PutUint64(id, spanCounter)
			if rootID == nil {
				rootID = id
			}

			statusCode := int64(200)
			if spanCounter%10 == 0 {
				statusCode = 500
			}

			span := &tracev1.Span{
				TraceId:           traceID,
				SpanId:            id,
				Name:              "GET /api/resource",
				StartTimeUnixNano: uint64(1000 * time.Second),
				EndTimeUnixNano:   uint64(1001 * time.Second),
				Status:            &tracev1.Status{Code: tracev1.Status_STATUS_CODE_OK},
				Attributes: []*commonv1.KeyValue{
					intAttr("http.status_code", statusCode),
					strAttr("http.method", "GET"),
					strAttr("http.url", "https://example.com/api/resource?token="+filler),
				},
			}
			if !bytes.Equal(id, rootID) {
				span.ParentSpanId = rootID
			}
			ss.Spans = append(ss.Spans, span)
		}

		batch := &tracev1.ResourceSpans{
			Resource:   &resourcev1.Resource{Attributes: []*commonv1.KeyValue{strAttr("service.name", "checkout")}},
			ScopeSpans: []*tracev1.ScopeSpans{ss},
		}
		accumulated += batch.Size()
		trace.ResourceSpans = append(trace.ResourceSpans, batch)
	}

	return trace, spanCount
}

func strAttr(key, value string) *commonv1.KeyValue {
	return &commonv1.KeyValue{Key: key, Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: value}}}
}

func intAttr(key string, value int64) *commonv1.KeyValue {
	return &commonv1.KeyValue{Key: key, Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_IntValue{IntValue: value}}}
}

func strArrayAttr(key string, values ...string) *commonv1.KeyValue {
	vals := make([]*commonv1.AnyValue, len(values))
	for i, v := range values {
		vals[i] = &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: v}}
	}
	return &commonv1.KeyValue{Key: key, Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_ArrayValue{ArrayValue: &commonv1.ArrayValue{Values: vals}}}}
}

func intArrayAttr(key string, values ...int64) *commonv1.KeyValue {
	vals := make([]*commonv1.AnyValue, len(values))
	for i, v := range values {
		vals[i] = &commonv1.AnyValue{Value: &commonv1.AnyValue_IntValue{IntValue: v}}
	}
	return &commonv1.KeyValue{Key: key, Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_ArrayValue{ArrayValue: &commonv1.ArrayValue{Values: vals}}}}
}

// matchedHexIDs returns the sorted hex span ids of every span in the trace.
func matchedHexIDs(trace *tempopb.Trace) []string {
	var ids []string
	for _, rs := range trace.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				ids = append(ids, util.SpanIDToHexString(span.SpanId))
			}
		}
	}
	sort.Strings(ids)
	return ids
}
