package traceqlmetrics

import (
	"context"
	"io"
	"math"

	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/pkg/errors"
)

type latencyHistogram struct {
	buckets [64]int // Exponential buckets, powers of 2
}

func (m *latencyHistogram) Record(durationNanos uint64) {
	// Increment bucket that matches log2(duration)
	var bucket int
	if durationNanos >= 2 {
		bucket = int(math.Ceil(math.Log2(float64(durationNanos))))
	}
	m.buckets[bucket]++
}

func (m *latencyHistogram) Count() int {
	total := 0
	for _, count := range m.buckets {
		total += count
	}
	return total
}

func (m *latencyHistogram) Combine(other latencyHistogram) {
	for i := range m.buckets {
		m.buckets[i] += other.buckets[i]
	}
}

// Percentile returns the estimated latency percentile in nanoseconds.
func (m *latencyHistogram) Percentile(p float32) uint64 {

	// Maximum amount of samples to include. We round up to better handle
	// percentiles on low sample counts (<100).
	maxSamples := int(math.Ceil(float64(p) * float64(m.Count())))

	// Find the bucket where the percentile falls in
	// and the total sample count less than or equal
	// to that bucket.
	var total, bucket int
	for b, count := range m.buckets {
		if total+count < maxSamples {
			bucket = b
			total += count
			continue
		}

		// We have enough
		break
	}

	// Fraction to interpolate between buckets, sample-count wise.
	// 0.5 means halfway
	interp := float64(maxSamples-total) / float64(m.buckets[bucket+1])

	// Exponential interpolation between buckets
	minDur := math.Pow(2, float64(bucket))
	dur := minDur * math.Pow(2, interp)

	return uint64(dur)
}

type MetricsResults struct {
	Estimated bool
	SpanCount int
	Series    map[traceql.Static]*latencyHistogram
}

func (m *MetricsResults) Record(durationNanos uint64, series traceql.Static) {
	s := m.Series[series]
	if s == nil {
		s = &latencyHistogram{}
		m.Series[series] = s
	}
	s.Record(durationNanos)
}

// GetMetrics
func GetMetrics(ctx context.Context, query string, groupBy string, spanLimit int, fetcher traceql.SpansetFetcher) (*MetricsResults, error) {
	groupByAttr, err := traceql.ParseIdentifier(groupBy)
	if err != nil {
		return nil, errors.Wrap(err, "parsing groupby")
	}

	eval, req, err := traceql.NewEngine().Compile(query)
	if err != nil {
		return nil, errors.Wrap(err, "compiling query")
	}

	// Ensure that we select the span duration and group-by attribute
	// if they are not already included in the query. These are fetched
	// without filtering.
	addConditionIfNotPresent := func(a traceql.Attribute) {
		for _, c := range req.Conditions {
			if c.Attribute == a {
				return
			}
		}

		req.Conditions = append(req.Conditions, traceql.Condition{Attribute: a})
	}
	addConditionIfNotPresent(traceql.NewIntrinsic(traceql.IntrinsicDuration))
	addConditionIfNotPresent(groupByAttr)

	spanCount := 0
	series := &MetricsResults{
		Series: map[traceql.Static]*latencyHistogram{},
	}

	// This filter callback processes the matching spans into the
	// bucketed metrics.  It returns nil because we don't need any
	// results after this.
	req.Filter = func(in *traceql.Spanset) ([]*traceql.Spanset, error) {

		// Run engine to assert final query conditions
		out, err := eval([]*traceql.Spanset{in})
		if err != nil {
			return nil, err
		}

		for _, ss := range out {
			for _, s := range ss.Spans {
				series.Record(s.DurationNanos(), s.Attributes()[groupByAttr])

				spanCount++
				if spanCount >= spanLimit {
					return nil, io.EOF
				}
			}
		}
		return nil, nil
	}

	// Perform the fetch and process the results inside the Filter
	// callback.  No actual results will be returned from this fetch call,
	// But we still need to call Next() at least once.
	res, err := fetcher.Fetch(ctx, *req)
	if err == util.ErrUnsupported {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	for {
		ss, err := res.Results.Next(ctx)
		if err != nil {
			return nil, err
		}
		if ss == nil {
			break
		}
	}

	// The results are estimated if we bailed early due to limit being reached.
	series.Estimated = spanCount >= spanLimit
	series.SpanCount = spanCount
	return series, nil
}
