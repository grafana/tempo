package traceql

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"hash/fnv"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1proto "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/grafana/tempo/pkg/util"
	"github.com/prometheus/prometheus/model/labels"
)

const internalLabelBucket = "__bucket"

func DefaultQueryRangeStep(start, end uint64) uint64 {
	delta := time.Duration(end - start)

	// Try to get this many data points
	// Our baseline is is 1 hour @ 15s intervals
	baseline := delta / 240

	// Round down in intervals of 5s
	interval := baseline / (5 * time.Second) * (5 * time.Second)

	if interval < 5*time.Second {
		// Round down in intervals of 1s
		interval = baseline / time.Second * time.Second
	}

	if interval < time.Second {
		return uint64(time.Second.Nanoseconds())
	}

	return uint64(interval.Nanoseconds())
}

// IntervalCount is the number of intervals in the range with step.
func IntervalCount(start, end, step uint64) int {
	intervals := (end - start) / step
	intervals++
	return int(intervals)
}

// TimestampOf the given interval with the start and step.
func TimestampOf(interval, start, step uint64) uint64 {
	return start + interval*step
}

// IntervalOf the given timestamp within the range and step.
func IntervalOf(ts, start, end, step uint64) int {
	if ts < start || ts > end || end == start || step == 0 {
		// Invalid
		return -1
	}

	return int((ts - start) / step)
}

// IntervalOfMs is the same as IntervalOf except the input timestamp is in unix milliseconds.
func IntervalOfMs(tsmills int64, start, end, step uint64) int {
	ts := uint64(time.Duration(tsmills) * time.Millisecond)
	return IntervalOf(ts, start, end, step)
}

// TrimToOverlap returns the aligned overlap between the two given time ranges.
func TrimToOverlap(start1, end1, step, start2, end2 uint64) (uint64, uint64) {
	start1 = max(start1, start2)
	end1 = min(end1, end2)
	start1 = (start1 / step) * step
	end1 = (end1/step)*step + step
	return start1, end1
}

type Label struct {
	Name  string
	Value Static
}

type Labels []Label

func LabelsFromProto(ls []v1.KeyValue) Labels {
	out := make(Labels, 0, len(ls))
	for _, l := range ls {
		out = append(out, Label{Name: l.Key, Value: StaticFromAnyValue(l.Value)})
	}
	return out
}

// String returns the prometheus-formatted version of the labels. Which is downcasting
// the typed TraceQL values to strings, with some special casing.
func (ls Labels) String() string {
	promLabels := labels.NewBuilder(nil)
	for _, l := range ls {
		var promValue string
		switch {
		case l.Value.Type == TypeNil:
			promValue = "<nil>"
		case l.Value.Type == TypeString:
			s := l.Value.EncodeToString(false)
			if s == "" {
				promValue = "<empty>"
			} else {
				promValue = s
			}
		default:
			promValue = l.Value.EncodeToString(false)
		}
		promLabels.Set(l.Name, promValue)
	}

	return promLabels.Labels().String()
}

type TimeSeries struct {
	Labels Labels
	Values []float64
}

// SeriesSet is a set of unique timeseries. They are mapped by the "Prometheus"-style
// text description: {x="a",y="b"} for convenience.
type SeriesSet map[string]TimeSeries

func (set SeriesSet) ToProto(req *tempopb.QueryRangeRequest) []*tempopb.TimeSeries {
	resp := make([]*tempopb.TimeSeries, 0, len(set))

	for promLabels, s := range set {
		labels := make([]commonv1proto.KeyValue, 0, len(s.Labels))
		for _, label := range s.Labels {
			labels = append(labels,
				commonv1proto.KeyValue{
					Key:   label.Name,
					Value: label.Value.AsAnyValue(),
				},
			)
		}

		intervals := IntervalCount(req.Start, req.End, req.Step)
		samples := make([]tempopb.Sample, 0, intervals)
		for i, value := range s.Values {
			ts := TimestampOf(uint64(i), req.Start, req.Step)
			samples = append(samples, tempopb.Sample{
				TimestampMs: time.Unix(0, int64(ts)).UnixMilli(),
				Value:       value,
			})
		}

		ss := &tempopb.TimeSeries{
			PromLabels: promLabels,
			Labels:     labels,
			Samples:    samples,
		}

		resp = append(resp, ss)
	}

	return resp
}

