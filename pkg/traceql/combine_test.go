package traceql

import (
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
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
		{
			name: "merge ServiceStats",
			existing: &tempopb.TraceSearchMetadata{
				ServiceStats: map[string]*tempopb.ServiceStats{
					"service1": {
						SpanCount:  5,
						ErrorCount: 1,
					},
				},
			},
			new: &tempopb.TraceSearchMetadata{
				ServiceStats: map[string]*tempopb.ServiceStats{
					"service1": {
						SpanCount:  3,
						ErrorCount: 2,
					},
				},
			},
			expected: &tempopb.TraceSearchMetadata{
				ServiceStats: map[string]*tempopb.ServiceStats{
					"service1": {
						SpanCount:  5,
						ErrorCount: 2,
					},
				},
			},
		},
		{
			name:     "existing ServiceStats is nil doesn't panic",
			existing: &tempopb.TraceSearchMetadata{},
			new: &tempopb.TraceSearchMetadata{
				ServiceStats: map[string]*tempopb.ServiceStats{
					"service1": {
						SpanCount:  3,
						ErrorCount: 2,
					},
				},
			},
			expected: &tempopb.TraceSearchMetadata{
				ServiceStats: map[string]*tempopb.ServiceStats{
					"service1": {
						SpanCount:  3,
						ErrorCount: 2,
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

func TestQueryRangeCombinerFinal(t *testing.T) {
	tcs := []struct {
		name  string
		start uint64
		end   uint64
		step  uint64
		resps []struct {
			resp             *tempopb.QueryRangeResponse
			expectedResponse *tempopb.QueryRangeResponse
			expectedDiff     *tempopb.QueryRangeResponse
		}
	}{
		{
			name:  "simple",
			start: uint64(100 * time.Millisecond),
			end:   uint64(150 * time.Millisecond),
			step:  uint64(10 * time.Millisecond),
			resps: []struct{ resp, expectedResponse, expectedDiff *tempopb.QueryRangeResponse }{
				// push nothing get nothing
				{
					resp: &tempopb.QueryRangeResponse{},
					expectedResponse: &tempopb.QueryRangeResponse{
						Series: []*tempopb.TimeSeries{},
					},
				},
				// push 3 data points, get them back
				{
					resp: &tempopb.QueryRangeResponse{
						Series: []*tempopb.TimeSeries{
							timeSeries("foo", "1", []tempopb.Sample{{100, 1}, {110, 2}, {120, 3}}),
						},
					},
					expectedResponse: &tempopb.QueryRangeResponse{
						Series: []*tempopb.TimeSeries{
							timeSeries("foo", "1", []tempopb.Sample{{100, 1}, {110, 2}, {120, 3}, {130, 0}, {140, 0}, {150, 0}}),
						},
					},
					expectedDiff: &tempopb.QueryRangeResponse{
						Series: []*tempopb.TimeSeries{
							timeSeries("foo", "1", []tempopb.Sample{{100, 1}, {110, 2}, {120, 3}}),
						},
					},
				},
				// push 2 data points, check aggregation
				{
					resp: &tempopb.QueryRangeResponse{
						Series: []*tempopb.TimeSeries{
							timeSeries("foo", "1", []tempopb.Sample{{120, 1}, {130, 2}, {150, 3}}),
						},
					},
					expectedResponse: &tempopb.QueryRangeResponse{
						Series: []*tempopb.TimeSeries{
							timeSeries("foo", "1", []tempopb.Sample{{100, 1}, {110, 2}, {120, 4}, {130, 2}, {140, 0}, {150, 3}}),
						},
					},
				},
				// push different series
				{
					resp: &tempopb.QueryRangeResponse{
						Series: []*tempopb.TimeSeries{
							timeSeries("bar", "1", []tempopb.Sample{{100, 1}, {110, 2}, {120, 3}}),
						},
					},
					expectedResponse: &tempopb.QueryRangeResponse{
						Series: []*tempopb.TimeSeries{
							timeSeries("foo", "1", []tempopb.Sample{{100, 1}, {110, 2}, {120, 4}, {130, 2}, {140, 0}, {150, 3}}),
							timeSeries("bar", "1", []tempopb.Sample{{100, 1}, {110, 2}, {120, 3}, {130, 0}, {140, 0}, {150, 0}}),
						},
					},
					// includes last 2 pushes
					expectedDiff: &tempopb.QueryRangeResponse{
						Series: []*tempopb.TimeSeries{
							timeSeries("foo", "1", []tempopb.Sample{{120, 4}, {130, 2}, {140, 0}, {150, 3}}),
							timeSeries("bar", "1", []tempopb.Sample{{100, 1}, {110, 2}, {120, 3}}),
						},
					},
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			req := &tempopb.QueryRangeRequest{
				Start: tc.start,
				End:   tc.end,
				Step:  tc.step,
				Query: "{} | rate()", // simple aggregate
			}
			combiner, err := QueryRangeCombinerFor(req, AggregateModeFinal, true)
			require.NoError(t, err)

			for _, tcResp := range tc.resps {
				combiner.Combine(tcResp.resp)

				resp := combiner.Response()
				resp.Metrics = nil // we want to ignore metrics for this test, just nil them out
				require.Equal(t, tcResp.expectedResponse, resp)

				if tcResp.expectedDiff != nil {
					// call diff and get expected
					diff := combiner.Diff()
					diff.Metrics = nil
					require.Equal(t, tcResp.expectedDiff, diff)

					// call diff again and get nothing!
					diff = combiner.Diff()
					diff.Metrics = nil
					require.Equal(t, &tempopb.QueryRangeResponse{
						Series: []*tempopb.TimeSeries{},
					}, diff)
				}
			}
		})
	}
}

func timeSeries(name, val string, samples []tempopb.Sample) *tempopb.TimeSeries {
	lbls := Labels{
		{
			Name:  name,
			Value: NewStaticString(val),
		},
	}

	return &tempopb.TimeSeries{
		Labels:     []v1.KeyValue{{Key: name, Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: val}}}},
		Samples:    samples,
		PromLabels: lbls.String(),
	}
}
