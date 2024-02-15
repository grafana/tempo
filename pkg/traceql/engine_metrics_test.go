package traceql

import (
	"fmt"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

func TestDefaultQueryRangeStep(t *testing.T) {
	tc := []struct {
		start, end time.Time
		expected   time.Duration
	}{
		{time.Unix(0, 0), time.Unix(100, 0), time.Second},
		{time.Unix(0, 0), time.Unix(600, 0), 2 * time.Second},
		{time.Unix(0, 0), time.Unix(3600, 0), 15 * time.Second},
	}

	for _, c := range tc {
		require.Equal(t, c.expected, time.Duration(DefaultQueryRangeStep(uint64(c.start.UnixNano()), uint64(c.end.UnixNano()))))
	}
}

func TestStepRangeToIntervals(t *testing.T) {
	tc := []struct {
		start, end, step uint64
		expected         int
	}{
		{
			start:    0,
			end:      1,
			step:     1,
			expected: 2, // 0, 1, even multiple
		},
		{
			start:    0,
			end:      10,
			step:     3,
			expected: 4, // 0, 3, 6, 9
		},
	}

	for _, c := range tc {
		require.Equal(t, c.expected, IntervalCount(c.start, c.end, c.step))
	}
}

func TestTimestampOf(t *testing.T) {
	tc := []struct {
		interval, start, step uint64
		expected              uint64
	}{
		{
			expected: 0,
		},
		{
			interval: 2,
			start:    10,
			step:     3,
			expected: 16,
		},
	}

	for _, c := range tc {
		require.Equal(t, c.expected, TimestampOf(c.interval, c.start, c.step))
	}
}

func TestIntervalOf(t *testing.T) {
	tc := []struct {
		ts, start, end, step uint64
		expected             int
	}{
		{expected: -1},
		{
			ts:   0,
			end:  1,
			step: 1,
		},
		{
			ts:       10,
			end:      10,
			step:     1,
			expected: 10,
		},
	}

	for _, c := range tc {
		require.Equal(t, c.expected, IntervalOf(c.ts, c.start, c.end, c.step))
	}
}

func TestTimeRangeOverlap(t *testing.T) {
	tc := []struct {
		reqStart, reqEnd, dataStart, dataEnd uint64
		expected                             float64
	}{
		{1, 2, 3, 4, 0.0},   // No overlap
		{0, 10, 0, 10, 1.0}, // Perfect overlap
		{0, 10, 1, 2, 1.0},  // Request covers 100% of data
		{3, 8, 0, 10, 0.5},  // 50% in the middle
		{0, 10, 5, 15, 0.5}, // 50% of the start
		{5, 15, 0, 10, 0.5}, // 50% of the end
	}

	for _, c := range tc {
		actual := timeRangeOverlap(c.reqStart, c.reqEnd, c.dataStart, c.dataEnd)
		require.Equal(t, c.expected, actual)
	}
}

func TestCompileMetricsQueryRange(t *testing.T) {
	tc := map[string]struct {
		q           string
		start, end  uint64
		step        uint64
		expectedErr error
	}{
		"start": {
			expectedErr: fmt.Errorf("start required"),
		},
		"end": {
			start:       1,
			expectedErr: fmt.Errorf("end required"),
		},
		"range": {
			start:       2,
			end:         1,
			expectedErr: fmt.Errorf("end must be greater than start"),
		},
		"step": {
			start:       1,
			end:         2,
			expectedErr: fmt.Errorf("step required"),
		},
		"notmetrics": {
			start:       1,
			end:         2,
			step:        3,
			q:           "{}",
			expectedErr: fmt.Errorf("not a metrics query"),
		},
		"notsupported": {
			start:       1,
			end:         2,
			step:        3,
			q:           "{} | rate() by (.a,.b,.c,.d,.e,.f)",
			expectedErr: fmt.Errorf("compiling query: metrics group by 6 values not yet supported"),
		},
		"ok": {
			start: 1,
			end:   2,
			step:  3,
			q:     "{} | rate()",
		},
	}

	for n, c := range tc {
		t.Run(n, func(t *testing.T) {
			_, err := NewEngine().CompileMetricsQueryRange(&tempopb.QueryRangeRequest{
				Query: c.q,
				Start: c.start,
				End:   c.end,
				Step:  c.step,
			}, false, false)

			if c.expectedErr != nil {
				require.EqualError(t, err, c.expectedErr.Error())
			}
		})
	}
}

func TestCompileMetricsQueryRangeFetchSpansRequest(t *testing.T) {
	tc := map[string]struct {
		q           string
		shardID     uint32
		shardCount  uint32
		dedupe      bool
		expectedReq FetchSpansRequest
	}{
		"minimal": {
			q: "{} | rate()",
			expectedReq: FetchSpansRequest{
				AllConditions: true,
				Conditions: []Condition{
					{
						// In this case start time is in the first pass
						Attribute: IntrinsicSpanStartTimeAttribute,
					},
				},
			},
		},
		"dedupe": {
			q:      "{} | rate()",
			dedupe: true,
			expectedReq: FetchSpansRequest{
				AllConditions: true,
				Conditions: []Condition{
					{
						Attribute: IntrinsicSpanStartTimeAttribute,
					},
					{
						Attribute: IntrinsicTraceIDAttribute, // Required for dedupe
					},
				},
			},
		},
		"secondPass": {
			q:          "{duration > 10s} | rate() by (resource.cluster)",
			shardID:    123,
			shardCount: 456,
			expectedReq: FetchSpansRequest{
				AllConditions: true,
				ShardID:       123,
				ShardCount:    456,
				Conditions: []Condition{
					{
						Attribute: IntrinsicDurationAttribute,
						Op:        OpGreater,
						Operands:  Operands{NewStaticDuration(10 * time.Second)},
					},
					{
						Attribute: IntrinsicTraceIDAttribute, // Required for sharding
					},
				},
				SecondPassConditions: []Condition{
					{
						// Group-by attributes (non-intrinsic) must be in the second pass
						Attribute: NewScopedAttribute(AttributeScopeResource, false, "cluster"),
					},
					{
						// Since there is already a second pass then span start time isn't optimized to the first pass.
						Attribute: IntrinsicSpanStartTimeAttribute,
					},
				},
			},
		},
		"optimizations": {
			q:          "{duration > 10s} | rate() by (name, resource.service.name)",
			shardID:    123,
			shardCount: 456,
			expectedReq: FetchSpansRequest{
				AllConditions: true,
				ShardID:       123,
				ShardCount:    456,
				Conditions: []Condition{
					{
						Attribute: IntrinsicDurationAttribute,
						Op:        OpGreater,
						Operands:  Operands{NewStaticDuration(10 * time.Second)},
					},
					{
						Attribute: IntrinsicTraceIDAttribute, // Required for sharding
					},
					{
						// Intrinsic moved to first pass
						Attribute: IntrinsicNameAttribute,
					},
					{
						// Resource service name is treated as an intrinsic and moved to the first pass
						Attribute: NewScopedAttribute(AttributeScopeResource, false, "service.name"),
					},
					{
						Attribute: IntrinsicSpanStartTimeAttribute,
					},
				},
			},
		},
	}

	for n, tc := range tc {
		t.Run(n, func(t *testing.T) {
			eval, err := NewEngine().CompileMetricsQueryRange(&tempopb.QueryRangeRequest{
				Query:      tc.q,
				ShardID:    tc.shardID,
				ShardCount: tc.shardCount,
				Start:      1,
				End:        2,
				Step:       3,
			}, tc.dedupe, false)
			require.NoError(t, err)

			// Nil out func to Equal works
			eval.storageReq.SecondPass = nil
			require.Equal(t, tc.expectedReq, *eval.storageReq)
		})
	}
}
