package traceql

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/prometheus/prometheus/model/labels"
)

// Average over time aggregator
type averageOverTimeAggregator struct {
	by   []Attribute
	attr Attribute
	// Average over time span aggregator
	agg SpanAggregator
	// Average over time series aggregator
	seriesAgg  SeriesAggregator
	exemplarFn getExemplar
	mode       AggregateMode
}

var _ metricsFirstStageElement = (*averageOverTimeAggregator)(nil)

func newAverageOverTimeMetricsAggregator(attr Attribute, by []Attribute) *averageOverTimeAggregator {
	return &averageOverTimeAggregator{
		attr: attr,
		by:   by,
	}
}

func (a *averageOverTimeAggregator) init(q *tempopb.QueryRangeRequest, mode AggregateMode) {
	exemplarFn := func(s Span) (float64, uint64) {
		return math.NaN(), a.spanStartTimeMs(s)
	}

	a.seriesAgg = &averageOverTimeSeriesAggregator{
		ss:              make(SeriesSet),
		countProm:       make(map[string]string),
		len:             IntervalCount(q.Start, q.End, q.Step),
		start:           q.Start,
		end:             q.End,
		step:            q.Step,
		exemplarBuckets: newBucketSet(IntervalCount(q.Start, q.End, q.Step)),
	}

	if mode == AggregateModeRaw {
		a.agg = newAvgOverTimeSpanAggregator(a.attr, a.by, q.Start, q.End, q.Step)
	}

	a.exemplarFn = exemplarFn
	a.mode = mode
}

func (a *averageOverTimeAggregator) observe(span Span) {
	a.agg.Observe(span)
}

func (a *averageOverTimeAggregator) observeExemplar(span Span) {
	v, ts := a.exemplarFn(span)
	a.agg.ObserveExemplar(span, v, ts)
}

func (a *averageOverTimeAggregator) observeSeries(ss []*tempopb.TimeSeries) {
	a.seriesAgg.Combine(ss)
}

func (a *averageOverTimeAggregator) result() SeriesSet {
	if a.agg != nil {
		return a.agg.Series()
	}

	// In the frontend-version the results come from
	// the job-level aggregator
	ss := a.seriesAgg.Results()
	if a.mode == AggregateModeFinal {
		for i := range ss {
			if strings.Contains(i, internalLabelMetaType) {
				delete(ss, i)
			}
		}
	}
	return ss
}

func (a *averageOverTimeAggregator) extractConditions(request *FetchSpansRequest) {
	// For metrics aggregators based on a span attribute we have to include it
	includeAttribute := a.attr != (Attribute{}) && !request.HasAttribute(a.attr)
	if includeAttribute {
		request.SecondPassConditions = append(request.SecondPassConditions, Condition{
			Attribute: a.attr,
		})
	}

	for _, b := range a.by {
		if !request.HasAttribute(b) {
			request.SecondPassConditions = append(request.SecondPassConditions, Condition{
				Attribute: b,
			})
		}
	}
}

func (a *averageOverTimeAggregator) validate() error {
	if len(a.by) >= maxGroupBys {
		return newUnsupportedError(fmt.Sprintf("metrics group by %v values", len(a.by)))
	}
	return nil
}

func (a *averageOverTimeAggregator) spanStartTimeMs(s Span) uint64 {
	return s.StartTimeUnixNanos() / uint64(time.Millisecond)
}

func (a *averageOverTimeAggregator) String() string {
	s := strings.Builder{}

	s.WriteString(metricsAggregateAvgOverTime.String())
	s.WriteString("(")
	if a.attr != (Attribute{}) {
		s.WriteString(a.attr.String())
	}
	s.WriteString(")")

	if len(a.by) > 0 {
		s.WriteString("by(")
		for i, b := range a.by {
			s.WriteString(b.String())
			if i < len(a.by)-1 {
				s.WriteString(",")
			}
		}
		s.WriteString(")")
	}
	return s.String()
}

type averageOverTimeSeriesAggregator struct {
	ss               SeriesSet
	countProm        map[string]string
	len              int
	start, end, step uint64
	exemplarBuckets  *bucketSet
}

var (
	_   SeriesAggregator = (*averageOverTimeSeriesAggregator)(nil)
	nan                  = math.Float64frombits(normalNaN)
)

