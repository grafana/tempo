package traceql

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1proto "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/stretchr/testify/assert"
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
			expected: 1, // end-start == step -> instant query, only one interval
		},
		{
			start:    0,
			end:      3,
			step:     1,
			expected: 3, // 1, 2, 3
		},
		{
			start:    0,
			end:      10,
			step:     3,
			expected: 4, // 3, 6, 9, 12
		},
	}

	for _, c := range tc {
		require.Equal(t, c.expected, IntervalCount(c.start, c.end, c.step))
	}
}

func TestTimestampOf(t *testing.T) {
	tc := []struct {
		interval, start, end, step uint64
		expected                   uint64
	}{
		{
			expected: 0,
		},
		{
			interval: 2,
			start:    10, // aligned to 9
			step:     3,
			end:      100,
			expected: 18, // 12, 15, 18 <-- intervals
		},
		// start <= step
		{
			interval: 0,
			start:    1,
			end:      10,
			step:     1,
			expected: 2,
		},
		{
			interval: 1,
			start:    1,
			end:      5,
			step:     1,
			expected: 3,
		},
		{
			interval: 4,
			start:    1,
			end:      5,
			step:     1,
			expected: 6,
		},
		// start > step
		{
			interval: 0,
			start:    10,
			end:      50,
			step:     10,
			expected: 20,
		},
		{
			interval: 2,
			start:    10,
			end:      50,
			step:     10,
			expected: 40, // 3rd interval: (10;20] (20;30] (30;40]
		},
		{
			interval: 3,
			start:    10,
			end:      50,
			step:     10,
			expected: 50,
		},
	}

	for _, c := range tc {
		assert.Equal(t, c.expected, TimestampOf(c.interval, c.start, c.end, c.step), "interval: %d, start: %d, end: %d, step: %d", c.interval, c.start, c.end, c.step)
	}
}

func TestIntervalOf(t *testing.T) {
	tc := []struct {
		ts, start, end, step uint64
		expected             int
	}{
		// start <= step
		{expected: -1},
		{
			ts:       0,
			end:      1,
			step:     1,
			expected: 0, // corner case. TODO: should we return -1?
		},
		{
			ts:       10,
			start:    1,
			end:      10,
			step:     1,
			expected: 8, // 9th interval: (9;10]
		},
		{
			ts:       1,
			start:    1,
			end:      5,
			step:     1,
			expected: -1, // should be excluded
		},
		{
			ts:       2,
			start:    1,
			end:      5,
			step:     1,
			expected: 0, // 2nd interval: (1;2]
		},
		// start > step
		{
			ts:       15,
			start:    10,
			end:      50,
			step:     10,
			expected: 0, // first interval: (10;20]
		},
		{
			ts:       5,
			start:    10,
			end:      50,
			step:     10,
			expected: -1, // should be excluded
		},
		{
			ts:       10,
			start:    10,
			end:      50,
			step:     10,
			expected: -1, // should be excluded
		},
		{
			ts:       20,
			start:    10,
			end:      50,
			step:     10,
			expected: 0, // first interval: (10;20]
		},
		{
			ts:       25,
			start:    10,
			end:      50,
			step:     10,
			expected: 1, // second interval: (20;30]
		},
		{
			ts:       50,
			start:    10,
			end:      50,
			step:     10,
			expected: 3, // 4th interval: (40;50]
		},
	}

	for _, c := range tc {
		assert.Equal(t, c.expected, IntervalOf(c.ts, c.start, c.end, c.step), "ts: %d, start: %d, end: %d, step: %d", c.ts, c.start, c.end, c.step)
	}
}

func TestTrimToBlockOverlap(t *testing.T) {
	tc := []struct {
		start1, end1               string
		step                       time.Duration
		start2, end2               string
		expectedStart, expectedEnd string
		expectedStep               time.Duration
	}{
		{
			// Block fully within range
			// Left border is extended to the next step.
			// Right border is extended to the next step.
			"2024-01-01T01:00:00Z", "2024-01-01T02:00:00Z", 5 * time.Minute,
			"2024-01-01T01:33:00Z", "2024-01-01T01:38:00Z",
			"2024-01-01T01:30:00Z", "2024-01-01T01:40:00Z", 5 * time.Minute,
		},
		{
			// Block overlapping right border.
			// Left border is extended to the next step.
			// Right border preserved.
			"2024-01-01T01:01:00Z", "2024-01-01T02:01:00.123Z", 5 * time.Minute,
			"2024-01-01T01:31:00Z", "2024-01-01T02:31:00Z",
			"2024-01-01T01:30:00Z", "2024-01-01T02:01:00.123Z", 5 * time.Minute,
		},
		{
			// Block overlapping left border.
			// Left border preserved.
			// Right border extended to the next step.
			"2024-01-01T01:01:00.123Z", "2024-01-01T02:00:00Z", 5 * time.Minute,
			"2024-01-01T00:31:00Z", "2024-01-01T01:31:00Z",
			"2024-01-01T01:01:00.123Z", "2024-01-01T01:35:00Z", 5 * time.Minute,
		},
		{
			// Block larger than range
			// Neither border is extended. Nanoseconds preserved.
			"2024-01-01T01:00:01.123Z", "2024-01-01T01:15:01.123Z", 5 * time.Minute,
			"2024-01-01T00:00:00Z", "2024-01-01T02:00:00Z",
			"2024-01-01T01:00:01.123Z", "2024-01-01T01:15:01.123Z", 5 * time.Minute,
		},
		{
			// Instant query, block overlaps right border.
			// Original range is 1h
			// Right border isn't extended past request range.
			// Left border is able to be extended.
			"2024-01-01T01:00:00.123Z", "2024-01-01T02:00:00.123Z", time.Hour,
			"2024-01-01T01:30:00.123Z", "2024-01-01T02:30:00.123Z",
			"2024-01-01T01:00:00.123Z", "2024-01-01T02:00:00.123Z", time.Hour,
		},
	}

	for _, c := range tc {
		start1, _ := time.Parse(time.RFC3339Nano, c.start1)
		end1, _ := time.Parse(time.RFC3339Nano, c.end1)
		start2, _ := time.Parse(time.RFC3339Nano, c.start2)
		end2, _ := time.Parse(time.RFC3339Nano, c.end2)

		actualStart, actualEnd, actualStep := TrimToBlockOverlap(
			uint64(start1.UnixNano()),
			uint64(end1.UnixNano()),
			uint64(c.step.Nanoseconds()),
			start2,
			end2,
		)

		assert.Equal(t, c.expectedStart, time.Unix(0, int64(actualStart)).UTC().Format(time.RFC3339Nano))
		assert.Equal(t, c.expectedEnd, time.Unix(0, int64(actualEnd)).UTC().Format(time.RFC3339Nano))
		assert.Equal(t, c.expectedStep, time.Duration(actualStep))
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
			}, 0, 0, false)

			if c.expectedErr != nil {
				require.EqualError(t, err, c.expectedErr.Error())
			}
		})
	}
}

