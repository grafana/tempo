package tempodb

import (
	"context"
	"math"
	"path"
	"sort"
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
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "rate")},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 15}, // Raw count: 15 spans in interval (0, 15]
					{TimestampMs: 30_000, Value: 15}, // Raw count: 15 spans in interval (15, 30]
					{TimestampMs: 45_000, Value: 15}, // Raw count: 15 spans in interval (30, 45]
					{TimestampMs: 60_000, Value: 5},  // Raw count: 5 spans in interval (45, 50]
				},
			},
		},
		expectedL2: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "rate")},
				// Level 2 sums counts from two sources (raw counts, not rates yet)
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 2 * 15},
					{TimestampMs: 30_000, Value: 2 * 15},
					{TimestampMs: 45_000, Value: 2 * 15},
					{TimestampMs: 60_000, Value: 2 * 5},
				},
			},
		},
		expectedL3: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "rate")},
				// Level 3 applies the rate multiplier (1/step_seconds = 1/15)
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 2 * 15 / 15.0}, // 2.0
					{TimestampMs: 30_000, Value: 2 * 15 / 15.0}, // 2.0
					{TimestampMs: 45_000, Value: 2 * 15 / 15.0}, // 2.0
					{TimestampMs: 60_000, Value: 2 * 5 / 15.0},  // 0.666...
				},
			},
		},
	},
	{
		name: "rate_with_filter",
		req:  requestWithDefaultRange(`{ .service.name="even" } | rate()`),
		expectedL1: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "rate")},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 7}, // Raw count: 7 spans at 2, 4, 6, 8, 10, 12, 14
					{TimestampMs: 30_000, Value: 8}, // Raw count: 8 spans at 16, 18, 20, 22, 24, 26, 28, 30
					{TimestampMs: 45_000, Value: 7}, // Raw count: 7 spans at 32, 34, 36, 38, 40, 42, 44
					{TimestampMs: 60_000, Value: 3}, // Raw count: 3 spans at 46, 48, 50
				},
			},
		},
		expectedL2: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "rate")},
				// Level 2 sums counts from two sources (raw counts, not rates yet)
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 2 * 7},
					{TimestampMs: 30_000, Value: 2 * 8},
					{TimestampMs: 45_000, Value: 2 * 7},
					{TimestampMs: 60_000, Value: 2 * 3},
				},
			},
		},
		expectedL3: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "rate")},
				// Level 3 applies the rate multiplier (1/step_seconds = 1/15)
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 2 * 7 / 15.0}, // 0.933...
					{TimestampMs: 30_000, Value: 2 * 8 / 15.0}, // 1.066...
					{TimestampMs: 45_000, Value: 2 * 7 / 15.0}, // 0.933...
					{TimestampMs: 60_000, Value: 2 * 3 / 15.0}, // 0.4
				},
			},
		},
	},
	{
		name: "rate_no_spans",
		req:  requestWithDefaultRange(`{ .service.name="does_not_exist" } | rate()`),
		expectedL1: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "rate")},
				Samples: []tempopb.Sample{
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
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "count_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 15}, // Interval (0, 15], 15 spans
					{TimestampMs: 30_000, Value: 15}, // Interval (15, 30], 15 spans
					{TimestampMs: 45_000, Value: 15}, // Interval (30, 45], 15 spans
					{TimestampMs: 60_000, Value: 5},  // Interval (45, 50], 5 spans
				},
			},
		},
		expectedL2: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "count_over_time")},
				// with two sources count will be doubled
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 2 * 15},
					{TimestampMs: 30_000, Value: 2 * 15},
					{TimestampMs: 45_000, Value: 2 * 15},
					{TimestampMs: 60_000, Value: 2 * 5},
				},
			},
		},
	},
	{
		name: "count_over_time",
		req:  requestWithDefaultRange(`{ } | count_over_time() by (.service.name)`),
		expectedL1: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString(".service.name", "even")},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 7}, // (0, 15]: [2, 4, 6, 8, 10, 12, 14] - total: 7
					{TimestampMs: 30_000, Value: 8}, // (15, 30]: [16, 18, 20, 22, 24, 26, 28, 30] - total: 8
					{TimestampMs: 45_000, Value: 7}, // (30, 45]: [32, 34, 36, 38, 40, 42, 44] - total: 7 (30 is now excluded)
					{TimestampMs: 60_000, Value: 3}, // (45, 50]: [46, 48, 50] - total: 3
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString(".service.name", "odd")},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 8}, // (0, 15]: [1, 3, 5, 7, 9, 11, 13, 15] - total: 8
					{TimestampMs: 30_000, Value: 7}, // (15, 30]: [17, 19, 21, 23, 25, 27, 29] - total: 7
					{TimestampMs: 45_000, Value: 8}, // (30, 45]: [31, 33, 35, 37, 39, 41, 43, 45] - total: 8
					{TimestampMs: 60_000, Value: 2}, // (45, 50]: [47, 49] - total: 2
				},
			},
		},
		expectedL2: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString(".service.name", "even")},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 2 * 7},
					{TimestampMs: 30_000, Value: 2 * 8},
					{TimestampMs: 45_000, Value: 2 * 7},
					{TimestampMs: 60_000, Value: 2 * 3},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString(".service.name", "odd")},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 2 * 8},
					{TimestampMs: 30_000, Value: 2 * 7},
					{TimestampMs: 45_000, Value: 2 * 8},
					{TimestampMs: 60_000, Value: 2 * 2},
				},
			},
		},
	},
	{
		name: "min_over_time",
		req:  requestWithDefaultRange("{ } | min_over_time(duration)"),
		expectedL1: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "min_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 1},  // Interval (0, 15], min is 1
					{TimestampMs: 30_000, Value: 16}, // Interval (15, 30], min is 16
					{TimestampMs: 45_000, Value: 31}, // Interval (30, 45], min is 31
					{TimestampMs: 60_000, Value: 46}, // Interval (45, 50], min is 46
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
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "max_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 15}, // Interval (0, 15], max is 15
					{TimestampMs: 30_000, Value: 30}, // Interval (15, 30], max is 30
					{TimestampMs: 45_000, Value: 45}, // Interval (30, 45], max is 45
					{TimestampMs: 60_000, Value: 50}, // Interval (45, 50], max is 50
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
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "avg_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 120 / 15.0}, // sum from 1 to 15 is 120
					{TimestampMs: 30_000, Value: 345 / 15.0}, // sum from 16 to 30 is 345
					{TimestampMs: 45_000, Value: 570 / 15.0}, // sum from 31 to 45 is 570
					{TimestampMs: 60_000, Value: 240 / 5.0},  // sum from 46 to 50 is 240
				},
			},
			{
				Labels: []common_v1.KeyValue{
					tempopb.MakeKeyValueString("__name__", "avg_over_time"),
					tempopb.MakeKeyValueString("__meta_type", "__count"),
				},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 15},
					{TimestampMs: 30_000, Value: 15},
					{TimestampMs: 45_000, Value: 15},
					{TimestampMs: 60_000, Value: 5},
				},
			},
		},
		expectedL2: nil,
		expectedL3: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "avg_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 120 / 15.0}, // sum from 1 to 15 is 120
					{TimestampMs: 30_000, Value: 345 / 15.0}, // sum from 16 to 30 is 345
					{TimestampMs: 45_000, Value: 570 / 15.0}, // sum from 31 to 45 is 570
					{TimestampMs: 60_000, Value: 240 / 5.0},  // sum from 46 to 50 is 240
				},
			},
		},
	},
	{
		name: "sum_over_time",
		req:  requestWithDefaultRange("{ } | sum_over_time(duration)"),
		expectedL1: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "sum_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 120}, // sum from 1 to 15 is 120
					{TimestampMs: 30_000, Value: 345}, // sum from 16 to 30 is 345 (including 30)
					{TimestampMs: 45_000, Value: 570}, // sum from 31 to 45 is 570
					{TimestampMs: 60_000, Value: 240}, // sum from 46 to 50 is 240
				},
			},
		},
		expectedL2: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "sum_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 2 * 120},
					{TimestampMs: 30_000, Value: 2 * 345},
					{TimestampMs: 45_000, Value: 2 * 570},
					{TimestampMs: 60_000, Value: 2 * 240},
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
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 1.073741824)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 1}, // 1 number (1) is less than 1.07
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 2.147483648)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 1}, // 1 number (2) is between 1.07 and 2.15
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 4.294967296)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 2}, // 2 numbers (3, 4) are between 2.15 and 4.29
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 8.589934592)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 4}, // 5, 6, 7, 8
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 17.179869184)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 7}, // 9, 10, 11, 12, 13, 14, 15 from interval (0,15]
					{TimestampMs: 30_000, Value: 2}, // 16, 17 from interval (15,30]
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 34.359738368)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 13}, // 18,19,20,21,22,23,24,25,26,27,28,29,30 from interval (15,30]
					{TimestampMs: 45_000, Value: 4},  // 31, 32, 33, 34 from interval (30,45]
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 68.719476736)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 11}, // 35,36,37,38,39,40,41,42,43,44,45 from interval (30,45]
					{TimestampMs: 60_000, Value: 5},  // 46,47,48,49,50 from interval (45,50]
				},
			},
		},
		expectedL2: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 1.073741824)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 2 * 1},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 2.147483648)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 2 * 1},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 4.294967296)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 2 * 2},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 8.589934592)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 2 * 4},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 17.179869184)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 2 * 7},
					{TimestampMs: 30_000, Value: 2 * 2},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 34.359738368)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 2 * 13},
					{TimestampMs: 45_000, Value: 2 * 4},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 68.719476736)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 2 * 11},
					{TimestampMs: 60_000, Value: 2 * 5},
				},
			},
		},
		expectedL3: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("p", 0.5)},
				Samples: []tempopb.Sample{
					// 1 2 3 4 5 6 7 || 8 || 9 10 11 12 13 14 15
					{TimestampMs: 15_000, Value: 7.877004751727669},
					// 16 17 18 19 20 21 22 || 23 || 24 25 26 27 28 29 30
					{TimestampMs: 30_000, Value: 23.03449508051292},
					// Actual q50 is 38. On low number of samples, the quantile can be inaccurate
					// 31 32 33 34 35 36 37 || 38 || 39 40 41 42 43 44 45
					{TimestampMs: 45_000, Value: 42.83828931515888},
					// Actual q50 is 48
					// 46 47 || 48 || 49 50
					{TimestampMs: 60_000, Value: 48.592007999616804},
				},
			},
		},
	},
	{
		name: "histogram_over_time",
		req:  requestWithDefaultRange("{ } | histogram_over_time(duration)"),
		expectedL1: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 1.073741824)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 1}, // 1 number (1) is less than 1.07
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 2.147483648)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 1}, // 1 number (2) is between 1.07 and 2.15
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 4.294967296)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 2}, // 2 numbers (3, 4) are between 2.15 and 4.29
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 8.589934592)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 4}, // 5, 6, 7, 8
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 17.179869184)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 7}, // 9, 10, 11, 12, 13, 14, 15 from interval (0,15]
					{TimestampMs: 30_000, Value: 2}, // 16, 17 from interval (15,30]
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 34.359738368)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 13}, // 18,19,20,21,22,23,24,25,26,27,28,29,30 from interval (15,30]
					{TimestampMs: 45_000, Value: 4},  // 31, 32, 33, 34 from interval (30,45]
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 68.719476736)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 11}, // 35,36,37,38,39,40,41,42,43,44,45 from interval (30,45]
					{TimestampMs: 60_000, Value: 5},  // 46,47,48,49,50 from interval (45,50]
				},
			},
		},
		expectedL2: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 1.073741824)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 2 * 1},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 2.147483648)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 2 * 1},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 4.294967296)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 2 * 2},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 8.589934592)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 2 * 4},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 17.179869184)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 2 * 7},
					{TimestampMs: 30_000, Value: 2 * 2},
					{TimestampMs: 45_000, Value: 0},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 34.359738368)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 2 * 13},
					{TimestampMs: 45_000, Value: 2 * 4},
					{TimestampMs: 60_000, Value: 0},
				},
			},
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueDouble("__bucket", 68.719476736)},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 0},
					{TimestampMs: 30_000, Value: 0},
					{TimestampMs: 45_000, Value: 2 * 11},
					{TimestampMs: 60_000, Value: 2 * 5},
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
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "count_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 5_000, Value: 3}, // 1, 2, 3
				},
			},
		},
		expectedL2: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "count_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 5_000, Value: 3 * 2},
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
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "count_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 5_000, Value: 5}, // 1, 2, 3, 4, 5
				},
			},
		},
		expectedL2: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "count_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 5_000, Value: 5 * 2},
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
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "count_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 500, Value: 0},
					{TimestampMs: 1000, Value: 1},
					{TimestampMs: 1500, Value: 0},
					{TimestampMs: 2000, Value: 1},
					{TimestampMs: 2500, Value: 0},
					{TimestampMs: 3000, Value: 1},
				},
			},
		},
		expectedL2: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "count_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 500, Value: 0},
					{TimestampMs: 1000, Value: 2 * 1},
					{TimestampMs: 1500, Value: 0},
					{TimestampMs: 2000, Value: 2 * 1},
					{TimestampMs: 2500, Value: 0},
					{TimestampMs: 3000, Value: 2 * 1},
				},
			},
		},
	},
	{
		name: "aligned start is not zero",
		req: &tempopb.QueryRangeRequest{
			Start: 20 * uint64(time.Second),
			End:   50 * uint64(time.Second),
			Step:  10 * uint64(time.Second),
			Query: `{ } | count_over_time()`,
		},
		expectedL1: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "count_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 30_000, Value: 10},
					{TimestampMs: 40_000, Value: 10},
					{TimestampMs: 50_000, Value: 10},
				},
			},
		},
		expectedL2: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "count_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 30_000, Value: 2 * 10},
					{TimestampMs: 40_000, Value: 2 * 10},
					{TimestampMs: 50_000, Value: 2 * 10},
				},
			},
		},
	},
	{
		name: "not aligned start is not zero",
		req: &tempopb.QueryRangeRequest{
			Start: 21 * uint64(time.Second),
			End:   50 * uint64(time.Second),
			Step:  10 * uint64(time.Second),
			Query: `{ } | count_over_time()`,
		},
		expectedL1: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "count_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 30_000, Value: 9},
					{TimestampMs: 40_000, Value: 10},
					{TimestampMs: 50_000, Value: 10},
				},
			},
		},
		expectedL2: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "count_over_time")},
				Samples: []tempopb.Sample{
					{TimestampMs: 30_000, Value: 2 * 9},
					{TimestampMs: 40_000, Value: 2 * 10},
					{TimestampMs: 50_000, Value: 2 * 10},
				},
			},
		},
	},
}

