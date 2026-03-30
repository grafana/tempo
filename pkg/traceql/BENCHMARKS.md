# TraceQL Metrics Math — Benchmark Specifications

This document defines the benchmark suite for the metrics math subsystem.
Benchmarks measure computation cost independently of I/O — inputs are built
outside the timed loop.

Benchmark IDs use the prefix `BENCH-TRQL`.

---

## Performance Targets

| Operation | Good | Warning | Critical |
|-----------|------|---------|----------|
| Element-wise combination — 100 series, 120 steps | < 100 µs | 100–300 µs | > 300 µs |
| Element-wise combination — 1 000 series, 120 steps | < 1 ms | 1–3 ms | > 3 ms |
| Full tree evaluation — depth-3, 100 series | < 500 µs | 500 µs–1.5 ms | > 1.5 ms |
| Parse: binary math expression | < 20 µs | 20–50 µs | > 50 µs |
| Frontend observe — 1 000 series per call | < 200 µs | 200–600 µs | > 600 µs |

*Targets are aspirational until baselines are established. Record actuals on first merge.*

---

## 1. Element-wise Combination

### BENCH-TRQL-01: Varying series count and step count

Measures the cost of combining two time-series sets.

**Setup (outside timed loop):**
- Build left and right sets with identical label keys.
- Pre-fill each series with N float64 values.

**Variants:**

| Sub-benchmark | Series | Steps |
|---------------|--------|-------|
| Small         | 10     | 120   |
| Medium        | 100    | 120   |
| Large         | 1 000  | 120   |
| Deep          | 100    | 1 440 |

**Metrics to report:** series count, step count, values processed per second.

---

### BENCH-TRQL-02: Scalar arithmetic throughput

Measures the per-value arithmetic hot path called inside element-wise combination.

**Setup:** None — pure computation, no allocation.

**Variants:** One sub-benchmark per operator: add, subtract, multiply, divide,
divide-by-zero.

**Metrics to report:** operations per second.

---

## 2. Tree Evaluation

### BENCH-TRQL-03: Full tree evaluation at varying depths

Measures complete evaluation of a balanced binary tree: leaf filtering,
label stripping, and recursive combination.

**Setup (outside timed loop):**
- Build a balanced binary tree of the specified depth.
- Build an input set containing `series-per-leaf × leaf-count` series, each tagged
  with the appropriate fragment key.
- Initialize the tree with a representative query parameters object.

**Variants:**

| Sub-benchmark | Depth | Series per leaf | Total series |
|---------------|-------|----------------|-------------|
| Single binary | 1     | 50             | 100         |
| Depth-2       | 2     | 25             | 100         |
| Depth-3       | 3     | 12             | 96          |

**Metrics to report:** tree depth, total series, series evaluations per second.

---

## 3. Parse

### BENCH-TRQL-04: Query parse overhead

Measures parsing cost for math expressions of varying complexity.

**Variants:**

| Sub-benchmark | Expression |
|---------------|-----------|
| Simple        | `({} \| rate()) / ({} \| rate())` |
| With filter   | `({status=error} \| rate()) / ({} \| rate())` |
| Nested        | `(({a} \| rate()) + ({b} \| rate())) / ({c} \| rate())` |

**Metrics to report:** parses per second.

---

## 4. Frontend Observe

### BENCH-TRQL-05: Streaming series throughput

Measures how quickly the frontend accumulator processes streamed backend results
for a math query.

**Setup (outside timed loop):**
- Build an accumulator configured for a math query.
- Pre-build batches of proto time-series of varying sizes, each carrying a valid
  fragment label.

**Variants:**

| Sub-benchmark | Series per call | Goroutines |
|---------------|----------------|------------|
| Serial small  | 100            | 1          |
| Serial large  | 1 000          | 1          |
| Parallel      | 100            | 8          |

**Metrics to report:** series processed per second, goroutine count.

---

## 5. Allocation Invariant

### BENCH-TRQL-06: O(1) allocations in element-wise combination

Verifies that the two-pass buffer pre-allocation achieves O(1) allocations
regardless of input size.

**Setup:** Same as BENCH-TRQL-01 medium variant.

**Assertions:**
- Allocations per call must be ≤ 3 regardless of series count.
- If allocations scale with series count, the pre-allocation has regressed.