// VectorAggregator turns a vector of spans into a single numeric scalar
type VectorAggregator interface {
	Observe(s Span)
	Sample() float64
}

// RangeAggregator sorts spans into time slots
// TODO - for efficiency we probably combine this with VectorAggregator (see todo about CountOverTimeAggregator)
type RangeAggregator interface {
	Observe(s Span)
	Samples() []float64
}

// SpanAggregator sorts spans into series
type SpanAggregator interface {
	Observe(Span)
	Series() SeriesSet
}

// CountOverTimeAggregator counts the number of spans. It can also
// calculate the rate when given a multiplier.
// TODO - Rewrite me to be []float64 which is more efficient
type CountOverTimeAggregator struct {
	count    float64
	rateMult float64
}

var _ VectorAggregator = (*CountOverTimeAggregator)(nil)

func NewCountOverTimeAggregator() *CountOverTimeAggregator {
	return &CountOverTimeAggregator{
		rateMult: 1.0,
	}
}

func NewRateAggregator(rateMult float64) *CountOverTimeAggregator {
	return &CountOverTimeAggregator{
		rateMult: rateMult,
	}
}

func (c *CountOverTimeAggregator) Observe(_ Span) {
	c.count++
}

func (c *CountOverTimeAggregator) Sample() float64 {
	return c.count * c.rateMult
}

// StepAggregator sorts spans into time slots using a step interval like 30s or 1m
type StepAggregator struct {
	start   uint64
	end     uint64
	step    uint64
	vectors []VectorAggregator
}

var _ RangeAggregator = (*StepAggregator)(nil)

func NewStepAggregator(start, end, step uint64, innerAgg func() VectorAggregator) *StepAggregator {
	intervals := IntervalCount(start, end, step)
	vectors := make([]VectorAggregator, intervals)
	for i := range vectors {
		vectors[i] = innerAgg()
	}

	return &StepAggregator{
		start:   start,
		end:     end,
		step:    step,
		vectors: vectors,
	}
}

func (s *StepAggregator) Observe(span Span) {
	interval := IntervalOf(span.StartTimeUnixNanos(), s.start, s.end, s.step)
	if interval == -1 {
		return
	}
	s.vectors[interval].Observe(span)
}

func (s *StepAggregator) Samples() []float64 {
	ss := make([]float64, len(s.vectors))
	for i, v := range s.vectors {
		ss[i] = v.Sample()
	}
	return ss
}

const maxGroupBys = 5 // TODO - This isn't ideal but see comment below.

// FastValues is an array of attribute values (static values) that can be used
// as a map key.  This offers good performance and works with native Go maps and
// has no chance for collisions (whereas a hash32 has a non-zero chance of
// collisions).  However, it means we have to arbitrarily set an upper limit on
// the maximum number of values.

type (
	FastValues1 [1]Static
	FastValues2 [2]Static
	FastValues3 [3]Static
	FastValues4 [4]Static
	FastValues5 [5]Static
)

// GroupingAggregator groups spans into series based on attribute values.
type GroupingAggregator[FV FastValues1 | FastValues2 | FastValues3 | FastValues4 | FastValues5] struct {
	// Config
	by          []Attribute               // Original attributes: .foo
	byLookups   [][]Attribute             // Lookups: span.foo resource.foo
	byFunc      func(Span) (Static, bool) // Dynamic label calculated by a callback
	byFuncLabel string                    // Name of the dynamic label
	innerAgg    func() RangeAggregator

	// Data
	series     map[FV]RangeAggregator
	lastSeries RangeAggregator
	buf        FV
	lastBuf    FV
}

var _ SpanAggregator = (*GroupingAggregator[FastValues5])(nil)

