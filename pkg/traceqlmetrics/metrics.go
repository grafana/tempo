package traceqlmetrics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
)

const maxBuckets = 64

type LatencyHistogram struct {
	buckets [maxBuckets]int // Exponential buckets, powers of 2
}

func New(buckets [maxBuckets]int) *LatencyHistogram {
	return &LatencyHistogram{buckets: buckets}
}

func (m *LatencyHistogram) Record(durationNanos uint64) {
	// Increment bucket that matches log2(duration)
	var bucket int
	if durationNanos >= 2 {
		bucket = traceql.Log2Bucket(durationNanos)
	}
	if bucket >= maxBuckets {
		bucket = maxBuckets - 1
	}

	m.buckets[bucket]++
}

func (m *LatencyHistogram) Count() int {
	total := 0
	for _, count := range m.buckets {
		total += count
	}
	return total
}

func (m *LatencyHistogram) Combine(other LatencyHistogram) {
	for i := range m.buckets {
		m.buckets[i] += other.buckets[i]
	}
}

// Percentile returns the estimated latency percentile in nanoseconds.
func (m *LatencyHistogram) Percentile(p float64) uint64 {
	return uint64(traceql.Log2Quantile(p, m.buckets))
}

// Buckets returns the bucket counts for each power of 2.
func (m *LatencyHistogram) Buckets() [64]int {
	return m.buckets
}

const maxGroupBys = 5

type KeyValue struct {
	Key   string
	Value traceql.Static
}

type MetricSeries [maxGroupBys]KeyValue

type MetricsResults struct {
	Estimated bool
	SpanCount int
	Series    map[MetricSeries]*LatencyHistogram
	Errors    map[MetricSeries]int
}

func NewMetricsResults() *MetricsResults {
	return &MetricsResults{
		Series: map[MetricSeries]*LatencyHistogram{},
		Errors: map[MetricSeries]int{},
	}
}

func (m *MetricsResults) Record(series MetricSeries, durationNanos uint64, err bool) {
	s := m.Series[series]
	if s == nil {
		s = &LatencyHistogram{}
		m.Series[series] = s
	}
	s.Record(durationNanos)

	if err {
		m.Errors[series]++
	}
}

func (m *MetricsResults) Combine(other *MetricsResults) {
	m.SpanCount += other.SpanCount
	if other.Estimated {
		m.Estimated = true
	}

	for k, v := range other.Series {
		s := m.Series[k]
		if s == nil {
			s = &LatencyHistogram{}
			m.Series[k] = s
		}
		s.Combine(*v)
	}

	for k, v := range other.Errors {
		m.Errors[k] += v
	}
}

