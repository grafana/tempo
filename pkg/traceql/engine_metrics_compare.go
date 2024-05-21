package traceql

import (
	"fmt"

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
		a.baselineAgg = NewGroupByEachAggregator([]Label{internalLabelTypeBaseline}, a.maxValues, func() RangeAggregator {
			return NewStepAggregator(q.Start, q.End, q.Step, func() VectorAggregator { return NewCountOverTimeAggregator() })
		})
		a.selectionAgg = NewGroupByEachAggregator([]Label{internalLabelTypeSelection}, a.maxValues, func() RangeAggregator {
			return NewStepAggregator(q.Start, q.End, q.Step, func() VectorAggregator { return NewCountOverTimeAggregator() })
		})

	case AggregateModeSum:
		a.seriesAgg = NewSimpleAdditionCombiner(q)
		return

	case AggregateModeFinal:
		a.seriesAgg = NewBaselineAggregator(q, a.maxValues)
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

	if a.maxValues <= 0 {
		return fmt.Errorf("compare() max number of values must be integer greater than 0")
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
	maxValues    int
	baselineAgg  SpanAggregator
	selectionAgg SpanAggregator
	seriesAgg    SeriesAggregator
}

func newMetricsCompare(f *SpansetFilter, maxValues, start, end int) *MetricsCompare {
	return &MetricsCompare{
		f:         f,
		maxValues: maxValues,
		start:     start,
		end:       end,
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
	prefix    Labels
	innerAgg  func() RangeAggregator
	maxValues int

	// Data
	maxed  map[Attribute]struct{}
	series map[Attribute]map[Static]RangeAggregator // Two layer map
}

func NewGroupByEachAggregator(prefix Labels, maxValues int, innerAgg func() RangeAggregator) *GroupByEach {
	return &GroupByEach{
		prefix:    prefix,
		maxValues: maxValues,
		innerAgg:  innerAgg,
		series:    map[Attribute]map[Static]RangeAggregator{},
		maxed:     make(map[Attribute]struct{}),
	}
}

func (g *GroupByEach) Observe(span Span) {
	m := span.AllAttributes()
	for a, v := range m {
		// These attributes get pulled back by select all but we never
		// group by them because I say so.
		switch a {
		case IntrinsicSpanStartTimeAttribute, IntrinsicDurationAttribute:
			continue
		}

		if _, ok := g.maxed[a]; ok {
			// This attribute reached max cardinality.
			// Stop counting
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

		if len(attrSeries) > g.maxValues {
			// This attribute is now exceeding max cardinality.
			// Delete the data and mark it for future input
			g.maxed[a] = struct{}{}
			delete(g.series, a)
			continue
		}

		agg.Observe(span)
	}
}

func (g *GroupByEach) Series() SeriesSet {
	ss := SeriesSet{}

	for a, m := range g.series {
		for v, agg := range m {
			labels := make(Labels, len(g.prefix)+1)
			copy(labels, g.prefix)
			labels[len(g.prefix)] = Label{Name: a.String(), Value: v}

			promLabels := labels.String()
			ss[promLabels] = TimeSeries{
				Labels: labels,
				Values: agg.Samples(),
			}
		}
	}

	// Create a series for each attribute that reached max cardinality.
	for a := range g.maxed {
		labels := make(Labels, len(g.prefix)+2)
		copy(labels, g.prefix)
		labels[len(g.prefix)] = Label{Name: a.String(), Value: NewStaticNil()}
		labels[len(g.prefix)+1] = internalLabelErrorTooManyValues

		promLabels := labels.String()
		ss[promLabels] = TimeSeries{
			Labels: labels,
			Values: nil,
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
	maxValues        int
	len              int
	start, end, step uint64
	baseline         map[string]map[Static]TimeSeries
	selection        map[string]map[Static]TimeSeries
	maxed            map[string]struct{}
}

func NewBaselineAggregator(req *tempopb.QueryRangeRequest, maxValues int) *BaselineAggregator {
	return &BaselineAggregator{
		baseline:  make(map[string]map[Static]TimeSeries),
		selection: make(map[string]map[Static]TimeSeries),
		maxed:     make(map[string]struct{}),
		len:       IntervalCount(req.Start, req.End, req.Step),
		start:     req.Start,
		end:       req.End,
		step:      req.Step,
		maxValues: maxValues,
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
				b.maxed[a] = struct{}{}
				delete(b.baseline, a)
				delete(b.selection, a)
			}
			// Skip remaining processing regardless of error type
			continue
		}

		if _, ok := b.maxed[a]; ok {
			// This attribute previous reached max values. Stop counting
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

		if len(attr) > b.maxValues {
			// This attribute just reached max cardinality overall (not within a sub-job)
			// Delete all data and mark it for future input
			b.maxed[a] = struct{}{}
			delete(b.baseline, a)
			delete(b.selection, a)
			continue
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

	add := func(buffer map[string]map[Static]TimeSeries, l Label) {
		for a, attr := range buffer {
			for v, ts := range attr {
				labels := Labels{
					l,
					{Name: a, Value: v},
				}
				output[labels.String()] = TimeSeries{
					Labels: labels,
					Values: ts.Values,
				}
			}
		}
	}

	add(b.baseline, internalLabelTypeBaseline)
	add(b.selection, internalLabelTypeSelection)

	for a := range b.maxed {
		labels := Labels{
			{Name: a, Value: NewStaticNil()},
			internalLabelErrorTooManyValues,
		}
		output[labels.String()] = TimeSeries{
			Labels: labels,
			Values: nil,
		}
	}

	return output
}

var _ SeriesAggregator = (*BaselineAggregator)(nil)
