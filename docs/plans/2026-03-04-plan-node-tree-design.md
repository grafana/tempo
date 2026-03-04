# Plan Node Tree Design

**Date:** 2026-03-04
**Status:** Draft

## Problem

The TraceQL engine currently uses `FetchSpansRequest` as the interface between the query engine and storage. This struct carries conditions, predicates, and a `SecondPassFn` callback that bundles pipeline evaluation logic into the storage layer. This design has two main problems:

1. **No optimization layer.** There is no place to apply query rewrites (predicate pushdown, condition merging) between parsing and execution.
2. **AST dual responsibility.** AST nodes carry both structure (`SpansetFilter`, `GroupOperation`, etc.) and evaluation logic (`evaluate()` methods). The `SecondPassFn` is a closure over AST nodes, coupling structure and execution.

## Goals

- Convert the TraceQL AST into a logical plan node tree.
- Support user-extensible optimization passes (predicate pushdown, condition merging as primary targets).
- Replace `FetchSpansRequest` as the storage interface.
- Plan nodes are **purely logical** — no execution methods on nodes.
- Handle both spanset queries and metrics queries.

## Plan Node Taxonomy

### Scan nodes (storage layer)

Map 1:1 to parquet iterator levels:

```
TraceScanNode
ResourceScanNode
InstrumentationScopeScanNode
SpanScanNode
EventScanNode
LinkScanNode
```

Each carries the conditions (columns + predicates) needed at that parquet level.

### Engine nodes (in-memory evaluation)

```
SpansetFilterNode   — per-span or structural predicate filter
GroupByNode         — group spans into spansets by attribute
CoalesceNode        — merge spansets
StructuralOpNode    — parent-child (>>), ancestor (>), sibling (~)
SelectOperation     — (handled via ProjectNode placement, see below)
```

### Project node

```
ProjectNode         — triggers second-pass parquet fetch for additional columns
```

`ProjectNode` sits between the filter phase and the engine evaluation phase. It uses surviving span row numbers from its child to build a second-pass parquet scan for attributes not needed during filtering.

**Placement rule:** `ProjectNode` is always below `GroupByNode` and other engine nodes. This means second-pass column fetching happens before grouping, eliminating any need to reconstruct group membership after the fetch.

### Metrics nodes (typed)

```
RateNode
CountOverTimeNode
HistogramNode
QuantileNode
```

Each carries its aggregation configuration. Metrics nodes sit at the top of the plan tree above all engine and scan nodes.

## Plan Node Interface

```go
type PlanNode interface {
    Children() []PlanNode
    Accept(PlanVisitor)
    String() string
}

type PlanVisitor interface {
    VisitPre(PlanNode) bool   // return false to skip children
    VisitPost(PlanNode)
}
```

Nodes are purely logical — no `Execute()` or `Eval()` methods.

## Optimizer

Rule-based optimizer with fixpoint iteration.

```go
type Rule interface {
    Name() string
    Apply(PlanNode) (PlanNode, bool)  // returns (rewritten node, changed)
}

type RuleSet struct {
    rules []Rule
}

func (rs *RuleSet) Optimize(plan PlanNode) PlanNode {
    // fixpoint: keep applying rules until no change
}

// Register custom rules
func (rs *RuleSet) Add(r Rule)

// Helper for function-based rules
type FuncRule struct {
    name string
    fn   func(PlanNode) (PlanNode, bool)
}
```

Built-in rules:

- `PredicatePushdownRule` — push conditions from `SpansetFilterNode` into scan node predicates
- `ConditionMergeRule` — merge conditions across adjacent scan nodes of the same type
- `SecondPassEliminatorRule` — remove `ProjectNode` when all conditions can be served in the first pass

Users can register additional rules via `RuleSet.Add()`.

## Backend Translation

### ScanBackend interface

Exposes one method per scan node type. Lives in `tempodb/encoding/common`.

