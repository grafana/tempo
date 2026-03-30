# TraceQL Metrics Math — Specification

This document is the authoritative specification for binary arithmetic on aggregated
TraceQL metrics. It is detailed enough to guide a complete re-implementation.
Companion documents: TESTS.md (test plan), BENCHMARKS.md (performance targets).

---

## 1. Scope

The TraceQL package is responsible for parsing, validating, and evaluating queries
that combine aggregated trace metrics using binary arithmetic (+, -, *, /).

| Concern | Owner |
|---------|-------|
| Parsing and validating math expressions | TraceQL package |
| Binary arithmetic on time-series sets | TraceQL package |
| Second-stage pipeline (topk, bottomk, filter) | TraceQL package |
| First-stage span aggregation (rate, count_over_time, …) | TraceQL package |
| Fragment label routing | TraceQL package |
| Protocol-buffer serialization | tempopb package |
| Storage layer | tempodb package |
| Distributed shard orchestration | modules/frontend |

The fragment label (`__query_fragment`) is the sole internal mechanism for tagging
and routing sub-query results. No code outside the TraceQL package may set or depend
on it.

---

## 2. Grammar

Math queries extend the existing metrics query syntax. Each operand must be a
fully parenthesized pipeline:

```
// Valid:   ({status=error} | rate()) / ({} | rate())
// Invalid: {status=error} | rate() / {} | rate()   ← parse error (no outer parens)
```

Supported binary operators: `+`, `-`, `*`, `/`.

Operators may be nested and grouped with additional parentheses:
```
(({a} | rate()) + ({b} | rate())) / ({c} | rate())
```

A trailing second-stage pipeline (topk, bottomk, filter) is permitted after the
math expression and applies after arithmetic is complete.

A single parenthesized pipeline with no binary operator is treated as a plain
(non-math) metrics query — the math wrapper is removed automatically during parsing.

---

## 3. Core Data Structures

### 3.1 Time-Series Set

A time-series set maps label-key → time series. Each time series carries:
- A set of key/value labels
- A slice of float64 values (one per query step)
- A slice of exemplars

This is the primary unit of data throughout the second stage.

### 3.2 Second-Stage Element

All second-stage processors share a common contract:
- **Initialize** — called exactly once before the first evaluate call, receiving
  the full query parameters (start, end, step, exemplar limit).
- **Evaluate** — receives the current time-series set and returns a new set.
  Must not mutate the input set.

Implementations: math expression node, top-k/bottom-k, metrics filter,
chained pipeline.

### 3.3 Math Expression Node

A single node type represents both leaf and binary cases:

- **Leaf node**: represents one sub-query, identified by a fragment key. May carry
  an optional per-leaf second-stage filter.
- **Binary node**: combines two child nodes (left and right) using one arithmetic
  operator (+, -, *, /).

### 3.4 Chained Pipeline

A sequence of second-stage elements where each element's output feeds the next
element's input. When a math expression is part of the chain it always runs first
(before topk/filter operations).

---

## 4. AST Construction

### 4.1 Leaf Construction

Each parenthesized pipeline becomes a leaf with a unique fragment key derived from
the canonical string representation of its spanset pipeline and aggregation
function. Identical sub-queries produce the same key, enabling automatic
deduplication — only one copy is stored and evaluated.

### 4.2 Binary Construction

Combining two expressions with a binary operator merges their sub-query maps.
Duplicate keys from both sides are collapsed to a single entry; both leaves
reference the same fragment key.

If either side carries an existing math expression node that cannot be cast to the
expected type, the construction panics — this indicates a grammar bug, not a
runtime condition.

### 4.3 Single-Leaf Unwrapping

When a parenthesized pipeline appears without a binary operator, the math wrapper
is removed and any per-leaf filter is promoted to the root level. The result is
indistinguishable from a plain metrics query and is not considered a math query.

### 4.4 Chain Assembly

When a root-level second-stage pipeline follows a math expression, the math node
is placed at index 0 and the remaining stages follow. This ordering ensures
arithmetic runs before any topk or filter operations.

---

## 5. Math Expression Evaluation

### 5.1 String Representation

- **Leaf**: fragment key, followed by the filter's string representation if present.
- **Binary**: `"(" + left + ") " + operator + " (" + right + ")"`

