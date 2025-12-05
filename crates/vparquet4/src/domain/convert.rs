//! Conversion utilities for parsing Arrow data into OTLP types
//!
//! This module provides functions to convert from Arrow RecordBatch (Parquet)
//! representation to OpenTelemetry Protocol (OTLP) types.

use crate::domain::otlp::{
    common::v1::{any_value, AnyValue, InstrumentationScope, KeyValue},
    resource::v1::Resource,
    trace::v1::{span, ResourceSpans, ScopeSpans, Span, Status},
};
use crate::error::{Result, VParquet4Error};
use arrow::array::{
    Array, BinaryArray, BooleanArray, Float64Array, Int32Array, Int64Array, ListArray,
    StringArray, StructArray, UInt32Array, UInt64Array,
};
use arrow::record_batch::RecordBatch;
use std::sync::Arc;

/// Parse a ResourceSpans list from a RecordBatch
///
/// Each row in the RecordBatch represents a trace, and the ResourceSpans field
/// contains the nested structure of resources, scopes, and spans.
pub fn parse_resource_spans_from_batch(
    batch: &RecordBatch,
    row_index: usize,
) -> Result<Vec<ResourceSpans>> {
    let rs_column = batch
        .column_by_name("rs")
        .or_else(|| batch.column_by_name("ResourceSpans"))
        .ok_or_else(|| {
            VParquet4Error::MissingColumn("ResourceSpans".to_string())
        })?;

    let rs_list = rs_column
        .as_any()
        .downcast_ref::<ListArray>()
        .ok_or_else(|| {
            VParquet4Error::ConversionError("ResourceSpans is not a ListArray".to_string())
        })?;

    parse_resource_spans_list(rs_list, row_index)
}

/// Parse a list of ResourceSpans from Arrow ListArray
fn parse_resource_spans_list(list_array: &ListArray, index: usize) -> Result<Vec<ResourceSpans>> {
    if list_array.is_null(index) {
        return Ok(Vec::new());
    }

    let values = list_array.value(index);
    let struct_array = values
        .as_any()
        .downcast_ref::<StructArray>()
        .ok_or_else(|| {
            VParquet4Error::ConversionError("Expected StructArray in ResourceSpans list".to_string())
        })?;

    let mut resource_spans_vec = Vec::new();

    let resource_column = struct_array.column_by_name("Resource");
    let scope_spans_column = struct_array.column_by_name("ScopeSpans");

    for i in 0..struct_array.len() {
        let resource = if let Some(col) = resource_column {
            if let Some(struct_arr) = col.as_any().downcast_ref::<StructArray>() {
                if !struct_arr.is_null(i) {
                    Some(parse_resource(struct_arr, i)?)
                } else {
                    None
                }
            } else {
                None
            }
        } else {
            None
        };

        let scope_spans = if let Some(col) = scope_spans_column {
            if let Some(list_arr) = col.as_any().downcast_ref::<ListArray>() {
                parse_scope_spans_list(list_arr, i)?
            } else {
                Vec::new()
            }
        } else {
            Vec::new()
        };

        resource_spans_vec.push(ResourceSpans {
            resource,
            scope_spans,
            schema_url: String::new(),
        });
    }

    Ok(resource_spans_vec)
}

/// Parse a Resource from Arrow StructArray
fn parse_resource(struct_array: &StructArray, index: usize) -> Result<Resource> {
    let mut attributes = Vec::new();

    // Parse dedicated columns
    let service_name_col = struct_array.column_by_name("ServiceName");
    if let Some(col) = service_name_col {
        if let Some(arr) = col.as_any().downcast_ref::<StringArray>() {
            if !arr.is_null(index) {
                attributes.push(KeyValue {
                    key: "service.name".to_string(),
                    value: Some(AnyValue {
                        value: Some(any_value::Value::StringValue(arr.value(index).to_string())),
                    }),
                });
            }
        }
    }

    // Parse generic attributes
    if let Some(attrs_col) = struct_array.column_by_name("Attrs") {
        if let Some(list_arr) = attrs_col.as_any().downcast_ref::<ListArray>() {
            attributes.extend(parse_key_values(list_arr, index)?);
        }
    }

    // Parse dedicated attributes (String01-String10)
    if let Some(dedicated_col) = struct_array.column_by_name("DedicatedAttributes") {
        if let Some(dedicated_arr) = dedicated_col.as_any().downcast_ref::<StructArray>() {
            if !dedicated_arr.is_null(index) {
                attributes.extend(parse_dedicated_attributes(dedicated_arr, index)?);
            }
        }
    }

    let dropped_attributes_count = struct_array
        .column_by_name("DroppedAttributesCount")
        .and_then(|col| {
            col.as_any()
                .downcast_ref::<UInt32Array>()
                .and_then(|arr| {
                    if !arr.is_null(index) {
                        Some(arr.value(index))
                    } else {
                        None
                    }
                })
        })
        .unwrap_or(0);

    Ok(Resource {
        attributes,
        dropped_attributes_count,
    })
}

