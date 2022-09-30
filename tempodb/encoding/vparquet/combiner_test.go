package vparquet

import (
	"testing"

	"github.com/dustin/go-humanize"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/assert"
)

func TestCombiner(t *testing.T) {

	methods := []func(a, b *Trace) (*Trace, int){
		func(a, b *Trace) (*Trace, int) {
			c := NewCombiner()
			c.Consume(a)
			c.Consume(b)
			return c.Result()
		},
	}

	tests := []struct {
		traceA        *Trace
		traceB        *Trace
		expectedTotal int
		expectedTrace *Trace
	}{
		{
			traceA:        nil,
			traceB:        &Trace{},
			expectedTotal: -1,
		},
		{
			traceA:        &Trace{},
			traceB:        nil,
			expectedTotal: -1,
		},
		{
			traceA:        &Trace{},
			traceB:        &Trace{},
			expectedTotal: 0,
		},
		{
			traceA: &Trace{
				TraceID:         []byte{0x00, 0x01},
				RootServiceName: "serviceNameA",
				ResourceSpans: []ResourceSpans{
					{
						Resource: Resource{
							ServiceName: "serviceNameA",
						},
						ScopeSpans: []ScopeSpan{
							{
								Spans: []Span{
									{
										ID:         []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
										StatusCode: 0,
									},
								},
							},
						},
					},
				},
			},
			traceB: &Trace{
				TraceID:         []byte{0x00, 0x01},
				RootServiceName: "serviceNameB",
				ResourceSpans: []ResourceSpans{
					{
						Resource: Resource{
							ServiceName: "serviceNameB",
						},
						ScopeSpans: []ScopeSpan{
							{
								Spans: []Span{
									{
										ID:           []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
										ParentSpanID: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
										StatusCode:   0,
									},
								},
							},
						},
					},
				},
			},
			expectedTotal: 2,
			expectedTrace: &Trace{
				TraceID:         []byte{0x00, 0x01},
				RootServiceName: "serviceNameA",
				ResourceSpans: []ResourceSpans{
					{
						Resource: Resource{
							ServiceName: "serviceNameA",
						},
						ScopeSpans: []ScopeSpan{
							{
								Spans: []Span{
									{
										ID:         []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
										StatusCode: 0,
									},
								},
							},
						},
					},
					{
						Resource: Resource{
							ServiceName: "serviceNameB",
						},
						ScopeSpans: []ScopeSpan{
							{
								Spans: []Span{
									{
										ID:           []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
										ParentSpanID: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
										StatusCode:   0,
									},
								},
							},
						},
					},
				},
			},
		},
		/*{
			traceA:        sameTrace,
			traceB:        sameTrace,
			expectedTotal: 100,
		},*/
	}

	for _, tt := range tests {
		for _, m := range methods {
			actualTrace, actualTotal := m(tt.traceA, tt.traceB)
			assert.Equal(t, tt.expectedTotal, actualTotal)
			if tt.expectedTrace != nil {
				assert.Equal(t, tt.expectedTrace, actualTrace)
			}
		}
	}
}

func BenchmarkCombine(b *testing.B) {

	batchCount := 100
	spanCounts := []int{
		100, 1000, 10000,
	}

	for _, spanCount := range spanCounts {
		b.Run("SpanCount:"+humanize.SI(float64(batchCount*spanCount), ""), func(b *testing.B) {
			id1 := test.ValidTraceID(nil)
			tr1 := traceToParquet(id1, test.MakeTraceWithSpanCount(batchCount, spanCount, id1))

			id2 := test.ValidTraceID(nil)
			tr2 := traceToParquet(id2, test.MakeTraceWithSpanCount(batchCount, spanCount, id2))

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				c := NewCombiner()
				c.ConsumeWithFinal(&tr1, false)
				c.ConsumeWithFinal(&tr2, true)
				c.Result()
			}
		})
	}
}

func BenchmarkSortTrace(b *testing.B) {

	batchCount := 100
	spanCounts := []int{
		100, 1000, 10000,
	}

	for _, spanCount := range spanCounts {
		b.Run("SpanCount:"+humanize.SI(float64(batchCount*spanCount), ""), func(b *testing.B) {

			id := test.ValidTraceID(nil)
			tr := traceToParquet(id, test.MakeTraceWithSpanCount(batchCount, spanCount, id))

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				SortTrace(&tr)
			}
		})
	}
}
