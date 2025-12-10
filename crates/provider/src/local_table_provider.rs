use async_trait::async_trait;
use datafusion::arrow::datatypes::SchemaRef;
use datafusion::catalog::Session;
use datafusion::datasource::{TableProvider, TableType};
use datafusion::error::{DataFusionError, Result};
use datafusion::execution::context::SessionContext;
use datafusion::logical_expr::{Expr, TableProviderFilterPushDown};
use datafusion::physical_plan::ExecutionPlan;
use std::any::Any;
use std::path::PathBuf;
use std::sync::Arc;
use tracing::info;

use crate::filter::extract_span_filters;
use crate::spanset_converter::flat_span_schema;
use crate::vparquet4_exec::VParquet4Exec;

/// LocalTempoTableProvider - Uses VParquet4Reader for optimized reading
/// with filter pushdown support
#[derive(Debug)]
pub struct LocalTempoTableProvider {
    /// The schema of the table (flat span schema)
    schema: SchemaRef,
    /// Path to the local data.parquet file
    file_path: PathBuf,
}

impl LocalTempoTableProvider {
    /// Create a new LocalTempoTableProvider from a local file path
    pub async fn try_new(file_path: String) -> Result<Self> {
        let path = PathBuf::from(&file_path);
        if !path.exists() {
            return Err(DataFusionError::Execution(format!(
                "Parquet file not found: {}",
                file_path
            )));
        }

        let metadata = std::fs::metadata(&file_path).map_err(|e| {
            DataFusionError::Execution(format!("Failed to read file metadata: {}", e))
        })?;
        let file_size = metadata.len();

        info!(
            "Loading Parquet file from: {} (size: {} bytes)",
            file_path, file_size
        );

        // Use flat span schema for query results
        let schema = flat_span_schema();

        Ok(Self {
            schema,
            file_path: path,
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

    /// Indicate which filters can be pushed down to VParquet4Reader
    fn supports_filters_pushdown(
        &self,
        filters: &[&Expr],
    ) -> Result<Vec<TableProviderFilterPushDown>> {
        filters
            .iter()
            .map(|filter| {
                if crate::filter::expr_to_span_filter(filter).is_some() {
                    Ok(TableProviderFilterPushDown::Exact)
                } else {
                    Ok(TableProviderFilterPushDown::Unsupported)
                }
            })
            .collect()
    }

    async fn scan(
        &self,
        _state: &dyn Session,
        projection: Option<&Vec<usize>>,
        filters: &[Expr],
        limit: Option<usize>,
    ) -> Result<Arc<dyn ExecutionPlan>> {
        // Convert DataFusion filters to SpanFilter
        let span_filter = extract_span_filters(filters);

        if let Some(ref filter) = span_filter {
            info!("Pushing down filter to VParquet4Reader: {:?}", filter);
        }

        let exec = VParquet4Exec::new(
            self.file_path.clone(),
            span_filter,
            projection.cloned(),
            limit,
        );

        Ok(Arc::new(exec))
    }
}

/// Register the local Tempo table in a DataFusion context
pub async fn register_local_tempo_table(ctx: &SessionContext, file_path: String) -> Result<()> {
    let provider = LocalTempoTableProvider::try_new(file_path).await?;
    ctx.register_table("spans", Arc::new(provider))?;
    info!("Registered 'spans' table from local file with VParquet4Reader");
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_try_new_missing_file() {
        let result =
            LocalTempoTableProvider::try_new("/nonexistent/file.parquet".to_string()).await;
        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("not found"));
    }

    #[test]
    fn test_schema_is_flat() {
        let schema = flat_span_schema();
        assert!(schema.field_with_name("trace_id").is_ok());
        assert!(schema.field_with_name("name").is_ok());
        assert!(schema.field_with_name("span_id").is_ok());
        assert_eq!(schema.fields().len(), 11);
    }

    #[test]
    fn test_supports_filters_pushdown_name_equals() {
        use datafusion::logical_expr::{col, lit};

        let schema = flat_span_schema();
        let provider = LocalTempoTableProvider {
            schema,
            file_path: PathBuf::from("test.parquet"),
        };

        let filter = col("name").eq(lit("test_span"));
        let result = provider.supports_filters_pushdown(&[&filter]).unwrap();

        assert_eq!(result.len(), 1);
        assert!(matches!(
            result[0],
            TableProviderFilterPushDown::Exact
        ));
    }

    #[test]
    fn test_supports_filters_pushdown_unsupported() {
        use datafusion::logical_expr::{col, lit};

        let schema = flat_span_schema();
        let provider = LocalTempoTableProvider {
            schema,
            file_path: PathBuf::from("test.parquet"),
        };

        // status_code filter is not (yet) supported
        let filter = col("status_code").eq(lit(1i64));
        let result = provider.supports_filters_pushdown(&[&filter]).unwrap();

        assert_eq!(result.len(), 1);
        assert!(matches!(
            result[0],
            TableProviderFilterPushDown::Unsupported
        ));
    }

    #[test]
    fn test_supports_filters_pushdown_multiple() {
        use datafusion::logical_expr::{col, lit};

        let schema = flat_span_schema();
        let provider = LocalTempoTableProvider {
            schema,
            file_path: PathBuf::from("test.parquet"),
        };

        let filter1 = col("name").eq(lit("test"));
        let filter2 = col("status_code").eq(lit(1i64));
        let result = provider
            .supports_filters_pushdown(&[&filter1, &filter2])
            .unwrap();

        assert_eq!(result.len(), 2);
        assert!(matches!(
            result[0],
            TableProviderFilterPushDown::Exact
        ));
        assert!(matches!(
            result[1],
            TableProviderFilterPushDown::Unsupported
        ));
    }
}