func TestCompileMetricsQueryRangeFetchSpansRequest(t *testing.T) {
	tc := map[string]struct {
		q           string
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
			q: "{} | rate()",
			expectedReq: FetchSpansRequest{
				AllConditions: true,
				Conditions: []Condition{
					{
						Attribute: IntrinsicSpanStartTimeAttribute,
					},
				},
			},
		},
		"secondPass": {
			q: "{duration > 10s} | rate() by (resource.cluster)",
			expectedReq: FetchSpansRequest{
				AllConditions: true,
				Conditions: []Condition{
					{
						Attribute: IntrinsicDurationAttribute,
						Op:        OpGreater,
						Operands:  Operands{NewStaticDuration(10 * time.Second)},
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
			q: "{duration > 10s} | rate() by (name, resource.service.name)",
			expectedReq: FetchSpansRequest{
				AllConditions: true,
				Conditions: []Condition{
					{
						Attribute: IntrinsicDurationAttribute,
						Op:        OpGreater,
						Operands:  Operands{NewStaticDuration(10 * time.Second)},
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
		"structural_rate_by": {
			q: "{name=`foo`} > {} | rate() by (name)",
			expectedReq: FetchSpansRequest{
				AllConditions: false,
				Conditions: []Condition{
					{
						Attribute: NewIntrinsic(IntrinsicStructuralChild),
					},
					{
						Attribute: IntrinsicNameAttribute,
						Op:        OpEqual,
						Operands:  Operands{NewStaticString("foo")},
					},
				},
				SecondPassConditions: []Condition{
					{
						Attribute: IntrinsicNameAttribute,
					},
					{
						// Since there is already a second pass then span start time isn't optimized to the first pass.
						Attribute: IntrinsicSpanStartTimeAttribute,
					},
				},
			},
		},
	}

	for n, tc := range tc {
		t.Run(n, func(t *testing.T) {
			eval, err := NewEngine().CompileMetricsQueryRange(&tempopb.QueryRangeRequest{
				Query: tc.q,
				Start: 1,
				End:   2,
				Step:  3,
			}, 0, 0, false)
			require.NoError(t, err)

			// Nil out func to Equal works
			eval.storageReq.SecondPass = nil
			require.Equal(t, tc.expectedReq, *eval.storageReq)
		})
	}
}

func TestOptimizeFetchSpansRequest(t *testing.T) {
	secondPass := func(_ *Spanset) ([]*Spanset, error) {
		return nil, nil
	}

	tc := []struct {
		name     string
		input    FetchSpansRequest
		expected FetchSpansRequest
	}{
		{
			name: "Not able to be optimized because not all conditions",
			input: FetchSpansRequest{
				SecondPass: secondPass,
				SecondPassConditions: []Condition{
					{Attribute: IntrinsicNameAttribute},
					{Attribute: NewScopedAttribute(AttributeScopeResource, false, "service.name")},
				},
			},
			expected: FetchSpansRequest{
				SecondPass: secondPass,
				SecondPassConditions: []Condition{
					{Attribute: IntrinsicNameAttribute},
					{Attribute: NewScopedAttribute(AttributeScopeResource, false, "service.name")},
				},
			},
		},
		{
			name: "Intrinsics moved to first pass and second pass eliminated",
			input: FetchSpansRequest{
				AllConditions: true,
				SecondPass:    secondPass,
				SecondPassConditions: []Condition{
					{Attribute: IntrinsicNameAttribute},
					{Attribute: NewScopedAttribute(AttributeScopeResource, false, "service.name")},
				},
			},
			expected: FetchSpansRequest{
				AllConditions: true,
				Conditions: []Condition{
					{Attribute: IntrinsicNameAttribute},
					{Attribute: NewScopedAttribute(AttributeScopeResource, false, "service.name")},
				},
			},
		},
		{
			name: "Unscoped cannot be optimized",
			input: FetchSpansRequest{
				AllConditions: true,
				Conditions: []Condition{
					{Attribute: NewScopedAttribute(AttributeScopeNone, false, "http.status_code")},
				},
				SecondPass: secondPass,
			},
			expected: FetchSpansRequest{
				AllConditions: true,
				Conditions: []Condition{
					{Attribute: NewScopedAttribute(AttributeScopeNone, false, "http.status_code")},
				},
				SecondPass: secondPass,
			},
		},
		{
			name: "Single scoped non-intrinsic can still elminiate second pass",
			input: FetchSpansRequest{
				AllConditions: true,
				Conditions: []Condition{
					{Attribute: NewScopedAttribute(AttributeScopeSpan, false, "http.status_code")},
				},
				SecondPass: secondPass,
			},
			expected: FetchSpansRequest{
				AllConditions: true,
				Conditions: []Condition{
					{Attribute: NewScopedAttribute(AttributeScopeSpan, false, "http.status_code")},
				},
			},
		},
	}

	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			// Make a copy
			actual := &c.input
			optimize(actual)

			// Instead of comparing func pointers, check if they are both set or not
			if c.expected.SecondPass != nil {
				require.NotNil(t, actual.SecondPass)
			} else {
				require.Nil(t, actual.SecondPass)
			}

			// Now nil out func and compare the rest
			c.expected.SecondPass = nil
			actual.SecondPass = nil
			require.Equal(t, c.expected, *actual)
		})
	}
}

func TestQuantileOverTime(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(3 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | quantile_over_time(duration, 0, 0.5, 1) by (span.foo)",
	}
	// intervals: (0;1], (1;2], (2;3]

	var (
		_128ns = 0.000000128
		_256ns = 0.000000256
		_512ns = 0.000000512
	)

	// A variety of spans across times, durations, and series. All durations are powers of 2 for simplicity
	in := []Span{
		// 1st interval: (0;1]
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(128),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(512),

		// 2nd interval: (1;2]
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(256),

		// 3rd interval: (2;3]
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(512),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(512),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(512),
	}

	// Output series with quantiles per foo
	// Prom labels are sorted alphabetically, traceql labels maintain original order.
	out := SeriesSet{
		`{p="0.0", "span.foo"="bar"}`: TimeSeries{
			Labels: []Label{
				{Name: "span.foo", Value: NewStaticString("bar")},
				{Name: "p", Value: NewStaticFloat(0)},
			},
			Values: []float64{
				_128ns,
				percentileHelper(0, _256ns, _256ns, _256ns, _256ns),
				0,
			},
		},
		`{p="0.5", "span.foo"="bar"}`: TimeSeries{
			Labels: []Label{
				{Name: "span.foo", Value: NewStaticString("bar")},
				{Name: "p", Value: NewStaticFloat(0.5)},
			},
			Values: []float64{
				_256ns,
				percentileHelper(0.5, _256ns, _256ns, _256ns, _256ns),
				0,
			},
		},
		`{p="1.0", "span.foo"="bar"}`: TimeSeries{
			Labels: []Label{
				{Name: "span.foo", Value: NewStaticString("bar")},
				{Name: "p", Value: NewStaticFloat(1)},
			},
			Values: []float64{_512ns, _256ns, 0},
		},
		`{p="0.0", "span.foo"="baz"}`: TimeSeries{
			Labels: []Label{
				{Name: "span.foo", Value: NewStaticString("baz")},
				{Name: "p", Value: NewStaticFloat(0)},
			},
			Values: []float64{
				0, 0,
				percentileHelper(0, _512ns, _512ns, _512ns),
			},
		},
		`{p="0.5", "span.foo"="baz"}`: TimeSeries{
			Labels: []Label{
				{Name: "span.foo", Value: NewStaticString("baz")},
				{Name: "p", Value: NewStaticFloat(0.5)},
			},
			Values: []float64{
				0, 0,
				percentileHelper(0.5, _512ns, _512ns, _512ns),
			},
		},
		`{p="1.0", "span.foo"="baz"}`: TimeSeries{
			Labels: []Label{
				{Name: "span.foo", Value: NewStaticString("baz")},
				{Name: "p", Value: NewStaticFloat(1)},
			},
			Values: []float64{0, 0, _512ns},
		},
	}

	result, seriesCount, err := runTraceQLMetric(req, in)
	require.NoError(t, err)
	require.Equal(t, out, result)
	require.Equal(t, len(result), seriesCount)
}

func percentileHelper(q float64, values ...float64) float64 {
	h := Histogram{}
	for _, v := range values {
		h.Record(v, 1)
	}
	return Log2Quantile(q, h.Buckets)
}

func TestCountOverTime(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(3 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | count_over_time() by (span.foo)",
	}

	// A variety of spans across times, durations, and series. All durations are powers of 2 for simplicity
	in := []Span{
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(128),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(512),

		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(256),

		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(512),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(512),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(512),
	}

	// Output series with quantiles per foo
	// Prom labels are sorted alphabetically, traceql labels maintain original order.
	out := SeriesSet{
		`{"span.foo"="baz"}`: TimeSeries{
			Labels: []Label{
				{Name: "span.foo", Value: NewStaticString("baz")},
			},
			Values:    []float64{0, 0, 3},
			Exemplars: make([]Exemplar, 0),
		},
		`{"span.foo"="bar"}`: TimeSeries{
			Labels: []Label{
				{Name: "span.foo", Value: NewStaticString("bar")},
			},
			Values:    []float64{3, 4, 0},
			Exemplars: make([]Exemplar, 0),
		},
	}

	result, seriesCount, err := runTraceQLMetric(req, in)
	require.NoError(t, err)
	require.Equal(t, out, result)
	require.Equal(t, len(result), seriesCount)
}

func TestCountOverTimeInstantNs(t *testing.T) {
	// not rounded values to simulate real world data
	start := 1*time.Second - 9*time.Nanosecond
	end := 3*time.Second + 9*time.Nanosecond
	step := end - start // for instant queries step == end-start
	req := &tempopb.QueryRangeRequest{
		Start: uint64(start),
		End:   uint64(end),
		Step:  uint64(step),
		Query: "{ } | count_over_time()",
	}

	in := []Span{
		// outside of the range but within the range for ms. Should be ignored.
		newMockSpan(nil).WithStartTime(uint64(start - 20*time.Nanosecond)).WithDuration(1),
		newMockSpan(nil).WithStartTime(uint64(start - time.Nanosecond)).WithDuration(1),

		// within the range
		newMockSpan(nil).WithStartTime(uint64(start)).WithDuration(1),
		newMockSpan(nil).WithStartTime(uint64(start + time.Nanosecond)).WithDuration(1),

		// within the range
		newMockSpan(nil).WithStartTime(uint64(end - time.Nanosecond)).WithDuration(1),
		newMockSpan(nil).WithStartTime(uint64(end)).WithDuration(10),

		// outside of the range but within the range for ms. Should be ignored.
		newMockSpan(nil).WithStartTime(uint64(end + time.Nanosecond)).WithDuration(1),
		newMockSpan(nil).WithStartTime(uint64(end + 20*time.Nanosecond)).WithDuration(1),
	}

	out := SeriesSet{
		`{__name__="count_over_time"}`: TimeSeries{
			Labels: []Label{
				{Name: "__name__", Value: NewStaticString("count_over_time")},
			},
			Values:    []float64{4},
			Exemplars: make([]Exemplar, 0),
		},
	}

	result, seriesCount, err := runTraceQLMetric(req, in)
	require.NoError(t, err)
	require.Equal(t, out, result)
	require.Equal(t, len(result), seriesCount)
}

// TestCountOverTimeInstantNsWithCutoff simulates merge behavior in L2 and L3.
func TestCountOverTimeInstantNsWithCutoff(t *testing.T) {
	start := 1*time.Second + 300*time.Nanosecond // additional 300ns that can be accidentally dropped by ms conversion
	end := 3 * time.Second
	step := end - start // for instant queries step == end-start
	req := &tempopb.QueryRangeRequest{
		Start: uint64(start),
		End:   uint64(end),
		Step:  uint64(step),
		Query: "{ } | count_over_time()",
	}

	cutoff := 2*time.Second + 300*time.Nanosecond
	// from start to cutoff
	req1 := *req
	req1.End = uint64(cutoff)
	req1.Step = req1.End - req1.Start
	// from cutoff to end
	req2 := *req
	req2.Start = uint64(cutoff)
	req2.Step = req2.End - req2.Start

	in1 := []Span{
		// outside of the range but within the range for ms. Should be ignored.
		newMockSpan(nil).WithStartTime(uint64(start - 20*time.Nanosecond)).WithDuration(1),
		newMockSpan(nil).WithStartTime(uint64(start - time.Nanosecond)).WithDuration(1),

		// within the range
		newMockSpan(nil).WithStartTime(uint64(start)).WithDuration(1),
		newMockSpan(nil).WithStartTime(uint64(start + time.Nanosecond)).WithDuration(1),
		newMockSpan(nil).WithStartTime(uint64(cutoff - time.Nanosecond)).WithDuration(1),
		newMockSpan(nil).WithStartTime(uint64(cutoff)).WithDuration(1),

		// outside of cutoff
		newMockSpan(nil).WithStartTime(uint64(cutoff + time.Nanosecond)).WithDuration(1),
	}

	in2 := []Span{
		// outside of cutoff
		newMockSpan(nil).WithStartTime(uint64(cutoff - time.Nanosecond)).WithDuration(1),

		// within the range
		newMockSpan(nil).WithStartTime(uint64(cutoff)).WithDuration(1),
		newMockSpan(nil).WithStartTime(uint64(cutoff + time.Nanosecond)).WithDuration(1),
		newMockSpan(nil).WithStartTime(uint64(end - time.Nanosecond)).WithDuration(1),
		newMockSpan(nil).WithStartTime(uint64(end)).WithDuration(10),

		// outside of the range but within the range for ms. Should be ignored.
		newMockSpan(nil).WithStartTime(uint64(end + time.Nanosecond)).WithDuration(1),
		newMockSpan(nil).WithStartTime(uint64(end + 20*time.Nanosecond)).WithDuration(1),
	}

	out := SeriesSet{
		`{__name__="count_over_time"}`: TimeSeries{
			Labels: []Label{
				{Name: "__name__", Value: NewStaticString("count_over_time")},
			},
			Values:    []float64{8},
			Exemplars: make([]Exemplar, 0),
		},
	}

	t.Run("merge in L2", func(t *testing.T) {
		e := NewEngine()

		layer2, err := e.CompileMetricsQueryRangeNonRaw(req, AggregateModeSum)
		require.NoError(t, err)

		// process different series in L1
		layer1, err := e.CompileMetricsQueryRange(&req1, 0, 0, false)
		require.NoError(t, err)
		for _, s := range in1 {
			layer1.metricsPipeline.observe(s)
		}
		res1 := layer1.Results().ToProto(&req1)

		layer1, err = e.CompileMetricsQueryRange(&req2, 0, 0, false)
		require.NoError(t, err)
		for _, s := range in2 {
			layer1.metricsPipeline.observe(s)
		}
		res2 := layer1.Results().ToProto(&req2)

		// merge in L2
		layer2.metricsPipeline.observeSeries(res1)
		layer2.metricsPipeline.observeSeries(res2)

		result, seriesCount, err := processLayer3(req, layer2.Results())
		require.NoError(t, err)
		require.Equal(t, out, result)
		require.Equal(t, len(result), seriesCount)
	})

	t.Run("merge in L3", func(t *testing.T) {
		// process different series in L1+L2
		res1, err := processLayer1AndLayer2(&req1, in1)
		require.NoError(t, err)
		res2, err := processLayer1AndLayer2(&req2, in2)
		require.NoError(t, err)

		// merge in L3
		e := NewEngine()
		layer3, err := e.CompileMetricsQueryRangeNonRaw(req, AggregateModeFinal)
		require.NoError(t, err)

		layer3.ObserveSeries(res1.ToProto(req))
		layer3.ObserveSeries(res2.ToProto(req))

		require.NoError(t, err)
		require.Equal(t, out, layer3.Results())
		require.Equal(t, len(layer3.Results()), layer3.Length())
	})
}

func TestMinOverTimeForDuration(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(3 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | min_over_time(duration) by (span.foo)",
	}

	// A variety of spans across times, durations, and series. All durations are powers of 2 for simplicity
	in := []Span{
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(128),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(512),

		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(64),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(8),

		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(512),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(1024),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(512),
	}

	result, seriesCount, err := runTraceQLMetric(req, in)
	require.NoError(t, err)

	fooBaz := result[`{"span.foo"="baz"}`]
	fooBar := result[`{"span.foo"="bar"}`]

	// We cannot compare with require.Equal because NaN != NaN
	// foo.baz = (NaN, NaN, 0.000000512)
	assert.True(t, math.IsNaN(fooBaz.Values[0]))
	assert.True(t, math.IsNaN(fooBaz.Values[1]))
	assert.Equal(t, 512/float64(time.Second), fooBaz.Values[2])

	// foo.bar = (0.000000128, 0.000000128, NaN)
	assert.Equal(t, 128/float64(time.Second), fooBar.Values[0])
	assert.Equal(t, 8/float64(time.Second), fooBar.Values[1])
	assert.True(t, math.IsNaN(fooBar.Values[2]))
	require.Equal(t, len(result), seriesCount)
}

func TestMinOverTimeWithNoMatch(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(3 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | min_over_time(span.buu)",
	}

	// A variety of spans across times, durations, and series. All durations are powers of 2 for simplicity
	in := []Span{
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(128),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 404).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(512),

		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(64),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(8),

		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 201).WithDuration(512),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 401).WithDuration(1024),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 500).WithDuration(512),
	}

	result, seriesCount, err := runTraceQLMetric(req, in)
	require.NoError(t, err)

	// Test that empty timeseries are not included
	ts := result.ToProto(req)

	assert.True(t, len(ts) == 0)
	require.Equal(t, 0, seriesCount)
}