func NewGroupingAggregator(aggName string, innerAgg func() RangeAggregator, by []Attribute, byFunc func(Span) (Static, bool), byFuncLabel string) SpanAggregator {
	if len(by) == 0 && byFunc == nil {
		return &UngroupedAggregator{
			name:     aggName,
			innerAgg: innerAgg(),
		}
	}

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
	if byFunc != nil {
		aggNum++
	}

	switch aggNum {
	case 1:
		return newGroupingAggregator[FastValues1](innerAgg, by, byFunc, byFuncLabel, lookups)
	case 2:
		return newGroupingAggregator[FastValues2](innerAgg, by, byFunc, byFuncLabel, lookups)
	case 3:
		return newGroupingAggregator[FastValues3](innerAgg, by, byFunc, byFuncLabel, lookups)
	case 4:
		return newGroupingAggregator[FastValues4](innerAgg, by, byFunc, byFuncLabel, lookups)
	case 5:
		return newGroupingAggregator[FastValues5](innerAgg, by, byFunc, byFuncLabel, lookups)
	default:
		panic("unsupported number of group-bys")
	}
}

func newGroupingAggregator[FV FastValues1 | FastValues2 | FastValues3 | FastValues4 | FastValues5](innerAgg func() RangeAggregator, by []Attribute, byFunc func(Span) (Static, bool), byFuncLabel string, lookups [][]Attribute) SpanAggregator {
	return &GroupingAggregator[FV]{
		series:      map[FV]RangeAggregator{},
		by:          by,
		byFunc:      byFunc,
		byFuncLabel: byFuncLabel,
		byLookups:   lookups,
		innerAgg:    innerAgg,
	}
}

// Observe the span by looking up its group-by attributes, mapping to the series,
// and passing to the inner aggregate.  This is a critical hot path.
func (g *GroupingAggregator[FV]) Observe(span Span) {
	// Get grouping values
	// Reuse same buffer
	// There is no need to reset, the number of group-by attributes
	// is fixed after creation.
	for i, lookups := range g.byLookups {
		g.buf[i] = lookup(lookups, span)
	}

	// If dynamic label exists calculate and append it
	if g.byFunc != nil {
		v, ok := g.byFunc(span)
		if !ok {
			// Totally drop this span
			return
		}
		g.buf[len(g.byLookups)] = v
	}

	if g.lastSeries != nil && g.lastBuf == g.buf {
		g.lastSeries.Observe(span)
		return
	}

	agg, ok := g.series[g.buf]
	if !ok {
		agg = g.innerAgg()
		g.series[g.buf] = agg
	}

	g.lastBuf = g.buf
	g.lastSeries = agg
	agg.Observe(span)
}

// labelsFor gives the final labels for the series. Slower and not on the hot path.
// This is tweaked to match what prometheus does where possible with an exception.
// In the case of all values missing.
// (1) Standard case: a label is created for each group-by value in the series:
//
//	Ex: rate() by (x,y) can yield:
//	{x=a,y=b}
//	{x=c,y=d}
//	etc
//
// (2) Nils are dropped. A nil can be present for any label, so any combination
// of the remaining labels is possible. Label order is preserved.
//
//	Ex: rate() by (x,y,z) can yield all of these combinations:
//	{x=..., y=..., z=...}
//	{x=...,        z=...}
//	{x=...              }
//	{       y=..., z=...}
//	{       y=...       }
//	etc
//
// (3) Exceptional case: All Nils. For the TraceQL data-type aware labels we still drop
// all nils which results in an empty label set. But Prometheus-style always have
// at least 1 label, so in that case we have to force at least 1 label or else things
// may not be handled correctly downstream.  In this case we take the first label and
// make it the string "nil"
//
//	Ex: rate() by (x,y,z) and all nil yields:
//	{x="nil"}
func (g *GroupingAggregator[FV]) labelsFor(vals FV) (Labels, string) {
	labels := make(Labels, 0, len(g.by)+1)
	for i := range g.by {
		if vals[i].Type == TypeNil {
			continue
		}
		labels = append(labels, Label{g.by[i].String(), vals[i]})
	}
	if g.byFunc != nil {
		labels = append(labels, Label{g.byFuncLabel, vals[len(g.by)]})
	}

	if len(labels) == 0 {
		// When all nil then force one
		labels = append(labels, Label{g.by[0].String(), NewStaticNil()})
	}

	return labels, labels.String()
}

