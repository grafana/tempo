package common

import (
	"context"
	"fmt"

	"github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
)

// Evaluatable is the result of translating a full plan tree.
// Call Do to drive execution; Results returns the accumulated series.
type Evaluatable interface {
	Do(ctx context.Context, start, end uint64, maxSeries int) error
	Results() traceql.SeriesSet
	Metrics() (inspectedBytes, spansTotal uint64, err error)
}

// Translate converts a full plan tree into an Evaluatable by recursively
// building the iterator/evaluator chain. The translator owns the boundary
// between storage iterators (parquetquery.Iterator) and engine evaluators
// (in-memory GroupBy, metrics aggregation, etc.).
func Translate(ctx context.Context, plan traceql.PlanNode, backend ScanBackend, opts SearchOptions) (Evaluatable, error) {
	t := &translator{ctx: ctx, backend: backend, opts: opts}
	return t.translate(plan)
}

type translator struct {
	ctx     context.Context
	backend ScanBackend
	opts    SearchOptions
}

func (t *translator) translate(n traceql.PlanNode) (Evaluatable, error) {
	switch node := n.(type) {

	// --- Metrics nodes: build aggregator wrapping a SpansetIterator ---
	case *traceql.RateNode:
		iter, err := t.translateToIter(node.Children()[0])
		if err != nil {
			return nil, err
		}
		return newRateEvaluatable(node, iter), nil

	case *traceql.CountOverTimeNode:
		iter, err := t.translateToIter(node.Children()[0])
		if err != nil {
			return nil, err
		}
		return newCountOverTimeEvaluatable(node, iter), nil

	// --- Engine nodes: build SpansetIterator chain, wrap as Evaluatable ---
	case *traceql.GroupByNode,
		*traceql.CoalesceNode,
		*traceql.SpansetFilterNode,
		*traceql.SpansetRelationNode:
		iter, err := t.translateToIter(n)
		if err != nil {
			return nil, err
		}
		return newSpansetEvaluatable(iter), nil

	// --- ProjectNode: second-pass fetch ---
	case *traceql.ProjectNode:
		return t.translateProject(node)

	// --- Scan tree: build parquet iterator chain bottom-up ---
	case *traceql.TraceScanNode:
		spansetIter, err := t.buildParquetChain(node, nil)
		if err != nil {
			return nil, err
		}
		return newSpansetEvaluatable(spansetIter), nil

	default:
		return nil, fmt.Errorf("plan_translator: unhandled plan node type %T", n)
	}
}

// translateToIter converts any pipeline or scan node into a SpansetIterator.
// Metrics nodes (RateNode, CountOverTimeNode) are not valid here and return an error.
func (t *translator) translateToIter(n traceql.PlanNode) (traceql.SpansetIterator, error) {
	switch node := n.(type) {

	case *traceql.TraceScanNode:
		return t.buildParquetChain(node, nil)

	case *traceql.SpansetFilterNode:
		child, err := t.translateToIter(node.Children()[0])
		if err != nil {
			return nil, err
		}
		return traceql.FilterSpansetIter(node.Expression, child), nil

	case *traceql.GroupByNode:
		child, err := t.translateToIter(node.Children()[0])
		if err != nil {
			return nil, err
		}
		return traceql.GroupBySpansetIter(node.By, child), nil

	case *traceql.CoalesceNode:
		child, err := t.translateToIter(node.Children()[0])
		if err != nil {
			return nil, err
		}
		return traceql.CoalesceSpansetIter(child), nil

	case *traceql.SpansetRelationNode:
		child, err := t.translateToIter(node.Children()[0])
		if err != nil {
			return nil, err
		}
		return traceql.RelationSpansetIter(node.Expr, child), nil

	case *traceql.ProjectNode:
		return t.translateProjectToIter(node)

	default:
		return nil, fmt.Errorf("plan_translator: cannot build iterator for node type %T", n)
	}
}

// buildParquetChain recursively builds a parquetquery.Iterator chain from the
// scan node subtree and calls backend.TraceIter to produce a SpansetIterator.
// primary is the second-pass row source (nil for first pass).
func (t *translator) buildParquetChain(trace *traceql.TraceScanNode, primary parquetquery.Iterator) (traceql.SpansetIterator, error) {
	var child parquetquery.Iterator
	if len(trace.Children()) > 0 {
		var err error
		child, err = t.buildInnerChain(trace.Children()[0])
		if err != nil {
			return nil, err
		}
	}
	return t.backend.TraceIter(t.ctx, trace, primary, child)
}

