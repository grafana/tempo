//! Iterator for traces in vParquet4 files
//!
//! This module provides [`TraceIterator`] which yields complete [`Trace`] objects,
//! including all ResourceSpans, ScopeSpans, and Spans from a vParquet4 file.

use crate::domain::{convert, ResourceSpans};
use crate::error::{Result, VParquet4Error};
use crate::reader::{VParquet4Reader, VParquet4ReaderTrait};
use arrow::array::{Array, BinaryArray};
use arrow::record_batch::RecordBatch;

/// A complete trace with all associated spans
#[derive(Debug, Clone)]
pub struct Trace {
    /// 16-byte trace ID
    pub trace_id: Vec<u8>,

    /// Resource spans grouped by resource and scope
    pub resource_spans: Vec<ResourceSpans>,

    /// Start time of the earliest span (unix nanoseconds)
    pub start_time_unix_nano: u64,

    /// End time of the latest span (unix nanoseconds)
    pub end_time_unix_nano: u64,
}

impl Trace {
    /// Creates a new Trace from a RecordBatch row
    pub fn from_batch(batch: &RecordBatch, row_index: usize) -> Result<Self> {
        // Parse trace ID
        let trace_id = Self::parse_trace_id(batch, row_index)?;

        // Parse ResourceSpans
        let resource_spans = convert::parse_resource_spans_from_batch(batch, row_index)?;

        // Calculate time range from spans
        let (start_time, end_time) = Self::calculate_time_range(&resource_spans);

        Ok(Self {
            trace_id,
            resource_spans,
            start_time_unix_nano: start_time,
            end_time_unix_nano: end_time,
        })
    }

    /// Parses the trace ID from a RecordBatch
    fn parse_trace_id(batch: &RecordBatch, row_index: usize) -> Result<Vec<u8>> {
        let trace_id_col = batch
            .column_by_name("TraceID")
            .ok_or_else(|| VParquet4Error::MissingColumn("TraceID".to_string()))?;

        let trace_id_array = trace_id_col
            .as_any()
            .downcast_ref::<BinaryArray>()
            .ok_or_else(|| {
                VParquet4Error::ConversionError("TraceID is not a BinaryArray".to_string())
            })?;

        if trace_id_array.is_null(row_index) {
            return Err(VParquet4Error::ConversionError(
                "TraceID is null".to_string(),
            ));
        }

        Ok(trace_id_array.value(row_index).to_vec())
    }

    /// Calculates the time range by examining all spans
    fn calculate_time_range(resource_spans: &[ResourceSpans]) -> (u64, u64) {
        let mut min_time = u64::MAX;
        let mut max_time = u64::MIN;

        for rs in resource_spans {
            for ss in &rs.scope_spans {
                for span in &ss.spans {
                    if span.start_time_unix_nano > 0 {
                        min_time = min_time.min(span.start_time_unix_nano);
                        max_time = max_time.max(span.end_time_unix_nano);
                    }
                }
            }
        }

        // Handle case where no valid timestamps found
        if min_time == u64::MAX {
            (0, 0)
        } else {
            (min_time, max_time)
        }
    }

    /// Returns the total number of spans in this trace
    pub fn total_spans(&self) -> usize {
        self.resource_spans
            .iter()
            .flat_map(|rs| &rs.scope_spans)
            .map(|ss| ss.spans.len())
            .sum()
    }

    /// Returns the trace ID as a hex string
    pub fn trace_id_hex(&self) -> String {
        hex::encode(&self.trace_id)
    }

    /// Returns the duration of this trace in nanoseconds
    pub fn duration_nanos(&self) -> u64 {
        if self.end_time_unix_nano >= self.start_time_unix_nano {
            self.end_time_unix_nano - self.start_time_unix_nano
        } else {
            0
        }
    }
}

/// Iterator over traces in a vParquet4 file
///
/// Yields complete [`Trace`] objects, reading row groups sequentially.
///
/// # Example
///
/// ```rust,no_run
/// use vparquet4::{VParquet4Reader, ReaderConfig};
/// use vparquet4::iter::TraceIterator;
///
/// # fn main() -> Result<(), Box<dyn std::error::Error>> {
/// let reader = VParquet4Reader::open("path/to/data.parquet", ReaderConfig::default())?;
/// let trace_iter = TraceIterator::new(reader)?;
///
/// for trace in trace_iter {
///     let trace = trace?;
///     println!("Trace {}: {} spans", trace.trace_id_hex(), trace.total_spans());
/// }
/// # Ok(())
/// # }
/// ```
pub struct TraceIterator {
    /// The underlying reader
    reader: VParquet4Reader,

    /// Current row group being processed
    current_row_group: usize,

    /// Current batch being processed
    current_batch: Option<RecordBatch>,

    /// Current row within the batch
    current_row: usize,
}

impl TraceIterator {
    /// Creates a new TraceIterator from a reader
    pub fn new(reader: VParquet4Reader) -> Result<Self> {
        Ok(Self {
            reader,
            current_row_group: 0,
            current_batch: None,
            current_row: 0,
        })
    }

    /// Loads the next row group into current_batch
    fn load_next_row_group(&mut self) -> Result<bool> {
        if self.current_row_group >= self.reader.num_row_groups() {
            return Ok(false);
        }

        let batch = self.reader.read_row_group(self.current_row_group)?;
        self.current_batch = Some(batch);
        self.current_row = 0;
        self.current_row_group += 1;

        Ok(true)
    }
}

impl Iterator for TraceIterator {
    type Item = Result<Trace>;

    fn next(&mut self) -> Option<Self::Item> {
        loop {
            // If we don't have a current batch, try to load the next row group
            if self.current_batch.is_none() {
                match self.load_next_row_group() {
                    Ok(true) => {} // Successfully loaded a batch
                    Ok(false) => return None, // No more row groups
                    Err(e) => return Some(Err(e)),
                }
            }

            // Process current batch
            if let Some(ref batch) = self.current_batch {
                if self.current_row < batch.num_rows() {
                    let row_index = self.current_row;
                    self.current_row += 1;

                    return Some(Trace::from_batch(batch, row_index));
                } else {
                    // Current batch exhausted, load next row group
                    self.current_batch = None;
                }
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_trace_time_range() {
        use crate::domain::{Span, ScopeSpans};

        let mut spans = vec![
            Span {
                start_time_unix_nano: 1000,
                end_time_unix_nano: 2000,
                ..Default::default()
            },
            Span {
                start_time_unix_nano: 1500,
                end_time_unix_nano: 3000,
                ..Default::default()
            },
        ];

        // Calculate time range
        let resource_spans = vec![ResourceSpans {
            resource: None,
            scope_spans: vec![ScopeSpans {
                scope: None,
                spans: spans.clone(),
                schema_url: String::new(),
            }],
            schema_url: String::new(),
        }];

        let (start, end) = Trace::calculate_time_range(&resource_spans);
        assert_eq!(start, 1000);
        assert_eq!(end, 3000);
    }

    #[test]
    fn test_trace_duration() {
        let trace = Trace {
            trace_id: vec![1, 2, 3],
            resource_spans: vec![],
            start_time_unix_nano: 1000,
            end_time_unix_nano: 5000,
        };

        assert_eq!(trace.duration_nanos(), 4000);
    }

    #[test]
    fn test_trace_id_hex() {
        let trace = Trace {
            trace_id: vec![0xDE, 0xAD, 0xBE, 0xEF],
            resource_spans: vec![],
            start_time_unix_nano: 0,
            end_time_unix_nano: 0,
        };

        assert_eq!(trace.trace_id_hex(), "deadbeef");
    }
}
