/// Filter conditions for querying vparquet4 files
use std::collections::HashSet;
use std::fmt;

use arrow::array::{Array, BooleanArray, ListArray, RecordBatch, StringArray, StructArray};
use arrow::error::ArrowError;
use parquet::arrow::arrow_reader::ArrowPredicate;
use parquet::arrow::ProjectionMask;
use parquet::data_type::ByteArray;
use parquet::file::metadata::RowGroupMetaData;
use parquet::schema::types::SchemaDescriptor;

use crate::schema::field_paths;

/// Cached dictionary for row group column
#[derive(Debug, Clone)]
pub struct CachedDictionary {
    pub values: Vec<ByteArray>,
    pub value_set: HashSet<Vec<u8>>,
}

/// A filter condition for querying spans
#[derive(Debug, Clone)]
pub enum SpanFilter {
    /// Filter spans where the Name field equals the given value
    NameEquals(String),
}

impl SpanFilter {
    /// Check if a span name matches this filter
    pub fn matches(&self, span_name: &str) -> bool {
        match self {
            SpanFilter::NameEquals(expected) => span_name == expected,
        }
    }

    /// Check if row group might contain matching spans based on statistics.
    /// Returns true (keep) if uncertain, false only if provably no match.
    pub fn keep_row_group(&self, rg_meta: &RowGroupMetaData, schema: &SchemaDescriptor) -> bool {
        match self {
            SpanFilter::NameEquals(expected) => {
                // Find column index for nested Name field
                let col_idx = schema
                    .columns()
                    .iter()
                    .position(|c| c.path().string() == field_paths::SPAN_NAME);

                let Some(idx) = col_idx else {
                    return true;
                };
                let col_meta = rg_meta.column(idx);
                let Some(stats) = col_meta.statistics() else {
                    return true;
                };

                // Check min/max range
                if let (Some(min), Some(max)) = (stats.min_bytes_opt(), stats.max_bytes_opt()) {
                    let expected_bytes = expected.as_bytes();
                    // Skip if expected < min or expected > max
                    if expected_bytes < min || expected_bytes > max {
                        return false;
                    }
                }
                true
            }
        }
    }

    /// Check if dictionary contains matching value
    pub fn matches_dictionary(&self, dict: &CachedDictionary) -> bool {
        match self {
            SpanFilter::NameEquals(expected) => dict.value_set.contains(expected.as_bytes()),
        }
    }
}

impl fmt::Display for SpanFilter {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            SpanFilter::NameEquals(name) => write!(f, "name = '{}'", name),
        }
    }
}

/// Arrow predicate for filtering spans by name during parquet read
pub struct SpanNamePredicate {
    expected: String,
    projection: ProjectionMask,
}

impl SpanNamePredicate {
    pub fn new(expected: String, schema: &SchemaDescriptor) -> Self {
        // Build projection mask for ONLY the span Name column
        // This avoids reading any Resource or Trace level fields during predicate evaluation
        let name_col_idx = schema
            .columns()
            .iter()
            .position(|c| c.path().string() == field_paths::SPAN_NAME);

        let mask = match name_col_idx {
            Some(idx) => ProjectionMask::leaves(schema, [idx]),
            None => ProjectionMask::all(), // Fallback if not found
        };

        Self {
            expected,
            projection: mask,
        }
    }
}

impl ArrowPredicate for SpanNamePredicate {
    fn projection(&self) -> &ProjectionMask {
        &self.projection
    }

    fn evaluate(&mut self, batch: RecordBatch) -> Result<BooleanArray, ArrowError> {
        let num_rows = batch.num_rows();
        let mut results = vec![false; num_rows];

        // Navigate: batch -> rs (list) -> element (struct) -> ss (list) -> element (struct) -> Spans (list) -> element (struct) -> Name
        let rs = batch
            .column_by_name("rs")
            .and_then(|c| c.as_any().downcast_ref::<ListArray>());

        let Some(rs) = rs else {
            return Ok(BooleanArray::from(results));
        };

        // For each trace (row), check if any span name matches
        for row in 0..num_rows {
            if has_matching_span_name(rs, row, &self.expected) {
                results[row] = true;
            }
        }

        Ok(BooleanArray::from(results))
    }
}

/// Navigate nested structure to find any matching span name
/// Returns true if ANY span in this trace has name == expected
fn has_matching_span_name(rs: &ListArray, row: usize, expected: &str) -> bool {
    // Navigate rs -> ss -> Spans -> Name
    let rs_offset = rs.value_offsets()[row] as usize;
    let rs_length = (rs.value_offsets()[row + 1] - rs.value_offsets()[row]) as usize;

    if rs_length == 0 {
        return false;
    }

    let rs_values = match rs.values().as_any().downcast_ref::<StructArray>() {
        Some(v) => v,
        None => return false,
    };

    // Get ScopeSpans (ss) from ResourceSpans
    let ss_array = match rs_values
        .column_by_name("ss")
        .and_then(|col| col.as_any().downcast_ref::<ListArray>())
    {
        Some(a) => a,
        None => return false,
    };

    // Iterate through each ResourceSpans
    for rs_idx in rs_offset..rs_offset + rs_length {
        let ss_offset = ss_array.value_offsets()[rs_idx] as usize;
        let ss_length =
            (ss_array.value_offsets()[rs_idx + 1] - ss_array.value_offsets()[rs_idx]) as usize;

        if ss_length == 0 {
            continue;
        }

        let ss_values = match ss_array.values().as_any().downcast_ref::<StructArray>() {
            Some(v) => v,
            None => continue,
        };

        // Get Spans from ScopeSpans
        let spans_array = match ss_values
            .column_by_name("Spans")
            .and_then(|col| col.as_any().downcast_ref::<ListArray>())
        {
            Some(a) => a,
            None => continue,
        };

        // Iterate through each ScopeSpans
        for ss_idx in ss_offset..ss_offset + ss_length {
            let spans_offset = spans_array.value_offsets()[ss_idx] as usize;
            let spans_length = (spans_array.value_offsets()[ss_idx + 1]
                - spans_array.value_offsets()[ss_idx]) as usize;

            if spans_length == 0 {
                continue;
            }

            let spans_values = match spans_array.values().as_any().downcast_ref::<StructArray>() {
                Some(v) => v,
                None => continue,
            };

            // Get Name field from Spans
            let names = match spans_values
                .column_by_name("Name")
                .and_then(|col| col.as_any().downcast_ref::<StringArray>())
            {
                Some(a) => a,
                None => continue,
            };

            // Check each span name
            for span_idx in spans_offset..spans_offset + spans_length {
                let name = names.value(span_idx);
                if name == expected {
                    return true;
                }
            }
        }
    }

    false
}
