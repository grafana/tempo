package traceql

import (
	"fmt"
	"sort"

	"github.com/grafana/tempo/pkg/tempopb"
)

const (
	internalLabelMetaType     = "__meta_type"
	internalMetaTypeBaseline  = "baseline"
	internalMetaTypeSelection = "selection"

	// internalLabelBaseline      = "__baseline"
	internalLabelError         = "__meta_error"
	internalErrorTooManyValues = "__too_many_values__"
)

var (
	internalLabelTypeBaseline = Label{
		Name:  internalLabelMetaType,
		Value: NewStaticString(internalMetaTypeBaseline),
	}
	internalLabelTypeSelection = Label{
		Name:  internalLabelMetaType,
		Value: NewStaticString(internalMetaTypeSelection),
	}
	internalLabelErrorTooManyValues = Label{
		Name:  internalLabelError,
		Value: NewStaticString(internalErrorTooManyValues),
	}
)

func (a *MetricsCompare) extractConditions(request *FetchSpansRequest) {
	request.SecondPassSelectAll = true
	if !request.HasAttribute(IntrinsicSpanStartTimeAttribute) {
		request.SecondPassConditions = append(request.SecondPassConditions, Condition{Attribute: IntrinsicSpanStartTimeAttribute})
	}
	// We don't need to extract conditions from the comparison expression
	// because we're already selecting all.
}

func (a *MetricsCompare) init(q *tempopb.QueryRangeRequest, mode AggregateMode) {
	switch mode {
	case AggregateModeRaw:
		a.baselineAgg = NewGroupByEachAggregator([]Label{internalLabelTypeBaseline}, a.topN, func() RangeAggregator {
			return NewStepAggregator(q.Start, q.End, q.Step, func() VectorAggregator { return NewCountOverTimeAggregator() })
		})
		a.selectionAgg = NewGroupByEachAggregator([]Label{internalLabelTypeSelection}, a.topN, func() RangeAggregator {
			return NewStepAggregator(q.Start, q.End, q.Step, func() VectorAggregator { return NewCountOverTimeAggregator() })
		})

	case AggregateModeSum:
		a.seriesAgg = NewSimpleAdditionCombiner(q)
		return

	case AggregateModeFinal:
		a.seriesAgg = NewBaselineAggregator(q, a.topN)
		return
	}
}

func (a *MetricsCompare) observe(span Span) {
	isComparison := StaticFalse

	if a.start > 0 && a.end > 0 {
		// Timestamp filtering
		st := span.StartTimeUnixNanos()
		if st >= uint64(a.start) && st < uint64(a.end) {
			isComparison, _ = a.f.Expression.execute(span)
		}
	} else {
		// No timestamp filtering
		isComparison, _ = a.f.Expression.execute(span)
	}

	if isComparison == StaticTrue {
		a.selectionAgg.Observe(span)
	} else {
		a.baselineAgg.Observe(span)
	}
}

func (a *MetricsCompare) observeSeries(ss []*tempopb.TimeSeries) {
	a.seriesAgg.Combine(ss)
}

func (a *MetricsCompare) result() SeriesSet {
	if a.baselineAgg != nil {
		// Combine output
		ss := a.baselineAgg.Series()
		ss2 := a.selectionAgg.Series()
		for k, v := range ss2 {
			ss[k] = v
		}
		return ss
	}

	// In the frontend-version the results come from
	// the job-level aggregator
	return a.seriesAgg.Results()
}

func (a *MetricsCompare) validate() error {
	err := a.f.validate()
	if err != nil {
		return err
	}

	if a.topN <= 0 {
		return fmt.Errorf("compare() top number of values must be integer greater than 0")
	}

	if a.start == 0 && a.end == 0 {
		return nil
	}

	if a.start <= 0 || a.end <= 0 {
		return fmt.Errorf("compare() timestamps must be positive integer unix nanoseconds")
	}
	if a.end <= a.start {
		return fmt.Errorf("compare() end timestamp must be greater than start timestamp")
	}
	return nil
}