/// Parse a list of ScopeSpans from Arrow ListArray
fn parse_scope_spans_list(list_array: &ListArray, index: usize) -> Result<Vec<ScopeSpans>> {
    if list_array.is_null(index) {
        return Ok(Vec::new());
    }

    let values = list_array.value(index);
    let struct_array = values
        .as_any()
        .downcast_ref::<StructArray>()
        .ok_or_else(|| {
            VParquet4Error::ConversionError("Expected StructArray in ScopeSpans list".to_string())
        })?;

    let mut scope_spans_vec = Vec::new();

    let scope_column = struct_array.column_by_name("Scope");
    let spans_column = struct_array.column_by_name("Spans");

    for i in 0..struct_array.len() {
        let scope = if let Some(col) = scope_column {
            if let Some(scope_arr) = col.as_any().downcast_ref::<StructArray>() {
                if !scope_arr.is_null(i) {
                    Some(parse_instrumentation_scope(scope_arr, i)?)
                } else {
                    None
                }
            } else {
                None
            }
        } else {
            None
        };

        let spans = if let Some(col) = spans_column {
            if let Some(list_arr) = col.as_any().downcast_ref::<ListArray>() {
                parse_spans_list(list_arr, i)?
            } else {
                Vec::new()
            }
        } else {
            Vec::new()
        };

        scope_spans_vec.push(ScopeSpans {
            scope,
            spans,
            schema_url: String::new(),
        });
    }

    Ok(scope_spans_vec)
}

/// Parse an InstrumentationScope from Arrow StructArray
fn parse_instrumentation_scope(
    struct_array: &StructArray,
    index: usize,
) -> Result<InstrumentationScope> {
    let name = struct_array
        .column_by_name("Name")
        .and_then(|col| {
            col.as_any()
                .downcast_ref::<StringArray>()
                .and_then(|arr| {
                    if !arr.is_null(index) {
                        Some(arr.value(index).to_string())
                    } else {
                        None
                    }
                })
        })
        .unwrap_or_default();

    let version = struct_array
        .column_by_name("Version")
        .and_then(|col| {
            col.as_any()
                .downcast_ref::<StringArray>()
                .and_then(|arr| {
                    if !arr.is_null(index) {
                        Some(arr.value(index).to_string())
                    } else {
                        None
                    }
                })
        })
        .unwrap_or_default();

    let attributes = if let Some(attrs_col) = struct_array.column_by_name("Attrs") {
        if let Some(list_arr) = attrs_col.as_any().downcast_ref::<ListArray>() {
            parse_key_values(list_arr, index)?
        } else {
            Vec::new()
        }
    } else {
        Vec::new()
    };

    let dropped_attributes_count = struct_array
        .column_by_name("DroppedAttributesCount")
        .and_then(|col| {
            col.as_any()
                .downcast_ref::<UInt32Array>()
                .and_then(|arr| {
                    if !arr.is_null(index) {
                        Some(arr.value(index))
                    } else {
                        None
                    }
                })
        })
        .unwrap_or(0);

    Ok(InstrumentationScope {
        name,
        version,
        attributes,
        dropped_attributes_count,
    })
}

