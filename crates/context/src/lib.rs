use config::Config;
use datafusion::arrow::record_batch::RecordBatch;
use datafusion::error::{DataFusionError, Result};
use datafusion::physical_plan::ExecutionPlan;
use datafusion::prelude::*;
use datafusion_physical_plan::metrics::MetricsSet;
use object_store::ObjectStore;
use provider::{create_flattened_view, display, register_tempo_table, register_udfs};
use std::collections::HashMap;
use std::sync::Arc;
use storage::BlockInfo;
use tracing::{debug, info};

/// Create a new DataFusion session context
///
/// Parameters:
/// - config_file: Optional path to TOML configuration file. If None, uses environment variables.
pub async fn create_context(config_file: Option<&str>) -> Result<SessionContext> {
    // Load configuration from file or environment variables first
    let app_config = Config::load(config_file).map_err(|e| {
        DataFusionError::External(format!("Failed to load configuration: {}", e).into())
    })?;

    if let Some(file) = config_file {
        info!("Loaded configuration from: {}", file);
    } else {
        info!("Loaded configuration from environment variables");
    }

    // Validate the configuration - fail if invalid
    app_config
        .validate()
        .map_err(|e| DataFusionError::External(format!("Invalid configuration: {}", e).into()))?;

    info!("Using S3 configuration:");
    info!("  Endpoint: {}", app_config.s3.endpoint);
    info!("  Bucket: {}", app_config.s3.bucket);
    info!("  Prefix: {}", app_config.s3.prefix);
    info!("  Region: {}", app_config.s3.region);
    info!(
        "  Pool Max Idle Per Host: {}",
        app_config.s3.pool_max_idle_per_host
    );
    info!(
        "  Pool Idle Timeout: {}s",
        app_config.s3.pool_idle_timeout_secs
    );
    info!("  Cutoff Hours: {}", app_config.s3.cutoff_hours);

    info!("Using DataFusion configuration:");
    info!(
        "  Parquet Pruning: {}",
        app_config.datafusion.parquet_pruning
    );

    // Determine optimal parallelism based on CPU count
    let cpu_count = std::thread::available_parallelism()
        .map(|n| n.get())
        .unwrap_or(8);

    // Configure session with optimizations
    let mut session_config = SessionConfig::default()
        .with_information_schema(true)
        .with_target_partitions(cpu_count)
        .with_repartition_file_scans(true)
        .with_repartition_joins(true)
        .with_repartition_aggregations(true)
        .with_parquet_pruning(app_config.datafusion.parquet_pruning);

    // Disable identifier normalization to preserve case sensitivity
    session_config =
        session_config.set_bool("datafusion.sql_parser.enable_ident_normalization", false);

    let ctx = SessionContext::new_with_config(session_config);

    // Register UDFs
    register_udfs(&ctx);

    // Register tempo table - fail if registration fails
    register_tempo_table(&ctx, &app_config.s3).await?;
    info!("Tempo table registered successfully");

    // Create the flattened spans view - fail if creation fails
    info!("Setting up views...");
    create_flattened_view(&ctx).await?;
    info!("Views created successfully");

    Ok(ctx)
}

/// Collect metrics from a physical plan by walking the execution tree
///
/// Returns a HashMap of metric name to value (as strings for flexibility)
pub fn collect_plan_metrics(physical_plan: Arc<dyn ExecutionPlan>) -> HashMap<String, String> {
    use std::collections::VecDeque;

    let mut metric_set = MetricsSet::new();

    // Walk the physical plan tree using BFS to collect metrics from all nodes
    let mut queue: VecDeque<Arc<dyn ExecutionPlan>> = VecDeque::new();
    queue.push_back(physical_plan);

    while let Some(plan_node) = queue.pop_front() {
        // Add all children to the queue
        for child in plan_node.children() {
            queue.push_back(child.clone());
        }

        // Extract metrics from this node
        if let Some(metrics) = plan_node.metrics() {
            // Iterate through all metrics to extract partition-level information
            for metric in metrics.iter() {
                metric_set.push(metric.clone());
            }
        }
    }
    let aggregated = metric_set.aggregate_by_name().sorted_for_display();
    let mut result = HashMap::new();
    for metric in aggregated.iter() {
        let name = metric.value().name().to_string();
        let value = metric.value().as_usize();
        result.insert(name.clone(), value.to_string());
    }
    result
}

