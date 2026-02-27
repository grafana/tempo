package combiner

import (
	"math"
	"math/rand/v2"
	"strconv"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/modules/frontend/shardtracker"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/stretchr/testify/require"
)

func TestAttachExemplars(t *testing.T) {
	start := uint64(10 * time.Second)
	end := uint64(20 * time.Second)
	step := traceql.DefaultQueryRangeStep(start, end)

	req := &tempopb.QueryRangeRequest{
		Start: start,
		End:   end,
		Step:  step,
	}

	tcs := []struct {
		name    string
		include func(i int) bool
	}{
		{
			name:    "include all",
			include: func(_ int) bool { return true },
		},
		{
			name:    "include none",
			include: func(_ int) bool { return false },
		},
		{
			name:    "include every other",
			include: func(i int) bool { return i%2 == 0 },
		},
		{
			name:    "include rando",
			include: func(_ int) bool { return rand.Int()%2 == 0 },
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			resp, expectedSeries := buildSeriesForExemplarTest(start, end, step, tc.include)

			attachExemplars(req, resp)
			require.Equal(t, expectedSeries, resp.Series)
		})
	}
}

func BenchmarkAttachExemplars(b *testing.B) {
	start := uint64(1 * time.Second)
	end := uint64(10000 * time.Second)
	step := uint64(time.Second)

	req := &tempopb.QueryRangeRequest{
		Start: start,
		End:   end,
		Step:  step,
	}

	resp, _ := buildSeriesForExemplarTest(start, end, step, func(_ int) bool { return true })

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		attachExemplars(req, resp)
	}
}

func buildSeriesForExemplarTest(start, end, step uint64, include func(i int) bool) (*tempopb.QueryRangeResponse, []*tempopb.TimeSeries) {
	resp := &tempopb.QueryRangeResponse{
		Series: []*tempopb.TimeSeries{
			{},
		},
	}

	expectedSeries := []*tempopb.TimeSeries{
		{},
	}

	// populate series and expected series based on step
	idx := 0
	for i := start; i < end; i += step {
		idx++
		tsMS := int64(i / uint64(time.Millisecond))
		val := float64(idx)

		sample := tempopb.Sample{
			TimestampMs: tsMS,
			Value:       val,
		}
		nanExemplar := tempopb.Exemplar{
			TimestampMs: tsMS,
			Value:       math.NaN(),
		}
		valExamplar := tempopb.Exemplar{
			TimestampMs: tsMS,
			Value:       val,
		}

		includeExemplar := include(idx)

		// copy the sample and nan exemplar into the response. the nan exemplar should be overwritten
		resp.Series[0].Samples = append(resp.Series[0].Samples, sample)
		if includeExemplar {
			resp.Series[0].Exemplars = append(resp.Series[0].Exemplars, nanExemplar)
		}

		// copy the sample and val exemplar into the expected response
		expectedSeries[0].Samples = append(expectedSeries[0].Samples, sample)
		if includeExemplar {
			expectedSeries[0].Exemplars = append(expectedSeries[0].Exemplars, valExamplar)
		}
	}

	return resp, expectedSeries
}

