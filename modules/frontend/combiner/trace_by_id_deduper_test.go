package combiner

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
)

func TestDedupeSpanIDs(t *testing.T) {
	tests := []struct {
		name        string
		trace       *tempopb.Trace
		expectedRes *tempopb.Trace
	}{
		{
			name: "no duplicates",
			trace: &tempopb.Trace{
				Batches: []*v1.ResourceSpans{
					{
						ScopeSpans: []*v1.ScopeSpans{
							{
								Spans: []*v1.Span{
									{
										SpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
										Kind:   v1.Span_SPAN_KIND_CLIENT,
									},
									{
										SpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
									},
									{
										SpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
										Kind:   v1.Span_SPAN_KIND_SERVER,
									},
								},
							},
						},
					},
				},
			},
			expectedRes: &tempopb.Trace{
				Batches: []*v1.ResourceSpans{
					{
						ScopeSpans: []*v1.ScopeSpans{
							{
								Spans: []*v1.Span{
									{
										SpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
										Kind:   v1.Span_SPAN_KIND_CLIENT,
									},
									{
										SpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
									},
									{
										SpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
										Kind:   v1.Span_SPAN_KIND_SERVER,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "duplicate span id",
			trace: &tempopb.Trace{
				Batches: []*v1.ResourceSpans{
					{
						ScopeSpans: []*v1.ScopeSpans{
							{
								Spans: []*v1.Span{
									{
										SpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
										Kind:   v1.Span_SPAN_KIND_CLIENT,
									},
									{
										SpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
									},
									{
										SpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
										Kind:   v1.Span_SPAN_KIND_SERVER,
									},
								},
							},
						},
					},
				},
			},
			expectedRes: &tempopb.Trace{
				Batches: []*v1.ResourceSpans{
					{
						ScopeSpans: []*v1.ScopeSpans{
							{
								Spans: []*v1.Span{
									{
										SpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
										Kind:   v1.Span_SPAN_KIND_CLIENT,
									},
									{
										SpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
									},
									{
										SpanId:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
										Kind:         v1.Span_SPAN_KIND_SERVER,
										ParentSpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "duplicate multi level",
			trace: &tempopb.Trace{
				Batches: []*v1.ResourceSpans{
					{
						ScopeSpans: []*v1.ScopeSpans{
							{
								Spans: []*v1.Span{
									{
										SpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
										Kind:   v1.Span_SPAN_KIND_CLIENT,
									},
									{
										SpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
									},
									{
										SpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
										Kind:   v1.Span_SPAN_KIND_SERVER,
									},
									{
										SpanId:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
										Kind:         v1.Span_SPAN_KIND_CLIENT,
										ParentSpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
									},
									{
										SpanId:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
										Kind:         v1.Span_SPAN_KIND_SERVER,
										ParentSpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
									},
								},
							},
						},
					},
				},
			},
			expectedRes: &tempopb.Trace{
				Batches: []*v1.ResourceSpans{
					{
						ScopeSpans: []*v1.ScopeSpans{
							{
								Spans: []*v1.Span{
									{
										SpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
										Kind:   v1.Span_SPAN_KIND_CLIENT,
									},
									{
										SpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
									},
									{
										SpanId:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04},
										Kind:         v1.Span_SPAN_KIND_SERVER,
										ParentSpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
									},
									{
										SpanId:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
										Kind:         v1.Span_SPAN_KIND_CLIENT,
										ParentSpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04},
									},
									{
										SpanId:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05},
										Kind:         v1.Span_SPAN_KIND_SERVER,
										ParentSpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &spanIDDeduper{}
			trace := s.dedupe(tt.trace)
			assert.Equal(t, tt.expectedRes, trace)
		})
	}
}

func BenchmarkDeduper100(b *testing.B) {
	benchmarkDeduper(b, 100)
}

func BenchmarkDeduper1000(b *testing.B) {
	benchmarkDeduper(b, 1000)
}

func BenchmarkDeduper10000(b *testing.B) {
	benchmarkDeduper(b, 10000)
}

func BenchmarkDeduper100000(b *testing.B) {
	benchmarkDeduper(b, 100000)
}

func benchmarkDeduper(b *testing.B, traceSpanCount int) {
	trace := test.MakeTraceWithSpanCount(1, traceSpanCount, []byte{0x00})
	s := newDeduper()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.dedupe(trace)
	}
}
