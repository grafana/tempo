package frontend

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

func TestDedupeSpanIDs(t *testing.T) {
	tests := []struct {
		name        string
		trace       *tempopb.Trace
		checkParent bool
	}{
		{
			name: "no duplicates",
			trace: &tempopb.Trace{
				Batches: []*v1.ResourceSpans{
					{
						InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
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
						InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
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
			checkParent: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &spanIDDeduper{
				trace: tt.trace,
			}
			s.dedupe()
			assert.Equal(t, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, s.trace.Batches[0].InstrumentationLibrarySpans[0].Spans[0].SpanId)
			assert.Equal(t, v1.Span_SPAN_KIND_CLIENT, s.trace.Batches[0].InstrumentationLibrarySpans[0].Spans[0].Kind)
			assert.Equal(t, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02}, s.trace.Batches[0].InstrumentationLibrarySpans[0].Spans[1].SpanId)
			assert.Equal(t, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03}, s.trace.Batches[0].InstrumentationLibrarySpans[0].Spans[2].SpanId)
			assert.Equal(t, v1.Span_SPAN_KIND_SERVER, s.trace.Batches[0].InstrumentationLibrarySpans[0].Spans[2].Kind)

			if tt.checkParent {
				assert.EqualValues(t, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, s.trace.Batches[0].InstrumentationLibrarySpans[0].Spans[2].ParentSpanId)
			}
		})
	}

}