### 5.2 Validation

- **Leaf**: delegates to the leaf filter's validation if one is set.
- **Binary**: rejects any non-arithmetic operator (only +, -, *, / are valid);
  then validates both children.

### 5.3 Initialization

- **Leaf**: initializes the leaf filter if present.
- **Binary**: initializes the left child, then the right child.

Initialization must be called exactly once per query lifetime. The implementation
does not guard against double-initialization; callers are responsible for ensuring
single invocation.

### 5.4 Evaluation (binary node)

1. Extract the metric name from the input set **before** children strip labels.
2. Pass the full input set to both children independently (fan-out).
3. Combine the two result sets element-wise using the binary operator.
4. Build a combined metric name `"(leftName op rightName)"` if both sides produced
   a non-empty name.
5. Re-label the result with the combined name and return the new set.

### 5.5 Evaluation (leaf node)

For each series in the input:
1. Read the fragment label value.
2. Skip the series if it does not match this leaf's key.
3. Build a new label set and map key, excluding the fragment label and the name
   label. **Do not modify the original label slice** — allocate a new one.
4. Apply the leaf filter if present.

The prohibition on in-place mutation is critical: when two leaves share a fragment
key (deduplicated sub-queries), both operate on the same underlying series. Mutating
the first leaf's labels would corrupt the second leaf's lookup.

### 5.6 Metric Name Extraction

- **Leaf**: scans the input for a series matching this leaf's key; returns its name
  label value, or empty string if absent.
- **Binary**: recurses into both children; returns `"(leftName op rightName)"` only
  if both are non-empty; otherwise empty string.

---

## 6. Element-wise Arithmetic

### 6.1 Binary Combination

Combines two time-series sets element-wise:

1. Choose the set more likely to have exact label-key matches as the target
   (the set without a no-labels entry).
2. **Two-pass buffer allocation** (O(1) allocations regardless of set size):
   - Pass 1: for each key in the target, find the matching series in both sets.
     Record the pair and the number of values to combine (`min(left, right)`).
     Sum all counts to get the total buffer size.
   - Pass 2: allocate one contiguous buffer. Slice it into per-series segments.
3. For each matched pair, apply scalar arithmetic to each aligned value.
4. Series present in only one set are dropped silently.

Output cardinality ≤ min(left set size, right set size).

The two-pass approach is required for correctness. A single-pass pre-allocation
using only the target set's value count under-counts when one side has fewer values
than the other, causing slice bounds violations.

### 6.2 Key-Match Lookup

To find the matching series for a given key:
1. Return the series at that exact key if present.
2. Otherwise return the series at the empty (no-labels) key if present.
3. Otherwise: no match.

The no-labels fallback supports scalar + vector arithmetic where one operand has no
label dimensions (e.g. a constant rate).

### 6.3 Scalar Arithmetic

For each pair of float64 values:
1. Normalize NaN inputs to 0 before operating (matches Prometheus convention).
2. Apply the operator: add, subtract, multiply, or divide.
3. Division by zero returns NaN. Invalid operators return NaN.
4. Never panics.

NaN normalization applies to inputs only. Division-by-zero in the result is not
normalized — it propagates as NaN to signal a missing or meaningless data point.

### 6.4 Exemplar Merging

- If one slice is empty, return the other unchanged.
- Otherwise allocate a new slice and append both. No deduplication is performed.
  Downstream consumers are responsible for filtering duplicates if required.

---

## 7. Fragment Routing

### 7.1 Backend Batch Evaluator

Used only for math queries on the backend. Holds one per-fragment evaluator keyed
by fragment string.

- **Execute**: runs each sub-evaluator sequentially against the same data source.
  Returns on first error.
- **Results**: merges per-fragment outputs into one set, tagging each series with
  its fragment key label before merging.

Sequential execution is safe because each shard evaluates all fragments against the
same underlying data.

### 7.2 Fragment Router (Frontend)

The frontend-side processor that receives streamed backend results for math queries.
Scans each incoming series for the fragment label. Routes the series to the
sub-processor registered for that fragment key. Silently discards series whose
fragment key is not recognized.

### 7.3 Series Aggregator Interface

