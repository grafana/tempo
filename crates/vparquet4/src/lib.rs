//! # vparquet4
//!
//! A standalone Rust library for reading Tempo vparquet4 files with async Stream API.
//!
//! This crate provides efficient reading of vparquet4 parquet files with support for:
//! - Async Stream API for non-blocking I/O
//! - Parallel row group processing with Tokio
//! - Dictionary-based filtering for improved performance
//! - Statistics-based row group pruning
//! - Filtering spans by name (with potential for more filter types)
//!
//! ## Example
//!
//! ```no_run
//! use vparquet4::{VParquet4Reader, ReadOptions, SpanFilter};
//! use futures::StreamExt;
//!
//! #[tokio::main]
//! async fn main() -> Result<(), Box<dyn std::error::Error>> {
//!     let options = ReadOptions {
//!         filter: Some(SpanFilter::NameEquals("distributor.ConsumeTraces".to_string())),
//!         batch_size: 4,
//!         parallelism: 8,
//!     };
//!
//!     let reader = VParquet4Reader::open("data.parquet", options).await?;
//!     let mut stream = reader.read();
//!
//!     while let Some(result) = stream.next().await {
//!         let spanset = result?;
//!         println!("Trace {} has {} matching spans",
//!                  hex::encode(&spanset.trace_id),
//!                  spanset.spans.len());
//!     }
//!
//!     Ok(())
//! }
//! ```

pub mod error;
pub mod filter;
pub mod reader;
pub mod schema;

pub use error::{Error, Result};
pub use filter::{CachedDictionary, SpanFilter};
pub use reader::{FileCache, ReadOptions, SpansetStream, VParquet4Reader};
pub use schema::{FilterResult, Span, Spanset};