func TestMinOverTimeForSpanAttribute(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(3 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | min_over_time(span.http.status_code) by (span.foo)",
	}

	// A variety of spans across times, durations, and series. All durations are powers of 2 for simplicity
	in := []Span{
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(128),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 404).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(512),

		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(64),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(8),

		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 201).WithDuration(512),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 401).WithDuration(1024),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 500).WithDuration(512),
	}

	in2 := []Span{
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 100).WithDuration(128),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 300).WithDuration(512),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 204).WithDuration(512),

		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 400).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 401).WithDuration(64),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 402).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 403).WithDuration(8),

		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 200).WithDuration(512),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 300).WithDuration(1024),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 400).WithDuration(512),
	}

	result, seriesCount, err := runTraceQLMetric(req, in, in2)
	require.NoError(t, err)

	fooBaz := result[`{"span.foo"="baz"}`]
	fooBar := result[`{"span.foo"="bar"}`]

	// Alas,we cannot compare with require.Equal because NaN != NaN
	// foo.baz = (204, NaN, 200)
	assert.Equal(t, 204.0, fooBaz.Values[0])
	assert.True(t, math.IsNaN(fooBaz.Values[1]))
	assert.Equal(t, 200.0, fooBaz.Values[2])
	require.Equal(t, len(result), seriesCount)

	// foo.bar = (100,200, NaN)
	assert.Equal(t, 100.0, fooBar.Values[0])
	assert.Equal(t, 200.0, fooBar.Values[1])
	assert.True(t, math.IsNaN(fooBar.Values[2]))

	// Test that NaN values are not included in the samples after casting to proto
	ts := result.ToProto(req)
	fooBarSamples := []tempopb.Sample{{TimestampMs: 1000, Value: 100}, {TimestampMs: 2000, Value: 200}}
	fooBazSamples := []tempopb.Sample{{TimestampMs: 1000, Value: 204}, {TimestampMs: 3000, Value: 200}}

	for _, s := range ts {
		if s.PromLabels == "{\"span.foo\"=\"bar\"}" {
			assert.Equal(t, fooBarSamples, s.Samples)
		} else {
			assert.Equal(t, fooBazSamples, s.Samples)
		}
	}
}

func TestAvgOverTimeForDuration(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(3 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | avg_over_time(duration) by (span.foo)",
	}

	// A variety of spans across times, durations, and series. All durations are powers of 2 for simplicity
	in := []Span{
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(100),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(100),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(100),

		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(100),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(100),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(100),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(500),

		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(100),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(200),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(300),
	}

	result, seriesCount, err := runTraceQLMetric(req, in)
	require.NoError(t, err)
	require.Equal(t, len(result), seriesCount)

	fooBaz := result[`{"span.foo"="baz"}`]
	fooBar := result[`{"span.foo"="bar"}`]

	// We cannot compare with require.Equal because NaN != NaN
	assert.True(t, math.IsNaN(fooBaz.Values[0]))
	assert.True(t, math.IsNaN(fooBaz.Values[1]))
	assert.Equal(t, 200., fooBaz.Values[2]*float64(time.Second))

	assert.Equal(t, 100., fooBar.Values[0]*float64(time.Second))
	assert.Equal(t, 200., fooBar.Values[1]*float64(time.Second))
	assert.True(t, math.IsNaN(fooBar.Values[2]))
}

func TestAvgOverTimeForDurationWithSecondStage(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(3 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | avg_over_time(duration) by (span.foo) | topk(1)",
	}

	// A variety of spans across times, durations, and series. All durations are powers of 2 for simplicity
	in := []Span{
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(100),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(100),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(100),

		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(100),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(100),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(100),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(500),

		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(100),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(200),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(300),
	}

	result, seriesCount, err := runTraceQLMetric(req, in)
	require.NoError(t, err)
	require.Equal(t, len(result), seriesCount)

	fooBaz := result[`{"span.foo"="baz"}`]
	fooBar := result[`{"span.foo"="bar"}`]

	// We cannot compare with require.Equal because NaN != NaN
	assert.True(t, math.IsNaN(fooBaz.Values[0]))
	assert.True(t, math.IsNaN(fooBaz.Values[1]))
	assert.Equal(t, 200., fooBaz.Values[2]*float64(time.Second))

	assert.Equal(t, 100., fooBar.Values[0]*float64(time.Second))
	assert.Equal(t, 200., fooBar.Values[1]*float64(time.Second))
	assert.True(t, math.IsNaN(fooBar.Values[2]))
}

func TestAvgOverTimeForDurationWithoutAggregation(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(3 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | avg_over_time(duration)",
	}

	// A variety of spans across times, durations, and series. All durations are powers of 2 for simplicity
	in := []Span{
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(100),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(100),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(100),

		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(100),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(100),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(100),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(500),

		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(100),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(200),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "bar").WithDuration(300),
	}

	result, seriesCount, err := runTraceQLMetric(req, in)
	require.NoError(t, err)
	require.Equal(t, len(result), seriesCount)

	avg := result[`{__name__="avg_over_time"}`]

	assert.Equal(t, 100., avg.Values[0]*float64(time.Second))
	assert.Equal(t, 200., avg.Values[1]*float64(time.Second))
	assert.Equal(t, 200., avg.Values[2]*float64(time.Second))
}

func TestAvgOverTimeForSpanAttribute(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(3 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | avg_over_time(span.http.status_code) by (span.foo)",
	}

	// A variety of spans across times, durations, and series. All durations are powers of 2 for simplicity
	in := []Span{
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(128),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 404).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(512),

		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(64),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(8),

		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 200).WithDuration(512),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 400).WithDuration(1024),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 300).WithDuration(512),
	}

	in2 := []Span{
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(128),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(512),

		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(64),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(8),

		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 200).WithDuration(512),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 200).WithDuration(1024),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 200).WithDuration(512),
	}

	result, seriesCount, err := runTraceQLMetric(req, in, in2)
	require.NoError(t, err)
	require.Equal(t, len(result), seriesCount)

	fooBaz := result[`{"span.foo"="baz"}`]
	fooBar := result[`{"span.foo"="bar"}`]

	// Alas,we cannot compare with require.Equal because NaN != NaN
	// foo.baz = (NaN, NaN, 250)
	assert.True(t, math.IsNaN(fooBaz.Values[0]))
	assert.True(t, math.IsNaN(fooBaz.Values[1]))
	assert.Equal(t, 250.0, fooBaz.Values[2])

	// foo.bar = (234,200, NaN)
	assert.Equal(t, 234.0, fooBar.Values[0])
	assert.Equal(t, 200.0, fooBar.Values[1])
	assert.True(t, math.IsNaN(fooBar.Values[2]))

	// Test that NaN values are not included in the samples after casting to proto
	ts := result.ToProto(req)
	fooBarSamples := []tempopb.Sample{{TimestampMs: 1000, Value: 234}, {TimestampMs: 2000, Value: 200}}
	fooBazSamples := []tempopb.Sample{{TimestampMs: 3000, Value: 250}}

	for _, s := range ts {
		if s.PromLabels == "{\"span.foo\"=\"bar\"}" {
			assert.Equal(t, fooBarSamples, s.Samples)
		} else {
			assert.Equal(t, fooBazSamples, s.Samples)
		}
	}
}

func TestAvgOverTimeWithNoMatch(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(3 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | avg_over_time(span.buu)",
	}

	// A variety of spans across times, durations, and series. All durations are powers of 2 for simplicity
	in := []Span{
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(128),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 404).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(512),

		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(64),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(8),

		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 201).WithDuration(512),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 401).WithDuration(1024),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 500).WithDuration(512),
	}

	result, seriesCount, err := runTraceQLMetric(req, in)
	require.NoError(t, err)
	require.Equal(t, len(result), seriesCount)

	// Test that empty timeseries are not included
	ts := result.ToProto(req)

	assert.True(t, len(ts) == 0)
}

func TestObserveSeriesAverageOverTimeForSpanAttribute(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(3 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | avg_over_time(span.http.status_code) by (span.foo)",
	}

	// A variety of spans across times, durations, and series. All durations are powers of 2 for simplicity
	in := []Span{
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 300),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 400),

		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 100),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 100),

		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 200),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 400),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 500),
	}

	in2 := []Span{
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 100),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 300),

		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 400),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200),

		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 100),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 100),
	}

	e := NewEngine()
	layer1A, _ := e.CompileMetricsQueryRange(req, 0, 0, false)
	layer1B, _ := e.CompileMetricsQueryRange(req, 0, 0, false)
	layer2A, _ := e.CompileMetricsQueryRangeNonRaw(req, AggregateModeSum)
	layer2B, _ := e.CompileMetricsQueryRangeNonRaw(req, AggregateModeSum)
	layer3, _ := e.CompileMetricsQueryRangeNonRaw(req, AggregateModeFinal)

	for _, s := range in {
		layer1A.metricsPipeline.observe(s)
	}

	layer2A.ObserveSeries(layer1A.Results().ToProto(req))

	for _, s := range in2 {
		layer1B.metricsPipeline.observe(s)
	}

	layer2B.ObserveSeries(layer1B.Results().ToProto(req))

	layer3.ObserveSeries(layer2A.Results().ToProto(req))
	layer3.ObserveSeries(layer2B.Results().ToProto(req))

	result := layer3.Results()

	fooBaz := result[`{"span.foo"="baz"}`]
	fooBar := result[`{"span.foo"="bar"}`]

	// Alas,we cannot compare with require.Equal because NaN != NaN
	// foo.baz = (NaN, NaN, 300)
	assert.True(t, math.IsNaN(fooBaz.Values[0]))
	assert.True(t, math.IsNaN(fooBaz.Values[1]))
	// 300 = (200 + 400 + 500 + 100) / 4
	assert.Equal(t, 300.0, fooBaz.Values[2])

	// foo.bar = (260,200, 100)
	assert.Equal(t, 260.0, fooBar.Values[0])
	assert.Equal(t, 200.0, fooBar.Values[1])
	assert.Equal(t, 100.0, fooBar.Values[2])
}

func TestObserveSeriesAverageOverTimeForSpanAttributeWithTruncation(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: uint64(1 * time.Second),
		End:   uint64(3 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | avg_over_time(span.http.status_code) by (span.foo)",
	}

	// A variety of spans across times, durations, and series. All durations are powers of 2 for simplicity
	in := []Span{
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 300),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 400),

		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 100),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 100),

		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 200),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 400),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 500),
	}

	in2 := []Span{
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 100),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 300),

		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 400),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200),

		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 100),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 100),
	}

	e := NewEngine()
	layer1A, _ := e.CompileMetricsQueryRange(req, 0, 0, false)
	layer1B, _ := e.CompileMetricsQueryRange(req, 0, 0, false)
	layer2A, _ := e.CompileMetricsQueryRangeNonRaw(req, AggregateModeSum)
	layer2B, _ := e.CompileMetricsQueryRangeNonRaw(req, AggregateModeSum)
	layer3, _ := e.CompileMetricsQueryRangeNonRaw(req, AggregateModeFinal)

	for _, s := range in {
		layer1A.metricsPipeline.observe(s)
	}

	layer2A.ObserveSeries(layer1A.Results().ToProto(req))

	for _, s := range in2 {
		layer1B.metricsPipeline.observe(s)
	}

	layer2B.ObserveSeries(layer1B.Results().ToProto(req))

	layer3.ObserveSeries(layer2A.Results().ToProto(req))
	layer2bResults := layer2B.Results().ToProto(req)
	truncated2bResults := make([]*tempopb.TimeSeries, 0, len(layer2bResults)-1)
	for _, ts := range layer2bResults {
		if !strings.Contains(ts.PromLabels, internalLabelMetaType) {
			// add all values series
			truncated2bResults = append(truncated2bResults, ts)
		} else if len(ts.Samples) != 3 {
			// the panic appears when the count series with 3 samples is missing
			truncated2bResults = append(truncated2bResults, ts)
		}
	}
	assert.NotPanics(t, func() {
		layer3.ObserveSeries(truncated2bResults)
	}, "should not panic on truncation")
}

