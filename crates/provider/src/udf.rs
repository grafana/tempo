use datafusion::arrow::array::{
    Array, ArrayRef, BooleanArray, Float64Array, Int64Array, ListArray, ListBuilder, MapArray,
    StringArray, StringBuilder, StructArray,
};
use datafusion::arrow::buffer::OffsetBuffer;
use datafusion::arrow::datatypes::{DataType, Field};
use datafusion::error::{DataFusionError, Result};
use datafusion::logical_expr::{ColumnarValue, ScalarUDF, Volatility};
use datafusion::prelude::*;
use std::sync::Arc;
use tracing::error;

/// Convert Attrs array structure to a Map<String, List<String>>
///
/// The Attrs structure from Tempo has the following fields:
/// - Key: String
/// - IsArray: Boolean (unused)
/// - Value: List<String>
/// - ValueInt: List<Int64>
/// - ValueDouble: List<Float64>
/// - ValueBool: List<Boolean>
/// - ValueUnsupported: String (nullable)
///
/// This function aggregates all values and converts them to strings for each key.
/// If keys are duplicated, all values are collected into the string array.
///
/// This is a wrapper function that logs errors before returning them to DataFusion.
pub fn attrs_to_map(args: &[ColumnarValue]) -> Result<ColumnarValue> {
    attrs_to_map_impl(args).map_err(|e| {
        error!("attrs_to_map UDF error: {}", e);
        e
    })
}

