package tempodb

import (
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
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
					{TimestampMs: 0, Value: 14},      // Raw count: 14 spans
					{TimestampMs: 15_000, Value: 15}, // Raw count: 15 spans
					{TimestampMs: 30_000, Value: 15}, // Raw count: 15 spans
					{TimestampMs: 45_000, Value: 5},  // Raw count: 5 spans
					{TimestampMs: 60_000, Value: 0},  // Raw count: 0 spans
				},
			},
		},
		expectedL2: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "rate")},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 2 * 15.0 / 15.0},
					{TimestampMs: 30_000, Value: 2 * 1.0},
					{TimestampMs: 45_000, Value: 2 * 1.0},
					{TimestampMs: 60_000, Value: 2 * 5.0 / 15.0},
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
					{TimestampMs: 15_000, Value: 7.0 / 15.0}, // Interval (0, 15], 7 spans at 2, 4, 6, 8, 10, 12, 14
					{TimestampMs: 30_000, Value: 8.0 / 15.0}, // Interval (15, 30], 8 spans at 16, 18, 20, 22, 24, 26, 28, 30
					{TimestampMs: 45_000, Value: 7.0 / 15.0}, // Interval (30, 45], 7 spans at 32, 34, 36, 38, 40, 42, 44
					{TimestampMs: 60_000, Value: 3.0 / 15.0}, // Interval (45, 50], 3 spans at 46, 48, 50
				},
			},
		},
		expectedL2: []*tempopb.TimeSeries{
			{
				Labels: []common_v1.KeyValue{tempopb.MakeKeyValueString("__name__", "rate")},
				Samples: []tempopb.Sample{
					{TimestampMs: 15_000, Value: 2 * 7.0 / 15.0},
					{TimestampMs: 30_000, Value: 2 * 8.0 / 15.0},
					{TimestampMs: 45_000, Value: 2 * 7.0 / 15.0},
					{TimestampMs: 60_000, Value: 2 * 3.0 / 15.0},
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