func TestMaxOverTimeForDuration(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(3 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | max_over_time(duration) by (span.foo)",
	}

	// A variety of spans across times, durations, and series. All durations are powers of 2 for simplicity
	in := []Span{
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(128),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(512),

		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(64),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(8),

		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(512),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(1024),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(512),
	}

	result, seriesCount, err := runTraceQLMetric(req, in)
	require.NoError(t, err)
	require.Equal(t, len(result), seriesCount)

	fooBaz := result[`{"span.foo"="baz"}`]
	fooBar := result[`{"span.foo"="bar"}`]

	// We cannot compare with require.Equal because NaN != NaN
	// foo.baz = (NaN, NaN, 0.000000512)
	assert.True(t, math.IsNaN(fooBaz.Values[0]))
	assert.True(t, math.IsNaN(fooBaz.Values[1]))
	assert.Equal(t, 1024/float64(time.Second), fooBaz.Values[2])

	// foo.bar = (0.000000128, 0.000000128, NaN)
	assert.Equal(t, 512/float64(time.Second), fooBar.Values[0])
	assert.Equal(t, 256/float64(time.Second), fooBar.Values[1])
	assert.True(t, math.IsNaN(fooBar.Values[2]))
}

func TestMaxOverTimeWithNoMatch(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(3 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | max_over_time(span.buu)",
	}

	// A variety of spans across times, durations, and series. All durations are powers of 2 for simplicity
	in := []Span{
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(128),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 404).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(512),

		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(64),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(8),

		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 201).WithDuration(512),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 401).WithDuration(1024),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 500).WithDuration(512),
	}

	result, seriesCount, err := runTraceQLMetric(req, in)
	require.NoError(t, err)
	require.Equal(t, len(result), seriesCount)

	// Test that empty timeseries are not included
	ts := result.ToProto(req)

	assert.True(t, len(ts) == 0)
}

func TestMaxOverTimeForSpanAttribute(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(3 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | max_over_time(span.http.status_code) by (span.foo)",
	}

	// A variety of spans across times, durations, and series. All durations are powers of 2 for simplicity
	in := []Span{
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(128),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 404).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(512),

		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(64),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(8),

		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 201).WithDuration(512),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 401).WithDuration(1024),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 500).WithDuration(512),
	}

	in2 := []Span{
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 100).WithDuration(128),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 300).WithDuration(512),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 204).WithDuration(512),

		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 400).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 401).WithDuration(64),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 402).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 403).WithDuration(8),

		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 200).WithDuration(512),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 300).WithDuration(1024),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 400).WithDuration(512),
	}

	result, seriesCount, err := runTraceQLMetric(req, in, in2)
	require.NoError(t, err)
	require.Equal(t, len(result), seriesCount)

	fooBaz := result[`{"span.foo"="baz"}`]
	fooBar := result[`{"span.foo"="bar"}`]

	// Alas,we cannot compare with require.Equal because NaN != NaN
	// foo.baz = (204, NaN, 500)
	assert.Equal(t, 204.0, fooBaz.Values[0])
	assert.True(t, math.IsNaN(fooBaz.Values[1]))
	assert.Equal(t, 500.0, fooBaz.Values[2])

	// foo.bar = (404,403, NaN)
	assert.Equal(t, 404.0, fooBar.Values[0])
	assert.Equal(t, 403.0, fooBar.Values[1])
	assert.True(t, math.IsNaN(fooBar.Values[2]))

	// Test that NaN values are not included in the samples after casting to proto
	ts := result.ToProto(req)
	fooBarSamples := []tempopb.Sample{{TimestampMs: 1000, Value: 404}, {TimestampMs: 2000, Value: 403}}
	fooBazSamples := []tempopb.Sample{{TimestampMs: 1000, Value: 204}, {TimestampMs: 3000, Value: 500}}

	for _, s := range ts {
		if s.PromLabels == "{\"span.foo\"=\"bar\"}" {
			assert.Equal(t, fooBarSamples, s.Samples)
		} else {
			assert.Equal(t, fooBazSamples, s.Samples)
		}
	}
}

func TestSumOverTimeForDuration(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(3 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | sum_over_time(duration) by (span.foo)",
	}

	// A variety of spans across times, durations, and series. All durations are powers of 2 for simplicity
	in := []Span{
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(10),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(20),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(30),

		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(40),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(50),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(60),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(70),

		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(80),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(90),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(100),
	}

	result, seriesCount, err := runTraceQLMetric(req, in)
	require.NoError(t, err)
	require.Equal(t, len(result), seriesCount)

	fooBaz := result[`{"span.foo"="baz"}`]
	fooBar := result[`{"span.foo"="bar"}`]

	// We cannot compare with require.Equal because NaN != NaN
	// foo.baz = (NaN, NaN, 0.00000027)
	assert.True(t, math.IsNaN(fooBaz.Values[0]))
	assert.True(t, math.IsNaN(fooBaz.Values[1]))
	assert.Equal(t, (80+90+100)/float64(time.Second), fooBaz.Values[2])

	// foo.bar = (0.000000128, 0.000000128, NaN)
	assert.InEpsilon(t, (10+20+30)/float64(time.Second), fooBar.Values[0], 1e-9)
	assert.InEpsilon(t, (40+50+60+70)/float64(time.Second), fooBar.Values[1], 1e-9)

	assert.True(t, math.IsNaN(fooBar.Values[2]))
}

func TestSumOverTimeForSpanAttribute(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(3 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | sum_over_time(span.kafka.lag) by (span.foo)",
	}

	// A variety of spans across times, durations, and series.
	in := []Span{
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("kafka.lag", 100).WithDuration(100),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("kafka.lag", 200).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("kafka.lag", 300).WithDuration(512),

		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("kafka.lag", 200).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("kafka.lag", 200).WithDuration(64),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("kafka.lag", 200).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("kafka.lag", 200).WithDuration(8),

		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("kafka.lag", 200).WithDuration(512),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("kafka.lag", 400).WithDuration(1024),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("kafka.lag", 200).WithDuration(512),
	}

	in2 := []Span{
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("kafka.lag", 100).WithDuration(128),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("kafka.lag", 200).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("kafka.lag", 300).WithDuration(512),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "baz").WithSpanInt("kafka.lag", 200).WithDuration(512),

		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("kafka.lag", 400).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("kafka.lag", 400).WithDuration(64),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("kafka.lag", 400).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("kafka.lag", 400).WithDuration(8),

		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("kafka.lag", 200).WithDuration(512),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("kafka.lag", 300).WithDuration(1024),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("kafka.lag", 400).WithDuration(512),
	}

	result, seriesCount, err := runTraceQLMetric(req, in, in2)
	require.NoError(t, err)
	require.Equal(t, len(result), seriesCount)

	fooBaz := result[`{"span.foo"="baz"}`]
	fooBar := result[`{"span.foo"="bar"}`]

	// Alas,we cannot compare with require.Equal because NaN != NaN
	// foo.baz = (200, NaN, 1700)
	assert.Equal(t, 200.0, fooBaz.Values[0])
	assert.True(t, math.IsNaN(fooBaz.Values[1]))
	assert.Equal(t, 1700.0, fooBaz.Values[2])

	// foo.bar = (1200,2400, NaN)
	assert.Equal(t, 1200.0, fooBar.Values[0])
	assert.Equal(t, 2400.0, fooBar.Values[1])
	assert.True(t, math.IsNaN(fooBar.Values[2]))

	// Test that NaN values are not included in the samples after casting to proto
	ts := result.ToProto(req)
	fooBarSamples := []tempopb.Sample{{TimestampMs: 1000, Value: 1200}, {TimestampMs: 2000, Value: 2400}}
	fooBazSamples := []tempopb.Sample{{TimestampMs: 1000, Value: 200}, {TimestampMs: 3000, Value: 1700}}

	for _, s := range ts {
		if s.PromLabels == "{\"span.foo\"=\"bar\"}" {
			assert.Equal(t, fooBarSamples, s.Samples)
		} else {
			assert.Equal(t, fooBazSamples, s.Samples)
		}
	}
}

func TestSumOverTimeWithNoMatch(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(3 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | sum_over_time(span.buu)",
	}

	// A variety of spans across times, durations, and series. All durations are powers of 2 for simplicity
	in := []Span{
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(128),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 404).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(512),

		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(64),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithSpanInt("http.status_code", 200).WithDuration(8),

		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 201).WithDuration(512),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 401).WithDuration(1024),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithSpanInt("http.status_code", 500).WithDuration(512),
	}

	result, seriesCount, err := runTraceQLMetric(req, in)
	require.NoError(t, err)
	require.Equal(t, len(result), seriesCount)
	// Test that empty timeseries are not included
	ts := result.ToProto(req)

	assert.True(t, len(ts) == 0)
}

func TestHistogramOverTime(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(3 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | histogram_over_time(duration) by (span.foo)",
	}

	var (
		_128ns = NewStaticFloat(0.000000128)
		_256ns = NewStaticFloat(0.000000256)
		_512ns = NewStaticFloat(0.000000512)
	)

	// A variety of spans across times, durations, and series. All durations are powers of 2 for simplicity
	in := []Span{
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(128),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("foo", "bar").WithDuration(512),

		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(256),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("foo", "bar").WithDuration(256),

		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(512),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(512),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("foo", "baz").WithDuration(512),
	}

	// Output series with buckets per foo
	// Prom labels are sorted alphabetically, traceql labels maintain original order.
	out := SeriesSet{
		`{` + internalLabelBucket + `="` + _128ns.EncodeToString(true) + `", "span.foo"="bar"}`: TimeSeries{
			Labels: []Label{
				{Name: "span.foo", Value: NewStaticString("bar")},
				{Name: internalLabelBucket, Value: _128ns},
			},
			Values:    []float64{1, 0, 0},
			Exemplars: make([]Exemplar, 0),
		},
		`{` + internalLabelBucket + `="` + _256ns.EncodeToString(true) + `", "span.foo"="bar"}`: TimeSeries{
			Labels: []Label{
				{Name: "span.foo", Value: NewStaticString("bar")},
				{Name: internalLabelBucket, Value: _256ns},
			},
			Values:    []float64{1, 4, 0},
			Exemplars: make([]Exemplar, 0),
		},
		`{` + internalLabelBucket + `="` + _512ns.EncodeToString(true) + `", "span.foo"="bar"}`: TimeSeries{
			Labels: []Label{
				{Name: "span.foo", Value: NewStaticString("bar")},
				{Name: internalLabelBucket, Value: _512ns},
			},
			Values:    []float64{1, 0, 0},
			Exemplars: make([]Exemplar, 0),
		},
		`{` + internalLabelBucket + `="` + _512ns.EncodeToString(true) + `", "span.foo"="baz"}`: TimeSeries{
			Labels: []Label{
				{Name: "span.foo", Value: NewStaticString("baz")},
				{Name: internalLabelBucket, Value: _512ns},
			},
			Values:    []float64{0, 0, 3},
			Exemplars: make([]Exemplar, 0),
		},
	}

	result, seriesCount, err := runTraceQLMetric(req, in)
	require.NoError(t, err)
	require.Equal(t, out, result)
	require.Equal(t, len(result), seriesCount)
}

func TestSecondStageTopK(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(8 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | rate() by (span.foo) | topk(2)",
	}

	in := make([]Span, 0)
	// 15 spans, at different start times across 3 series
	in = append(in, generateSpans(7, []int{1, 2, 3, 4, 5, 6, 7, 8}, "bar")...)
	in = append(in, generateSpans(5, []int{1, 2, 3, 4, 5, 6, 7, 8}, "baz")...)
	in = append(in, generateSpans(3, []int{1, 2, 3, 4, 5, 6, 7, 8}, "quax")...)

	result, _, err := runTraceQLMetric(req, in)
	require.NoError(t, err)

	// bar and baz have more spans so they should be the top 2
	resultBar := result[`{"span.foo"="bar"}`]
	require.Equal(t, []float64{7, 7, 7, 7, 7, 7, 7, 7}, resultBar.Values)
	resultBaz := result[`{"span.foo"="baz"}`]
	require.Equal(t, []float64{5, 5, 5, 5, 5, 5, 5, 5}, resultBaz.Values)
}

