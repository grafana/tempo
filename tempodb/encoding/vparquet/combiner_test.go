package vparquet

import (
	"testing"

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
						InstrumentationLibrarySpans: []ILS{
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
						InstrumentationLibrarySpans: []ILS{
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
						InstrumentationLibrarySpans: []ILS{
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
						InstrumentationLibrarySpans: []ILS{
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