// GetMetrics
func GetMetrics(ctx context.Context, query, groupBy string, spanLimit int, start, end uint64, fetcher traceql.SpansetFetcher) (*MetricsResults, error) {
	identifiers := strings.Split(groupBy, ",")

	if len(identifiers) > maxGroupBys {
		return nil, fmt.Errorf("max group by %d attributes exceeded", maxGroupBys)
	}

	if len(identifiers) == 0 {
		return nil, errors.New("must group by at least one attribute")
	}

	// Parse each identifier to group by.
	// We also take any unscoped parameter and flatten it into the
	// scoped lookups. I.e. if we tell traceql storage we want
	// .foo it actually comes back as span.foo or resource.foo.
	// This is computed once upfront here to make the downstream
	// collection as efficient as possible.
	groupBys := make([][]traceql.Attribute, 0, len(identifiers))
	for _, id := range identifiers {

		id = strings.TrimSpace(id)

		attr, err := traceql.ParseIdentifier(id)
		if err != nil {
			return nil, fmt.Errorf("parsing groupby attribute: %w", err)
		}

		var lookups []traceql.Attribute
		if attr.Intrinsic == traceql.IntrinsicNone && attr.Scope == traceql.AttributeScopeNone {
			// Unscoped attribute. Also check span-level, then resource-level.
			lookups = []traceql.Attribute{
				attr,
				traceql.NewScopedAttribute(traceql.AttributeScopeSpan, false, attr.Name),
				traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, attr.Name),
			}
		} else {
			lookups = []traceql.Attribute{attr}
		}

		groupBys = append(groupBys, lookups)
	}

	groupByKeys := make([]string, len(groupBys))
	for i := range groupBys {
		groupByKeys[i] = groupBys[i][0].String()
	}

	_, eval, _, req, err := traceql.NewEngine().Compile(query)
	if err != nil {
		return nil, fmt.Errorf("compiling query: %w", err)
	}

	var (
		duration   = traceql.NewIntrinsic(traceql.IntrinsicDuration)
		startTime  = traceql.NewIntrinsic(traceql.IntrinsicSpanStartTime)
		startValue = traceql.NewStaticInt(int(start))
		status     = traceql.NewIntrinsic(traceql.IntrinsicStatus)
		statusErr  = traceql.NewStaticStatus(traceql.StatusError)
		spanCount  = 0
		results    = NewMetricsResults()
	)

	if start > 0 {
		req.StartTimeUnixNanos = start
		req.Conditions = append(req.Conditions, traceql.Condition{Attribute: startTime, Op: traceql.OpGreaterEqual, Operands: []traceql.Static{startValue}})
	}
	if end > 0 {
		req.EndTimeUnixNanos = end
		// There is only an intrinsic for the span start time, so use it as the cutoff.
		req.Conditions = append(req.Conditions, traceql.Condition{Attribute: startTime, Op: traceql.OpLess, Operands: []traceql.Static{startValue}})
	}

	// Ensure that we select the span duration, status, and group-by attributes
	// in the second pass if they are not already part of the first pass.
	addConditionIfNotPresent := func(a traceql.Attribute) {
		for _, c := range req.Conditions {
			if c.Attribute == a {
				return
			}
		}

		req.SecondPassConditions = append(req.SecondPassConditions, traceql.Condition{Attribute: a})
	}
	addConditionIfNotPresent(status)
	addConditionIfNotPresent(duration)
	for _, g := range groupBys {
		addConditionIfNotPresent(g[0])
	}

	req.SecondPass = func(s *traceql.Spanset) ([]*traceql.Spanset, error) {
		return eval([]*traceql.Spanset{s})
	}

	// Perform the fetch and process the results inside the SecondPass
	// callback.  No actual results will be returned from this fetch call,
	// But we still need to call Next() at least once.
	res, err := fetcher.Fetch(ctx, *req)
	if errors.Is(err, util.ErrUnsupported) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	defer res.Results.Close()

	for {
		ss, err := res.Results.Next(ctx)
		if err != nil {
			return nil, err
		}
		if ss == nil {
			break
		}

		for _, s := range ss.Spans {

			if start > 0 && s.StartTimeUnixNanos() < start {
				continue
			}
			if end > 0 && s.StartTimeUnixNanos() >= end {
				continue
			}

			var (
				series    = MetricSeries{}
				status, _ = s.AttributeFor(status)
				err       = status == statusErr
			)

			for i, g := range groupBys {
				series[i] = KeyValue{Key: groupByKeys[i], Value: lookup(g, s)}
			}

			results.Record(series, s.DurationNanos(), err)

			spanCount++
			if spanLimit > 0 && spanCount >= spanLimit {
				return nil, io.EOF
			}
		}

		ss.Release()
	}

	// The results are estimated if we bailed early due to limit being reached, but only if spanLimit has been set.
	results.Estimated = spanCount >= spanLimit && spanLimit > 0
	results.SpanCount = spanCount
	return results, nil
}

func lookup(needles []traceql.Attribute, span traceql.Span) traceql.Static {
	for _, n := range needles {
		if v, ok := span.AttributeFor(n); ok {
			return v
		}
	}

	return traceql.Static{}
}
