# TraceQL Query Benchmarks

This directory contains benchmarks for TraceQL query execution using the Rust DataFusion-based implementation.

## Overview

The benchmarks in `benches/traceql_queries.rs` mirror the Go benchmarks found in `tempodb/encoding/vparquet4/block_traceql_test.go`, allowing performance comparison between the Go and Rust implementations.

### What Gets Benchmarked

The benchmark suite includes 18 different query patterns:

1. **Span Attribute Matching**
   - Exact match: `{ span.component = 'net/http' }`
   - Regex match: `{ span.component =~ 'database/sql' }`
   - No match scenarios

2. **Resource Attribute Matching**
   - Custom attributes: `{ resource.opencensus.exporterversion = 'Jaeger-Go-2.30.0' }`
   - Intrinsic attributes: `{ resource.service.name = 'tempo-gateway' }`

3. **Trace-Level Queries**
   - OR operators: `{ rootServiceName = 'tempo-distributor' && (status = error || span.http.status_code = 500)}`

4. **Mixed Queries**
   - Combining resource and span conditions with AND/OR

5. **Complex Queries**
   - Multiple conditions with regex: `{resource.k8s.cluster.name =~ "prod.*" && resource.k8s.namespace.name = "hosted-grafana" && ...}`

### Architecture Differences from Go

The Rust implementation uses a different approach:

- **Go**: Custom iterator-based Parquet scanning with TraceQL native execution
- **Rust**: TraceQL → SQL conversion + DataFusion execution engine

This means the benchmarks measure:
- **Go**: Direct Parquet iteration + predicate pushdown
- **Rust**: TraceQL parsing + SQL generation + DataFusion optimization + Parquet scanning

## Setup

### 1. Prepare Test Data

You need a Tempo vparquet4 block on local disk. The Rust benchmarks use the **same environment variables as the Go benchmarks** for consistency.

#### Option A: Use Existing Local Tempo Storage

If you have Tempo blocks in local storage:

```bash
# Set the environment variables (same as Go benchmarks)
export BENCH_BLOCKID="030c8c4f-9d47-4916-aadc-26b90b1d2bc4"
export BENCH_PATH="/path/to/tempo/storage"
export BENCH_TENANTID="single-tenant"  # Optional, defaults to "1"

# Verify the block exists
ls -lh "$BENCH_PATH/$BENCH_TENANTID/$BENCH_BLOCKID/data.parquet"
```

#### Option B: Copy Block from S3 or Remote Storage

If your blocks are in S3:

```bash
# Download the block structure
export BLOCK_ID="030c8c4f-9d47-4916-aadc-26b90b1d2bc4"
export TENANT_ID="single-tenant"

# Create local directory structure
mkdir -p "/tmp/tempo-bench/$TENANT_ID/$BLOCK_ID"

# Copy data.parquet from S3 (adjust to your setup)
aws s3 cp "s3://your-tempo-bucket/$TENANT_ID/$BLOCK_ID/data.parquet" \
    "/tmp/tempo-bench/$TENANT_ID/$BLOCK_ID/data.parquet"

# Set environment variables
export BENCH_BLOCKID="$BLOCK_ID"
export BENCH_PATH="/tmp/tempo-bench"
export BENCH_TENANTID="$TENANT_ID"
```

#### Option C: Generate Test Data

If you don't have existing blocks:

1. Run Tempo locally with trace ingestion
2. Wait for a block to be flushed to storage
3. Note the block ID from the storage directory
4. Set the environment variables pointing to the storage directory

### 2. Verify Environment Variables

```bash
# Required variables (same as Go)
echo "BENCH_BLOCKID: $BENCH_BLOCKID"
echo "BENCH_PATH: $BENCH_PATH"
echo "BENCH_TENANTID: $BENCH_TENANTID"  # Optional, defaults to "1"

# Verify the file exists at the expected path
FILE_PATH="$BENCH_PATH/${BENCH_TENANTID:-1}/$BENCH_BLOCKID/data.parquet"
echo "File path: $FILE_PATH"
ls -lh "$FILE_PATH"
```

### 3. Run Benchmarks (Same Command Line as Go)

```bash
# Run all benchmarks
cargo bench --bench traceql_queries

# Run specific benchmark group
cargo bench --bench traceql_queries traceql

# Run a specific query benchmark
cargo bench --bench traceql_queries spanAttValMatch
```

## Understanding Output

### Standard Criterion Output

```
traceql/query/spanAttValMatch
                        time:   [123.45 ms 125.67 ms 127.89 ms]
                        thrpt:  [7.82 Kelem/s 7.96 Kelem/s 8.10 Kelem/s]
```

- **time**: Mean execution time with confidence intervals
- **thrpt**: Throughput (elements per second)

### Custom Metrics Output

During execution, you'll see additional metrics printed:

```
spanAttValMatch: 1234 rows returned
resourceAttIntrinsicMatch: 567 rows returned
complex: 89 rows returned
```

This shows how many rows (spans) matched each query, similar to the `spans/op` metric in Go benchmarks.

## Comparing with Go Benchmarks

### Go Benchmark Output

```
BenchmarkBackendBlockTraceQL/spanAttValMatch-8    100    10234567 ns/op   12.5 MB_io/op   1234 spans/op
```

### Rust Benchmark Output

```
traceql/query/spanAttValMatch  time: [10.234 ms 10.567 ms 10.890 ms]
spanAttValMatch: 1234 rows returned
```

### Key Differences

