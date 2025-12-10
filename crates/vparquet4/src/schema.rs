/// vparquet4 schema definitions
///
/// This module defines the schema structure for vparquet4 files, matching the Go implementation
/// in tempodb/encoding/vparquet4/schema.go
use bytes::Bytes;

/// Represents a trace-level record in the parquet file
#[derive(Debug, Clone)]
pub struct Trace {
    pub trace_id: Bytes,
    pub trace_id_text: String,
    pub start_time_unix_nano: u64,
    pub end_time_unix_nano: u64,
    pub duration_nano: u64,
    pub root_service_name: String,
    pub root_span_name: String,
}

/// Represents a span within a trace
#[derive(Debug, Clone)]
pub struct Span {
    pub span_id: Bytes,
    pub parent_span_id: Bytes,
    pub parent_id: i32,
    pub nested_set_left: i32,
    pub nested_set_right: i32,
    pub name: String,
    pub kind: i64,
    pub start_time_unix_nano: u64,
    pub duration_nano: u64,
    pub status_code: i64,
}

/// A spanset represents all spans from a single trace that match the filter conditions
#[derive(Debug, Clone)]
pub struct Spanset {
    pub trace_id: Bytes,
    pub spans: Vec<Span>,
}

/// Field paths in the parquet schema
pub mod field_paths {
    pub const TRACE_ID: &str = "TraceID";
    pub const TRACE_ID_TEXT: &str = "TraceIDText";
    pub const START_TIME_UNIX_NANO: &str = "StartTimeUnixNano";
    pub const END_TIME_UNIX_NANO: &str = "EndTimeUnixNano";
    pub const DURATION_NANO: &str = "DurationNano";
    pub const ROOT_SERVICE_NAME: &str = "RootServiceName";
    pub const ROOT_SPAN_NAME: &str = "RootSpanName";

    // Span fields - nested under rs.list.element.ss.list.element.Spans.list.element
    pub const SPAN_ID: &str = "rs.list.element.ss.list.element.Spans.list.element.SpanID";
    pub const SPAN_PARENT_SPAN_ID: &str =
        "rs.list.element.ss.list.element.Spans.list.element.ParentSpanID";
    pub const SPAN_PARENT_ID: &str = "rs.list.element.ss.list.element.Spans.list.element.ParentID";
    pub const SPAN_NESTED_SET_LEFT: &str =
        "rs.list.element.ss.list.element.Spans.list.element.NestedSetLeft";
    pub const SPAN_NESTED_SET_RIGHT: &str =
        "rs.list.element.ss.list.element.Spans.list.element.NestedSetRight";
    pub const SPAN_NAME: &str = "rs.list.element.ss.list.element.Spans.list.element.Name";
    pub const SPAN_KIND: &str = "rs.list.element.ss.list.element.Spans.list.element.Kind";
    pub const SPAN_START_TIME_UNIX_NANO: &str =
        "rs.list.element.ss.list.element.Spans.list.element.StartTimeUnixNano";
    pub const SPAN_DURATION_NANO: &str =
        "rs.list.element.ss.list.element.Spans.list.element.DurationNano";
    pub const SPAN_STATUS_CODE: &str =
        "rs.list.element.ss.list.element.Spans.list.element.StatusCode";
}

/// Definition levels in the parquet schema hierarchy
pub mod definition_levels {
    pub const TRACE: i16 = 0;
    pub const RESOURCE_SPANS: i16 = 1;
    pub const SCOPE_SPANS: i16 = 2;
    pub const SPAN: i16 = 3;
}

/// Result of filtering a row group
#[derive(Debug, Clone, Copy)]
pub enum FilterResult {
    Passed,
    SkippedByStatistics,
    SkippedByBloomFilter,
    SkippedByDictionary,
}
