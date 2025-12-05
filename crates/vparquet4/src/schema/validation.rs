//! Schema validation for vParquet4 files

use crate::error::{Result, VParquet4Error};
use crate::schema::field_paths::trace;
use arrow::datatypes::{DataType, Schema};

/// Validates that a Parquet file schema matches the expected vParquet4 schema
pub fn validate_schema(schema: &Schema) -> Result<()> {
    // Check for required top-level columns
    validate_column(schema, trace::TRACE_ID, &DataType::Binary, false)?;
    validate_column(schema, trace::TRACE_ID_TEXT, &DataType::Utf8, false)?;
    validate_column(schema, trace::START_TIME_UNIX_NANO, &DataType::UInt64, false)?;
    validate_column(schema, trace::END_TIME_UNIX_NANO, &DataType::UInt64, false)?;
    validate_column(schema, trace::DURATION_NANO, &DataType::UInt64, false)?;
    validate_column(schema, trace::ROOT_SERVICE_NAME, &DataType::Utf8, false)?;
    validate_column(schema, trace::ROOT_SPAN_NAME, &DataType::Utf8, false)?;

    // ServiceStats and ResourceSpans are List types (more complex validation)
    validate_column_exists(schema, trace::SERVICE_STATS)?;
    validate_column_exists(schema, trace::RESOURCE_SPANS)?;

    Ok(())
}

/// Validates that a column exists and has the expected type and nullability
fn validate_column(
    schema: &Schema,
    column_name: &str,
    expected_type: &DataType,
    nullable: bool,
) -> Result<()> {
    let field = schema
        .field_with_name(column_name)
        .map_err(|_| VParquet4Error::MissingColumn(column_name.to_string()))?;

    // Check data type
    if !types_match(field.data_type(), expected_type) {
        return Err(VParquet4Error::InvalidColumnType {
            column: column_name.to_string(),
            expected: format!("{:?}", expected_type),
            actual: format!("{:?}", field.data_type()),
        });
    }

    // Check nullability
    if field.is_nullable() != nullable {
        return Err(VParquet4Error::InvalidSchema(format!(
            "Column {} has incorrect nullability: expected {}, got {}",
            column_name,
            nullable,
            field.is_nullable()
        )));
    }

    Ok(())
}

/// Validates that a column exists (without checking type)
fn validate_column_exists(schema: &Schema, column_name: &str) -> Result<()> {
    schema
        .field_with_name(column_name)
        .map_err(|_| VParquet4Error::MissingColumn(column_name.to_string()))?;
    Ok(())
}

/// Checks if two Arrow data types match (handling nested types)
fn types_match(actual: &DataType, expected: &DataType) -> bool {
    match (actual, expected) {
        // Exact match
        (a, b) if a == b => true,
        // List types - just check that both are lists (nested type checking is complex)
        (DataType::List(_), DataType::List(_)) => true,
        (DataType::LargeList(_), DataType::LargeList(_)) => true,
        // Struct types - just check that both are structs
        (DataType::Struct(_), DataType::Struct(_)) => true,
        // Otherwise, no match
        _ => false,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use arrow::datatypes::Field;

    #[test]
    fn test_validate_minimal_schema() {
        let schema = Schema::new(vec![
            Field::new(trace::TRACE_ID, DataType::Binary, false),
            Field::new(trace::TRACE_ID_TEXT, DataType::Utf8, false),
            Field::new(trace::START_TIME_UNIX_NANO, DataType::UInt64, false),
            Field::new(trace::END_TIME_UNIX_NANO, DataType::UInt64, false),
            Field::new(trace::DURATION_NANO, DataType::UInt64, false),
            Field::new(trace::ROOT_SERVICE_NAME, DataType::Utf8, false),
            Field::new(trace::ROOT_SPAN_NAME, DataType::Utf8, false),
            Field::new(
                trace::SERVICE_STATS,
                DataType::List(Box::new(Field::new("item", DataType::Utf8, true)).into()),
                false,
            ),
            Field::new(
                trace::RESOURCE_SPANS,
                DataType::List(Box::new(Field::new("item", DataType::Utf8, true)).into()),
                true,
            ),
        ]);

        assert!(validate_schema(&schema).is_ok());
    }

    #[test]
    fn test_validate_missing_column() {
        let schema = Schema::new(vec![
            Field::new(trace::TRACE_ID, DataType::Binary, false),
            // Missing other required columns
        ]);

        let result = validate_schema(&schema);
        assert!(result.is_err());
        assert!(matches!(
            result.unwrap_err(),
            VParquet4Error::MissingColumn(_)
        ));
    }

    #[test]
    fn test_validate_wrong_type() {
        let schema = Schema::new(vec![
            Field::new(trace::TRACE_ID, DataType::Utf8, false), // Wrong type (should be Binary)
            Field::new(trace::TRACE_ID_TEXT, DataType::Utf8, false),
            Field::new(trace::START_TIME_UNIX_NANO, DataType::UInt64, false),
            Field::new(trace::END_TIME_UNIX_NANO, DataType::UInt64, false),
            Field::new(trace::DURATION_NANO, DataType::UInt64, false),
            Field::new(trace::ROOT_SERVICE_NAME, DataType::Utf8, false),
            Field::new(trace::ROOT_SPAN_NAME, DataType::Utf8, false),
            Field::new(
                trace::SERVICE_STATS,
                DataType::List(Box::new(Field::new("item", DataType::Utf8, true)).into()),
                false,
            ),
            Field::new(
                trace::RESOURCE_SPANS,
                DataType::List(Box::new(Field::new("item", DataType::Utf8, true)).into()),
                true,
            ),
        ]);

        let result = validate_schema(&schema);
        assert!(result.is_err());
        assert!(matches!(
            result.unwrap_err(),
            VParquet4Error::InvalidColumnType { .. }
        ));
    }
}
