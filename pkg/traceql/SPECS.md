# traceql — Interface and Behaviour Specification (Metrics Math)

This document is the authoritative specification for the TraceQL metrics math
subsystem. It is detailed enough to re-implement the feature from scratch. It
complements NOTES.md (design rationale) and TESTS.md (test plan).

---

## 1. Responsibility Boundary

The `traceql` package **parses, validates, and evaluates TraceQL queries including
binary arithmetic (+, -, *, /) on aggregated trace metrics**.

| Concern | Owner |
|---------|-------|
| AST for math expressions (`mathExpression`) | **traceql** |
| Grammar rules for `metricsExpression` | **traceql** |
| Binary arithmetic on `SeriesSet` | **traceql** |
| Second-stage pipeline (topk, bottomk, filter) | **traceql** |
| First-stage span aggregation (rate, count_over_time, …) | **traceql** |
| Fragment label routing (`batchSeriesProcessor`) | **traceql** |
| Protocol-buffer serialization | `tempopb` |
| Storage layer | `tempodb` |
| Distributed shard orchestration | `modules/frontend` |

---

## 2. Internal Constants

```go
const internalLabelQueryFragment = "__query_fragment"  // engine_metrics.go:27
```

This label is the sole mechanism by which sub-query results are tagged and routed.
No code outside `pkg/traceql` may set or depend on it.

---

## 3. Core Types

### 3.1 `SeriesSet`

```go
type SeriesSet map[SeriesMapKey]TimeSeries
```

A map from a fixed-width label key to a time series. Used as the unit of data
throughout the second stage.

```go
type TimeSeries struct {
    Labels    Labels
    Values    []float64
    Exemplars []Exemplar
}
```

### 3.2 `secondStageElement` Interface

```go
// ast_metrics.go:356-360
type secondStageElement interface {
    Element                                           // String() string; validate() error
    init(req *tempopb.QueryRangeRequest)
    process(input SeriesSet) SeriesSet
}
```

**Contract:**
- `init` is called exactly once before the first `process` call; it receives the
  full `QueryRangeRequest` (start, end, step, exemplar limit).
- `process` must not mutate `input`. It returns a new `SeriesSet`.

Implementations in this package: `mathExpression`, `TopKBottomK`, `MetricsFilter`,
`ChainedSecondStage`.

### 3.3 `mathExpression` Struct

```go
// ast_metrics_math.go:22-28
type mathExpression struct {
    op     Operator           // OpAdd/OpSub/OpMult/OpDiv for binary; OpNone for leaf
    key    string             // leaf only: __query_fragment value to match
    lhs    *mathExpression    // binary only: left child
    rhs    *mathExpression    // binary only: right child
    filter secondStageElement // leaf only: optional per-leaf second stage
}
```

`mathExpression` implements `secondStageElement`. The same struct represents both
leaf and binary nodes — the `op` field distinguishes them.

### 3.4 `ChainedSecondStage`

```go
// ast_metrics.go:521
type ChainedSecondStage struct {
    elements   []secondStageElement
    separators []string
}
func (c *ChainedSecondStage) Append(element secondStageElement, separator string)
```

Each element's output becomes the next element's input. All `secondStageElement`
methods delegate sequentially. Separators are stored alongside elements via
`Append` rather than being part of the `secondStageElement` interface.

---

## 4. Grammar

The parser (`expr.y`) produces math ASTs through these rules:

