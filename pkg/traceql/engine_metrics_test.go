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
			}, false, 0, false)

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
			}, tc.dedupe, 0, false)
			require.NoError(t, err)

			// Nil out func to Equal works
			eval.storageReq.SecondPass = nil
			require.Equal(t, tc.expectedReq, *eval.storageReq)
		})
	}
}

func TestQuantileOverTime(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: uint64(1 * time.Second),
		End:   uint64(3 * time.Second),
		Step:  uint64(1 * time.Second),
	}

	var (
		attr   = IntrinsicDurationAttribute
		qs     = []float64{0, 0.5, 1}
		by     = []Attribute{NewScopedAttribute(AttributeScopeSpan, false, "foo")}
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
		`{p="0", span.foo="bar"}`: TimeSeries{
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
		`{p="0.5", span.foo="bar"}`: TimeSeries{
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
		`{p="1", span.foo="bar"}`: TimeSeries{
			Labels: []Label{
				{Name: "span.foo", Value: NewStaticString("bar")},
				{Name: "p", Value: NewStaticFloat(1)},
			},
			Values: []float64{_512ns, _256ns, 0},
		},
		`{p="0", span.foo="baz"}`: TimeSeries{
			Labels: []Label{
				{Name: "span.foo", Value: NewStaticString("baz")},
				{Name: "p", Value: NewStaticFloat(0)},
			},
			Values: []float64{
				0, 0,
				percentileHelper(0, _512ns, _512ns, _512ns),
			},
		},
		`{p="0.5", span.foo="baz"}`: TimeSeries{
			Labels: []Label{
				{Name: "span.foo", Value: NewStaticString("baz")},
				{Name: "p", Value: NewStaticFloat(0.5)},
			},
			Values: []float64{
				0, 0,
				percentileHelper(0.5, _512ns, _512ns, _512ns),
			},
		},
		`{p="1", span.foo="baz"}`: TimeSeries{
			Labels: []Label{
				{Name: "span.foo", Value: NewStaticString("baz")},
				{Name: "p", Value: NewStaticFloat(1)},
			},
			Values: []float64{0, 0, _512ns},
		},
	}

	// 3 layers of processing matches:  query-frontend -> queriers -> generators -> blocks
	layer1 := newMetricsAggregateQuantileOverTime(attr, qs, by)
	layer1.init(req, AggregateModeRaw)

	layer2 := newMetricsAggregateQuantileOverTime(attr, qs, by)
	layer2.init(req, AggregateModeSum)

	layer3 := newMetricsAggregateQuantileOverTime(attr, qs, by)
	layer3.init(req, AggregateModeFinal)

	// Pass spans to layer 1
	for _, s := range in {
		layer1.observe(s)
	}

	// Pass layer 1 to layer 2
	// These are partial counts over time by bucket
	res := layer1.result()
	layer2.observeSeries(res.ToProto(req))

	// Pass layer 2 to layer 3
	// These are summed counts over time by bucket
	res = layer2.result()
	layer3.observeSeries(res.ToProto(req))

	// Layer 3 final results
	// The quantiles
	final := layer3.result()
	require.Equal(t, out, final)
}

func percentileHelper(q float64, values ...float64) float64 {
	h := Histogram{}
	for _, v := range values {
		h.Record(v, 1)
	}
	return Log2Quantile(q, h.Buckets)
}
