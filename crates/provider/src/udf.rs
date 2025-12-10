use datafusion::arrow::array::{
    Array, ArrayRef, BooleanArray, Float64Array, Int64Array, ListArray, ListBuilder, MapArray,
    StringArray, StringBuilder, StringViewArray, StructArray,
};
use datafusion::arrow::buffer::OffsetBuffer;
use datafusion::arrow::datatypes::{DataType, Field};
use datafusion::common::ScalarValue;
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

            let values = map
                .entry(key)
                .or_insert_with(|| Vec::with_capacity(total_values));

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
    use datafusion::logical_expr::{create_udf, ScalarFunctionImplementation};

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

/// Check if an Attrs array contains a specific key-value pair
///
/// Takes three arguments:
/// - attrs: The Attrs list structure
/// - key: The key to search for
/// - value: The value to search for (will be converted to string for comparison)
///
/// Returns true if any attribute has the specified key and contains the specified value
/// in any of its value fields (Value, ValueInt, ValueDouble, ValueBool).
///
/// This is a wrapper function that logs errors before returning them to DataFusion.
pub fn attrs_contain_string(args: &[ColumnarValue]) -> Result<ColumnarValue> {
    attrs_contain_string_impl(args).map_err(|e| {
        error!("attrs_contain_string UDF error: {}", e);
        e
    })
}

