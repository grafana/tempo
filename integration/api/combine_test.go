package api

import (
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/stretchr/testify/require"
)

func TestNaiveTagValuesV2Combine(t *testing.T) {
	tests := []struct {
		name     string
		rNew     *tempopb.SearchTagValuesV2Response
		rInto    *tempopb.SearchTagValuesV2Response
		expected *tempopb.SearchTagValuesV2Response
	}{
		{
			name: "combine unique tag values",
			rNew: &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{
					{Type: "string", Value: "value1"},
					{Type: "string", Value: "value2"},
				},
				Metrics: &tempopb.MetadataMetrics{
					InspectedBytes: 100,
					CompletedJobs:  1,
				},
			},
			rInto: &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{
					{Type: "string", Value: "value3"},
				},
				Metrics: &tempopb.MetadataMetrics{
					InspectedBytes: 50,
					CompletedJobs:  1,
				},
			},
			expected: &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{
					{Type: "string", Value: "value3"},
					{Type: "string", Value: "value1"},
					{Type: "string", Value: "value2"},
				},
				Metrics: &tempopb.MetadataMetrics{
					InspectedBytes: 150,
					CompletedJobs:  2,
				},
			},
		},
		{
			name: "skip duplicate tag values",
			rNew: &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{
					{Type: "string", Value: "value1"},
					{Type: "string", Value: "value2"},
				},
				Metrics: &tempopb.MetadataMetrics{
					InspectedBytes: 100,
					CompletedJobs:  1,
				},
			},
			rInto: &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{
					{Type: "string", Value: "value1"},
				},
				Metrics: &tempopb.MetadataMetrics{
					InspectedBytes: 50,
					CompletedJobs:  1,
				},
			},
			expected: &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{
					{Type: "string", Value: "value1"},
					{Type: "string", Value: "value2"},
				},
				Metrics: &tempopb.MetadataMetrics{
					InspectedBytes: 150,
					CompletedJobs:  2,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			naiveTagValuesV2Combine(tt.rNew, tt.rInto)
			require.Equal(t, tt.expected, tt.rInto)
		})
	}
}

func TestNaiveTagsV2Combine(t *testing.T) {
	tests := []struct {
		name     string
		rNew     *tempopb.SearchTagsV2Response
		rInto    *tempopb.SearchTagsV2Response
		expected int // expected number of unique tags across all scopes
	}{
		{
			name: "combine unique tags from different scopes",
			rNew: &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
					{
						Name: "resource",
						Tags: []string{"tag1", "tag2"},
					},
				},
				Metrics: &tempopb.MetadataMetrics{
					InspectedBytes: 100,
					CompletedJobs:  1,
				},
			},
			rInto: &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
					{
						Name: "span",
						Tags: []string{"tag3"},
					},
				},
				Metrics: &tempopb.MetadataMetrics{
					InspectedBytes: 50,
					CompletedJobs:  1,
				},
			},
			expected: 3, // tag1, tag2, tag3
		},
		{
			name: "deduplicate tags within same scope",
			rNew: &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
					{
						Name: "resource",
						Tags: []string{"tag1", "tag2"},
					},
				},
				Metrics: &tempopb.MetadataMetrics{
					InspectedBytes: 100,
					CompletedJobs:  1,
				},
			},
			rInto: &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
					{
						Name: "resource",
						Tags: []string{"tag1", "tag3"},
					},
				},
				Metrics: &tempopb.MetadataMetrics{
					InspectedBytes: 50,
					CompletedJobs:  1,
				},
			},
			expected: 3, // tag1, tag2, tag3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			naiveTagsV2Combine(tt.rNew, tt.rInto)

			// Count total unique tags across all scopes
			totalTags := 0
			for _, scope := range tt.rInto.Scopes {
				totalTags += len(scope.Tags)
			}
			require.Equal(t, tt.expected, totalTags)

			// Verify metrics were combined
			expectedBytes := tt.rNew.Metrics.InspectedBytes + 50 // original rInto value
			expectedJobs := tt.rNew.Metrics.CompletedJobs + 1    // original rInto value
			require.Equal(t, expectedBytes, tt.rInto.Metrics.InspectedBytes)
			require.Equal(t, expectedJobs, tt.rInto.Metrics.CompletedJobs)
		})
	}
}

