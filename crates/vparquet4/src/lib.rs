//! # vparquet4
//!
//! A standalone Rust library for reading Tempo vparquet4 files.
//!
//! This crate provides efficient reading of vparquet4 parquet files with support for:
//! - Row group selection for parallel processing
//! - Filtering spans by name (with potential for more filter types)
//! - Lazy iteration over spansets (groups of matching spans per trace)
//!
//! ## Example
//!
//! ```no_run
//! use vparquet4::{VParquet4Reader, ReadOptions, SpanFilter};
//!
//! let options = ReadOptions {
//!     start_row_group: 0,
//!     total_row_groups: 0, // 0 = read all
//!     filter: Some(SpanFilter::NameEquals("distributor.ConsumeTraces".to_string())),
//! };
//!
//! let reader = VParquet4Reader::open("data.parquet", options).unwrap();
//!
//! for result in reader {
//!     let spanset = result.unwrap();
//!     println!("Trace {} has {} matching spans",
//!              hex::encode(&spanset.trace_id),
//!              spanset.spans.len());
//! }
//! ```

pub mod error;
pub mod filter;
pub mod reader;
pub mod schema;

pub use error::{Error, Result};
pub use filter::SpanFilter;
pub use reader::{ReadOptions, VParquet4Reader};
pub use schema::{Span, Spanset};
