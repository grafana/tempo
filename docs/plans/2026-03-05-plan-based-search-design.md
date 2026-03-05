# Plan-Based Search Traces Design

**Date:** 2026-03-05
**Status:** Draft
**Builds on:** `docs/plans/2026-03-04-plan-node-tree-design.md`

## Problem

The plan-based execution path currently handles only metrics queries (`queryBlock` in `querier_query_range.go`). The search traces path (`SearchBlock`) still uses the legacy `Engine.ExecuteSearch` + `SpansetFetcherWrapper` + `FetchSpansRequest.SecondPassFn` approach.

This design extends the plan-based architecture to search traces by solving two missing pieces:

1. **ProjectNode late materialization** — currently stubbed as `panic("not yet implemented")`. Search results need metadata columns (root service name, root span name, duration, service stats, etc.) that aren't fetched during the filter pass.
2. **SearchEvaluatable** — the existing `Evaluatable` interface returns `SeriesSet` (metrics). Search returns `*tempopb.SearchResponse` with trace metadata.

## Scope

- **Block-level search only** (`SearchBlock`). The recent/ingester path continues using existing gRPC fan-out, same as how the metrics plan path works today.
- Falls back to the legacy `ExecuteSearch` path on any error or when disabled via config.

## Design

### 1. ProjectNode as Late Materialization Operator

`ProjectNode` is **not** SQL's projection. It is a late materialization operator with two logical sides:

- **Driving side** (`child`): the filtered `SpansetIterator` producing spans with `rowNum` set
- **Fetch side** (`fetchTree`): a scan tree with `OpNone` conditions that reads metadata columns by seeking to the driving side's row numbers

```go
type ProjectNode struct {
    Columns   []Attribute  // declarative: what to late-materialize
    child     PlanNode     // driving side: filtered spans with rowNums
    fetchTree PlanNode     // fetch side: scan tree with OpNone conditions
}

func (n *ProjectNode) Children() []PlanNode {
    var ch []PlanNode
    if n.child != nil     { ch = append(ch, n.child) }
    if n.fetchTree != nil { ch = append(ch, n.fetchTree) }
    return ch
}
```

#### Building the fetch scan tree

A helper classifies metadata columns by parquet level and builds a minimal scan tree:

```go
func BuildFetchScanTree(columns []Attribute) *TraceScanNode {
    var traceConds, resConds, spanConds []Condition
    for _, col := range columns {
        cond := Condition{Attribute: col, Op: OpNone}
        switch levelOf(col) {
        case levelTrace:    traceConds = append(traceConds, cond)
        case levelResource: resConds = append(resConds, cond)
        case levelSpan:     spanConds = append(spanConds, cond)
        }
    }
    span := &SpanScanNode{Conditions: spanConds}
    instrScope := &InstrumentationScopeScanNode{child: span}
    resource := &ResourceScanNode{Conditions: resConds, child: instrScope}
    return &TraceScanNode{Conditions: traceConds, child: resource}
}
```

`levelOf()` uses the same intrinsic-to-scope mapping as `categorizeConditions` in vparquet5, lifted to `pkg/traceql`.

#### Translation to iterators

The translator builds both sides during translation — iterators are created **once**, not per spanset:

```go
func (t *translator) translateProjectToIter(node *ProjectNode) (SpansetIterator, error) {
    // 1. Build driving side — SpansetIterator (engine-level)
    drivingIter, err := t.translateToIter(node.child)
    if err != nil { return nil, err }

    // 2. Build fetch side — parquetquery.Iterator chain (storage-level)
    //    Needs SeekTo capability, so stays as pq.Iterator, not SpansetIterator
    fetchPqIter, err := t.buildFetchChain(node.fetchTree)
    if err != nil { return nil, err }

    // 3. Wrap in lateMaterializeIter
    return newLateMaterializeIter(drivingIter, fetchPqIter), nil
}
```

#### lateMaterializeIter