func (b *averageOverTimeSeriesAggregator) Combine(in []*tempopb.TimeSeries) {
	newCountersTS := make(map[string][]float64)

	b.initSeriesAggregator(in, newCountersTS)
	for _, ts := range in {
		counterLabel, ok := b.countProm[ts.PromLabels]
		if !ok {
			// This is a counter label, we can skip it
			continue
		}
		existing := b.ss[ts.PromLabels]
		for _, sample := range ts.Samples {
			pos := IntervalOfMs(sample.TimestampMs, b.start, b.end, b.step)
			if pos < 0 || pos > len(b.ss[ts.PromLabels].Values) {
				continue
			}

			currentAvg := b.ss[ts.PromLabels].Values[pos]
			newAvg := sample.Value
			currentCount := b.ss[counterLabel].Values[pos]
			newCount := newCountersTS[ts.PromLabels][pos]

			if math.IsNaN(currentAvg) && !math.IsNaN(newAvg) {
				b.ss[ts.PromLabels].Values[pos] = newAvg
				b.ss[counterLabel].Values[pos] = newCount
			} else if !math.IsNaN(newAvg) {
				// Weighted mean
				avg := (currentAvg*currentCount + newAvg*newCount) / (currentCount + newCount)
				b.ss[ts.PromLabels].Values[pos] = avg
				b.ss[counterLabel].Values[pos] = currentCount + newCount
			}
		}

		b.aggregateExemplars(ts, &existing)
		b.ss[ts.PromLabels] = existing
	}
}

func (b *averageOverTimeSeriesAggregator) initSeriesAggregator(in []*tempopb.TimeSeries, newCountersTS map[string][]float64) {
	for _, ts := range in {
		counterPromLabel := ""
		if strings.Contains(ts.PromLabels, internalLabelMetaType) {
			counterPromLabel = getLabels(ts.Labels, internalLabelMetaType).String()
			newCountersTS[counterPromLabel] = make([]float64, b.len)
			for i, sample := range ts.Samples {
				newCountersTS[counterPromLabel][i] = sample.Value
			}
		}
		_, ok := b.ss[ts.PromLabels]
		if !ok {
			labels := getLabels(ts.Labels, "")
			n := TimeSeries{
				Labels:    labels,
				Values:    make([]float64, b.len),
				Exemplars: make([]Exemplar, 0, len(ts.Exemplars)),
			}
			if counterPromLabel != "" {
				b.countProm[counterPromLabel] = ts.PromLabels
			} else {
				for i := range n.Values {
					n.Values[i] = nan
				}
			}
			b.ss[ts.PromLabels] = n
		}
	}
}

func (b *averageOverTimeSeriesAggregator) aggregateExemplars(ts *tempopb.TimeSeries, existing *TimeSeries) {
	for _, exemplar := range ts.Exemplars {
		if b.exemplarBuckets.testTotal() {
			break
		}
		interval := IntervalOfMs(exemplar.TimestampMs, b.start, b.end, b.step)
		if b.exemplarBuckets.addAndTest(interval) {
			continue // Skip this exemplar and continue, next exemplar might fit in a different bucket	}
		}
		labels := make(Labels, 0, len(exemplar.Labels))
		for _, l := range exemplar.Labels {
			labels = append(labels, Label{
				Name:  l.Key,
				Value: StaticFromAnyValue(l.Value),
			})
		}
		value := exemplar.Value
		if math.IsNaN(value) {
			value = 0 // TODO: Use the value of the series at the same timestamp
		}
		existing.Exemplars = append(existing.Exemplars, Exemplar{
			Labels:      labels,
			Value:       value,
			TimestampMs: uint64(exemplar.TimestampMs),
		})
	}
}

func getLabels(vals []v1.KeyValue, skipKey string) Labels {
	labels := make(Labels, 0, len(vals))
	for _, l := range vals {
		if skipKey != "" && l.Key == skipKey {
			continue
		}
		labels = append(labels, Label{
			Name:  l.Key,
			Value: StaticFromAnyValue(l.Value),
		})
	}
	return labels
}

func (b *averageOverTimeSeriesAggregator) Results() SeriesSet {
	return b.ss
}

// Accumulated results of average over time
type avgOverTimeSeries[S StaticVals] struct {
	avg             []float64
	count           []float64
	compensation    []float64
	exemplars       []Exemplar
	exemplarBuckets *bucketSet
	init            bool
	vals            S
}