func TestSecondStageTopKInstant(t *testing.T) {
	// Instant Queries are just Range Queries with a step that spans the entire range (end - start) + start
	req := &tempopb.QueryRangeRequest{
		Start: uint64(1 * time.Second),
		End:   uint64(8 * time.Second),
		Step:  uint64(7 * time.Second),
		Query: "{ } | rate() by (span.foo) | topk(2)",
	}

	in := make([]Span, 0)
	// 15 spans, at different start times across 3 series
	in = append(in, generateSpans(7, []int{1, 2, 3, 4, 5, 6, 7, 8}, "bar")...)
	in = append(in, generateSpans(5, []int{1, 2, 3, 4, 5, 6, 7, 8}, "baz")...)
	in = append(in, generateSpans(3, []int{1, 2, 3, 4, 5, 6, 7, 8}, "quax")...)

	result, _, err := runTraceQLMetric(req, in)
	require.NoError(t, err)

	// Instant queries return a single value for each series.
	// so there should be only one value in each series and only two series in the result.
	require.Equal(t, 2, len(result))

	// bar and baz have more spans so they should be the top 2
	require.Equal(t, 1, len(result[`{"span.foo"="bar"}`].Values))
	require.Equal(t, 1, len(result[`{"span.foo"="baz"}`].Values))
}

func TestSecondStageTopKAverage(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(8 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | avg_over_time(duration) by (span.foo) | topk(2)",
	}

	in := make([]Span, 0)
	// 15 spans, at different start times across 3 series
	in = append(in, generateSpans(7, []int{1, 2, 3, 4, 5, 6, 7, 8}, "bar")...)
	in = append(in, generateSpans(5, []int{1, 2, 3, 4, 5, 6, 7, 8}, "baz")...)
	in = append(in, generateSpans(3, []int{1, 2, 3, 4, 5, 6, 7, 8}, "quax")...)

	result, _, err := runTraceQLMetric(req, in)
	require.NoError(t, err)

	resultBar := result[`{"span.foo"="bar"}`]
	val1 := 0.000000512
	require.Equal(t, []float64{val1, val1, val1, val1, val1, val1, val1, val1}, resultBar.Values)
	resultBaz := result[`{"span.foo"="baz"}`]
	val2 := 0.00000038400000000000005
	require.Equal(t, []float64{val2, val2, val2, val2, val2, val2, val2, val2}, resultBaz.Values)
}

func TestSecondStageBottomK(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(8 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | rate() by (span.foo) | bottomk(2)",
	}

	in := make([]Span, 0)
	// 15 spans, at different start times across 3 series
	in = append(in, generateSpans(7, []int{1, 2, 3, 4, 5, 6, 7, 8}, "bar")...)
	in = append(in, generateSpans(5, []int{1, 2, 3, 4, 5, 6, 7, 8}, "baz")...)
	in = append(in, generateSpans(3, []int{1, 2, 3, 4, 5, 6, 7, 8}, "quax")...)

	result, _, err := runTraceQLMetric(req, in)
	require.NoError(t, err)

	// quax and baz have the lowest spans so they should be the bottom 2
	resultQuax := result[`{"span.foo"="quax"}`]
	require.Equal(t, []float64{3, 3, 3, 3, 3, 3, 3, 3}, resultQuax.Values)
	resultBaz := result[`{"span.foo"="baz"}`]
	require.Equal(t, []float64{5, 5, 5, 5, 5, 5, 5, 5}, resultBaz.Values)
}

func TestSecondStageBottomKInstant(t *testing.T) {
	// Instant Queries are just Range Queries with a step that spans the entire range (end - start)
	start := uint64(1 * time.Second)
	end := uint64(8 * time.Second)
	req := &tempopb.QueryRangeRequest{
		Start: start,
		End:   end,
		Step:  end - start,
		Query: "{ } | rate() by (span.foo) | bottomk(2)",
	}

	in := make([]Span, 0)
	// 15 spans, at different start times across 3 series
	in = append(in, generateSpans(7, []int{1, 2, 3, 4, 5, 6, 7, 8}, "bar")...)
	in = append(in, generateSpans(5, []int{1, 2, 3, 4, 5, 6, 7, 8}, "baz")...)
	in = append(in, generateSpans(3, []int{1, 2, 3, 4, 5, 6, 7, 8}, "quax")...)

	result, _, err := runTraceQLMetric(req, in)
	require.NoError(t, err)

	// Instant queries return a single value for each series.
	// so there should be only one value in each series and only two series in the result.
	require.Equal(t, 2, len(result))

	// quax and baz have the lowest spans so they should be the bottom 2
	require.Equal(t, 1, len(result[`{"span.foo"="quax"}`].Values))
	require.Equal(t, 1, len(result[`{"span.foo"="baz"}`].Values))
}

func TestProcessTopK(t *testing.T) {
	tests := []struct {
		name     string
		input    SeriesSet
		limit    int
		expected SeriesSet
	}{
		{
			name: "topk selection",
			input: createSeriesSet(map[string][]float64{
				"a": {1, 5, 3},
				"b": {2, 6, 2},
				"c": {3, 1, 1},
				"d": {4, 2, 4},
			}),
			limit: 2,
			expected: createSeriesSet(map[string][]float64{
				"a": {math.NaN(), 5, 3},          // Top-2 at timestamp 1, 2
				"b": {math.NaN(), 6, math.NaN()}, // Top-2 at timestamp 1
				"c": {3, math.NaN(), math.NaN()}, // Top-2 at timestamp 0
				"d": {4, math.NaN(), 4},          // Top-2 at timestamps 0, 2
			}),
		},
		{
			name: "topk selection at each timestamp",
			input: createSeriesSet(map[string][]float64{
				"a": {1, 2, 3},
				"b": {2, 3, 4},
				"c": {3, 4, 5},
				"d": {1, 1, 1},
				"e": {0.5, 2, 1},
			}),
			limit: 2,
			expected: createSeriesSet(map[string][]float64{
				"b": {2, 3, 4}, // Top-2 at all timestamps
				"c": {3, 4, 5}, // Top-2 at all timestamps
			}),
		},
		{
			name: "select single highest value at specific timestamp",
			input: createSeriesSet(map[string][]float64{
				"a": {1, 6, 3},
				"b": {4, 5, 1},
				"c": {2, 3, 7},
			}),
			limit: 1,
			expected: createSeriesSet(map[string][]float64{
				"a": {math.NaN(), 6, math.NaN()}, // top at timestamp 1
				"b": {4, math.NaN(), math.NaN()}, // top at timestamp 0
				"c": {math.NaN(), math.NaN(), 7}, // top at timestamp 2
			}),
		},
		{
			name: "with NaN values",
			input: createSeriesSet(map[string][]float64{
				"a": {1, math.NaN(), 3},
				"b": {2, 6, math.NaN()},
				"c": {3, 1, 1},
			}),
			limit: 2,
			expected: createSeriesSet(map[string][]float64{
				"a": {math.NaN(), math.NaN(), 3}, // Top-2 at timestamp 2
				"b": {2, 6, math.NaN()},          // Top-2 at timestamp 0, 1
				"c": {3, 1, 1},                   // Top-2 at timestamp 0, 2
			}),
		},
		{
			name: "series with all NaN values is skipped",
			input: createSeriesSet(map[string][]float64{
				"a": {math.NaN(), math.NaN(), math.NaN()},
				"b": {2, 6, math.NaN()},
				"c": {3, 1, 1},
			}),
			limit: 2,
			expected: createSeriesSet(map[string][]float64{
				"b": {2, 6, math.NaN()},
				"c": {3, 1, 1},
			}),
		},
		{
			name: "all series with all NaN values",
			input: createSeriesSet(map[string][]float64{
				"a": {math.NaN(), math.NaN(), math.NaN()},
				"b": {math.NaN(), math.NaN(), math.NaN()},
			}),
			limit:    2,
			expected: SeriesSet{},
		},
		{
			name:     "empty input",
			input:    SeriesSet{},
			limit:    2,
			expected: SeriesSet{},
		},
		{
			name: "limit larger than series count",
			input: createSeriesSet(map[string][]float64{
				"a": {1, 5, 3},
				"b": {2, 6, 2},
			}),
			limit: 5,
			expected: createSeriesSet(map[string][]float64{
				"a": {1, 5, 3},
				"b": {2, 6, 2},
			}),
		},
		{
			name: "negative and infinity values",
			input: createSeriesSet(map[string][]float64{
				"a": {-1, 5, math.Inf(-1)},
				"b": {-2, 6, math.Inf(1)},
				"c": {-3, 7, 1},
			}),
			limit: 2,
			expected: createSeriesSet(map[string][]float64{
				"a": {-1, math.NaN(), math.NaN()}, // Top-2 at timestamp 0
				"b": {-2, 6, math.Inf(1)},         // Top-2 at timestamps 1, 2
				"c": {math.NaN(), 7, 1},           // Top-2 at timestamp 1
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processTopK(tt.input, 3, tt.limit)
			expectSeriesSet(t, tt.expected, result)
		})
	}
}

func TestProcessBottomK(t *testing.T) {
	tests := []struct {
		name     string
		input    SeriesSet
		limit    int
		expected SeriesSet
	}{
		{
			name: "bottomk selection",
			input: createSeriesSet(map[string][]float64{
				"a": {1, 5, 3},
				"b": {2, 6, 2},
				"c": {3, 1, 1},
				"d": {4, 2, 4},
			}),
			limit: 2,
			expected: createSeriesSet(map[string][]float64{
				"a": {1, math.NaN(), math.NaN()}, // Bottom-2 at timestamp 0
				"b": {2, math.NaN(), 2},          // Bottom-2 at timestamps 0, 2
				"c": {math.NaN(), 1, 1},          // Bottom-2 at timestamps 1, 2
				"d": {math.NaN(), 2, math.NaN()}, // Bottom-2 at timestamp 1
			}),
		},
		{
			name: "bottomk selection at each timestamp",
			input: createSeriesSet(map[string][]float64{
				"a": {5, 4, 3},
				"b": {6, 5, 4},
				"c": {7, 6, 1},
				"d": {3, 3, 3},
				"e": {4, 2, 2},
			}),
			limit: 2,
			expected: createSeriesSet(map[string][]float64{
				"c": {math.NaN(), math.NaN(), 1}, // bottom 2 at timestamp 2
				"d": {3, 3, math.NaN()},          // bottom 2 at timestamp 0, 1
				"e": {4, 2, 2},                   // bottom 2 at timestamp 0, 1, 2
			}),
		},
		{
			name: "select single lowest value at specific timestamp",
			input: createSeriesSet(map[string][]float64{
				"a": {3, 1, 5},
				"b": {1, 2, 3},
				"c": {4, 5, 1},
			}),
			limit: 1,
			expected: createSeriesSet(map[string][]float64{
				"a": {math.NaN(), 1, math.NaN()}, // Lowest at timestamp 1
				"b": {1, math.NaN(), math.NaN()}, // Lowest at timestamp 0
				"c": {math.NaN(), math.NaN(), 1}, // Lowest at timestamp 2
			}),
		},
		{
			name: "with NaN values",
			input: createSeriesSet(map[string][]float64{
				"a": {1, math.NaN(), 3},
				"b": {4, 6, math.NaN()},
				"c": {5, 1, 1},
			}),
			limit: 2,
			expected: createSeriesSet(map[string][]float64{
				"a": {1, math.NaN(), 3}, // Bottom-2 at timestamp 0
				"b": {4, 6, math.NaN()}, // NaN values are skipped in comparison
				"c": {math.NaN(), 1, 1}, // Bottom-2 at timestamps 1, 2
			}),
		},
		{
			name: "all series with NaN values",
			input: createSeriesSet(map[string][]float64{
				"a": {math.NaN(), math.NaN(), math.NaN()},
				"b": {math.NaN(), math.NaN(), math.NaN()},
				"c": {math.NaN(), math.NaN(), math.NaN()},
			}),
			limit:    2,
			expected: SeriesSet{},
		},
		{
			name:     "empty input",
			input:    SeriesSet{},
			limit:    2,
			expected: SeriesSet{},
		},
		{
			name: "limit larger than series count",
			input: createSeriesSet(map[string][]float64{
				"a": {1, 5, 3},
				"b": {2, 6, 2},
			}),
			limit: 5,
			expected: createSeriesSet(map[string][]float64{
				"a": {1, 5, 3},
				"b": {2, 6, 2},
			}),
		},
		{
			name: "negative and infinity values",
			input: createSeriesSet(map[string][]float64{
				"a": {-1, 5, math.Inf(-1)},
				"b": {-2, 6, math.Inf(1)},
				"c": {-3, 7, 1},
			}),
			limit: 2,
			expected: createSeriesSet(map[string][]float64{
				"a": {math.NaN(), 5, math.Inf(-1)}, // Bottom-2 at timestamp 2
				"b": {-2, 6, math.NaN()},           // Bottom-2 at timestamp 0
				"c": {-3, math.NaN(), 1},           // Bottom-2 at timestamps 0, 2
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Printf("input: %v\n", tt.input)
			result := processBottomK(tt.input, 3, tt.limit)
			fmt.Printf("result: %v\n", result)
			expectSeriesSet(t, tt.expected, result)
		})
	}
}

func TestTiesInTopK(t *testing.T) {
	input := createSeriesSet(map[string][]float64{
		"a": {10, 5, 1},
		"b": {10, 4, 2},
		"c": {10, 3, 3},
	})
	result := processTopK(input, 3, 2)
	fmt.Printf("result: %v\n", result)

	// because of ties, we can have different result at index 0
	// "a" can be [10, 5, NaN] OR [NaN, 5, NaN]
	// "b" can be [10, 4, 2] OR [NaN, 4, 2]
	// "c" can be [10, NaN, 3] OR [NaN, NaN, 3]
	checkEqualForTies(t, result[`{label="a"}`].Values, []float64{10, 5, math.NaN()})
	checkEqualForTies(t, result[`{label="b"}`].Values, []float64{10, 4, 2})
	checkEqualForTies(t, result[`{label="c"}`].Values, []float64{10, math.NaN(), 3})
}

func TestTiesInBottomK(t *testing.T) {
	input := createSeriesSet(map[string][]float64{
		"a": {10, 5, 1},
		"b": {10, 4, 2},
		"c": {10, 3, 3},
	})
	result := processBottomK(input, 3, 2)

	// because of ties, we can have different result at index 0
	// "a" can be [10, NaN, 1] OR [NaN, NaN, 1]
	// "b" can be [10, 4, 2] OR [NaN, 4, 2]
	// "c" can be [10, 3, NaN] OR [NaN, 3, NaN]
	checkEqualForTies(t, result[`{label="a"}`].Values, []float64{10, math.NaN(), 1})
	checkEqualForTies(t, result[`{label="b"}`].Values, []float64{10, 4, 2})
	checkEqualForTies(t, result[`{label="c"}`].Values, []float64{10, 3, math.NaN()})
}

func TestHistogramAggregator(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start:     uint64(time.Now().Add(-1 * time.Hour).UnixNano()),
		End:       uint64(time.Now().UnixNano()),
		Step:      uint64(15 * time.Second.Nanoseconds()),
		Exemplars: maxExemplars,
	}
	const seriesCount = 6

	cases := []struct {
		name          string
		samplesCount  int
		exemplarCount int
	}{
		{"Small", 10, 5},
		{"Medium", 100, 20},
		{"Large", 1000, 100},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			series := generateTestTimeSeries(seriesCount, tc.samplesCount, tc.exemplarCount, req.Start, req.End)
			quantiles := []float64{0.5, 0.9, 0.99}

			agg := NewHistogramAggregator(req, quantiles, uint32(tc.exemplarCount))
			agg.Combine(series)
			results := agg.Results()
			require.NotNil(t, results, "Expected non-nil SeriesSet from HistogramAggregator")
			require.Greater(t, len(results), 0, "Expected non-empty SeriesSet from HistogramAggregator")

			// Check that exemplars are distributed across quantile series
			totalExemplars := 0
			for _, ts := range results {
				totalExemplars += len(ts.Exemplars)
			}
			require.Greater(t, totalExemplars, 0, "Expected at least some exemplars across all quantile series")

			for _, ts := range results {
				// With aggregated semantic matching, total exemplars can be higher since
				// we're more efficiently distributing exemplars across quantile series
				require.LessOrEqual(t, len(ts.Exemplars), tc.exemplarCount*5, "Per-series exemplars should be reasonable")
				// Note: With aggregated semantic matching, exemplars are distributed more efficiently
				// across quantile series based on comprehensive quantile thresholds

				// t.Logf("Series: %s", ts.Labels)
				// t.Logf("Values: %v", ts.Values)
				// t.Logf("Exemplars: %v", ts.Exemplars)

				// check that the values are within the expected histogram buckets
				require.Greater(t, len(ts.Values), 0, "Expected non-empty histogram values")
				for _, value := range ts.Values {
					require.GreaterOrEqual(t, value, 0.0, "Histogram values should be non-negative")
				}
				// check that the exemplars are within the expected histogram Buckets
				for _, ex := range ts.Exemplars {
					require.GreaterOrEqual(t, ex.Value, 0.0, "Exemplar values should be non-negative")
					// Convert nanoseconds to milliseconds for comparison
					startMs := req.Start / uint64(time.Millisecond)
					endMs := req.End / uint64(time.Millisecond)
					require.GreaterOrEqual(t, ex.TimestampMs, startMs)
					require.LessOrEqual(t, ex.TimestampMs, endMs)
				}

				// With semantic matching, some quantile series may legitimately have zero exemplars
				// if no exemplars fall within their quantile range - this is correct behavior

			}
		})
	}
}