func (g *GroupingAggregator[FV]) Series() SeriesSet {
	ss := SeriesSet{}

	for vals, agg := range g.series {
		labels, promLabels := g.labelsFor(vals)

		ss[promLabels] = TimeSeries{
			Labels: labels,
			Values: agg.Samples(),
		}
	}

	return ss
}

// UngroupedAggregator builds a single series with no labels. e.g. {} | rate()
type UngroupedAggregator struct {
	name     string
	innerAgg RangeAggregator
}

var _ SpanAggregator = (*UngroupedAggregator)(nil)

func (u *UngroupedAggregator) Observe(span Span) {
	u.innerAgg.Observe(span)
}

// Series output.
// This is tweaked to match what prometheus does.  For ungrouped metrics we
// fill in a placeholder metric name with the name of the aggregation.
// rate() => {__name__=rate}
func (u *UngroupedAggregator) Series() SeriesSet {
	l := labels.FromStrings(labels.MetricName, u.name)
	return SeriesSet{
		l.String(): {
			Labels: []Label{{labels.MetricName, NewStaticString(u.name)}},
			Values: u.innerAgg.Samples(),
		},
	}
}

func (e *Engine) CompileMetricsQueryRangeNonRaw(req *tempopb.QueryRangeRequest, mode AggregateMode) (*MetricsFrontendEvaluator, error) {
	if req.Start <= 0 {
		return nil, fmt.Errorf("start required")
	}
	if req.End <= 0 {
		return nil, fmt.Errorf("end required")
	}
	if req.End <= req.Start {
		return nil, fmt.Errorf("end must be greater than start")
	}
	if req.Step <= 0 {
		return nil, fmt.Errorf("step required")
	}

	_, _, metricsPipeline, _, err := e.Compile(req.Query)
	if err != nil {
		return nil, fmt.Errorf("compiling query: %w", err)
	}

	if metricsPipeline == nil {
		return nil, fmt.Errorf("not a metrics query")
	}

	metricsPipeline.init(req, mode)

	return &MetricsFrontendEvaluator{
		metricsPipeline: metricsPipeline,
	}, nil
}

// CompileMetricsQueryRange returns an evalulator that can be reused across multiple data sources.
// Dedupe spans parameter is an indicator of whether to expect duplicates in the datasource. For
// example if the datasource is replication factor=1 or only a single block then we know there
// aren't duplicates and we can make some optimizations.
func (e *Engine) CompileMetricsQueryRange(req *tempopb.QueryRangeRequest, dedupeSpans bool, timeOverlapCutoff float64, allowUnsafeQueryHints bool) (*MetricsEvalulator, error) {
	if req.Start <= 0 {
		return nil, fmt.Errorf("start required")
	}
	if req.End <= 0 {
		return nil, fmt.Errorf("end required")
	}
	if req.End <= req.Start {
		return nil, fmt.Errorf("end must be greater than start")
	}
	if req.Step <= 0 {
		return nil, fmt.Errorf("step required")
	}

	expr, eval, metricsPipeline, storageReq, err := e.Compile(req.Query)
	if err != nil {
		return nil, fmt.Errorf("compiling query: %w", err)
	}

	if metricsPipeline == nil {
		return nil, fmt.Errorf("not a metrics query")
	}

	if v, ok := expr.Hints.GetBool(HintDedupe, allowUnsafeQueryHints); ok {
		dedupeSpans = v
	}

	// This initializes all step buffers, counters, etc
	metricsPipeline.init(req, AggregateModeRaw)

	me := &MetricsEvalulator{
		storageReq:        storageReq,
		metricsPipeline:   metricsPipeline,
		dedupeSpans:       dedupeSpans,
		timeOverlapCutoff: timeOverlapCutoff,
	}

	// TraceID (optional)
	if req.ShardCount > 1 {
		// For sharding it must be in the first pass so that we only evalulate our traces.
		storageReq.ShardID = req.ShardID
		storageReq.ShardCount = req.ShardCount
		if !storageReq.HasAttribute(IntrinsicTraceIDAttribute) {
			storageReq.Conditions = append(storageReq.Conditions, Condition{Attribute: IntrinsicTraceIDAttribute})
		}
	}

	if dedupeSpans {
		// For dedupe we only need the trace ID on matching spans, so it can go in the second pass.
		// This is a no-op if we are already sharding and it's in the first pass.
		// Finally, this is often optimized back to the first pass when it lets us avoid a second pass altogether.
		if !storageReq.HasAttribute(IntrinsicTraceIDAttribute) {
			storageReq.SecondPassConditions = append(storageReq.SecondPassConditions, Condition{Attribute: IntrinsicTraceIDAttribute})
		}
	}

	// Span start time (always required)
	if !storageReq.HasAttribute(IntrinsicSpanStartTimeAttribute) {
		// Technically we only need the start time of matching spans, so we add it to the second pass.
		// However this is often optimized back to the first pass when it lets us avoid a second pass altogether.
		storageReq.SecondPassConditions = append(storageReq.SecondPassConditions, Condition{Attribute: IntrinsicSpanStartTimeAttribute})
	}

	// Timestamp filtering
	// (1) Include any overlapping trace
	//     It can be faster to skip the trace-level timestamp check
	//     when all or most of the traces overlap the window.
	//     So this is done dynamically on a per-fetcher basis in Do()
	// (2) Only include spans that started in this time frame.
	//     This is checked outside the fetch layer in the evaluator. Timestamp
	//     is only checked on the spans that are the final results.
	// TODO - I think there are cases where we can push this down.
	// Queries like {status=error} | rate() don't assert inter-span conditions
	// and we could filter on span start time without affecting correctness.
	// Queries where we can't are like:  {A} >> {B} | rate() because only require
	// that {B} occurs within our time range but {A} is allowed to occur any time.
	me.checkTime = true
	me.start = req.Start
	me.end = req.End

	// Setup second pass callback.  It might be optimized away
	storageReq.SecondPass = func(s *Spanset) ([]*Spanset, error) {
		// The traceql engine isn't thread-safe.
		// But parallelization is required for good metrics performance.
		// So we do external locking here.
		me.mtx.Lock()
		defer me.mtx.Unlock()
		return eval([]*Spanset{s})
	}

	optimize(storageReq)

	return me, nil
}

