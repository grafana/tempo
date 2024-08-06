package vparquet4

import (
	"testing"

	"github.com/dustin/go-humanize"
	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/v2/pkg/util/test"
	"github.com/grafana/tempo/v2/tempodb/backend"
)

func TestCombiner(t *testing.T) {
	methods := []func(a, b *Trace) (*Trace, int, bool){
		func(a, b *Trace) (*Trace, int, bool) {
			c := NewCombiner()
			c.Consume(a)
			c.Consume(b)
			return c.Result()
		},
	}

	tests := []struct {
		name          string
		traceA        *Trace
		traceB        *Trace
		expectedTotal int
		expectedTrace *Trace
	}{
		{
			name:          "nil traceA",
			traceA:        nil,
			traceB:        &Trace{},
			expectedTotal: -1,
		},
		{
			name:          "nil traceB",
			traceA:        &Trace{},
			traceB:        nil,
			expectedTotal: -1,
		},
		{
			name:          "empty traces",
			traceA:        &Trace{},
			traceB:        &Trace{},
			expectedTotal: 0,
		},
		{
			name: "root meta from second overrides empty first",
			traceA: &Trace{
				TraceID:      []byte{0x00, 0x01},
				ServiceStats: map[string]ServiceStats{},
			},
			traceB: &Trace{
				TraceID:           []byte{0x00, 0x01},
				RootServiceName:   "serviceNameB",
				RootSpanName:      "spanNameB",
				StartTimeUnixNano: 10,
				EndTimeUnixNano:   20,
				DurationNano:      10,
				ServiceStats:      map[string]ServiceStats{},
			},
			expectedTrace: &Trace{
				TraceID:           []byte{0x00, 0x01},
				RootServiceName:   "serviceNameB",
				RootSpanName:      "spanNameB",
				StartTimeUnixNano: 10,
				EndTimeUnixNano:   20,
				DurationNano:      10,
				ServiceStats:      map[string]ServiceStats{},
			},
		},
		{
			name: "if both set first root name wins",
			traceA: &Trace{
				TraceID:         []byte{0x00, 0x01},
				RootServiceName: "serviceNameA",
				RootSpanName:    "spanNameA",
				ServiceStats:    map[string]ServiceStats{},
			},
			traceB: &Trace{
				TraceID:         []byte{0x00, 0x01},
				RootServiceName: "serviceNameB",
				RootSpanName:    "spanNameB",
				ServiceStats:    map[string]ServiceStats{},
			},
			expectedTrace: &Trace{
				TraceID:         []byte{0x00, 0x01},
				RootServiceName: "serviceNameA",
				RootSpanName:    "spanNameA",
				ServiceStats:    map[string]ServiceStats{},
			},
		},
		{
			name: "second trace start/end override",
			traceA: &Trace{
				TraceID:           []byte{0x00, 0x01},
				StartTimeUnixNano: 10,
				EndTimeUnixNano:   20,
				DurationNano:      10,
			},
			traceB: &Trace{
				TraceID:           []byte{0x00, 0x01},
				StartTimeUnixNano: 5,
				EndTimeUnixNano:   25,
				DurationNano:      20,
			},
			expectedTrace: &Trace{
				TraceID:           []byte{0x00, 0x01},
				StartTimeUnixNano: 5,
				EndTimeUnixNano:   25,
				DurationNano:      20,
				ServiceStats:      map[string]ServiceStats{},
			},
		},
		{
			name: "second trace start/end ignored",
			traceA: &Trace{
				TraceID:           []byte{0x00, 0x01},
				StartTimeUnixNano: 10,
				EndTimeUnixNano:   20,
				DurationNano:      10,
			},
			traceB: &Trace{
				TraceID:           []byte{0x00, 0x01},
				StartTimeUnixNano: 12,
				EndTimeUnixNano:   18,
				DurationNano:      6,
			},
			expectedTrace: &Trace{
				TraceID:           []byte{0x00, 0x01},
				StartTimeUnixNano: 10,
				EndTimeUnixNano:   20,
				DurationNano:      10,
				ServiceStats:      map[string]ServiceStats{},
			},
		},
		{
			name: "combine spans",
			traceA: &Trace{
				TraceID:         []byte{0x00, 0x01},
				RootServiceName: "serviceNameA",
				ResourceSpans: []ResourceSpans{
					{
						Resource: Resource{
							ServiceName: "serviceNameA",
						},
						ScopeSpans: []ScopeSpans{
							{
								Spans: []Span{
									{
										SpanID:         []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
										StatusCode:     0,
										NestedSetLeft:  1,
										NestedSetRight: 2,
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
						ScopeSpans: []ScopeSpans{
							{
								Spans: []Span{
									{
										SpanID:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
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
				ServiceStats: map[string]ServiceStats{
					"serviceNameA": {
						SpanCount: 1,
					},
					"serviceNameB": {
						SpanCount: 1,
					},
				},
				ResourceSpans: []ResourceSpans{
					{
						Resource: Resource{
							ServiceName: "serviceNameA",
						},
						ScopeSpans: []ScopeSpans{
							{
								Spans: []Span{
									{
										SpanID:         []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
										StatusCode:     0,
										NestedSetLeft:  1,
										NestedSetRight: 4,
										ParentID:       -1,
									},
								},
							},
						},
					},
					{
						Resource: Resource{
							ServiceName: "serviceNameB",
						},
						ScopeSpans: []ScopeSpans{
							{
								Spans: []Span{
									{
										SpanID:         []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
										ParentSpanID:   []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
										StatusCode:     0,
										ParentID:       1,
										NestedSetLeft:  2,
										NestedSetRight: 3,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "combine spans with errors",
			traceA: &Trace{
				TraceID:         []byte{0x00, 0x01},
				RootServiceName: "serviceNameA",
				ResourceSpans: []ResourceSpans{
					{
						Resource: Resource{
							ServiceName: "serviceNameA",
						},
						ScopeSpans: []ScopeSpans{
							{
								Spans: []Span{
									{
										SpanID:         []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
										StatusCode:     0,
										NestedSetLeft:  1,
										NestedSetRight: 2,
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
						ScopeSpans: []ScopeSpans{
							{
								Spans: []Span{
									{
										SpanID:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
										ParentSpanID: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
										StatusCode:   0,
									},
									{
										SpanID:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
										ParentSpanID: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
										StatusCode:   2,
									},
								},
							},
						},
					},
				},
			},
			expectedTotal: 3,
			expectedTrace: &Trace{
				TraceID:         []byte{0x00, 0x01},
				RootServiceName: "serviceNameA",
				ServiceStats: map[string]ServiceStats{
					"serviceNameA": {
						SpanCount: 1,
					},
					"serviceNameB": {
						SpanCount:  2,
						ErrorCount: 1,
					},
				},
				ResourceSpans: []ResourceSpans{
					{
						Resource: Resource{
							ServiceName: "serviceNameA",
						},
						ScopeSpans: []ScopeSpans{
							{
								Spans: []Span{
									{
										SpanID:         []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
										StatusCode:     0,
										ParentID:       -1,
										NestedSetLeft:  1,
										NestedSetRight: 6,
									},
								},
							},
						},
					},
					{
						Resource: Resource{
							ServiceName: "serviceNameB",
						},
						ScopeSpans: []ScopeSpans{
							{
								Spans: []Span{
									{
										SpanID:         []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
										ParentSpanID:   []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
										StatusCode:     0,
										ParentID:       1,
										NestedSetLeft:  2,
										NestedSetRight: 3,
									},
									{
										SpanID:         []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
										ParentSpanID:   []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
										StatusCode:     2,
										ParentID:       1,
										NestedSetLeft:  4,
										NestedSetRight: 5,
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
		t.Run(tt.name, func(t *testing.T) {
			for _, m := range methods {
				actualTrace, actualTotal, _ := m(tt.traceA, tt.traceB)
				assert.Equal(t, tt.expectedTotal, actualTotal)
				if tt.expectedTrace != nil {
					assert.Equal(t, tt.expectedTrace, actualTrace)
				}
			}
		})
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
			tr1, _ := traceToParquet(&backend.BlockMeta{}, id1, test.MakeTraceWithSpanCount(batchCount, spanCount, id1), nil)

			id2 := test.ValidTraceID(nil)
			tr2, _ := traceToParquet(&backend.BlockMeta{}, id2, test.MakeTraceWithSpanCount(batchCount, spanCount, id2), nil)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				c := NewCombiner()
				c.ConsumeWithFinal(tr1, false)
				c.ConsumeWithFinal(tr2, true)
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
			tr, _ := traceToParquet(&backend.BlockMeta{}, id, test.MakeTraceWithSpanCount(batchCount, spanCount, id), nil)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				SortTrace(tr)
			}
		})
	}
}
