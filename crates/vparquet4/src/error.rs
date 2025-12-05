//! Error types for vParquet4 reader operations

use thiserror::Error;

/// Result type for vParquet4 operations
pub type Result<T> = std::result::Result<T, VParquet4Error>;

/// Errors that can occur during vParquet4 operations
#[derive(Error, Debug)]
pub enum VParquet4Error {
    /// Error reading or parsing Parquet file
    #[error("Parquet error: {0}")]
    Parquet(#[from] parquet::errors::ParquetError),

    /// Error with Arrow data structures
    #[error("Arrow error: {0}")]
    Arrow(#[from] arrow::error::ArrowError),

    /// Schema validation error
    #[error("Invalid schema: {0}")]
    InvalidSchema(String),

    /// Missing required column
    #[error("Missing required column: {0}")]
    MissingColumn(String),

    /// Invalid column type
    #[error("Invalid column type for {column}: expected {expected}, got {actual}")]
    InvalidColumnType {
        column: String,
        expected: String,
        actual: String,
    },

    /// Row group filtering error
    #[error("Row group filter error: {0}")]
    FilterError(String),

    /// Data conversion error
    #[error("Data conversion error: {0}")]
    ConversionError(String),

    /// I/O error
    #[error("I/O error: {0}")]
    Io(#[from] std::io::Error),

    /// Object store error
    #[error("Object store error: {0}")]
    ObjectStore(#[from] object_store::Error),

    /// Invalid trace ID
    #[error("Invalid trace ID: {0}")]
    InvalidTraceId(String),

    /// Invalid span ID
    #[error("Invalid span ID: {0}")]
    InvalidSpanId(String),

    /// Generic error
    #[error("{0}")]
    Generic(String),
}

impl From<String> for VParquet4Error {
    fn from(s: String) -> Self {
        VParquet4Error::Generic(s)
    }
}

impl From<&str> for VParquet4Error {
    fn from(s: &str) -> Self {
        VParquet4Error::Generic(s.to_string())
    }
}