func (a *MetricsCompare) String() string {
	return "compare(" + a.f.String() + "}"
}

type MetricsCompare struct {
	f            *SpansetFilter
	start, end   int
	topN         int
	baselineAgg  SpanAggregator
	selectionAgg SpanAggregator
	seriesAgg    SeriesAggregator
}

func newMetricsCompare(f *SpansetFilter, topN, start, end int) *MetricsCompare {
	return &MetricsCompare{
		f:     f,
		topN:  topN,
		start: start,
		end:   end,
	}
}

var _ metricsFirstStageElement = (*MetricsCompare)(nil)

type AttributeValue struct {
	Attribute Attribute
	Value     Static
}

// GroupByEach is like the standard by() clause but flat.  It creates a single-depth series for every value
// of every attribute. I.e. a series for all distinct names, a series for all distinct services, etc.
type GroupByEach struct {
	// Config
	prefix   Labels
	innerAgg func() RangeAggregator
	topN     int

	// Data
	series map[Attribute]map[Static]RangeAggregator // Two layer map
}

func NewGroupByEachAggregator(prefix Labels, topN int, innerAgg func() RangeAggregator) *GroupByEach {
	return &GroupByEach{
		prefix:   prefix,
		topN:     topN,
		innerAgg: innerAgg,
		series:   map[Attribute]map[Static]RangeAggregator{},
	}
}

func (g *GroupByEach) Observe(span Span) {
	m := span.AllAttributes()
	for a, v := range m {
		// These attributes get pulled back by select all but we never
		// group by them because I say so.
		// TODO - can we check type instead?
		switch a {
		case IntrinsicSpanStartTimeAttribute, IntrinsicDurationAttribute:
			continue
		}

		attrSeries, ok := g.series[a]
		if !ok {
			attrSeries = make(map[Static]RangeAggregator)
			g.series[a] = attrSeries
		}

		agg, ok := attrSeries[v]
		if !ok {
			agg = g.innerAgg()
			attrSeries[v] = agg
		}

		agg.Observe(span)
	}
}

func (g *GroupByEach) Series() SeriesSet {
	ss := SeriesSet{}
	top := &topN[Static]{}

	add := func(labels Labels, agg RangeAggregator) {
		ls := make(Labels, 0, len(g.prefix)+len(labels))
		ls = append(ls, g.prefix...)
		ls = append(ls, labels...)

		promLabels := ls.String()
		ts := TimeSeries{}
		ts.Labels = ls

		if agg != nil {
			ts.Values = agg.Samples()
		}

		ss[promLabels] = ts
	}

	for a, m := range g.series {

		top.reset()
		for v, agg := range m {
			top.add(v, agg.Samples())
		}

		more := top.get(g.topN, func(key Static) {
			add(Labels{Label{Name: a.String(), Value: key}}, m[key])
		})

		if more {
			// Add too many values error
			add(Labels{internalLabelErrorTooManyValues, Label{a.String(), NewStaticNil()}}, nil)
		}
	}

	return ss
}

var _ SpanAggregator = (*GroupByEach)(nil)

// BaselineAggregate is a special series combiner for the compare() function.
// It resplits job-level results into baseline and selection buffers, and if
// an attribute reached max cardinality at the job-level, it will be marked
// as such at the query-level.
type BaselineAggregator struct {
	topN             int
	len              int
	start, end, step uint64
	baseline         map[string]map[Static]TimeSeries
	selection        map[string]map[Static]TimeSeries
	maxed            map[string]struct{}
}

func NewBaselineAggregator(req *tempopb.QueryRangeRequest, topN int) *BaselineAggregator {
	return &BaselineAggregator{
		baseline:  make(map[string]map[Static]TimeSeries),
		selection: make(map[string]map[Static]TimeSeries),
		maxed:     make(map[string]struct{}),
		len:       IntervalCount(req.Start, req.End, req.Step),
		start:     req.Start,
		end:       req.End,
		step:      req.Step,
		topN:      topN,
	}
}