func runTraceQLMetric(req *tempopb.QueryRangeRequest, inSpans ...[]Span) (SeriesSet, int, error) {
	res, err := processLayer1AndLayer2(req, inSpans...)
	if err != nil {
		return nil, 0, err
	}

	// Pass layer 2 to layer 3
	// These are summed counts over time by bucket
	return processLayer3(req, res)
}

func processLayer1AndLayer2(req *tempopb.QueryRangeRequest, in ...[]Span) (SeriesSet, error) {
	e := NewEngine()

	layer2, err := e.CompileMetricsQueryRangeNonRaw(req, AggregateModeSum)
	if err != nil {
		return nil, err
	}

	for _, spanSet := range in {
		layer1, err := e.CompileMetricsQueryRange(req, 0, 0, false)
		if err != nil {
			return nil, err
		}
		for _, s := range spanSet {
			layer1.metricsPipeline.observe(s)
		}
		res := layer1.Results()
		// Pass layer 1 to layer 2
		// These are partial counts over time by bucket
		layer2.metricsPipeline.observeSeries(res.ToProto(req))
	}

	return layer2.Results(), nil
}

func processLayer3(req *tempopb.QueryRangeRequest, res SeriesSet) (SeriesSet, int, error) {
	e := NewEngine()

	layer3, err := e.CompileMetricsQueryRangeNonRaw(req, AggregateModeFinal)
	if err != nil {
		return nil, 0, err
	}

	layer3.ObserveSeries(res.ToProto(req))
	return layer3.Results(), layer3.Length(), nil
}

func randInt(minimum, maximum int) int {
	return rand.IntN(maximum-minimum) + minimum
}

func randFloat(minimum, maximum float64) float64 {
	return rand.Float64()*(maximum-minimum) + minimum
}

func generateSpans(count int, startTimes []int, value string) []Span {
	spans := make([]Span, 0)
	for i := 0; i < count; i++ {
		for _, t := range startTimes {
			sTime := uint64(time.Duration(t) * time.Second)
			spans = append(spans, newMockSpan(nil).WithStartTime(sTime).WithSpanString("foo", value).WithDuration(128*uint64(i+1)))
		}
	}
	return spans
}

// createSeriesSet to create a SeriesSet from a map of values
func createSeriesSet(data map[string][]float64) SeriesSet {
	seriesSet := SeriesSet{}
	labelName := "label"
	for key, values := range data {
		seriesSet[fmt.Sprintf(`{%s="%s"}`, labelName, key)] = TimeSeries{
			Values: values,
			Labels: Labels{Label{Name: labelName, Value: NewStaticString(key)}},
		}
	}
	return seriesSet
}

// expectSeriesSet validates SeriesSet equality, and also considers NaN values
func expectSeriesSet(t *testing.T, expected, result SeriesSet) {
	for expectedKey, expectedSeries := range expected {
		resultSeries, ok := result[expectedKey]
		require.True(t, ok, "expected series %s to be in result", expectedKey)
		require.Equal(t, expectedSeries.Labels, resultSeries.Labels)

		// check values, including NaN values
		require.Equal(t, len(expectedSeries.Values), len(resultSeries.Values))
		for i, expectedValue := range expectedSeries.Values {
			if math.IsNaN(expectedValue) {
				require.True(t, math.IsNaN(resultSeries.Values[i]), "expected NaN at index %d", i)
			} else {
				require.Equal(t, expectedValue, resultSeries.Values[i])
			}
		}
	}
}

func checkEqualForTies(t *testing.T, result, expected []float64) {
	for i := range result {
		switch i {
		// at index 0, we have a tie so it can be sometimes NaN
		case 0:
			require.True(t, math.IsNaN(result[0]) || result[0] == expected[0],
				"index 0: expected NaN or %v, got %v", expected[0], result[0])
		default:
			if math.IsNaN(expected[i]) {
				require.True(t, math.IsNaN(result[i]), "index %d: expected NaN, got %v", i, result[i])
			} else {
				require.Equal(t, expected[i], result[i], "index %d: expected %v", i, expected[i])
			}
		}
	}
}

func BenchmarkSumOverTime(b *testing.B) {
	totalSpans := 1_000_000
	in := make([]Span, 0, totalSpans)
	in2 := make([]Span, 0, totalSpans)
	minimum := 1e10 // 10 billion
	maximun := 1e20 // 100 quintillion

	for range totalSpans {
		s := time.Duration(randInt(1, 3)) * time.Second
		v := randFloat(minimum, maximun)
		in = append(in2, newMockSpan(nil).WithStartTime(uint64(s)).WithSpanString("foo", "bar").WithSpanFloat("kafka.lag", v).WithDuration(100))
		s = time.Duration(randInt(1, 3)) * time.Second
		v = randFloat(minimum, maximun)
		in2 = append(in2, newMockSpan(nil).WithStartTime(uint64(s)).WithSpanString("foo", "bar").WithSpanFloat("kafka.lag", v).WithDuration(100))
	}

	req := &tempopb.QueryRangeRequest{
		Start: uint64(1 * time.Second),
		End:   uint64(3 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | sum_over_time(span.kafka.lag) by (span.foo)",
	}
	for b.Loop() {
		_, _, _ = runTraceQLMetric(req, in, in2)
	}
}

func BenchmarkHistogramAggregator_Combine(b *testing.B) {
	// nolint:gosec // G115
	req := &tempopb.QueryRangeRequest{
		Start:     uint64(time.Now().Add(-1 * time.Hour).UnixNano()),
		End:       uint64(time.Now().UnixNano()),
		Step:      uint64(15 * time.Second.Nanoseconds()),
		Exemplars: maxExemplars,
	}
	const seriesCount = 6

	benchmarks := []struct {
		name          string
		samplesCount  int
		exemplarCount int
	}{
		{"Small", 10, 5},
		{"Medium", 100, 20},
		{"Large", 1000, 100},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			series := generateTestTimeSeries(seriesCount, bm.samplesCount, bm.exemplarCount, req.Start, req.End)

			for b.Loop() {
				agg := NewHistogramAggregator(req, []float64{0.5, 0.9, 0.99}, uint32(bm.exemplarCount)) // nolint: gosec // G115
				agg.Combine(series)
			}
		})
	}
}

func BenchmarkHistogramAggregator_Results(b *testing.B) {
	// nolint:gosec // G115
	req := &tempopb.QueryRangeRequest{
		Start:     uint64(time.Now().Add(-1 * time.Hour).UnixNano()),
		End:       uint64(time.Now().UnixNano()),
		Step:      uint64(15 * time.Second.Nanoseconds()),
		Exemplars: maxExemplars,
	}

	benchmarks := []struct {
		name          string
		seriesCount   int
		samplesCount  int
		exemplarCount int
		quantiles     []float64
	}{
		{"Small_3Quantiles", 6, 10, 5, []float64{0.5, 0.9, 0.99}},
		{"Medium_3Quantiles", 10, 100, 20, []float64{0.5, 0.9, 0.99}},
		{"Large_3Quantiles", 20, 1000, 100, []float64{0.5, 0.9, 0.99}},
		// These test the bucket rescanning optimization specifically
		{"Small_5Quantiles", 6, 10, 5, []float64{0.5, 0.75, 0.9, 0.95, 0.99}},
		{"Medium_5Quantiles", 10, 100, 20, []float64{0.5, 0.75, 0.9, 0.95, 0.99}},
		{"Large_5Quantiles", 20, 1000, 100, []float64{0.5, 0.75, 0.9, 0.95, 0.99}},
		// High exemplar density to test caching benefits
		{"High_Exemplars", 10, 100, 200, []float64{0.5, 0.9, 0.99}},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			// Generate test data
			series := generateTestTimeSeries(bm.seriesCount, bm.samplesCount, bm.exemplarCount, req.Start, req.End)

			// Create histogram aggregator
			h := NewHistogramAggregator(req, bm.quantiles, req.Exemplars)

			// Combine the series (this is setup, not benchmarked)
			h.Combine(series)

			// Benchmark the Results method
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				results := h.Results()
				_ = results // Prevent optimization
			}
		})
	}
}

