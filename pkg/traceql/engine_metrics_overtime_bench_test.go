package traceql

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	commonv1proto "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	"github.com/stretchr/testify/require"
)

// BenchmarkMetricsFrontendOverTime measures the frontend combining job
// results for the over_time functions: SimpleAggregator for sum/min/max and
// the average series aggregator for avg.
func BenchmarkMetricsFrontendOverTime(b *testing.B) {
	start := uint64(time.Now().Add(-1 * time.Hour).UnixNano())
	end := uint64(time.Now().UnixNano())
	step := uint64(15 * time.Second.Nanoseconds())

	const (
		jobs             = 20
		samplesPerSeries = 240
	)

	for _, fn := range []string{"sum", "min", "max", "avg"} {
		query := fmt.Sprintf("{ } | %s_over_time(duration) by (span.foo)", fn)
		req := &tempopb.QueryRangeRequest{Start: start, End: end, Step: step, Query: query}

		for _, seriesCount := range []int{100, 1000} {
			data := make([][]*tempopb.TimeSeries, jobs)
			for i := range data {
				data[i] = overTimeJobSeries(rand.New(rand.NewSource(int64(i))), seriesCount, samplesPerSeries, start, end, fn == "avg")
			}

			b.Run(fmt.Sprintf("%s/series_%d", fn, seriesCount), func(b *testing.B) {
				for b.Loop() {
					eval, err := NewEngine().CompileMetricsQueryRangeNonRaw(req, AggregateModeFinal)
					require.NoError(b, err)
					for _, job := range data {
						eval.ObserveSeries(job)
					}
					_ = eval.Results()
				}
			})
		}
	}
}

// overTimeJobSeries is one job's result: a series per span.foo value with a
// sample per interval, plus the count series avg sends alongside its means.
func overTimeJobSeries(r *rand.Rand, seriesCount, samplesCount int, start, end uint64, withCounts bool) []*tempopb.TimeSeries {
	out := make([]*tempopb.TimeSeries, 0, seriesCount*2)
	for i := range seriesCount {
		labels := []commonv1proto.KeyValue{{
			Key:   "span.foo",
			Value: &commonv1proto.AnyValue{Value: &commonv1proto.AnyValue_StringValue{StringValue: fmt.Sprintf("value-%d", i)}},
		}}
		values := make([]tempopb.Sample, samplesCount)
		counts := make([]tempopb.Sample, samplesCount)
		for j := range values {
			ts := time.Unix(0, int64(start+uint64(j)*(end-start)/uint64(samplesCount))).UnixMilli() // nolint: gosec // G115
			values[j] = tempopb.Sample{TimestampMs: ts, Value: r.ExpFloat64()}
			counts[j] = tempopb.Sample{TimestampMs: ts, Value: float64(1 + r.Intn(100))}
		}
		out = append(out, &tempopb.TimeSeries{Labels: labels, Samples: values})
		if withCounts {
			countLabels := append([]commonv1proto.KeyValue{{
				Key:   internalLabelMetaType,
				Value: &commonv1proto.AnyValue{Value: &commonv1proto.AnyValue_StringValue{StringValue: internalMetaTypeCount}},
			}}, labels...)
			out = append(out, &tempopb.TimeSeries{Labels: countLabels, Samples: counts})
		}
	}
	return out
}