func TestNaiveQueryRangeCombine(t *testing.T) {
	tests := []struct {
		name     string
		rNew     *tempopb.QueryRangeResponse
		rInto    *tempopb.QueryRangeResponse
		expected *tempopb.QueryRangeResponse
	}{
		{
			name: "combine series with different labels",
			rNew: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					{
						Labels: []v1.KeyValue{
							{Key: "foo", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "bar"}}},
						},
						Samples: []tempopb.Sample{{TimestampMs: 1000, Value: 1.0}},
					},
				},
				Metrics: &tempopb.SearchMetrics{
					InspectedBytes:  100,
					CompletedJobs:   1,
					InspectedTraces: 10,
					InspectedSpans:  50,
				},
			},
			rInto: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					{
						Labels: []v1.KeyValue{
							{Key: "baz", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "qux"}}},
						},
						Samples: []tempopb.Sample{{TimestampMs: 2000, Value: 2.0}},
					},
				},
				Metrics: &tempopb.SearchMetrics{
					InspectedBytes:  50,
					CompletedJobs:   1,
					InspectedTraces: 5,
					InspectedSpans:  25,
				},
			},
			expected: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					{
						Labels: []v1.KeyValue{
							{Key: "baz", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "qux"}}},
						},
						Samples: []tempopb.Sample{{TimestampMs: 2000, Value: 2.0}},
					},
					{
						Labels: []v1.KeyValue{
							{Key: "foo", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "bar"}}},
						},
						Samples: []tempopb.Sample{{TimestampMs: 1000, Value: 1.0}},
					},
				},
				Metrics: &tempopb.SearchMetrics{
					InspectedBytes:  150,
					CompletedJobs:   2,
					InspectedTraces: 15,
					InspectedSpans:  75,
				},
			},
		},
		{
			name: "combine series with same labels",
			rNew: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					{
						Labels: []v1.KeyValue{
							{Key: "foo", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "bar"}}},
						},
						Samples: []tempopb.Sample{{TimestampMs: 2000, Value: 2.0}},
					},
				},
				Metrics: &tempopb.SearchMetrics{
					InspectedBytes:  100,
					CompletedJobs:   1,
					InspectedTraces: 10,
					InspectedSpans:  50,
				},
			},
			rInto: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					{
						Labels: []v1.KeyValue{
							{Key: "foo", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "bar"}}},
						},
						Samples: []tempopb.Sample{{TimestampMs: 1000, Value: 1.0}},
					},
				},
				Metrics: &tempopb.SearchMetrics{
					InspectedBytes:  50,
					CompletedJobs:   1,
					InspectedTraces: 5,
					InspectedSpans:  25,
				},
			},
			expected: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					{
						Labels: []v1.KeyValue{
							{Key: "foo", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "bar"}}},
						},
						Samples: []tempopb.Sample{
							{TimestampMs: 1000, Value: 1.0},
							{TimestampMs: 2000, Value: 2.0},
						},
					},
				},
				Metrics: &tempopb.SearchMetrics{
					InspectedBytes:  150,
					CompletedJobs:   2,
					InspectedTraces: 15,
					InspectedSpans:  75,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			naiveQueryRangeCombine(tt.rNew, tt.rInto)
			require.Equal(t, tt.expected.Metrics, tt.rInto.Metrics)
			require.Len(t, tt.rInto.Series, len(tt.expected.Series))

			// Build a map of expected series by label key for comparison
			expectedMap := map[string]*tempopb.TimeSeries{}
			for _, series := range tt.expected.Series {
				expectedMap[keyFromLabels(series.Labels)] = series
			}

			// Check that all series match expected (order doesn't matter due to map iteration)
			for _, series := range tt.rInto.Series {
				key := keyFromLabels(series.Labels)
				expectedSeries, ok := expectedMap[key]
				require.True(t, ok, "unexpected series with key %s", key)
				require.Equal(t, expectedSeries.Labels, series.Labels)
				require.Len(t, series.Samples, len(expectedSeries.Samples))
			}
		})
	}
}