func TestQueryRangemaxSeriesShouldQuit(t *testing.T) {
	start := uint64(1100 * time.Second)
	end := uint64(1300 * time.Second)
	step := traceql.DefaultQueryRangeStep(start, end)
	bar := &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "bar"}}

	req := &tempopb.QueryRangeRequest{
		Query:     "{} | rate()",
		Start:     start,
		End:       end,
		Step:      step,
		MaxSeries: 4,
	}

	queryRangeCombiner, err := NewQueryRange(req, 4)
	require.NoError(t, err)

	// Add metadata that establishes shard information
	metadata := &QueryRangeJobResponse{
		JobMetadata: shardtracker.JobMetadata{
			TotalBlocks: 10,
			TotalJobs:   2,
			TotalBytes:  1000,
			Shards: []shardtracker.Shard{
				{
					TotalJobs:               2,
					CompletedThroughSeconds: 1200,
				},
			},
		},
	}

	err = queryRangeCombiner.AddResponse(metadata)
	require.NoError(t, err)

	// add 3 series, should not quit
	resp := &tempopb.QueryRangeResponse{
		Metrics: &tempopb.SearchMetrics{
			InspectedTraces: 1,
			InspectedBytes:  1,
		},
		Series: []*tempopb.TimeSeries{
			{
				Labels: []v1.KeyValue{
					{Key: "foo", Value: bar},
				},
				Samples: []tempopb.Sample{
					{
						TimestampMs: 1200_000,
						Value:       2,
					},
				},
			},
			{
				Labels: []v1.KeyValue{
					{Key: "boo", Value: bar},
				},
				Samples: []tempopb.Sample{
					{
						TimestampMs: 1200_000,
						Value:       2,
					},
				},
			},
			{
				Labels: []v1.KeyValue{
					{Key: "moo", Value: bar},
				},
				Samples: []tempopb.Sample{
					{
						TimestampMs: 1200_000,
						Value:       2,
					},
				},
			},
		},
	}

	err = queryRangeCombiner.AddResponse(toHTTPResponseWithFormat(t, resp, 200, 0, api.HeaderAcceptJSON))
	require.NoError(t, err)
	require.False(t, queryRangeCombiner.ShouldQuit())

	// add 4th & 5th series, should quit after shard completes
	secondResp := &tempopb.QueryRangeResponse{
		Metrics: &tempopb.SearchMetrics{
			InspectedTraces: 1,
			InspectedBytes:  1,
		},
		Series: []*tempopb.TimeSeries{
			{
				Labels: []v1.KeyValue{
					{Key: "woo", Value: bar},
				},
				Samples: []tempopb.Sample{
					{
						TimestampMs: 1200_000,
						Value:       2,
					},
				},
			},
			{
				Labels: []v1.KeyValue{
					{Key: "zoo", Value: bar},
				},
				Samples: []tempopb.Sample{
					{
						TimestampMs: 1200_000,
						Value:       2,
					},
				},
			},
		},
	}

	err = queryRangeCombiner.AddResponse(toHTTPResponseWithFormat(t, secondResp, 200, 0, api.HeaderAcceptJSON))
	require.NoError(t, err)
	require.True(t, queryRangeCombiner.ShouldQuit())
}

func TestQueryRangeMaxSeriesQuitRequiresCompletedShards(t *testing.T) {
	start := uint64(1100 * time.Second)
	end := uint64(1300 * time.Second)
	step := traceql.DefaultQueryRangeStep(start, end)
	bar := &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "bar"}}

	req := &tempopb.QueryRangeRequest{
		Query:     "{} | rate()",
		Start:     start,
		End:       end,
		Step:      step,
		MaxSeries: 2,
	}

	t.Run("max series reached but no shards completed should not quit", func(t *testing.T) {
		queryRangeCombiner, err := NewQueryRange(req, 2)
		require.NoError(t, err)

		// Add responses that exceed max series limit but without any metadata about completed shards
		// This simulates the case where we've received enough data to hit the limit
		// but haven't completed any shards yet
		resp := &tempopb.QueryRangeResponse{
			Metrics: &tempopb.SearchMetrics{
				InspectedTraces: 1,
				InspectedBytes:  1,
			},
			Series: []*tempopb.TimeSeries{
				{
					Labels: []v1.KeyValue{
						{Key: "series1", Value: bar},
					},
					Samples: []tempopb.Sample{
						{TimestampMs: 1200_000, Value: 1.0},
					},
				},
				{
					Labels: []v1.KeyValue{
						{Key: "series2", Value: bar},
					},
					Samples: []tempopb.Sample{
						{TimestampMs: 1200_000, Value: 2.0},
					},
				},
				{
					Labels: []v1.KeyValue{
						{Key: "series3", Value: bar},
					},
					Samples: []tempopb.Sample{
						{TimestampMs: 1200_000, Value: 3.0},
					},
				},
			},
		}

		err = queryRangeCombiner.AddResponse(toHTTPResponse(t, resp, 200))
		require.NoError(t, err)

		// Even though we've exceeded max series (3 > 2), we should NOT quit
		// because no shards have been marked as completed yet
		require.False(t, queryRangeCombiner.ShouldQuit())
	})

	t.Run("max series reached with completed shards should quit", func(t *testing.T) {
		queryRangeCombiner, err := NewQueryRange(req, 2)
		require.NoError(t, err)

		// First, add metadata that establishes shard information
		metadata := &QueryRangeJobResponse{
			JobMetadata: shardtracker.JobMetadata{
				TotalBlocks: 10,
				TotalJobs:   5,
				TotalBytes:  1000,
				Shards: []shardtracker.Shard{
					{
						TotalJobs:               2,
						CompletedThroughSeconds: 1200,
					},
					{
						TotalJobs:               3,
						CompletedThroughSeconds: 1250,
					},
				},
			},
		}

		err = queryRangeCombiner.AddResponse(metadata)
		require.NoError(t, err)

		// Add responses for the first shard with shard index to simulate completion
		resp1 := &tempopb.QueryRangeResponse{
			Metrics: &tempopb.SearchMetrics{
				InspectedTraces: 1,
				InspectedBytes:  1,
			},
			Series: []*tempopb.TimeSeries{
				{
					Labels: []v1.KeyValue{
						{Key: "series1", Value: bar},
					},
					Samples: []tempopb.Sample{
						{TimestampMs: 1200_000, Value: 1.0},
					},
				},
			},
		}

		// Add first response for shard 0
		err = queryRangeCombiner.AddResponse(toHTTPResponseWithFormat(t, resp1, 200, 0, api.HeaderAcceptJSON))
		require.NoError(t, err)
		require.False(t, queryRangeCombiner.ShouldQuit())

		// Add second response for shard 0 with more series, pushing us over the limit
		resp2 := &tempopb.QueryRangeResponse{
			Metrics: &tempopb.SearchMetrics{
				InspectedTraces: 1,
				InspectedBytes:  1,
			},
			Series: []*tempopb.TimeSeries{
				{
					Labels: []v1.KeyValue{
						{Key: "series2", Value: bar},
					},
					Samples: []tempopb.Sample{
						{TimestampMs: 1200_000, Value: 2.0},
					},
				},
				{
					Labels: []v1.KeyValue{
						{Key: "series3", Value: bar},
					},
					Samples: []tempopb.Sample{
						{TimestampMs: 1200_000, Value: 3.0},
					},
				},
			},
		}

		err = queryRangeCombiner.AddResponse(toHTTPResponseWithFormat(t, resp2, 200, 0, api.HeaderAcceptJSON))
		require.NoError(t, err)

		// Now we've exceeded max series (3 > 2) AND completed shard 0 (2 jobs received),
		// so we SHOULD quit
		require.True(t, queryRangeCombiner.ShouldQuit())
	})
}

