use comfy_table::{presets::UTF8_FULL, Cell, ContentArrangement, Table};
use datafusion::arrow::array::Array;
use datafusion::arrow::record_batch::RecordBatch;
use datafusion::error::Result;

/// Extract a scalar value from an Arrow array at the given index
pub fn format_array_value(column: &dyn Array, row_idx: usize) -> String {
    use datafusion::arrow::array::*;
    use datafusion::arrow::datatypes::DataType;

    if column.is_null(row_idx) {
        return String::from("NULL");
    }

    // Extract scalar value based on data type
    match column.data_type() {
        DataType::Utf8 => column
            .as_any()
            .downcast_ref::<StringArray>()
            .map(|arr| arr.value(row_idx).to_string())
            .unwrap_or_else(|| "ERROR".to_string()),
        DataType::LargeUtf8 => column
            .as_any()
            .downcast_ref::<LargeStringArray>()
            .map(|arr| arr.value(row_idx).to_string())
            .unwrap_or_else(|| "ERROR".to_string()),
        DataType::Utf8View => column
            .as_any()
            .downcast_ref::<StringViewArray>()
            .map(|arr| arr.value(row_idx).to_string())
            .unwrap_or_else(|| "ERROR".to_string()),
        DataType::Int8 => column
            .as_any()
            .downcast_ref::<Int8Array>()
            .map(|arr| arr.value(row_idx).to_string())
            .unwrap_or_else(|| "ERROR".to_string()),
        DataType::Int16 => column
            .as_any()
            .downcast_ref::<Int16Array>()
            .map(|arr| arr.value(row_idx).to_string())
            .unwrap_or_else(|| "ERROR".to_string()),
        DataType::Int32 => column
            .as_any()
            .downcast_ref::<Int32Array>()
            .map(|arr| arr.value(row_idx).to_string())
            .unwrap_or_else(|| "ERROR".to_string()),
        DataType::Int64 => column
            .as_any()
            .downcast_ref::<Int64Array>()
            .map(|arr| arr.value(row_idx).to_string())
            .unwrap_or_else(|| "ERROR".to_string()),
        DataType::UInt8 => column
            .as_any()
            .downcast_ref::<UInt8Array>()
            .map(|arr| arr.value(row_idx).to_string())
            .unwrap_or_else(|| "ERROR".to_string()),
        DataType::UInt16 => column
            .as_any()
            .downcast_ref::<UInt16Array>()
            .map(|arr| arr.value(row_idx).to_string())
            .unwrap_or_else(|| "ERROR".to_string()),
        DataType::UInt32 => column
            .as_any()
            .downcast_ref::<UInt32Array>()
            .map(|arr| arr.value(row_idx).to_string())
            .unwrap_or_else(|| "ERROR".to_string()),
        DataType::UInt64 => column
            .as_any()
            .downcast_ref::<UInt64Array>()
            .map(|arr| arr.value(row_idx).to_string())
            .unwrap_or_else(|| "ERROR".to_string()),
        DataType::Float32 => column
            .as_any()
            .downcast_ref::<Float32Array>()
            .map(|arr| arr.value(row_idx).to_string())
            .unwrap_or_else(|| "ERROR".to_string()),
        DataType::Float64 => column
            .as_any()
            .downcast_ref::<Float64Array>()
            .map(|arr| arr.value(row_idx).to_string())
            .unwrap_or_else(|| "ERROR".to_string()),
        DataType::Boolean => column
            .as_any()
            .downcast_ref::<BooleanArray>()
            .map(|arr| arr.value(row_idx).to_string())
            .unwrap_or_else(|| "ERROR".to_string()),
        DataType::Timestamp(_, tz) => column
            .as_any()
            .downcast_ref::<TimestampNanosecondArray>()
            .map(|arr| {
                if let Some(tz_str) = tz {
                    format!("{} {}", arr.value(row_idx), tz_str)
                } else {
                    arr.value(row_idx).to_string()
                }
            })
            .or_else(|| {
                column
                    .as_any()
                    .downcast_ref::<TimestampMicrosecondArray>()
                    .map(|arr| arr.value(row_idx).to_string())
            })
            .or_else(|| {
                column
                    .as_any()
                    .downcast_ref::<TimestampMillisecondArray>()
                    .map(|arr| arr.value(row_idx).to_string())
            })
            .or_else(|| {
                column
                    .as_any()
                    .downcast_ref::<TimestampSecondArray>()
                    .map(|arr| arr.value(row_idx).to_string())
            })
            .unwrap_or_else(|| "ERROR".to_string()),
        DataType::Date32 => column
            .as_any()
            .downcast_ref::<Date32Array>()
            .map(|arr| arr.value(row_idx).to_string())
            .unwrap_or_else(|| "ERROR".to_string()),
        DataType::Date64 => column
            .as_any()
            .downcast_ref::<Date64Array>()
            .map(|arr| arr.value(row_idx).to_string())
            .unwrap_or_else(|| "ERROR".to_string()),
        DataType::Binary => column
            .as_any()
            .downcast_ref::<BinaryArray>()
            .map(|arr| {
                let bytes = arr.value(row_idx);
                format!(
                    "0x{}",
                    bytes
                        .iter()
                        .map(|b| format!("{:02x}", b))
                        .collect::<String>()
                )
            })
            .unwrap_or_else(|| "ERROR".to_string()),
        DataType::LargeBinary => column
            .as_any()
            .downcast_ref::<LargeBinaryArray>()
            .map(|arr| {
                let bytes = arr.value(row_idx);
                format!(
                    "0x{}",
                    bytes
                        .iter()
                        .map(|b| format!("{:02x}", b))
                        .collect::<String>()
                )
            })
            .unwrap_or_else(|| "ERROR".to_string()),
        DataType::BinaryView => column
            .as_any()
            .downcast_ref::<BinaryViewArray>()
            .map(|arr| {
                let bytes = arr.value(row_idx);
                format!(
                    "0x{}",
                    bytes
                        .iter()
                        .map(|b| format!("{:02x}", b))
                        .collect::<String>()
                )
            })
            .unwrap_or_else(|| "ERROR".to_string()),
        DataType::Map(_, _) => {
            // Format MapArray as JSON-like object
            format_map_array(column, row_idx)
        }
        // For other complex types (List, Struct, etc.), use Arrow's display format
        _ => format!("{:?}", column.slice(row_idx, 1))
            .trim_start_matches('[')
            .trim_end_matches(']')
            .to_string(),
    }
}