func (t *translator) buildInnerChain(n traceql.PlanNode) (parquetquery.Iterator, error) {
	switch node := n.(type) {
	case *traceql.ResourceScanNode:
		var child parquetquery.Iterator
		if len(node.Children()) > 0 {
			var err error
			child, err = t.buildInnerChain(node.Children()[0])
			if err != nil {
				return nil, err
			}
		}
		return t.backend.ResourceIter(t.ctx, node, child)

	case *traceql.InstrumentationScopeScanNode:
		var child parquetquery.Iterator
		if len(node.Children()) > 0 {
			var err error
			child, err = t.buildInnerChain(node.Children()[0])
			if err != nil {
				return nil, err
			}
		}
		return t.backend.InstrumentationScopeIter(t.ctx, node, child)

	case *traceql.SpanScanNode:
		var child parquetquery.Iterator
		if len(node.Children()) > 0 {
			var err error
			child, err = t.buildInnerChain(node.Children()[0])
			if err != nil {
				return nil, err
			}
		}
		return t.backend.SpanIter(t.ctx, node, child)

	case *traceql.EventScanNode:
		return t.backend.EventIter(t.ctx, node, nil)

	case *traceql.LinkScanNode:
		return t.backend.LinkIter(t.ctx, node, nil)

	default:
		return nil, fmt.Errorf("plan_translator: unexpected inner node %T", n)
	}
}

// translateProject builds an Evaluatable from ProjectNode using the iterator path.
func (t *translator) translateProject(node *traceql.ProjectNode) (Evaluatable, error) {
	iter, err := t.translateProjectToIter(node)
	if err != nil {
		return nil, err
	}
	return newSpansetEvaluatable(iter), nil
}

// translateProjectToIter builds a lateMaterializeIter that wraps:
// - driving side: SpansetIterator from child plan
// - fetch side: parquetquery.Iterator chain from fetchTree
func (t *translator) translateProjectToIter(node *traceql.ProjectNode) (traceql.SpansetIterator, error) {
	drivingIter, err := t.translateToIter(node.Children()[0])
	if err != nil {
		return nil, err
	}

	fetchTree := node.FetchTree()
	if fetchTree == nil {
		return drivingIter, nil
	}

	fetchTraceScan, ok := fetchTree.(*traceql.TraceScanNode)
	if !ok {
		return nil, fmt.Errorf("plan_translator: ProjectNode fetchTree must be *TraceScanNode, got %T", fetchTree)
	}

	// Build fetch-side pq.Iterator chain
	var fetchChild parquetquery.Iterator
	if len(fetchTraceScan.Children()) > 0 {
		fetchChild, err = t.buildInnerChain(fetchTraceScan.Children()[0])
		if err != nil {
			return nil, err
		}
	}

	fetchPqIter, err := t.backend.TraceIterRaw(t.ctx, fetchTraceScan, nil, fetchChild)
	if err != nil {
		return nil, err
	}

	// DefinitionLevel 0 = trace level: the fetch iterator operates per-trace,
	// producing a full spanset with metadata in OtherEntries["spanset"].
	return newLateMaterializeIter(drivingIter, fetchPqIter, 0, t.backend.SpanMerger()), nil
}

// ---------------------------------------------------------------------------
// rateEvaluatable
// ---------------------------------------------------------------------------

type rateEvaluatable struct {
	node       *traceql.RateNode
	iter       traceql.SpansetIterator
	agg        *traceql.MetricsAggregate
	spansTotal uint64
}

func newRateEvaluatable(node *traceql.RateNode, iter traceql.SpansetIterator) Evaluatable {
	return &rateEvaluatable{
		node: node,
		iter: iter,
		agg:  traceql.NewMetricsAggregate(traceql.MetricsAggregateRate, node.By),
	}
}

func (e *rateEvaluatable) Do(ctx context.Context, _, _ uint64, maxSeries int) error {
	defer e.iter.Close()
	req := &tempopb.QueryRangeRequest{
		Start:     e.node.Start,
		End:       e.node.End,
		Step:      e.node.Step,
		Exemplars: e.node.Exemplars,
	}
	e.agg.Init(req, traceql.AggregateModeRaw)
	for {
		ss, err := e.iter.Next(ctx)
		if err != nil {
			return err
		}
		if ss == nil {
			return nil
		}
		for _, span := range ss.Spans {
			e.agg.Observe(span)
			e.spansTotal++
		}
		ss.Release()
		if maxSeries > 0 && e.agg.Length() >= maxSeries {
			return nil
		}
	}
}

func (e *rateEvaluatable) Results() traceql.SeriesSet {
	return e.agg.Result(1.0)
}

func (e *rateEvaluatable) Metrics() (uint64, uint64, error) {
	return 0, e.spansTotal, nil
}

