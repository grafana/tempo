package combiner

import (
	"math"
	"math/rand/v2"
	"strconv"
	"testing"
	"time"

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

// nolint:govet # govet doesn't like the structs with unkeyed fields, but it's much cleaner in this test
func TestDiffSeries(t *testing.T) {
	tcs := []struct {
		name     string
		prev     *tempopb.QueryRangeResponse
		curr     *tempopb.QueryRangeResponse
		expected *tempopb.QueryRangeResponse
	}{
		{
			name: "copy through curr if no prev",
			prev: nil,
			curr: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{1000, 1.0},
						{2000, 3.0},
						{3000, 4.0},
					}, "foo", "bar"),
				},
			},
			expected: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{1000, 1.0},
						{2000, 3.0},
						{3000, 4.0},
					}, "foo", "bar"),
				},
			},
		},
		{
			name: "prev == curr so actual is empty",
			prev: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{1000, 1.0},
						{2000, 3.0},
						{3000, 4.0},
					}, "foo", "bar"),
				},
			},
			curr: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{1000, 1.0},
						{2000, 3.0},
						{3000, 4.0},
					}, "foo", "bar"),
				},
			},
			expected: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{},
			},
		},
		{
			name: "add one series after",
			prev: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{1000, 1.0},
						{2000, 3.0},
						{3000, 4.0},
					}, "foo", "bar"),
				},
			},
			curr: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{1000, 1.0},
						{2000, 3.0},
						{3000, 4.0},
					}, "foo", "bar"),
					ts([]tempopb.Sample{
						{1500, 1.5},
						{2500, 3.5},
						{3500, 4.5},
					}, "baz", "bat"),
				},
			},
			expected: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{1500, 1.5},
						{2500, 3.5},
						{3500, 4.5},
					}, "baz", "bat"),
				},
			},
		},
		{
			name: "add one series before",
			prev: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{1000, 1.0},
						{2000, 3.0},
						{3000, 4.0},
					}, "foo", "bar"),
				},
			},
			curr: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{1500, 1.5},
						{2500, 3.5},
						{3500, 4.5},
					}, "baz", "bat"),
					ts([]tempopb.Sample{
						{1000, 1.0},
						{2000, 3.0},
						{3000, 4.0},
					}, "foo", "bar"),
				},
			},
			expected: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{1500, 1.5},
						{2500, 3.5},
						{3500, 4.5},
					}, "baz", "bat"),
				},
			},
		},
		{
			name: "add samples",
			prev: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{1000, 1.0},
						{2000, 3.0},
						{3000, 4.0},
					}, "foo", "bar"),
				},
			},
			curr: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{500, .5},
						{1000, 1.0},
						{1500, 1.5},
						{2000, 3.0},
						{2500, 2.5},
						{3000, 4.0},
						{3500, 3.5},
					}, "foo", "bar"),
				},
			},
			expected: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{500, .5},
						{1500, 1.5},
						{2500, 2.5},
						{3500, 3.5},
					}, "foo", "bar"),
				},
			},
		},
		{
			name: "modify samples",
			prev: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{1000, 1.0},
						{2000, 3.0},
						{3000, 4.0},
					}, "foo", "bar"),
				},
			},
			curr: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{1000, 1.5},
						{2000, 3.5},
						{3000, 4.5},
					}, "foo", "bar"),
				},
			},
			expected: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{1000, 1.5},
						{2000, 3.5},
						{3000, 4.5},
					}, "foo", "bar"),
				},
			},
		},
		{
			name: "all things",
			prev: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{1000, 1.0},
						{2000, 3.0},
						{3000, 4.0},
					}, "foo", "bar"),
					ts([]tempopb.Sample{
						{1000, 1.0},
						{2000, 3.0},
						{3000, 4.0},
					}, "baz", "bat"),
				},
			},
			curr: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{ // add one before
						{1000, 1.1},
						{2000, 3.1},
						{3000, 4.2},
					}, "pre", "pre"),
					ts([]tempopb.Sample{ // samples added and modified
						{500, 0.5},
						{1000, 1.0},
						{2000, 3.5},
						{3000, 4.0},
						{3500, 3.5},
					}, "foo", "bar"),
					ts([]tempopb.Sample{ // add one between
						{1000, 1.1},
						{2000, 3.1},
						{3000, 4.1},
					}, "inter", "inter"),
					ts([]tempopb.Sample{ // no change! should not appear in output
						{1000, 1.0},
						{2000, 3.0},
						{3000, 4.0},
					}, "baz", "bat"),
					ts([]tempopb.Sample{ // add one after
						{1000, 1.1},
						{2000, 3.1},
						{3000, 4.1},
					}, "post", "post"),
				},
			},
			expected: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{ // add one before
						{1000, 1.1},
						{2000, 3.1},
						{3000, 4.2},
					}, "pre", "pre"),
					ts([]tempopb.Sample{ // samples added and modified
						{500, 0.5},
						{2000, 3.5},
						{3500, 3.5},
					}, "foo", "bar"),
					ts([]tempopb.Sample{ // add one between
						{1000, 1.1},
						{2000, 3.1},
						{3000, 4.1},
					}, "inter", "inter"),
					ts([]tempopb.Sample{ // add one after
						{1000, 1.1},
						{2000, 3.1},
						{3000, 4.1},
					}, "post", "post"),
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			actual := diffResponse(tc.prev, tc.curr)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func BenchmarkDiffSeriesAllSamplesSeriesEqual(b *testing.B) {
	prev := &tempopb.QueryRangeResponse{}
	curr := &tempopb.QueryRangeResponse{}

	numSeries := 1000
	numSamples := 1000

	for s := range numSeries {
		samples := make([]tempopb.Sample, numSamples)
		for i := range 1000 {
			samples[i] = tempopb.Sample{
				TimestampMs: int64(i) * 1000,
				Value:       rand.Float64(),
			}
		}

		series := ts(samples, "foo"+strconv.Itoa(s), "bar")
		prev.Series = append(prev.Series, series)
		curr.Series = append(curr.Series, series)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		diffResponse(prev, curr)
	}
}

func BenchmarkDiffSeriesAllSamplesSeriesRandom(b *testing.B) {
	prev := &tempopb.QueryRangeResponse{}
	curr := &tempopb.QueryRangeResponse{}

	numSeries := 1000
	numSamples := 1000

	for s := range numSeries {
		samples := make([]tempopb.Sample, numSamples)
		for i := range 1000 {
			samples[i] = tempopb.Sample{
				TimestampMs: int64(i) * 1000,
				Value:       rand.Float64(),
			}
		}

		prev.Series = append(prev.Series, ts(samples, "foo"+strconv.Itoa(s), "bar"))

		// create a new slice with different sample values
		samples = make([]tempopb.Sample, numSamples)
		for i := range 1000 {
			samples[i] = tempopb.Sample{
				TimestampMs: int64(i) * 1000,
				Value:       rand.Float64(),
			}
		}
		curr.Series = append(curr.Series, ts(samples, "foo"+strconv.Itoa(s), "bar"))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		diffResponse(prev, curr)
	}
}

func ts(samples []tempopb.Sample, kvs ...string) *tempopb.TimeSeries {
	ts := &tempopb.TimeSeries{
		Samples: samples,
		Labels:  []v1.KeyValue{},
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
	ts.PromLabels = traceql.LabelsFromProto(ts.Labels).String()
	ts.Samples = samples

	return ts
}