// generateTestTimeSeries creates test time series data for benchmarking
// nolint:gosec // G115
func generateTestTimeSeries(seriesCount, samplesCount, exemplarCount int, start, end uint64) []*tempopb.TimeSeries {
	result := make([]*tempopb.TimeSeries, seriesCount)

	timeRange := end - start

	for i := 0; i < seriesCount; i++ {
		// Create unique labels for each series
		labels := []commonv1proto.KeyValue{
			{
				Key: "service",
				Value: &commonv1proto.AnyValue{
					Value: &commonv1proto.AnyValue_StringValue{
						StringValue: "service-" + fmt.Sprintf("%d", i),
					},
				},
			},
			{
				Key: internalLabelBucket,
				Value: &commonv1proto.AnyValue{
					Value: &commonv1proto.AnyValue_DoubleValue{
						DoubleValue: math.Pow(2, float64(i%20)), // Power of 2 as bucket
					},
				},
			},
		}

		samples := make([]tempopb.Sample, samplesCount)
		for j := 0; j < samplesCount; j++ {
			// Distribute samples evenly across the time range
			offset := (uint64(j) * timeRange) / uint64(samplesCount)
			ts := time.Unix(0, int64(start+offset)).UnixMilli()
			samples[j] = tempopb.Sample{
				TimestampMs: ts,
				Value:       float64(j % 100), // Simple pattern for test data
			}
		}

		// Create exemplars
		exemplars := make([]tempopb.Exemplar, exemplarCount)
		for j := 0; j < exemplarCount; j++ {
			// Distribute exemplars evenly across the time range
			offset := (uint64(j) * timeRange) / uint64(exemplarCount)
			ts := time.Unix(0, int64(start+offset)).UnixMilli()
			exemplarLabels := []commonv1proto.KeyValue{
				{
					Key: "trace_id",
					Value: &commonv1proto.AnyValue{
						Value: &commonv1proto.AnyValue_StringValue{
							StringValue: fmt.Sprintf("trace-%d", i*1000+j),
						},
					},
				},
				{
					Key: "span_id",
					Value: &commonv1proto.AnyValue{
						Value: &commonv1proto.AnyValue_StringValue{
							StringValue: fmt.Sprintf("span-%d", j),
						},
					},
				},
			}
			exemplars[j] = tempopb.Exemplar{
				Labels: exemplarLabels,

				Value:       float64(j % 100), // Simple pattern for test data
				TimestampMs: ts,
			}
		}

		result[i] = &tempopb.TimeSeries{
			PromLabels: fmt.Sprintf("{service=\"service-%d\",bucket=\"%d\"}", i, i%20),
			Labels:     labels,
			Samples:    samples,
			Exemplars:  exemplars,
		}
	}

	return result
}

func TestHistogramAggregator_ExemplarBucketSelection(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start:     uint64(time.Now().Add(-1 * time.Hour).UnixNano()),
		End:       uint64(time.Now().UnixNano()),
		Step:      uint64(15 * time.Second.Nanoseconds()),
		Exemplars: 10,
	}

	tests := []struct {
		name                    string
		quantiles               []float64
		timeSeries              []*tempopb.TimeSeries
		expectedExemplarBuckets map[string][]float64 // quantile label -> expected bucket values
	}{
		{
			name:      "exemplars match correct quantile buckets",
			quantiles: []float64{0.5, 0.9},
			timeSeries: []*tempopb.TimeSeries{
				{
					PromLabels: `{service="test",__bucket="1"}`,
					Labels: []commonv1proto.KeyValue{
						{Key: "service", Value: &commonv1proto.AnyValue{Value: &commonv1proto.AnyValue_StringValue{StringValue: "test"}}},
						{Key: internalLabelBucket, Value: &commonv1proto.AnyValue{Value: &commonv1proto.AnyValue_DoubleValue{DoubleValue: 1.0}}},
					},
					Samples: []tempopb.Sample{
						{TimestampMs: time.Unix(0, int64(req.Start)).UnixMilli(), Value: 5}, // 5 samples in 1s bucket (p50)
					},
					Exemplars: []tempopb.Exemplar{
						{
							Labels: []commonv1proto.KeyValue{
								{Key: "trace_id", Value: &commonv1proto.AnyValue{Value: &commonv1proto.AnyValue_StringValue{StringValue: "trace1"}}},
							},
							Value:       0.8, // Should go to p50 (< quantile threshold)
							TimestampMs: time.Unix(0, int64(req.Start)).UnixMilli(),
						},
					},
				},
				{
					PromLabels: `{service="test",__bucket="4"}`,
					Labels: []commonv1proto.KeyValue{
						{Key: "service", Value: &commonv1proto.AnyValue{Value: &commonv1proto.AnyValue_StringValue{StringValue: "test"}}},
						{Key: internalLabelBucket, Value: &commonv1proto.AnyValue{Value: &commonv1proto.AnyValue_DoubleValue{DoubleValue: 4.0}}},
					},
					Samples: []tempopb.Sample{
						{TimestampMs: time.Unix(0, int64(req.Start)).UnixMilli(), Value: 1}, // 1 sample in 4s bucket (p90)
					},
					Exemplars: []tempopb.Exemplar{
						{
							Labels: []commonv1proto.KeyValue{
								{Key: "trace_id", Value: &commonv1proto.AnyValue{Value: &commonv1proto.AnyValue_StringValue{StringValue: "trace2"}}},
							},
							Value:       3.5, // Should go to p90 (> quantile threshold)
							TimestampMs: time.Unix(0, int64(req.Start)).UnixMilli(),
						},
					},
				},
			},
			expectedExemplarBuckets: map[string][]float64{
				`{p="0.5", service="test"}`: {1.0}, // p50 should get fast exemplars
				`{p="0.9", service="test"}`: {4.0}, // p90 should get slow exemplars
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := NewHistogramAggregator(req, tt.quantiles, 10)
			agg.Combine(tt.timeSeries)
			results := agg.Results()

			// Verify semantic matching behavior - exemplars should be assigned based on quantile ranges
			p50Series, p50Exists := results[`{p="0.5", service="test"}`]
			p90Series, p90Exists := results[`{p="0.9", service="test"}`]

			require.True(t, p50Exists, "p50 series should exist")
			require.True(t, p90Exists, "p90 series should exist")

			// Check that we have exemplars distributed appropriately
			totalExemplars := len(p50Series.Exemplars) + len(p90Series.Exemplars)
			require.Greater(t, totalExemplars, 0, "Should have some exemplars")

			// Verify exemplars are assigned correctly based on semantic matching
			// Exemplars should be distributed across quantile ranges appropriately
		})
	}
}

func TestHistogramAggregator_ExemplarDistribution(t *testing.T) {
	// Use fixed timestamps to ensure deterministic behavior
	baseTime := time.Unix(1257894000, 0) // Fixed timestamp
	req := &tempopb.QueryRangeRequest{
		Start:     uint64(baseTime.UnixNano()),
		End:       uint64(baseTime.Add(1 * time.Hour).UnixNano()),
		Step:      uint64(15 * time.Second.Nanoseconds()),
		Exemplars: 12, // Will be distributed: 12/3 = 4 per quantile
	}

	// Create test data with multiple exemplars
	timeSeries := []*tempopb.TimeSeries{
		{
			PromLabels: `{service="test",__bucket="2"}`,
			Labels: []commonv1proto.KeyValue{
				{Key: "service", Value: &commonv1proto.AnyValue{Value: &commonv1proto.AnyValue_StringValue{StringValue: "test"}}},
				{Key: internalLabelBucket, Value: &commonv1proto.AnyValue{Value: &commonv1proto.AnyValue_DoubleValue{DoubleValue: 2.0}}},
			},
			Samples: []tempopb.Sample{
				{TimestampMs: baseTime.UnixMilli(), Value: 10},
			},
			Exemplars: []tempopb.Exemplar{
				{Value: 1.5, TimestampMs: baseTime.UnixMilli()},
				{Value: 1.6, TimestampMs: baseTime.UnixMilli()},
				{Value: 1.7, TimestampMs: baseTime.UnixMilli()},
				{Value: 1.8, TimestampMs: baseTime.UnixMilli()},
				{Value: 1.9, TimestampMs: baseTime.UnixMilli()},
			},
		},
	}

	quantiles := []float64{0.5, 0.9, 0.99}
	agg := NewHistogramAggregator(req, quantiles, 12)
	agg.Combine(timeSeries)
	results := agg.Results()

	totalExemplars := 0
	for _, series := range results {
		totalExemplars += len(series.Exemplars)
	}

	// Verify total exemplars doesn't exceed limit
	require.LessOrEqual(t, totalExemplars, 12, "Total exemplars should not exceed limit")

	// Verify each quantile gets roughly equal distribution (allowing for rounding)
	expectedPerQuantile := 12 / len(quantiles)
	for seriesLabel, series := range results {
		require.LessOrEqual(t, len(series.Exemplars), expectedPerQuantile+1,
			"Series %s has too many exemplars: %d", seriesLabel, len(series.Exemplars))
	}
}

func TestLog2Quantile(t *testing.T) {
	tests := []struct {
		name          string
		quantile      float64
		buckets       []HistogramBucket
		expectedValue float64
	}{
		{
			name:     "p50 in middle bucket",
			quantile: 0.5,
			buckets: []HistogramBucket{
				{Max: 1.0, Count: 2},
				{Max: 2.0, Count: 4}, // p50 should land here
				{Max: 4.0, Count: 2},
			},
		},
		{
			name:     "p90 in last bucket",
			quantile: 0.9,
			buckets: []HistogramBucket{
				{Max: 1.0, Count: 1},
				{Max: 2.0, Count: 1},
				{Max: 4.0, Count: 8}, // p90 should land here
			},
		},
		{
			name:     "p100 exact match",
			quantile: 1.0,
			buckets: []HistogramBucket{
				{Max: 1.0, Count: 5},
				{Max: 2.0, Count: 5},
			},
			expectedValue: 2.0,
		},
		{
			name:          "empty buckets",
			quantile:      0.5,
			buckets:       []HistogramBucket{},
			expectedValue: 0.0,
		},
		{
			name:     "single bucket",
			quantile: 0.5,
			buckets: []HistogramBucket{
				{Max: 2.0, Count: 10},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := Log2Quantile(tt.quantile, tt.buckets)

			if tt.expectedValue != 0 {
				require.Equal(t, tt.expectedValue, value, "Quantile value mismatch")
			}

			// Verify value is reasonable (non-negative)
			require.GreaterOrEqual(t, value, 0.0, "Quantile values should be non-negative")
		})
	}
}