// optimize numerous things within the request that is specific to metrics.
func optimize(req *FetchSpansRequest) {
	if !req.AllConditions || req.SecondPassSelectAll {
		return
	}

	// There is an issue where multiple conditions &&'ed on the same
	// attribute can look like AllConditions==true, but are implemented
	// in the storage layer like ||'ed and require the second pass callback (engine).
	// TODO(mdisibio) - This would be a big performance improvement if we can fix the storage layer
	// Example:
	//   { span.http.status_code >= 500 && span.http.status_code < 600 } | rate() by (span.http.status_code)
	exists := make(map[Attribute]struct{}, len(req.Conditions))
	for _, c := range req.Conditions {
		if _, ok := exists[c.Attribute]; ok {
			// Don't optimize
			return
		}
		exists[c.Attribute] = struct{}{}
	}

	// Special optimization for queries like:
	//  {} | rate()
	//  {} | rate() by (rootName)
	//  {} | rate() by (resource.service.name)
	// When the second pass consists only of intrinsics, then it's possible to
	// move them to the first pass and increase performance. It avoids the second pass/bridge
	// layer and doesn't alter the correctness of the query.
	// This can't be done for plain attributes or in all cases.
	if len(req.SecondPassConditions) > 0 {
		secondLayerAlwaysPresent := true
		for _, cond := range req.SecondPassConditions {
			if cond.Attribute.Intrinsic != IntrinsicNone {
				continue
			}

			// This is a very special case. resource.service.name is also always present
			// (required by spec) so it can be moved too.
			if cond.Attribute.Scope == AttributeScopeResource && cond.Attribute.Name == "service.name" {
				continue
			}

			secondLayerAlwaysPresent = false
		}

		if secondLayerAlwaysPresent {
			// Move all to first pass
			req.Conditions = append(req.Conditions, req.SecondPassConditions...)
			req.SecondPass = nil
			req.SecondPassConditions = nil
		}
	}

	if len(req.SecondPassConditions) == 0 {
		req.SecondPass = nil
	}
}

func lookup(needles []Attribute, haystack Span) Static {
	for _, n := range needles {
		if v, ok := haystack.AttributeFor(n); ok {
			return v
		}
	}

	return Static{}
}

