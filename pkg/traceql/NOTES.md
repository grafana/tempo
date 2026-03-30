# traceql — Design Notes

This document captures non-obvious design decisions, rationale, and invariants for
the `traceql` package (metrics math subsystem). These notes complement SPECS.md and
are intended to prevent re-introducing decisions that were deliberately reversed.

Entries are append-only. If a decision is reversed, add an *Addendum* to the original
entry and a new entry for the replacement decision.

---

## 1. Tree Structure in Second Stage, Flat Maps in RootExpr
*Added: 2026-03-30*

**Decision:** Math results are combined by a `mathExpression` binary tree in the second
stage, but all sub-queries are stored in flat maps (`Pipeline`, `BatchSpanProcessor`)
in `RootExpr`.

**Why:** Flat maps enable deduplication of identical sub-queries (same key → same entry),
allow independent backend evaluation of each pipeline, and simplify the backend/frontend
split: backends produce per-pipeline `SeriesSet`; the frontend combines them via the tree.
The tree is only needed at result-combination time.

**How to apply:** When adding new math operators or pipeline types, register sub-queries
in the flat maps via `newRootExprMath`; do not embed evaluation logic in the AST walker.

---

## 2. Fragment Labeling Strategy
*Added: 2026-03-30*

**Decision:** An internal `__query_fragment` label is attached to every series at
evaluation time and consumed (stripped) by `mathExpression.process()`.

**Why:** Avoids rewriting the AST for math queries. Enables clean batch routing:
`batchSeriesProcessor` partitions incoming series by this label without knowing about
math semantics. Keeps span processors oblivious to arithmetic.

**How to apply:** The label is an internal implementation detail. No code outside
`pkg/traceql` should set or depend on `__query_fragment`. If you add a new internal
label, document it here and strip it in the appropriate second-stage element.

---

## 3. Leaf Node vs. Binary Node via `op` Field, Not Type Hierarchy
*Added: 2026-03-30*

**Decision:** A single `mathExpression` struct represents both leaf nodes (`op == OpNone`)
and binary nodes (`op != OpNone`). No separate `LeafExpression` / `BinaryExpression` types.

**Why:** Simplifies the grammar (one `metricsExpression` rule covers all), avoids an extra
pointer indirection for leaf nodes, and keeps `String()`, `init()`, `validate()` logic
unified. `OpNone` is an unambiguous sentinel for "this is a leaf".

**How to apply:** When adding new node types to the math AST, prefer extending this struct
with an extra field guarded by an `op`-like sentinel, rather than creating parallel types,
unless the behaviour diverges significantly enough to warrant it.

---

## 4. NaN-Based Error Handling in Arithmetic
*Added: 2026-03-30*

**Decision:** `applyArithmeticOp` coerces NaN inputs to `0`, returns NaN for
division-by-zero and invalid operators. It never panics or returns an error.

**Why:** Metrics pipelines process large volumes of time-series samples; error
propagation per sample would add overhead and complicate aggregation. NaN propagates
naturally and is filterable downstream. Prometheus uses the same convention.

**How to apply:** Do not add error returns to `applyArithmeticOp` or `applyBinaryOp`.
If a new operator needs to signal an error condition, use NaN. Document the NaN
semantics in SPECS.md.

---

## 5. No-Labels Fallback in `applyBinaryOp`
*Added: 2026-03-30*

**Decision:** If an exact label-key match is not found, `applyBinaryOp` falls back to
`SeriesMapKey{}` (empty key, no labels).

**Why:** Supports scalar + vector arithmetic where one query produces un-grouped results
(no labels). Prevents hard failures when grouping dimensions differ. The conservative
strategy — skip mismatched series rather than fail — was chosen to match Prometheus
semantics.

**How to apply:** If you add label-matching modes (e.g., `by(...)` / `without(...)`)
in the future, this fallback should become conditional on those modes.

---