/// Parse a list of Spans from Arrow ListArray
fn parse_spans_list(list_array: &ListArray, index: usize) -> Result<Vec<Span>> {
    if list_array.is_null(index) {
        return Ok(Vec::new());
    }

    let values = list_array.value(index);
    let struct_array = values
        .as_any()
        .downcast_ref::<StructArray>()
        .ok_or_else(|| {
            VParquet4Error::ConversionError("Expected StructArray in Spans list".to_string())
        })?;

    let mut spans = Vec::new();

    // Extract columns
    let span_id_col = struct_array.column_by_name("SpanID");
    let parent_span_id_col = struct_array.column_by_name("ParentSpanID");
    let name_col = struct_array.column_by_name("Name");
    let kind_col = struct_array.column_by_name("Kind");
    let start_time_col = struct_array.column_by_name("StartTimeUnixNano");
    let duration_col = struct_array.column_by_name("DurationNano");
    let status_code_col = struct_array.column_by_name("StatusCode");
    let status_message_col = struct_array.column_by_name("StatusMessage");
    let attrs_col = struct_array.column_by_name("Attrs");
    let events_col = struct_array.column_by_name("Events");
    let links_col = struct_array.column_by_name("Links");

    for i in 0..struct_array.len() {
        let span_id = parse_binary_field(span_id_col, i).unwrap_or_default();
        let parent_span_id = parse_binary_field(parent_span_id_col, i).unwrap_or_default();

        // Note: vParquet4 stores TraceID at the top level, not in each span
        // For now, we'll use an empty trace_id and populate it from the parent trace
        let trace_id = Vec::new();

        let name = parse_string_field(name_col, i).unwrap_or_default();

        let kind = parse_i32_field(kind_col, i).unwrap_or(0);

        let start_time_unix_nano = parse_u64_field(start_time_col, i).unwrap_or(0);

        let end_time_unix_nano = if let Some(duration) = parse_u64_field(duration_col, i) {
            start_time_unix_nano + duration
        } else {
            start_time_unix_nano
        };

        let attributes = if let Some(col) = attrs_col {
            if let Some(list_arr) = col.as_any().downcast_ref::<ListArray>() {
                parse_key_values(list_arr, i)?
            } else {
                Vec::new()
            }
        } else {
            Vec::new()
        };

        let events = if let Some(col) = events_col {
            if let Some(list_arr) = col.as_any().downcast_ref::<ListArray>() {
                parse_events_list(list_arr, i, start_time_unix_nano)?
            } else {
                Vec::new()
            }
        } else {
            Vec::new()
        };

        let links = if let Some(col) = links_col {
            if let Some(list_arr) = col.as_any().downcast_ref::<ListArray>() {
                parse_links_list(list_arr, i)?
            } else {
                Vec::new()
            }
        } else {
            Vec::new()
        };

        let status_code = parse_i32_field(status_code_col, i).unwrap_or(0);
        let status_message = parse_string_field(status_message_col, i).unwrap_or_default();

        let status = Some(Status {
            message: status_message,
            code: status_code,
        });

        spans.push(Span {
            trace_id,
            span_id,
            trace_state: String::new(),
            parent_span_id,
            flags: 0,
            name,
            kind,
            start_time_unix_nano,
            end_time_unix_nano,
            attributes,
            dropped_attributes_count: 0,
            events,
            dropped_events_count: 0,
            links,
            dropped_links_count: 0,
            status,
        });
    }

    Ok(spans)
}

/// Parse KeyValue attributes from Arrow ListArray
fn parse_key_values(list_array: &ListArray, index: usize) -> Result<Vec<KeyValue>> {
    if list_array.is_null(index) {
        return Ok(Vec::new());
    }

    let values = list_array.value(index);
    let struct_array = values
        .as_any()
        .downcast_ref::<StructArray>()
        .ok_or_else(|| {
            VParquet4Error::ConversionError("Expected StructArray in attributes list".to_string())
        })?;

    let mut key_values = Vec::new();

    let key_col = struct_array.column_by_name("Key");
    let value_str_col = struct_array.column_by_name("Value");
    let value_int_col = struct_array.column_by_name("ValueInt");
    let value_double_col = struct_array.column_by_name("ValueDouble");
    let value_bool_col = struct_array.column_by_name("ValueBool");

    for i in 0..struct_array.len() {
        let key = if let Some(col) = key_col {
            parse_string_field(Some(col), i).unwrap_or_default()
        } else {
            continue;
        };

        // Try each value type in order
        let value = parse_attribute_value(
            value_str_col,
            value_int_col,
            value_double_col,
            value_bool_col,
            i,
        )?;

        if let Some(v) = value {
            key_values.push(KeyValue {
                key,
                value: Some(v),
            });
        }
    }

    Ok(key_values)
}

