package combiner

import (
	"math"
	"math/rand/v2"
	"strconv"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
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

// TestDifSeries has govet disabled b/c the linter doesn't like the unkeyed structs, but the test reads a lot cleaner without them.
// nolint:govet
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
					}, nil, "foo", "bar"),
				},
			},
			expected: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{1000, 1.0},
						{2000, 3.0},
						{3000, 4.0},
					}, nil, "foo", "bar"),
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
					}, nil, "foo", "bar"),
				},
			},
			curr: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{1000, 1.0},
						{2000, 3.0},
						{3000, 4.0},
					}, nil, "foo", "bar"),
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
					}, nil, "foo", "bar"),
				},
			},
			curr: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{1000, 1.0},
						{2000, 3.0},
						{3000, 4.0},
					}, nil, "foo", "bar"),
					ts([]tempopb.Sample{
						{1500, 1.5},
						{2500, 3.5},
						{3500, 4.5},
					}, nil, "baz", "bat"),
				},
			},
			expected: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{1500, 1.5},
						{2500, 3.5},
						{3500, 4.5},
					}, nil, "baz", "bat"),
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
					}, nil, "foo", "bar"),
				},
			},
			curr: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{1500, 1.5},
						{2500, 3.5},
						{3500, 4.5},
					}, nil, "baz", "bat"),
					ts([]tempopb.Sample{
						{1000, 1.0},
						{2000, 3.0},
						{3000, 4.0},
					}, nil, "foo", "bar"),
				},
			},
			expected: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{1500, 1.5},
						{2500, 3.5},
						{3500, 4.5},
					}, nil, "baz", "bat"),
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
					}, nil, "foo", "bar"),
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
					}, nil, "foo", "bar"),
				},
			},
			expected: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{500, .5},
						{1500, 1.5},
						{2500, 2.5},
						{3500, 3.5},
					}, nil, "foo", "bar"),
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
					}, nil, "foo", "bar"),
				},
			},
			curr: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{1000, 1.5},
						{2000, 3.5},
						{3000, 4.5},
					}, nil, "foo", "bar"),
				},
			},
			expected: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{
						{1000, 1.5},
						{2000, 3.5},
						{3000, 4.5},
					}, nil, "foo", "bar"),
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
					}, nil, "foo", "bar"),
					ts([]tempopb.Sample{
						{1000, 1.0},
						{2000, 3.0},
						{3000, 4.0},
					}, nil, "baz", "bat"),
				},
			},
			curr: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{ // add one before
						{1000, 1.1},
						{2000, 3.1},
						{3000, 4.2},
					}, nil, "pre", "pre"),
					ts([]tempopb.Sample{ // samples added and modified
						{500, 0.5},
						{1000, 1.0},
						{2000, 3.5},
						{3000, 4.0},
						{3500, 3.5},
					}, nil, "foo", "bar"),
					ts([]tempopb.Sample{ // add one between
						{1000, 1.1},
						{2000, 3.1},
						{3000, 4.1},
					}, nil, "inter", "inter"),
					ts([]tempopb.Sample{ // no change! should not appear in output
						{1000, 1.0},
						{2000, 3.0},
						{3000, 4.0},
					}, nil, "baz", "bat"),
					ts([]tempopb.Sample{ // add one after
						{1000, 1.1},
						{2000, 3.1},
						{3000, 4.1},
					}, nil, "post", "post"),
				},
			},
			expected: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts([]tempopb.Sample{ // add one before
						{1000, 1.1},
						{2000, 3.1},
						{3000, 4.2},
					}, nil, "pre", "pre"),
					ts([]tempopb.Sample{ // samples added and modified
						{500, 0.5},
						{2000, 3.5},
						{3500, 3.5},
					}, nil, "foo", "bar"),
					ts([]tempopb.Sample{ // add one between
						{1000, 1.1},
						{2000, 3.1},
						{3000, 4.1},
					}, nil, "inter", "inter"),
					ts([]tempopb.Sample{ // add one after
						{1000, 1.1},
						{2000, 3.1},
						{3000, 4.1},
					}, nil, "post", "post"),
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

func TestDiffExemplars(t *testing.T) {
	tcs := []struct {
		name     string
		prev     *tempopb.QueryRangeResponse
		curr     *tempopb.QueryRangeResponse
		expected *tempopb.QueryRangeResponse
	}{
		{
			name: "exemplar no change",
			prev: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts(nil, []tempopb.Exemplar{
						{TimestampMs: 1000, Value: 1.0},
						{TimestampMs: 2000, Value: 2.0},
					}, "foo", "bar"),
				},
			},
			curr: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts(nil, []tempopb.Exemplar{
						{TimestampMs: 1000, Value: 1.0},
						{TimestampMs: 2000, Value: 2.0},
					}, "foo", "bar"),
				},
			},
			expected: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{},
			},
		},
		{
			name: "exemplar value changed",
			prev: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts(nil, []tempopb.Exemplar{
						{TimestampMs: 1000, Value: 1.0},
					}, "foo", "bar"),
				},
			},
			curr: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts(nil, []tempopb.Exemplar{
						{TimestampMs: 1000, Value: 2.0},
					}, "foo", "bar"),
				},
			},
			expected: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts(nil, []tempopb.Exemplar{
						{TimestampMs: 1000, Value: 2.0},
					}, "foo", "bar"),
				},
			},
		},
		{
			name: "exemplar added",
			prev: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts(nil, nil, "foo", "bar"),
				},
			},
			curr: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts(nil, []tempopb.Exemplar{
						{TimestampMs: 1000, Value: 1.0},
					}, "foo", "bar"),
				},
			},
			expected: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts(nil, []tempopb.Exemplar{
						{TimestampMs: 1000, Value: 1.0},
					}, "foo", "bar"),
				},
			},
		},
		{
			name: "several exemplars changes",
			prev: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts(nil, []tempopb.Exemplar{
						{TimestampMs: 1000, Value: 1.0},
						{TimestampMs: 2000, Value: 2.0},
					}, "foo", "bar"),
				},
			},
			curr: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts(nil, []tempopb.Exemplar{
						{TimestampMs: 500, Value: .5},   // add before
						{TimestampMs: 1000, Value: 1.0}, // same
						{TimestampMs: 1500, Value: 1.5}, // add between
						{TimestampMs: 2000, Value: 2.1}, // modified
						{TimestampMs: 2500, Value: 2.5}, // add after
					}, "foo", "bar"),
				},
			},
			expected: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{
					ts(nil, []tempopb.Exemplar{
						{TimestampMs: 500, Value: .5},   // add before
						{TimestampMs: 1500, Value: 1.5}, // add between
						{TimestampMs: 2000, Value: 2.1}, // modified
						{TimestampMs: 2500, Value: 2.5}, // add after
					}, "foo", "bar"),
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

func BenchmarkDiffSeriesAndMarshal(b *testing.B) {
	prev, curr := seriesWithTenPercentDiff()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		diff := diffResponse(prev, curr)
		_, err := proto.Marshal(diff)
		require.NoError(b, err)
	}
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
	ts.PromLabels = traceql.LabelsFromProto(ts.Labels).String()
	if samples == nil {
		ts.Samples = []tempopb.Sample{}
	}
	if exemplars == nil {
		ts.Exemplars = []tempopb.Exemplar{}
	}

	return ts
}
