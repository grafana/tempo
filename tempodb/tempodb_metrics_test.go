package tempodb

import (
	"context"
	"math"
	"path"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/go-cmp/cmp"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resource_v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/encoding/vparquet4"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/stretchr/testify/require"
)

func requestWithDefaultRange(q string) *tempopb.QueryRangeRequest {
	return &tempopb.QueryRangeRequest{
		Start: 1,
		End:   50 * uint64(time.Second),
		Step:  15 * uint64(time.Second),
		Query: q,
	}
}

var queryRangeTestCases = []struct {
	name string
	req  *tempopb.QueryRangeRequest
	// expectedL1 is the expected result of the first level of aggregation
	expectedL1 []*tempopb.TimeSeries
	// expectedL2 is the expected result of the second level of aggregation
	// if nil, the data is not changed and expected results are from previous step
	expectedL2 []*tempopb.TimeSeries
	// expectedL3 is the expected result of the third level of aggregation
	// if nil, the data is not changed and expected results are from previous step
	expectedL3 []*tempopb.TimeSeries
}{
	{
		name: "rate",
		req:  requestWithDefaultRange("{ } | rate()"),
		expectedL1: []*tempopb.TimeSeries{
			{
				PromLabels: `{__name__="rate"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "rate")},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 14.0 / 15.0},     // First interval starts at 1, so it only has 14 spans
					{TimestampMs: 15_000, Value: 1.0},        // Spans every 1 second
					{TimestampMs: 30_000, Value: 1.0},        // Spans every 1 second
					{TimestampMs: 45_000, Value: 5.0 / 15.0}, // Interval [45,50) has 5 spans
					{TimestampMs: 60_000, Value: 0},          // I think this is a bug that we extend out an extra interval
				},
			},
		},
		expectedL2: []*tempopb.TimeSeries{
			{
				PromLabels: `{__name__="rate"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "rate")},
				// with two sources rate will be doubled
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 2 * 14.0 / 15.0},
					{TimestampMs: 15_000, Value: 2 * 1.0},
					{TimestampMs: 30_000, Value: 2 * 1.0},
					{TimestampMs: 45_000, Value: 2 * 5.0 / 15.0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
		},
	},
	{
		name: "rate_with_filter",
		req:  requestWithDefaultRange(`{ .service.name="even" } | rate()`),
		expectedL1: []*tempopb.TimeSeries{
			{
				PromLabels: `{__name__="rate"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "rate")},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 7.0 / 15.0},      // Interval [ 1, 14], 7 spans at 2, 4, 6, 8, 10, 12, 14
					{TimestampMs: 15_000, Value: 7.0 / 15.0}, // Interval [15, 29], 7 spans at 16, 18, 20, 22, 24, 26, 28
					{TimestampMs: 30_000, Value: 8.0 / 15.0}, // Interval [30, 44], 8 spans at 30, 32, 34, 36, 38, 40, 42, 44
					{TimestampMs: 45_000, Value: 2.0 / 15.0}, // Interval [45, 50), 2 spans at 46, 48
					{TimestampMs: 60_000, Value: 0},          // I think this is a bug that we extend out an extra interval
				},
			},
		},
		expectedL2: []*tempopb.TimeSeries{
			{
				PromLabels: `{__name__="rate"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "rate")},
				// with two sources rate will be doubled
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 2 * 7.0 / 15.0},
					{TimestampMs: 15_000, Value: 2 * 7.0 / 15.0},
					{TimestampMs: 30_000, Value: 2 * 8.0 / 15.0},
					{TimestampMs: 45_000, Value: 2 * 2.0 / 15.0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
		},
	},
	{
		name: "rate_no_spans",
		req:  requestWithDefaultRange(`{ .service.name="does_not_exist" } | rate()`),
		expectedL1: []*tempopb.TimeSeries{
			{
				PromLabels: `{__name__="rate"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "rate")},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 0},
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
		},
	},
	{
		name: "count_over_time",
		req:  requestWithDefaultRange(`{ } | count_over_time()`),
		expectedL1: []*tempopb.TimeSeries{
			{
				PromLabels: `{__name__="count_over_time"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "count_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 14},      // Interval [1, 14], 14 spans
					{TimestampMs: 15_000, Value: 15}, // Interval [15, 29], 15 spans
					{TimestampMs: 30_000, Value: 15}, // Interval [30, 44], 15 spans
					{TimestampMs: 45_000, Value: 5},  // Interval [45, 50), 5 spans
					{TimestampMs: 60_000, Value: 0},
				},
			},
		},
		expectedL2: []*tempopb.TimeSeries{
			{
				PromLabels: `{__name__="count_over_time"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "count_over_time")},
				// with two sources count will be doubled
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 2 * 14},
					{TimestampMs: 15_000, Value: 2 * 15},
					{TimestampMs: 30_000, Value: 2 * 15},
					{TimestampMs: 45_000, Value: 2 * 5},
					{TimestampMs: 60_000, Value: 0},
				},
			},
		},
	},
	{
		name: "count_over_time",
		req:  requestWithDefaultRange(`{ } | count_over_time() by (.service.name)`),
		expectedL1: []*tempopb.TimeSeries{
			{
				PromLabels: `{".service.name"="even"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueString(".service.name", "even")},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 7},      // [2, 4, 6, 8, 10, 12, 14] - total: 7
					{TimestampMs: 15_000, Value: 7}, // [16, 18, 20, 22, 24, 26, 28] - total: 7
					{TimestampMs: 30_000, Value: 8}, // [30, 32, 34, 36, 38, 40, 42, 44] - total: 8
					{TimestampMs: 45_000, Value: 2}, // [46, 48]
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{".service.name"="odd"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueString(".service.name", "odd")},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 7},      // [1, 3, 5, 7, 9, 11, 13] - total: 7
					{TimestampMs: 15_000, Value: 8}, // [15, 17, 19, 21, 23, 25, 27, 29] - total: 8
					{TimestampMs: 30_000, Value: 7}, // [31, 33, 35, 37, 39, 41, 43] - total: 7
					{TimestampMs: 45_000, Value: 3}, // [45, 47, 49] - total: 3
					{TimestampMs: 60_000, Value: 0},
				},
			},
		},
		expectedL2: []*tempopb.TimeSeries{
			{
				PromLabels: `{".service.name"="even"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueString(".service.name", "even")},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 2 * 7},
					{TimestampMs: 15_000, Value: 2 * 7},
					{TimestampMs: 30_000, Value: 2 * 8},
					{TimestampMs: 45_000, Value: 2 * 2},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{".service.name"="odd"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueString(".service.name", "odd")},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 2 * 7},
					{TimestampMs: 15_000, Value: 2 * 8},
					{TimestampMs: 30_000, Value: 2 * 7},
					{TimestampMs: 45_000, Value: 2 * 3},
					{TimestampMs: 60_000, Value: 0},
				},
			},
		},
	},
	{
		name: "min_over_time",
		req:  requestWithDefaultRange("{ } | min_over_time(duration)"),
		expectedL1: []*tempopb.TimeSeries{
			{
				PromLabels: `{__name__="min_over_time"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "min_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 1},       // Interval [1, 14], min is 1
					{TimestampMs: 15_000, Value: 15}, // Interval [15, 29], min is 15
					{TimestampMs: 30_000, Value: 30}, // Interval [30, 44], min is 30
					{TimestampMs: 45_000, Value: 45}, // Interval [45, 50), min is 45
				},
			},
		},
		expectedL2: nil, // results should be the same: min(a, a) = a
	},
	{
		name: "max_over_time",
		req:  requestWithDefaultRange("{ } | max_over_time(duration)"),
		expectedL1: []*tempopb.TimeSeries{
			{
				PromLabels: `{__name__="max_over_time"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "max_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 14},      // Interval [1, 14], max is 14
					{TimestampMs: 15_000, Value: 29}, // Interval [15, 29], max is 29
					{TimestampMs: 30_000, Value: 44}, // Interval [30, 44], max is 44
					{TimestampMs: 45_000, Value: 49}, // Interval [45, 50), max is 49
				},
			},
		},
		expectedL2: nil, // results should be the same: max(a, a) = a
	},
	{
		name: "avg_over_time",
		req:  requestWithDefaultRange("{ } | avg_over_time(duration)"),
		expectedL1: []*tempopb.TimeSeries{
			{
				PromLabels: `{__name__="avg_over_time"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "avg_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 105 / 14.0},      // sum from 1 to 14 is 105
					{TimestampMs: 15_000, Value: 330 / 15.0}, // sum from 15 to 29 is 330
					{TimestampMs: 30_000, Value: 555 / 15.0}, // sum from 30 to 44 is 555
					{TimestampMs: 45_000, Value: 235 / 5.0},  // sum from 45 to 49 is 235
				},
			},
			{
				PromLabels: `{__meta_type="__count", __name__="avg_over_time"}`,
				Labels: []common_v1.KeyValue{
					tempopb.MakeKeyValueString("__name__", "avg_over_time"),
					tempopb.MakeKeyValueString("__meta_type", "__count"),
				},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 14},
					{TimestampMs: 15_000, Value: 15},
					{TimestampMs: 30_000, Value: 15},
					{TimestampMs: 45_000, Value: 5},
				},
			},
		},
		expectedL2: nil,
		expectedL3: []*tempopb.TimeSeries{
			{
				PromLabels: `{__name__="avg_over_time"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "avg_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 105 / 14.0},      // sum from 1 to 14 is 105
					{TimestampMs: 15_000, Value: 330 / 15.0}, // sum from 15 to 29 is 330
					{TimestampMs: 30_000, Value: 555 / 15.0}, // sum from 30 to 44 is 555
					{TimestampMs: 45_000, Value: 235 / 5.0},  // sum from 45 to 49 is 235
				},
			},
		},
	},
	{
		name: "sum_over_time",
		req:  requestWithDefaultRange("{ } | sum_over_time(duration)"),
		expectedL1: []*tempopb.TimeSeries{
			{
				PromLabels: `{__name__="sum_over_time"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "sum_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 105},      // sum from 1 to 14 is 105
					{TimestampMs: 15_000, Value: 330}, // sum from 15 to 29 is 330
					{TimestampMs: 30_000, Value: 555}, // sum from 30 to 44 is 555
					{TimestampMs: 45_000, Value: 235}, // sum from 45 to 49 is 235
				},
			},
		},
		expectedL2: []*tempopb.TimeSeries{
			{
				PromLabels: `{__name__="sum_over_time"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "sum_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 2 * 105},
					{TimestampMs: 15_000, Value: 2 * 330},
					{TimestampMs: 30_000, Value: 2 * 555},
					{TimestampMs: 45_000, Value: 2 * 235},
				},
			},
		},
	},
	{
		name: "quantile_over_time",
		req:  requestWithDefaultRange("{ } | quantile_over_time(duration, .5)"),
		// first two levels for quantile are buckets and count, then on level 3 we can compute the quantile
		expectedL1: []*tempopb.TimeSeries{
			{
				PromLabels: `{__bucket="1.073741824"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 1.073741824)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 1}, // 1 number (1) is less than 1.07
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{__bucket="2.147483648"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 2.147483648)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 1}, // 1 number (2) is between 1.07 and 2.15
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{__bucket="4.294967296"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 4.294967296)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 2}, // 2 numbers (3, 4) are between 2.15 and 4.29
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{__bucket="8.589934592"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 8.589934592)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 4}, // 5, 6, 7, 8
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{__bucket="17.179869184"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 17.179869184)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 6}, // 9, 10, 11, 12, 13, 14
					{TimestampMs: 15_000, Value: 3},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{__bucket="34.359738368"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 34.359738368)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 0},
					{TimestampMs: 15_000, Value: 12}, // and so on
					{TimestampMs: 30_000, Value: 5},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{__bucket="68.719476736"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 68.719476736)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 0},
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 10},
					{TimestampMs: 45_000, Value: 5},
					{TimestampMs: 60_000, Value: 0},
				},
			},
		},
		expectedL2: []*tempopb.TimeSeries{
			{
				PromLabels: `{__bucket="1.073741824"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 1.073741824)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 2 * 1},
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{__bucket="2.147483648"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 2.147483648)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 2 * 1},
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{__bucket="4.294967296"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 4.294967296)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 2 * 2},
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{__bucket="8.589934592"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 8.589934592)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 2 * 4},
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{__bucket="17.179869184"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 17.179869184)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 2 * 6},
					{TimestampMs: 15_000, Value: 2 * 3},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{__bucket="34.359738368"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 34.359738368)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 0},
					{TimestampMs: 15_000, Value: 2 * 12},
					{TimestampMs: 30_000, Value: 2 * 5},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{__bucket="68.719476736"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 68.719476736)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 0},
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 2 * 10},
					{TimestampMs: 45_000, Value: 2 * 5},
					{TimestampMs: 60_000, Value: 0},
				},
			},
		},
		expectedL3: []*tempopb.TimeSeries{
			{
				PromLabels: `{p="0.5"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("p", 0.5)},
				Samples: []tempopb.Sample{
					// Actual q50 is 7.5: half of the number is less than 7.5 and half is greater
					// 1 2 3 4 5 6 7 || 8 9 10 11 12 13 14
					{TimestampMs: 0, Value: 7.22324520567668},
					// 15 16 17 18 19 20 21 || 22 || 23 24 25 26 27 28 29
					{TimestampMs: 15_000, Value: 22.279533901541168},
					// Actual q50 is 37. On low number of samples, the quantile can be not accurate
					// 30 31 32 33 34 35 36 || 37 || 38 39 40 41 42 43 44
					{TimestampMs: 30_000, Value: 40.86084533685759},
					// Actual q50 is 47
					// 45 46 || 47 || 48 49
					{TimestampMs: 45_000, Value: 48.592007999616804},
					{TimestampMs: 60_000, Value: 0},
				},
			},
		},
	},
	{
		name: "histogram_over_time",
		req:  requestWithDefaultRange("{ } | histogram_over_time(duration)"),
		expectedL1: []*tempopb.TimeSeries{
			{
				PromLabels: `{__bucket="1.073741824"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 1.073741824)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 1},
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{__bucket="2.147483648"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 2.147483648)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 1},
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{__bucket="4.294967296"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 4.294967296)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 2},
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{__bucket="8.589934592"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 8.589934592)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 4},
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{__bucket="17.179869184"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 17.179869184)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 6},
					{TimestampMs: 15_000, Value: 3},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{__bucket="34.359738368"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 34.359738368)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 0},
					{TimestampMs: 15_000, Value: 12},
					{TimestampMs: 30_000, Value: 5},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{__bucket="68.719476736"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 68.719476736)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 0},
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 10},
					{TimestampMs: 45_000, Value: 5},
					{TimestampMs: 60_000, Value: 0},
				},
			},
		},
		expectedL2: []*tempopb.TimeSeries{
			{
				PromLabels: `{__bucket="1.073741824"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 1.073741824)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 2 * 1},
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{__bucket="2.147483648"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 2.147483648)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 2 * 1},
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{__bucket="4.294967296"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 4.294967296)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 2 * 2},
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{__bucket="8.589934592"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 8.589934592)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 2 * 4},
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{__bucket="17.179869184"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 17.179869184)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 2 * 6},
					{TimestampMs: 15_000, Value: 2 * 3},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{__bucket="34.359738368"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 34.359738368)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 0},
					{TimestampMs: 15_000, Value: 2 * 12},
					{TimestampMs: 30_000, Value: 2 * 5},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				PromLabels: `{__bucket="68.719476736"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 68.719476736)},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 0},
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 2 * 10},
					{TimestampMs: 45_000, Value: 2 * 5},
					{TimestampMs: 60_000, Value: 0},
				},
			},
		},
	},
	{
		name:       "compare_no_spans",
		req:        requestWithDefaultRange(`{ .service.name="does_not_exist" } | compare({ })`),
		expectedL1: []*tempopb.TimeSeries{},
	},
	{
		name:       "compare_no_spans",
		req:        requestWithDefaultRange(`{ .service.name="does_not_exist" } | compare({ .service.name="does_not_exist_for_sure" })`),
		expectedL1: []*tempopb.TimeSeries{},
	},
	// --- Non-standard range queries ---
	{
		name: "end<step",
		req: &tempopb.QueryRangeRequest{
			Start: 1,
			End:   3 * uint64(time.Second),
			Step:  5 * uint64(time.Second),
			Query: `{ } | count_over_time()`,
		},
		expectedL1: []*tempopb.TimeSeries{
			{
				PromLabels: `{__name__="count_over_time"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "count_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 2}, // 1, 2
					{TimestampMs: 5_000, Value: 0},
				},
			},
		},
		expectedL2: []*tempopb.TimeSeries{
			{
				PromLabels: `{__name__="count_over_time"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "count_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 2 * 2},
					{TimestampMs: 5_000, Value: 0},
				},
			},
		},
	},
	{
		name: "end=step",
		req: &tempopb.QueryRangeRequest{
			Start: 1,
			End:   5 * uint64(time.Second),
			Step:  5 * uint64(time.Second),
			Query: `{ } | count_over_time()`,
		},
		expectedL1: []*tempopb.TimeSeries{
			{
				PromLabels: `{__name__="count_over_time"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "count_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 4}, // 1, 2, 3, 4
					{TimestampMs: 5_000, Value: 0},
				},
			},
		},
		expectedL2: []*tempopb.TimeSeries{
			{
				PromLabels: `{__name__="count_over_time"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "count_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 2 * 4},
					{TimestampMs: 5_000, Value: 0},
				},
			},
		},
	},
	{
		name: "small step",
		req: &tempopb.QueryRangeRequest{
			Start: 1,
			End:   3 * uint64(time.Second),
			Step:  500 * uint64(time.Millisecond),
			Query: `{ } | count_over_time()`,
		},
		expectedL1: []*tempopb.TimeSeries{
			{
				PromLabels: `{__name__="count_over_time"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "count_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 0},
					{TimestampMs: 500, Value: 0},
					{TimestampMs: 1000, Value: 1},
					{TimestampMs: 1500, Value: 0},
					{TimestampMs: 2000, Value: 1},
					{TimestampMs: 2500, Value: 0},
					{TimestampMs: 3000, Value: 0},
				},
			},
		},
		expectedL2: []*tempopb.TimeSeries{
			{
				PromLabels: `{__name__="count_over_time"}`,
				Labels:     []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "count_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 0, Value: 0},
					{TimestampMs: 500, Value: 0},
					{TimestampMs: 1000, Value: 2 * 1},
					{TimestampMs: 1500, Value: 0},
					{TimestampMs: 2000, Value: 2 * 1},
					{TimestampMs: 2500, Value: 0},
					{TimestampMs: 3000, Value: 0},
				},
			},
		},
	},
}

