package traceql

import (
	"fmt"

	"github.com/grafana/tempo/pkg/tempopb"
)

const (
	internalLabelBaseline      = "__baseline"
	internalLabelError         = "__error"
	internalErrorTooManyValues = "__too_many_values__"
)

var internalLabelErrorTooManyValues = Label{
	Name:  internalLabelError,
	Value: NewStaticString(internalErrorTooManyValues),
}

/*type BaselineCompareRequest struct {
	Baseline      string
	BaselineStart uint64
	BaselineEnd   uint64
	Compare       string
	CompareStart  uint64
	CompareEnd    uint64
	MaxValues     uint32 // Max cardinality for a single attribute
}

type baselineComparisonSeries struct {
	baseline  bool
	Attribute Attribute
	Value     Static
}

func (e Engine) ExecuteBaselineComparison(ctx context.Context, req BaselineCompareRequest, f SpansetFetcher) (SeriesSet, error) {
	// Validation
	if req.BaselineStart == 0 || req.BaselineEnd <= req.BaselineStart {
		return nil, fmt.Errorf("invalid baseline time range")
	}

	if req.CompareStart == 0 ||
		req.CompareEnd <= req.CompareStart ||
		req.CompareEnd < req.BaselineStart ||
		req.CompareStart > req.BaselineEnd {
		return nil, fmt.Errorf("invalid comparison time range. must be enclosed within baseline time range (for now)")
	}

	exprB, err := Parse(req.Baseline)
	if err != nil {
		return nil, err
	}

	exprC, err := Parse(req.Compare)
	if err != nil {
		return nil, err
	}

	filterC, ok := exprC.Pipeline.Elements[0].(*SpansetFilter)
	if !ok {
		return nil, fmt.Errorf("invalid comparison query. must be single spanset filter like { <conditions> }")
	}

	storageReq := &FetchSpansRequest{
		StartTimeUnixNanos: req.BaselineStart,
		EndTimeUnixNanos:   req.BaselineEnd,
		SelectAll:          true,
		AllConditions:      true,
		SecondPass: func(ss *Spanset) ([]*Spanset, error) {
			return exprB.Pipeline.evaluate([]*Spanset{ss})
		},
		SecondPassConditions: []Condition{{Attribute: IntrinsicSpanStartTimeAttribute}},
	}

	// The baseline filters must be true for all spans,
	// so we can try for AllConditions and require them to
	// match.  Since we are selecting all, we don't need to
	// add it in conditions for the comparison.
	exprB.extractConditions(storageReq)

	res, err := f.Fetch(ctx, *storageReq)
	if err != nil {
		return nil, err
	}

	defer res.Results.Close()

	// Span count by attribute name, value, and baseline type
	output := map[baselineComparisonSeries]int{}
	buf := baselineComparisonSeries{}
	// cardinality := map[Attribute]int{}

	for {
		ss, err := res.Results.Next(ctx)
		if err != nil {
			return nil, err
		}
		if ss == nil {
			break
		}

		for _, s := range ss.Spans {

			// Check the span against the comparison window and expression.
			// If it matches then it's labeled with the comparison series
			// Else it's part of the baseline.
			buf.baseline = true
			st := s.StartTimeUnixNanos()
			if st >= req.CompareStart && st < req.CompareEnd {
				isC, err := filterC.Expression.execute(s)
				if err != nil {
					return nil, err
				}
				if isC == StaticTrue {
					buf.baseline = false
				}
			}

			// Increment counter for every attribute
			attrs := s.AllAttributes()
			for k, v := range attrs {
				// These attributes get pulled back by select all but we never
				// group by them because I say so.
				switch k {
				case IntrinsicDurationAttribute:
					continue
				}

				buf.Attribute = k
				buf.Value = v

				output[buf]++
			}
		}

		ss.Release()
	}

	// Deleted attributes that reach too high cardinality
	for k, v := range output {
		if v < 10 {
			delete(output, k)
		}
	}

	// Convert output
	output2 := SeriesSet{}
	for k, v := range output {
		labels := Labels{
			{Name: k.Attribute.String(), Value: k.Value},
			{Name: internalLabelBaseline, Value: NewStaticBool(k.baseline)},
		}
		s := TimeSeries{
			Labels: labels,
			Values: []float64{float64(v)},
		}
		output2[labels.String()] = s
	}

	return output2, nil
}*/

func (a *MetricsCompare) extractConditions(request *FetchSpansRequest) {
	request.SelectAll = true
	if !request.HasAttribute(IntrinsicSpanStartTimeAttribute) {
		request.SecondPassConditions = append(request.SecondPassConditions, Condition{Attribute: IntrinsicSpanStartTimeAttribute})
	}
	// We don't need to extract conditions from the comparison expression
	// because we're already selecting all.
}

