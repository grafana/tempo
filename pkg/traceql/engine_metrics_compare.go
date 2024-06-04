package traceql

import (
	"fmt"
	"sort"

	"github.com/grafana/tempo/pkg/tempopb"
)

const (
	internalLabelMetaType          = "__meta_type"
	internalMetaTypeBaseline       = "baseline"
	internalMetaTypeSelection      = "selection"
	internalMetaTypeBaselineTotal  = "baseline_total"
	internalMetaTypeSelectionTotal = "selection_total"

	// internalLabelBaseline      = "__baseline"
	internalLabelError         = "__meta_error"
	internalErrorTooManyValues = "__too_many_values__"
)

var (
	internalLabelTypeBaseline       = Label{Name: internalLabelMetaType, Value: NewStaticString(internalMetaTypeBaseline)}
	internalLabelTypeBaselineTotal  = Label{Name: internalLabelMetaType, Value: NewStaticString(internalMetaTypeBaselineTotal)}
	internalLabelTypeSelection      = Label{Name: internalLabelMetaType, Value: NewStaticString(internalMetaTypeSelection)}
	internalLabelTypeSelectionTotal = Label{Name: internalLabelMetaType, Value: NewStaticString(internalMetaTypeSelectionTotal)}
	internalLabelErrorTooManyValues = Label{Name: internalLabelError, Value: NewStaticString(internalErrorTooManyValues)}
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
		a.baselines = make(map[Attribute]map[Static]int)
		a.selections = make(map[Attribute]map[Static]int)
		a.baselineTotals = make(map[Attribute]int)
		a.selectionTotals = make(map[Attribute]int)

	case AggregateModeSum:
		a.seriesAgg = NewSimpleAdditionCombiner(q)
		return

	case AggregateModeFinal:
		a.seriesAgg = NewBaselineAggregator(q, a.topN)
		return
	}
}

func (c *MetricsCompare) observe(span Span) {
	// Determine if this span is inside the selection
	isSelection := StaticFalse
	if c.start > 0 && c.end > 0 {
		// Timestamp filtering
		st := span.StartTimeUnixNanos()
		if st >= uint64(c.start) && st < uint64(c.end) {
			isSelection, _ = c.f.Expression.execute(span)
		}
	} else {
		// No timestamp filtering
		isSelection, _ = c.f.Expression.execute(span)
	}

	// Choose destination buffers
	dest := c.baselines
	destTotals := c.selectionTotals
	if isSelection == StaticTrue {
		dest = c.selections
		destTotals = c.selectionTotals
	}

	// Increment values for all attributes of this span
	span.AllAttributesFunc(func(a Attribute, v Static) {
		// These attributes get pulled back by select all but we never
		// group by them because I say so.
		// TODO - can we check type instead?
		switch a {
		case IntrinsicSpanStartTimeAttribute, IntrinsicDurationAttribute:
			return
		}

		counts, ok := dest[a]
		if !ok {
			counts = make(map[Static]int)
			dest[a] = counts
		}
		counts[v]++
		destTotals[a]++
	})
}

func (a *MetricsCompare) observeSeries(ss []*tempopb.TimeSeries) {
	a.seriesAgg.Combine(ss)
}

func (c *MetricsCompare) result() SeriesSet {
	// In the other modes return these results
	if c.seriesAgg != nil {
		return c.seriesAgg.Results()
	}

	var (
		top   = topN[Static]{}
		ss    = make(SeriesSet)
		erred = make(map[Attribute]struct{})
	)

	add := func(ls Labels, count int) {
		ss[ls.String()] = TimeSeries{
			Labels: ls,
			Values: []float64{0, float64(count)},
		}
	}

	addValues := func(prefix Label, data map[Attribute]map[Static]int) {
		for a, values := range data {
			// Compute topN values for this attribute
			top.reset()
			for v, count := range values {
				top.addOne(v, float64(count))
			}

			top.get(c.topN, func(v Static) {
				add(Labels{
					prefix,
					{Name: a.String(), Value: v},
				}, values[v])
			})

			if len(values) > c.topN {
				erred[a] = struct{}{}
			}
		}
	}

	addValues(internalLabelTypeBaseline, c.baselines)
	addValues(internalLabelTypeSelection, c.selections)

	// Add errors for attributes that hit the limit in either area
	for a := range erred {
		add(Labels{
			internalLabelErrorTooManyValues,
			{Name: a.String()},
		}, 0)
	}

	addTotals := func(prefix Label, data map[Attribute]int) {
		for a, count := range data {
			add(Labels{
				prefix,
				{Name: a.String()},
			}, count)
		}
	}

	addTotals(internalLabelTypeBaselineTotal, c.baselineTotals)
	addTotals(internalLabelTypeSelectionTotal, c.selectionTotals)

	return ss
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
	f               *SpansetFilter
	start, end, len int
	topN            int
	baselines       map[Attribute]map[Static]int
	selections      map[Attribute]map[Static]int
	baselineTotals  map[Attribute]int
	selectionTotals map[Attribute]int
	// aggFn           func() RangeAggregator
	seriesAgg SeriesAggregator
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
	totals           map[string]TimeSeries
	maxed            map[string]struct{}
}

func NewBaselineAggregator(req *tempopb.QueryRangeRequest, topN int) *BaselineAggregator {
	return &BaselineAggregator{
		baseline:  make(map[string]map[Static]TimeSeries),
		selection: make(map[string]map[Static]TimeSeries),
		totals:    make(map[string]TimeSeries),
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
		case internalMetaTypeBaselineTotal:
			ts, ok := b.totals[a]
			if !ok {
				ts = TimeSeries{
					Labels: LabelsFromProto(s.Labels),
					Values: make([]float64, b.len),
				}
				b.totals[a] = ts
			}
			for _, sample := range s.Samples {
				j := IntervalOfMs(sample.TimestampMs, b.start, b.end, b.step)
				if j >= 0 && j < len(ts.Values) {
					ts.Values[j] += sample.Value
				}
			}
			continue
		case internalMetaTypeSelectionTotal:
			ts, ok := b.totals[a]
			if !ok {
				ts = TimeSeries{
					Labels: LabelsFromProto(s.Labels),
					Values: make([]float64, b.len),
				}
				b.totals[a] = ts
			}
			for _, sample := range s.Samples {
				j := IntervalOfMs(sample.TimestampMs, b.start, b.end, b.step)
				if j >= 0 && j < len(ts.Values) {
					ts.Values[j] += sample.Value
				}
			}
			continue
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

	// Add series for every total
	for str, ts := range b.totals {
		output[str] = ts
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

func (t *topN[T]) addOne(key T, value float64) {
	t.entries = append(t.entries, struct {
		key   T
		total float64
	}{key, value})
}

// get the top N values. Given as a callback to avoid allocating.
// bool result indicates if there were more than N values
func (t *topN[T]) get(n int, cb func(key T)) {
	if len(t.entries) <= n {
		// <= N, no need to sort
		for _, e := range t.entries {
			cb(e.key)
		}
		return
	}

	sort.Slice(t.entries, func(i, j int) bool {
		return t.entries[i].total > t.entries[j].total // Sort descending
	})

	for i := 0; i < n; i++ {
		cb(t.entries[i].key)
	}
}

func (t *topN[T]) reset() {
	t.entries = t.entries[:0]
}