```yacc
// Top-level root can be a metricsExpression (with optional second-stage pipeline).
root:
  | metricsExpression                          { unwrapSingleMathExpr($1) }
  | metricsExpression metricsSecondStagePipeline { chainMathSecondStage($1, $2) }
  ;

// metricsExpression is a recursive binary tree of wrapped pipelines.
metricsExpression:
    OPEN_PARENS metricsExpression CLOSE_PARENS  { $$ = $2 }
  | metricsExpression ADD metricsExpression      { $$ = newRootExprMath(OpAdd, $1, $3) }
  | metricsExpression SUB metricsExpression      { $$ = newRootExprMath(OpSub, $1, $3) }
  | metricsExpression MUL metricsExpression      { $$ = newRootExprMath(OpMult, $1, $3) }
  | metricsExpression DIV metricsExpression      { $$ = newRootExprMath(OpDiv, $1, $3) }
  | wrappedMetricsPipeline                       { $$ = $1 }
  ;

// wrappedMetricsPipeline is a leaf: a parenthesized spanset pipeline + aggregation.
wrappedMetricsPipeline:
    OPEN_PARENS spansetPipeline PIPE metricsAggregation CLOSE_PARENS
      { $$ = newWrappedMetricsPipeline($2, $4, nil) }
  | OPEN_PARENS spansetPipeline PIPE metricsAggregation metricsSecondStagePipeline CLOSE_PARENS
      { $$ = newWrappedMetricsPipeline($2, $4, $5) }
  ;

// metricsSecondStagePipeline chains topk/bottomk/filter ops after math.
metricsSecondStagePipeline:
    PIPE metricsSecondStage                              { $$ = ChainedSecondStage{$2} }
  | metricsFilter                                        { $$ = ChainedSecondStage{$1} }
  | metricsSecondStagePipeline PIPE metricsSecondStage   { $$ = append($1, $3) }
  | metricsSecondStagePipeline metricsFilter             { $$ = append($1, $2) }
  ;

metricsSecondStage:
    TOPK OPEN_PARENS INTEGER CLOSE_PARENS    { $$ = newTopKBottomK(OpTopK, $3, " | ") }
  | BOTTOMK OPEN_PARENS INTEGER CLOSE_PARENS { $$ = newTopKBottomK(OpBottomK, $3, " | ") }
  ;
```

**Parse constraint:** Each leaf pipeline MUST be parenthesized:
```
// Valid:   ({status=error} | rate()) / ({} | rate())
// Invalid: {status=error} | rate() / {} | rate()   ← parse error
```

---

## 5. AST Construction Functions

### 5.1 `newWrappedMetricsPipeline`

```go
// ast.go:223-239
func newWrappedMetricsPipeline(e PipelineElement, m1 firstStageElement, m2 secondStageElement) *RootExpr
```

Creates a leaf `RootExpr` for a single parenthesized pipeline. Called by the
`wrappedMetricsPipeline` grammar rule.

**Algorithm:**
1. If `e` is not a `Pipeline`, wrap it: `p = newPipeline(e)`.
2. Compute fragment key: `key = p.String() + " | " + m1.String()`.
   If `m2 != nil`, append `m2.String()` to the key.
3. Return `&RootExpr{Pipeline: {key: p}, BatchSpanProcessor: {key: m1}, SeriesProcessor: m1, MetricsSecondStage: &mathExpression{op: OpNone, key: key, filter: m2}}`.

**Key invariant:** The key uniquely identifies the sub-query. Identical sub-queries
in a larger expression map to the same key, deduplicating them automatically.

---

### 5.2 `newRootExprMath`

```go
// ast.go:157-205
func newRootExprMath(op Operator, lhs, rhs *RootExpr) *RootExpr
```

Creates a binary math `RootExpr` combining two sub-expressions. Called by the
binary arithmetic grammar rules (`ADD`, `SUB`, `MUL`, `DIV`).

**Algorithm:**
1. Allocate merged maps: `pipelines`, `spanProcs`, `seriesProcs` (capacity = sum of both sides).
2. Copy all entries from `lhs.Pipeline` and `rhs.Pipeline` into `pipelines`.
3. Copy all entries from `lhs.BatchSpanProcessor` and `rhs.BatchSpanProcessor` into `spanProcs`.
4. Copy all per-fragment series processors from both sides into a `batchSeriesProcessor` (`seriesProcs`).
   - If a side's `SeriesProcessor` is already a `batchSeriesProcessor`, range over its entries.
   - Otherwise add the single processor keyed by the one key in `BatchSpanProcessor`.
5. Extract `lhsMath = asMathExpression(lhs.MetricsSecondStage)`.
6. Extract `rhsMath = asMathExpression(rhs.MetricsSecondStage)`.
7. Return `&RootExpr{Pipeline: pipelines, BatchSpanProcessor: spanProcs, SeriesProcessor: seriesProcs, MetricsSecondStage: &mathExpression{op: op, lhs: lhsMath, rhs: rhsMath}}`.

**Panic condition:** `asMathExpression` panics if `MetricsSecondStage` is non-nil
and not a `*mathExpression`. This indicates a grammar bug.

**Deduplication:** If `lhs` and `rhs` share a sub-query string, only one copy is
kept in the merged map. Both leaves will reference the same fragment key.

---

### 5.3 `unwrapSingleMathExpr`

```go
// ast.go:244-253
func unwrapSingleMathExpr(r *RootExpr) *RootExpr
```