type MetricsEvalulator struct {
	start, end        uint64
	checkTime         bool
	dedupeSpans       bool
	deduper           *SpanDeduper2
	timeOverlapCutoff float64
	storageReq        *FetchSpansRequest
	metricsPipeline   metricsFirstStageElement
	spansTotal        uint64
	spansDeduped      uint64
	bytes             uint64
	mtx               sync.Mutex
}

func timeRangeOverlap(reqStart, reqEnd, dataStart, dataEnd uint64) float64 {
	st := max(reqStart, dataStart)
	end := min(reqEnd, dataEnd)

	if end <= st {
		return 0
	}

	return float64(end-st) / float64(dataEnd-dataStart)
}

// Do metrics on the given source of data and merge the results into the working set.  Optionally, if provided,
// uses the known time range of the data for last-minute optimizations. Time range is unix nanos
func (e *MetricsEvalulator) Do(ctx context.Context, f SpansetFetcher, fetcherStart, fetcherEnd uint64) error {
	// Make a copy of the request so we can modify it.
	storageReq := *e.storageReq

	if fetcherStart > 0 && fetcherEnd > 0 {
		// Dynamically decide whether to use the trace-level timestamp columns
		// for filtering.
		overlap := timeRangeOverlap(e.start, e.end, fetcherStart, fetcherEnd)

		if overlap == 0.0 {
			// This shouldn't happen but might as well check.
			// No overlap == nothing to do
			return nil
		}

		// Our heuristic is if the overlap between the given fetcher (i.e. block)
		// and the request is less than X%, use them.  Above X%, the cost of loading
		// them doesn't outweight the benefits. The default 20% was measured in
		// local benchmarking.
		if overlap < e.timeOverlapCutoff {
			storageReq.StartTimeUnixNanos = e.start
			storageReq.EndTimeUnixNanos = e.end // Should this be exclusive?
		}
	}

	fetch, err := f.Fetch(ctx, storageReq)
	if errors.Is(err, util.ErrUnsupported) {
		return nil
	}
	if err != nil {
		return err
	}

	if e.dedupeSpans && e.deduper == nil {
		e.deduper = NewSpanDeduper2()
	}

	defer fetch.Results.Close()

	for {
		ss, err := fetch.Results.Next(ctx)
		if err != nil {
			return err
		}
		if ss == nil {
			break
		}

		e.mtx.Lock()

		for _, s := range ss.Spans {
			if e.checkTime {
				st := s.StartTimeUnixNanos()
				if st < e.start || st >= e.end {
					continue
				}
			}

			if e.dedupeSpans && e.deduper.Skip(ss.TraceID, s.StartTimeUnixNanos()) {
				e.spansDeduped++
				continue
			}

			e.spansTotal++
			e.metricsPipeline.observe(s)

		}
		e.mtx.Unlock()
		ss.Release()
	}

	e.mtx.Lock()
	defer e.mtx.Unlock()
	e.bytes += fetch.Bytes()

	return nil
}

func (e *MetricsEvalulator) Metrics() (uint64, uint64, uint64) {
	e.mtx.Lock()
	defer e.mtx.Unlock()

	return e.bytes, e.spansTotal, e.spansDeduped
}

func (e *MetricsEvalulator) Results() SeriesSet {
	return e.metricsPipeline.result()
}

// SpanDeduper2 is EXTREMELY LAZY. It attempts to dedupe spans for metrics
// without requiring any new data fields.  It uses trace ID and span start time
// which are already loaded. This of course terrible, but did I mention that
// this is extremely lazy?  Additionally it uses sharded maps by the lowest byte
// of the trace ID to reduce the pressure on any single map.  Maybe it's good enough.  Let's find out!
type SpanDeduper2 struct {
	m       []map[uint32]struct{}
	h       hash.Hash32
	buf     []byte
	traceID Attribute
}

func NewSpanDeduper2() *SpanDeduper2 {
	maps := make([]map[uint32]struct{}, 256)
	for i := range maps {
		maps[i] = make(map[uint32]struct{}, 1000)
	}
	return &SpanDeduper2{
		m:       maps,
		h:       fnv.New32a(),
		buf:     make([]byte, 8),
		traceID: NewIntrinsic(IntrinsicTraceID),
	}
}

