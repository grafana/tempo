//! Row group statistics extraction for vParquet4 files
//!
//! This module provides utilities for extracting statistics from row group metadata,
//! which are used for efficient filtering without reading the actual data.

use crate::error::Result;
use crate::schema::field_paths::trace;
use parquet::data_type::AsBytes;
use parquet::file::metadata::RowGroupMetaData;
use parquet::file::statistics::Statistics;

/// Statistics extracted from a row group for filtering
#[derive(Debug, Clone)]
pub struct RowGroupStats {
    /// Minimum StartTimeUnixNano value in this row group
    pub min_start_time_ns: Option<u64>,

    /// Maximum StartTimeUnixNano value in this row group
    pub max_start_time_ns: Option<u64>,

    /// Minimum EndTimeUnixNano value in this row group
    pub min_end_time_ns: Option<u64>,

    /// Maximum EndTimeUnixNano value in this row group
    pub max_end_time_ns: Option<u64>,

    /// Minimum TraceID value in this row group (binary)
    pub min_trace_id: Option<Vec<u8>>,

    /// Maximum TraceID value in this row group (binary)
    pub max_trace_id: Option<Vec<u8>>,

    /// Number of rows in this row group
    pub num_rows: i64,
}

impl RowGroupStats {
    /// Extracts statistics from a row group's metadata
    pub fn from_metadata(row_group: &RowGroupMetaData) -> Result<Self> {
        let num_rows = row_group.num_rows();

        // Extract start time statistics
        let (min_start_time_ns, max_start_time_ns) =
            Self::extract_u64_range(row_group, trace::START_TIME_UNIX_NANO)?;

        // Extract end time statistics
        let (min_end_time_ns, max_end_time_ns) =
            Self::extract_u64_range(row_group, trace::END_TIME_UNIX_NANO)?;

        // Extract trace ID statistics
        let (min_trace_id, max_trace_id) =
            Self::extract_binary_range(row_group, trace::TRACE_ID)?;

        Ok(Self {
            min_start_time_ns,
            max_start_time_ns,
            min_end_time_ns,
            max_end_time_ns,
            min_trace_id,
            max_trace_id,
            num_rows,
        })
    }

    /// Extracts u64 min/max statistics for a column
    fn extract_u64_range(
        row_group: &RowGroupMetaData,
        column_name: &str,
    ) -> Result<(Option<u64>, Option<u64>)> {
        let column_index = Self::find_column_index(row_group, column_name)?;

        if let Some(idx) = column_index {
            let column_chunk = row_group.column(idx);
            if let Some(stats) = column_chunk.statistics() {
                return match stats {
                    Statistics::Int64(ref s) => {
                        // StartTimeUnixNano and EndTimeUnixNano are stored as i64 in Parquet
                        // but represent u64 values
                        if let (Some(min), Some(max)) = (s.min_opt(), s.max_opt()) {
                            let min = *min as u64;
                            let max = *max as u64;
                            Ok((Some(min), Some(max)))
                        } else {
                            Ok((None, None))
                        }
                    }
                    _ => Ok((None, None)),
                };
            }
        }

        Ok((None, None))
    }

    /// Extracts binary min/max statistics for a column
    fn extract_binary_range(
        row_group: &RowGroupMetaData,
        column_name: &str,
    ) -> Result<(Option<Vec<u8>>, Option<Vec<u8>>)> {
        let column_index = Self::find_column_index(row_group, column_name)?;

        if let Some(idx) = column_index {
            let column_chunk = row_group.column(idx);
            if let Some(stats) = column_chunk.statistics() {
                return match stats {
                    Statistics::FixedLenByteArray(ref s) => {
                        if let (Some(min), Some(max)) = (s.min_opt(), s.max_opt()) {
                            let min = min.as_bytes().to_vec();
                            let max = max.as_bytes().to_vec();
                            Ok((Some(min), Some(max)))
                        } else {
                            Ok((None, None))
                        }
                    }
                    Statistics::ByteArray(ref s) => {
                        if let (Some(min), Some(max)) = (s.min_opt(), s.max_opt()) {
                            let min = min.as_bytes().to_vec();
                            let max = max.as_bytes().to_vec();
                            Ok((Some(min), Some(max)))
                        } else {
                            Ok((None, None))
                        }
                    }
                    _ => Ok((None, None)),
                };
            }
        }

        Ok((None, None))
    }

    /// Finds the column index for a given column name
    fn find_column_index(row_group: &RowGroupMetaData, column_name: &str) -> Result<Option<usize>> {
        // Parquet stores columns in a flat structure, so we need to match by name
        for (idx, column) in row_group.columns().iter().enumerate() {
            if column.column_descr().name() == column_name {
                return Ok(Some(idx));
            }
        }
        Ok(None)
    }

