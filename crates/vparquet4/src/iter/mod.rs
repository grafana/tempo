//! High-level iteration APIs for vParquet4 traces and spans
//!
//! This module provides convenient iterators for working with vParquet4 data:
//!
//! - [`TraceIterator`]: Iterates over complete traces (with all ResourceSpans, ScopeSpans, and Spans)
//! - [`SpanIterator`]: Flattens the nested structure and iterates over individual spans
//!
//! # Example
//!
//! ```rust,no_run
//! use vparquet4::{VParquet4Reader, ReaderConfig};
//! use vparquet4::iter::TraceIterator;
//!
//! # fn main() -> Result<(), Box<dyn std::error::Error>> {
//! let reader = VParquet4Reader::open("path/to/data.parquet", ReaderConfig::default())?;
//! let trace_iter = TraceIterator::new(reader)?;
//!
//! for trace in trace_iter {
//!     let trace = trace?;
//!     println!("Trace ID: {:?}", hex::encode(&trace.trace_id));
//!     println!("Spans: {}", trace.total_spans());
//! }
//! # Ok(())
//! # }
//! ```

pub mod span_iter;
pub mod trace_iter;

pub use span_iter::{SpanIterator, SpanWithContext};
pub use trace_iter::{Trace, TraceIterator};
