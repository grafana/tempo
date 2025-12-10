use async_trait::async_trait;
use config::S3Config;
use datafusion::arrow::datatypes::{DataType, Field, Schema, SchemaRef};
use datafusion::catalog::Session;
use datafusion::datasource::file_format::parquet::ParquetFormat;
use datafusion::datasource::file_format::FileFormat;
use datafusion::datasource::listing::ListingTableUrl;
use datafusion::datasource::physical_plan::FileScanConfigBuilder;
use datafusion::datasource::{TableProvider, TableType};
use datafusion::error::{DataFusionError, Result};
use datafusion::execution::context::SessionContext;
use datafusion::logical_expr::Expr;
use datafusion::physical_plan::ExecutionPlan;
use datafusion_datasource::PartitionedFile;
use std::any::Any;
use std::sync::Arc;
use storage::{create_object_store, DiscoveredBlock, TempoStorage};
use tracing::{info, warn};

/// Convert a schema to use Binary instead of Utf8 for string fields
/// This allows reading Parquet files that contain non-UTF-8 data in string columns
fn schema_utf8_to_binary(schema: &Schema) -> Schema {
    let new_fields: Vec<Field> = schema
        .fields()
        .iter()
        .map(|field| {
            let new_data_type = match field.data_type() {
                DataType::Utf8 => {
                    warn!("Converting field '{}' from Utf8 to Binary to handle non-UTF-8 data", field.name());
                    DataType::Binary
                }
                DataType::LargeUtf8 => {
                    warn!("Converting field '{}' from LargeUtf8 to LargeBinary to handle non-UTF-8 data", field.name());
                    DataType::LargeBinary
                }
                // Recursively handle nested types
                DataType::List(inner_field) => {
                    let inner_schema = Schema::new(vec![inner_field.as_ref().clone()]);
                    let converted = schema_utf8_to_binary(&inner_schema);
                    DataType::List(Arc::new(converted.field(0).clone()))
                }
                DataType::LargeList(inner_field) => {
                    let inner_schema = Schema::new(vec![inner_field.as_ref().clone()]);
                    let converted = schema_utf8_to_binary(&inner_schema);
                    DataType::LargeList(Arc::new(converted.field(0).clone()))
                }
                DataType::Struct(inner_fields) => {
                    let inner_schema = Schema::new(inner_fields.to_vec());
                    let converted = schema_utf8_to_binary(&inner_schema);
                    DataType::Struct(converted.fields().clone())
                }
                DataType::Map(inner_field, sorted) => {
                    let inner_schema = Schema::new(vec![inner_field.as_ref().clone()]);
                    let converted = schema_utf8_to_binary(&inner_schema);
                    DataType::Map(Arc::new(converted.field(0).clone()), *sorted)
                }
                other => other.clone(),
            };

            Field::new(field.name(), new_data_type, field.is_nullable())
                .with_metadata(field.metadata().clone())
        })
        .collect();

    Schema::new(new_fields).with_metadata(schema.metadata().clone())
}

/// TempoTableProvider - A custom table provider that reads parquet files
/// by directly discovering data.parquet files in S3
#[derive(Debug)]
pub struct TempoTableProvider {
    /// The schema of the table
    schema: SchemaRef,
    /// The tempo storage instance for S3 operations (kept for potential future use)
    #[allow(dead_code)]
    tempo_storage: TempoStorage,
    /// List of block information including paths, sizes, and time ranges
    blocks: Vec<DiscoveredBlock>,
    /// The parquet file format
    file_format: Arc<ParquetFormat>,
    /// Base URL for the table (e.g., s3://tempo/single-tenant)
    table_url: ListingTableUrl,
}

impl TempoTableProvider {
    /// Create a new TempoTableProvider by discovering data.parquet files in S3
    pub async fn try_new(
        state: &dyn Session,
        tempo_storage: TempoStorage,
        bucket: String,
        prefix: String,
    ) -> Result<Self> {
        // Discover blocks using the tempo storage
        let blocks = tempo_storage.discover_blocks().await?;

        info!("Discovered {} parquet files", blocks.len());
        if blocks.is_empty() {
            return Err(DataFusionError::Execution(
                "No data.parquet files found".to_string(),
            ));
        }

        // Create the table URL
        let table_url =
            ListingTableUrl::parse(format!("s3://{}/{}", bucket, prefix)).map_err(|e| {
                DataFusionError::External(format!("Failed to parse table URL: {}", e).into())
            })?;

        // Create parquet format
        let file_format = Arc::new(ParquetFormat::default());

        // Infer schema from the first parquet file
        let first_block = &blocks[0];

        // Get metadata for the first file to infer schema
        let first_file_meta = tempo_storage
            .object_store()
            .head(&first_block.path)
            .await
            .map_err(|e| {
                DataFusionError::External(
                    format!("Failed to get metadata for first parquet file: {}", e).into(),
                )
            })?;

        let inferred_schema = file_format
            .infer_schema(state, tempo_storage.object_store(), &[first_file_meta])
            .await?;

        // Convert Utf8 fields to Binary to handle non-UTF-8 data in trace attributes
        // This prevents "invalid utf-8 sequence" errors when reading Tempo parquet files
        info!("Converting schema to handle non-UTF-8 data");
        let schema = Arc::new(schema_utf8_to_binary(&inferred_schema));

        Ok(Self {
            schema,
            tempo_storage,
            blocks,
            file_format,
            table_url,
        })
    }
}

#[async_trait]
impl TableProvider for TempoTableProvider {
    fn as_any(&self) -> &dyn Any {
        self
    }

    fn schema(&self) -> SchemaRef {
        Arc::clone(&self.schema)
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
        // Get the file source from ParquetFormat
        let file_source = self.file_format.file_source();

        // Build file scan config using the builder pattern
        let mut config_builder =
            FileScanConfigBuilder::new(self.table_url.object_store(), self.schema(), file_source);

        for block in &self.blocks {
            config_builder =
                config_builder.with_file(PartitionedFile::new(block.path.to_string(), block.size));
        }

        if let Some(proj) = projection {
            config_builder = config_builder.with_projection_indices(Some(proj.clone()));
        }

        config_builder = config_builder.with_limit(limit);

        let file_scan_config = config_builder.build();

        // Use the ParquetFormat to create the physical plan
        // Note: filters are applied inside the physical plan
        let exec = self
            .file_format
            .create_physical_plan(state, file_scan_config)
            .await?;

        Ok(exec)
    }
}

/// Register the tempo table with the given session context
///
/// Parameters:
/// - ctx: The DataFusion session context
/// - config: S3 configuration containing endpoint, bucket, credentials, etc.
pub async fn register_tempo_table(ctx: &SessionContext, config: &S3Config) -> Result<()> {
    // Create the S3 object store from configuration
    let s3_store = create_object_store(config)?;

    // Register the S3 store with s3:// scheme
    let s3_url = url::Url::parse(&format!("s3://{}", config.bucket))
        .map_err(|e| DataFusionError::External(format!("Failed to parse S3 URL: {}", e).into()))?;

    ctx.register_object_store(&s3_url, s3_store.clone());

    // Create the TempoStorage instance
    let tempo_storage = TempoStorage::new(s3_store, config.prefix.clone(), config.cutoff_hours);

    // Create the TempoTableProvider
    let state = ctx.state();
    let table_provider = TempoTableProvider::try_new(
        &state,
        tempo_storage,
        config.bucket.clone(),
        config.prefix.clone(),
    )
    .await?;

    // Register the table provider as "traces"
    ctx.register_table("traces", Arc::new(table_provider))?;

    Ok(())
}
