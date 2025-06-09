package traceql

import (
	"fmt"
	"math"
	"math/rand/v2"
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
			expected: 2, // 0, 1, even multiple
		},
		{
			start:    0,
			end:      10,
			step:     3,
			expected: 5, // 0, 3, 6, 9, 12
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
			start:    10, // aligned to 9
			step:     3,
			expected: 15, // 9, 12, 15 <-- intervals
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

func TestTrimToBlockOverlap(t *testing.T) {
	tc := []struct {
		start1, end1               string
		step                       time.Duration
		start2, end2               string
		expectedStart, expectedEnd string
		expectedStep               time.Duration
	}{
		{
			// Fully overlapping
			"2024-01-01 01:00:00", "2024-01-01 02:00:00", 5 * time.Minute,
			"2024-01-01 01:33:00", "2024-01-01 01:38:00",
			"2024-01-01 01:33:00", "2024-01-01 01:38:01", 5 * time.Minute, // added 1 second to include the last second of the block
		},
		{
			// Partially Overlapping
			"2024-01-01 01:01:00", "2024-01-01 02:01:00", 5 * time.Minute,
			"2024-01-01 01:31:00", "2024-01-01 02:31:00",
			"2024-01-01 01:31:00", "2024-01-01 02:01:00", 5 * time.Minute,
		},
		{
			// Instant query
			// Original range is 1h
			// Inner overlap is only 30m and step is updated to match
			"2024-01-01 01:00:00", "2024-01-01 02:00:00", time.Hour,
			"2024-01-01 01:30:00", "2024-01-01 02:30:00",
			"2024-01-01 01:30:00", "2024-01-01 02:00:00", 30 * time.Minute,
		},
	}

	for _, c := range tc {
		start1, _ := time.Parse(time.DateTime, c.start1)
		end1, _ := time.Parse(time.DateTime, c.end1)
		start2, _ := time.Parse(time.DateTime, c.start2)
		end2, _ := time.Parse(time.DateTime, c.end2)

		actualStart, actualEnd, actualStep := TrimToBlockOverlap(
			uint64(start1.UnixNano()),
			uint64(end1.UnixNano()),
			uint64(c.step.Nanoseconds()),
			start2,
			end2,
		)

		require.Equal(t, c.expectedStart, time.Unix(0, int64(actualStart)).UTC().Format(time.DateTime))
		require.Equal(t, c.expectedEnd, time.Unix(0, int64(actualEnd)).UTC().Format(time.DateTime))
		require.Equal(t, c.expectedStep, time.Duration(actualStep))
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
		Start: uint64(1 * time.Second),
		End:   uint64(3 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | quantile_over_time(duration, 0, 0.5, 1) by (span.foo)",
	}

	var (
		_128ns = 0.000000128
		_256ns = 0.000000256
		_512ns = 0.000000512
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
		Start: uint64(1 * time.Second),
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

func TestMinOverTimeForDuration(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: uint64(1 * time.Second),
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
		Start: uint64(1 * time.Second),
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
		Start: uint64(1 * time.Second),
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
		Start: uint64(1 * time.Second),
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
		Start: uint64(1 * time.Second),
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
		Start: uint64(1 * time.Second),
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
		Start: uint64(1 * time.Second),
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
		Start: uint64(1 * time.Second),
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

func TestMaxOverTimeForDuration(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: uint64(1 * time.Second),
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
		Start: uint64(1 * time.Second),
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
		Start: uint64(1 * time.Second),
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
		Start: uint64(1 * time.Second),
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
		Start: uint64(1 * time.Second),
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
		Start: uint64(1 * time.Second),
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
		Start: uint64(1 * time.Second),
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
		Start: uint64(1 * time.Second),
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

func TestSecondStageTopKAverage(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: uint64(1 * time.Second),
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
		Start: uint64(1 * time.Second),
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
	resultBar := result[`{"span.foo"="quax"}`]
	require.Equal(t, []float64{3, 3, 3, 3, 3, 3, 3, 3}, resultBar.Values)
	resultBaz := result[`{"span.foo"="baz"}`]
	require.Equal(t, []float64{5, 5, 5, 5, 5, 5, 5, 5}, resultBaz.Values)
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

func runTraceQLMetric(req *tempopb.QueryRangeRequest, inSpans ...[]Span) (SeriesSet, int, error) {
	e := NewEngine()

	layer2, err := e.CompileMetricsQueryRangeNonRaw(req, AggregateModeSum)
	if err != nil {
		return nil, 0, err
	}

	layer3, err := e.CompileMetricsQueryRangeNonRaw(req, AggregateModeFinal)
	if err != nil {
		return nil, 0, err
	}

	for _, spanSet := range inSpans {
		layer1, err := e.CompileMetricsQueryRange(req, 0, 0, false)
		if err != nil {
			return nil, 0, err
		}
		for _, s := range spanSet {
			layer1.metricsPipeline.observe(s)
		}
		res := layer1.Results()
		// Pass layer 1 to layer 2
		// These are partial counts over time by bucket
		layer2.metricsPipeline.observeSeries(res.ToProto(req))
	}

	// Pass layer 2 to layer 3
	// These are summed counts over time by bucket
	res := layer2.Results()
	layer3.ObserveSeries(res.ToProto(req))
	seriesCount := layer3.Length()
	// Layer 3 final results

	return layer3.Results(), seriesCount, nil
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