var expectedCompareTs = []*tempopb.TimeSeries{
	{
		PromLabels: `{__meta_type="baseline", "resource.service.name"="odd"}`,
		Labels: []common_v1.KeyValue{
			tempopb.MakeKeyValueString("__meta_type", "baseline"),
			tempopb.MakeKeyValueString("resource.service.name", "odd"),
		},
		Samples: []tempopb.Sample{
			{TimestampMs: 0, Value: 7},
			{TimestampMs: 15_000, Value: 8},
			{TimestampMs: 30_000, Value: 7},
			{TimestampMs: 45_000, Value: 3},
			{TimestampMs: 60_000, Value: 0},
		},
	},
	{
		PromLabels: `{__meta_type="baseline_total", "resource.service.name"="<nil>"}`,
		Labels: []common_v1.KeyValue{
			tempopb.MakeKeyValueString("__meta_type", "baseline_total"),
			tempopb.MakeKeyValueString("resource.service.name", "nil"),
		},
		Samples: []tempopb.Sample{
			{TimestampMs: 0, Value: 7},
			{TimestampMs: 15_000, Value: 8},
			{TimestampMs: 30_000, Value: 7},
			{TimestampMs: 45_000, Value: 3},
			{TimestampMs: 60_000, Value: 0},
		},
	},
	{
		PromLabels: `{__meta_type="selection", "resource.service.name"="even"}`,
		Labels: []common_v1.KeyValue{
			tempopb.MakeKeyValueString("__meta_type", "selection"),
			tempopb.MakeKeyValueString("resource.service.name", "even"),
		},
		Samples: []tempopb.Sample{
			{TimestampMs: 0, Value: 7},
			{TimestampMs: 15_000, Value: 7},
			{TimestampMs: 30_000, Value: 8},
			{TimestampMs: 45_000, Value: 2},
			{TimestampMs: 60_000, Value: 0},
		},
	},
	{
		PromLabels: `{__meta_type="selection_total", "resource.service.name"="<nil>"}`,
		Labels: []common_v1.KeyValue{
			tempopb.MakeKeyValueString("__meta_type", "selection_total"),
			tempopb.MakeKeyValueString("resource.service.name", "nil"),
		},
		Samples: []tempopb.Sample{
			{TimestampMs: 0, Value: 7},
			{TimestampMs: 15_000, Value: 7},
			{TimestampMs: 30_000, Value: 8},
			{TimestampMs: 45_000, Value: 2},
			{TimestampMs: 60_000, Value: 0},
		},
	},
}