/// Format a MapArray as a JSON-like object
fn format_map_array(column: &dyn Array, row_idx: usize) -> String {
    use datafusion::arrow::array::{ListArray, MapArray, StringArray, StructArray};

    let map_array = match column.as_any().downcast_ref::<MapArray>() {
        Some(arr) => arr,
        None => return "ERROR: Not a MapArray".to_string(),
    };

    if map_array.is_null(row_idx) {
        return "NULL".to_string();
    }

    // Get the entries struct for this row
    let entries = map_array.value(row_idx);
    let entries_struct = match entries.as_any().downcast_ref::<StructArray>() {
        Some(s) => s,
        None => return "ERROR: Map entries not a struct".to_string(),
    };

    // Extract keys and values
    let keys_column = match entries_struct.column_by_name("key") {
        Some(col) => col,
        None => return "ERROR: No 'key' field in map".to_string(),
    };

    let values_column = match entries_struct.column_by_name("value") {
        Some(col) => col,
        None => return "ERROR: No 'value' field in map".to_string(),
    };

    let keys = match keys_column.as_any().downcast_ref::<StringArray>() {
        Some(arr) => arr,
        None => return "ERROR: Keys not strings".to_string(),
    };

    let values = match values_column.as_any().downcast_ref::<ListArray>() {
        Some(arr) => arr,
        None => return "ERROR: Values not lists".to_string(),
    };

    // Build JSON-like representation
    let mut result = String::from("{\n");

    for i in 0..keys.len() {
        if keys.is_null(i) {
            continue;
        }

        let key = keys.value(i);
        result.push_str(&format!("  \"{}\": ", key));

        if values.is_null(i) {
            result.push_str("null");
        } else {
            let value_list = values.value(i);
            let value_strings = match value_list.as_any().downcast_ref::<StringArray>() {
                Some(arr) => arr,
                None => {
                    result.push_str("ERROR");
                    continue;
                }
            };

            // Format as JSON array
            result.push('[');
            for j in 0..value_strings.len() {
                if j > 0 {
                    result.push_str(", ");
                }
                if value_strings.is_null(j) {
                    result.push_str("null");
                } else {
                    // Escape special characters for JSON
                    let value = value_strings.value(j);
                    let escaped = value
                        .replace('\\', "\\\\")
                        .replace('"', "\\\"")
                        .replace('\n', "\\n")
                        .replace('\r', "\\r")
                        .replace('\t', "\\t");
                    result.push_str(&format!("\"{}\"", escaped));
                }
            }
            result.push(']');
        }

        if i < keys.len() - 1 {
            result.push(',');
        }
        result.push('\n');
    }

    result.push('}');
    result
}

/// Format RecordBatches as a table with terminal width awareness
pub fn format_batches(batches: &[RecordBatch]) -> Result<String> {
    if batches.is_empty() {
        return Ok(String::new());
    }

    let schema = batches[0].schema();
    let mut table = Table::new();

    // Apply UTF8 preset for nice borders
    table.load_preset(UTF8_FULL);

    // Set content arrangement to dynamically adjust to terminal width
    table.set_content_arrangement(ContentArrangement::DynamicFullWidth);

    // Add header row
    let mut header = Vec::new();
    for field in schema.fields() {
        header.push(Cell::new(field.name()));
    }
    table.set_header(header);

    // Add data rows
    for batch in batches {
        for row_idx in 0..batch.num_rows() {
            let mut row = Vec::new();
            for col_idx in 0..batch.num_columns() {
                let column = batch.column(col_idx);
                let value = format_array_value(column.as_ref(), row_idx);
                row.push(Cell::new(value));
            }
            table.add_row(row);
        }
    }

    Ok(table.to_string())
}