/// Internal implementation of attrs_to_map
fn attrs_to_map_impl(args: &[ColumnarValue]) -> Result<ColumnarValue> {
    if args.is_empty() {
        return Err(DataFusionError::Execution(
            "attrs_to_map requires 1 argument".to_string(),
        ));
    }

    let ColumnarValue::Array(attrs_array) = &args[0] else {
        return Err(DataFusionError::Execution(
            "attrs_to_map expects array input".to_string(),
        ));
    };

    // Downcast to ListArray (Attrs is a list of struct)
    let list_array = attrs_array
        .as_any()
        .downcast_ref::<ListArray>()
        .ok_or_else(|| DataFusionError::Execution("Expected ListArray for Attrs".to_string()))?;

    // Process each row (each list of attrs)
    // Pre-allocate capacity for result arrays
    let mut result_arrays = Vec::with_capacity(list_array.len());

    for row_idx in 0..list_array.len() {
        if list_array.is_null(row_idx) {
            // For null input, we'll create an empty map
            result_arrays.push(create_empty_map_array()?);
            continue;
        }

        let attrs_list = list_array.value(row_idx);

        // Get the struct array from the list
        let struct_array = attrs_list
            .as_any()
            .downcast_ref::<StructArray>()
            .ok_or_else(|| {
                DataFusionError::Execution("Expected StructArray in Attrs list".to_string())
            })?;

        // Extract the Key field
        let key_array = struct_array
            .column_by_name("Key")
            .ok_or_else(|| DataFusionError::Execution("Key field not found".to_string()))?
            .as_any()
            .downcast_ref::<StringArray>()
            .ok_or_else(|| DataFusionError::Execution("Key is not StringArray".to_string()))?;

        // Extract value fields
        let value_array = struct_array
            .column_by_name("Value")
            .ok_or_else(|| DataFusionError::Execution("Value field not found".to_string()))?
            .as_any()
            .downcast_ref::<ListArray>()
            .ok_or_else(|| DataFusionError::Execution("Value is not ListArray".to_string()))?;

        let value_int_array = struct_array
            .column_by_name("ValueInt")
            .ok_or_else(|| DataFusionError::Execution("ValueInt field not found".to_string()))?
            .as_any()
            .downcast_ref::<ListArray>()
            .ok_or_else(|| DataFusionError::Execution("ValueInt is not ListArray".to_string()))?;

        let value_double_array = struct_array
            .column_by_name("ValueDouble")
            .ok_or_else(|| DataFusionError::Execution("ValueDouble field not found".to_string()))?
            .as_any()
            .downcast_ref::<ListArray>()
            .ok_or_else(|| {
                DataFusionError::Execution("ValueDouble is not ListArray".to_string())
            })?;

        let value_bool_array = struct_array
            .column_by_name("ValueBool")
            .ok_or_else(|| DataFusionError::Execution("ValueBool field not found".to_string()))?
            .as_any()
            .downcast_ref::<ListArray>()
            .ok_or_else(|| DataFusionError::Execution("ValueBool is not ListArray".to_string()))?;

        // Build the result map using a HashMap to handle potential duplicate keys
        use std::collections::HashMap;
        // Pre-allocate capacity for better performance
        let mut map: HashMap<String, Vec<String>> = HashMap::with_capacity(key_array.len());

        // Iterate through each attribute
        // Cache length to avoid repeated method calls
        let key_len = key_array.len();
        for i in 0..key_len {
            if key_array.is_null(i) {
                continue;
            }

            let key = key_array.value(i).to_string();

            // Calculate total values to pre-allocate Vec
            let mut total_values = 0;
            if !value_array.is_null(i) {
                let value_list = value_array.value(i);
                total_values += value_list.len();
            }
            if !value_int_array.is_null(i) {
                let value_list = value_int_array.value(i);
                total_values += value_list.len();
            }
            if !value_double_array.is_null(i) {
                let value_list = value_double_array.value(i);
                total_values += value_list.len();
            }
            if !value_bool_array.is_null(i) {
                let value_list = value_bool_array.value(i);
                total_values += value_list.len();
            }

            let values = map.entry(key).or_insert_with(|| Vec::with_capacity(total_values));

            // Collect string values
            if !value_array.is_null(i) {
                let value_list = value_array.value(i);
                let string_values = value_list
                    .as_any()
                    .downcast_ref::<StringArray>()
                    .ok_or_else(|| {
                        DataFusionError::Execution("Value list items are not strings".to_string())
                    })?;

                let string_len = string_values.len();
                for j in 0..string_len {
                    if !string_values.is_null(j) {
                        values.push(string_values.value(j).to_string());
                    }
                }
            }

            // Collect int values (convert to string)
            if !value_int_array.is_null(i) {
                let value_list = value_int_array.value(i);
                let int_values = value_list
                    .as_any()
                    .downcast_ref::<Int64Array>()
                    .ok_or_else(|| {
                        DataFusionError::Execution("ValueInt list items are not Int64".to_string())
                    })?;

                let int_len = int_values.len();
                for j in 0..int_len {
                    if !int_values.is_null(j) {
                        values.push(int_values.value(j).to_string());
                    }
                }
            }

            // Collect double values (convert to string)
            if !value_double_array.is_null(i) {
                let value_list = value_double_array.value(i);
                let double_values = value_list
                    .as_any()
                    .downcast_ref::<Float64Array>()
                    .ok_or_else(|| {
                        DataFusionError::Execution(
                            "ValueDouble list items are not Float64".to_string(),
                        )
                    })?;

                let double_len = double_values.len();
                for j in 0..double_len {
                    if !double_values.is_null(j) {
                        values.push(double_values.value(j).to_string());
                    }
                }
            }

            // Collect bool values (convert to string)
            if !value_bool_array.is_null(i) {
                let value_list = value_bool_array.value(i);
                let bool_values = value_list
                    .as_any()
                    .downcast_ref::<BooleanArray>()
                    .ok_or_else(|| {
                        DataFusionError::Execution(
                            "ValueBool list items are not Boolean".to_string(),
                        )
                    })?;

                let bool_len = bool_values.len();
                for j in 0..bool_len {
                    if !bool_values.is_null(j) {
                        values.push(bool_values.value(j).to_string());
                    }
                }
            }
        }

        // Convert HashMap to Arrow MapArray
        result_arrays.push(create_map_from_hashmap(&map)?);
    }

    // Concatenate all map arrays into a single MapArray
    if result_arrays.is_empty() {
        return Ok(ColumnarValue::Array(Arc::new(create_empty_map_array()?)));
    }

    // Concatenate all MapArrays into one
    let concatenated = concatenate_map_arrays(&result_arrays)?;
    Ok(ColumnarValue::Array(concatenated))
}

/// Concatenate multiple MapArrays into a single MapArray
fn concatenate_map_arrays(arrays: &[MapArray]) -> Result<ArrayRef> {
    use datafusion::arrow::compute::concat;

    if arrays.is_empty() {
        return Ok(Arc::new(create_empty_map_array()?));
    }

    if arrays.len() == 1 {
        return Ok(Arc::new(arrays[0].clone()));
    }

    // Convert MapArrays to ArrayRefs for concatenation
    let array_refs: Vec<&dyn Array> = arrays.iter().map(|a| a as &dyn Array).collect();

    concat(&array_refs)
        .map_err(|e| DataFusionError::Execution(format!("Failed to concatenate MapArrays: {}", e)))
}

