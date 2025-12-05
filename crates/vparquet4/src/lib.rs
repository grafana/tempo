//! # vParquet4 - High-Performance Rust Reader for Tempo vParquet4 Format
//!
//! This crate provides a standalone, high-performance reader for Tempo's vParquet4
//! trace format. It offers both low-level Parquet access and high-level domain APIs
//! for working with distributed traces.
//!
//! ## Features
//!
//! - **Read-only access**: Efficient reading of vParquet4 trace files
//! - **Standalone**: No DataFusion dependency
//! - **Column pruning**: Read only the columns you need via projection
//! - **Row group filtering**: Skip irrelevant row groups based on statistics
//! - **Zero-copy**: Reuse Arrow RecordBatch where possible
//! - **Async I/O**: Non-blocking reads from object stores (with `async` feature)
//!
//! ## Example
//!
//! ```rust,no_run
//! use vparquet4::{VParquet4Reader, ReaderConfig, VParquet4ReaderTrait};
//!
//! # fn main() -> Result<(), Box<dyn std::error::Error>> {
//! // Open a vParquet4 file
//! let mut reader = VParquet4Reader::open(
//!     "path/to/data.parquet",
//!     ReaderConfig::default()
//! )?;
//!
//! // Read all row groups
//! println!("Total rows: {}", reader.num_rows());
//! println!("Row groups: {}", reader.num_row_groups());
//!
//! // Read a specific row group
//! let batch = reader.read_row_group(0)?;
//! println!("Batch has {} rows", batch.num_rows());
//! # Ok(())
//! # }
//! ```

pub mod error;
pub mod filter;
pub mod projection;
pub mod reader;
pub mod schema;

// Re-export commonly used types
pub use error::{Result, VParquet4Error};
pub use filter::{RowGroupFilter, RowGroupFilterTrait, RowGroupStats};
pub use projection::{ProjectionBuilder, ProjectionMode};
pub use reader::{ReaderConfig, VParquet4ReaderTrait};
pub use reader::sync_reader::VParquet4Reader;

#[cfg(feature = "async")]
pub use reader::async_reader::AsyncVParquet4Reader;
