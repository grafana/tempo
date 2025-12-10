// DataFusion schema definition for Tempo vparquet4 format
//
// This schema mirrors the Go schema defined in schema.go (lines 119-256)
// Key differences from typical Parquet schemas:
// - TraceID, SpanID, ParentSpanID are Binary (not Utf8) for sort order and space efficiency
// - No logical type annotations - only physical types
// - Nested structures use Arrow Struct types
// - Repeated fields use Arrow List types

use datafusion::arrow::datatypes::{DataType, Field, Fields, Schema};
use std::sync::Arc;

/// Creates the DataFusion Schema for Tempo vparquet4 trace data.
///
/// This schema is derived from the Go struct definition in schema.go and represents
/// the physical layout of Tempo Parquet files. Note that ID fields (TraceID, SpanID)
/// are stored as Binary rather than Utf8 strings.
pub fn tempo_trace_schema() -> Schema {
    Schema::new(vec![
        // TraceID is Binary (16 bytes) for maintaining sort order in Parquet files
        Field::new("TraceID", DataType::Binary, false),
        // TraceIDText is Utf8 hex string for human readability/debugging
        Field::new("TraceIDText", DataType::Utf8, false),
        // Timestamp fields (nanoseconds since Unix epoch)
        Field::new("StartTimeUnixNano", DataType::UInt64, false),
        Field::new("EndTimeUnixNano", DataType::UInt64, false),
        Field::new("DurationNano", DataType::UInt64, false),
        // Root-level trace metadata
        Field::new("RootServiceName", DataType::Utf8, false),
        Field::new("RootSpanName", DataType::Utf8, false),
        // ServiceStats is a map[string]ServiceStats in Go
        // Represented as List of Struct(key, stats)
        Field::new(
            "ServiceStats",
            DataType::List(Arc::new(Field::new("item", service_stats_struct(), true))),
            false,
        ),
        // ResourceSpans is the main nested data structure
        Field::new(
            "ResourceSpans",
            DataType::List(Arc::new(Field::new(
                "item",
                resource_spans_struct(),
                true, // Allow null items in the list
            ))),
            true, // Allow null list
        ),
    ])
}

/// ServiceStats structure for per-service aggregated metrics
fn service_stats_struct() -> DataType {
    DataType::Struct(Fields::from(vec![
        Field::new("Key", DataType::Utf8, false),
        Field::new("SpanCount", DataType::UInt64, false),
        Field::new("ErrorCount", DataType::UInt64, false),
    ]))
}

/// ResourceSpans contains Resource and list of ScopeSpans
/// Reference: schema.go lines 230-233
fn resource_spans_struct() -> DataType {
    DataType::Struct(Fields::from(vec![
        Field::new("Resource", resource_struct(), true), // Allow null resource
        Field::new(
            "ScopeSpans",
            DataType::List(Arc::new(Field::new(
                "item",
                scope_spans_struct(),
                true, // Allow null items in the list
            ))),
            true, // Allow null list
        ),
    ]))
}

/// Resource contains attributes and dedicated columns for common resource attributes
/// Reference: schema.go lines 211-228
fn resource_struct() -> DataType {
    DataType::Struct(Fields::from(vec![
        // Dedicated columns for high-cardinality resource attributes
        Field::new("ServiceName", DataType::Utf8, true),
        Field::new("Cluster", DataType::Utf8, true),
        Field::new("Namespace", DataType::Utf8, true),
        Field::new("Pod", DataType::Utf8, true),
        Field::new("Container", DataType::Utf8, true),
        Field::new("K8sClusterName", DataType::Utf8, true),
        Field::new("K8sNamespaceName", DataType::Utf8, true),
        Field::new("K8sPodName", DataType::Utf8, true),
        Field::new("K8sContainerName", DataType::Utf8, true),
        // Generic attributes list
        Field::new(
            "Attrs",
            DataType::List(Arc::new(Field::new("item", attribute_struct(), true))),
            true, // Allow null list
        ),
        // Dropped attributes count from OTLP
        Field::new("DroppedAttributesCount", DataType::UInt32, true), // Allow null
        // Dedicated attributes storage (String01-String10)
        Field::new("DedicatedAttributes", dedicated_attributes_struct(), true), // Allow null
    ]))
}

