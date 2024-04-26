package traceql

import (
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/opentelemetry/proto/common/v1"
	"github.com/stretchr/testify/require"
)

func TestCombineResults(t *testing.T) {
	tcs := []struct {
		name     string
		existing *tempopb.TraceSearchMetadata
		new      *tempopb.TraceSearchMetadata
		expected *tempopb.TraceSearchMetadata
	}{
		{
			name: "overwrite nothing",
			existing: &tempopb.TraceSearchMetadata{
				SpanSet:  &tempopb.SpanSet{},
				SpanSets: []*tempopb.SpanSet{},
			},
			new: &tempopb.TraceSearchMetadata{
				TraceID:           "trace-1",
				RootServiceName:   "service-1",
				RootTraceName:     "root-trace-1",
				StartTimeUnixNano: 123,
				DurationMs:        100,
				SpanSets:          []*tempopb.SpanSet{},
			},
			expected: &tempopb.TraceSearchMetadata{
				TraceID:           "trace-1",
				RootServiceName:   "service-1",
				RootTraceName:     "root-trace-1",
				StartTimeUnixNano: 123,
				DurationMs:        100,
				SpanSets:          []*tempopb.SpanSet{},
			},
		},
		{
			name: "mixed copying in fields",
			existing: &tempopb.TraceSearchMetadata{
				TraceID:           "existing-trace",
				RootServiceName:   "existing-service",
				RootTraceName:     "existing-root-trace",
				StartTimeUnixNano: 100,
				DurationMs:        200,
				SpanSets:          []*tempopb.SpanSet{},
			},
			new: &tempopb.TraceSearchMetadata{
				TraceID:           "new-trace",
				RootServiceName:   "new-service",
				RootTraceName:     "new-root-trace",
				StartTimeUnixNano: 150,
				DurationMs:        300,
				SpanSets:          []*tempopb.SpanSet{},
			},
			expected: &tempopb.TraceSearchMetadata{
				TraceID:           "existing-trace",
				RootServiceName:   "existing-service",
				RootTraceName:     "existing-root-trace",
				StartTimeUnixNano: 100,
				DurationMs:        300,
				SpanSets:          []*tempopb.SpanSet{},
			},
		},
		{
			name: "copy in spansets",
			existing: &tempopb.TraceSearchMetadata{
				SpanSet:  &tempopb.SpanSet{},
				SpanSets: []*tempopb.SpanSet{},
			},
			new: &tempopb.TraceSearchMetadata{
				SpanSets: []*tempopb.SpanSet{
					{
						Matched:    3,
						Spans:      []*tempopb.Span{{SpanID: "span-1"}},
						Attributes: []*v1.KeyValue{{Key: "avg(test)", Value: &v1.AnyValue{Value: &v1.AnyValue_DoubleValue{DoubleValue: 1}}}},
					},
				},
			},
			expected: &tempopb.TraceSearchMetadata{
				SpanSets: []*tempopb.SpanSet{
					{
						Matched:    3,
						Spans:      []*tempopb.Span{{SpanID: "span-1"}},
						Attributes: []*v1.KeyValue{{Key: "avg(test)", Value: &v1.AnyValue{Value: &v1.AnyValue_DoubleValue{DoubleValue: 1}}}},
					},
				},
			},
		},
		{
			name: "take higher matches",
			existing: &tempopb.TraceSearchMetadata{
				SpanSet: &tempopb.SpanSet{},
				SpanSets: []*tempopb.SpanSet{
					{
						Matched:    3,
						Spans:      []*tempopb.Span{{SpanID: "span-1"}},
						Attributes: []*v1.KeyValue{{Key: "avg(test)", Value: &v1.AnyValue{Value: &v1.AnyValue_DoubleValue{DoubleValue: 1}}}},
					},
				},
			},
			new: &tempopb.TraceSearchMetadata{
				SpanSets: []*tempopb.SpanSet{
					{
						Matched:    5,
						Spans:      []*tempopb.Span{{SpanID: "span-2"}},
						Attributes: []*v1.KeyValue{{Key: "avg(test)", Value: &v1.AnyValue{Value: &v1.AnyValue_DoubleValue{DoubleValue: 3}}}},
					},
				},
			},
			expected: &tempopb.TraceSearchMetadata{
				SpanSets: []*tempopb.SpanSet{
					{
						Matched:    5,
						Spans:      []*tempopb.Span{{SpanID: "span-2"}},
						Attributes: []*v1.KeyValue{{Key: "avg(test)", Value: &v1.AnyValue{Value: &v1.AnyValue_DoubleValue{DoubleValue: 3}}}},
					},
				},
			},
		},
		{
			name: "keep higher matches",
			existing: &tempopb.TraceSearchMetadata{
				SpanSet: &tempopb.SpanSet{},
				SpanSets: []*tempopb.SpanSet{
					{
						Matched:    7,
						Spans:      []*tempopb.Span{{SpanID: "span-1"}},
						Attributes: []*v1.KeyValue{{Key: "avg(test)", Value: &v1.AnyValue{Value: &v1.AnyValue_DoubleValue{DoubleValue: 1}}}},
					},
				},
			},
			new: &tempopb.TraceSearchMetadata{
				SpanSets: []*tempopb.SpanSet{
					{
						Matched:    5,
						Spans:      []*tempopb.Span{{SpanID: "span-2"}},
						Attributes: []*v1.KeyValue{{Key: "avg(test)", Value: &v1.AnyValue{Value: &v1.AnyValue_DoubleValue{DoubleValue: 3}}}},
					},
				},
			},
			expected: &tempopb.TraceSearchMetadata{
				SpanSets: []*tempopb.SpanSet{
					{
						Matched:    7,
						Spans:      []*tempopb.Span{{SpanID: "span-1"}},
						Attributes: []*v1.KeyValue{{Key: "avg(test)", Value: &v1.AnyValue{Value: &v1.AnyValue_DoubleValue{DoubleValue: 1}}}},
					},
				},
			},
		},
		{
			name: "respect by()",
			existing: &tempopb.TraceSearchMetadata{
				SpanSet: &tempopb.SpanSet{},
				SpanSets: []*tempopb.SpanSet{
					{
						Matched:    7,
						Spans:      []*tempopb.Span{{SpanID: "span-1"}},
						Attributes: []*v1.KeyValue{{Key: "by(name)", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "a"}}}},
					},
					{
						Matched:    3,
						Spans:      []*tempopb.Span{{SpanID: "span-1"}},
						Attributes: []*v1.KeyValue{{Key: "by(duration)", Value: &v1.AnyValue{Value: &v1.AnyValue_DoubleValue{DoubleValue: 1.1}}}},
					},
				},
			},
			new: &tempopb.TraceSearchMetadata{
				SpanSets: []*tempopb.SpanSet{
					{
						Matched:    5,
						Spans:      []*tempopb.Span{{SpanID: "span-2"}},
						Attributes: []*v1.KeyValue{{Key: "by(name)", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "a"}}}},
					},
				},
			},
			expected: &tempopb.TraceSearchMetadata{
				SpanSets: []*tempopb.SpanSet{
					{
						Matched:    7,
						Spans:      []*tempopb.Span{{SpanID: "span-1"}},
						Attributes: []*v1.KeyValue{{Key: "by(name)", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "a"}}}},
					},
					{
						Matched:    3,
						Spans:      []*tempopb.Span{{SpanID: "span-1"}},
						Attributes: []*v1.KeyValue{{Key: "by(duration)", Value: &v1.AnyValue{Value: &v1.AnyValue_DoubleValue{DoubleValue: 1.1}}}},
					},
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			combineSearchResults(tc.existing, tc.new)

			// confirm that the SpanSet on tc.existing is contained in the slice of SpanSets
			// then nil out. the actual spanset chosen is based on map iteration order
			found := len(tc.existing.SpanSets) == 0
			for _, ss := range tc.existing.SpanSets {
				if ss == tc.existing.SpanSet {
					found = true
					break
				}
			}
			require.True(t, found)
			tc.expected.SpanSet = nil
			tc.existing.SpanSet = nil

			require.Equal(t, tc.expected, tc.existing)
		})
	}
}
