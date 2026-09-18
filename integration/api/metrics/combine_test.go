package metrics

import (
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/stretchr/testify/require"
)

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