/// Internal implementation of attrs_contain_string
fn attrs_contain_string_impl(args: &[ColumnarValue]) -> Result<ColumnarValue> {
    if args.len() != 3 {
        return Err(DataFusionError::Execution(
            "attrs_contain_string requires 3 arguments: (attrs, key, value)".to_string(),
        ));
    }

    let ColumnarValue::Array(attrs_array) = &args[0] else {
        return Err(DataFusionError::Execution(
            "attrs_contain_string expects array input for first argument".to_string(),
        ));
    };

    // Downcast to ListArray (Attrs is a list of struct)
    let list_array = attrs_array
        .as_any()
        .downcast_ref::<ListArray>()
        .ok_or_else(|| DataFusionError::Execution("Expected ListArray for Attrs".to_string()))?;

    // Extract key and value arguments - handle scalar vs array efficiently
    // For scalar values, we extract once and reuse; for arrays, we reference directly
    // Support both Utf8 and Utf8View for better performance
    let key_scalar = match &args[1] {
        ColumnarValue::Scalar(ScalarValue::Utf8(Some(s)))
        | ColumnarValue::Scalar(ScalarValue::Utf8View(Some(s)))
        | ColumnarValue::Scalar(ScalarValue::LargeUtf8(Some(s))) => Some(s.as_str()),
        ColumnarValue::Scalar(ScalarValue::Utf8(None))
        | ColumnarValue::Scalar(ScalarValue::Utf8View(None))
        | ColumnarValue::Scalar(ScalarValue::LargeUtf8(None)) => Some(""),
        ColumnarValue::Scalar(_) => {
            return Err(DataFusionError::Execution(
                "Second argument must be a string".to_string(),
            ))
        }
        ColumnarValue::Array(_) => None,
    };

    // Helper to get string value from array at index
    enum StringArrayType {
        Utf8(StringArray),
        Utf8View(StringViewArray),
    }

    let key_array = if key_scalar.is_none() {
        let arr = args[1].clone().into_array(list_array.len())?;

        if let Some(string_array) = arr.as_any().downcast_ref::<StringArray>() {
            Some(StringArrayType::Utf8(string_array.clone()))
        } else if let Some(string_view_array) = arr.as_any().downcast_ref::<StringViewArray>() {
            Some(StringArrayType::Utf8View(string_view_array.clone()))
        } else {
            return Err(DataFusionError::Execution(
                "Second argument must be a string".to_string(),
            ));
        }
    } else {
        None
    };

    let value_scalar = match &args[2] {
        ColumnarValue::Scalar(ScalarValue::Utf8(Some(s)))
        | ColumnarValue::Scalar(ScalarValue::Utf8View(Some(s)))
        | ColumnarValue::Scalar(ScalarValue::LargeUtf8(Some(s))) => Some(s.as_str()),
        ColumnarValue::Scalar(ScalarValue::Utf8(None))
        | ColumnarValue::Scalar(ScalarValue::Utf8View(None))
        | ColumnarValue::Scalar(ScalarValue::LargeUtf8(None)) => Some(""),
        ColumnarValue::Scalar(_) => {
            return Err(DataFusionError::Execution(
                "Third argument must be a string".to_string(),
            ))
        }
        ColumnarValue::Array(_) => None,
    };

    let value_array = if value_scalar.is_none() {
        let arr = args[2].clone().into_array(list_array.len())?;

        if let Some(string_array) = arr.as_any().downcast_ref::<StringArray>() {
            Some(StringArrayType::Utf8(string_array.clone()))
        } else if let Some(string_view_array) = arr.as_any().downcast_ref::<StringViewArray>() {
            Some(StringArrayType::Utf8View(string_view_array.clone()))
        } else {
            return Err(DataFusionError::Execution(
                "Third argument must be a string".to_string(),
            ));
        }
    } else {
        None
    };

    // Pre-parse numeric values if we're searching for them (avoid repeated parsing)
    let search_value_str = value_scalar.or_else(|| {
        value_array.as_ref().and_then(|arr| match arr {
            StringArrayType::Utf8(a) if !a.is_null(0) => Some(a.value(0)),
            StringArrayType::Utf8View(a) if !a.is_null(0) => Some(a.value(0)),
            _ => None,
        })
    });

    let value_as_int = search_value_str.and_then(|s| s.parse::<i64>().ok());
    let value_as_double = search_value_str.and_then(|s| s.parse::<f64>().ok());
    let value_as_bool = search_value_str.and_then(|s| match s {
        "true" => Some(true),
        "false" => Some(false),
        _ => None,
    });

    // Get field indices once to avoid repeated HashMap lookups
    // We need to peek at the first non-null row to get the schema
    let field_indices = if list_array.len() > 0 {
        let first_non_null_opt = (0..list_array.len()).find(|&i| !list_array.is_null(i));

        // If all rows are null, return all false
        let first_non_null = match first_non_null_opt {
            Some(idx) => idx,
            None => {
                let all_false = vec![false; list_array.len()];
                return Ok(ColumnarValue::Array(
                    Arc::new(BooleanArray::from(all_false)) as ArrayRef,
                ));
            }
        };

        let sample_array = list_array.value(first_non_null);
        let sample_struct = sample_array
            .as_any()
            .downcast_ref::<StructArray>()
            .ok_or_else(|| {
                DataFusionError::Execution("Expected StructArray in Attrs list".to_string())
            })?;

        // Get field indices by name once
        let fields = sample_struct.fields();
        let key_idx = fields
            .iter()
            .position(|f| f.name() == "Key")
            .ok_or_else(|| DataFusionError::Execution("Key field not found".to_string()))?;

        let value_idx = fields
            .iter()
            .position(|f| f.name() == "Value")
            .ok_or_else(|| DataFusionError::Execution("Value field not found".to_string()))?;

        let value_int_idx = fields
            .iter()
            .position(|f| f.name() == "ValueInt")
            .ok_or_else(|| DataFusionError::Execution("ValueInt field not found".to_string()))?;

        let value_double_idx = fields
            .iter()
            .position(|f| f.name() == "ValueDouble")
            .ok_or_else(|| DataFusionError::Execution("ValueDouble field not found".to_string()))?;

        let value_bool_idx = fields
            .iter()
            .position(|f| f.name() == "ValueBool")
            .ok_or_else(|| DataFusionError::Execution("ValueBool field not found".to_string()))?;

        (
            key_idx,
            value_idx,
            value_int_idx,
            value_double_idx,
            value_bool_idx,
        )
    } else {
        // Empty array, return early
        return Ok(ColumnarValue::Array(
            Arc::new(BooleanArray::from(Vec::<bool>::new())) as ArrayRef,
        ));
    };

    let (key_idx, value_idx, value_int_idx, value_double_idx, value_bool_idx) = field_indices;

    // Process each row
    let mut result = Vec::with_capacity(list_array.len());

    for row_idx in 0..list_array.len() {
        if list_array.is_null(row_idx) {
            result.push(false);
            continue;
        }

        // Get search key and value for this row
        let search_key = if let Some(scalar) = key_scalar {
            scalar
        } else if let Some(ref arr) = key_array {
            match arr {
                StringArrayType::Utf8(a) if !a.is_null(row_idx) => a.value(row_idx),
                StringArrayType::Utf8View(a) if !a.is_null(row_idx) => a.value(row_idx),
                _ => "",
            }
        } else {
            ""
        };

        let search_value = if let Some(scalar) = value_scalar {
            scalar
        } else if let Some(ref arr) = value_array {
            match arr {
                StringArrayType::Utf8(a) if !a.is_null(row_idx) => a.value(row_idx),
                StringArrayType::Utf8View(a) if !a.is_null(row_idx) => a.value(row_idx),
                _ => "",
            }
        } else {
            ""
        };

        let attrs_list = list_array.value(row_idx);

        // Get the struct array from the list
        let struct_array = attrs_list
            .as_any()
            .downcast_ref::<StructArray>()
            .ok_or_else(|| {
                DataFusionError::Execution("Expected StructArray in Attrs list".to_string())
            })?;

        // Use direct column access by index (no HashMap lookups!)
        let key_array_field = struct_array
            .column(key_idx)
            .as_any()
            .downcast_ref::<StringArray>()
            .ok_or_else(|| DataFusionError::Execution("Key is not StringArray".to_string()))?;

        let value_array_field = struct_array
            .column(value_idx)
            .as_any()
            .downcast_ref::<ListArray>()
            .ok_or_else(|| DataFusionError::Execution("Value is not ListArray".to_string()))?;

        let value_int_array_field = struct_array
            .column(value_int_idx)
            .as_any()
            .downcast_ref::<ListArray>()
            .ok_or_else(|| DataFusionError::Execution("ValueInt is not ListArray".to_string()))?;

        let value_double_array_field = struct_array
            .column(value_double_idx)
            .as_any()
            .downcast_ref::<ListArray>()
            .ok_or_else(|| {
                DataFusionError::Execution("ValueDouble is not ListArray".to_string())
            })?;

        let value_bool_array_field = struct_array
            .column(value_bool_idx)
            .as_any()
            .downcast_ref::<ListArray>()
            .ok_or_else(|| DataFusionError::Execution("ValueBool is not ListArray".to_string()))?;

        // Search through all attributes
        let mut found = false;
        for i in 0..key_array_field.len() {
            if key_array_field.is_null(i) {
                continue;
            }

            let key = key_array_field.value(i);
            if key != search_key {
                continue;
            }

            // Check string values
            if !value_array_field.is_null(i) {
                let value_list = value_array_field.value(i);
                let string_values = value_list
                    .as_any()
                    .downcast_ref::<StringArray>()
                    .ok_or_else(|| {
                        DataFusionError::Execution("Value list items are not strings".to_string())
                    })?;

                for j in 0..string_values.len() {
                    if !string_values.is_null(j) && string_values.value(j) == search_value {
                        found = true;
                        break;
                    }
                }
            }

            if found {
                break;
            }

            // Check int values - use pre-parsed value if available
            if !value_int_array_field.is_null(i) {
                if let Some(target_int) = value_as_int {
                    let value_list = value_int_array_field.value(i);
                    let int_values = value_list
                        .as_any()
                        .downcast_ref::<Int64Array>()
                        .ok_or_else(|| {
                            DataFusionError::Execution(
                                "ValueInt list items are not Int64".to_string(),
                            )
                        })?;

                    for j in 0..int_values.len() {
                        if !int_values.is_null(j) && int_values.value(j) == target_int {
                            found = true;
                            break;
                        }
                    }
                }
            }

            if found {
                break;
            }

            // Check double values - use pre-parsed value if available
            if !value_double_array_field.is_null(i) {
                if let Some(target_double) = value_as_double {
                    let value_list = value_double_array_field.value(i);
                    let double_values = value_list
                        .as_any()
                        .downcast_ref::<Float64Array>()
                        .ok_or_else(|| {
                            DataFusionError::Execution(
                                "ValueDouble list items are not Float64".to_string(),
                            )
                        })?;

                    for j in 0..double_values.len() {
                        if !double_values.is_null(j) && double_values.value(j) == target_double {
                            found = true;
                            break;
                        }
                    }
                }
            }

            if found {
                break;
            }

            // Check bool values - use pre-parsed value if available
            if !value_bool_array_field.is_null(i) {
                if let Some(target_bool) = value_as_bool {
                    let value_list = value_bool_array_field.value(i);
                    let bool_values = value_list
                        .as_any()
                        .downcast_ref::<BooleanArray>()
                        .ok_or_else(|| {
                            DataFusionError::Execution(
                                "ValueBool list items are not Boolean".to_string(),
                            )
                        })?;

                    for j in 0..bool_values.len() {
                        if !bool_values.is_null(j) && bool_values.value(j) == target_bool {
                            found = true;
                            break;
                        }
                    }
                }
            }

            if found {
                break;
            }
        }

        result.push(found);
    }

    let result_array = Arc::new(BooleanArray::from(result)) as ArrayRef;
    Ok(ColumnarValue::Array(result_array))
}