1. **Time Units**: Go uses ns/op, Criterion uses ms (default) or auto-scales
2. **I/O Metrics**: Go tracks MB_io/op directly; Rust implementation would need DataFusion instrumentation for equivalent metrics
3. **Iterations**: Both frameworks automatically determine optimal iteration count

## Benchmarking Best Practices

### 1. Use the Same Setup as Go Benchmarks

The Rust benchmarks use the **exact same environment variables** as the Go benchmarks, making it easy to compare:

```bash
# Set the same variables you use for Go benchmarks
export BENCH_BLOCKID="030c8c4f-9d47-4916-aadc-26b90b1d2bc4"
export BENCH_PATH="/path/to/tempo/storage"
export BENCH_TENANTID="single-tenant"  # Optional, defaults to "1"
```

### 2. System Preparation

```bash
# Close unnecessary applications
# Disable CPU frequency scaling (Linux)
sudo cpupower frequency-set --governor performance

# Verify your machine is idle
top  # or htop
```

### 2. Block Selection

Choose a representative block:
- **Size**: 10-100 MB for reasonable benchmark times
- **Content**: Contains diverse trace data matching your queries
- **Time range**: Recent data with realistic attribute distributions

### 3. Consistent Environment

```bash
# Create a benchmark script (using same variables as Go)
cat > run-bench.sh <<'EOF'
#!/bin/bash
set -e

# Same variables as Go benchmarks
export BENCH_BLOCKID="030c8c4f-9d47-4916-aadc-26b90b1d2bc4"
export BENCH_PATH="/path/to/tempo/storage"
export BENCH_TENANTID="${BENCH_TENANTID:-1}"  # Defaults to "1" like Go

# Construct path and verify file exists
FILE_PATH="$BENCH_PATH/$BENCH_TENANTID/$BENCH_BLOCKID/data.parquet"
if [ ! -f "$FILE_PATH" ]; then
    echo "Error: $FILE_PATH not found"
    echo "Block structure: $BENCH_PATH/$BENCH_TENANTID/$BENCH_BLOCKID/"
    exit 1
fi

# Show file info
echo "Benchmarking with file:"
ls -lh "$FILE_PATH"
echo

# Run Rust benchmarks
cargo bench --bench traceql_queries -- --save-baseline rust-baseline

# Optionally run Go benchmarks for comparison
# cd /path/to/tempo
# go test -bench=BenchmarkBackendBlockTraceQL ./tempodb/encoding/vparquet4/
EOF

chmod +x run-bench.sh
./run-bench.sh
```

### 4. Baseline Comparison

```bash
# Save initial baseline
cargo bench --bench traceql_queries -- --save-baseline before

# Make changes to code...

# Compare against baseline
cargo bench --bench traceql_queries -- --baseline before
```

## Extending the Benchmarks

### Adding New Queries

Edit `benches/traceql_queries.rs` and add to `get_test_cases()`:

```rust
BenchCase {
    name: "myNewQuery",
    traceql: "{ span.http.method = `POST` && duration > 100ms }",
},
```

### Adding Custom Metrics

To track additional metrics (e.g., I/O bytes), you can:

1. Instrument the DataFusion execution plan
2. Access metrics via `ExecutionPlan::metrics()`
3. Report custom metrics using Criterion's API

Example in `execute_query()`:

```rust
// Get execution plan metrics
if let Some(plan) = df.execution_plan() {
    let metrics = plan.metrics();
    // Extract and report custom metrics
}
```

## Troubleshooting

### Error: "BENCH_BLOCKID is not set"

```bash
# Set the required environment variables (same as Go)
export BENCH_BLOCKID="030c8c4f-9d47-4916-aadc-26b90b1d2bc4"
export BENCH_PATH="/path/to/tempo/storage"
export BENCH_TENANTID="single-tenant"  # Optional, defaults to "1"
```

### Error: "BENCH_PATH is not set"

```bash
export BENCH_PATH="/path/to/tempo/storage"
```

### Error: "Parquet file not found"

Verify the block structure exists:

```bash
FILE_PATH="$BENCH_PATH/${BENCH_TENANTID:-1}/$BENCH_BLOCKID/data.parquet"
echo "Looking for: $FILE_PATH"
ls -l "$FILE_PATH"
file "$FILE_PATH"  # Should show "Parquet file"

# Check the block directory structure
ls -la "$BENCH_PATH/${BENCH_TENANTID:-1}/$BENCH_BLOCKID/"
```

### Error: "Failed to parse TraceQL query"

Some complex queries (structural operators like `>>`, aggregations like `count()`, projections like `select()`) may not be fully supported yet in the TraceQL parser. These queries will be skipped with a warning message.

### Slow Benchmarks

If benchmarks take too long:

1. Use a smaller block (< 50 MB)
2. Reduce sample size in the benchmark code
3. Use `--sample-size 10` flag

```bash
cargo bench --bench traceql_queries -- --sample-size 10
```

## Future Improvements

- [ ] Add I/O byte tracking via DataFusion metrics
- [ ] Implement MB/s throughput calculation
- [ ] Add memory profiling using `criterion::profiler`
- [ ] Support pagination parameters (StartPage, TotalPages)
- [ ] Add flamegraph generation for performance analysis
- [ ] Implement parallel query execution benchmarks

## Resources

- [Criterion.rs Documentation](https://bheisler.github.io/criterion.rs/book/)
- [DataFusion Benchmarking Guide](https://arrow.apache.org/datafusion/user-guide/introduction.html)
- [Tempo TraceQL Documentation](https://grafana.com/docs/tempo/latest/traceql/)