/// Parse attribute value from multiple possible type columns
fn parse_attribute_value(
    value_str_col: Option<&Arc<dyn Array>>,
    value_int_col: Option<&Arc<dyn Array>>,
    value_double_col: Option<&Arc<dyn Array>>,
    value_bool_col: Option<&Arc<dyn Array>>,
    index: usize,
) -> Result<Option<AnyValue>> {
    // Try string value
    if let Some(col) = value_str_col {
        if let Some(list_arr) = col.as_any().downcast_ref::<ListArray>() {
            if !list_arr.is_null(index) {
                let values = list_arr.value(index);
                if let Some(str_arr) = values.as_any().downcast_ref::<StringArray>() {
                    if str_arr.len() == 1 && !str_arr.is_null(0) {
                        return Ok(Some(AnyValue {
                            value: Some(any_value::Value::StringValue(
                                str_arr.value(0).to_string(),
                            )),
                        }));
                    }
                }
            }
        }
    }

    // Try int value
    if let Some(col) = value_int_col {
        if let Some(list_arr) = col.as_any().downcast_ref::<ListArray>() {
            if !list_arr.is_null(index) {
                let values = list_arr.value(index);
                if let Some(int_arr) = values.as_any().downcast_ref::<Int64Array>() {
                    if int_arr.len() == 1 && !int_arr.is_null(0) {
                        return Ok(Some(AnyValue {
                            value: Some(any_value::Value::IntValue(int_arr.value(0))),
                        }));
                    }
                }
            }
        }
    }

    // Try double value
    if let Some(col) = value_double_col {
        if let Some(list_arr) = col.as_any().downcast_ref::<ListArray>() {
            if !list_arr.is_null(index) {
                let values = list_arr.value(index);
                if let Some(double_arr) = values.as_any().downcast_ref::<Float64Array>() {
                    if double_arr.len() == 1 && !double_arr.is_null(0) {
                        return Ok(Some(AnyValue {
                            value: Some(any_value::Value::DoubleValue(double_arr.value(0))),
                        }));
                    }
                }
            }
        }
    }

    // Try bool value
    if let Some(col) = value_bool_col {
        if let Some(list_arr) = col.as_any().downcast_ref::<ListArray>() {
            if !list_arr.is_null(index) {
                let values = list_arr.value(index);
                if let Some(bool_arr) = values.as_any().downcast_ref::<BooleanArray>() {
                    if bool_arr.len() == 1 && !bool_arr.is_null(0) {
                        return Ok(Some(AnyValue {
                            value: Some(any_value::Value::BoolValue(bool_arr.value(0))),
                        }));
                    }
                }
            }
        }
    }

    Ok(None)
}

/// Parse dedicated attributes from DedicatedAttributes struct
fn parse_dedicated_attributes(
    struct_array: &StructArray,
    index: usize,
) -> Result<Vec<KeyValue>> {
    let mut attributes = Vec::new();

    for i in 1..=10 {
        let field_name = format!("String{:02}", i);
        if let Some(col) = struct_array.column_by_name(&field_name) {
            if let Some(str_arr) = col.as_any().downcast_ref::<StringArray>() {
                if !str_arr.is_null(index) {
                    let value = str_arr.value(index).to_string();
                    // The key would need to come from metadata or a mapping
                    // For now, we'll use a generic key
                    attributes.push(KeyValue {
                        key: format!("dedicated.{}", field_name.to_lowercase()),
                        value: Some(AnyValue {
                            value: Some(any_value::Value::StringValue(value)),
                        }),
                    });
                }
            }
        }
    }

    Ok(attributes)
}

