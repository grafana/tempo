//! Reader implementations for vParquet4 files

pub mod sync_reader;

#[cfg(feature = "async")]
pub mod async_reader;

// Re-export commonly used types
pub use sync_reader::VParquet4Reader;

#[cfg(feature = "async")]
pub use async_reader::AsyncVParquet4Reader;

use crate::error::Result;
use arrow::record_batch::RecordBatch;

/// Configuration for vParquet4 readers
#[derive(Debug, Clone)]
pub struct ReaderConfig {
    /// Number of rows to read per batch (default: 8192)
    pub batch_size: usize,

    /// Number of row groups to prefetch (default: 2)
    pub prefetch_row_groups: usize,

    /// Whether to parallelize column decoding (default: true)
    pub parallel_column_decode: bool,

    /// Whether to validate the schema on open (default: true)
    pub validate_schema: bool,
}

impl Default for ReaderConfig {
    fn default() -> Self {
        Self {
            batch_size: 8192,
            prefetch_row_groups: 2,
            parallel_column_decode: true,
            validate_schema: true,
        }
    }
}

/// Trait for vParquet4 readers
pub trait VParquet4ReaderTrait {
    /// Returns the number of row groups in the file
    fn num_row_groups(&self) -> usize;

    /// Returns the total number of rows in the file
    fn num_rows(&self) -> i64;

    /// Reads a specific row group and returns a RecordBatch
    fn read_row_group(&mut self, row_group_index: usize) -> Result<RecordBatch>;

    /// Returns metadata about the file
    fn metadata(&self) -> &parquet::file::metadata::ParquetMetaData;
}