var expectedCompareTs = []*tempopb.TimeSeries{
	{
		Labels: []common_v1.KeyValue{
			tempopb.MakeKeyValueString("__meta_type", "baseline"),
			tempopb.MakeKeyValueString("resource.service.name", "odd"),
		},
		Samples: []tempopb.Sample{
			{TimestampMs: 15_000, Value: 8},
			{TimestampMs: 30_000, Value: 7},
			{TimestampMs: 45_000, Value: 8},
			{TimestampMs: 60_000, Value: 2},
		},
	},
	{
		Labels: []common_v1.KeyValue{
			tempopb.MakeKeyValueString("__meta_type", "baseline_total"),
			tempopb.MakeKeyValueString("resource.service.name", "nil"),
		},
		Samples: []tempopb.Sample{
			{TimestampMs: 15_000, Value: 8},
			{TimestampMs: 30_000, Value: 7},
			{TimestampMs: 45_000, Value: 8},
			{TimestampMs: 60_000, Value: 2},
		},
	},
	{
		Labels: []common_v1.KeyValue{
			tempopb.MakeKeyValueString("__meta_type", "selection"),
			tempopb.MakeKeyValueString("resource.service.name", "even"),
		},
		Samples: []tempopb.Sample{
			{TimestampMs: 15_000, Value: 7},
			{TimestampMs: 30_000, Value: 8},
			{TimestampMs: 45_000, Value: 7},
			{TimestampMs: 60_000, Value: 3},
		},
	},
	{
		Labels: []common_v1.KeyValue{
			tempopb.MakeKeyValueString("__meta_type", "selection_total"),
			tempopb.MakeKeyValueString("resource.service.name", "nil"),
		},
		Samples: []tempopb.Sample{
			{TimestampMs: 15_000, Value: 7},
			{TimestampMs: 30_000, Value: 8},
			{TimestampMs: 45_000, Value: 7},
			{TimestampMs: 60_000, Value: 3},
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
	r.EnablePolling(ctx, &mockJobSharder{}, false)

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

		const labelName = `resource.service.name`
		targetTs := filterTimeSeriesByLabel(actual, labelName)
		sortTimeSeries(targetTs)
		require.Equal(t, expectedCompareTs, targetTs)

		// Level 2
		evalLevel2, err := e.CompileMetricsQueryRangeNonRaw(req, traceql.AggregateModeSum)
		require.NoError(t, err)
		evalLevel2.ObserveSeries(actual)
		actual = evalLevel2.Results().ToProto(req)

		targetTs = filterTimeSeriesByLabel(actual, labelName)
		sortTimeSeries(targetTs)
		require.Equal(t, expectedCompareTs, targetTs)

		// Level 3
		evalLevel3, err := e.CompileMetricsQueryRangeNonRaw(req, traceql.AggregateModeFinal)
		require.NoError(t, err)
		evalLevel3.ObserveSeries(actual)
		actual = evalLevel3.Results().ToProto(req)

		targetTs = filterTimeSeriesByLabel(actual, labelName)
		sortTimeSeries(targetTs)
		require.Equal(t, expectedCompareTs, targetTs)
	})
}

