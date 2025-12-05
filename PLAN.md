# High-Performance vParquet4 Rust Reader Implementation Plan

## Progress Summary

- ✅ **Phase 1: Foundation + Tests** - COMPLETED
- ⏳ **Phase 2: Filtering & Projection** - TODO
- ⏳ **Phase 3: Domain Types** - TODO
- 🚧 **Phase 4: Iterators & Async** - PARTIAL (placeholder files)
- 🚧 **Phase 5: TraceQL Benchmarks** - PARTIAL (basic benchmarks)
- ✅ **Phase 6: Integration** - COMPLETED

**Current Status:** Phase 1 complete with all tests passing. Ready to proceed with Phase 2 (filtering & projection).

## Overview

Implement a standalone Rust crate `crates/vparquet4` for high-performance **read-only** access to Tempo's vParquet4 trace format. The crate will be **standalone** (no DataFusion dependency) and provide both low-level Parquet access and high-level domain APIs.

## Key Resources

- **Existing Rust schema**: `crates/storage/src/vparquet4.rs` (369 lines, complete Arrow schema)
- **Go reference schema**: `tempodb/encoding/vparquet4/schema.go`
- **Test data**: `tempodb/encoding/vparquet4/test-data/single-tenant/b27b0e53-66a0-4505-afd6-434ae3cd4a10/data.parquet` (77KB, 134 traces)
- **Go tests**: 18 test files in `tempodb/encoding/vparquet4/`

## Crate Structure

```
crates/vparquet4/
├── Cargo.toml
├── src/
│   ├── lib.rs                    # Public API exports
│   ├── error.rs                  # Error types (VParquet4Error)
│   ├── schema/
│   │   ├── mod.rs
│   │   ├── field_paths.rs        # Column path constants (trace::*, span::*, attr::*)
│   │   └── validation.rs         # Schema validation
│   ├── reader/
│   │   ├── mod.rs                # Reader trait + ReaderConfig
│   │   ├── sync_reader.rs        # VParquet4Reader<File>
│   │   └── async_reader.rs       # AsyncVParquet4Reader (object_store)
│   ├── filter/
│   │   ├── mod.rs
│   │   ├── statistics.rs         # RowGroupStats extraction
│   │   └── row_group.rs          # RowGroupFilter (time range, trace ID prefix)
│   ├── projection/
│   │   ├── mod.rs
│   │   └── builder.rs            # ProjectionBuilder (trace_summary_only, spans_without_attrs, full_spans)
│   ├── domain/
│   │   ├── mod.rs                # Re-exports
│   │   ├── trace.rs              # Trace struct
│   │   ├── span.rs               # Span struct + SpanKind, StatusCode enums
│   │   ├── resource.rs           # Resource, ResourceSpans, ScopeSpans
│   │   ├── attribute.rs          # Attribute, AttributeValue, DedicatedAttributes
│   │   ├── event.rs              # Event struct
│   │   └── link.rs               # Link struct
│   └── iter/
│       ├── mod.rs
│       ├── trace_iter.rs         # TraceIterator
│       └── span_iter.rs          # SpanIterator (flattened)
└── benches/
    └── read_benchmark.rs         # Criterion benchmarks
```

## Implementation Phases

### Phase 1: Foundation + Tests ✅ COMPLETED
Create crate structure, error types, schema constants, basic sync reader.

**Files to create:**
- [x] `crates/vparquet4/Cargo.toml`
- [x] `crates/vparquet4/src/lib.rs`
- [x] `crates/vparquet4/src/error.rs`
- [x] `crates/vparquet4/src/schema/mod.rs`
- [x] `crates/vparquet4/src/schema/field_paths.rs`
- [x] `crates/vparquet4/src/schema/validation.rs`
- [x] `crates/vparquet4/src/reader/mod.rs`
- [x] `crates/vparquet4/src/reader/sync_reader.rs`

**Tests:** ✅ Read Go test data file, validate schema, count traces (7 integration tests + 2 unit tests passing).

### Phase 2: Filtering & Projection + Tests ⏳ TODO
Row group statistics, filtering, column projection.

**Files to create:**
- [ ] `crates/vparquet4/src/filter/mod.rs`
- [ ] `crates/vparquet4/src/filter/statistics.rs`
- [ ] `crates/vparquet4/src/filter/row_group.rs`
- [ ] `crates/vparquet4/src/projection/mod.rs`
- [ ] `crates/vparquet4/src/projection/builder.rs`

**Tests:** Time range filtering, trace ID prefix filtering, projection modes.

### Phase 3: Domain Types + Tests ⏳ TODO
Trace/Span/Attribute structs with Arrow parsing.

**Files to create:**
- [ ] `crates/vparquet4/src/domain/mod.rs`
- [ ] `crates/vparquet4/src/domain/trace.rs`
- [ ] `crates/vparquet4/src/domain/span.rs`
- [ ] `crates/vparquet4/src/domain/resource.rs`
- [ ] `crates/vparquet4/src/domain/attribute.rs`
- [ ] `crates/vparquet4/src/domain/event.rs`
- [ ] `crates/vparquet4/src/domain/link.rs`