func BenchmarkMarshalOnly(b *testing.B) {
	_, curr := seriesWithTenPercentDiff()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := proto.Marshal(curr)
		require.NoError(b, err)
	}
}

func seriesWithTenPercentDiff() (*tempopb.QueryRangeResponse, *tempopb.QueryRangeResponse) {
	a := &tempopb.QueryRangeResponse{}
	b := &tempopb.QueryRangeResponse{}

	numSeries := 1000
	numSamples := 1000

	for s := range numSeries {
		aSamples := make([]tempopb.Sample, numSamples)
		bSamples := make([]tempopb.Sample, numSamples)

		for i := range 1000 {
			aSamples[i] = tempopb.Sample{
				TimestampMs: int64(i) * 1000,
				Value:       rand.Float64(),
			}

			// 10% of samples are different
			if i%10 == 0 {
				bSamples[i] = tempopb.Sample{
					TimestampMs: int64(i) * 1000,
					Value:       rand.Float64(),
				}
			} else {
				bSamples[i] = aSamples[i]
			}
		}

		a.Series = append(a.Series, ts(aSamples, nil, "foo"+strconv.Itoa(s), "bar"))
		b.Series = append(b.Series, ts(bSamples, nil, "foo"+strconv.Itoa(s), "bar"))

	}

	return a, b
}

func ts(samples []tempopb.Sample, exemplars []tempopb.Exemplar, kvs ...string) *tempopb.TimeSeries {
	ts := &tempopb.TimeSeries{
		Samples:   samples,
		Exemplars: exemplars,
		Labels:    []v1.KeyValue{},
	}

	for i := 0; i < len(kvs); i += 2 {
		ts.Labels = append(ts.Labels, v1.KeyValue{
			Key: kvs[i],
			Value: &v1.AnyValue{
				Value: &v1.AnyValue_StringValue{
					StringValue: kvs[i+1],
				},
			},
		})
	}

	if samples == nil {
		ts.Samples = []tempopb.Sample{}
	}
	if exemplars == nil {
		ts.Exemplars = []tempopb.Exemplar{}
	}

	return ts
}