```go
type lateMaterializeIter struct {
    driving  traceql.SpansetIterator     // filtered spans from child plan
    fetcher  parquetquery.Iterator       // fetch scan tree iterators (built once)
}

func (it *lateMaterializeIter) Next(ctx context.Context) (*traceql.Spanset, error) {
    ss, err := it.driving.Next(ctx)
    if ss == nil || err != nil { return ss, err }

    // For each span, seek the fetch iterator to its rowNum.
    // Spans within a spanset are in row-number order.
    // Across spansets, row numbers increase (traces are sequential in parquet).
    // So SeekTo always moves forward — efficient sequential access.
    for _, span := range ss.Spans {
        res, err := it.fetcher.SeekTo(span.RowNum(), definitionLevel)
        if err != nil { return nil, err }
        if res != nil {
            mergeResultIntoSpan(span, res)
        }
    }
    return ss, nil
}

func (it *lateMaterializeIter) Close() {
    it.driving.Close()
    it.fetcher.Close()
}
```

**Key property:** No per-spanset iterator creation. The fetch-side parquet iterators are created once during translation and seeked forward as spans arrive. This mirrors how `SyncIterator.SeekTo` works — it skips row groups and pages efficiently.

**No ScanBackend interface change required.** The fetch scan tree is translated using the existing `buildInnerChain` / `buildParquetChain` methods.

### 2. SearchEvaluatable Interface

Separate from the metrics `Evaluatable` since they return fundamentally different result types:

```go
// tempodb/encoding/common/plan_translator.go

type SearchEvaluatable interface {
    Do(ctx context.Context) error
    Response() *tempopb.SearchResponse
}
```

#### searchEvaluatable implementation

```go
type searchEvaluatable struct {
    iter     traceql.SpansetIterator
    limit    int
    combiner traceql.MetadataCombiner
}

func (e *searchEvaluatable) Do(ctx context.Context) error {
    defer e.iter.Close()
    for {
        ss, err := e.iter.Next(ctx)
        if err != nil { return err }
        if ss == nil { return nil }

        e.combiner.AddSpanset(ss)
        ss.Release()

        if e.combiner.IsCompleteFor(traceql.TimestampNever) {
            return nil  // limit reached
        }
    }
}

func (e *searchEvaluatable) Response() *tempopb.SearchResponse {
    return &tempopb.SearchResponse{
        Traces:  e.combiner.Metadata(),
        Metrics: &tempopb.SearchMetrics{},
    }
}
```

### 3. TranslateSearch Entry Point

`TranslateSearch` is responsible for wrapping the plan in a `ProjectNode` with the metadata columns. The querier passes the columns but doesn't know about `ProjectNode`.

```go
func TranslateSearch(ctx context.Context, plan traceql.PlanNode,
    backend ScanBackend, opts SearchOptions,
    limit int, metaColumns []traceql.Attribute) (SearchEvaluatable, error) {

    // Build fetch scan tree and wrap plan in ProjectNode
    fetchTree := traceql.BuildFetchScanTree(metaColumns)
    plan = traceql.NewProjectNode(metaColumns, plan, fetchTree)

    t := &translator{ctx: ctx, backend: backend, opts: opts}
    iter, err := t.translateToIter(plan)
    if err != nil { return nil, err }

    return newSearchEvaluatable(iter, limit), nil
}
```

### 4. Querier Wiring

#### Config

```go
type SearchConfig struct {
    QueryTimeout            time.Duration `yaml:"query_timeout"`
    EnablePlanBasedExecution bool          `yaml:"enable_plan_based_execution,omitempty"`
}
// Default: false (opt-in during development)
```

#### SearchBlock integration

```go
func (q *Querier) SearchBlock(ctx context.Context, req *tempopb.SearchBlockRequest) (*tempopb.SearchResponse, error) {
    // ... existing: extract tenantID, blockID, meta, opts ...

    if api.IsTraceQLQuery(req.SearchReq) && q.cfg.Search.EnablePlanBasedExecution {
        if expr, err := traceql.Parse(req.SearchReq.Query); err == nil {
            if scanBackend, cleanup, sbErr := q.store.OpenScanBackend(ctx, meta, opts); sbErr == nil {
                defer cleanup()
                if plan, planErr := traceql.BuildPlan(expr, nil); planErr == nil {
                    metaCols := traceql.SearchMetaColumns()
                    searchEval, tErr := common.TranslateSearch(ctx, plan, scanBackend, opts,
                        int(req.SearchReq.Limit), metaCols)
                    if tErr == nil {
                        if doErr := searchEval.Do(ctx); doErr == nil {
                            return searchEval.Response(), nil
                        }
                    }
                }
            }
        }
    }

    // Legacy fallback
    if api.IsTraceQLQuery(req.SearchReq) {
        fetcher := traceql.NewSpansetFetcherWrapper(func(ctx context.Context,
            req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
            return q.store.Fetch(ctx, meta, req, opts)
        })
        return q.engine.ExecuteSearch(ctx, req.SearchReq, fetcher, q.limits.UnsafeQueryHints(tenantID))
    }
    return q.store.Search(ctx, meta, req.SearchReq, opts)
}
```

