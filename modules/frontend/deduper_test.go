package frontend

import (
	"testing"
	"context"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/assert"
)

func TestDedupeSpanIDs(t *testing.T) {
	firstSpanId := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}
	secondSpanId := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02}
	thirdSpanId := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03}
	tests := []struct {
		name        string
		trace *tempopb.Trace
		shouldChange bool
	}{
		{
			name:        "no duplicates",
			trace: &tempopb.Trace{
				Batches: []*v1.ResourceSpans{
					{
						InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
							{
								Spans: []*v1.Span{
									{
										SpanId: firstSpanId,
										Kind: v1.Span_SPAN_KIND_SERVER,
									},
									{
										SpanId: secondSpanId,
									},
									{
										SpanId: thirdSpanId,
										Kind: v1.Span_SPAN_KIND_CLIENT,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:        "duplicate span id",
			trace: &tempopb.Trace{
				Batches: []*v1.ResourceSpans{
					{
						InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
							{
								Spans: []*v1.Span{
									{
										SpanId: firstSpanId,
										Kind: v1.Span_SPAN_KIND_SERVER,
									},
									{
										SpanId: secondSpanId,
									},
									{
										SpanId: firstSpanId,
										Kind: v1.Span_SPAN_KIND_CLIENT,
									},
								},
							},
						},
					},
				},
			},
			shouldChange: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deduped, err := DedupeSpanIDs(context.TODO(), tt.trace)
			assert.NoError(t, err)
			if tt.shouldChange {
				assert.NotEqual(t, firstSpanId, deduped.Batches[0].InstrumentationLibrarySpans[0].Spans[2])
			} else {
				assert.Equal(t, deduped.Batches[0].InstrumentationLibrarySpans[0].Spans[2].SpanId, thirdSpanId)
			}

		})
	}

}