// In charge of calculating the average over time for a set of spans
// First aggregation layer
type avgOverTimeSpanAggregator[F FastStatic, S StaticVals] struct {
	// Config
	by              []Attribute   // Original attributes: .foo
	byLookups       [][]Attribute // Lookups: span.foo resource.foo
	getSpanAttValue func(s Span) float64
	start           uint64
	end             uint64
	step            uint64

	// Data
	series     map[F]avgOverTimeSeries[S]
	lastSeries avgOverTimeSeries[S]
	buf        fastStaticWithValues[F, S]
	lastBuf    fastStaticWithValues[F, S]
}

var _ SpanAggregator = (*avgOverTimeSpanAggregator[FastStatic1, StaticVals1])(nil)

func newAvgOverTimeSpanAggregator(attr Attribute, by []Attribute, start, end, step uint64) SpanAggregator {
	lookups := make([][]Attribute, len(by))
	for i, attr := range by {
		if attr.Intrinsic == IntrinsicNone && attr.Scope == AttributeScopeNone {
			// Unscoped attribute. Check span-level, then resource-level.
			// TODO - Is this taken care of by span.AttributeFor now?
			lookups[i] = []Attribute{
				NewScopedAttribute(AttributeScopeSpan, false, attr.Name),
				NewScopedAttribute(AttributeScopeResource, false, attr.Name),
			}
		} else {
			lookups[i] = []Attribute{attr}
		}
	}

	aggNum := len(lookups)

	switch aggNum {
	case 2:
		return newAvgAggregator[FastStatic2, StaticVals2](attr, by, lookups, start, end, step)
	case 3:
		return newAvgAggregator[FastStatic3, StaticVals3](attr, by, lookups, start, end, step)
	case 4:
		return newAvgAggregator[FastStatic4, StaticVals4](attr, by, lookups, start, end, step)
	case 5:
		return newAvgAggregator[FastStatic5, StaticVals5](attr, by, lookups, start, end, step)
	default:
		return newAvgAggregator[FastStatic1, StaticVals1](attr, by, lookups, start, end, step)
	}
}

func newAvgAggregator[F FastStatic, S StaticVals](attr Attribute, by []Attribute, lookups [][]Attribute, start, end, step uint64) SpanAggregator {
	var fn func(s Span) float64

	switch attr {
	case IntrinsicDurationAttribute:
		fn = func(s Span) float64 {
			return float64(s.DurationNanos()) / float64(time.Second)
		}
	default:
		fn = func(s Span) float64 {
			f, a := FloatizeAttribute(s, attr)
			if a == TypeNil {
				return math.Float64frombits(normalNaN)
			}
			return f
		}
	}

	return &avgOverTimeSpanAggregator[F, S]{
		series:          map[F]avgOverTimeSeries[S]{},
		getSpanAttValue: fn,
		by:              by,
		byLookups:       lookups,
		start:           start,
		end:             end,
		step:            step,
	}
}

func (g *avgOverTimeSpanAggregator[F, S]) Observe(span Span) {
	interval := IntervalOf(span.StartTimeUnixNanos(), g.start, g.end, g.step)
	if interval == -1 {
		return
	}

	inc := g.getSpanAttValue(span)
	if math.IsNaN(inc) {
		return
	}

	s := g.getSeries(span)

	s.count[interval]++
	mean, c := averageInc(s.avg[interval], inc, s.count[interval], s.compensation[interval])
	s.avg[interval] = mean
	s.compensation[interval] = c
}

func averageInc(mean, inc, count, compensation float64) (float64, float64) {
	if math.IsNaN(mean) && !math.IsNaN(inc) {
		// When we have a proper value in the span we need to initialize to 0
		mean = 0
	}
	if math.IsInf(mean, 0) {
		if math.IsInf(inc, 0) && (mean > 0) == (inc > 0) {
			// The `current.val` and `new` values are `Inf` of the same sign.  They
			// can't be subtracted, but the value of `current.val` is correct
			// already.
			return mean, compensation
		}
		if !math.IsInf(inc, 0) && !math.IsNaN(inc) {
			// At this stage, the current.val is an infinite. If the added
			// value is neither an Inf or a Nan, we can keep that mean
			// value.
			// This is required because our calculation below removes
			// the mean value, which would look like Inf += x - Inf and
			// end up as a NaN.
			return mean, compensation
		}
	}
	mean, c := kahanSumInc(inc/count-mean/count, mean, compensation)
	return mean, c
}

