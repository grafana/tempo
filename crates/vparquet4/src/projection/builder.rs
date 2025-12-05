//! Projection builder for creating column projections
//!
//! This module provides a builder API for constructing column projections,
//! with presets for common use cases.

use crate::error::Result;
use crate::projection::ProjectionMode;
use crate::schema::field_paths::{resource, span, trace};
use arrow::datatypes::SchemaRef;
use parquet::arrow::ProjectionMask;
use parquet::schema::types::SchemaDescriptor;
use std::collections::HashSet;

/// Builder for creating column projections
///
/// Provides a fluent API for selecting which columns to read from a vParquet4 file.
#[derive(Debug, Clone)]
pub struct ProjectionBuilder {
    /// Column names to include in the projection
    columns: HashSet<String>,

    /// Projection mode (for preset configurations)
    mode: ProjectionMode,
}

impl ProjectionBuilder {
    /// Creates a new projection builder with no columns selected
    pub fn new() -> Self {
        Self {
            columns: HashSet::new(),
            mode: ProjectionMode::All,
        }
    }

    /// Creates a projection for trace summary only
    ///
    /// Includes:
    /// - TraceID, TraceIDText
    /// - StartTimeUnixNano, EndTimeUnixNano, DurationNano
    /// - RootServiceName, RootSpanName
    /// - ServiceStats
    pub fn trace_summary_only() -> Self {
        let mut columns = HashSet::new();
        columns.insert(trace::TRACE_ID.to_string());
        columns.insert(trace::TRACE_ID_TEXT.to_string());
        columns.insert(trace::START_TIME_UNIX_NANO.to_string());
        columns.insert(trace::END_TIME_UNIX_NANO.to_string());
        columns.insert(trace::DURATION_NANO.to_string());
        columns.insert(trace::ROOT_SERVICE_NAME.to_string());
        columns.insert(trace::ROOT_SPAN_NAME.to_string());
        columns.insert(trace::SERVICE_STATS.to_string());

        Self {
            columns,
            mode: ProjectionMode::TraceSummaryOnly,
        }
    }

    /// Creates a projection for spans without attributes
    ///
    /// Note: Due to the nested structure of vParquet4, this projection includes
    /// the entire ResourceSpans ("rs") column. To actually exclude attributes,
    /// you would need to filter them during domain type construction (Phase 3).
    ///
    /// For now, this is equivalent to FullSpans but documented differently
    /// to indicate the intent for future optimization.
    ///
    /// Includes:
    /// - All trace-level fields
    /// - ResourceSpans column (contains all span data including attributes)
    pub fn spans_without_attrs() -> Self {
        let mut columns = HashSet::new();

        // All top-level fields including ResourceSpans
        columns.insert(trace::TRACE_ID.to_string());
        columns.insert(trace::TRACE_ID_TEXT.to_string());
        columns.insert(trace::START_TIME_UNIX_NANO.to_string());
        columns.insert(trace::END_TIME_UNIX_NANO.to_string());
        columns.insert(trace::DURATION_NANO.to_string());
        columns.insert(trace::ROOT_SERVICE_NAME.to_string());
        columns.insert(trace::ROOT_SPAN_NAME.to_string());
        columns.insert(trace::SERVICE_STATS.to_string());
        columns.insert(trace::RESOURCE_SPANS.to_string());

        Self {
            columns,
            mode: ProjectionMode::SpansWithoutAttrs,
        }
    }

    /// Creates a projection for full spans (all columns)
    pub fn full_spans() -> Self {
        Self {
            columns: HashSet::new(),
            mode: ProjectionMode::FullSpans,
        }
    }

    /// Adds a column to the projection by name
    pub fn add_column(mut self, column_name: impl Into<String>) -> Self {
        self.columns.insert(column_name.into());
        self.mode = ProjectionMode::All; // Custom projection
        self
    }

    /// Adds multiple columns to the projection
    pub fn add_columns(mut self, column_names: impl IntoIterator<Item = impl Into<String>>) -> Self {
        for name in column_names {
            self.columns.insert(name.into());
        }
        self.mode = ProjectionMode::All; // Custom projection
        self
    }

    /// Builds a ParquetProjectionMask from the selected columns
    ///
    /// If no columns are selected (FullSpans mode), returns None (read all columns).
    /// Otherwise, creates a projection mask that selects only the specified columns.
    ///
    /// # Arguments
    /// * `arrow_schema` - The Arrow schema for finding column indices
    /// * `parquet_schema` - The Parquet schema descriptor for creating the projection mask
    pub fn build(
        &self,
        arrow_schema: &SchemaRef,
        parquet_schema: &SchemaDescriptor,
    ) -> Result<Option<ProjectionMask>> {
        // If FullSpans mode or empty columns, don't create a projection (read all)
        if self.mode == ProjectionMode::FullSpans || (self.columns.is_empty() && self.mode == ProjectionMode::All) {
            return Ok(None);
        }

        // Find indices of columns in the schema
        let mut column_indices = Vec::new();
        for (idx, field) in arrow_schema.fields().iter().enumerate() {
            if self.columns.contains(field.name()) {
                column_indices.push(idx);
            }
        }

        // If we didn't find any columns, return None (read all)
        if column_indices.is_empty() {
            return Ok(None);
        }

        // Create projection mask using root-level indices
        let mask = ProjectionMask::roots(
            parquet_schema,
            column_indices.into_iter(),
        );

        Ok(Some(mask))
    }

    /// Returns the projection mode
    pub fn mode(&self) -> ProjectionMode {
        self.mode
    }

    /// Returns the selected column names
    pub fn columns(&self) -> &HashSet<String> {
        &self.columns
    }
}

impl Default for ProjectionBuilder {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use arrow::datatypes::{DataType, Field, Schema};
    use std::sync::Arc;

    #[test]
    fn test_trace_summary_only() {
        let builder = ProjectionBuilder::trace_summary_only();
        assert_eq!(builder.mode(), ProjectionMode::TraceSummaryOnly);
        assert!(builder.columns().contains(trace::TRACE_ID));
        assert!(builder.columns().contains(trace::START_TIME_UNIX_NANO));
        assert!(!builder.columns().contains(span::SPAN_ID));
    }

    #[test]
    fn test_spans_without_attrs() {
        let builder = ProjectionBuilder::spans_without_attrs();
        assert_eq!(builder.mode(), ProjectionMode::SpansWithoutAttrs);
        assert!(builder.columns().contains(trace::TRACE_ID));
        assert!(builder.columns().contains(trace::RESOURCE_SPANS));
        // Note: Individual span fields are nested and not selectable at top level
        // so we can't check for span::SPAN_ID directly in the column list
    }

    #[test]
    fn test_full_spans() {
        let builder = ProjectionBuilder::full_spans();
        assert_eq!(builder.mode(), ProjectionMode::FullSpans);
        // Note: Can't test build() without a real ParquetSchemaDescriptor
    }

    #[test]
    fn test_custom_columns() {
        let builder = ProjectionBuilder::new()
            .add_column(trace::TRACE_ID)
            .add_column(span::SPAN_ID);

        assert_eq!(builder.columns().len(), 2);
        assert!(builder.columns().contains(trace::TRACE_ID));
        assert!(builder.columns().contains(span::SPAN_ID));
    }

    #[test]
    fn test_empty_builder() {
        let builder = ProjectionBuilder::new();
        assert_eq!(builder.mode(), ProjectionMode::All);
        assert!(builder.columns().is_empty());
    }
}