// TestRateCountOverTimeRelationship validates the mathematical relationship between rate() and count_over_time() functions.
// This test ensures that rate = count_over_time / step_seconds at the final aggregation level.
//
// The test validates the expected behavior at each aggregation level:
// - Level 1 (raw) & Level 2 (sum): rate() should return raw counts (same as count_over_time)
// - Level 3 (final): rate() should return actual rates (counts / step_seconds)
func TestRateCountOverTimeRelationship(t *testing.T) {
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
	r.EnablePolling(ctx, &mockJobSharder{}, false)

	// Write to wal
	wal := w.WAL()

	meta := &backend.BlockMeta{BlockID: backend.NewUUID(), TenantID: testTenantID, DedicatedColumns: dc}
	head, err := wal.NewBlock(meta, model.CurrentEncoding)
	require.NoError(t, err)
	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	// Create test data with 50 spans
	totalSpans := 50
	for i := 1; i <= totalSpans; i++ {
		tid := test.ValidTraceID(nil)
		sp := test.MakeSpan(tid)
		sp.StartTimeUnixNano = uint64(i * int(time.Second))
		sp.EndTimeUnixNano = sp.StartTimeUnixNano + uint64(i*int(time.Second))

		tr := &tempopb.Trace{
			ResourceSpans: []*v1.ResourceSpans{
				{
					Resource: &resource_v1.Resource{
						Attributes: []*common_v1.KeyValue{tempopb.MakeKeyValueStringPtr("service.name", "test")},
					},
					ScopeSpans: []*v1.ScopeSpans{
						{
							Spans: []*v1.Span{sp},
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

	// Test with different step sizes to validate the relationship
	testCases := []struct {
		name        string
		req         *tempopb.QueryRangeRequest
		description string
	}{
		{
			name:        "15_second_step",
			req:         requestWithDefaultRange("{ } | count_over_time()"),
			description: "Default 15-second step",
		},
		{
			name: "30_second_step",
			req: &tempopb.QueryRangeRequest{
				Start: 1,
				End:   50 * uint64(time.Second),
				Step:  30 * uint64(time.Second),
				Query: "{ } | count_over_time()",
			},
			description: "30-second step",
		},
		{
			name: "10_second_step",
			req: &tempopb.QueryRangeRequest{
				Start: 1,
				End:   50 * uint64(time.Second),
				Step:  10 * uint64(time.Second),
				Query: "{ } | count_over_time()",
			},
			description: "10-second step",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := traceql.NewEngine()

			// Run count_over_time query
			countReq := &tempopb.QueryRangeRequest{
				Start: tc.req.Start,
				End:   tc.req.End,
				Step:  tc.req.Step,
				Query: "{ } | count_over_time()",
			}
			countEval, err := e.CompileMetricsQueryRange(countReq, 0, 0, false)
			require.NoError(t, err)
			err = countEval.Do(ctx, f, 0, 0, 0)
			require.NoError(t, err)

			// Run rate query
			rateReq := &tempopb.QueryRangeRequest{
				Start: tc.req.Start,
				End:   tc.req.End,
				Step:  tc.req.Step,
				Query: "{ } | rate()",
			}
			rateEval, err := e.CompileMetricsQueryRange(rateReq, 0, 0, false)
			require.NoError(t, err)
			err = rateEval.Do(ctx, f, 0, 0, 0)
			require.NoError(t, err)

			// Level 1: Raw results - rate() should return raw counts (same as count_over_time)
			countResults := countEval.Results().ToProto(countReq)
			rateResults := rateEval.Results().ToProto(rateReq)
			validateRateCountRelationship(t, countResults, rateResults, 1.0, "Level 1 (raw)", tc.description)

			// Level 2: Sum aggregation (simulate multiple sources) - rate() should return raw counts
			countEvalL2, err := e.CompileMetricsQueryRangeNonRaw(countReq, traceql.AggregateModeSum)
			require.NoError(t, err)
			// Observe twice to simulate multiple sources
			countEvalL2.ObserveSeries(countResults)
			countEvalL2.ObserveSeries(countResults)
			countResultsL2 := countEvalL2.Results().ToProto(countReq)

			rateEvalL2, err := e.CompileMetricsQueryRangeNonRaw(rateReq, traceql.AggregateModeSum)
			require.NoError(t, err)
			// Observe twice to simulate multiple sources
			rateEvalL2.ObserveSeries(rateResults)
			rateEvalL2.ObserveSeries(rateResults)
			rateResultsL2 := rateEvalL2.Results().ToProto(rateReq)
			validateRateCountRelationship(t, countResultsL2, rateResultsL2, 1.0, "Level 2 (sum)", tc.description)

			// Level 3: Final aggregation - rate() should return actual rates (counts / step_seconds)
			// Test both single-source and multi-source scenarios at final level since different
			// distributed architectures may feed the final aggregator differently:
			// - Single source: Multiple queriers -> Intermediate aggregator -> Query frontend
			// - Multi source: Multiple queriers -> Multiple intermediate aggregators -> Query frontend

			stepSeconds := float64(tc.req.Step) / float64(time.Second)

			// Scenario A: Final aggregation from single intermediate source
			countEvalL3, err := e.CompileMetricsQueryRangeNonRaw(countReq, traceql.AggregateModeFinal)
			require.NoError(t, err)
			countEvalL3.ObserveSeries(countResultsL2)
			countResultsL3 := countEvalL3.Results().ToProto(countReq)

			rateEvalL3, err := e.CompileMetricsQueryRangeNonRaw(rateReq, traceql.AggregateModeFinal)
			require.NoError(t, err)
			rateEvalL3.ObserveSeries(rateResultsL2)
			rateResultsL3 := rateEvalL3.Results().ToProto(rateReq)
			validateRateCountRelationship(t, countResultsL3, rateResultsL3, stepSeconds, "Level 3 (final-single)", tc.description)

			// Scenario B: Final aggregation from multiple intermediate sources
			countEvalL3Multi, err := e.CompileMetricsQueryRangeNonRaw(countReq, traceql.AggregateModeFinal)
			require.NoError(t, err)
			// Observe twice to simulate multiple sources
			countEvalL3Multi.ObserveSeries(countResultsL2)
			countEvalL3Multi.ObserveSeries(countResultsL2)
			countResultsL3Multi := countEvalL3Multi.Results().ToProto(countReq)

			rateEvalL3Multi, err := e.CompileMetricsQueryRangeNonRaw(rateReq, traceql.AggregateModeFinal)
			require.NoError(t, err)
			// Observe twice to simulate multiple sources
			rateEvalL3Multi.ObserveSeries(rateResultsL2)
			rateEvalL3Multi.ObserveSeries(rateResultsL2)
			rateResultsL3Multi := rateEvalL3Multi.Results().ToProto(rateReq)
			validateRateCountRelationship(t, countResultsL3Multi, rateResultsL3Multi, stepSeconds, "Level 3 (final-multi)", tc.description)
		})
	}
}

// validateRateCountRelationship validates that rate = count_over_time / step_seconds
func validateRateCountRelationship(t *testing.T, countResults, rateResults []*tempopb.TimeSeries, stepSeconds float64, level, description string) {
	t.Helper()

	require.Equal(t, len(countResults), len(rateResults), "Number of time series should match between count_over_time and rate")

	if len(countResults) == 0 {
		return
	}

	// For simplicity, we'll validate the first time series (they should have the same structure)
	countTS := countResults[0]
	rateTS := rateResults[0]

	require.Equal(t, len(countTS.Samples), len(rateTS.Samples), "Number of samples should match")

	for i, countSample := range countTS.Samples {
		rateSample := rateTS.Samples[i]
		require.Equal(t, countSample.TimestampMs, rateSample.TimestampMs, "Timestamps should match")

		// Calculate expected rate: count / step_seconds
		expectedRate := countSample.Value / stepSeconds

		// Handle zero values (rate should be 0 if count is 0)
		if countSample.Value == 0 {
			require.Equal(t, 0.0, rateSample.Value,
				"Rate should be 0 when count is 0 at timestamp %d (%s - %s)",
				countSample.TimestampMs, level, description)
		} else {
			// Use a small tolerance for floating point comparison
			tolerance := 0.000001
			require.InDelta(t, expectedRate, rateSample.Value, tolerance,
				"Rate should equal count_over_time / step_seconds at timestamp %d (%s - %s). Expected: %f, Got: %f, Count: %f, Step: %f",
				countSample.TimestampMs, level, description, expectedRate, rateSample.Value, countSample.Value, stepSeconds)
		}
	}
}

func sortTimeSeries(ts []*tempopb.TimeSeries) {
	sort.Slice(ts, func(i, j int) bool {
		if len(ts[i].Labels) != len(ts[j].Labels) {
			return len(ts[i].Labels) < len(ts[j].Labels)
		}

		for l := range ts[i].Labels {
			if ts[i].Labels[l].Key != ts[j].Labels[l].Key {
				return ts[i].Labels[l].Key < ts[j].Labels[l].Key
			}
			if ts[i].Labels[l].Value != ts[j].Labels[l].Value {
				return ts[i].Labels[l].Value.String() < ts[j].Labels[l].Value.String()
			}
		}

		// All equal
		return false
	})
}

var floatComparer = cmp.Comparer(func(x, y float64) bool {
	return math.Abs(x-y) < 1e-6
})

// filterTimeSeriesByLabel filters the time series to the ones including the given label name.
func filterTimeSeriesByLabel(ts []*tempopb.TimeSeries, labelName string) []*tempopb.TimeSeries {
	var targetTs []*tempopb.TimeSeries
	for _, ts := range ts {
		for _, l := range ts.Labels {
			if l.Key == labelName {
				targetTs = append(targetTs, ts)
				break
			}
		}
	}
	return targetTs
}
