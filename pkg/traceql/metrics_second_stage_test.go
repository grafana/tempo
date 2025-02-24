package traceql

import (
	"math"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsSecondStageTopKBottomK(t *testing.T) {
	testCases := []struct {
		name     string
		op       SecondStageOp
		limit    int
		input    []*tempopb.TimeSeries
		expected []*tempopb.TimeSeries
	}{
		{
			name:  "topk basic",
			op:    OpTopK,
			limit: 2,
			input: []*tempopb.TimeSeries{
				makeTimeSeries(1.0, 2.0, 3.0), // avg: 2.0
				makeTimeSeries(4.0, 5.0, 6.0), // avg: 5.0
				makeTimeSeries(7.0, 8.0, 9.0), // avg: 8.0
			},
			expected: []*tempopb.TimeSeries{
				makeTimeSeries(7.0, 8.0, 9.0), // highest
				makeTimeSeries(4.0, 5.0, 6.0), // second highest
			},
		},
		{
			name:  "bottomk basic",
			op:    OpBottomK,
			limit: 2,
			input: []*tempopb.TimeSeries{
				makeTimeSeries(1.0, 2.0, 3.0), // avg: 2.0
				makeTimeSeries(4.0, 5.0, 6.0), // avg: 5.0
				makeTimeSeries(7.0, 8.0, 9.0), // avg: 8.0
			},
			expected: []*tempopb.TimeSeries{
				makeTimeSeries(1.0, 2.0, 3.0), // lowest
				makeTimeSeries(4.0, 5.0, 6.0), // second lowest
			},
		},
		{
			name:  "topk with NaN values",
			op:    OpTopK,
			limit: 2,
			input: []*tempopb.TimeSeries{
				makeTimeSeriesWithNaN(1.0, 2.0, 3.0),        // avg: 2.0
				makeTimeSeriesWithNaN(4.0, float64NaN, 6.0), // avg: 5.0 (ignoring NaN)
				makeTimeSeriesWithNaN(7.0, 8.0, 9.0),        // avg: 8.0
			},
			expected: []*tempopb.TimeSeries{
				makeTimeSeriesWithNaN(7.0, 8.0, 9.0),
				makeTimeSeriesWithNaN(4.0, float64NaN, 6.0),
			},
		},
		{
			name:  "limit larger than input",
			op:    OpTopK,
			limit: 5,
			input: []*tempopb.TimeSeries{
				makeTimeSeries(1.0, 2.0, 3.0),
				makeTimeSeries(4.0, 5.0, 6.0),
			},
			expected: []*tempopb.TimeSeries{
				makeTimeSeries(4.0, 5.0, 6.0),
				makeTimeSeries(1.0, 2.0, 3.0),
			},
		},
		{
			name:     "empty input",
			op:       OpTopK,
			limit:    2,
			input:    []*tempopb.TimeSeries{},
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stage := &MetricsSecondStage{
				op:    tc.op,
				limit: tc.limit,
			}

			// Test initialization
			stage.init(nil, AggregateMode(0))
			assert.Nil(t, stage.input)

			// Test series observation
			stage.observeSeries(tc.input)
			assert.Equal(t, tc.input, stage.input)

			// Test result
			result := stage.result()
			if tc.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestMetricsSecondStageValidation(t *testing.T) {
	testCases := []struct {
		name        string
		limit       int
		op          SecondStageOp
		expectError error
	}{
		{
			name:        "valid limit - topk",
			limit:       1,
			op:          OpTopK,
			expectError: nil,
		},
		{
			name:        "zero limit - topk",
			limit:       0,
			op:          OpTopK,
			expectError: errInvalidLimit,
		},
		{
			name:        "negative limit - topk",
			limit:       -1,
			op:          OpTopK,
			expectError: errInvalidLimit,
		},
		{
			name:        "valid limit - bottomk",
			limit:       1,
			op:          OpBottomK,
			expectError: nil,
		},
		{
			name:        "zero limit - bottomk",
			limit:       0,
			op:          OpBottomK,
			expectError: errInvalidLimit,
		},
		{
			name:        "negative limit - bottomk",
			limit:       -1,
			op:          OpBottomK,
			expectError: errInvalidLimit,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stage := &MetricsSecondStage{op: tc.op, limit: tc.limit}

			err := stage.validate()
			require.Equal(t, tc.expectError, err)
		})
	}
}

// Helper functions
func makeTimeSeries(values ...float64) *tempopb.TimeSeries {
	samples := make([]tempopb.Sample, len(values))
	for i, v := range values {
		samples[i] = tempopb.Sample{
			TimestampMs: time.Now().UnixMilli(),
			Value:       v,
		}
	}
	return &tempopb.TimeSeries{
		Samples: samples,
	}
}

var float64NaN = math.NaN()

func makeTimeSeriesWithNaN(values ...float64) *tempopb.TimeSeries {
	samples := make([]tempopb.Sample, len(values))
	for i, v := range values {
		samples[i] = tempopb.Sample{
			TimestampMs: time.Now().UnixMilli(),
			Value:       v,
		}
	}
	return &tempopb.TimeSeries{
		Samples: samples,
	}
}