/// Create and register the attrs_contain_string UDF
pub fn create_attrs_contain_string_udf() -> ScalarUDF {
    use datafusion::logical_expr::{create_udf, ScalarFunctionImplementation};

    let func: ScalarFunctionImplementation = Arc::new(attrs_contain_string);

    create_udf(
        "attrs_contain_string",
        vec![
            DataType::List(Arc::new(Field::new(
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
            ))),
            DataType::Utf8,
            DataType::Utf8,
        ],
        DataType::Boolean,
        Volatility::Immutable,
        func,
    )
}

/// Register all UDFs with the DataFusion context
pub fn register_udfs(ctx: &SessionContext) {
    let attrs_to_map_udf = create_attrs_to_map_udf();
    ctx.register_udf(attrs_to_map_udf);

    let attrs_contain_string_udf = create_attrs_contain_string_udf();
    ctx.register_udf(attrs_contain_string_udf);
}

#[cfg(test)]
mod tests {
    use super::*;
    use datafusion::arrow::array::{
        BinaryViewBuilder, BooleanBuilder, Float64Builder, Int64Builder, ListBuilder, StructBuilder,
    };
    use datafusion::arrow::datatypes::Fields;
    use datafusion::scalar::ScalarValue;

    /// Helper function to create test Attrs array
    fn create_test_attrs(
        keys: Vec<&str>,
        string_values: Vec<Vec<&str>>,
        int_values: Vec<Vec<i64>>,
        double_values: Vec<Vec<f64>>,
        bool_values: Vec<Vec<bool>>,
    ) -> ArrayRef {
        // Build individual arrays for each field
        let mut key_builder = StringBuilder::new();
        let mut is_array_builder = BooleanBuilder::new();
        let mut value_builder = ListBuilder::new(StringBuilder::new());
        let mut value_int_builder = ListBuilder::new(Int64Builder::new());
        let mut value_double_builder = ListBuilder::new(Float64Builder::new());
        let mut value_bool_builder = ListBuilder::new(BooleanBuilder::new());
        let mut value_unsupported_builder = BinaryViewBuilder::new();

        for i in 0..keys.len() {
            // Key
            key_builder.append_value(keys[i]);

            // IsArray
            is_array_builder.append_value(false);

            // Value (strings)
            for val in &string_values[i] {
                value_builder.values().append_value(*val);
            }
            value_builder.append(true);

            // ValueInt
            for val in &int_values[i] {
                value_int_builder.values().append_value(*val);
            }
            value_int_builder.append(true);

            // ValueDouble
            for val in &double_values[i] {
                value_double_builder.values().append_value(*val);
            }
            value_double_builder.append(true);

            // ValueBool
            for val in &bool_values[i] {
                value_bool_builder.values().append_value(*val);
            }
            value_bool_builder.append(true);

            // ValueUnsupported
            value_unsupported_builder.append_null();
        }

        // Build the struct array
        let struct_array = StructArray::from(vec![
            (
                Arc::new(Field::new("Key", DataType::Utf8, true)),
                Arc::new(key_builder.finish()) as ArrayRef,
            ),
            (
                Arc::new(Field::new("IsArray", DataType::Boolean, true)),
                Arc::new(is_array_builder.finish()) as ArrayRef,
            ),
            (
                Arc::new(Field::new(
                    "Value",
                    DataType::List(Arc::new(Field::new("item", DataType::Utf8, true))),
                    true,
                )),
                Arc::new(value_builder.finish()) as ArrayRef,
            ),
            (
                Arc::new(Field::new(
                    "ValueInt",
                    DataType::List(Arc::new(Field::new("item", DataType::Int64, true))),
                    true,
                )),
                Arc::new(value_int_builder.finish()) as ArrayRef,
            ),
            (
                Arc::new(Field::new(
                    "ValueDouble",
                    DataType::List(Arc::new(Field::new("item", DataType::Float64, true))),
                    true,
                )),
                Arc::new(value_double_builder.finish()) as ArrayRef,
            ),
            (
                Arc::new(Field::new(
                    "ValueBool",
                    DataType::List(Arc::new(Field::new("item", DataType::Boolean, true))),
                    true,
                )),
                Arc::new(value_bool_builder.finish()) as ArrayRef,
            ),
            (
                Arc::new(Field::new("ValueUnsupported", DataType::BinaryView, true)),
                Arc::new(value_unsupported_builder.finish()) as ArrayRef,
            ),
        ]);

        // Wrap in a list array
        let offsets = vec![0, struct_array.len() as i32];
        let offset_buffer = OffsetBuffer::new(offsets.into());

        let field = Arc::new(Field::new("item", struct_array.data_type().clone(), true));

        Arc::new(ListArray::try_new(field, offset_buffer, Arc::new(struct_array), None).unwrap())
    }