func kahanSumInc(inc, sum, c float64) (newSum, newC float64) {
	t := sum + inc
	switch {
	case math.IsInf(t, 0):
		c = 0

	// Using Neumaier improvement, swap if next term larger than sum.
	case math.Abs(sum) >= math.Abs(inc):
		c += (sum - t) + inc
	default:
		c += (inc - t) + sum
	}
	return t, c
}

func (g *avgOverTimeSpanAggregator[F, S]) ObserveExemplar(span Span, value float64, ts uint64) {
	// Observe exemplar
	all := span.AllAttributes()
	lbls := make(Labels, 0, len(all))
	for k, v := range span.AllAttributes() {
		lbls = append(lbls, Label{k.String(), v})
	}

	s := g.getSeries(span)

	if s.exemplarBuckets.testTotal() {
		return
	}
	interval := IntervalOfMs(int64(ts), g.start, g.end, g.step)
	if s.exemplarBuckets.addAndTest(interval) {
		return
	}

	s.exemplars = append(s.exemplars, Exemplar{
		Labels:      lbls,
		Value:       value,
		TimestampMs: ts,
	})
	g.series[g.buf.fast] = s
}

func (g *avgOverTimeSpanAggregator[F, S]) labelsFor(vals S, t string) (Labels, string) {
	if g.by == nil {
		serieLabel := make(Labels, 1, 2)
		serieLabel[0] = Label{labels.MetricName, NewStaticString(metricsAggregateAvgOverTime.String())}
		if t != "" {
			serieLabel = append(serieLabel, Label{internalLabelMetaType, NewStaticString(t)})
		}
		return serieLabel, serieLabel.String()
	}
	labels := make(Labels, 0, len(g.by)+1)
	for i := range g.by {
		if vals[i].Type == TypeNil {
			continue
		}
		labels = append(labels, Label{g.by[i].String(), vals[i]})
	}

	if len(labels) == 0 {
		// When all nil then force one
		labels = append(labels, Label{g.by[0].String(), NewStaticNil()})
	}

	if t != "" {
		labels = append(labels, Label{internalLabelMetaType, NewStaticString(t)})
	}

	return labels, labels.String()
}

func (g *avgOverTimeSpanAggregator[F, S]) Series() SeriesSet {
	ss := SeriesSet{}

	for _, s := range g.series {
		// First, get the regular series
		labels, promLabelsAvg := g.labelsFor(s.vals, "")
		ss[promLabelsAvg] = TimeSeries{
			Labels:    labels,
			Values:    s.avg,
			Exemplars: s.exemplars,
		}
		// Second, get the "count" series
		labels, promLabelsCount := g.labelsFor(s.vals, internalMetaTypeCount)
		ss[promLabelsCount] = TimeSeries{
			Labels:    labels,
			Values:    s.count,
			Exemplars: []Exemplar{},
		}
	}

	return ss
}

// getSeries gets the series for the current span.
// It will reuse the last series if possible.
func (g *avgOverTimeSpanAggregator[F, S]) getSeries(span Span) avgOverTimeSeries[S] {
	// Get Grouping values
	for i, lookups := range g.byLookups {
		val := lookup(lookups, span)
		g.buf.vals[i] = val
		g.buf.fast[i] = val.MapKey()
	}

	// Fast path
	if g.lastSeries.init && g.lastBuf.fast == g.buf.fast {
		return g.lastSeries
	}

	s, ok := g.series[g.buf.fast]
	if !ok {
		intervals := IntervalCount(g.start, g.end, g.step)
		s = avgOverTimeSeries[S]{
			init:            true,
			vals:            g.buf.vals,
			count:           make([]float64, intervals),
			avg:             make([]float64, intervals),
			compensation:    make([]float64, intervals),
			exemplars:       make([]Exemplar, 0, maxExemplars),
			exemplarBuckets: newBucketSet(intervals),
		}
		for i := 0; i < intervals; i++ {
			s.avg[i] = math.Float64frombits(normalNaN)
		}

		g.series[g.buf.fast] = s
	}

	g.lastBuf = g.buf
	g.lastSeries = s
	return s
}
