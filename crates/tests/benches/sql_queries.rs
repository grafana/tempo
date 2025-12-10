use anyhow;
use context::{collect_plan_metrics, create_block_context};
use criterion::{criterion_group, criterion_main, BenchmarkId, Criterion, Throughput};
use datafusion::execution::context::SessionContext;
use datafusion::physical_plan::execute_stream;
use futures::StreamExt;
use std::sync::Arc;
use std::time::Duration;
use storage::BlockInfo;
use tokio::runtime::Runtime;

/// Metrics collected from query execution
#[derive(Debug, Clone)]
struct QueryMetrics {
    rows_returned: usize,
    bytes_scanned: usize,
    elapsed_nanos: u128,
}

impl QueryMetrics {
    /// Calculate throughput in MB/s
    #[allow(dead_code)]
    fn throughput_mbps(&self) -> f64 {
        if self.elapsed_nanos == 0 {
            return 0.0;
        }
        let bytes_per_sec =
            (self.bytes_scanned as f64) / (self.elapsed_nanos as f64 / 1_000_000_000.0);
        bytes_per_sec / (1024.0 * 1024.0)
    }

    /// Calculate MB of I/O per operation
    #[allow(dead_code)]
    fn mb_io_per_op(&self) -> f64 {
        (self.bytes_scanned as f64) / (1024.0 * 1024.0)
    }
}

/// Execute a SQL query and return metrics using collect_plan_metrics from context
/// Bypasses DataFrame to avoid performance overhead: SQL -> LogicalPlan -> PhysicalPlan -> Execute
async fn execute(ctx: &SessionContext, sql: &str) -> anyhow::Result<QueryMetrics> {
    use datafusion::sql::parser::DFParser;
    use datafusion::sql::sqlparser::dialect::GenericDialect;
    use std::time::Instant;

    let start = Instant::now();

    // Parse SQL to AST
    let dialect = GenericDialect {};
    let statements = DFParser::parse_sql_with_dialect(sql, &dialect)?;
    let statement = &statements[0];

    // Create logical plan
    let logical_plan = ctx.state().statement_to_plan(statement.clone()).await?;

    // Optimize logical plan
    let optimized_plan = ctx.state().optimize(&logical_plan)?;

    // Create physical plan
    let physical_plan = ctx.state().create_physical_plan(&optimized_plan).await?;

    // Execute the physical plan using execute_stream
    let task_ctx = ctx.task_ctx();
    let mut stream = execute_stream(physical_plan.clone(), task_ctx)?;

    // Process and discard each batch as soon as it's received
    let mut rows_returned: usize = 0;
    while let Some(batch_result) = stream.next().await {
        let batch = batch_result?;
        rows_returned += batch.num_rows();
        // Batch is immediately dropped here after counting rows
    }

    let elapsed = start.elapsed();

    // Collect metrics from physical plan
    let metrics = collect_plan_metrics(physical_plan);

    // Extract bytes_scanned
    let bytes_scanned = metrics
        .get("bytes_scanned")
        .and_then(|s| s.parse::<usize>().ok())
        .unwrap_or(0);

    Ok(QueryMetrics {
        rows_returned,
        bytes_scanned,
        elapsed_nanos: elapsed.as_nanos(),
    })
}

/// Benchmark test case with pure SQL query
struct BenchCase {
    name: &'static str,
    sql: &'static str,
}

