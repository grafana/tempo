//! Asynchronous reader for vParquet4 files
//!
//! This module provides async readers for reading vParquet4 files from
//! object stores (S3, GCS, Azure, local filesystem, etc.) using the `object_store` crate.

use crate::error::{Result, VParquet4Error};
use crate::reader::{ReaderConfig, VParquet4ReaderTrait};
use crate::schema::validation::validate_schema;
use arrow::record_batch::RecordBatch;
use object_store::path::Path as ObjectPath;
use object_store::ObjectStore;
use parquet::arrow::async_reader::ParquetObjectReader;
use parquet::arrow::ParquetRecordBatchStreamBuilder;
use parquet::arrow::ProjectionMask;
use parquet::file::metadata::ParquetMetaData;
use std::sync::Arc;

/// Asynchronous reader for vParquet4 files from object stores
///
/// This reader supports reading from S3, GCS, Azure, local filesystem, and other
/// backends supported by the `object_store` crate.
///
/// # Example
///
/// ```rust,no_run
/// use object_store::local::LocalFileSystem;
/// use object_store::path::Path;
/// use vparquet4::{AsyncVParquet4Reader, ReaderConfig};
///
/// # async fn example() -> Result<(), Box<dyn std::error::Error>> {
/// let store = Arc::new(LocalFileSystem::new());
/// let path = Path::from("path/to/data.parquet");
///
/// let reader = AsyncVParquet4Reader::new(store, path, ReaderConfig::default()).await?;
/// println!("Total rows: {}", reader.num_rows());
/// # Ok(())
/// # }
/// ```
pub struct AsyncVParquet4Reader {
    /// Object store for reading the file
    store: Arc<dyn ObjectStore>,

    /// Path to the file in the object store
    path: ObjectPath,

    /// Parquet file metadata
    metadata: Arc<ParquetMetaData>,

    /// Reader configuration
    config: ReaderConfig,

    /// Optional projection mask for column pruning
    projection: Option<ProjectionMask>,
}

impl AsyncVParquet4Reader {
    /// Creates a new async reader for a file in an object store
    pub async fn new(
        store: Arc<dyn ObjectStore>,
        path: ObjectPath,
        config: ReaderConfig,
    ) -> Result<Self> {
        // Create a ParquetObjectReader to read metadata
        let reader = ParquetObjectReader::new(store.clone(), path.clone());

        // Get metadata using a builder
        let builder = ParquetRecordBatchStreamBuilder::new(reader).await?;

        let metadata = builder.metadata().clone();

        // Validate schema if requested
        if config.validate_schema {
            let arrow_schema = builder.schema();
            validate_schema(&arrow_schema)?;
        }

        Ok(Self {
            store,
            path,
            metadata: Arc::new(metadata.as_ref().clone()),
            config,
            projection: None,
        })
    }

    /// Sets a projection mask for column pruning
    pub fn with_projection(mut self, projection: ProjectionMask) -> Self {
        self.projection = Some(projection);
        self
    }

    /// Creates a streaming reader that yields batches asynchronously
    pub async fn into_stream(
        self,
    ) -> Result<impl futures_util::Stream<Item = Result<RecordBatch>>> {
        let reader = ParquetObjectReader::new(self.store.clone(), self.path.clone());

        let mut builder = ParquetRecordBatchStreamBuilder::new(reader).await?;

        // Apply batch size
        builder = builder.with_batch_size(self.config.batch_size);

        // Apply projection if set
        if let Some(projection) = self.projection {
            builder = builder.with_projection(projection);
        }

        let stream = builder.build()?;

        // Convert parquet errors to our error type
        use futures_util::stream::StreamExt;
        Ok(stream.map(|result| result.map_err(|e| VParquet4Error::Parquet(e))))
    }

