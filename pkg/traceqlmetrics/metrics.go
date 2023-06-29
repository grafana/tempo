package traceqlmetrics

import (
	"context"
	"io"
	"math"

	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/pkg/errors"
)

type LatencyHistogram struct {
	buckets [64]int // Exponential buckets, powers of 2
}

func New(buckets [64]int) *LatencyHistogram {
	return &LatencyHistogram{buckets: buckets}
}

func (m *LatencyHistogram) Record(durationNanos uint64) {
	// Increment bucket that matches log2(duration)
	var bucket int
	if durationNanos >= 2 {
		bucket = int(math.Ceil(math.Log2(float64(durationNanos))))
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

	if math.IsNaN(p) ||
		p < 0 ||
		p > 1 ||
		m.Count() == 0 {
		return 0
	}

	// Maximum amount of samples to include. We round up to better handle
	// percentiles on low sample counts (<100).
	maxSamples := int(math.Ceil(p * float64(m.Count())))

	// Find the bucket where the percentile falls in
	// and the total sample count less than or equal
	// to that bucket.
	var total, bucket int
	for b, count := range m.buckets {
		if total+count <= maxSamples {
			bucket = b
			total += count

			if total < maxSamples {
				continue
			}
		}

		// We have enough
		break
	}

	// Fraction to interpolate between buckets, sample-count wise.
	// 0.5 means halfway
	var interp float64
	if maxSamples-total > 0 {
		interp = float64(maxSamples-total) / float64(m.buckets[bucket+1])
	}

	// Exponential interpolation between buckets
	minDur := math.Pow(2, float64(bucket))
	dur := minDur * math.Pow(2, interp)

	return uint64(dur)
}

// Buckets returns the bucket counts for each power of 2.
func (m *LatencyHistogram) Buckets() [64]int {
	return m.buckets
}

type MetricsResults struct {
	Estimated bool
	SpanCount int
	Series    map[traceql.Static]*LatencyHistogram
	Errors    map[traceql.Static]int
}

func NewMetricsResults() *MetricsResults {
	return &MetricsResults{
		Series: map[traceql.Static]*LatencyHistogram{},
		Errors: map[traceql.Static]int{},
	}
}

func (m *MetricsResults) Record(series traceql.Static, durationNanos uint64, err bool) {
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
func GetMetrics(ctx context.Context, query string, groupBy string, spanLimit int, start, end uint64, fetcher traceql.SpansetFetcher) (*MetricsResults, error) {
	groupByAttr, err := traceql.ParseIdentifier(groupBy)
	if err != nil {
		return nil, errors.Wrap(err, "parsing groupby")
	}

	eval, req, err := traceql.NewEngine().Compile(query)
	if err != nil {
		return nil, errors.Wrap(err, "compiling query")
	}

	var (
		duration   = traceql.NewIntrinsic(traceql.IntrinsicDuration)
		startTime  = traceql.NewIntrinsic(traceql.IntrinsicSpanStartTime)
		startValue = traceql.NewStaticInt(int(start))
		status     = traceql.NewIntrinsic(traceql.IntrinsicStatus)
		statusErr  = traceql.NewStaticStatus(traceql.StatusError)
		spanCount  = 0
		series     = NewMetricsResults()
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

	// Ensure that we select the span duration, status, and group-by attribute
	// if they are not already included in the query.
	addConditionIfNotPresent := func(a traceql.Attribute) {
		for _, c := range req.Conditions {
			if c.Attribute == a {
				return
			}
		}

		req.Conditions = append(req.Conditions, traceql.Condition{Attribute: a})
	}
	addConditionIfNotPresent(status)
	addConditionIfNotPresent(duration)
	addConditionIfNotPresent(groupByAttr)

	// Read the spans in the second pass callback and return nil to discard them.
	// We do this because it lets the fetch layer repool the spans because it
	// knows we discarded them.
	// TODO - Add span.Release() or something that we could use in the loop
	// at the bottom to repool the spans?
	req.SecondPass = func(s *traceql.Spanset) ([]*traceql.Spanset, error) {
		out, err := eval([]*traceql.Spanset{s})
		if err != nil {
			return nil, err
		}

		for _, ss := range out {
			for _, s := range ss.Spans {

				if start > 0 && s.StartTimeUnixNanos() < start {
					continue
				}
				if end > 0 && s.StartTimeUnixNanos() >= end {
					continue
				}

				var (
					attr  = s.Attributes()
					group = attr[groupByAttr]
					err   = attr[status] == statusErr
				)

				series.Record(group, s.DurationNanos(), err)

				spanCount++
				if spanLimit > 0 && spanCount >= spanLimit {
					return nil, io.EOF
				}
			}
		}

		return nil, err
	}

	// Perform the fetch and process the results inside the SecondPass
	// callback.  No actual results will be returned from this fetch call,
	// But we still need to call Next() at least once.
	res, err := fetcher.Fetch(ctx, *req)
	if err == util.ErrUnsupported {
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
	}

	// The results are estimated if we bailed early due to limit being reached, but only if spanLimit has been set.
	series.Estimated = spanCount >= spanLimit && spanLimit > 0
	series.SpanCount = spanCount
	return series, nil
}
