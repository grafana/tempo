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

type AvgSeries[S StaticVals] struct {
	avg          []float64
	count        []float64
	compensation []float64
	init         bool
	vals         S
}

type AvgAggregator[F FastStatic, S StaticVals] struct {
	// Config
	by              []Attribute   // Original attributes: .foo
	byLookups       [][]Attribute // Lookups: span.foo resource.foo
	getSpanAttValue func(s Span) float64
	start           uint64
	end             uint64
	step            uint64

	// Data
	series     map[F]AvgSeries[S]
	lastSeries AvgSeries[S]
	buf        fastStaticWithValues[F, S]
	lastBuf    fastStaticWithValues[F, S]
}

type SimpleAverageSeriesAggregator struct {
	ss               SeriesSet
	countProm        map[string]string
	len              int
	start, end, step uint64
}

var (
	_ SpanAggregator           = (*AvgAggregator[FastStatic1, StaticVals1])(nil)
	_ metricsFirstStageElement = (*MetricsAverageAggregate)(nil)
	_ SeriesAggregator         = (*SimpleAverageSeriesAggregator)(nil)
)

func (b *SimpleAverageSeriesAggregator) Combine(in []*tempopb.TimeSeries) {
	newCountersTS := make(map[string][]float64)
	nan := math.Float64frombits(normalNaN)

	for _, ts := range in {
		counterPromLabel := ""
		if strings.Contains(ts.PromLabels, "_type") {
			counterPromLabel = getLabels(ts.Labels, "_type").String()
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
	for _, ts := range in {
		counterLabel, ok := b.countProm[ts.PromLabels]
		if !ok {
			// This is a counter label, we can skip it
			continue
		}
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
				avg := (currentAvg*currentCount + newAvg*newCount) / (currentCount + newCount)
				b.ss[ts.PromLabels].Values[pos] = avg
				b.ss[counterLabel].Values[pos] = currentCount + newCount
			}
		}
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

func (b *SimpleAverageSeriesAggregator) Results() SeriesSet {
	return b.ss
}

func (g *AvgAggregator[F, S]) Observe(span Span) {
	if !g.getGroupingValues(span) {
		return
	}

	s := g.getSeries()
	interval := IntervalOf(span.StartTimeUnixNanos(), g.start, g.end, g.step)
	if interval == -1 {
		return
	}
	inc := g.getSpanAttValue(span)
	if math.IsNaN(inc) {
		return
	}
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

func (g *AvgAggregator[F, S]) ObserveExemplar(_ Span, _ float64, _ uint64) {
}

func (g *AvgAggregator[F, S]) labelsFor(vals S, t string) (Labels, string) {
	if g.by == nil {
		serieLabel := make(Labels, 2)
		serieLabel[0] = Label{labels.MetricName, NewStaticString(metricsAggregateAvgOverTime.String())}
		if t != "" {
			serieLabel[1] = Label{"_type", NewStaticString(t)}
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
		labels = append(labels, Label{"_type", NewStaticString(t)})
	}

	return labels, labels.String()
}

func (g *AvgAggregator[F, S]) Series() SeriesSet {
	ss := SeriesSet{}

	for _, s := range g.series {
		labels, promLabelsAvg := g.labelsFor(s.vals, "")

		ss[promLabelsAvg] = TimeSeries{
			Labels:    labels,
			Values:    s.avg,
			Exemplars: []Exemplar{},
		}

		labels, promLabelsCount := g.labelsFor(s.vals, "count")
		ss[promLabelsCount] = TimeSeries{
			Labels:    labels,
			Values:    s.count,
			Exemplars: []Exemplar{},
		}
	}

	return ss
}

func (g *AvgAggregator[F, S]) getGroupingValues(span Span) bool {
	for i, lookups := range g.byLookups {
		val := lookup(lookups, span)
		g.buf.vals[i] = val
		g.buf.fast[i] = val.MapKey()
	}
	return true
}

// getSeries gets the series for the current span.
// It will reuse the last series if possible.
func (g *AvgAggregator[F, S]) getSeries() AvgSeries[S] {
	// Fast path
	if g.lastSeries.init && g.lastBuf.fast == g.buf.fast {
		return g.lastSeries
	}

	s, ok := g.series[g.buf.fast]
	if !ok {
		intervals := IntervalCount(g.start, g.end, g.step)
		s = AvgSeries[S]{
			init:         true,
			vals:         g.buf.vals,
			count:        make([]float64, intervals),
			avg:          make([]float64, intervals),
			compensation: make([]float64, intervals),
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

func NewAvgAggregator(attr Attribute, by []Attribute, start, end, step uint64) SpanAggregator {
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

	return &AvgAggregator[F, S]{
		series:          map[F]AvgSeries[S]{},
		getSpanAttValue: fn,
		by:              by,
		byLookups:       lookups,
		start:           start,
		end:             end,
		step:            step,
	}
}

type MetricsAverageAggregate struct {
	by         []Attribute
	attr       Attribute
	agg        SpanAggregator
	seriesAgg  SeriesAggregator
	exemplarFn getExemplar
	mode       AggregateMode
}

var _ metricsFirstStageElement = (*MetricsAverageAggregate)(nil)

func newMetricsAverageAggregateWithAttr(attr Attribute, by []Attribute) *MetricsAverageAggregate {
	return &MetricsAverageAggregate{
		attr: attr,
		by:   by,
	}
}

func (a *MetricsAverageAggregate) init(q *tempopb.QueryRangeRequest, mode AggregateMode) {
	exemplarFn := func(s Span) (float64, uint64) {
		return math.NaN(), a.spanStartTimeMs(s)
	}

	a.seriesAgg = &SimpleAverageSeriesAggregator{
		ss:        make(SeriesSet),
		countProm: make(map[string]string),
		len:       IntervalCount(q.Start, q.End, q.Step),
		start:     q.Start,
		end:       q.End,
		step:      q.Step,
	}

	if mode == AggregateModeRaw {
		a.agg = NewAvgAggregator(a.attr, a.by, q.Start, q.End, q.Step)
	}

	a.exemplarFn = exemplarFn
	a.mode = mode
}

func (a *MetricsAverageAggregate) observe(span Span) {
	a.agg.Observe(span)
}

func (a *MetricsAverageAggregate) observeExemplar(span Span) {
	v, ts := a.exemplarFn(span)
	a.agg.ObserveExemplar(span, v, ts)
}

func (a *MetricsAverageAggregate) observeSeries(ss []*tempopb.TimeSeries) {
	a.seriesAgg.Combine(ss)
}

func (a *MetricsAverageAggregate) result() SeriesSet {
	if a.agg != nil {
		return a.agg.Series()
	}

	// In the frontend-version the results come from
	// the job-level aggregator
	ss := a.seriesAgg.Results()
	if a.mode == AggregateModeFinal {
		for i := range ss {
			if strings.Contains(i, "_type") {
				delete(ss, i)
			}
		}
	}
	return ss
}

func (a *MetricsAverageAggregate) extractConditions(request *FetchSpansRequest) {
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

func (a *MetricsAverageAggregate) validate() error {
	if len(a.by) >= maxGroupBys {
		return newUnsupportedError(fmt.Sprintf("metrics group by %v values", len(a.by)))
	}
	return nil
}

func (a *MetricsAverageAggregate) spanStartTimeMs(s Span) uint64 {
	return s.StartTimeUnixNanos() / uint64(time.Millisecond)
}

func (a *MetricsAverageAggregate) String() string {
	return "avg(" + a.attr.String() + ")"
}