/// Execute a SQL or TraceQL query and return the results as formatted string with performance statistics
pub async fn execute_query(ctx: &SessionContext, query: &str) -> Result<String> {
    use datafusion::physical_plan::collect;
    use std::time::Instant;

    // Start timing
    let start = Instant::now();

    // Detect if this is a TraceQL query (starts with |)
    let sql = if query.trim_start().starts_with('|') {
        // Strip the leading pipe and process as TraceQL
        let traceql_query = query.trim_start().strip_prefix('|').unwrap().trim();

        info!("Detected TraceQL query: {}", traceql_query);

        // Convert TraceQL to SQL
        let converted_sql = traceql::traceql_to_sql_string(traceql_query).map_err(|e| {
            DataFusionError::External(format!("TraceQL conversion failed: {}", e).into())
        })?;

        debug!("Converted to SQL: {}", converted_sql);

        converted_sql
    } else {
        // Regular SQL query
        query.to_string()
    };

    // Execute the query
    let df = ctx.sql(&sql).await?;

    // Get the physical plan for execution
    let physical_plan = df.create_physical_plan().await?;
    let task_ctx = ctx.task_ctx();

    // Execute and collect results through the physical plan to populate metrics
    let results: Vec<RecordBatch> = collect(physical_plan.clone(), task_ctx).await?;

    // Format results as a pretty table with terminal width awareness
    let formatted = display::format_batches(&results)?;

    // Collect metrics from the physical plan
    let metrics = collect_plan_metrics(physical_plan);

    for (key, value) in metrics.iter() {
        debug!("{}: {}", key, value);
    }

    // Calculate and log total execution time
    let elapsed = start.elapsed();
    debug!("Query execution time: {:.3} seconds", elapsed.as_secs_f64());

    Ok(formatted)
}

/// Create a DataFusion context for a specific block
///
/// Parameters:
/// - object_store: The object store containing the block's parquet file
/// - block_info: Information about the block (block_id and tenant_id)
///
/// Returns:
/// - A configured SessionContext with the block's data registered as "traces" table
pub async fn create_block_context(
    object_store: Arc<dyn ObjectStore>,
    block_info: BlockInfo,
) -> Result<SessionContext> {
    info!(
        "Creating context for block {} in tenant {}",
        block_info.block_id, block_info.tenant_id
    );

    // Create a new session context
    let ctx = SessionContext::new();

    // Register UDFs
    register_udfs(&ctx);

    // Get the path to the parquet file
    let parquet_path = block_info.data_parquet_path();
    info!("Registering parquet file: {}", parquet_path);

    // Register the object store with a URL scheme
    let store_url = url::Url::parse("tempo://bucket").map_err(|e| {
        DataFusionError::External(format!("Failed to parse store URL: {}", e).into())
    })?;
    ctx.register_object_store(&store_url, object_store);

    // Register the parquet file as the "traces" table
    let full_path = format!("tempo://bucket/{}", parquet_path);
    let options = ParquetReadOptions::default()
        .parquet_pruning(true)
        .skip_metadata(false);

    //let schema = tempo_trace_schema();
    //options.schema = Some(&schema);
    ctx.register_parquet("traces", &full_path, options).await?;
    info!("Registered 'traces' table for block");

    // Create the flattened spans view
    create_flattened_view(&ctx).await?;
    info!("Created flattened view");

    Ok(ctx)
}