func (a *MetricsCompare) init(q *tempopb.QueryRangeRequest, mode AggregateMode) {
	maxCardinality := 10

	switch mode {
	case AggregateModeRaw:
		a.baselineAgg = NewGroupByEachAggregator([]Label{{Name: internalLabelBaseline, Value: NewStaticString("true")}}, maxCardinality, func() RangeAggregator {
			return NewStepAggregator(q.Start, q.End, q.Step, func() VectorAggregator { return NewCountOverTimeAggregator() })
		})
		a.compareAgg = NewGroupByEachAggregator([]Label{{Name: internalLabelBaseline, Value: NewStaticString("false")}}, maxCardinality, func() RangeAggregator {
			return NewStepAggregator(q.Start, q.End, q.Step, func() VectorAggregator { return NewCountOverTimeAggregator() })
		})

	case AggregateModeSum:
		a.seriesAgg = NewSimpleAdditionCombiner(q)
		return

	case AggregateModeFinal:
		// a.seriesAgg = NewSimpleAdditionCombiner(q)
		// TODO
		// a.seriesAgg = NewComparisonCombiner
		a.seriesAgg = NewBaselineAggregator(q)
		return
	}

	// Raw mode:
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
		a.compareAgg.Observe(span)
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
		ss2 := a.compareAgg.Series()
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

	if a.start == 0 && a.end == 0 {
		return nil
	}

	if a.start <= 0 || a.end <= 0 {
		return fmt.Errorf("comparison timestamps must be positive integer unix nanoseconds")
	}
	if a.end <= a.start {
		return fmt.Errorf("comparison end timestamp must be greater than start timestamp")
	}
	return nil
}

func (a *MetricsCompare) String() string {
	return "compare_over_time(" + a.f.String() + "}"
}

type MetricsCompare struct {
	f           *SpansetFilter
	start, end  int
	baselineAgg SpanAggregator
	compareAgg  SpanAggregator
	seriesAgg   SeriesAggregator
}

func newMetricsCompare(f *SpansetFilter, start, end int) *MetricsCompare {
	return &MetricsCompare{
		f:     f,
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

type BaselineAggregator struct {
	maxValues        int
	len              int
	start, end, step uint64
	baseline         map[string]map[Static]TimeSeries
	compare          map[string]map[Static]TimeSeries
	maxed            map[string]struct{}
}

func NewBaselineAggregator(req *tempopb.QueryRangeRequest) *BaselineAggregator {
	return &BaselineAggregator{
		baseline:  make(map[string]map[Static]TimeSeries),
		compare:   make(map[string]map[Static]TimeSeries),
		maxed:     make(map[string]struct{}),
		len:       IntervalCount(req.Start, req.End, req.Step),
		start:     req.Start,
		end:       req.End,
		step:      req.Step,
		maxValues: 10,
	}
}

func (b *BaselineAggregator) Combine(ss []*tempopb.TimeSeries) {
	for _, s := range ss {
		isBaseline := false
		var error string
		var a string
		var v Static

		// Scan all labels
		for _, l := range s.Labels {
			switch l.Key {
			case internalLabelBaseline:
				isBaseline = l.Value.GetStringValue() == "true"
			case internalLabelError:
				error = l.Value.GetStringValue()
			default:
				a = l.Key
				v = StaticFromAnyValue(l.Value)
			}
		}

		// Check for errors on this attribute
		if error != "" {
			if error == internalErrorTooManyValues {
				// A sub-job reached max values for this attribute.
				b.maxed[a] = struct{}{}
				delete(b.baseline, a)
				delete(b.compare, a)
			}
			continue
		}

		if _, ok := b.maxed[a]; ok {
			// This attribute previous reached max values. Stop counting
			continue
		}

		// Merge this time series into the destination buffer
		dest := b.compare
		if isBaseline {
			dest = b.baseline
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
			delete(b.compare, a)
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

	append := func(buffer map[string]map[Static]TimeSeries, baseline string) {
		for a, attr := range buffer {
			for v, ts := range attr {
				labels := Labels{
					{Name: internalLabelBaseline, Value: NewStaticString(baseline)},
					{Name: a, Value: v},
				}
				output[labels.String()] = TimeSeries{
					Labels: labels,
					Values: ts.Values,
				}
			}
		}
	}

	append(b.baseline, "true")
	append(b.compare, "false")

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

	// Get list of all unique attributes in both datasets
	/*allAttributes := maps.Keys(b.baseline)
	for k := range b.compare {
		if _, ok := b.baseline[k]; !ok {
			// Attribute in comparison and not baseline
			allAttributes = append(allAttributes, k)
		}
	}

	for _, k := range allAttributes {

		allValues := maps.Keys(b.baseline[k])
		for v := range b.compare[k] {
			if _, ok := b.baseline[k][v]; !ok {
				allValues = append(allValues, v)
			}
		}

		if len(allValues) > b.maxValues {
		}
	}*/

	return output
}

var _ SeriesAggregator = (*BaselineAggregator)(nil)
