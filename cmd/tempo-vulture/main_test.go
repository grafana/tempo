package main

import (
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	v1common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

func TestHasMissingSpans(t *testing.T) {
	cases := []struct {
		trace   *tempopb.Trace
		expeted bool
	}{
		{
			&tempopb.Trace{
				Batches: []*v1.ResourceSpans{
					{
						InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
							{
								Spans: []*v1.Span{
									{
										ParentSpanId: []byte("01234"),
									},
								},
							},
						},
					},
				},
			},
			true,
		},
		{
			&tempopb.Trace{
				Batches: []*v1.ResourceSpans{
					{
						InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
							{
								Spans: []*v1.Span{
									{
										SpanId: []byte("01234"),
									},
									{
										ParentSpanId: []byte("01234"),
									},
								},
							},
						},
					},
				},
			},
			false,
		},
	}

	for _, tc := range cases {
		require.Equal(t, tc.expeted, hasMissingSpans(tc.trace))
	}
}

func TestPbToAttributes(t *testing.T) {
	now := time.Now()
	cases := []struct {
		kvs        []*v1common.KeyValue
		attributes []attribute.KeyValue
	}{
		{
			kvs: []*v1common.KeyValue{
				{Key: "one", Value: &v1common.AnyValue{Value: &v1common.AnyValue_IntValue{IntValue: 2}}},
			},
			attributes: []attribute.KeyValue{
				attribute.Int("one", 2),
			},
		},
		{
			kvs: []*v1common.KeyValue{
				{Key: "time", Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: now.String()}}},
			},
			attributes: []attribute.KeyValue{
				attribute.String("time", now.String()),
			},
		},
	}

	for _, tc := range cases {
		attrs := pbToAttributes(tc.kvs)

		require.Equal(t, tc.attributes, attrs)
	}

}