Both the math path and the non-math path share a common aggregation interface with
three operations:
- **Observe** — receive a batch of raw proto time-series.
- **Result** — produce the final time-series set, applying any internal multipliers.
- **Length** — return the current series count.

`Result` is called exactly once per query evaluation.

---

## 8. Frontend Accumulator

Receives streamed shard results and produces the final response.

- Protected by a single mutex covering both the series processor and the
  second-stage pipeline. Concurrent observe and result calls are safe.
- **Observe**: locks, routes the batch through the series processor, unlocks.
- **Result**: locks, calls the series processor to get the merged set, then applies
  the second-stage pipeline (which for math queries contains the math expression
  tree), then unlocks.
- **Length**: locks, queries the series processor, unlocks.

For math queries the series processor is the fragment router (§7.2) and the
second-stage is the math expression tree. For non-math queries, the series processor
is a single aggregator and the second-stage may be nil or a topk/filter chain.

---

## 9. Root Expression Properties

### 9.1 IsMath predicate

Returns true if and only if the root's second-stage contains a **binary** math
expression node (one with an operator set). A leaf math expression node (no
operator) is **not** math — it was unwrapped during parsing.

The predicate checks both the direct case (second-stage is a math node) and the
chained case (second-stage is a pipeline whose first element is a math node). The
bounds check (length > 0) is always performed; the assumption that index 0 is a
math node when it is present is enforced structurally during parse-time chain
assembly.

---

## 10. End-to-End Data Flow

```
Parse time:
  "({a}|rate()) + ({b}|rate())"
      ↓ leaf construction × 2
  Two leaf expressions, each with a unique fragment key.
      ↓ binary construction
  Root: two sub-query maps merged, math tree at second stage.

Backend execution (per shard):
  Batch evaluator executes each sub-query fragment against local data.
  Results tagged with fragment key label.
  Tagged time-series returned to frontend.

Frontend execution:
  Fragment router receives tagged series, routes to per-fragment sub-processor.
  On Results():
    Merge all sub-processor outputs.
    Math expression tree evaluates:
      → Each leaf filters to its fragment, strips internal labels.
      → Binary node combines matched series element-wise.
      → Names rebuilt as "(leftName op rightName)".
    Final clean time-series set returned.
```

---

## 11. Validation Rules

| Condition | When checked |
|-----------|-------------|
| Each leaf pipeline must be parenthesized | Parser (grammar) |
| Math operator must be arithmetic (+, -, *, /) | Math expression validate() |
| Top-k/bottom-k limit must be > 0 | Top-k/bottom-k validate() |

---

## 12. Concurrency Contract

| Component | Concurrent observe calls | Concurrent result calls | Notes |
|-----------|--------------------------|------------------------|-------|
| Frontend accumulator | Safe (mutex) | Safe (mutex) | Single mutex covers both |
| Per-shard evaluator | Safe (per-spanset mutex) | N/A | Lock is per-spanset |
| Math expression node | N/A (stateless after init) | Safe | No mutable state after init |
| Backend batch evaluator | Sequential only | Safe | Do not call execute concurrently |

---

## 13. Backend Caller Contract

When executing a math query on a backend shard:

1. Parse the query and check whether it is a math query.
2. If math: compile using the math-aware backend compiler, which returns a batch
   evaluator (one per fragment). Execute the batch evaluator against the data
   source. Collect results — these will be tagged with fragment labels.
3. If not math: compile using the standard single-evaluator compiler. Execute
   and collect results — these will be clean time-series without fragment labels.

In both cases, serialize the results and return them to the frontend. The frontend
accumulator handles routing and combination transparently.

---

## 14. Frontend Caller Contract

When constructing the frontend accumulator for a query range request:

1. Parse the query and determine whether it is math.
2. For both math and non-math: use the same accumulator constructor with the
   "final aggregation" mode. The constructor internally detects math and configures
   the appropriate series processor (fragment router vs. single aggregator) and
   second-stage pipeline (math tree vs. nil/topk/filter).
3. For each streamed shard response, call Observe with the series batch.
4. When all shards are complete, call Result to produce the final response.

No special branching is required at the handler level — the accumulator constructor
handles the math vs. non-math distinction internally.