/// ScopeSpans contains InstrumentationScope and list of Spans
/// Reference: schema.go lines 206-209
fn scope_spans_struct() -> DataType {
    DataType::Struct(Fields::from(vec![
        Field::new("Scope", instrumentation_scope_struct(), true), // Allow null scope
        Field::new(
            "Spans",
            DataType::List(Arc::new(Field::new(
                "item",
                span_struct(),
                true, // Allow null items in the list
            ))),
            true, // Allow null list
        ),
    ]))
}

/// InstrumentationScope metadata
/// Reference: schema.go lines 195-203
fn instrumentation_scope_struct() -> DataType {
    DataType::Struct(Fields::from(vec![
        Field::new("Name", DataType::Utf8, true),    // Allow null
        Field::new("Version", DataType::Utf8, true), // Allow null
        Field::new(
            "Attrs",
            DataType::List(Arc::new(Field::new("item", attribute_struct(), true))),
            true, // Allow null list
        ),
        Field::new("DroppedAttributesCount", DataType::UInt32, true), // Allow null
    ]))
}

/// Span is the core data structure containing span data
/// Reference: schema.go lines 164-193
fn span_struct() -> DataType {
    DataType::Struct(Fields::from(vec![
        // SpanID is Binary (8 bytes) to save space - half the size of string representation
        Field::new("SpanID", DataType::Binary, false),
        Field::new("ParentSpanID", DataType::Binary, false),
        // Parent reference and nested set model for tree structure
        Field::new("ParentID", DataType::Int64, false),
        Field::new("NestedSetLeft", DataType::Int64, false),
        Field::new("NestedSetRight", DataType::Int64, false),
        // Span metadata
        Field::new("Name", DataType::Utf8, false),
        Field::new("Kind", DataType::Int32, false),
        Field::new("TraceState", DataType::Utf8, false),
        // Timestamps
        Field::new("StartTimeUnixNano", DataType::UInt64, false),
        Field::new("DurationNano", DataType::UInt64, false),
        // Status
        Field::new("StatusCode", DataType::Int32, false),
        Field::new("StatusMessage", DataType::Utf8, false),
        // Dedicated HTTP attribute columns for common access patterns
        Field::new("HttpMethod", DataType::Utf8, true),
        Field::new("HttpUrl", DataType::Utf8, true),
        Field::new("HttpStatusCode", DataType::Int64, true),
        // Generic attributes
        Field::new(
            "Attrs",
            DataType::List(Arc::new(Field::new("item", attribute_struct(), true))),
            true, // Allow null list
        ),
        // Events (logs associated with span)
        Field::new(
            "Events",
            DataType::List(Arc::new(Field::new("item", event_struct(), true))),
            true, // Allow null list
        ),
        // Links to other spans
        Field::new(
            "Links",
            DataType::List(Arc::new(Field::new("item", link_struct(), true))),
            true, // Allow null list
        ),
        // OTLP metadata
        Field::new("DroppedAttributesCount", DataType::UInt32, true), // Allow null
        Field::new("DroppedEventsCount", DataType::UInt32, true),     // Allow null
        Field::new("DroppedLinksCount", DataType::UInt32, true),      // Allow null
        // Dedicated attributes storage (String01-String10)
        Field::new("DedicatedAttributes", dedicated_attributes_struct(), true), // Allow null
    ]))
}

/// Attribute structure with multiple value type columns
/// Reference: schema.go lines 122-131
///
/// Attributes are stored in a columnar format where each value type gets its own column.
/// This allows efficient filtering and reduces storage for sparse attribute types.
fn attribute_struct() -> DataType {
    DataType::Struct(Fields::from(vec![
        Field::new("Key", DataType::Utf8, false),
        // Flag indicating if the attribute is an array
        Field::new("IsArray", DataType::Boolean, false),
        // Different value type columns - each attribute uses only one
        // Lists allow multi-valued attributes
        Field::new(
            "Value",
            DataType::List(Arc::new(Field::new("item", DataType::Utf8, true))),
            true,
        ),
        Field::new(
            "ValueInt",
            DataType::List(Arc::new(Field::new("item", DataType::Int64, true))),
            true,
        ),
        Field::new(
            "ValueDouble",
            DataType::List(Arc::new(Field::new("item", DataType::Float64, true))),
            true,
        ),
        Field::new(
            "ValueBool",
            DataType::List(Arc::new(Field::new("item", DataType::Boolean, true))),
            true,
        ),
        // Unsupported types are JSON-serialized as Utf8 (see schema.go lines 335-338)
        Field::new("ValueUnsupported", DataType::Utf8, true),
    ]))
}

