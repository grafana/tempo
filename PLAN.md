# High-Performance vParquet4 Rust Reader Implementation Plan

## Progress Summary

- ✅ **Phase 1: Foundation + Tests** - COMPLETED
- ✅ **Phase 2: Filtering & Projection** - COMPLETED
- ✅ **Phase 3: Domain Types** - COMPLETED
- ✅ **Phase 4: Iterators & Async** - COMPLETED
- 🚧 **Phase 5: TraceQL Benchmarks** - PARTIAL (basic benchmarks)
- ✅ **Phase 6: Integration** - COMPLETED

**Current Status:** Phase 4 complete with TraceIterator, SpanIterator, and AsyncVParquet4Reader implementations. All library unit tests passing (26/26). Ready to proceed with Phase 5 (TraceQL benchmarks).

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

### Phase 2: Filtering & Projection + Tests ✅ COMPLETED
Row group statistics, filtering, column projection.

**Files created:**
- [x] `crates/vparquet4/src/filter/mod.rs`
- [x] `crates/vparquet4/src/filter/statistics.rs`
- [x] `crates/vparquet4/src/filter/row_group.rs`
- [x] `crates/vparquet4/src/projection/mod.rs`
- [x] `crates/vparquet4/src/projection/builder.rs`
- [x] `crates/vparquet4/tests/test_filtering.rs` (7 integration tests)
- [x] `crates/vparquet4/tests/test_projection.rs` (6 integration tests)

**Tests:** ✅ Time range filtering, trace ID prefix filtering, projection modes (13 integration tests + 6 unit tests passing).

**Implementation Notes:**
- Row group statistics extraction supports time ranges and trace ID prefixes
- Filtering uses Parquet metadata to skip irrelevant row groups
- Projection supports top-level column selection (TraceSummaryOnly, SpansWithoutAttrs, FullSpans)
- Note: Due to nested vParquet4 structure, attribute filtering requires Phase 3 domain types

### Phase 3: Domain Types + Tests ✅ COMPLETED
OTLP domain types using prost-compiled protobuf definitions with Arrow-to-OTLP conversion layer.

**Files created:**
- [x] `crates/vparquet4/build.rs` - Compiles OpenTelemetry proto files
- [x] `crates/vparquet4/src/domain/mod.rs` - Exports prost-generated OTLP types
- [x] `crates/vparquet4/src/domain/convert.rs` - Arrow to OTLP conversion utilities

**Implementation Notes:**
- Uses official OpenTelemetry protobuf definitions (`opentelemetry-proto`) compiled with prost
- Provides conversion functions from Arrow RecordBatch to OTLP types (Span, Resource, etc.)
- vParquet4 uses a **denormalized schema** where Resource (`rs`) and ScopeSpans (`ss`) are sibling lists at the top level, not nested as in standard OTLP
- Conversion layer handles mapping from flat vParquet4 structure to nested OTLP hierarchy
- Integration tests verify parsing infrastructure (8 tests in `test_domain.rs` and `test_debug_domain.rs`)

**Dependencies added:**
- `prost = "0.13"` - Runtime protobuf support
- `prost-types = "0.13"` - Well-known protobuf types
- `prost-build = "0.13"` - Build-time proto compilation
- `hex = "0.4"` - For trace/span ID hex encoding

### Phase 4: Iterators & Async + Tests ✅ COMPLETED
High-level iteration APIs, async reader for object stores.

**Files created:**
- [x] `crates/vparquet4/src/iter/mod.rs` - Module exports for TraceIterator and SpanIterator
- [x] `crates/vparquet4/src/iter/trace_iter.rs` - TraceIterator with Trace struct
- [x] `crates/vparquet4/src/iter/span_iter.rs` - SpanIterator with SpanWithContext
- [x] `crates/vparquet4/src/reader/async_reader.rs` - Full async implementation with object_store support
- [x] `crates/vparquet4/tests/test_iterators.rs` - Integration tests (10 tests)

**Tests:** ✅ All library unit tests passing (26/26), including async reader tests with LocalFileSystem.

**Implementation Notes:**
- TraceIterator yields complete Trace objects with all ResourceSpans, ScopeSpans, and Spans
- SpanIterator flattens the nested structure and yields individual SpanWithContext objects
- AsyncVParquet4Reader supports reading from S3, GCS, Azure, and local filesystem via object_store
- Upgraded to parquet 57.0.0 and arrow 57.0.0 for better object_store compatibility
- All iterator types exported in lib.rs for public API access

### Phase 5: TraceQL-Style Benchmarks 🚧 PARTIAL
Replicate the Go benchmark `BenchmarkBackendBlockTraceQL` from `tempodb/encoding/vparquet4/block_traceql_test.go:1448`.

**Files to create:**
- [x] `crates/vparquet4/benches/read_benchmark.rs` (basic benchmarks)
- [ ] `crates/vparquet4/benches/traceql_benchmark.rs`

**Benchmark scenarios** (from Go test BenchmarkBackendBlockTraceQL):
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
- Use workspace .env file

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
