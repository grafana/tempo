package traceql

import (
	"fmt"
	"math"
	"sort"
	"testing"
	"time"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	"github.com/stretchr/testify/require"
)

func histogramAlgorithmInput(buckets, quantiles int, exemplars uint32, exemplarGroups int) (*tempopb.QueryRangeRequest, *tempopb.QueryRangeResponse) {
	query := "{} | quantile_over_time(duration, .9) by (resource.service.name)"
	if quantiles > 1 {
		query = "{} | quantile_over_time(duration, .5, .9, .99) by (resource.service.name)"
	}
	req := &tempopb.QueryRangeRequest{Query: query, Start: 1699999980000000000, End: 1700003580000000000, Step: uint64(time.Minute), Exemplars: exemplars}
	resp := &tempopb.QueryRangeResponse{}
	for group := range 16 {
		for bucket := range buckets {
			ts := &tempopb.TimeSeries{Labels: []commonv1.KeyValue{
				{Key: "resource.service.name", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: fmt.Sprintf("service-%d", group)}}},
				{Key: internalLabelBucket, Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_DoubleValue{DoubleValue: math.Exp2(float64(bucket - 16))}}},
			}}
			for step := range 60 {
				timestamp := int64(1699999980000) + int64(step+1)*60000
				ts.Samples = append(ts.Samples, tempopb.Sample{TimestampMs: timestamp, Value: float64(step%7 + 1)})
				if group < exemplarGroups && bucket == 0 && step < int(exemplars) {
					ts.Exemplars = append(ts.Exemplars, tempopb.Exemplar{TimestampMs: timestamp, Value: math.Exp2(float64(step%max(buckets, 1) - 16)), Labels: []commonv1.KeyValue{{Key: "trace_id", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: fmt.Sprintf("trace-%d-%d", group, step)}}}}})
				}
			}
			resp.Series = append(resp.Series, ts)
		}
	}
	return req, resp
}

func BenchmarkHistogramQueryRange(b *testing.B) {
	for _, buckets := range []int{1, 32} {
		for _, quantiles := range []int{1, 3} {
			for _, exemplarGroups := range []int{0, 1, 16} {
				b.Run(fmt.Sprintf("buckets=%d/quantiles=%d/exemplarGroups=%d", buckets, quantiles, exemplarGroups), func(b *testing.B) {
					req, resp := histogramAlgorithmInput(buckets, quantiles, 20, exemplarGroups)
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						c, err := QueryRangeCombinerFor(req, AggregateModeFinal, 0)
						if err != nil {
							b.Fatal(err)
						}
						c.Combine(resp)
						if len(c.Response().Series) != 16*quantiles {
							b.Fatal("unexpected result size")
						}
					}
				})
			}
		}
	}
}

func TestHistogramQueryRangeSamplesUnaffectedByExemplars(t *testing.T) {
	for _, buckets := range []int{1, 32} {
		for _, quantiles := range []int{1, 3} {
			var expected []*tempopb.TimeSeries
			for _, exemplarGroups := range []int{16, 1, 0} {
				req, resp := histogramAlgorithmInput(buckets, quantiles, 20, exemplarGroups)
				c, err := QueryRangeCombinerFor(req, AggregateModeFinal, 0)
				require.NoError(t, err)
				c.Combine(resp)
				got := c.Response()
				require.Len(t, got.Series, 16*quantiles)
				totalExemplars := 0
				for _, s := range got.Series {
					require.Len(t, s.Samples, 60)
					totalExemplars += len(s.Exemplars)
					s.Exemplars = nil
				}
				if exemplarGroups > 0 {
					require.Positive(t, totalExemplars)
				} else {
					require.Zero(t, totalExemplars)
				}
				sort.Slice(got.Series, func(i, j int) bool { return got.Series[i].String() < got.Series[j].String() })
				if expected == nil {
					expected = got.Series
				} else {
					require.Equal(t, expected, got.Series)
				}
			}
		}
	}
}

func TestHistogramExemplarsUseAllGroups(t *testing.T) {
	req := &tempopb.QueryRangeRequest{Start: 1699999980000000000, End: 1700000100000000000, Step: uint64(time.Minute), Exemplars: 20}
	agg := NewHistogramAggregator(req, []float64{.25, .75}, 20)
	series := func(name string, bucket, count float64) *tempopb.TimeSeries {
		return &tempopb.TimeSeries{Labels: []commonv1.KeyValue{
			{Key: "service", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: name}}},
			{Key: internalLabelBucket, Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_DoubleValue{DoubleValue: bucket}}},
		}, Samples: []tempopb.Sample{{TimestampMs: 1700000040000, Value: count}}}
	}
	with := series("with", 128, 10)
	with.Exemplars = []tempopb.Exemplar{{TimestampMs: 1700000040000, Value: 80}}
	agg.Combine([]*tempopb.TimeSeries{with, series("without", 2, 90)})
	results := agg.Results()
	require.Len(t, results, 4)

	totalExemplars := 0
	for _, result := range results {
		totalExemplars += len(result.Exemplars)
		if result.Labels[0].Value.EncodeToString(false) == "with" && result.Labels[1].Value.Float() == .75 {
			require.Len(t, result.Exemplars, 1)
			require.Equal(t, 80.0, result.Exemplars[0].Value)
		} else {
			require.Empty(t, result.Exemplars)
		}
	}
	require.Equal(t, 1, totalExemplars)
}
