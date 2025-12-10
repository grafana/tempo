use datafusion::arrow::array::{
    ArrayRef, BinaryBuilder, Int32Builder, Int64Builder, RecordBatch, StringBuilder, UInt64Builder,
};
use datafusion::arrow::datatypes::{DataType, Field, Schema, SchemaRef};
use datafusion::error::Result;
use std::sync::Arc;
use vparquet4::Spanset;

/// Schema for flat span output
/// Returns a flattened schema where each row represents a single span
pub fn flat_span_schema() -> SchemaRef {
    Arc::new(Schema::new(vec![
        Field::new("trace_id", DataType::Binary, false),
        Field::new("span_id", DataType::Binary, false),
        Field::new("parent_span_id", DataType::Binary, false),
        Field::new("name", DataType::Utf8, false),
        Field::new("kind", DataType::Int64, false),
        Field::new("start_time_unix_nano", DataType::UInt64, false),
        Field::new("duration_nano", DataType::UInt64, false),
        Field::new("status_code", DataType::Int64, false),
        Field::new("parent_id", DataType::Int32, false),
        Field::new("nested_set_left", DataType::Int32, false),
        Field::new("nested_set_right", DataType::Int32, false),
    ]))
}

/// Convert a vector of Spansets to a RecordBatch
/// Each span becomes a row in the output batch
pub fn spansets_to_record_batch(spansets: Vec<Spanset>) -> Result<RecordBatch> {
    let schema = flat_span_schema();

    // Pre-calculate total span count for capacity optimization
    let total_spans: usize = spansets.iter().map(|s| s.spans.len()).sum();

    // Build arrays with pre-allocated capacity
    let mut trace_id_builder = BinaryBuilder::with_capacity(total_spans, total_spans * 16);
    let mut span_id_builder = BinaryBuilder::with_capacity(total_spans, total_spans * 8);
    let mut parent_span_id_builder = BinaryBuilder::with_capacity(total_spans, total_spans * 8);
    let mut name_builder = StringBuilder::with_capacity(total_spans, total_spans * 32);
    let mut kind_builder = Int64Builder::with_capacity(total_spans);
    let mut start_time_builder = UInt64Builder::with_capacity(total_spans);
    let mut duration_builder = UInt64Builder::with_capacity(total_spans);
    let mut status_code_builder = Int64Builder::with_capacity(total_spans);
    let mut parent_id_builder = Int32Builder::with_capacity(total_spans);
    let mut nested_set_left_builder = Int32Builder::with_capacity(total_spans);
    let mut nested_set_right_builder = Int32Builder::with_capacity(total_spans);

    // Convert each spanset to rows
    for spanset in spansets {
        for span in spanset.spans {
            trace_id_builder.append_value(&spanset.trace_id);
            span_id_builder.append_value(&span.span_id);
            parent_span_id_builder.append_value(&span.parent_span_id);
            name_builder.append_value(&span.name);
            kind_builder.append_value(span.kind);
            start_time_builder.append_value(span.start_time_unix_nano);
            duration_builder.append_value(span.duration_nano);
            status_code_builder.append_value(span.status_code);
            parent_id_builder.append_value(span.parent_id);
            nested_set_left_builder.append_value(span.nested_set_left);
            nested_set_right_builder.append_value(span.nested_set_right);
        }
    }

    // Build the final arrays
    let columns: Vec<ArrayRef> = vec![
        Arc::new(trace_id_builder.finish()),
        Arc::new(span_id_builder.finish()),
        Arc::new(parent_span_id_builder.finish()),
        Arc::new(name_builder.finish()),
        Arc::new(kind_builder.finish()),
        Arc::new(start_time_builder.finish()),
        Arc::new(duration_builder.finish()),
        Arc::new(status_code_builder.finish()),
        Arc::new(parent_id_builder.finish()),
        Arc::new(nested_set_left_builder.finish()),
        Arc::new(nested_set_right_builder.finish()),
    ];

    RecordBatch::try_new(schema, columns)
        .map_err(|e| datafusion::error::DataFusionError::ArrowError(Box::new(e), None))
}

#[cfg(test)]
mod tests {
    use super::*;
    use bytes::Bytes;
    use vparquet4::{Span, Spanset};

    fn create_test_span(name: &str, span_id: u8) -> Span {
        Span {
            span_id: Bytes::from(vec![span_id; 8]),
            parent_span_id: Bytes::from(vec![0u8; 8]),
            parent_id: -1,
            nested_set_left: 1,
            nested_set_right: 2,
            name: name.to_string(),
            kind: 1,
            start_time_unix_nano: 1000,
            duration_nano: 500,
            status_code: 0,
        }
    }

    #[test]
    fn test_empty_spansets() {
        let batch = spansets_to_record_batch(vec![]).unwrap();
        assert_eq!(batch.num_rows(), 0);
        assert_eq!(batch.num_columns(), 11);
    }

    #[test]
    fn test_single_spanset_single_span() {
        let spanset = Spanset {
            trace_id: Bytes::from(vec![1u8; 16]),
            spans: vec![create_test_span("test_span", 2)],
        };

        let batch = spansets_to_record_batch(vec![spanset]).unwrap();
        assert_eq!(batch.num_rows(), 1);
        assert_eq!(batch.num_columns(), 11);

        // Verify the schema fields
        let schema = batch.schema();
        assert!(schema.field_with_name("trace_id").is_ok());
        assert!(schema.field_with_name("span_id").is_ok());
        assert!(schema.field_with_name("name").is_ok());
    }

    #[test]
    fn test_multiple_spansets() {
        let spansets = vec![
            Spanset {
                trace_id: Bytes::from(vec![1u8; 16]),
                spans: vec![create_test_span("span1", 1), create_test_span("span2", 2)],
            },
            Spanset {
                trace_id: Bytes::from(vec![2u8; 16]),
                spans: vec![create_test_span("span3", 3)],
            },
        ];

        let batch = spansets_to_record_batch(spansets).unwrap();
        assert_eq!(batch.num_rows(), 3);
        assert_eq!(batch.num_columns(), 11);
    }

    #[test]
    fn test_schema_field_types() {
        let schema = flat_span_schema();

        assert_eq!(
            schema.field_with_name("trace_id").unwrap().data_type(),
            &DataType::Binary
        );
        assert_eq!(
            schema.field_with_name("span_id").unwrap().data_type(),
            &DataType::Binary
        );
        assert_eq!(
            schema.field_with_name("name").unwrap().data_type(),
            &DataType::Utf8
        );
        assert_eq!(
            schema.field_with_name("kind").unwrap().data_type(),
            &DataType::Int64
        );
        assert_eq!(
            schema
                .field_with_name("start_time_unix_nano")
                .unwrap()
                .data_type(),
            &DataType::UInt64
        );
    }

    #[test]
    fn test_empty_spanset_no_spans() {
        let spansets = vec![Spanset {
            trace_id: Bytes::from(vec![1u8; 16]),
            spans: vec![],
        }];

        let batch = spansets_to_record_batch(spansets).unwrap();
        assert_eq!(batch.num_rows(), 0);
    }
}