Called when `metricsExpression` is used as the root without a binary operator.
If the root is NOT math (i.e. a single leaf), it removes the leaf `mathExpression`
wrapper and promotes any per-leaf filter to root-level `MetricsSecondStage`.

**Algorithm:**
1. If `r.IsMath()` → return `r` unchanged (binary math stays as-is).
2. If `r.MetricsSecondStage` is `*mathExpression` with `op == OpNone`:
   - Replace `r.MetricsSecondStage = m.filter` (may be `nil`).
3. Return `r`.

**Effect:** A single `({} | rate())` parses through `newWrappedMetricsPipeline` into
a leaf `mathExpression`, then `unwrapSingleMathExpr` drops the wrapper, making the
result identical to a plain `{} | rate()` query. `IsMath()` returns `false`.

---

### 5.4 `chainMathSecondStage`

```go
// ast.go:258-272
func chainMathSecondStage(r *RootExpr, stage ChainedSecondStage) *RootExpr
```

Appends a root-level `metricsSecondStagePipeline` after a `metricsExpression`.

**Algorithm — non-math path (`!r.IsMath()`):**
1. Unwrap the single leaf: `r = unwrapSingleMathExpr(r)`.
2. If `r.MetricsSecondStage != nil`, prepend it: `r.MetricsSecondStage = ChainedSecondStage{r.MetricsSecondStage, stage...}`.
3. Otherwise: `r.MetricsSecondStage = stage`.
4. Return `r`.

**Algorithm — math path (`r.IsMath()`):**
1. Prepend the existing math expression: `r.MetricsSecondStage = ChainedSecondStage{r.MetricsSecondStage, stage...}`.
2. Return `r`.

**Ordering invariant:** The `mathExpression` is always at index 0 of any
`ChainedSecondStage` so that arithmetic runs before topk/filter.

---

## 6. `mathExpression` Methods

### 6.1 `String() string`

```
Leaf:   key + filter.String() (filter omitted if nil)
Binary: "(" + lhs.String() + ") " + op.String() + " (" + rhs.String() + ")"
```

Example for `({a} | rate()) / ({b} | rate())`:
```
"({a} | rate()) / ({b} | rate())"
```

---

### 6.2 `validate() error`

Leaf:
1. If `filter != nil`, return `filter.validate()`.
2. Otherwise return `nil`.

Binary:
1. If `!op.isArithmetic()` → return `fmt.Errorf("unsupported math operation between queries: %s", op)`.
2. If `lhs.validate() != nil` → return that error.
3. Return `rhs.validate()`.

**Arithmetic operators:** `OpAdd`, `OpSub`, `OpMult`, `OpDiv` (defined in `enum_operators.go`).

---

### 6.3 `init(req *tempopb.QueryRangeRequest)`

Leaf: If `filter != nil`, call `filter.init(req)`.
Binary: Call `lhs.init(req)` then `rhs.init(req)`.

---

### 6.4 `separator() string`

`separator()` is NOT part of the `secondStageElement` interface. Per-element
separator strings are passed explicitly to `ChainedSecondStage.Append(element,
separator)` at construction time and stored in the `separators` field.

---

### 6.5 `process(input SeriesSet) SeriesSet`

Entry point for tree evaluation. `input` must contain all fragments for this
expression (i.e. the merged `SeriesSet` from `batchMetricsEvaluator.Results()`).

**Leaf path** (`op == OpNone`): delegates to `processLeaf(input)`.

**Binary path:**
1. `lhsName = m.lhs.metricName(input)` — extract `__name__` BEFORE children strip it.
2. `rhsName = m.rhs.metricName(input)` — same.
3. `lhs = m.lhs.process(input)` — fan-out: both children receive the same full input.
4. `rhs = m.rhs.process(input)`.
5. `result = applyBinaryOp(m.op, lhs, rhs)`.
6. Build `combinedName = "(lhsName op rhsName)"` if both names are non-empty.
7. Re-key `result`: for each series, prepend `{__name__: combinedName}` to its labels
   (if `combinedName != ""`), then re-insert into a new SeriesSet keyed by updated MapKey.
8. Return the new SeriesSet.

---

### 6.6 `processLeaf(input SeriesSet) SeriesSet`

Filters and strips internal labels from the combined input.