/// Parse events list from Arrow ListArray
fn parse_events_list(
    list_array: &ListArray,
    index: usize,
    span_start_time: u64,
) -> Result<Vec<span::Event>> {
    if list_array.is_null(index) {
        return Ok(Vec::new());
    }

    let values = list_array.value(index);
    let struct_array = values
        .as_any()
        .downcast_ref::<StructArray>()
        .ok_or_else(|| {
            VParquet4Error::ConversionError("Expected StructArray in events list".to_string())
        })?;

    let mut events = Vec::new();

    let name_col = struct_array.column_by_name("Name");
    let time_col = struct_array.column_by_name("TimeSinceStartNano");
    let attrs_col = struct_array.column_by_name("Attrs");

    for i in 0..struct_array.len() {
        let name = parse_string_field(name_col, i).unwrap_or_default();

        let time_unix_nano = if let Some(offset) = parse_u64_field(time_col, i) {
            span_start_time + offset
        } else {
            span_start_time
        };

        let attributes = if let Some(col) = attrs_col {
            if let Some(list_arr) = col.as_any().downcast_ref::<ListArray>() {
                parse_key_values(list_arr, i)?
            } else {
                Vec::new()
            }
        } else {
            Vec::new()
        };

        events.push(span::Event {
            time_unix_nano,
            name,
            attributes,
            dropped_attributes_count: 0,
        });
    }

    Ok(events)
}

/// Parse links list from Arrow ListArray
fn parse_links_list(list_array: &ListArray, index: usize) -> Result<Vec<span::Link>> {
    if list_array.is_null(index) {
        return Ok(Vec::new());
    }

    let values = list_array.value(index);
    let struct_array = values
        .as_any()
        .downcast_ref::<StructArray>()
        .ok_or_else(|| {
            VParquet4Error::ConversionError("Expected StructArray in links list".to_string())
        })?;

    let mut links = Vec::new();

    let trace_id_col = struct_array.column_by_name("TraceID");
    let span_id_col = struct_array.column_by_name("SpanID");
    let trace_state_col = struct_array.column_by_name("TraceState");
    let attrs_col = struct_array.column_by_name("Attrs");

    for i in 0..struct_array.len() {
        let trace_id = parse_binary_field(trace_id_col, i).unwrap_or_default();
        let span_id = parse_binary_field(span_id_col, i).unwrap_or_default();
        let trace_state = parse_string_field(trace_state_col, i).unwrap_or_default();

        let attributes = if let Some(col) = attrs_col {
            if let Some(list_arr) = col.as_any().downcast_ref::<ListArray>() {
                parse_key_values(list_arr, i)?
            } else {
                Vec::new()
            }
        } else {
            Vec::new()
        };

        links.push(span::Link {
            trace_id,
            span_id,
            trace_state,
            attributes,
            dropped_attributes_count: 0,
            flags: 0,
        });
    }

    Ok(links)
}

// Helper functions for parsing primitive fields

fn parse_binary_field(column: Option<&Arc<dyn Array>>, index: usize) -> Option<Vec<u8>> {
    column.and_then(|col| {
        col.as_any()
            .downcast_ref::<BinaryArray>()
            .and_then(|arr| {
                if !arr.is_null(index) {
                    Some(arr.value(index).to_vec())
                } else {
                    None
                }
            })
    })
}

fn parse_string_field(column: Option<&Arc<dyn Array>>, index: usize) -> Option<String> {
    column.and_then(|col| {
        col.as_any()
            .downcast_ref::<StringArray>()
            .and_then(|arr| {
                if !arr.is_null(index) {
                    Some(arr.value(index).to_string())
                } else {
                    None
                }
            })
    })
}

fn parse_i32_field(column: Option<&Arc<dyn Array>>, index: usize) -> Option<i32> {
    column.and_then(|col| {
        col.as_any()
            .downcast_ref::<Int32Array>()
            .and_then(|arr| if !arr.is_null(index) { Some(arr.value(index)) } else { None })
    })
}

fn parse_u64_field(column: Option<&Arc<dyn Array>>, index: usize) -> Option<u64> {
    column.and_then(|col| {
        col.as_any()
            .downcast_ref::<UInt64Array>()
            .and_then(|arr| if !arr.is_null(index) { Some(arr.value(index)) } else { None })
    })
}