    #[test]
    fn test_attrs_contain_string_with_string_value() {
        let attrs = create_test_attrs(
            vec!["service.name", "http.method"],
            vec![vec!["my-service"], vec!["GET"]],
            vec![vec![], vec![]],
            vec![vec![], vec![]],
            vec![vec![], vec![]],
        );

        let key = ColumnarValue::Scalar(ScalarValue::Utf8(Some("service.name".to_string())));
        let value = ColumnarValue::Scalar(ScalarValue::Utf8(Some("my-service".to_string())));

        let result = attrs_contain_string(&[ColumnarValue::Array(attrs), key, value]).unwrap();

        if let ColumnarValue::Array(result_array) = result {
            let bool_array = result_array
                .as_any()
                .downcast_ref::<BooleanArray>()
                .unwrap();
            assert_eq!(bool_array.len(), 1);
            assert_eq!(bool_array.value(0), true);
        } else {
            panic!("Expected Array result");
        }
    }

    #[test]
    fn test_attrs_contain_string_with_int_value() {
        let attrs = create_test_attrs(
            vec!["http.status_code", "http.method"],
            vec![vec![], vec!["GET"]],
            vec![vec![200, 404], vec![]],
            vec![vec![], vec![]],
            vec![vec![], vec![]],
        );

        let key = ColumnarValue::Scalar(ScalarValue::Utf8(Some("http.status_code".to_string())));
        let value = ColumnarValue::Scalar(ScalarValue::Utf8(Some("200".to_string())));

        let result = attrs_contain_string(&[ColumnarValue::Array(attrs), key, value]).unwrap();

        if let ColumnarValue::Array(result_array) = result {
            let bool_array = result_array
                .as_any()
                .downcast_ref::<BooleanArray>()
                .unwrap();
            assert_eq!(bool_array.len(), 1);
            assert_eq!(bool_array.value(0), true);
        } else {
            panic!("Expected Array result");
        }
    }

