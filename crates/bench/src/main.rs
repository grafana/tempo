use anyhow::{Context as _, Result};
use clap::Parser;
use context::{collect_plan_metrics, create_block_context};
use datafusion::execution::context::SessionContext;
use datafusion::physical_plan::execute_stream;
use datafusion::sql::parser::DFParser;
use datafusion::sql::sqlparser::dialect::GenericDialect;
use futures_util::StreamExt;
use std::sync::Arc;
use std::time::Instant;
use storage::BlockInfo;
use traceql::traceql_to_sql_string;

#[derive(Parser, Debug)]
#[command(name = "bench")]
#[command(about = "Execute TraceQL or SQL queries against Tempo blocks", long_about = None)]
struct Args {
    /// TraceQL query to execute
    #[arg(long, conflicts_with = "sql")]
    traceql: Option<String>,

    /// SQL query to execute directly
    #[arg(long, conflicts_with = "traceql")]
    sql: Option<String>,
}

/// Metrics collected from query execution
#[derive(Debug)]
struct QueryMetrics {
    rows_returned: usize,
    bytes_scanned: usize,
    elapsed_nanos: u128,
}

impl QueryMetrics {
    /// Calculate throughput in MB/s
    fn throughput_mbps(&self) -> f64 {
        if self.elapsed_nanos == 0 {
            return 0.0;
        }
        let bytes_per_sec =
            (self.bytes_scanned as f64) / (self.elapsed_nanos as f64 / 1_000_000_000.0);
        bytes_per_sec / (1024.0 * 1024.0)
    }

    /// Calculate MB of I/O per operation
    fn mb_io_per_op(&self) -> f64 {
        (self.bytes_scanned as f64) / (1024.0 * 1024.0)
    }
}

/// Execute a SQL query and return metrics
async fn execute(ctx: &SessionContext, sql: &str) -> Result<QueryMetrics> {
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

    // Execute the physical plan with streaming (no in-memory collection)
    let task_ctx = ctx.task_ctx();

    let start = Instant::now();
    let mut stream = execute_stream(physical_plan.clone(), task_ctx)?;

    // Stream through results, counting rows without keeping batches in memory
    let mut rows_returned: usize = 0;
    while let Some(batch_result) = stream.next().await {
        let batch = batch_result?;
        rows_returned += batch.num_rows();
        // Batch is dropped here, not kept in memory
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

/// Get block info and object store from environment variables
/// - BENCH_BLOCKID: GUID of the block (e.g., "030c8c4f-9d47-4916-aadc-26b90b1d2bc4")
/// - BENCH_PATH: Root path to backend (e.g., "/path/to/tempo/storage")
/// - BENCH_TENANTID: Tenant ID (defaults to "1")
fn get_bench_block_info() -> Result<(Arc<dyn object_store::ObjectStore>, BlockInfo)> {
    let block_id = std::env::var("BENCH_BLOCKID").context(
        "BENCH_BLOCKID is not set. Set BENCH_BLOCKID to the guid of the block to run against. \
         e.g. `export BENCH_BLOCKID=030c8c4f-9d47-4916-aadc-26b90b1d2bc4`",
    )?;

    let bench_path = std::env::var("BENCH_PATH").context(
        "BENCH_PATH is not set. Set BENCH_PATH to the root of the backend such that the block \
         to benchmark is at <BENCH_PATH>/<BENCH_TENANTID>/<BENCH_BLOCKID>.",
    )?;

    let tenant_id = std::env::var("BENCH_TENANTID").unwrap_or_else(|_| "1".to_string());

    // Create a local filesystem object store
    let store = Arc::new(object_store::local::LocalFileSystem::new_with_prefix(
        bench_path,
    )?);

    // Create BlockInfo
    let block_info = BlockInfo::new(block_id, tenant_id);

    Ok((store, block_info))
}

#[tokio::main]
async fn main() -> Result<()> {
    let args = Args::parse();

    // Validate that at least one query type is provided
    let sql = match (&args.traceql, &args.sql) {
        (Some(traceql), None) => {
            println!("Converting TraceQL to SQL...");
            let sql = traceql_to_sql_string(traceql)?;
            println!("SQL: {}\n", sql);
            sql
        }
        (None, Some(sql)) => {
            println!("Using SQL query directly...");
            println!("SQL: {}\n", sql);
            sql.clone()
        }
        (None, None) => {
            anyhow::bail!("Either --traceql or --sql must be provided");
        }
        (Some(_), Some(_)) => {
            unreachable!("clap should prevent both flags from being set");
        }
    };

    // Get block info and object store from environment
    let (object_store, block_info) = get_bench_block_info()?;

    // Create context
    println!("Setting up context...");
    let ctx = create_block_context(object_store, block_info).await?;

    // Execute query
    println!("Executing query...");
    let metrics = execute(&ctx, &sql).await?;

    // Print results
    println!("\nQuery Results:");
    println!("  Rows returned:  {}", metrics.rows_returned);
    println!(
        "  Bytes scanned:  {} ({:.2} MB)",
        metrics.bytes_scanned,
        metrics.mb_io_per_op()
    );
    println!(
        "  Elapsed time:   {:.2} ms",
        metrics.elapsed_nanos as f64 / 1_000_000.0
    );
    println!("  Throughput:     {:.2} MB/s", metrics.throughput_mbps());

    Ok(())
}