**Tests:** Parse traces from Go test data, verify all fields, attribute extraction.

### Phase 4: Iterators & Async + Tests 🚧 PARTIAL
High-level iteration APIs, async reader for object stores.

**Files to create:**
- [ ] `crates/vparquet4/src/iter/mod.rs`
- [ ] `crates/vparquet4/src/iter/trace_iter.rs`
- [ ] `crates/vparquet4/src/iter/span_iter.rs`
- [x] `crates/vparquet4/src/reader/async_reader.rs` (placeholder)

**Tests:** TraceIterator, SpanIterator (flattened), async reader with local object store.

### Phase 5: TraceQL-Style Benchmarks 🚧 PARTIAL
Replicate the Go benchmark `BenchmarkBackendBlockTraceQL` from `tempodb/encoding/vparquet4/block_traceql_test.go:1448`.

**Files to create:**
- [x] `crates/vparquet4/benches/read_benchmark.rs` (basic benchmarks)
- [ ] `crates/vparquet4/benches/traceql_benchmark.rs`

**Benchmark scenarios** (from Go test):
- **Span attributes**: Match by value (`span.component = net/http`), regex, no match
- **Span intrinsics**: Match by name, few matches, no match
- **Resource attributes**: Match by value, intrinsic (service.name), no match
- **Trace-level**: OR queries with rootServiceName, status, http.status_code
- **Mixed**: Cross-level AND/OR queries, regex on k8s.cluster.name
- **Complex**: Multiple conditions with duration, regex, dedicated columns
- **Aggregations**: count(), rate(), select()

**Metrics to report**:
- `MB_io/op`: Megabytes read per operation (from row group statistics)
- `spans/op`: Average spans matched per query
- Time per operation (default Criterion metric)

**Environment variables** (matching Go):
- `BENCH_BLOCKID`: Block UUID to benchmark against
- `BENCH_PATH`: Path to backend storage
- `BENCH_TENANTID`: Tenant ID (default: "single-tenant")

**Implementation approach**:
1. Create benchmark harness that loads a real vParquet4 block from disk
2. Implement simplified TraceQL predicate evaluation (no full parser needed)
3. Use projection + row group filtering to minimize I/O
4. Track bytes read via reader statistics
5. Compare results with Go benchmark for validation

### Phase 6: Integration ✅ COMPLETED
Update workspace, wire up with existing crates.

**Files to modify:**
- [x] `Cargo.toml` (add `crates/vparquet4` to workspace members)

## Key Types

```rust
// Reader configuration
pub struct ReaderConfig {
    pub batch_size: usize,              // default: 8192
    pub prefetch_row_groups: usize,     // default: 2
    pub parallel_column_decode: bool,   // default: true
}

// Row group filter
pub struct RowGroupFilter {
    time_range: Option<(u64, u64)>,     // (min_ns, max_ns)
    trace_id_prefix: Option<Vec<u8>>,
}

// Projection presets
impl ProjectionBuilder {
    pub fn trace_summary_only() -> Self;   // TraceID, times, root service/span
    pub fn spans_without_attrs() -> Self;  // Core span fields, no attrs
    pub fn full_spans() -> Self;           // Everything
}

// Domain types
pub struct Trace { trace_id: [u8; 16], resource_spans: Vec<ResourceSpans>, ... }
pub struct Span { span_id: [u8; 8], name: String, attrs: Vec<Attribute>, ... }
pub enum AttributeValue { String(String), Int(i64), Double(f64), Bool(bool), ... }
```

## Cargo.toml Dependencies

```toml
[dependencies]
parquet = { workspace = true }             # 54.3.1, async feature
arrow = "54.0.0"                           # Arrow arrays for zero-copy
object_store = { workspace = true }        # S3/GCS/Azure/local backends
async-trait = { workspace = true }
tokio = { workspace = true }
futures-util = { workspace = true }
anyhow = { workspace = true }
thiserror = "1.0"
bytes = "1.0"
tracing = { workspace = true }

[dev-dependencies]
criterion = { workspace = true }
tokio-test = "0.4"

[features]
default = ["async"]
async = []

[[bench]]
name = "read_benchmark"
harness = false
```

**Note:** No DataFusion dependency - standalone parquet/arrow crate. The existing `crates/storage/src/vparquet4.rs` schema can be referenced but not depended on directly to keep the crate standalone.

## Test Data Reference

From `meta.json`:
- Format: vParquet4
- Traces: 134
- Time range: 2022-07-04T11:11:09Z to 2022-07-04T11:11:35Z
- Dedicated columns: ip (resource), instance, version, region, sampler.type (span)

## Performance Targets

1. **Column pruning**: Read only needed columns via ProjectionMask
2. **Row group skipping**: Filter by StartTimeUnixNano/TraceID statistics
3. **Zero-copy**: Reuse Arrow RecordBatch where possible
4. **Async I/O**: Non-blocking reads from object stores
5. **Parallelism**: Concurrent row group decoding