    #[test]
    fn test_attrs_contain_string_with_double_value() {
        let attrs = create_test_attrs(
            vec!["temperature", "humidity"],
            vec![vec![], vec![]],
            vec![vec![], vec![]],
            vec![vec![98.6, 100.4], vec![45.5]],
            vec![vec![], vec![]],
        );

        let key = ColumnarValue::Scalar(ScalarValue::Utf8(Some("temperature".to_string())));
        let value = ColumnarValue::Scalar(ScalarValue::Utf8(Some("98.6".to_string())));

        let result = attrs_contain_string(&[ColumnarValue::Array(attrs), key, value]).unwrap();

        if let ColumnarValue::Array(result_array) = result {
            let bool_array = result_array
                .as_any()
                .downcast_ref::<BooleanArray>()
                .unwrap();
            assert_eq!(bool_array.len(), 1);
            assert_eq!(bool_array.value(0), true);
        } else {
            panic!("Expected Array result");
        }
    }

    #[test]
    fn test_attrs_contain_string_with_bool_value() {
        let attrs = create_test_attrs(
            vec!["is_enabled", "is_admin"],
            vec![vec![], vec![]],
            vec![vec![], vec![]],
            vec![vec![], vec![]],
            vec![vec![true, false], vec![false]],
        );

        let key = ColumnarValue::Scalar(ScalarValue::Utf8(Some("is_enabled".to_string())));
        let value = ColumnarValue::Scalar(ScalarValue::Utf8(Some("true".to_string())));

        let result = attrs_contain_string(&[ColumnarValue::Array(attrs), key, value]).unwrap();

        if let ColumnarValue::Array(result_array) = result {
            let bool_array = result_array
                .as_any()
                .downcast_ref::<BooleanArray>()
                .unwrap();
            assert_eq!(bool_array.len(), 1);
            assert_eq!(bool_array.value(0), true);
        } else {
            panic!("Expected Array result");
        }
    }