/// Optimized SQL queries matching the TraceQL benchmark queries
/// These are hand-written SQL queries optimized for performance:
/// - Minimal projections (only what's needed)
/// - Filters pushed down early in the unnesting chain
/// - Direct column access when available (no UDF overhead)
/// - Avoiding unnecessary attribute map conversions
///
/// Each query is loaded from a separate SQL file in the queries/ directory
/// with the original TraceQL query as a comment at the top of each file.
fn get_test_cases() -> Vec<BenchCase> {
    vec![
        //BenchCase { name: "mixed", sql: include_str!("queries/mixed.sql") },
        //BenchCase { name: "mixedValMixedMatchAnd", sql: include_str!("queries/mixedValMixedMatchAnd.sql") },
        //BenchCase { name: "mixedValMixedMatchOr", sql: include_str!("queries/mixedValMixedMatchOr.sql") },
        //BenchCase { name: "mixedValNoMatch", sql: include_str!("queries/mixedValNoMatch.sql") },
        //BenchCase { name: "resourceAttValNoMatch", sql: include_str!("queries/resourceAttValNoMatch.sql") },
        //BenchCase { name: "spanAttIntrinsicMatchFew", sql: include_str!("queries/spanAttIntrinsicMatchFew.sql") },
        //BenchCase { name: "spanAttIntrinsicNoMatch", sql: include_str!("queries/spanAttIntrinsicNoMatch.sql") },
        //BenchCase { name: "spanAttValMatch", sql: include_str!("queries/spanAttValMatch.sql") },
        //BenchCase { name: "spanAttValMatchFew", sql: include_str!("queries/spanAttValMatchFew.sql") },
        //BenchCase { name: "struct", sql: include_str!("queries/struct.sql") },
        //BenchCase { name: "traceOrMatch", sql: include_str!("queries/traceOrMatch.sql") },
        //BenchCase { name: "traceOrNoMatch", sql: include_str!("queries/traceOrNoMatch.sql") },
        BenchCase {
            name: "||",
            sql: include_str!("queries/pipeOr.sql"),
        },
        BenchCase {
            name: "complex",
            sql: include_str!("queries/complex.sql"),
        },
        BenchCase {
            name: "count",
            sql: include_str!("queries/count.sql"),
        },
        BenchCase {
            name: "rate",
            sql: include_str!("queries/rate.sql"),
        },
        BenchCase {
            name: "resourceAttIntrinsicMatch",
            sql: include_str!("queries/resourceAttIntrinsicMatch.sql"),
        },
        BenchCase {
            name: "resourceAttIntrinsicNoMatch",
            sql: include_str!("queries/resourceAttIntrinsicNoMatch.sql"),
        },
        BenchCase {
            name: "resourceAttValMatch",
            sql: include_str!("queries/resourceAttValMatch.sql"),
        },
        BenchCase {
            name: "select",
            sql: include_str!("queries/select.sql"),
        },
        BenchCase {
            name: "spanAttIntrinsicMatch",
            sql: include_str!("queries/spanAttIntrinsicMatch.sql"),
        },
        BenchCase {
            name: "spanAttValNoMatch",
            sql: include_str!("queries/spanAttValNoMatch.sql"),
        },
        BenchCase {
            name: "traceOrMatchFew",
            sql: include_str!("queries/traceOrMatchFew.sql"),
        },
    ]
}

/// Get block info and object store from environment variables
/// - BENCH_BLOCKID: GUID of the block (e.g., "030c8c4f-9d47-4916-aadc-26b90b1d2bc4")
/// - BENCH_PATH: Root path to backend (e.g., "/path/to/tempo/storage")
/// - BENCH_TENANTID: Tenant ID (defaults to "1")
fn get_bench_block_info() -> anyhow::Result<(Arc<dyn object_store::ObjectStore>, BlockInfo)> {
    let block_id = std::env::var("BENCH_BLOCKID").map_err(|_| {
        anyhow::anyhow!(
            "BENCH_BLOCKID is not set. These benchmarks are designed to run against a block on local disk. \
            Set BENCH_BLOCKID to the guid of the block to run benchmarks against. \
            e.g. `export BENCH_BLOCKID=030c8c4f-9d47-4916-aadc-26b90b1d2bc4`"
        )
    })?;

    let bench_path = std::env::var("BENCH_PATH").map_err(|_| {
        anyhow::anyhow!(
            "BENCH_PATH is not set. These benchmarks are designed to run against a block on local disk. \
            Set BENCH_PATH to the root of the backend such that the block to benchmark is at \
            <BENCH_PATH>/<BENCH_TENANTID>/<BENCH_BLOCKID>."
        )
    })?;

    let tenant_id = std::env::var("BENCH_TENANTID").unwrap_or_else(|_| "1".to_string());

    // Create a local filesystem object store
    let store = Arc::new(object_store::local::LocalFileSystem::new_with_prefix(
        bench_path,
    )?);

    // Create BlockInfo
    let block_info = BlockInfo::new(block_id, tenant_id);

    Ok((store, block_info))
}