// ---------------------------------------------------------------------------
// countOverTimeEvaluatable
// ---------------------------------------------------------------------------

type countOverTimeEvaluatable struct {
	node       *traceql.CountOverTimeNode
	iter       traceql.SpansetIterator
	agg        *traceql.MetricsAggregate
	spansTotal uint64
}

func newCountOverTimeEvaluatable(node *traceql.CountOverTimeNode, iter traceql.SpansetIterator) Evaluatable {
	return &countOverTimeEvaluatable{
		node: node,
		iter: iter,
		agg:  traceql.NewMetricsAggregate(traceql.MetricsAggregateCountOverTime, node.By),
	}
}

func (e *countOverTimeEvaluatable) Do(ctx context.Context, _, _ uint64, maxSeries int) error {
	defer e.iter.Close()
	req := &tempopb.QueryRangeRequest{
		Start:     e.node.Start,
		End:       e.node.End,
		Step:      e.node.Step,
		Exemplars: e.node.Exemplars,
	}
	e.agg.Init(req, traceql.AggregateModeRaw)
	for {
		ss, err := e.iter.Next(ctx)
		if err != nil {
			return err
		}
		if ss == nil {
			return nil
		}
		for _, span := range ss.Spans {
			e.agg.Observe(span)
			e.spansTotal++
		}
		ss.Release()
		if maxSeries > 0 && e.agg.Length() >= maxSeries {
			return nil
		}
	}
}

func (e *countOverTimeEvaluatable) Results() traceql.SeriesSet {
	return e.agg.Result(1.0)
}

func (e *countOverTimeEvaluatable) Metrics() (uint64, uint64, error) {
	return 0, e.spansTotal, nil
}

// ---------------------------------------------------------------------------
// spansetEvaluatable — leaf evaluatable for pure scan-tree plans
// ---------------------------------------------------------------------------

type spansetEvaluatable struct {
	iter traceql.SpansetIterator
}

func newSpansetEvaluatable(iter traceql.SpansetIterator) Evaluatable {
	return &spansetEvaluatable{iter: iter}
}

func (e *spansetEvaluatable) Do(ctx context.Context, _, _ uint64, _ int) error {
	// Drain the iterator.
	defer e.iter.Close()
	for {
		ss, err := e.iter.Next(ctx)
		if err != nil {
			return err
		}
		if ss == nil {
			return nil
		}
		ss.Release()
	}
}

func (e *spansetEvaluatable) Results() traceql.SeriesSet { return nil }

func (e *spansetEvaluatable) Metrics() (uint64, uint64, error) { return 0, 0, nil }

// ---------------------------------------------------------------------------
// SearchEvaluatable — search-specific evaluatable
// ---------------------------------------------------------------------------

// SearchEvaluatable is the result of translating a plan tree for search.
// Call Do to drive execution; Response returns the accumulated search results.
type SearchEvaluatable interface {
	Do(ctx context.Context) error
	Response() *tempopb.SearchResponse
}

// searchEvaluatable iterates spansets from the plan, converts them to
// trace metadata, and collects results up to the limit.
type searchEvaluatable struct {
	iter     traceql.SpansetIterator
	limit    int
	combiner traceql.MetadataCombiner
}

func newSearchEvaluatable(iter traceql.SpansetIterator, limit int) SearchEvaluatable {
	return &searchEvaluatable{
		iter:     iter,
		limit:    limit,
		combiner: traceql.NewMetadataCombiner(limit, false),
	}
}

func (e *searchEvaluatable) Do(ctx context.Context) error {
	defer e.iter.Close()
	for {
		ss, err := e.iter.Next(ctx)
		if err != nil {
			return err
		}
		if ss == nil {
			return nil
		}

		e.combiner.AddSpanset(ss)

		if e.combiner.IsCompleteFor(traceql.TimestampNever) {
			return nil
		}
	}
}

func (e *searchEvaluatable) Response() *tempopb.SearchResponse {
	return &tempopb.SearchResponse{
		Traces:  e.combiner.Metadata(),
		Metrics: &tempopb.SearchMetrics{},
	}
}

// TranslateSearch converts a plan tree into a SearchEvaluatable.
// The plan must already contain a ProjectNode at the root (built by
// BuildSearchTracePlan) that carries the fetch-side scan tree for late
// materialization of search metadata columns.
func TranslateSearch(
	ctx context.Context,
	plan traceql.PlanNode,
	backend ScanBackend,
	opts SearchOptions,
	limit int,
) (SearchEvaluatable, error) {
	t := &translator{ctx: ctx, backend: backend, opts: opts}
	iter, err := t.translateToIter(plan)
	if err != nil {
		return nil, err
	}

	return newSearchEvaluatable(iter, limit), nil
}