func (d *SpanDeduper2) Skip(tid []byte, startTime uint64) bool {
	d.h.Reset()
	d.h.Write(tid)
	binary.BigEndian.PutUint64(d.buf, startTime)
	d.h.Write(d.buf)

	v := d.h.Sum32()

	// Use last byte of the trace to choose the submap.
	// Empty ID uses submap 0.
	mapIdx := byte(0)
	if len(tid) > 0 {
		mapIdx = tid[len(tid)-1]
	}

	m := d.m[mapIdx]

	if _, ok := m[v]; ok {
		return true
	}

	m[v] = struct{}{}
	return false
}

// MetricsFrontendEvaluator pipes the sharded job results back into the engine for the rest
// of the pipeline.  i.e. This evaluator is for the query-frontend.
type MetricsFrontendEvaluator struct {
	metricsPipeline metricsFirstStageElement
}

func (m *MetricsFrontendEvaluator) ObserveSeries(in []*tempopb.TimeSeries) {
	m.metricsPipeline.observeSeries(in)
}

func (m *MetricsFrontendEvaluator) Results() SeriesSet {
	return m.metricsPipeline.result()
}

type SeriesAggregator interface {
	Combine([]*tempopb.TimeSeries)
	Results() SeriesSet
}

type SimpleAdditionAggregator struct {
	ss               SeriesSet
	len              int
	start, end, step uint64
}

func NewSimpleAdditionCombiner(req *tempopb.QueryRangeRequest) *SimpleAdditionAggregator {
	return &SimpleAdditionAggregator{
		ss:    make(SeriesSet),
		len:   IntervalCount(req.Start, req.End, req.Step),
		start: req.Start,
		end:   req.End,
		step:  req.Step,
	}
}

func (b *SimpleAdditionAggregator) Combine(in []*tempopb.TimeSeries) {
	for _, ts := range in {
		existing, ok := b.ss[ts.PromLabels]
		if !ok {
			// Convert proto labels to traceql labels
			labels := make(Labels, 0, len(ts.Labels))
			for _, l := range ts.Labels {
				labels = append(labels, Label{
					Name:  l.Key,
					Value: StaticFromAnyValue(l.Value),
				})
			}

			existing = TimeSeries{
				Labels: labels,
				Values: make([]float64, b.len),
			}
			b.ss[ts.PromLabels] = existing
		}

		for _, sample := range ts.Samples {
			j := IntervalOfMs(sample.TimestampMs, b.start, b.end, b.step)
			if j >= 0 && j < len(existing.Values) {
				existing.Values[j] += sample.Value
			}
		}
	}
}

func (b *SimpleAdditionAggregator) Results() SeriesSet {
	return b.ss
}

type HistogramBucket struct {
	Max   float64
	Count int
}

type Histogram struct {
	Buckets []HistogramBucket
}

func (h *Histogram) Record(bucket float64, count int) {
	for i := range h.Buckets {
		if h.Buckets[i].Max == bucket {
			h.Buckets[i].Count += count
			return
		}
	}

	h.Buckets = append(h.Buckets, HistogramBucket{
		Max:   bucket,
		Count: count,
	})
}

type histSeries struct {
	labels Labels
	hist   []Histogram
}

type HistogramAggregator struct {
	ss               map[string]histSeries
	qs               []float64
	len              int
	start, end, step uint64
}

func NewHistogramAggregator(req *tempopb.QueryRangeRequest, qs []float64) *HistogramAggregator {
	return &HistogramAggregator{
		qs:    qs,
		ss:    make(map[string]histSeries),
		len:   IntervalCount(req.Start, req.End, req.Step),
		start: req.Start,
		end:   req.End,
		step:  req.Step,
	}
}