/// Create an empty map array
fn create_empty_map_array() -> Result<MapArray> {
    let keys_array = Arc::new(StringArray::from(Vec::<String>::new())) as ArrayRef;
    let mut values_builder = ListBuilder::new(StringBuilder::new());
    let values_array = Arc::new(values_builder.finish()) as ArrayRef;

    let struct_array = StructArray::from(vec![
        (
            Arc::new(Field::new("key", DataType::Utf8, false)),
            keys_array,
        ),
        (
            Arc::new(Field::new(
                "value",
                DataType::List(Arc::new(Field::new("item", DataType::Utf8, true))),
                true,
            )),
            values_array,
        ),
    ]);

    let offsets = vec![0, 0];
    let offset_buffer = OffsetBuffer::new(offsets.into());

    let field = Arc::new(Field::new(
        "entries",
        struct_array.data_type().clone(),
        false,
    ));

    MapArray::try_new(field, offset_buffer, struct_array, None, false)
        .map_err(|e| DataFusionError::Execution(format!("Failed to create empty MapArray: {}", e)))
}

/// Create a MapArray from a HashMap<String, Vec<String>>
fn create_map_from_hashmap(
    map: &std::collections::HashMap<String, Vec<String>>,
) -> Result<MapArray> {
    let mut keys_builder = StringBuilder::new();
    let mut values_builder = ListBuilder::new(StringBuilder::new());

    for (key, vals) in map.iter() {
        keys_builder.append_value(key);

        for val in vals {
            values_builder.values().append_value(val);
        }
        values_builder.append(true);
    }

    let keys_array = Arc::new(keys_builder.finish()) as ArrayRef;
    let values_array = Arc::new(values_builder.finish()) as ArrayRef;

    // Create struct array for map entries
    let struct_array = StructArray::from(vec![
        (
            Arc::new(Field::new("key", DataType::Utf8, false)),
            keys_array,
        ),
        (
            Arc::new(Field::new(
                "value",
                DataType::List(Arc::new(Field::new("item", DataType::Utf8, true))),
                true,
            )),
            values_array,
        ),
    ]);

    // Wrap in list array (MapArray is a specialized ListArray)
    let offsets = vec![0, struct_array.len() as i32];
    let offset_buffer = OffsetBuffer::new(offsets.into());

    let field = Arc::new(Field::new(
        "entries",
        struct_array.data_type().clone(),
        false,
    ));

    let map_array = MapArray::try_new(field, offset_buffer, struct_array, None, false)
        .map_err(|e| DataFusionError::Execution(format!("Failed to create MapArray: {}", e)))?;

    Ok(map_array)
}

/// Create and register the attrs_to_map UDF
pub fn create_attrs_to_map_udf() -> ScalarUDF {
    use datafusion::logical_expr::{ScalarFunctionImplementation, create_udf};

    let func: ScalarFunctionImplementation = Arc::new(attrs_to_map);

    create_udf(
        "attrs_to_map",
        vec![DataType::List(Arc::new(Field::new(
            "item",
            DataType::Struct(
                vec![
                    Arc::new(Field::new("Key", DataType::Utf8, true)),
                    Arc::new(Field::new("IsArray", DataType::Boolean, true)),
                    Arc::new(Field::new(
                        "Value",
                        DataType::List(Arc::new(Field::new("item", DataType::Utf8, true))),
                        true,
                    )),
                    Arc::new(Field::new(
                        "ValueInt",
                        DataType::List(Arc::new(Field::new("item", DataType::Int64, true))),
                        true,
                    )),
                    Arc::new(Field::new(
                        "ValueDouble",
                        DataType::List(Arc::new(Field::new("item", DataType::Float64, true))),
                        true,
                    )),
                    Arc::new(Field::new(
                        "ValueBool",
                        DataType::List(Arc::new(Field::new("item", DataType::Boolean, true))),
                        true,
                    )),
                    Arc::new(Field::new("ValueUnsupported", DataType::BinaryView, true)),
                ]
                .into(),
            ),
            true,
        )))],
        DataType::Map(
            Arc::new(Field::new(
                "entries",
                DataType::Struct(
                    vec![
                        Arc::new(Field::new("key", DataType::Utf8, false)),
                        Arc::new(Field::new(
                            "value",
                            DataType::List(Arc::new(Field::new("item", DataType::Utf8, true))),
                            true,
                        )),
                    ]
                    .into(),
                ),
                false,
            )),
            false,
        ),
        Volatility::Immutable,
        func,
    )
}

/// Register all UDFs with the DataFusion context
pub fn register_udfs(ctx: &SessionContext) {
    let attrs_to_map_udf = create_attrs_to_map_udf();
    ctx.register_udf(attrs_to_map_udf);
}