func (b *BaselineAggregator) Combine(ss []*tempopb.TimeSeries) {
	for _, s := range ss {
		var metaType string
		var err string
		var a string
		var v Static

		// Scan all labels
		for _, l := range s.Labels {
			switch l.Key {
			case internalLabelMetaType:
				metaType = l.Value.GetStringValue()
			case internalLabelError:
				err = l.Value.GetStringValue()
			default:
				a = l.Key
				v = StaticFromAnyValue(l.Value)
			}
		}

		// Check for errors on this attribute
		if err != "" {
			if err == internalErrorTooManyValues {
				// A sub-job reached max values for this attribute.
				// Record the error
				b.maxed[a] = struct{}{}
			}
			// Skip remaining processing regardless of error type
			continue
		}

		// Merge this time series into the destination buffer
		// based on meta type
		var dest map[string]map[Static]TimeSeries
		switch metaType {
		case internalMetaTypeBaseline:
			dest = b.baseline
		case internalMetaTypeSelection:
			dest = b.selection
		default:
			// Unknown type, ignore
			continue
		}

		attr, ok := dest[a]
		if !ok {
			attr = make(map[Static]TimeSeries)
			dest[a] = attr
		}

		val, ok := attr[v]
		if !ok {
			val = TimeSeries{
				Labels: Labels{
					{Name: a, Value: v},
				},
				Values: make([]float64, b.len),
			}
			attr[v] = val
		}

		if len(attr) > b.topN {
			// This attribute just reached max cardinality overall (not within a sub-job)
			// Record the error
			b.maxed[a] = struct{}{}
		}

		for _, sample := range s.Samples {
			j := IntervalOfMs(sample.TimestampMs, b.start, b.end, b.step)
			if j >= 0 && j < len(val.Values) {
				val.Values[j] += sample.Value
			}
		}
	}
}

func (b *BaselineAggregator) Results() SeriesSet {
	output := make(SeriesSet)
	topN := &topN[Static]{}

	addSeries := func(prefix Label, name string, value Static, samples []float64) {
		ls := Labels{
			prefix,
			{Name: name, Value: value},
		}
		output[ls.String()] = TimeSeries{
			Labels: ls,
			Values: samples,
		}
	}

	do := func(buffer map[string]map[Static]TimeSeries, prefix Label) {
		for a, m := range buffer {

			topN.reset()
			for v, ts := range m {
				topN.add(v, ts.Values)
			}

			topN.get(b.topN, func(key Static) {
				addSeries(prefix, a, key, m[key].Values)
			})
		}
	}

	do(b.baseline, internalLabelTypeBaseline)
	do(b.selection, internalLabelTypeSelection)

	// Add series for every attribute that exceeded max value.
	for a := range b.maxed {
		addSeries(internalLabelErrorTooManyValues, a, NewStaticNil(), nil)
	}

	return output
}

var _ SeriesAggregator = (*BaselineAggregator)(nil)

// topN is a helper struct that gets the topN keys based on total sum
type topN[T any] struct {
	entries []struct {
		key   T
		total float64
	}
}

func (t *topN[T]) add(key T, values []float64) {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	t.entries = append(t.entries, struct {
		key   T
		total float64
	}{key, sum})
}

// get the top N values. Given as a callback to avoid allocating.
// bool result indicates if there were more than N values
func (t *topN[T]) get(n int, cb func(key T)) (more bool) {
	if len(t.entries) <= n {
		for _, e := range t.entries {
			cb(e.key)
		}
		return false
	}

	sort.Slice(t.entries, func(i, j int) bool {
		return t.entries[i].total > t.entries[j].total // Sort descending
	})

	for i := 0; i < n; i++ {
		cb(t.entries[i].key)
	}

	return true
}

func (t *topN[T]) reset() {
	t.entries = t.entries[:0]
}
