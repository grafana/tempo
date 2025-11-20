use async_trait::async_trait;
use datafusion::arrow::datatypes::SchemaRef;
use datafusion::catalog::Session;
use datafusion::datasource::file_format::parquet::ParquetFormat;
use datafusion::datasource::listing::ListingTableUrl;
use datafusion::datasource::physical_plan::FileScanConfigBuilder;
use datafusion::datasource::{TableProvider, TableType};
use datafusion::error::{DataFusionError, Result};
use datafusion::execution::context::SessionContext;
use datafusion::logical_expr::Expr;
use datafusion::physical_plan::ExecutionPlan;
use datafusion_datasource::file_format::FileFormat;
use datafusion_datasource::PartitionedFile;
use std::any::Any;
use std::path::PathBuf;
use std::sync::Arc;
use storage::tempo_trace_schema;
use tracing::info;

/// LocalTempoTableProvider - A simplified table provider for benchmarking
/// that reads from a local Parquet file instead of S3
#[derive(Debug)]
pub struct LocalTempoTableProvider {
    /// The schema of the table
    schema: SchemaRef,
    /// Path to the local data.parquet file
    file_path: String,
    /// The parquet file format
    file_format: Arc<ParquetFormat>,
    /// Base URL for the table (local file path)
    table_url: ListingTableUrl,
}

impl LocalTempoTableProvider {
    /// Create a new LocalTempoTableProvider from a local file path
    pub async fn try_new(file_path: String) -> Result<Self> {
        // Validate file exists
        let path = PathBuf::from(&file_path);
        if !path.exists() {
            return Err(DataFusionError::Execution(format!(
                "Parquet file not found: {}",
                file_path
            )));
        }

        info!("Loading Parquet file from: {}", file_path);

        // Use the Tempo trace schema from storage crate
        let schema = Arc::new(tempo_trace_schema());

        // Create parquet format
        let file_format = Arc::new(ParquetFormat::default());

        // Convert file path to URL format
        let table_url: ListingTableUrl = ListingTableUrl::parse(format!("file://{}", file_path))?;

        Ok(Self {
            schema,
            file_path,
            file_format,
            table_url,
        })
    }

}

#[async_trait]
impl TableProvider for LocalTempoTableProvider {
    fn as_any(&self) -> &dyn Any {
        self
    }

    fn schema(&self) -> SchemaRef {
        self.schema.clone()
    }

    fn table_type(&self) -> TableType {
        TableType::Base
    }

    async fn scan(
        &self,
        state: &dyn Session,
        projection: Option<&Vec<usize>>,
        _filters: &[Expr],
        limit: Option<usize>,
    ) -> Result<Arc<dyn ExecutionPlan>> {
        // Get file statistics
        let metadata = std::fs::metadata(&self.file_path).map_err(|e| {
            DataFusionError::Execution(format!("Failed to read file metadata: {}", e))
        })?;
        let file_size = metadata.len();

        // Create a partitioned file entry
        let partitioned_file = PartitionedFile::new(&self.file_path, file_size);

        // Get the file source from ParquetFormat
        let file_source = self.file_format.file_source();

        // Build file scan config using the builder pattern
        let mut config_builder =
            FileScanConfigBuilder::new(self.table_url.object_store(), self.schema.clone(), file_source)
                .with_file(partitioned_file);

        if let Some(proj) = projection {
            config_builder = config_builder.with_projection(Some(proj.clone()));
        }

        config_builder = config_builder.with_limit(limit);

        let file_scan_config = config_builder.build();

        // Create the physical plan using the ParquetFormat
        self.file_format
            .create_physical_plan(state, file_scan_config)
            .await
    }
}

/// Register the local Tempo table in a DataFusion context for benchmarking
pub async fn register_local_tempo_table(ctx: &SessionContext, file_path: String) -> Result<()> {
    let provider = LocalTempoTableProvider::try_new(file_path).await?;
    ctx.register_table("traces", Arc::new(provider))?;
    info!("Registered 'traces' table from local file");
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_try_new_missing_file() {
        let result = LocalTempoTableProvider::try_new("/nonexistent/file.parquet".to_string()).await;
        assert!(result.is_err());
        assert!(result
            .unwrap_err()
            .to_string()
            .contains("not found"));
    }
}
