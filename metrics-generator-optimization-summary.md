# Metrics Generator Optimization Summary

This branch optimizes the Tempo metrics-generator hot path without intentionally changing config defaults, metric names, labels, label values, filtering behavior, or user-visible semantics.

## Main Improvements

- **Less label allocation:** registry label builders now reuse scratch storage and borrowed labels for hot updates, avoiding repeated label copying and hash recomputation.
- **Cheaper spanmetrics processing:** spanmetrics avoids unnecessary target_info work for filtered spans, caches/reuses common resource-derived values, fast-paths intrinsic enum labels, and avoids allocating trace ID strings for classic histogram exemplars.
- **Cheaper servicegraph processing:** servicegraphs reuses edge/key buffers, avoids trace ID formatting in server lookup paths, reduces peer/resource scans, and avoids allocation-heavy edge update patterns.
- **Lower registry overhead:** counters, gauges, histograms, limiter updates, timestamps, and exemplar handling were optimized to reduce per-span/per-series CPU and memory churn.
- **Faster filter matching:** common span kind intrinsic filters and regex patterns now use fast paths instead of general-purpose matching.

## Corrected Benchmark Results vs `origin/main`

| Benchmark group | What it measures | CPU | Memory | Allocations |
|---|---|---:|---:|---:|
| **All config benchmarks** | Broad coverage of default, spanmetrics, servicegraphs, combined, target_info, dimensions, and native histogram configs | **-45.2%** | **-92.1%** | **-94.5%** |
| **Prod-shaped configs** | Production-like configs with 7 dimensions, filters, target_info, servicegraph prefixes, prod buckets, and native histograms | **-51.8%** | **-92.9%** | **-96.0%** |
| **100k active-series** | Steady-state updates after seeding about 100k active series | **-33.4%** | **-97.4%** | **-99.3%** |
| **1M active-series validation** | Heavier combined prod-shaped workload after seeding about 1M active series | **-27.3%** | **-95.8%** | **-99.5%** |

## Interpretation

The biggest gains come from removing allocation-heavy label, trace ID, and servicegraph key work from the per-span/per-update path.

The benchmark evidence is strong for metrics-generator CPU and memory hot paths, especially high-cardinality tenants. The exact percentages should still be validated with production-like replay or profiles before treating them as production-wide savings.