### 5. Search Metadata Columns

The metadata columns for search are the same as `SearchMetaConditions()` but expressed as `[]Attribute`:

```go
func SearchMetaColumns() []Attribute {
    return []Attribute{
        NewIntrinsic(IntrinsicTraceRootService),   // trace level
        NewIntrinsic(IntrinsicTraceRootSpan),       // trace level
        NewIntrinsic(IntrinsicTraceDuration),        // trace level
        NewIntrinsic(IntrinsicTraceID),              // trace level
        NewIntrinsic(IntrinsicTraceStartTime),       // trace level
        NewIntrinsic(IntrinsicSpanID),               // span level
        NewIntrinsic(IntrinsicSpanStartTime),        // span level
        NewIntrinsic(IntrinsicDuration),             // span level
        NewIntrinsic(IntrinsicServiceStats),         // resource level
    }
}
```

`BuildFetchScanTree` classifies these by level and places them as `OpNone` conditions on the appropriate scan nodes.

## Data Flow

```
SearchBlock request: { span.http.status_code >= 500 }
    |
    v
Parse → BuildPlan(expr, nil)
    |
    v
Plan tree (no metrics nodes):
  SpansetFilterNode(status_code >= 500)
    └─ TraceScanNode
        └─ ResourceScanNode
            └─ InstrumentationScopeScanNode
                └─ SpanScanNode(Conditions=[status_code >= 500])
    |
    v
TranslateSearch(plan, backend, opts, limit=20, metaCols)
    |
    ├─ Wraps plan in ProjectNode:
    |    ProjectNode
    |    ├── child: [above plan tree]
    |    └── fetchTree: TraceScanNode(RootService, RootSpan, Duration, ...)
    |                    └─ ResourceScanNode(ServiceStats)
    |                        └─ InstrumentationScopeScanNode
    |                            └─ SpanScanNode(SpanID, SpanStartTime, Duration)
    |
    ├─ Translates to iterators:
    |    lateMaterializeIter
    |    ├── driving: FilterIter → TraceScanIter → ResourceIter → SpanIter
    |    └── fetcher: pq.Iterator chain (trace→resource→span columns)
    |
    └─ Wraps in searchEvaluatable with MetadataCombiner(limit=20)
    |
    v
searchEvaluatable.Do():
    for each spanset from lateMaterializeIter:
        combiner.AddSpanset(ss)  →  asTraceSearchMetadata(ss)
        if limit reached: stop
    |
    v
SearchResponse with traces
```

## Optimizer Interaction

- **SecondPassEliminatorRule** can inspect `ProjectNode.Columns` against the driving child's scan conditions. If all metadata columns are already fetched by the first pass (e.g., the query filters on `rootServiceName`), the `ProjectNode` can be eliminated.
- **PredicatePushdownRule** operates on the driving child as before.
- The fetch scan tree is not subject to optimization — it's purely declarative (`OpNone` conditions only).

## File Changes

| File | Change |
|------|--------|
| `pkg/traceql/plan.go` | Update `ProjectNode` to have `fetchTree` child |
| `pkg/traceql/plan.go` | Add `BuildFetchScanTree()`, `SearchMetaColumns()`, `levelOf()` |
| `tempodb/encoding/common/plan_translator.go` | Add `SearchEvaluatable`, `searchEvaluatable`, `TranslateSearch()` |
| `tempodb/encoding/common/plan_translator.go` | Implement `translateProjectToIter()` with `lateMaterializeIter` |
| `modules/querier/config.go` | Add `EnablePlanBasedExecution` to `SearchConfig` |
| `modules/querier/querier.go` | Add plan-based path to `SearchBlock` |