## 6. Single Mutex per `MetricsFrontendEvaluator`
*Added: 2026-03-30*

**Decision:** A single `sync.Mutex` protects both `seriesProcessor` and `metricsSecondStage`
inside `MetricsFrontendEvaluator`. No fine-grained per-fragment locking.

**Why:** Simplicity: the evaluator is created per-query and lives for the duration of one
range query. Lock contention is bounded by the number of concurrent shard responses, which
is controlled by the frontend. The simpler model was preferred over per-fragment maps with
their own locks.

**How to apply:** If profiling shows mutex contention as a bottleneck, consider per-fragment
locks on the `seriesProcessor` map. Until then, keep the single mutex.

---

## 7. `ChainedSecondStage` — Math Always at Index 0
*Added: 2026-03-30*

**Decision:** When a `ChainedSecondStage` contains a `mathExpression`, it is always at
index 0 (arithmetic executes before topk/filter).

**Why:** Math operates on raw fragment-labelled series; topk/filter expect clean,
label-stripped results. Reversing the order would produce incorrect results (topk would
see `__query_fragment` labels).

**How to apply:** `chainMathSecondStage` enforces this order at parse time. If you add
new second-stage elements, ensure they are appended after index 0, never inserted before
the math expression.

---

## 8. `unwrapSingleMathExpr` — Avoids Leaf Wrapper for Non-Math Queries
*Added: 2026-03-30*

**Decision:** When a `metricsExpression` resolves to a single leaf (no binary op),
`unwrapSingleMathExpr` discards the leaf `mathExpression` wrapper, returning a plain
`RootExpr`.

**Why:** Single-query metrics pipelines do not need fragment labelling or second-stage
fan-in. Keeping the wrapper would add unnecessary complexity (a second-stage pass that
does nothing except strip an internal label). The unwrapped path re-uses the existing
simpler single-evaluator path.

**How to apply:** Maintain this invariant: `IsMath()` must return `false` for single-leaf
expressions. Tests in `parse_test.go` (TestMetricsMathExpression) verify this.

---

## 9. Value Buffer Pre-allocation in `applyBinaryOp`
*Added: 2026-03-30*

**Decision:** All combined value slices for the result SeriesSet are allocated in a
single backing array before the arithmetic loop.

**Why:** Benchmarks (commit `6026f7ec3 Benchmarks`) showed that N separate allocations
for N series caused measurable GC pressure. A single pre-allocated buffer reduces
allocations to O(1) per binary operation regardless of series count.

**How to apply:** Do not introduce per-series allocations inside `applyBinaryOp`. If the
result structure changes (e.g., adding histogram buckets), extend the pre-allocation to
cover the new fields.

---

## 10. Two-Stage Evaluation Architecture
*Added: 2026-03-30*

**Decision:** Math evaluation is split into Stage 1 (backends produce per-fragment
time series) and Stage 2 (frontend combines via `mathExpression` tree).

**Why:** Enables distributed execution without shipping raw spans to a central node.
Each backend independently evaluates its assigned pipelines and streams compressed
time series. The frontend only needs to do arithmetic on small series — not re-process
spans. This matches the existing backend/frontend split used for non-math metrics queries.

**How to apply:** Any new aggregation type that needs cross-query combination (e.g.,
per-query quantiles) should follow this pattern: backend produces labelled partial
results, frontend combines. The combination logic belongs in a new `secondStageElement`
implementation.

---

## 11. Math Evaluator Initialization — checkTime/start/end Are Required
*Added: 2026-03-30*

**Decision:** `CompileMetricsQueryRangeMath` now sets `me.checkTime = true`, `me.start`,
`me.end`, and applies the span-start-time precision and exemplar condition setup for each
sub-evaluator in the batch — mirroring what `CompileMetricsQueryRange` does.

**Why:** Without `checkTime`, the span-time filter in `MetricsEvaluator.Do` is bypassed,
causing spans outside the query window to be counted. This produces inflated metric values
for any math query (CRITICAL correctness bug). The precision and exemplar setups are needed
for correct span-start-time fetching and exemplar metadata, respectively.

