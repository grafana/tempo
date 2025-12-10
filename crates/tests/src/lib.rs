use context::create_block_context;
use datafusion::execution::context::SessionContext;
use provider::{create_flattened_view, register_local_tempo_table, register_udfs};
use std::path::PathBuf;
use std::sync::Arc;
use storage::BlockInfo;

/// Setup DataFusion context with a local Tempo parquet file
pub async fn setup_context_with_file(
    file_path: impl Into<String>,
) -> anyhow::Result<SessionContext> {
    let ctx = SessionContext::new();

    // Register UDFs
    register_udfs(&ctx);

    // Register local table with the file path
    register_local_tempo_table(&ctx, file_path.into()).await?;

    // Create the flattened spans view
    create_flattened_view(&ctx).await?;

    Ok(ctx)
}

/// Execute a SQL query and return the number of rows
pub async fn execute_query(ctx: &SessionContext, sql: &str) -> anyhow::Result<usize> {
    let df = ctx.sql(sql).await?;
    let results = df.collect().await?;

    let rows_returned: usize = results.iter().map(|batch| batch.num_rows()).sum();

    Ok(rows_returned)
}

/// Get the path to test data file
pub fn get_test_data_path(block_id: &str) -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .parent()
        .unwrap()
        .parent()
        .unwrap()
        .join("tempodb/encoding/vparquet4/test-data/single-tenant")
        .join(block_id)
        .join("data.parquet")
}

/// Setup DataFusion context with a block using object store
///
/// Parameters:
/// - block_id: The block ID (GUID)
/// - tenant_id: The tenant ID (defaults to "single-tenant")
pub async fn setup_context_with_block(
    block_id: impl Into<String>,
    tenant_id: impl Into<String>,
) -> anyhow::Result<SessionContext> {
    let block_id_str = block_id.into();
    let tenant_id_str = tenant_id.into();

    // Get the base path to test data
    let base_path = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .parent()
        .unwrap()
        .parent()
        .unwrap()
        .join("tempodb/encoding/vparquet4/test-data");

    // Create a local filesystem object store
    let store = Arc::new(object_store::local::LocalFileSystem::new_with_prefix(
        base_path,
    )?);

    // Create BlockInfo
    let block_info = BlockInfo::new(block_id_str, tenant_id_str);

    // Create context using the new function
    let ctx = create_block_context(store, block_info).await?;

    Ok(ctx)
}

/// Get list of TraceQL query files
pub fn get_traceql_query_files() -> anyhow::Result<Vec<(String, PathBuf)>> {
    let queries_dir = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .parent()
        .unwrap()
        .join("traceql/queries");

    let mut query_files = Vec::new();

    for entry in std::fs::read_dir(&queries_dir)? {
        let entry = entry?;
        let path = entry.path();

        if path.extension().and_then(|s| s.to_str()) == Some("tql") {
            let name = path
                .file_stem()
                .and_then(|s| s.to_str())
                .unwrap_or("unknown")
                .to_string();
            query_files.push((name, path));
        }
    }

    query_files.sort_by(|a, b| a.0.cmp(&b.0));
    Ok(query_files)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_query_files_exist() {
        let files = get_traceql_query_files().unwrap();
        assert!(!files.is_empty(), "Should find TraceQL query files");
    }
}