**Algorithm** (iterates over all entries in `input`):
1. For each `(smk, v)` in `input`:
   a. Get `fragmentValue = v.Labels.GetValue(internalLabelQueryFragment)`.
   b. If `fragmentValue` is not a string OR its string value `!= m.key` → `continue`.
   c. Build new `SeriesMapKey` by copying `smk` entries where `Name != __name__` AND
      `Name != __query_fragment`.
   d. Build a NEW `Labels` slice containing only labels where `Name != __name__` AND
      `Name != __query_fragment`. Assign to `v.Labels`.
   e. `result[newKey] = v`.
2. If `m.filter != nil`, apply: `result = m.filter.process(result)`.
3. Return `result`.

**Side-effect on labels:** `v.Labels` is replaced with a new Labels slice. The
original input series' label slices are NOT mutated.

---

### 6.7 `metricName(input SeriesSet) string`

Extracts the `__name__` label for use before `process` strips it.

**Leaf:** Iterate `input`. For each series whose `__query_fragment` matches `m.key`,
scan its labels for `__name__` and return the string value. Returns `""` if not found.

**Binary:** Recurse into `lhs.metricName(input)` and `rhs.metricName(input)`.
If both are non-empty, return `"(lhsName op rhsName)"`. Otherwise `""`.

---

## 7. `applyBinaryOp`

```go
// ast_metrics_math.go:176-217
func applyBinaryOp(op Operator, lhs, rhs SeriesSet) SeriesSet
```

Combines two SeriesSets element-wise using `op`.

**Algorithm:**
1. `target = lhs`. If `rhs` does NOT contain `noLabelsSeriesMapKey` (`SeriesMapKey{}`),
   use `target = rhs`. (This selects the set more likely to have exact-key matches.)
2. `result = make(SeriesSet, len(target))`.
3. **Pre-allocate value buffer (two passes):**
   - Pass 1: For each key `k` in `target`, call `getTSMatch` for both `lhs` and `rhs`.
     If both ok, record `keyAndN{k, n=min(len(l.Values), len(r.Values))}`. Sum all `n` → `totalN`.
   - Pass 2: Allocate `buf = make([]float64, totalN)`. Track `offset int`.
4. For each `{k, n}` pair from pass 1 (both sides already confirmed ok):
   a. Retrieve the pre-matched `l` and `r` from `getTSMatch` (no second lookup needed).
   b. `values = buf[offset : offset+n]`.
   c. For `j` in `[0, n)`: `values[j] = applyArithmeticOp(op, l.Values[j], r.Values[j])`.
   d. Pick labels: `ll = l.Labels` if non-empty, else `r.Labels`.
   e. `result[k] = TimeSeries{Labels: ll, Values: values, Exemplars: mergeExemplars(l.Exemplars, r.Exemplars)}`.
   f. `offset += n`.
5. Return `result`.

**Output cardinality:** ≤ min(len(lhs), len(rhs)). Series without a counterpart are dropped.

---

## 8. `getTSMatch`

```go
// ast_metrics_math.go:219-227
func getTSMatch(set SeriesSet, key SeriesMapKey) (TimeSeries, bool)
```

1. If `set[key]` exists, return it.
2. Else if `set[SeriesMapKey{}]` exists (no-labels fallback), return it.
3. Else return `TimeSeries{}, false`.

Purpose: supports scalar + vector arithmetic where one side has no labels.

---

## 9. `applyArithmeticOp`

```go
// ast_metrics_math.go:242-264
func applyArithmeticOp(op Operator, lhs, rhs float64) float64
```

**NaN normalisation (applied first):**
```
if math.IsNaN(lhs): lhs = 0
if math.IsNaN(rhs): rhs = 0
```

**Operation:**
```
OpAdd:  return lhs + rhs
OpSub:  return lhs - rhs
OpMult: return lhs * rhs
OpDiv:  if rhs == 0: return math.NaN()
        return lhs / rhs
default: return math.NaN()
```

Never panics. Invalid ops and division-by-zero return `NaN`.

---

## 10. `mergeExemplars`

```go
// ast_metrics_math.go:229-240
func mergeExemplars(a, b []Exemplar) []Exemplar
```

Returns `b` if `len(a)==0`, `a` if `len(b)==0`, otherwise allocates and appends
`a` then `b`. No deduplication.

---

## 11. `batchMetricsEvaluator`

```go
// engine_metrics.go:1608
type batchMetricsEvaluator map[string]*MetricsEvaluator
```

Implements `MetricsEvaluator`. Used only for math queries.

### 11.1 `Do`

