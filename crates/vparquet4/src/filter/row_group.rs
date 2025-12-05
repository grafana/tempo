//! Row group filtering implementation for vParquet4 files
//!
//! This module provides the RowGroupFilter type, which can filter row groups
//! based on time ranges and trace ID prefixes without reading the actual data.

use crate::error::Result;
use crate::filter::{RowGroupFilterTrait, RowGroupStats};
use parquet::file::metadata::RowGroupMetaData;

/// Row group filter for vParquet4 files
///
/// Filters row groups based on metadata and statistics to skip irrelevant data.
/// Supports filtering by:
/// - Time range (using StartTimeUnixNano and EndTimeUnixNano)
/// - Trace ID prefix
#[derive(Debug, Clone, Default)]
pub struct RowGroupFilter {
    /// Time range filter: (min_ns, max_ns)
    time_range: Option<(u64, u64)>,

    /// Trace ID prefix filter (binary)
    trace_id_prefix: Option<Vec<u8>>,
}

impl RowGroupFilter {
    /// Creates a new empty filter (no filtering)
    pub fn new() -> Self {
        Self::default()
    }

    /// Sets a time range filter
    ///
    /// Only row groups with traces overlapping this time range will be included.
    /// Time is in nanoseconds since Unix epoch.
    pub fn with_time_range(mut self, min_ns: u64, max_ns: u64) -> Self {
        self.time_range = Some((min_ns, max_ns));
        self
    }

    /// Sets a trace ID prefix filter
    ///
    /// Only row groups that might contain traces with this ID prefix will be included.
    pub fn with_trace_id_prefix(mut self, prefix: Vec<u8>) -> Self {
        self.trace_id_prefix = Some(prefix);
        self
    }

    /// Checks if the given row group should be included based on statistics
    fn matches_statistics(&self, stats: &RowGroupStats) -> bool {
        // Check time range filter
        if let Some((min_ns, max_ns)) = self.time_range {
            if !stats.overlaps_time_range(min_ns, max_ns) {
                return false;
            }
        }

        // Check trace ID prefix filter
        if let Some(ref prefix) = self.trace_id_prefix {
            if !stats.matches_trace_id_prefix(prefix) {
                return false;
            }
        }

        // All filters passed
        true
    }
}

impl RowGroupFilterTrait for RowGroupFilter {
    fn should_include(&self, row_group: &RowGroupMetaData) -> Result<bool> {
        // Extract statistics from the row group
        let stats = RowGroupStats::from_metadata(row_group)?;

        // Check if statistics match our filters
        Ok(self.matches_statistics(&stats))
    }
}

/// Helper function to filter row groups and return indices of included groups
///
/// This is a convenience function that applies a filter to all row groups
/// and returns the indices of row groups that should be included.
pub fn filter_row_groups(
    row_groups: &[RowGroupMetaData],
    filter: &impl RowGroupFilterTrait,
) -> Result<Vec<usize>> {
    let mut included = Vec::new();
    for (idx, rg) in row_groups.iter().enumerate() {
        if filter.should_include(rg)? {
            included.push(idx);
        }
    }
    Ok(included)
}

#[cfg(test)]
mod tests {
    use super::*;

    // Note: Full integration tests with real Parquet data will be in the tests/ directory
    // These are just unit tests for the filter logic

    #[test]
    fn test_empty_filter() {
        let filter = RowGroupFilter::new();
        assert!(filter.time_range.is_none());
        assert!(filter.trace_id_prefix.is_none());
    }

    #[test]
    fn test_time_range_filter() {
        let filter = RowGroupFilter::new().with_time_range(1000, 2000);
        assert_eq!(filter.time_range, Some((1000, 2000)));
        assert!(filter.trace_id_prefix.is_none());
    }

    #[test]
    fn test_trace_id_prefix_filter() {
        let prefix = vec![0x12, 0x34, 0x56, 0x78];
        let filter = RowGroupFilter::new().with_trace_id_prefix(prefix.clone());
        assert!(filter.time_range.is_none());
        assert_eq!(filter.trace_id_prefix, Some(prefix));
    }

    #[test]
    fn test_combined_filters() {
        let prefix = vec![0x12, 0x34];
        let filter = RowGroupFilter::new()
            .with_time_range(1000, 2000)
            .with_trace_id_prefix(prefix.clone());
        assert_eq!(filter.time_range, Some((1000, 2000)));
        assert_eq!(filter.trace_id_prefix, Some(prefix));
    }
}