fn bench_sql_queries(c: &mut Criterion) {
    use std::sync::{Arc, Mutex};

    let rt = Runtime::new().unwrap();

    // Setup context once
    let (object_store, block_info) = get_bench_block_info().unwrap_or_else(|e| {
        panic!("Failed to get block info: {}. \
               Make sure BENCH_BLOCKID, BENCH_PATH, and optionally BENCH_TENANTID environment variables are set.", e)
    });

    let ctx = Arc::new(rt.block_on(async {
        create_block_context(object_store, block_info)
            .await
            .unwrap_or_else(|e| panic!("Failed to setup context: {}", e))
    }));

    let mut group = c.benchmark_group("sql");

    // Set measurement time to get more stable results
    group.measurement_time(Duration::from_secs(60));
    group.sample_size(10);

    // Store all metrics for printing at the end
    let mut all_results: Vec<(String, usize, f64, f64, f64, usize)> = Vec::new();

    // Run benchmarks for each query
    for case in get_test_cases() {
        // Collect metrics from actual benchmark iterations
        let collected_metrics: Arc<Mutex<Vec<QueryMetrics>>> = Arc::new(Mutex::new(Vec::new()));
        let metrics_clone = collected_metrics.clone();
        let ctx_clone = ctx.clone();

        // Set throughput based on a warmup run
        let warmup_metrics = rt.block_on(async { execute(&ctx, case.sql).await.unwrap() });

        if warmup_metrics.bytes_scanned > 0 {
            group.throughput(Throughput::Bytes(warmup_metrics.bytes_scanned as u64));
        }

        group.bench_with_input(
            BenchmarkId::new("query", case.name),
            &case.sql,
            |b, sql_query| {
                let metrics_ref = metrics_clone.clone();
                let ctx_ref = ctx_clone.clone();
                b.to_async(&rt).iter(|| {
                    let metrics_capture = metrics_ref.clone();
                    let ctx_iter = ctx_ref.clone();
                    async move {
                        if let Ok(m) = execute(&ctx_iter, sql_query).await {
                            metrics_capture.lock().unwrap().push(m);
                        }
                    }
                });
            },
        );

        // Calculate aggregated metrics from actual runs and store for later printing
        let metrics_vec = collected_metrics.lock().unwrap();
        if !metrics_vec.is_empty() {
            let count = metrics_vec.len();
            let total_spans: usize = metrics_vec.iter().map(|m| m.rows_returned).sum();
            let total_bytes: usize = metrics_vec.iter().map(|m| m.bytes_scanned).sum();
            let total_nanos: u128 = metrics_vec.iter().map(|m| m.elapsed_nanos).sum();

            let avg_spans = total_spans / count;
            let avg_bytes = total_bytes / count;
            let avg_nanos: u128 = total_nanos / count as u128;

            let mb_io_per_op = (avg_bytes as f64) / (1024.0 * 1024.0);
            let throughput_mbps = if avg_nanos > 0 {
                let bytes_per_sec = (avg_bytes as f64) / (avg_nanos as f64 / 1_000_000_000.0);
                bytes_per_sec / (1024.0 * 1024.0)
            } else {
                0.0
            };

            all_results.push((
                case.name.to_string(),
                count,
                avg_nanos as f64,
                throughput_mbps,
                mb_io_per_op,
                avg_spans,
            ));
        }
    }

    group.finish();

    // Print all performance metrics together at the end
    println!("\n{:-<120}", "");
    println!("Performance Metrics Summary");
    println!("{:-<120}", "");
    for (name, count, avg_nanos, throughput_mbps, mb_io_per_op, avg_spans) in all_results {
        println!(
            "{:<40} {:>8} iterations  {:>12.0} ns/op  {:>10.2} MB/s  {:>10.2} MB_io/op  {:>10} spans/op",
            name,
            count,
            avg_nanos,
            throughput_mbps,
            mb_io_per_op,
            avg_spans
        );
    }
    println!("{:-<120}", "");
}

criterion_group!(benches, bench_sql_queries);
criterion_main!(benches);