Calls `Do` on each sub-evaluator sequentially. Returns on first error.

### 11.2 `Results() SeriesSet`

Merges per-fragment results into a single `SeriesSet`, tagging each series with
the fragment key:

```
for q, eval in e:
    for each v in eval.Results():
        v.Labels = v.Labels.Add({Name: "__query_fragment", Value: q})
        merged[v.Labels.MapKey()] = v
```

The `mathExpression` tree then consumes these tagged series via `processLeaf`.

---

## 12. `MetricsFrontendEvaluator`

```go
// engine_metrics.go:1619-1623
type MetricsFrontendEvaluator struct {
    mtx                sync.Mutex
    seriesProcessor    seriesProcessor
    metricsSecondStage secondStageElement
}
```

Frontend-side accumulator. Receives streamed backend results, applies the math tree.

### 12.1 `ObserveSeries(in []*tempopb.TimeSeries)`

```
lock(mtx)
seriesProcessor.observeSeries(in)
unlock(mtx)
```

For math queries, `seriesProcessor` is a `batchSeriesProcessor` which partitions
incoming series by their `__query_fragment` label and routes each batch to the
corresponding per-fragment sub-processor.

### 12.2 `Results() SeriesSet`

```
lock(mtx)
results = seriesProcessor.result(1.0)
if metricsSecondStage != nil:
    results = metricsSecondStage.process(results)
unlock(mtx)
return results
```

For math queries, `metricsSecondStage` is either a `*mathExpression` or a
`ChainedSecondStage` with it at index 0.

### 12.3 `Length() int`

```
lock(mtx)
return seriesProcessor.length()
unlock(mtx)
```

---

## 13. `batchSeriesProcessor`

```go
// ast_metrics.go:48
type batchSeriesProcessor map[string]seriesProcessor
```

Routes incoming `[]*tempopb.TimeSeries` batches to per-fragment sub-processors.

**`observeSeries(in []*tempopb.TimeSeries)` algorithm:**
1. Build `fragments map[string][]*tempopb.TimeSeries` by scanning labels of each series
   for the `__query_fragment` key.
2. For each `(fragmentKey, batch)` in `fragments`:
   - Look up `proc = batchSeriesProcessor[fragmentKey]`.
   - If found: `proc.observeSeries(batch)`.
   - If not found: silently discard.

---

## 14. `RootExpr` Math-related Methods

### 14.1 `IsMath() bool`

Returns `true` iff `MetricsSecondStage` contains a *binary* `mathExpression`:

```
if MetricsSecondStage is *mathExpression:
    return m.op != OpNone
if MetricsSecondStage is ChainedSecondStage and len > 0:
    if cs[0] is *mathExpression:
        return m.op != OpNone
return false
```

A leaf `mathExpression` (op == OpNone) is NOT math.

### 14.2 `SinglePipeline() (Pipeline, spanProcessor, bool)`

Returns the sole pipeline for non-math queries:
1. If `r.IsMath()` → return `(Pipeline{}, nil, false)`.
2. Iterate `r.Pipeline` and `r.BatchSpanProcessor` and return the first pair.

---

## 15. End-to-End Data Flow