func TestTrimSeriesToCompletedWindow(t *testing.T) {
	tests := []struct {
		name                     string
		inputSamples             []tempopb.Sample
		inputExemplars           []tempopb.Exemplar
		lastCompletedThroughSecs uint32
		completedThroughSecs     uint32
		expectedSamples          []tempopb.Sample
		expectedExemplars        []tempopb.Exemplar
	}{
		{
			name: "basic window filtering",
			inputSamples: []tempopb.Sample{
				{TimestampMs: 5000, Value: 1.0},
				{TimestampMs: 10000, Value: 2.0},
				{TimestampMs: 15000, Value: 3.0},
				{TimestampMs: 20000, Value: 4.0},
				{TimestampMs: 25000, Value: 5.0},
			},
			inputExemplars: []tempopb.Exemplar{
				{TimestampMs: 5000, Value: 1.0},
				{TimestampMs: 15000, Value: 3.0},
				{TimestampMs: 25000, Value: 5.0},
			},
			lastCompletedThroughSecs: 20, // 20000ms
			completedThroughSecs:     10, // 10000ms
			expectedSamples: []tempopb.Sample{
				{TimestampMs: 15000, Value: 3.0},
				{TimestampMs: 20000, Value: 4.0}, // completedThrough is inclusive
			},
			expectedExemplars: []tempopb.Exemplar{
				{TimestampMs: 15000, Value: 3.0},
			},
		},
		{
			name: "no samples in window",
			inputSamples: []tempopb.Sample{
				{TimestampMs: 5000, Value: 1.0},
				{TimestampMs: 10000, Value: 2.0},
				{TimestampMs: 30000, Value: 4.0},
			},
			inputExemplars:           []tempopb.Exemplar{},
			lastCompletedThroughSecs: 20,
			completedThroughSecs:     10,
			expectedSamples:          []tempopb.Sample{},
			expectedExemplars:        []tempopb.Exemplar{},
		},
		{
			name: "all samples in window",
			inputSamples: []tempopb.Sample{
				{TimestampMs: 11000, Value: 1.0},
				{TimestampMs: 15000, Value: 2.0},
				{TimestampMs: 19000, Value: 3.0},
			},
			inputExemplars: []tempopb.Exemplar{
				{TimestampMs: 12000, Value: 1.0},
				{TimestampMs: 18000, Value: 2.0},
			},
			lastCompletedThroughSecs: 20,
			completedThroughSecs:     10,
			expectedSamples: []tempopb.Sample{
				{TimestampMs: 11000, Value: 1.0},
				{TimestampMs: 15000, Value: 2.0},
				{TimestampMs: 19000, Value: 3.0},
			},
			expectedExemplars: []tempopb.Exemplar{
				{TimestampMs: 12000, Value: 1.0},
				{TimestampMs: 18000, Value: 2.0},
			},
		},
		{
			name:                     "empty series",
			inputSamples:             []tempopb.Sample{},
			inputExemplars:           []tempopb.Exemplar{},
			lastCompletedThroughSecs: 20,
			completedThroughSecs:     10,
			expectedSamples:          []tempopb.Sample{},
			expectedExemplars:        []tempopb.Exemplar{},
		},
		{
			name: "boundary conditions - samples at exact timestamps",
			inputSamples: []tempopb.Sample{
				{TimestampMs: 10000, Value: 1.0}, // exactly at lastCompletedThrough (exclusive)
				{TimestampMs: 15000, Value: 2.0},
				{TimestampMs: 20000, Value: 3.0}, // exactly at completedThrough (inclusive)
				{TimestampMs: 25000, Value: 4.0},
			},
			inputExemplars:           []tempopb.Exemplar{},
			lastCompletedThroughSecs: 20,
			completedThroughSecs:     10,
			expectedSamples: []tempopb.Sample{
				{TimestampMs: 15000, Value: 2.0},
				{TimestampMs: 20000, Value: 3.0}, // completedThrough is inclusive
			},
			expectedExemplars: []tempopb.Exemplar{},
		},
		{
			name: "zero timestamps",
			inputSamples: []tempopb.Sample{
				{TimestampMs: 0, Value: 1.0},
				{TimestampMs: 5000, Value: 2.0},
				{TimestampMs: 10000, Value: 3.0},
			},
			inputExemplars:           []tempopb.Exemplar{},
			lastCompletedThroughSecs: 8,
			completedThroughSecs:     0,
			expectedSamples: []tempopb.Sample{
				{TimestampMs: 5000, Value: 2.0},
			},
			expectedExemplars: []tempopb.Exemplar{},
		},
		{
			name: "multiple series",
			inputSamples: []tempopb.Sample{
				{TimestampMs: 5000, Value: 1.0},
				{TimestampMs: 15000, Value: 2.0},
				{TimestampMs: 25000, Value: 3.0},
			},
			inputExemplars: []tempopb.Exemplar{
				{TimestampMs: 5000, Value: 1.0},
				{TimestampMs: 15000, Value: 2.0},
				{TimestampMs: 25000, Value: 3.0},
			},
			lastCompletedThroughSecs: 20,
			completedThroughSecs:     10,
			expectedSamples: []tempopb.Sample{
				{TimestampMs: 15000, Value: 2.0},
			},
			expectedExemplars: []tempopb.Exemplar{
				{TimestampMs: 15000, Value: 2.0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			series := []*tempopb.TimeSeries{
				ts(tt.inputSamples, tt.inputExemplars, "test", "value"),
			}

			trimSeriesToCompletedWindow(series, tt.lastCompletedThroughSecs, tt.completedThroughSecs)

			require.Equal(t, tt.expectedSamples, series[0].Samples, "samples mismatch")
			require.Equal(t, tt.expectedExemplars, series[0].Exemplars, "exemplars mismatch")
		})
	}
}

func TestTrimSeriesToCompletedWindow_MultipleSeries(t *testing.T) {
	series := []*tempopb.TimeSeries{
		ts([]tempopb.Sample{
			{TimestampMs: 5000, Value: 1.0},
			{TimestampMs: 15000, Value: 2.0},
			{TimestampMs: 25000, Value: 3.0},
		}, []tempopb.Exemplar{
			{TimestampMs: 15000, Value: 2.0},
		}, "series", "one"),
		ts([]tempopb.Sample{
			{TimestampMs: 8000, Value: 4.0},
			{TimestampMs: 12000, Value: 5.0},
			{TimestampMs: 18000, Value: 6.0},
		}, []tempopb.Exemplar{
			{TimestampMs: 12000, Value: 5.0},
			{TimestampMs: 18000, Value: 6.0},
		}, "series", "two"),
	}

	trimSeriesToCompletedWindow(series, 20, 10)

	// First series should have only the sample at 15000
	require.Equal(t, []tempopb.Sample{
		{TimestampMs: 15000, Value: 2.0},
	}, series[0].Samples)
	require.Equal(t, []tempopb.Exemplar{
		{TimestampMs: 15000, Value: 2.0},
	}, series[0].Exemplars)

	// Second series should have samples at 12000 and 18000
	require.Equal(t, []tempopb.Sample{
		{TimestampMs: 12000, Value: 5.0},
		{TimestampMs: 18000, Value: 6.0},
	}, series[1].Samples)
	require.Equal(t, []tempopb.Exemplar{
		{TimestampMs: 12000, Value: 5.0},
		{TimestampMs: 18000, Value: 6.0},
	}, series[1].Exemplars)
}

func TestQueryRangeResponseWithNaN(t *testing.T) {
	// Test that NaN values are correctly marshaled to JSON
	start := uint64(1100 * time.Second)
	end := uint64(1300 * time.Second)
	step := traceql.DefaultQueryRangeStep(start, end)

	req := &tempopb.QueryRangeRequest{
		Query: "{} | rate()",
		Start: start,
		End:   end,
		Step:  step,
	}

	queryRangeCombiner, err := NewQueryRange(req, 100)
	require.NoError(t, err)

	// Add metadata
	metadata := &QueryRangeJobResponse{
		JobMetadata: shardtracker.JobMetadata{
			TotalBlocks: 1,
			TotalJobs:   1,
			TotalBytes:  100,
			Shards: []shardtracker.Shard{
				{
					TotalJobs:               1,
					CompletedThroughSeconds: 1200,
				},
			},
		},
	}
	err = queryRangeCombiner.AddResponse(metadata)
	require.NoError(t, err)

	// Add a response with NaN values
	bar := &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "bar"}}
	resp := &tempopb.QueryRangeResponse{
		Metrics: &tempopb.SearchMetrics{
			InspectedTraces: 1,
			InspectedBytes:  1,
		},
		Series: []*tempopb.TimeSeries{
			{
				Labels: []v1.KeyValue{
					{Key: "foo", Value: bar},
				},
				Samples: []tempopb.Sample{
					{TimestampMs: 1110_000, Value: 1.0},
					{TimestampMs: 1120_000, Value: math.NaN()},
					{TimestampMs: 1130_000, Value: 3.0},
				},
			},
		},
	}

	err = queryRangeCombiner.AddResponse(toHTTPResponseWithFormat(t, resp, 200, 0, api.HeaderAcceptJSON))
	require.NoError(t, err)

	// Get the HTTP response (which marshals to JSON)
	httpResp, err := queryRangeCombiner.HTTPFinal()
	require.NoError(t, err)
	require.Equal(t, 200, httpResp.StatusCode)

	// Read and verify the body contains "NaN" as a string
	body, err := api.ReadBodyToBuffer(httpResp)
	require.NoError(t, err)
	bodyStr := body.String()

	// The jsonpb marshaler should output NaN as "NaN" (quoted string)
	require.Contains(t, bodyStr, `"NaN"`, "Response should contain NaN as a string")
	require.Contains(t, bodyStr, `"value":1`, "Response should contain normal value 1")
	require.Contains(t, bodyStr, `"value":3`, "Response should contain normal value 3")
}
