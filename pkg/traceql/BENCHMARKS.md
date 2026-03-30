# traceql — Benchmark Specifications (Metrics Math)

This document defines the benchmark suite for the metrics math subsystem of the
`traceql` package. Benchmarks measure cost independently of I/O by pre-building
inputs outside the timed loop.

Benchmark IDs use the prefix `BENCH-TRQL`.

---

## Metric Targets (Never Regress Below)

| Metric | Good | Warning | Critical |
|--------|------|---------|----------|
| `applyBinaryOp` — 100 series, 120 steps | < 100 µs | 100–300 µs | > 300 µs |
| `applyBinaryOp` — 1 000 series, 120 steps | < 1 ms | 1–3 ms | > 3 ms |
| `mathExpression.process` — depth-3 tree, 100 series | < 500 µs | 500 µs–1.5 ms | > 1.5 ms |
| Parse: binary math expression | < 20 µs | 20–50 µs | > 50 µs |
| `MetricsFrontendEvaluator.ObserveSeries` — 1 000 series | < 200 µs | 200–600 µs | > 600 µs |

*Targets are aspirational until baselines are established. Run the benchmarks on merge
to main and record actuals in this table.*

---

## 1. `applyBinaryOp` Benchmarks

### BENCH-TRQL-01: BenchmarkApplyBinaryOp

Measures the cost of combining two SeriesSets via a binary operator.

**Setup (outside timed loop):**
- Build `lhs` and `rhs` SeriesSets with identical label keys.
- Pre-fill each series with `steps` float64 values (simulate one query range).
- Select `op = OpDiv` (most representative: all four ops have similar cost).

**Variants:**

| Sub-benchmark | Series count | Steps per series |
|---------------|-------------|-----------------|
| `_small`      | 10          | 120             |
| `_medium`     | 100         | 120             |
| `_large`      | 1 000       | 120             |
| `_deep`       | 100         | 1 440           |

**Required custom metrics:**
```go
b.ReportMetric(float64(seriesCount), "series")
b.ReportMetric(float64(steps), "steps")
b.ReportMetric(float64(seriesCount*steps*b.N)/elapsed.Seconds(), "values/sec")
```

---

### BENCH-TRQL-02: BenchmarkApplyArithmeticOp

Measures the scalar arithmetic hot path (called per-value inside `applyBinaryOp`).

**Setup (outside timed loop):** None (pure computation, no allocation).

**Variants:**

| Sub-benchmark | Operator |
|---------------|----------|
| `_add`        | OpAdd    |
| `_sub`        | OpSub    |
| `_mul`        | OpMult   |
| `_div`        | OpDiv    |
| `_divzero`    | OpDiv (rhs=0) |

**Required custom metrics:**
```go
b.ReportMetric(float64(b.N)/elapsed.Seconds(), "ops/sec")
```

---

## 2. `mathExpression.process` Benchmarks

### BENCH-TRQL-03: BenchmarkMathExpressionProcess

Measures full tree evaluation including leaf filtering, label stripping, and
recursive binary combination.

**Setup (outside timed loop):**
- Build a balanced binary tree of `mathExpression` nodes of depth D.
- Build a SeriesSet containing `seriesPerLeaf * leafCount` series, each tagged with
  the appropriate `__query_fragment` key.
- Call `init(req)` with a representative `QueryRangeRequest`.

**Variants:**

| Sub-benchmark | Tree depth | Series per leaf | Total series |
|---------------|-----------|----------------|-------------|
| `_depth1`     | 1 (single binary) | 50 | 100 |
| `_depth2`     | 2         | 25             | 100 |
| `_depth3`     | 3         | 12             | 96  |

**Required custom metrics:**
```go
b.ReportMetric(float64(treeDepth), "depth")
b.ReportMetric(float64(totalSeries), "total_series")
b.ReportMetric(float64(totalSeries*b.N)/elapsed.Seconds(), "series_evals/sec")
```

---

## 3. Parse Benchmarks

### BENCH-TRQL-04: BenchmarkParseMetricsMath

Measures query parsing overhead for math expressions.

**Setup (outside timed loop):**
- Prepare query strings of varying complexity.

**Variants:**

| Sub-benchmark | Expression |
|---------------|-----------|
| `_simple`     | `({} \| rate()) / ({} \| rate())` |
| `_grouped`    | `({status=error} \| rate()) / ({} \| rate())` |
| `_nested`     | `(({a} \| rate()) + ({b} \| rate())) / ({c} \| rate())` |

**Required custom metrics:**
```go
b.ReportMetric(float64(b.N)/elapsed.Seconds(), "parses/sec")
```

---

## 4. Frontend Evaluator Benchmarks

### BENCH-TRQL-05: BenchmarkMetricsFrontendEvaluatorObserveSeries

Measures throughput of `ObserveSeries` under concurrent shard responses.

**Setup (outside timed loop):**
- Build a `MetricsFrontendEvaluator` from a math `RootExpr`.
- Pre-build `[]*tempopb.TimeSeries` batches of varying sizes.
- Each series carries a `__query_fragment` label matching a fragment in the evaluator.

**Variants:**

| Sub-benchmark | Series per call | Goroutines |
|---------------|----------------|------------|
| `_serial_100`   | 100  | 1  |
| `_serial_1000`  | 1000 | 1  |
| `_parallel_100` | 100  | 8  |

**Required custom metrics:**
```go
b.ReportMetric(float64(seriesPerCall*b.N)/elapsed.Seconds(), "series/sec")
b.ReportMetric(float64(goroutines), "goroutines")
```

---

## 5. Allocation Benchmarks

### BENCH-TRQL-06: BenchmarkApplyBinaryOpAllocs

Verifies the O(1) allocation invariant of `applyBinaryOp`.

**Setup:** Same as BENCH-TRQL-01 `_medium` variant.

**Assertions (via `b.ReportAllocs` + test assertions):**
- Allocations per op must be ≤ 3 regardless of series count.
- If allocations grow with series count, the pre-allocation optimisation has regressed.

```go
// In the test body, after b.ResetTimer():
allocs := testing.AllocsPerRun(100, func() {
    applyBinaryOp(OpDiv, lhs, rhs)
})
if allocs > 3 {
    t.Errorf("expected ≤3 allocs per call, got %.0f", allocs)
}
```