func (h *HistogramAggregator) Combine(in []*tempopb.TimeSeries) {
	// var min, max time.Time

	for _, ts := range in {
		// Convert proto labels to traceql labels
		// while at the same time stripping the bucket label
		withoutBucket := make(Labels, 0, len(ts.Labels))
		var bucket Static
		for _, l := range ts.Labels {
			if l.Key == internalLabelBucket {
				// bucket = int(l.Value.GetIntValue())
				bucket = StaticFromAnyValue(l.Value)
				continue
			}
			withoutBucket = append(withoutBucket, Label{
				Name:  l.Key,
				Value: StaticFromAnyValue(l.Value),
			})
		}

		if bucket.Type == TypeNil {
			// Bad __bucket label?
			continue
		}

		withoutBucketStr := withoutBucket.String()

		existing, ok := h.ss[withoutBucketStr]
		if !ok {
			existing = histSeries{
				labels: withoutBucket,
				hist:   make([]Histogram, h.len),
			}
			h.ss[withoutBucketStr] = existing
		}

		b := bucket.asFloat()

		for _, sample := range ts.Samples {
			if sample.Value == 0 {
				continue
			}
			j := IntervalOfMs(sample.TimestampMs, h.start, h.end, h.step)
			if j >= 0 && j < len(existing.hist) {
				existing.hist[j].Record(b, int(sample.Value))
			}
		}
	}
}

func (h *HistogramAggregator) Results() SeriesSet {
	results := make(SeriesSet, len(h.ss)*len(h.qs))

	for _, in := range h.ss {
		// For each input series, we create a new series for each quantile.
		for _, q := range h.qs {
			// Append label for the quantile
			labels := append((Labels)(nil), in.labels...)
			labels = append(labels, Label{"p", NewStaticFloat(q)})
			s := labels.String()

			ts := TimeSeries{
				Labels: labels,
				Values: make([]float64, len(in.hist)),
			}
			for i := range in.hist {

				buckets := in.hist[i].Buckets
				sort.Slice(buckets, func(i, j int) bool {
					return buckets[i].Max < buckets[j].Max
				})

				ts.Values[i] = Log2Quantile(q, buckets)
			}
			results[s] = ts
		}
	}
	return results
}

// Log2Bucketize rounds the given value to the next powers-of-two bucket.
func Log2Bucketize(v uint64) float64 {
	if v < 2 {
		return -1
	}

	return math.Pow(2, math.Ceil(math.Log2(float64(v))))
}

// Log2Quantile returns the quantile given bucket labeled with float ranges and counts. Uses
// exponential power-of-two interpolation between buckets as needed.
func Log2Quantile(p float64, buckets []HistogramBucket) float64 {
	if math.IsNaN(p) ||
		p < 0 ||
		p > 1 ||
		len(buckets) == 0 {
		return 0
	}

	totalCount := 0
	for _, b := range buckets {
		totalCount += b.Count
	}

	if totalCount == 0 {
		return 0
	}

	// Maximum amount of samples to include. We round up to better handle
	// percentiles on low sample counts (<100).
	maxSamples := int(math.Ceil(p * float64(totalCount)))

	if maxSamples == 0 {
		// We have to read at least one sample.
		maxSamples = 1
	}

	// Find the bucket where the percentile falls in.
	var total, bucket int
	for i, b := range buckets {
		// Next bucket
		bucket = i

		// If we can't fully consume the samples in this bucket
		// then we are done.
		if total+b.Count > maxSamples {
			break
		}

		// Consume all samples in this bucket
		total += b.Count

		// p100 or happen to read the exact number of samples.
		// Quantile is the max range for the bucket. No reason
		// to enter interpolation below.
		if total == maxSamples {
			return b.Max
		}
	}

	// Fraction to interpolate between buckets, sample-count wise.
	// 0.5 means halfway
	interp := float64(maxSamples-total) / float64(buckets[bucket].Count)

	// Exponential interpolation between buckets
	// The current bucket represents the maximum value
	max := math.Log2(buckets[bucket].Max)
	var min float64
	if bucket > 0 {
		// Prior bucket represents the min
		min = math.Log2(buckets[bucket-1].Max)
	} else {
		// There is no prior bucket, assume powers of 2
		min = max - 1
	}
	mid := math.Pow(2, min+(max-min)*interp)
	return mid
}

var (
	_ SeriesAggregator = (*SimpleAdditionAggregator)(nil)
	_ SeriesAggregator = (*HistogramAggregator)(nil)
)