/// Event structure for span events/logs
/// Reference: schema.go lines 147-152
fn event_struct() -> DataType {
    DataType::Struct(Fields::from(vec![
        Field::new("Name", DataType::Utf8, true), // Allow null
        Field::new("TimeSinceStartNano", DataType::UInt64, true), // Allow null
        Field::new(
            "Attrs",
            DataType::List(Arc::new(Field::new("item", attribute_struct(), true))),
            true, // Allow null list
        ),
        Field::new("DroppedAttributesCount", DataType::UInt32, true), // Allow null
    ]))
}

/// Link structure for span-to-span relationships
/// Reference: schema.go lines 154-160
fn link_struct() -> DataType {
    DataType::Struct(Fields::from(vec![
        // TraceID and SpanID are Binary for consistency with root trace IDs
        Field::new("TraceID", DataType::Binary, true), // Allow null
        Field::new("SpanID", DataType::Binary, true),  // Allow null
        Field::new("TraceState", DataType::Utf8, true), // Allow null
        Field::new(
            "Attrs",
            DataType::List(Arc::new(Field::new("item", attribute_struct(), true))),
            true, // Allow null list
        ),
        Field::new("DroppedAttributesCount", DataType::UInt32, true), // Allow null
    ]))
}

/// Dedicated attributes structure for storing additional string attributes
/// String01-String10 provide storage for commonly used attribute values
fn dedicated_attributes_struct() -> DataType {
    DataType::Struct(Fields::from(vec![
        Field::new("String01", DataType::Utf8, true),
        Field::new("String02", DataType::Utf8, true),
        Field::new("String03", DataType::Utf8, true),
        Field::new("String04", DataType::Utf8, true),
        Field::new("String05", DataType::Utf8, true),
        Field::new("String06", DataType::Utf8, true),
        Field::new("String07", DataType::Utf8, true),
        Field::new("String08", DataType::Utf8, true),
        Field::new("String09", DataType::Utf8, true),
        Field::new("String10", DataType::Utf8, true),
    ]))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_schema_creation() {
        let schema = tempo_trace_schema();

        // Verify top-level fields
        assert_eq!(schema.fields().len(), 9);

        // Verify TraceID is Binary (not Utf8)
        assert_eq!(schema.field(0).name(), "TraceID");
        assert!(matches!(schema.field(0).data_type(), DataType::Binary));

        // Verify TraceIDText is Utf8
        assert_eq!(schema.field(1).name(), "TraceIDText");
        assert!(matches!(schema.field(1).data_type(), DataType::Utf8));

        // Verify timestamps are UInt64
        assert_eq!(schema.field(2).name(), "StartTimeUnixNano");
        assert!(matches!(schema.field(2).data_type(), DataType::UInt64));

        // Verify ResourceSpans is a List
        assert_eq!(schema.field(8).name(), "ResourceSpans");
        assert!(matches!(schema.field(8).data_type(), DataType::List(_)));
    }

    #[test]
    fn test_binary_id_fields() {
        let schema = tempo_trace_schema();

        // TraceID should be Binary
        let trace_id_field = schema.field_with_name("TraceID").unwrap();
        assert!(matches!(trace_id_field.data_type(), DataType::Binary));

        // SpanID within span should also be Binary
        // (This is nested deep, so we just verify the top-level structure exists)
        let resource_spans_field = schema.field_with_name("ResourceSpans").unwrap();
        assert!(matches!(
            resource_spans_field.data_type(),
            DataType::List(_)
        ));
    }
}
