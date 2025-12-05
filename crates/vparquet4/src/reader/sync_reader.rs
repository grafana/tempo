//! Synchronous reader for vParquet4 files

use crate::error::{Result, VParquet4Error};
use crate::reader::{ReaderConfig, VParquet4ReaderTrait};
use crate::schema::validation::validate_schema;
use arrow::record_batch::RecordBatch;
use parquet::arrow::arrow_reader::{ParquetRecordBatchReader, ParquetRecordBatchReaderBuilder};
use parquet::arrow::ProjectionMask;
use parquet::file::metadata::ParquetMetaData;
use std::fs::File;
use std::path::Path;
use std::sync::Arc;

/// Synchronous reader for vParquet4 files
pub struct VParquet4Reader {
    /// Parquet file metadata
    metadata: Arc<ParquetMetaData>,

    /// Reader configuration
    config: ReaderConfig,

    /// File handle
    file: File,

    /// Optional projection mask for column pruning
    projection: Option<ProjectionMask>,
}

impl VParquet4Reader {
    /// Opens a vParquet4 file for reading
    pub fn open<P: AsRef<Path>>(path: P, config: ReaderConfig) -> Result<Self> {
        let file = File::open(path.as_ref())?;
        Self::from_file(file, config)
    }

    /// Creates a reader from an existing file handle
    pub fn from_file(file: File, config: ReaderConfig) -> Result<Self> {
        // Build a reader to get metadata
        let builder = ParquetRecordBatchReaderBuilder::try_new(file.try_clone()?)?;
        let metadata = builder.metadata().clone();

        // Validate schema if requested
        if config.validate_schema {
            let arrow_schema = builder.schema();
            validate_schema(&arrow_schema)?;
        }

        Ok(Self {
            metadata: Arc::new(metadata.as_ref().clone()),
            config,
            file,
            projection: None,
        })
    }

    /// Sets a projection mask for column pruning
    pub fn with_projection(mut self, projection: ProjectionMask) -> Self {
        self.projection = Some(projection);
        self
    }

    /// Reads all row groups and returns an iterator over record batches
    pub fn read_all(&mut self) -> Result<impl Iterator<Item = Result<RecordBatch>> + '_> {
        let mut batches = Vec::new();
        for i in 0..self.num_row_groups() {
            batches.push(self.read_row_group(i)?);
        }
        Ok(batches.into_iter().map(Ok))
    }

    /// Creates a streaming reader that yields batches
    pub fn into_stream(self) -> Result<ParquetRecordBatchReader> {
        let mut builder = ParquetRecordBatchReaderBuilder::try_new(self.file)?;

        // Apply batch size
        builder = builder.with_batch_size(self.config.batch_size);

        // Apply projection if set
        if let Some(projection) = self.projection {
            builder = builder.with_projection(projection);
        }

        Ok(builder.build()?)
    }

    /// Returns the Arrow schema for the file
    pub fn arrow_schema(&self) -> Result<arrow::datatypes::SchemaRef> {
        let file = self.file.try_clone()?;
        let builder = ParquetRecordBatchReaderBuilder::try_new(file)?;
        Ok(builder.schema().clone())
    }
}

impl VParquet4ReaderTrait for VParquet4Reader {
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

    fn read_row_group(&mut self, row_group_index: usize) -> Result<RecordBatch> {
        if row_group_index >= self.num_row_groups() {
            return Err(VParquet4Error::Generic(format!(
                "Row group index {} out of bounds (total: {})",
                row_group_index,
                self.num_row_groups()
            )));
        }

        // Create a new builder for reading this specific row group
        let file = self.file.try_clone()?;
        let mut builder = ParquetRecordBatchReaderBuilder::try_new(file)?;

        // Apply batch size
        builder = builder.with_batch_size(self.config.batch_size);

        // Apply projection if set
        if let Some(ref projection) = self.projection {
            builder = builder.with_projection(projection.clone());
        }

        // Select only the requested row group
        builder = builder.with_row_groups(vec![row_group_index]);

        // Build reader and collect all batches
        let mut reader = builder.build()?;

        // Read all batches from this row group
        let mut batches = Vec::new();
        while let Some(batch) = reader.next() {
            batches.push(batch?);
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

    fn metadata(&self) -> &ParquetMetaData {
        &self.metadata
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    // Note: Tests will be added once we have test data access
    // For now, we'll test the basic structure

    #[test]
    fn test_reader_config_default() {
        let config = ReaderConfig::default();
        assert_eq!(config.batch_size, 8192);
        assert_eq!(config.prefetch_row_groups, 2);
        assert!(config.parallel_column_decode);
        assert!(config.validate_schema);
    }
}