// TestTempoDBQueryRange tests the metrics query functionality of TempoDB by verifying various types of
// time-series queries on trace data. The test:
//
// 1. Sets up a test environment
//
// 2. Generates test data:
//   - 100 test spans distributed across 1-100 seconds
//   - Each span's duration equals its start time (e.g. span at 2s has duration 2s)
//   - Spans are tagged with service names "even" or "odd" based on their start time
//
// 3. Tests various query types:
//   - Rate queries (rate(), rate() with filters)
//   - Count queries (count_over_time(), count_over_time() by service)
//   - Statistical queries (min, max, avg, sum, quantile, histogram)
//   - Edge cases with different time ranges and step sizes
//
// 4. Validates results at three processing levels covering the whole query pipeline:
//   - Level 1: Initial query results
//   - Level 2: Results after first aggregation simulating multiple sources
//   - Level 3: Final aggregation results
func TestTempoDBQueryRange(t *testing.T) {
	var (
		tempDir      = t.TempDir()
		blockVersion = vparquet4.VersionString
	)

	dc := backend.DedicatedColumns{
		{Scope: "resource", Name: "res-dedicated.01", Type: "string"},
		{Scope: "resource", Name: "res-dedicated.02", Type: "string"},
		{Scope: "span", Name: "span-dedicated.01", Type: "string"},
		{Scope: "span", Name: "span-dedicated.02", Type: "string"},
	}
	r, w, c, err := New(&Config{
		Backend: backend.Local,
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &common.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              .01,
			BloomShardSizeBytes:  100_000,
			Version:              blockVersion,
			IndexPageSizeBytes:   1000,
			RowGroupSizeBytes:    10000,
			DedicatedColumns:     dc,
		},
		WAL: &wal.Config{
			Filepath:       path.Join(tempDir, "wal"),
			IngestionSlack: time.Since(time.Time{}),
		},
		Search: &SearchConfig{
			ChunkSizeBytes:  1_000_000,
			ReadBufferCount: 8, ReadBufferSizeBytes: 4 * 1024 * 1024,
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	err = c.EnableCompaction(context.Background(), &CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	ctx := context.Background()
	r.EnablePolling(ctx, &mockJobSharder{})

	// Write to wal
	wal := w.WAL()

	meta := &backend.BlockMeta{BlockID: backend.NewUUID(), TenantID: testTenantID, DedicatedColumns: dc}
	head, err := wal.NewBlock(meta, model.CurrentEncoding)
	require.NoError(t, err)
	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	totalSpans := 100
	for i := 1; i <= totalSpans; i++ {
		tid := test.ValidTraceID(nil)

		sp := test.MakeSpan(tid)

		// Start time is i seconds
		sp.StartTimeUnixNano = uint64(i * int(time.Second))

		// Duration is i seconds
		sp.EndTimeUnixNano = sp.StartTimeUnixNano + uint64(i*int(time.Second))

		// Service name
		var svcName string
		if i%2 == 0 {
			svcName = "even"
		} else {
			svcName = "odd"
		}

		tr := &tempopb.Trace{
			ResourceSpans: []*v1.ResourceSpans{
				{
					Resource: &resource_v1.Resource{
						Attributes: []*common_v1.KeyValue{tempopb.MakeKeyValueStringPtr("service.name", svcName)},
					},
					ScopeSpans: []*v1.ScopeSpans{
						{
							Spans: []*v1.Span{
								sp,
							},
						},
					},
				},
			},
		}

		b1, err := dec.PrepareForWrite(tr, 0, 0)
		require.NoError(t, err)

		b2, err := dec.ToObject([][]byte{b1})
		require.NoError(t, err)
		err = head.Append(tid, b2, 0, 0, true)
		require.NoError(t, err)
	}

	// Complete block
	block, err := w.CompleteBlock(context.Background(), head)
	require.NoError(t, err)

	f := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
		return block.Fetch(ctx, req, common.DefaultSearchOptions())
	})

	for _, tc := range queryRangeTestCases {
		t.Run(tc.name, func(t *testing.T) {
			e := traceql.NewEngine()
			eval, err := e.CompileMetricsQueryRange(tc.req, 0, 0, false)
			require.NoError(t, err)

			err = eval.Do(ctx, f, 0, 0, 0)
			require.NoError(t, err)

			actual := eval.Results().ToProto(tc.req)
			expected := tc.expectedL1

			// Slice order is not deterministic, so we sort the slices before comparing
			sortTimeSeries(actual)
			sortTimeSeries(expected)

			if diff := cmp.Diff(expected, actual, floatComparer); diff != "" {
				t.Errorf("Unexpected results for Level 1 processing. Query: %v\n Diff: %v", tc.req.Query, diff)
			}

			evalLevel2, err := e.CompileMetricsQueryRangeNonRaw(tc.req, traceql.AggregateModeSum)
			require.NoError(t, err)
			evalLevel2.ObserveSeries(actual)
			evalLevel2.ObserveSeries(actual) // emulate merging from two sources
			actual = evalLevel2.Results().ToProto(tc.req)
			sortTimeSeries(actual)

			if tc.expectedL2 != nil {
				expected = tc.expectedL2
				sortTimeSeries(expected)
			}

			if diff := cmp.Diff(expected, actual, floatComparer); diff != "" {
				t.Errorf("Unexpected results for Level 2 processing. Query: %v\n Diff: %v", tc.req.Query, diff)
			}

			evalLevel3, err := e.CompileMetricsQueryRangeNonRaw(tc.req, traceql.AggregateModeFinal)
			require.NoError(t, err)
			evalLevel3.ObserveSeries(actual)
			actual = evalLevel3.Results().ToProto(tc.req)
			sortTimeSeries(actual)

			if tc.expectedL3 != nil {
				expected = tc.expectedL3
				sortTimeSeries(expected)
			}

			if diff := cmp.Diff(expected, actual, floatComparer); diff != "" {
				t.Errorf("Unexpected results for Level 3 processing. Query: %v\n Diff: %v", tc.req.Query, diff)
			}
		})
	}

	t.Run("compare", func(t *testing.T) {
		// compare operation generates enormous amount of time series,
		// so we filter by service.name to test at least part of the results
		req := requestWithDefaultRange(`{} | compare({ .service.name="even" })`)
		e := traceql.NewEngine()

		// Level 1
		eval, err := e.CompileMetricsQueryRange(req, 0, 0, false)
		require.NoError(t, err)

		err = eval.Do(ctx, f, 0, 0, 0)
		require.NoError(t, err)

		actual := eval.Results().ToProto(req)

		const pattern = `"resource.service.name"=`
		targetTs := filterTimeSeriesByPromLabel(actual, pattern)
		sortTimeSeries(targetTs)
		require.Equal(t, expectedCompareTs, targetTs)

		// Level 2
		evalLevel2, err := e.CompileMetricsQueryRangeNonRaw(req, traceql.AggregateModeSum)
		require.NoError(t, err)
		evalLevel2.ObserveSeries(actual)
		actual = evalLevel2.Results().ToProto(req)

		targetTs = filterTimeSeriesByPromLabel(actual, pattern)
		sortTimeSeries(targetTs)
		require.Equal(t, expectedCompareTs, targetTs)

		// Level 3
		evalLevel3, err := e.CompileMetricsQueryRangeNonRaw(req, traceql.AggregateModeFinal)
		require.NoError(t, err)
		evalLevel3.ObserveSeries(actual)
		actual = evalLevel3.Results().ToProto(req)

		targetTs = filterTimeSeriesByPromLabel(actual, pattern)
		sortTimeSeries(targetTs)
		require.Equal(t, expectedCompareTs, targetTs)
	})
}

func sortTimeSeries(ts []*tempopb.TimeSeries) {
	sort.Slice(ts, func(i, j int) bool {
		return ts[i].PromLabels < ts[j].PromLabels
	})
}

var floatComparer = cmp.Comparer(func(x, y float64) bool {
	return math.Abs(x-y) < 1e-6
})

// filterTimeSeries filters the time series by the pattern in the PromLabels
func filterTimeSeriesByPromLabel(ts []*tempopb.TimeSeries, pattern string) []*tempopb.TimeSeries {
	var targetTs []*tempopb.TimeSeries
	for _, ts := range ts {
		if strings.Contains(ts.PromLabels, pattern) {
			targetTs = append(targetTs, ts)
		}
	}
	return targetTs
}