    /// Reads a specific row group asynchronously
    pub async fn read_row_group(&self, row_group_index: usize) -> Result<RecordBatch> {
        if row_group_index >= self.num_row_groups() {
            return Err(VParquet4Error::Generic(format!(
                "Row group index {} out of bounds (total: {})",
                row_group_index,
                self.num_row_groups()
            )));
        }

        let reader = ParquetObjectReader::new(self.store.clone(), self.path.clone());

        let mut builder = ParquetRecordBatchStreamBuilder::new(reader).await?;

        // Apply batch size
        builder = builder.with_batch_size(self.config.batch_size);

        // Apply projection if set
        if let Some(ref projection) = self.projection {
            builder = builder.with_projection(projection.clone());
        }

        // Select only the requested row group
        builder = builder.with_row_groups(vec![row_group_index]);

        // Build stream and collect all batches
        let stream = builder.build()?;

        use futures_util::stream::StreamExt;

        let mut batches = Vec::new();
        let mut stream = Box::pin(stream);
        while let Some(result) = stream.next().await {
            batches.push(result?);
        }

        // Combine batches if there are multiple
        if batches.is_empty() {
            return Err(VParquet4Error::Generic(
                "No batches read from row group".to_string(),
            ));
        } else if batches.len() == 1 {
            Ok(batches.into_iter().next().unwrap())
        } else {
            // Concatenate multiple batches
            let schema = batches[0].schema();
            let columns: Vec<_> = (0..schema.fields().len())
                .map(|i| {
                    let arrays: Vec<_> = batches.iter().map(|batch| batch.column(i).clone()).collect();
                    arrow::compute::concat(&arrays.iter().map(|a| a.as_ref()).collect::<Vec<_>>())
                        .expect("Failed to concatenate arrays")
                })
                .collect();

            Ok(RecordBatch::try_new(schema, columns)?)
        }
    }

    /// Returns the Arrow schema for the file
    pub async fn arrow_schema(&self) -> Result<arrow::datatypes::SchemaRef> {
        let reader = ParquetObjectReader::new(self.store.clone(), self.path.clone());

        let builder = ParquetRecordBatchStreamBuilder::new(reader).await?;

        Ok(builder.schema().clone())
    }
}

impl VParquet4ReaderTrait for AsyncVParquet4Reader {
    fn num_row_groups(&self) -> usize {
        self.metadata.num_row_groups()
    }

    fn num_rows(&self) -> i64 {
        self.metadata
            .row_groups()
            .iter()
            .map(|rg| rg.num_rows())
            .sum()
    }

    fn read_row_group(&mut self, _row_group_index: usize) -> Result<RecordBatch> {
        // This sync method cannot be implemented for async readers
        // Users should use the async read_row_group method instead
        Err(VParquet4Error::Generic(
            "read_row_group is not available for AsyncVParquet4Reader. Use the async version instead.".to_string()
        ))
    }

    fn metadata(&self) -> &ParquetMetaData {
        &self.metadata
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_async_reader_local_file() {
        use object_store::local::LocalFileSystem;
        use std::env;

        // Test with the Go test data file - convert to absolute path
        let test_data_path = "../../tempodb/encoding/vparquet4/test-data/single-tenant/b27b0e53-66a0-4505-afd6-434ae3cd4a10";
        let file_name = "data.parquet";
        let file_path = format!("{}/{}", test_data_path, file_name);

        // Check if test file exists
        if !std::path::Path::new(&file_path).exists() {
            println!("Test data not found at {}, skipping test", file_path);
            return;
        }

        // Get absolute path for the directory
        let abs_dir_path = env::current_dir()
            .unwrap()
            .join(test_data_path)
            .canonicalize()
            .expect("Failed to canonicalize path");

        let store = Arc::new(LocalFileSystem::new_with_prefix(abs_dir_path).unwrap());
        let path = ObjectPath::from(file_name);

        let reader = AsyncVParquet4Reader::new(store, path, ReaderConfig::default())
            .await
            .expect("Failed to create async reader");

        assert!(reader.num_rows() > 0);
        assert!(reader.num_row_groups() > 0);
    }

    #[tokio::test]
    async fn test_async_read_row_group() {
        use object_store::local::LocalFileSystem;
        use std::env;

        let test_data_path = "../../tempodb/encoding/vparquet4/test-data/single-tenant/b27b0e53-66a0-4505-afd6-434ae3cd4a10";
        let file_name = "data.parquet";
        let file_path = format!("{}/{}", test_data_path, file_name);

        if !std::path::Path::new(&file_path).exists() {
            println!("Test data not found, skipping test");
            return;
        }

        // Get absolute path for the directory
        let abs_dir_path = env::current_dir()
            .unwrap()
            .join(test_data_path)
            .canonicalize()
            .expect("Failed to canonicalize path");

        let store = Arc::new(LocalFileSystem::new_with_prefix(abs_dir_path).unwrap());
        let path = ObjectPath::from(file_name);

        let reader = AsyncVParquet4Reader::new(store, path, ReaderConfig::default())
            .await
            .expect("Failed to create async reader");

        let batch = reader
            .read_row_group(0)
            .await
            .expect("Failed to read row group");

        assert!(batch.num_rows() > 0);
    }
}
