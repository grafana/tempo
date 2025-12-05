//! Row group filtering for vParquet4 files
//!
//! This module provides functionality for filtering row groups based on metadata
//! and statistics, allowing efficient skipping of irrelevant data.

pub mod row_group;
pub mod statistics;

pub use row_group::RowGroupFilter;
pub use statistics::RowGroupStats;

use crate::error::Result;
use parquet::file::metadata::RowGroupMetaData;

/// Trait for filtering row groups based on metadata
pub trait RowGroupFilterTrait {
    /// Returns true if the row group should be included (not filtered out)
    fn should_include(&self, row_group: &RowGroupMetaData) -> Result<bool>;
}
