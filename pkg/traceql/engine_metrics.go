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
	"github.com/grafana/tempo/pkg/util"
)

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

type TimeSeries struct {
	Labels labels.Labels
	Values []float64
}

// SeriesSet is a set of unique timeseries. They are mapped by the "Prometheus"-style
// text description: {x="a",y="b"}
type SeriesSet map[string]TimeSeries

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
	innerAgg  func() RangeAggregator

	// Data
	series map[FastValues]RangeAggregator
	buf    FastValues
}

var _ SpanAggregator = (*GroupingAggregator)(nil)

func NewGroupingAggregator(aggName string, innerAgg func() RangeAggregator, by []Attribute) SpanAggregator {
	if len(by) == 0 {
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

	return &GroupingAggregator{
		series:    map[FastValues]RangeAggregator{},
		by:        by,
		byLookups: lookups,
		innerAgg:  innerAgg,
	}
}

// Observe the span by looking up its group-by attributes, mapping to the series,
// and passing to the inner aggregate.  This is a critical hot path.
func (g *GroupingAggregator) Observe(span Span) {
	// Get grouping values
	// Reuse same buffer
	for i, lookups := range g.byLookups {
		g.buf[i] = lookup(lookups, span)
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
// (3) Exceptional case: All Nils. Real Prometheus-style metrics have a name, so there is
// always at least 1 label. Not so here. We have to force at least 1 label or else things
// may not be handled correctly downstream.  In this case we take the first label and
// make it the string "nil"
//
//	Ex: rate() by (x,y,z) and all nil yields:
//	{x="nil"}
func (g *GroupingAggregator) labelsFor(vals FastValues) labels.Labels {
	b := labels.NewBuilder(nil)

	present := false
	for i, v := range vals {
		if v.Type != TypeNil {
			b.Set(g.by[i].String(), v.EncodeToString(false))
			present = true
		}
	}

	if !present {
		b.Set(g.by[0].String(), "<nil>")
	}

	return b.Labels()
}

func (g *GroupingAggregator) Series() SeriesSet {
	ss := SeriesSet{}

	for vals, agg := range g.series {
		l := g.labelsFor(vals)

		ss[l.String()] = TimeSeries{
			Labels: l,
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
			Labels: l,
			Values: u.innerAgg.Samples(),
		},
	}
}

// ExecuteMetricsQueryRange - Execute the given metrics query. Just a wrapper around CompileMetricsQueryRange
func (e *Engine) ExecuteMetricsQueryRange(ctx context.Context, req *tempopb.QueryRangeRequest, fetcher SpansetFetcher) (results SeriesSet, err error) {
	eval, err := e.CompileMetricsQueryRange(req, false)
	if err != nil {
		return nil, err
	}

	err = eval.Do(ctx, fetcher)
	if err != nil {
		return nil, err
	}

	return eval.Results()
}

// CompileMetricsQueryRange returns an evalulator that can be reused across multiple data sources.
func (e *Engine) CompileMetricsQueryRange(req *tempopb.QueryRangeRequest, dedupeSpans bool) (*MetricsEvalulator, error) {
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

	eval, metricsPipeline, storageReq, err := e.Compile(req.Query)
	if err != nil {
		return nil, fmt.Errorf("compiling query: %w", err)
	}

	if metricsPipeline == nil {
		return nil, fmt.Errorf("not a metrics query")
	}

	// This initializes all step buffers, counters, etc
	metricsPipeline.init(req)

	me := &MetricsEvalulator{
		storageReq:      storageReq,
		metricsPipeline: metricsPipeline,
		dedupeSpans:     dedupeSpans,
	}

	if req.Of > 1 {
		// Trace id sharding
		// Select traceID if not already present.  It must be in the first pass
		// so that we only evalulate our traces.
		storageReq.Shard = int(req.Shard)
		storageReq.Of = int(req.Of)
		traceID := NewIntrinsic(IntrinsicTraceID)
		if !storageReq.HasAttribute(traceID) {
			storageReq.Conditions = append(storageReq.Conditions, Condition{Attribute: traceID})
		}
	}

	// Sharding algorithm
	// In order to scale out queries like {A} >> {B} | rate()  we need specific
	// rules about the data is divided across time boundary shards.  These
	// spans can cross hours or days and the simple idea to just check span
	// start time won't work.
	// Therefore results are matched with the following rules:
	// (1) Evalulate the query for any overlapping trace
	// (2) For any matching spans: only include the ones that started in this time frame.
	// This will increase redundant trace evalulation, but it ensures that matching spans are
	// guaranteed to be found and included in the metrcs.
	startTime := NewIntrinsic(IntrinsicSpanStartTime)
	if !storageReq.HasAttribute(startTime) {
		if storageReq.AllConditions {
			// The most efficient case.  We can add it to the primary pass
			// without affecting correctness. And this lets us avoid the
			// entire second pass.
			storageReq.Conditions = append(storageReq.Conditions, Condition{Attribute: startTime})
		} else {
			// Complex query with a second pass. In this case it is better to
			// add it to the second pass so that it's only returned for the matches.
			storageReq.SecondPassConditions = append(storageReq.SecondPassConditions, Condition{Attribute: startTime})
		}
	}

	// Special optimization for queries like {} | rate() by (rootName)
	// If first pass is only StartTime, then move any intrinsics to the first
	// pass and try to avoid a second pass.  It's ok and beneficial to move
	// intrinsics because they exist for every span and are never nil.
	// But we can't move attributes because that changes the handling of nils.
	// Moving attributes to the first pass is like asserting non-nil on them.
	// TODO

	if dedupeSpans {
		// We dedupe based on trace ID and start time.  Obviously this doesn't
		// work if 2 spans have the same start time, but this doesn't impose any
		// more data on the query when sharding is already present above (most cases)
		traceID := NewIntrinsic(IntrinsicTraceID)
		if !storageReq.HasAttribute(traceID) {
			storageReq.Conditions = append(storageReq.Conditions, Condition{Attribute: traceID})
		}
	}

	// (1) any overlapping trace
	// TODO - Make this dynamic since it can be faster to skip
	// the trace-level timestamp check when all or most of the traces
	// overlap the window.
	// storageReq.StartTimeUnixNanos = req.Start
	// storageReq.EndTimeUnixNanos = req.End // Should this be exclusive?
	// (2) Only include spans that started in this time frame.
	//     This is checked inside the evaluator
	me.checkTime = true
	me.start = req.Start
	me.end = req.End

	// Avoid a second pass when not needed for much better performance.
	// TODO - Is there any case where eval() needs to be called but AllConditions=true?
	if !storageReq.AllConditions || len(storageReq.SecondPassConditions) > 0 {
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

func lookup(needles []Attribute, haystack Span) Static {
	for _, n := range needles {
		if v, ok := haystack.AttributeFor(n); ok {
			return v
		}
	}

	return Static{}
}

type MetricsEvalulator struct {
	start, end      uint64
	checkTime       bool
	dedupeSpans     bool
	deduper         *SpanDeduper2
	storageReq      *FetchSpansRequest
	metricsPipeline metricsFirstStageElement
	count           int
	deduped         int
	mtx             sync.Mutex
}

func (e *MetricsEvalulator) Do(ctx context.Context, f SpansetFetcher) error {
	fetch, err := f.Fetch(ctx, *e.storageReq)
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
				e.deduped++
				continue
			}

			e.count++
			e.metricsPipeline.observe(s)

		}
		e.mtx.Unlock()
		ss.Release()
	}

	return nil
}

func (e *MetricsEvalulator) SpanCount() {
	fmt.Println(e.count, e.deduped)
}

func (e *MetricsEvalulator) Results() (SeriesSet, error) {
	return e.metricsPipeline.result(), nil
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