```
Parse time:
  "({a}|rate()) + ({b}|rate())"
      ↓ wrappedMetricsPipeline × 2
  newWrappedMetricsPipeline({a}, rate(), nil) → RootExpr_A
      Pipeline: {"{ a } | rate()": Pipeline_A}
      BatchSpanProcessor: {"{ a } | rate()": SpanProc_A}
      MetricsSecondStage: &mathExpression{op: OpNone, key: "{ a } | rate()", filter: nil}

  newWrappedMetricsPipeline({b}, rate(), nil) → RootExpr_B
      ...similar...

      ↓ newRootExprMath(OpAdd, A, B)
  RootExpr_Math:
      Pipeline: {"{ a } | rate()": P_A, "{ b } | rate()": P_B}
      BatchSpanProcessor: {"{ a } | rate()": SP_A, "{ b } | rate()": SP_B}
      SeriesProcessor: batchSeriesProcessor{...}
      MetricsSecondStage: &mathExpression{
          op: OpAdd,
          lhs: &mathExpression{op: OpNone, key: "{ a } | rate()"},
          rhs: &mathExpression{op: OpNone, key: "{ b } | rate()"},
      }

Execution time (backend):
  batchMetricsEvaluator.Do(...)
      → SpanProc_A produces SeriesSet_A  (series with user labels)
      → SpanProc_B produces SeriesSet_B

  batchMetricsEvaluator.Results()
      → Each series in A gets label: __query_fragment = "{ a } | rate()"
      → Each series in B gets label: __query_fragment = "{ b } | rate()"
      → merged SeriesSet returned

Execution time (frontend, MetricsFrontendEvaluator):
  ObserveSeries([...tagged proto TimeSeries...])
      → batchSeriesProcessor routes A-tagged series to sub-proc_A
      → batchSeriesProcessor routes B-tagged series to sub-proc_B

  Results()
      → seriesProcessor.result(1.0) → merged SeriesSet with __query_fragment labels
      → metricsSecondStage.process(merged)
          → mathExpression{OpAdd}.process(merged)
              → lhsName = "rate"  (from A's __name__)
              → rhsName = "rate"
              → lhs = mathExpression{OpNone, "{ a } | rate()"}.process(merged)
                   → filters series where __query_fragment == "{ a } | rate()"
                   → strips __query_fragment, __name__
                   → returns cleaned SeriesSet_A
              → rhs = mathExpression{OpNone, "{ b } | rate()"}.process(merged)
                   → ...same for B...
              → result = applyBinaryOp(OpAdd, lhs, rhs)
              → re-key result with __name__ = "(rate + rate)"
              → return result
```

---

## 16. Validation Rules Summary

| Condition | Where checked | Error |
|-----------|---------------|-------|
| Leaf pipeline must be parenthesized | Parser (grammar) | parse error |
| Math operator must be arithmetic | `mathExpression.validate()` | "unsupported math operation between queries: …" |
| topk/bottomk limit must be > 0 | `TopKBottomK.validate()` | "limit must be greater than 0" |
| `compare()` incompatible with second stage | `ast_validate.go` | validated separately |

---

## 17. Concurrency Contract

| Type | Concurrent `ObserveSeries` calls | Concurrent `Results()` | Notes |
|------|----------------------------------|------------------------|-------|
| `MetricsFrontendEvaluator` | Safe (single mutex) | Safe (single mutex) | `ObserveSeries` and `Results()` both acquire `mtx` |
| `metricsEvaluator` | Safe (per-spanset mutex) | N/A | Lock held per spanset, not across full Fetch |
| `mathExpression` | N/A (stateless after `init`) | Safe | No mutable state after init |
| `batchMetricsEvaluator` | Sequential `Do` only | Safe | No concurrent `Do` calls |

---

## 18. Additional Invariants

### 18.1 `seriesProcessor` Interface

```go
// ast_metrics_math.go:14-18
type seriesProcessor interface {
    observeSeries(batch []*tempopb.TimeSeries)
    result(multiplier float64) SeriesSet
    length() int
}
```

Both the math path (`batchSeriesProcessor`) and individual `firstStageElement`
aggregators implement this interface. `result` is called exactly once per query
evaluation, applying any internal multipliers (e.g. rate normalisation). `length`
returns the number of unique series accumulated so far.

### 18.2 Exemplar Cardinality

`mergeExemplars` (ast_metrics_math.go:229-240) concatenates both slices without
deduplication. Combined exemplar slices from a binary operation may therefore
contain duplicate trace references. Downstream consumers (e.g. proto serialisation
via `SeriesSet.ToProto`) are responsible for filtering if uniqueness is required.
This is an intentional design choice — exemplar deduplication at combine time adds
overhead not justified for most query volumes.

### 18.3 `IsMath()` Access Safety

`IsMath()` (ast.go:47-59) accesses `ChainedSecondStage[0]` only after confirming
`len > 0`. The deeper assumption — that index 0 is always a `*mathExpression` for
math queries — is enforced structurally by `chainMathSecondStage`, which always
prepends the math expression at position 0. No runtime assertion guards this
assumption at the access site; correctness depends on the invariant in §5.4
(ordering invariant) being maintained by all callers.

### 18.4 `init()` Call Discipline

`secondStageElement.init()` and `mathExpression.init()` are designed to be called
**exactly once** before the first `process()` call. The interface and implementations
do not enforce idempotency (no `sync.Once` guard). Calling `init()` a second time
overwrites child element state with the new `QueryRangeRequest`, producing undefined
behaviour. Callers (`MetricsFrontendEvaluator`, `CompileMetricsQueryRangeMath`) must
ensure `init()` is invoked once per query lifetime.
