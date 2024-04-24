package traceql

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"hash/fnv"
	"sync"
	"time"

	"github.com/prometheus/prometheus/model/labels"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1proto "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/grafana/tempo/pkg/util"
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

type Label struct {
	Name  string
	Value Static
}

type TimeSeries struct {
	Labels []Label
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

func SeriesSetFromProto(req *tempopb.QueryRangeRequest, in []*tempopb.TimeSeries) SeriesSet {
	ss := make(SeriesSet)

	for _, i := range in {
		dest := TimeSeries{}

		dest.Labels = make([]Label, 0, len(i.Labels))
		for _, l := range i.Labels {
			dest.Labels = append(dest.Labels, Label{
				Name:  l.Key,
				Value: StaticFromAnyValue(l.Value),
			})
		}

		dest.Values = make([]float64, IntervalCount(req.Start, req.End, req.Step))
		for _, sample := range i.Samples {
			ts := uint64(time.Duration(sample.TimestampMs) * time.Millisecond)
			j := IntervalOf(ts, req.Start, req.End, req.Step)
			// TODO(mdisibio) sometimes interval is out of range
			// there must be a job alignment issue
			if j >= 0 && j < len(dest.Values) {
				dest.Values[j] = sample.Value
			}
		}

		ss[i.PromLabels] = dest
	}

	return ss
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
// collisions).  However it means we have to arbitrarily set an upper limit on
// the maximum number of values.
type FastValues [maxGroupBys]Static

// GroupingAggregator groups spans into series based on attribute values.
type GroupingAggregator struct {
	// Config
	by        []Attribute   // Original attributes: .foo
	byLookups [][]Attribute // Lookups: span.foo resource.foo
	byFunc    func(Span) (Static, bool)
	innerAgg  func() RangeAggregator

	// Data
	series map[FastValues]RangeAggregator
	buf    FastValues
}

var _ SpanAggregator = (*GroupingAggregator)(nil)

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

	// Add the dynamic by-label to the list of labels
	// It's picked up in LabelsFor
	if byFunc != nil {
		by = append(by, NewAttribute(byFuncLabel))
	}

	return &GroupingAggregator{
		series:    map[FastValues]RangeAggregator{},
		by:        by,
		byFunc:    byFunc,
		byLookups: lookups,
		innerAgg:  innerAgg,
	}
}

// Observe the span by looking up its group-by attributes, mapping to the series,
// and passing to the inner aggregate.  This is a critical hot path.
func (g *GroupingAggregator) Observe(span Span) {
	// Get grouping values
	// Reuse same buffer
	// There is no need to reset, the number of group-by attributes
	// is fixed after creation.
	for i, lookups := range g.byLookups {
		g.buf[i] = lookup(lookups, span)
	}
	if g.byFunc != nil {
		v, ok := g.byFunc(span)
		if !ok {
			// Totally drop this span
			return
		}
		g.buf[len(g.byLookups)] = v
	}

	agg, ok := g.series[g.buf]
	if !ok {
		agg = g.innerAgg()
		g.series[g.buf] = agg
	}

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
func (g *GroupingAggregator) labelsFor(vals FastValues) ([]Label, string) {
	tempoLabels := make([]Label, 0, len(g.by)+1)
	for i, v := range vals {
		if v.Type == TypeNil {
			continue
		}
		tempoLabels = append(tempoLabels, Label{g.by[i].String(), v})
	}

	// Prometheus-style version for convenience
	promLabels := labels.NewBuilder(nil)
	for _, l := range tempoLabels {
		promValue := l.Value.EncodeToString(false)
		if promValue == "" {
			promValue = "<empty>"
		}
		promLabels.Set(l.Name, promValue)
	}
	// When all nil then force one.
	if promLabels.Labels().IsEmpty() {
		promLabels.Set(g.by[0].String(), "<nil>")
	}

	return tempoLabels, promLabels.Labels().String()
}

func (g *GroupingAggregator) Series() SeriesSet {
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

func (e *Engine) CompileMetricsQueryRangeFrontend(req *tempopb.QueryRangeRequest) (*MetricsFrontendEvaluator, error) {
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

	metricsPipeline.init(req, false)

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
	metricsPipeline.init(req, true)

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

	optimize(storageReq)

	// Setup second pass callback.  Only as needed.
	if len(storageReq.SecondPassConditions) > 0 {
		storageReq.SecondPass = func(s *Spanset) ([]*Spanset, error) {
			// The traceql engine isn't thread-safe.
			// But parallelization is required for good metrics performance.
			// So we do external locking here.
			me.mtx.Lock()
			defer me.mtx.Unlock()
			return eval([]*Spanset{s})
		}
	}

	return me, nil
}

// optimize numerous things within the request that is specific to metrics.
func optimize(req *FetchSpansRequest) {
	// Special optimization for queries like:
	//  {} | rate()
	//  {} | rate() by (rootName)
	//  {} | rate() by (resource.service.name)
	// When the second pass consists only of intrinsics, then it's possible to
	// move them to the first pass and increase performance. It avoids the second pass/bridge
	// layer and doesn't alter the correctness of the query.
	// This can't be done for plain attributes or in all cases.
	if req.AllConditions && len(req.SecondPassConditions) > 0 {
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
			req.SecondPassConditions = nil
		}
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
	m := d.m[tid[len(tid)-1]]

	if _, ok := m[v]; ok {
		return true
	}

	m[v] = struct{}{}
	return false
}

type MetricsFrontendEvaluator struct {
	metricsPipeline metricsFirstStageElement
}

func (m *MetricsFrontendEvaluator) ObserveJob(in SeriesSet) {
	m.metricsPipeline.observeSeries(in)
}

func (m *MetricsFrontendEvaluator) Results() SeriesSet {
	return m.metricsPipeline.result()
}
