package test

import v1 "go.opentelemetry.io/proto/otlp/trace/v1"

func MakeProtoSpans(count int) []*v1.ResourceSpans {
	spans := make([]*v1.ResourceSpans, 0)
	for i := 0; i < count; i++ {
		spans = append(spans, MakeProtoSpan())
	}
	return spans
}

func MakeProtoSpan() *v1.ResourceSpans {
	return &v1.ResourceSpans{
		ScopeSpans: []*v1.ScopeSpans{
			{
				Spans: []*v1.Span{
					{
						TraceId: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
						SpanId:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
					},
				},
			},
		},
	}
}