    /// Checks if this row group might contain traces in the given time range
    ///
    /// Returns true if the row group's time range overlaps with the query range.
    /// Uses both start and end times for more accurate filtering.
    pub fn overlaps_time_range(&self, query_min_ns: u64, query_max_ns: u64) -> bool {
        // If we don't have time statistics, we must include this row group
        let Some(min_start) = self.min_start_time_ns else {
            return true;
        };
        let Some(max_end) = self.max_end_time_ns else {
            return true;
        };

        // Check if ranges overlap: row group is included if its time range overlaps with query range
        // Overlap occurs if: max_end >= query_min_ns AND min_start <= query_max_ns
        max_end >= query_min_ns && min_start <= query_max_ns
    }

    /// Checks if this row group might contain traces with the given ID prefix
    ///
    /// Returns true if any trace ID in this row group could start with the prefix.
    pub fn matches_trace_id_prefix(&self, prefix: &[u8]) -> bool {
        // If we don't have trace ID statistics, we must include this row group
        let Some(ref min_trace_id) = self.min_trace_id else {
            return true;
        };
        let Some(ref max_trace_id) = self.max_trace_id else {
            return true;
        };

        // Check if the prefix range [prefix, prefix + 1) overlaps with [min, max]
        // A trace ID starting with 'prefix' must be >= prefix and < prefix + 1
        //
        // The row group contains matching IDs if:
        // - max_trace_id >= prefix (the range extends into or past the prefix)
        // - min_trace_id < prefix + 1 (the range starts before the end of prefix range)
        //
        // Simplified: max >= prefix AND min < prefix_upper_bound
        // But since we're doing prefix matching, we just check if min or max starts with prefix,
        // or if prefix falls within [min, max]

        // Check if min or max starts with the prefix
        if min_trace_id.starts_with(prefix) || max_trace_id.starts_with(prefix) {
            return true;
        }

        // Check if prefix falls between min and max
        // This handles the case where the prefix is between min and max
        if min_trace_id.as_slice() <= prefix && prefix <= max_trace_id.as_slice() {
            return true;
        }

        // Also need to check if any ID in the range [min, max] could start with prefix
        // This is true if min <= prefix and max >= prefix (lexicographically)
        let prefix_len = prefix.len();
        let min_prefix = &min_trace_id[..prefix_len.min(min_trace_id.len())];
        let max_prefix = &max_trace_id[..prefix_len.min(max_trace_id.len())];

        // If min_prefix <= prefix <= max_prefix, then there might be IDs with this prefix
        min_prefix <= prefix && prefix <= max_prefix
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_time_range_overlap() {
        let stats = RowGroupStats {
            min_start_time_ns: Some(1000),
            max_start_time_ns: Some(2000),
            min_end_time_ns: Some(1500),
            max_end_time_ns: Some(2500),
            min_trace_id: None,
            max_trace_id: None,
            num_rows: 100,
        };

        // Query range completely before row group - no overlap
        assert!(!stats.overlaps_time_range(0, 999));

        // Query range completely after row group - no overlap
        assert!(!stats.overlaps_time_range(2501, 3000));

        // Query range overlaps with row group start
        assert!(stats.overlaps_time_range(500, 1500));

        // Query range overlaps with row group end
        assert!(stats.overlaps_time_range(2000, 3000));

        // Query range completely contains row group
        assert!(stats.overlaps_time_range(0, 5000));

        // Query range completely inside row group
        assert!(stats.overlaps_time_range(1200, 1800));

        // Exact match at boundaries
        assert!(stats.overlaps_time_range(1000, 2500));
    }

    #[test]
    fn test_trace_id_prefix_match() {
        let stats = RowGroupStats {
            min_start_time_ns: None,
            max_start_time_ns: None,
            min_end_time_ns: None,
            max_end_time_ns: None,
            min_trace_id: Some(vec![0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00]),
            max_trace_id: Some(vec![0x20, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff]),
            num_rows: 100,
        };

        // Prefix matches min
        assert!(stats.matches_trace_id_prefix(&[0x10]));
        assert!(stats.matches_trace_id_prefix(&[0x10, 0x00]));

        // Prefix matches max
        assert!(stats.matches_trace_id_prefix(&[0x20]));
        assert!(stats.matches_trace_id_prefix(&[0x20, 0xff]));

        // Prefix in between min and max
        assert!(stats.matches_trace_id_prefix(&[0x15]));
        assert!(stats.matches_trace_id_prefix(&[0x18, 0x50]));

        // Prefix before min
        assert!(!stats.matches_trace_id_prefix(&[0x05]));

        // Prefix after max
        assert!(!stats.matches_trace_id_prefix(&[0x30]));
        assert!(!stats.matches_trace_id_prefix(&[0x21]));
    }

    #[test]
    fn test_missing_statistics() {
        let stats = RowGroupStats {
            min_start_time_ns: None,
            max_start_time_ns: None,
            min_end_time_ns: None,
            max_end_time_ns: None,
            min_trace_id: None,
            max_trace_id: None,
            num_rows: 100,
        };

        // Without statistics, all filters should return true (conservative approach)
        assert!(stats.overlaps_time_range(0, 1000));
        assert!(stats.matches_trace_id_prefix(&[0x10, 0x20]));
    }
}