    #[test]
    fn test_attrs_contain_string_not_found() {
        let attrs = create_test_attrs(
            vec!["service.name", "http.method"],
            vec![vec!["my-service"], vec!["GET"]],
            vec![vec![], vec![]],
            vec![vec![], vec![]],
            vec![vec![], vec![]],
        );

        let key = ColumnarValue::Scalar(ScalarValue::Utf8(Some("service.name".to_string())));
        let value = ColumnarValue::Scalar(ScalarValue::Utf8(Some("other-service".to_string())));

        let result = attrs_contain_string(&[ColumnarValue::Array(attrs), key, value]).unwrap();

        if let ColumnarValue::Array(result_array) = result {
            let bool_array = result_array
                .as_any()
                .downcast_ref::<BooleanArray>()
                .unwrap();
            assert_eq!(bool_array.len(), 1);
            assert_eq!(bool_array.value(0), false);
        } else {
            panic!("Expected Array result");
        }
    }

    #[test]
    fn test_attrs_contain_string_key_not_found() {
        let attrs = create_test_attrs(
            vec!["service.name", "http.method"],
            vec![vec!["my-service"], vec!["GET"]],
            vec![vec![], vec![]],
            vec![vec![], vec![]],
            vec![vec![], vec![]],
        );

        let key = ColumnarValue::Scalar(ScalarValue::Utf8(Some("non.existent.key".to_string())));
        let value = ColumnarValue::Scalar(ScalarValue::Utf8(Some("my-service".to_string())));

        let result = attrs_contain_string(&[ColumnarValue::Array(attrs), key, value]).unwrap();

        if let ColumnarValue::Array(result_array) = result {
            let bool_array = result_array
                .as_any()
                .downcast_ref::<BooleanArray>()
                .unwrap();
            assert_eq!(bool_array.len(), 1);
            assert_eq!(bool_array.value(0), false);
        } else {
            panic!("Expected Array result");
        }
    }

    #[test]
    fn test_attrs_contain_string_with_null_attrs() {
        let attrs_fields = Fields::from(vec![
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
        ]);

        let mut list_builder = ListBuilder::new(StructBuilder::from_fields(attrs_fields, 0));
        list_builder.append_null();
        let null_attrs = Arc::new(list_builder.finish()) as ArrayRef;

        let key = ColumnarValue::Scalar(ScalarValue::Utf8(Some("service.name".to_string())));
        let value = ColumnarValue::Scalar(ScalarValue::Utf8(Some("my-service".to_string())));

        let result = attrs_contain_string(&[ColumnarValue::Array(null_attrs), key, value]).unwrap();

        if let ColumnarValue::Array(result_array) = result {
            let bool_array = result_array
                .as_any()
                .downcast_ref::<BooleanArray>()
                .unwrap();
            assert_eq!(bool_array.len(), 1);
            assert_eq!(bool_array.value(0), false);
        } else {
            panic!("Expected Array result");
        }
    }

    #[test]
    fn test_attrs_contain_string_multiple_values_in_list() {
        let attrs = create_test_attrs(
            vec!["tags"],
            vec![vec!["prod", "us-west", "critical"]],
            vec![vec![]],
            vec![vec![]],
            vec![vec![]],
        );

        let key = ColumnarValue::Scalar(ScalarValue::Utf8(Some("tags".to_string())));
        let value = ColumnarValue::Scalar(ScalarValue::Utf8(Some("us-west".to_string())));

        let result = attrs_contain_string(&[ColumnarValue::Array(attrs), key, value]).unwrap();

        if let ColumnarValue::Array(result_array) = result {
            let bool_array = result_array
                .as_any()
                .downcast_ref::<BooleanArray>()
                .unwrap();
            assert_eq!(bool_array.len(), 1);
            assert_eq!(bool_array.value(0), true);
        } else {
            panic!("Expected Array result");
        }
    }

    #[test]
    fn test_attrs_contain_string_wrong_arguments_count() {
        let attrs = create_test_attrs(
            vec!["service.name"],
            vec![vec!["my-service"]],
            vec![vec![]],
            vec![vec![]],
            vec![vec![]],
        );

        let key = ColumnarValue::Scalar(ScalarValue::Utf8(Some("service.name".to_string())));

        // Only 2 arguments instead of 3
        let result = attrs_contain_string(&[ColumnarValue::Array(attrs), key]);
        assert!(result.is_err());
    }
}