```go
type ScanBackend interface {
    TraceIter(ctx context.Context, node *TraceScanNode, primary Iterator, child Iterator) (Iterator, error)
    ResourceIter(ctx context.Context, node *ResourceScanNode, child Iterator) (Iterator, error)
    InstrumentationScopeIter(ctx context.Context, node *InstrumentationScopeScanNode, child Iterator) (Iterator, error)
    SpanIter(ctx context.Context, node *SpanScanNode, child Iterator) (Iterator, error)
    EventIter(ctx context.Context, node *EventScanNode, child Iterator) (Iterator, error)
    LinkIter(ctx context.Context, node *LinkScanNode, child Iterator) (Iterator, error)
}
```

vparquet5 implements `ScanBackend` by delegating to its existing `create*Iterator` functions.

### Translator

Lives in `tempodb/encoding/common/plan_translator.go`. Takes the full plan tree and a `ScanBackend`, produces a single `Evaluatable` that drives the whole pipeline end-to-end.

```go
func Translate(ctx context.Context, plan PlanNode, backend ScanBackend, opts SearchOptions) (Evaluatable, error)
```

The translator recurses bottom-up over the plan tree:

| Plan node | Translated to |
|---|---|
| `SpanScanNode` | `backend.SpanIter(node, child)` |
| `ResourceScanNode` | `backend.ResourceIter(node, child)` |
| `TraceScanNode` | `backend.TraceIter(node, primary, child)` |
| `SpansetFilterNode` | filter iterator wrapping child |
| `StructuralOpNode` | structural filter iterator wrapping child |
| `ProjectNode` | buffers child output, collects row numbers, builds second-pass scan via `backend.TraceIter(primary=childOutput, ...)` |
| `GroupByNode` | in-memory group evaluator |
| `RateNode` / `CountOverTimeNode` / ... | metrics aggregator |

**No bridge/rebatch.** Every operation is a first-class iterator or evaluator wrapping its child. There is no callback-based second pass. `ProjectNode` is the only node that performs a second parquet scan, using the surviving span row numbers from its child.

Example translation for `{ span.status = error } | group_by(resource.service.name) | select(span.region)`:

```
Plan tree:                          Iterator chain:
──────────────────────────────      ──────────────────────────────────────
GroupByNode(service.name)       →   GroupByEvaluator
  ProjectNode(region)           →     ProjectIter (second-pass TraceIter)
    SpansetFilterNode(status)   →       FilterIter
      ResourceScanNode(svc)     →         ResourceIter
        SpanScanNode(status)    →           SpanIter(predicate: status=error)
```

### Querier

```go
plan, err := engine.BuildPlan(expr, req)
plan = optimizer.Optimize(plan)

backend := q.store.ScanBackend(meta, opts)
eval, err := translate.Translate(ctx, plan, backend, opts)
err = eval.Do(ctx, start, end, maxSeries)
res := eval.Results()
```

## File Layout

```
pkg/traceql/
  plan.go                    PlanNode interface + all node types
  plan_builder.go            AST → plan tree
  plan_optimizer.go          Rule, RuleSet, fixpoint optimizer
  plan_rules.go              built-in optimization rules

tempodb/encoding/common/
  scan_backend.go            ScanBackend interface
  plan_translator.go         plan tree → Evaluatable via ScanBackend

tempodb/encoding/vparquet5/
  scan_backend.go            ScanBackend implementation
```

## Migration Path

**Phase 1 — infrastructure only, no behavior change**
Add `plan.go`, `plan_builder.go`, `plan_optimizer.go`, `plan_rules.go` in `pkg/traceql`. Nothing existing changes.

**Phase 2 — storage side, no behavior change**
Add `scan_backend.go` and `plan_translator.go` in `common`. Add `scan_backend.go` in vparquet5. Both the old `FetchSpansRequest` path and the new plan path exist; vparquet5 can serve either.

**Phase 3 — switch querier**
Update `queryBlock` in the querier to build a plan tree and use the translator. `FetchSpansRequest` remains for any other callers.

**Phase 4 — cleanup**
Remove `FetchSpansRequest` from `storage.go`. Remove `Pipeline.evaluate()` dual responsibility from AST nodes. Remove bridge/rebatch iterators if no longer used.
