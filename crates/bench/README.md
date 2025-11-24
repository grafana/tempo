# Bench - TraceQL Query Execution Tool

A command-line utility for executing TraceQL queries against Tempo blocks.

**Note:** This tool uses streaming execution - results are processed incrementally without loading them into memory, making it suitable for profiling large queries without excessive memory usage.

## Usage

### With TraceQL queries

```bash
cargo run -p bench -- --traceql "{ } | count()"
```

### With SQL queries directly

```bash
cargo run -p bench -- --sql "SELECT COUNT(*) FROM spans"
```

## Environment Variables

Before running the tool, you need to set the following environment variables:

- `BENCH_BLOCKID`: GUID of the block to query (e.g., "030c8c4f-9d47-4916-aadc-26b90b1d2bc4")
- `BENCH_PATH`: Root path to the backend storage (e.g., "/path/to/tempo/storage")
- `BENCH_TENANTID`: Tenant ID (optional, defaults to "1")

## Examples

```bash
export BENCH_BLOCKID=030c8c4f-9d47-4916-aadc-26b90b1d2bc4
export BENCH_PATH=/path/to/tempo/storage
export BENCH_TENANTID=1

# Run a TraceQL query
cargo run -p bench -- --traceql "{ } | count()"

# Run a SQL query directly (bypass TraceQL conversion)
cargo run -p bench -- --sql "SELECT COUNT(*) FROM spans WHERE status = 'error'"
```

## Output

The tool will display:
- The converted SQL query
- Number of rows returned
- Bytes scanned (with MB)
- Elapsed time in milliseconds
- Throughput in MB/s

## Building

```bash
# Build the bench crate
cargo build -p bench

# Build with optimizations
cargo build -p bench --release

# Run directly with TraceQL
./target/release/bench --traceql "{ span.http.status_code = 500 }"

# Run directly with SQL
./target/release/bench --sql "SELECT * FROM spans WHERE span_http_status_code = 500"
```

## Profiling

Use with `cargo flamegraph` for performance profiling:

```bash
# Profile a TraceQL query
cargo flamegraph -p bench -- --traceql "{ } | count()"

# Profile a SQL query directly
cargo flamegraph -p bench -- --sql "SELECT COUNT(*) FROM spans"
```