**How to apply:** Any new evaluator construction path must copy this initialization block.
Extract it into a helper if the pattern is repeated more than twice.

---

## 12. NeedsFullTrace for Math Queries — Check r.Pipelines, Not r.Pipeline
*Added: 2026-03-30*

**Decision:** `RootExpr.NeedsFullTrace()` now also iterates `r.Pipelines` (the math
sub-query map) in addition to `r.Pipeline.Elements`.

**Why:** For math queries, `r.Pipeline` is always empty; sub-pipelines live in
`r.Pipelines`. Without this fix, `NeedsFullTrace()` always returned `false` for math
queries, so trace-level aggregates (`count()`, `>>`) always attempted the span-only fast
path and produced wrong results. Additionally, `CompileMetricsQueryRangeMath` was changed
to call `NeedsFullTrace` per sub-pipeline (iterating the pipeline's Elements) rather than
on the root expr, so each sub-evaluator gets the correct value.

**How to apply:** If new query fields are added that hold pipelines (analogous to
`r.Pipelines`), update `NeedsFullTrace()` to check them too.

---

## 13. MapKey() Truncates at maxGroupBys — Safe for Internal Label Overflow
*Added: 2026-03-30*

**Decision:** `Labels.MapKey()` now breaks early if the label index reaches `maxGroupBys`,
instead of panicking with an index-out-of-bounds.

**Why:** `batchSeriesProcessor.result()` and `batchMetricsEvaluator.Results()` append
`__query_fragment` to every series before calling `MapKey()`. If a series already has
`maxGroupBys` user-facing labels (e.g., 5-dimension group-by), the append would push the
length to 6, causing a panic. Truncation is safe: the extra labels are internal and only
need to produce a stable map key, which the truncated key still provides.

**Implication:** For math queries, the effective user-visible group-by dimension limit is
`maxGroupBys - 1 = 4` (one slot is reserved for `__query_fragment`). Document this in
user-facing docs if exposed.

---

## 14. processLeaf Must Not Mutate Input Labels — Use New Slice
*Added: 2026-03-30*

**Decision:** `processLeaf` now builds a new `Labels` slice instead of compacting
in-place. The original series entries in `input` are never modified.

**Why:** When two leaves share the same fragment key (deduplicated identical sub-queries
like `({} | rate()) / ({} | rate())`), both leaves operate on the same `input` map
entries. In-place compaction of the first leaf would strip `__query_fragment` from the
shared backing array; the second leaf's `GetValue(internalLabelQueryFragment)` call would
then return `false`, causing it to return an empty set and producing `0 / x = 0` instead
of `x / x = 1`.

**How to apply:** Any `secondStageElement.process` that modifies series labels must
allocate a new slice. The spec contract (§3.2) that `process` must not mutate `input`
is now enforced by construction.

---

## 15. applyBinaryOp Two-Pass Buffer Allocation
*Added: 2026-03-30*

**Decision:** `applyBinaryOp` now uses a two-pass approach: pass 1 determines the actual
`n = min(len(l.Values), len(r.Values))` per key; pass 2 allocates a single backing buffer
of size `sum(n)` and fills results.

**Why:** The previous single-pass approach pre-allocated using `len(target[k].Values)` but
advanced `offset += n` (the actual consumed count). When `n < len(target[k].Values)` (e.g.,
scalar fallback with fewer values), `offset` advanced by less than the budget, causing the
next iteration's `buf[offset : offset+n_next]` to exceed the buffer bounds and panic.

**Trade-off:** Two passes over the target keys (first to compute n, second to fill values).
Since `target` is a hash map of series (typically tens to hundreds of entries, not millions),
the extra pass has negligible overhead. The O(1) allocation invariant (one backing buffer per
`applyBinaryOp` call) is preserved.