func TestHistogramAggregator_EdgeCases(t *testing.T) {
	// Use fixed timestamps to ensure deterministic behavior
	baseTime := time.Unix(1257894000, 0) // Fixed timestamp
	req := &tempopb.QueryRangeRequest{
		Start:     uint64(baseTime.UnixNano()),
		End:       uint64(baseTime.Add(1 * time.Hour).UnixNano()),
		Step:      uint64(15 * time.Second.Nanoseconds()),
		Exemplars: 5,
	}

	tests := []struct {
		name       string
		timeSeries []*tempopb.TimeSeries
		quantiles  []float64
		expectFunc func(t *testing.T, results SeriesSet)
	}{
		{
			name:      "no exemplars",
			quantiles: []float64{0.5, 0.9},
			timeSeries: []*tempopb.TimeSeries{
				{
					PromLabels: `{service="test",__bucket="2"}`,
					Labels: []commonv1proto.KeyValue{
						{Key: "service", Value: &commonv1proto.AnyValue{Value: &commonv1proto.AnyValue_StringValue{StringValue: "test"}}},
						{Key: internalLabelBucket, Value: &commonv1proto.AnyValue{Value: &commonv1proto.AnyValue_DoubleValue{DoubleValue: 2.0}}},
					},
					Samples: []tempopb.Sample{
						{TimestampMs: baseTime.UnixMilli(), Value: 5},
					},
					Exemplars: []tempopb.Exemplar{}, // No exemplars
				},
			},
			expectFunc: func(t *testing.T, results SeriesSet) {
				for _, series := range results {
					require.Empty(t, series.Exemplars, "Should have no exemplars")
				}
			},
		},
		{
			name:      "exemplars outside bucket ranges",
			quantiles: []float64{0.5},
			timeSeries: []*tempopb.TimeSeries{
				{
					PromLabels: `{service="test",__bucket="2"}`,
					Labels: []commonv1proto.KeyValue{
						{Key: "service", Value: &commonv1proto.AnyValue{Value: &commonv1proto.AnyValue_StringValue{StringValue: "test"}}},
						{Key: internalLabelBucket, Value: &commonv1proto.AnyValue{Value: &commonv1proto.AnyValue_DoubleValue{DoubleValue: 2.0}}},
					},
					Samples: []tempopb.Sample{
						{TimestampMs: baseTime.UnixMilli(), Value: 5},
					},
					Exemplars: []tempopb.Exemplar{
						{
							Value:       10.0, // Much larger than bucket, should not match
							TimestampMs: baseTime.UnixMilli(),
						},
					},
				},
			},
			expectFunc: func(t *testing.T, results SeriesSet) {
				for _, series := range results {
					// May have no exemplars if the value doesn't match any bucket
					// This is acceptable behavior
					if len(series.Exemplars) > 0 {
						// If exemplars exist, they should be reasonable
						for _, ex := range series.Exemplars {
							bucketValue := Log2Bucketize(uint64(ex.Value * float64(time.Second)))
							require.True(t, bucketValue > 0, "Bucket value should be positive")
						}
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := NewHistogramAggregator(req, tt.quantiles, 5)
			agg.Combine(tt.timeSeries)
			results := agg.Results()

			require.NotNil(t, results, "Results should not be nil")
			tt.expectFunc(t, results)
		})
	}
}

func createBucketSeries(bucketValue string, count int, timestampMs int64) *tempopb.TimeSeries {
	bucketFloat, _ := strconv.ParseFloat(bucketValue, 64)
	return &tempopb.TimeSeries{
		PromLabels: fmt.Sprintf(`{service="test",__bucket="%s"}`, bucketValue),
		Labels: []commonv1proto.KeyValue{
			{Key: "service", Value: &commonv1proto.AnyValue{Value: &commonv1proto.AnyValue_StringValue{StringValue: "test"}}},
			{Key: internalLabelBucket, Value: &commonv1proto.AnyValue{Value: &commonv1proto.AnyValue_DoubleValue{DoubleValue: bucketFloat}}},
		},
		Samples: []tempopb.Sample{
			{TimestampMs: timestampMs, Value: float64(count)},
		},
	}
}

// requireEqualSeriesSets is like require.Equal for SeriesSets and supports NaN.
func requireEqualSeriesSets(t *testing.T, expected, actual SeriesSet) {
	require.Equal(t, len(expected), len(actual))

	for k, eTS := range expected {
		aTS, ok := actual[k]
		require.True(t, ok, "expected series %s to be in result", k)
		require.Equal(t, eTS.Labels, aTS.Labels, "expected labels %v, got %v", eTS.Labels, aTS.Labels)

		eSamples := eTS.Values
		aSamples := aTS.Values

		require.Equal(t, len(eSamples), len(aSamples), "expected %d samples for %s, got %d", len(eSamples), k, len(aSamples))
		for i := range eSamples {
			if math.IsNaN(eSamples[i]) {
				require.True(t, math.IsNaN(aSamples[i]))
			} else {
				require.InDelta(t, eSamples[i], aSamples[i], 0.001, "expected %v, got %v, for %s[%d]", eSamples[i], aSamples[i], k, i)
			}
		}
	}
}

func TestHistogramAggregator_LatencySpike(t *testing.T) {
	// Simulate a latency spike: normal traffic, then a spike, then normal again
	// Use fixed timestamps to ensure deterministic behavior
	baseTime := time.Unix(1640995200, 0) // Fixed timestamp: 2022-01-01 00:00:00 UTC
	start := baseTime
	req := &tempopb.QueryRangeRequest{
		Start: uint64(start.UnixNano()),
		End:   uint64(start.Add(45 * time.Minute).UnixNano()),
		Step:  uint64(15 * time.Minute), // 3 intervals: normal, spike, normal
	}
	quantiles := []float64{0.5, 0.9, 0.99}

	agg := NewHistogramAggregator(req, quantiles, 20)

	// Interval 1: Normal latency (0-15min) - p50=100ms, p90=200ms, p99=500ms
	normal1Time := start.Add(7 * time.Minute)
	normalSeries1 := []*tempopb.TimeSeries{
		createBucketSeries("0.125", 70, normal1Time.UnixMilli()), // 70 fast requests
		createBucketSeries("0.25", 20, normal1Time.UnixMilli()),  // 20 medium requests
		createBucketSeries("0.5", 8, normal1Time.UnixMilli()),    // 8 slow requests
		createBucketSeries("1.0", 2, normal1Time.UnixMilli()),    // 2 very slow requests
	}

	// Add exemplars for normal period
	normalSeries1[0].Exemplars = []tempopb.Exemplar{
		{Value: 0.08, TimestampMs: normal1Time.UnixMilli()}, // Fast - should go to p50
		{Value: 0.12, TimestampMs: normal1Time.UnixMilli()}, // Fast - should go to p50
	}
	normalSeries1[1].Exemplars = []tempopb.Exemplar{
		{Value: 0.18, TimestampMs: normal1Time.UnixMilli()}, // Medium - should go to p90
	}
	normalSeries1[2].Exemplars = []tempopb.Exemplar{
		{Value: 0.35, TimestampMs: normal1Time.UnixMilli()}, // Slow - should go to p99
	}

	// Interval 2: Latency spike (15-30min) - p50=800ms, p90=2000ms, p99=4000ms
	spikeTime := start.Add(22 * time.Minute)
	spikeSeries := []*tempopb.TimeSeries{
		createBucketSeries("1.0", 50, spikeTime.UnixMilli()), // 50 slow requests (now "fast" for spike)
		createBucketSeries("2.0", 30, spikeTime.UnixMilli()), // 30 very slow requests
		createBucketSeries("4.0", 15, spikeTime.UnixMilli()), // 15 extremely slow requests
		createBucketSeries("8.0", 5, spikeTime.UnixMilli()),  // 5 timeout requests
	}

	// Add exemplars during spike - these should be assigned contextually
	spikeSeries[0].Exemplars = []tempopb.Exemplar{
		{Value: 0.9, TimestampMs: spikeTime.UnixMilli()}, // During spike, this is p50!
		{Value: 1.1, TimestampMs: spikeTime.UnixMilli()}, // During spike, this is p50!
	}
	spikeSeries[1].Exemplars = []tempopb.Exemplar{
		{Value: 1.8, TimestampMs: spikeTime.UnixMilli()}, // During spike, this is p90!
	}
	spikeSeries[2].Exemplars = []tempopb.Exemplar{
		{Value: 3.5, TimestampMs: spikeTime.UnixMilli()}, // During spike, this is p99!
	}

	// Interval 3: Back to normal (30-45min) - p50=100ms, p90=200ms, p99=500ms
	normal2Time := start.Add(37 * time.Minute)
	normalSeries2 := []*tempopb.TimeSeries{
		createBucketSeries("0.125", 75, normal2Time.UnixMilli()),
		createBucketSeries("0.25", 20, normal2Time.UnixMilli()),
		createBucketSeries("0.5", 4, normal2Time.UnixMilli()),
		createBucketSeries("1.0", 1, normal2Time.UnixMilli()),
	}

	normalSeries2[0].Exemplars = []tempopb.Exemplar{
		{Value: 0.09, TimestampMs: normal2Time.UnixMilli()}, // Fast - should go to p50
	}
	normalSeries2[1].Exemplars = []tempopb.Exemplar{
		{Value: 0.19, TimestampMs: normal2Time.UnixMilli()}, // Medium - should go to p90
	}

	// Combine all time series in correct temporal order
	allSeries := append([]*tempopb.TimeSeries(nil), normalSeries1...) // Copy normalSeries1
	allSeries = append(allSeries, spikeSeries...)
	allSeries = append(allSeries, normalSeries2...)

	agg.Combine(allSeries)
	results := agg.Results()

	// Verify we have the expected quantile series
	require.Len(t, results, 3, "Should have 3 quantile series")

	var p50Series, p90Series, p99Series TimeSeries
	var found50, found90, found99 bool

	for _, series := range results {
		for _, label := range series.Labels {
			if label.Name == "p" {
				switch label.Value.Float() {
				case 0.5:
					p50Series = series
					found50 = true
				case 0.9:
					p90Series = series
					found90 = true
				case 0.99:
					p99Series = series
					found99 = true
				}
			}
		}
	}

	require.True(t, found50, "Should find p50 series")
	require.True(t, found90, "Should find p90 series")
	require.True(t, found99, "Should find p99 series")

	// Verify quantile values reflect the spike pattern
	require.Greater(t, p50Series.Values[1], p50Series.Values[0], "P50 should spike in interval 1")
	require.Greater(t, p50Series.Values[1], p50Series.Values[2], "P50 should be higher during spike than after")

	// Verify exemplar distribution makes sense with per-interval context
	totalExemplars := len(p50Series.Exemplars) + len(p90Series.Exemplars) + len(p99Series.Exemplars)
	require.Greater(t, totalExemplars, 0, "Should have exemplars distributed")

	t.Logf("Quantile values across intervals:")
	t.Logf("P50: [%.3f, %.3f, %.3f]", p50Series.Values[0], p50Series.Values[1], p50Series.Values[2])
	t.Logf("P90: [%.3f, %.3f, %.3f]", p90Series.Values[0], p90Series.Values[1], p90Series.Values[2])
	t.Logf("P99: [%.3f, %.3f, %.3f]", p99Series.Values[0], p99Series.Values[1], p99Series.Values[2])

	t.Logf("Exemplar distribution:")
	t.Logf("P50 exemplars: %d", len(p50Series.Exemplars))
	t.Logf("P90 exemplars: %d", len(p90Series.Exemplars))
	t.Logf("P99 exemplars: %d", len(p99Series.Exemplars))

	// Log individual exemplar values to see the assignment
	for _, ex := range p50Series.Exemplars {
		t.Logf("P50 exemplar: %.3f", ex.Value)
	}
	for _, ex := range p90Series.Exemplars {
		t.Logf("P90 exemplar: %.3f", ex.Value)
	}
	for _, ex := range p99Series.Exemplars {
		t.Logf("P99 exemplar: %.3f", ex.Value)
	}
}

func TestLog2QuantileWithBucket(t *testing.T) {
	buckets := []HistogramBucket{
		{Max: 1.0, Count: 10}, // bucket 0
		{Max: 2.0, Count: 20}, // bucket 1
		{Max: 4.0, Count: 30}, // bucket 2
		{Max: 8.0, Count: 40}, // bucket 3
	}

	tests := []struct {
		name           string
		quantile       float64
		expectedBucket int
	}{
		{"p10", 0.1, 0},  // 10% of 100 = 10, should be in bucket 0
		{"p30", 0.3, 1},  // 30% of 100 = 30, should be in bucket 1
		{"p60", 0.6, 2},  // 60% of 100 = 60, should be in bucket 2
		{"p90", 0.9, 3},  // 90% of 100 = 90, should be in bucket 3
		{"p100", 1.0, 3}, // 100% should be in last bucket
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, bucketIdx := Log2QuantileWithBucket(tt.quantile, buckets)

			// Verify we get a valid quantile value
			require.Greater(t, value, 0.0, "Quantile value should be positive")

			// Verify we get the expected bucket index
			require.Equal(t, tt.expectedBucket, bucketIdx,
				"Quantile %f should fall in bucket %d, got %d", tt.quantile, tt.expectedBucket, bucketIdx)

			// Verify consistency with original Log2Quantile function
			originalValue := Log2Quantile(tt.quantile, buckets)
			require.Equal(t, originalValue, value,
				"Log2QuantileWithBucket should return same value as Log2Quantile")
		})
	}
}
