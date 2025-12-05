//! Column projection for vParquet4 files
//!
//! This module provides functionality for selecting specific columns to read,
//! reducing I/O and memory usage by only loading the data you need.

pub mod builder;

pub use builder::ProjectionBuilder;

/// Projection mode presets for common use cases
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ProjectionMode {
    /// Read all columns (no projection)
    All,

    /// Read only trace-level summary fields:
    /// - TraceID, TraceIDText
    /// - StartTimeUnixNano, EndTimeUnixNano, DurationNano
    /// - RootServiceName, RootSpanName
    /// - ServiceStats
    TraceSummaryOnly,

    /// Read trace and span fields, but exclude attributes:
    /// - All trace-level fields
    /// - Span core fields (ID, name, times, status, kind)
    /// - No attributes (resource, span, event, link)
    SpansWithoutAttrs,

    /// Read all trace and span fields including attributes
    /// (same as All, but more explicit)
    FullSpans,
}
