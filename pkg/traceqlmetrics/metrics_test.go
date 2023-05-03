package traceqlmetrics

import (
	"context"
	"testing"

	"github.com/grafana/tempo/pkg/traceql"
	"github.com/stretchr/testify/require"
)

func TestPercentile(t *testing.T) {

	testCases := []struct {
		name      string
		durations []uint64
		quartile  float32
		value     uint64
	}{
		{
			name:      "easy mode",
			durations: []uint64{2, 4, 6, 8},
			quartile:  0.5,
			value:     uint64(4),
		},
		{
			// 10 samples
			// p75 rounds means 7.5 samples, rounds up to 8
			// 5 samples from the 2048 bucket
			// 3 samples from the 4096 bucket
			// interpolation: 3/5ths from 2048 to 4096 exponentially
			// = 2048 * 2^0.6 = 3104.1875...
			name:      "interpolate between buckets",
			durations: []uint64{2000, 2000, 2000, 2000, 2000, 4000, 4000, 4000, 4000, 4000},
			quartile:  0.75,
			value:     uint64(3104),
		},
	}

	for _, tc := range testCases {
		m := &latencyHistogram{}
		for _, d := range tc.durations {
			m.Record(d)
		}
		got := m.Percentile(tc.quartile)
		require.Equal(t, tc.value, got, tc.name)
	}
}

func TestGetMetrics(t *testing.T) {

	ctx := context.TODO()
	query := "{}"
	groupBy := "span.foo"

	m := &mockFetcher{
		Spansets: []*traceql.Spanset{
			{
				Spans: []traceql.Span{
					newMockSpan(128, "span.foo", "1"),
					newMockSpan(128, "span.foo", "1"), // p50 for foo=1
					newMockSpan(256, "span.foo", "1"),
					newMockSpan(256, "span.foo", "1"),
					newMockSpan(256, "span.foo", "2"),
					newMockSpan(256, "span.foo", "2"), // p50 for foo=2
					newMockSpan(512, "span.foo", "2"),
					newMockSpan(512, "span.foo", "2").WithErr(),
				},
			},
		},
	}

	res, err := GetMetrics(ctx, query, groupBy, 1000, m)
	require.NoError(t, err)
	require.NotNil(t, res)

	one := traceql.NewStaticString("1")
	two := traceql.NewStaticString("2")

	require.Equal(t, 0, res.Errors[one])
	require.Equal(t, 1, res.Errors[two])

	require.NotNil(t, res.Series[one])
	require.NotNil(t, res.Series[two])

	require.Equal(t, uint64(128), res.Series[one].Percentile(0.5))  // p50
	require.Equal(t, uint64(181), res.Series[one].Percentile(0.75)) // p75, 128 * 2^0.5 = 181
	require.Equal(t, uint64(256), res.Series[one].Percentile(1.0))  // p100

	require.Equal(t, uint64(256), res.Series[two].Percentile(0.5))  // p50
	require.Equal(t, uint64(362), res.Series[two].Percentile(0.75)) // p75, 256 * 2^0.5 = 362
	require.Equal(t, uint64(512), res.Series[two].Percentile(1.0))  // p100
}
