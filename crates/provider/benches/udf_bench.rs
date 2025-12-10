use criterion::{black_box, criterion_group, criterion_main, BenchmarkId, Criterion, Throughput};
use datafusion::arrow::array::{
    Array, ArrayRef, BinaryViewArray, BooleanArray, Float64Array, Int64Array, ListBuilder,
    StringBuilder, StructArray,
};
use datafusion::arrow::buffer::OffsetBuffer;
use datafusion::arrow::datatypes::{DataType, Field, Fields};
use datafusion::logical_expr::ColumnarValue;
use datafusion::scalar::ScalarValue;
use provider::udf::{attrs_contain_string, attrs_to_map};
use std::sync::Arc;

/// Generate synthetic Attrs array for benchmarking
///
/// Parameters:
/// - num_rows: Number of rows (traces) in the batch
/// - attrs_per_row: Number of attributes per row
/// - values_per_attr: Number of values per attribute (for multi-valued attrs)
/// - use_int: Include integer values
/// - use_double: Include double values
/// - use_bool: Include boolean values
fn generate_synthetic_attrs(
    num_rows: usize,
    attrs_per_row: usize,
    values_per_attr: usize,
    use_int: bool,
    use_double: bool,
    use_bool: bool,
) -> ArrayRef {
    // Build all the data arrays first
    let total_attrs = num_rows * attrs_per_row;

    let mut all_keys_builder = StringBuilder::new();
    let mut all_is_array_builder = BooleanArray::builder(total_attrs);
    let mut all_value_list_builder = ListBuilder::new(StringBuilder::new());
    let mut all_value_int_list_builder = ListBuilder::new(Int64Array::builder(0));
    let mut all_value_double_list_builder = ListBuilder::new(Float64Array::builder(0));
    let mut all_value_bool_list_builder = ListBuilder::new(BooleanArray::builder(0));

    for row_idx in 0..num_rows {
        for attr_idx in 0..attrs_per_row {
            // Generate key
            all_keys_builder.append_value(format!("attr_{}_row_{}", attr_idx, row_idx));
            all_is_array_builder.append_value(values_per_attr > 1);

            // Distribute attributes across different value types
            let attr_type = attr_idx % 4;

            match attr_type {
                0 => {
                    // String values
                    for val_idx in 0..values_per_attr {
                        all_value_list_builder
                            .values()
                            .append_value(format!("value_{}", val_idx));
                    }
                    all_value_list_builder.append(true);
                    all_value_int_list_builder.append(false);
                    all_value_double_list_builder.append(false);
                    all_value_bool_list_builder.append(false);
                }
                1 if use_int => {
                    // Int values
                    for val_idx in 0..values_per_attr {
                        all_value_int_list_builder
                            .values()
                            .append_value(42 + val_idx as i64);
                    }
                    all_value_list_builder.append(false);
                    all_value_int_list_builder.append(true);
                    all_value_double_list_builder.append(false);
                    all_value_bool_list_builder.append(false);
                }
                2 if use_double => {
                    // Double values
                    for val_idx in 0..values_per_attr {
                        all_value_double_list_builder
                            .values()
                            .append_value(3.14 + val_idx as f64);
                    }
                    all_value_list_builder.append(false);
                    all_value_int_list_builder.append(false);
                    all_value_double_list_builder.append(true);
                    all_value_bool_list_builder.append(false);
                }
                3 if use_bool => {
                    // Bool values
                    for val_idx in 0..values_per_attr {
                        all_value_bool_list_builder
                            .values()
                            .append_value(val_idx % 2 == 0);
                    }
                    all_value_list_builder.append(false);
                    all_value_int_list_builder.append(false);
                    all_value_double_list_builder.append(false);
                    all_value_bool_list_builder.append(true);
                }
                _ => {
                    // Default to string values
                    for val_idx in 0..values_per_attr {
                        all_value_list_builder
                            .values()
                            .append_value(format!("default_{}", val_idx));
                    }
                    all_value_list_builder.append(true);
                    all_value_int_list_builder.append(false);
                    all_value_double_list_builder.append(false);
                    all_value_bool_list_builder.append(false);
                }
            }
        }
    }

    // Build all arrays
    let keys_array = Arc::new(all_keys_builder.finish()) as ArrayRef;
    let is_array_array = Arc::new(all_is_array_builder.finish()) as ArrayRef;
    let value_array = Arc::new(all_value_list_builder.finish()) as ArrayRef;
    let value_int_array = Arc::new(all_value_int_list_builder.finish()) as ArrayRef;
    let value_double_array = Arc::new(all_value_double_list_builder.finish()) as ArrayRef;
    let value_bool_array = Arc::new(all_value_bool_list_builder.finish()) as ArrayRef;
    let value_unsupported_array = Arc::new(BinaryViewArray::new_null(total_attrs)) as ArrayRef;

    // Create the struct array containing all attributes
    let struct_fields: Fields = vec![
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
    .into();

    let struct_array = StructArray::new(
        struct_fields,
        vec![
            keys_array,
            is_array_array,
            value_array,
            value_int_array,
            value_double_array,
            value_bool_array,
            value_unsupported_array,
        ],
        None,
    );

    // Create offsets for the list array (each row has attrs_per_row attributes)
    let mut offsets = Vec::with_capacity(num_rows + 1);
    for i in 0..=num_rows {
        offsets.push((i * attrs_per_row) as i32);
    }
    let offset_buffer = OffsetBuffer::new(offsets.into());

    // Create the list array wrapping the struct array
    let list_field = Arc::new(Field::new("item", struct_array.data_type().clone(), true));
    let list_array = datafusion::arrow::array::GenericListArray::new(
        list_field,
        offset_buffer,
        Arc::new(struct_array),
        None,
    );

    Arc::new(list_array)
}

/// Benchmark scenarios
struct BenchScenario {
    name: &'static str,
    num_rows: usize,
    attrs_per_row: usize,
    values_per_attr: usize,
    use_int: bool,
    use_double: bool,
    use_bool: bool,
}

impl BenchScenario {
    fn id(&self) -> String {
        format!(
            "{}_{}rows_{}attrs_{}vals",
            self.name, self.num_rows, self.attrs_per_row, self.values_per_attr
        )
    }
}

fn get_scenarios() -> Vec<BenchScenario> {
    vec![
        // Small batches - typical span attributes
        BenchScenario {
            name: "small_single_values",
            num_rows: 100,
            attrs_per_row: 10,
            values_per_attr: 1,
            use_int: true,
            use_double: true,
            use_bool: true,
        },
        BenchScenario {
            name: "small_multi_values",
            num_rows: 100,
            attrs_per_row: 10,
            values_per_attr: 5,
            use_int: true,
            use_double: true,
            use_bool: true,
        },
        // Medium batches - realistic trace data
        BenchScenario {
            name: "medium_single_values",
            num_rows: 1000,
            attrs_per_row: 20,
            values_per_attr: 1,
            use_int: true,
            use_double: true,
            use_bool: true,
        },
        BenchScenario {
            name: "medium_multi_values",
            num_rows: 1000,
            attrs_per_row: 20,
            values_per_attr: 3,
            use_int: true,
            use_double: true,
            use_bool: true,
        },
        // Large batches - stress test
        BenchScenario {
            name: "large_single_values",
            num_rows: 10000,
            attrs_per_row: 15,
            values_per_attr: 1,
            use_int: true,
            use_double: true,
            use_bool: true,
        },
        BenchScenario {
            name: "large_many_attrs",
            num_rows: 1000,
            attrs_per_row: 100,
            values_per_attr: 1,
            use_int: true,
            use_double: true,
            use_bool: true,
        },
        // String-only (common case)
        BenchScenario {
            name: "string_only",
            num_rows: 1000,
            attrs_per_row: 20,
            values_per_attr: 1,
            use_int: false,
            use_double: false,
            use_bool: false,
        },
        // Mixed types with complex values
        BenchScenario {
            name: "mixed_complex",
            num_rows: 1000,
            attrs_per_row: 30,
            values_per_attr: 10,
            use_int: true,
            use_double: true,
            use_bool: true,
        },
    ]
}

fn bench_attrs_to_map(c: &mut Criterion) {
    let mut group = c.benchmark_group("attrs_to_map");

    for scenario in get_scenarios() {
        let attrs_array = generate_synthetic_attrs(
            scenario.num_rows,
            scenario.attrs_per_row,
            scenario.values_per_attr,
            scenario.use_int,
            scenario.use_double,
            scenario.use_bool,
        );

        // Calculate total attributes processed
        let total_attrs = scenario.num_rows * scenario.attrs_per_row;
        group.throughput(Throughput::Elements(total_attrs as u64));

        group.bench_with_input(
            BenchmarkId::new("scenario", scenario.id()),
            &attrs_array,
            |b, input| {
                let args = vec![ColumnarValue::Array(input.clone())];
                b.iter(|| {
                    let result = attrs_to_map(black_box(&args));
                    black_box(result)
                });
            },
        );
    }

    group.finish();
}

/// Benchmark with different row sizes to test scalability
fn bench_attrs_to_map_scalability(c: &mut Criterion) {
    let mut group = c.benchmark_group("attrs_to_map_scalability");

    let row_counts = vec![10, 100, 1000, 5000, 10000];
    let attrs_per_row = 15;
    let values_per_attr = 1;

    for num_rows in row_counts {
        let attrs_array =
            generate_synthetic_attrs(num_rows, attrs_per_row, values_per_attr, true, true, true);

        group.throughput(Throughput::Elements((num_rows * attrs_per_row) as u64));

        group.bench_with_input(
            BenchmarkId::from_parameter(num_rows),
            &attrs_array,
            |b, input| {
                let args = vec![ColumnarValue::Array(input.clone())];
                b.iter(|| {
                    let result = attrs_to_map(black_box(&args));
                    black_box(result)
                });
            },
        );
    }

    group.finish();
}

/// Benchmark with different attribute counts per row
fn bench_attrs_to_map_attr_count(c: &mut Criterion) {
    let mut group = c.benchmark_group("attrs_to_map_attr_count");

    let num_rows = 1000;
    let attr_counts = vec![5, 10, 20, 50, 100];
    let values_per_attr = 1;

    for attrs_per_row in attr_counts {
        let attrs_array =
            generate_synthetic_attrs(num_rows, attrs_per_row, values_per_attr, true, true, true);

        group.throughput(Throughput::Elements((num_rows * attrs_per_row) as u64));

        group.bench_with_input(
            BenchmarkId::from_parameter(attrs_per_row),
            &attrs_array,
            |b, input| {
                let args = vec![ColumnarValue::Array(input.clone())];
                b.iter(|| {
                    let result = attrs_to_map(black_box(&args));
                    black_box(result)
                });
            },
        );
    }

    group.finish();
}

/// Benchmark attrs_contain_string with different scenarios
fn bench_attrs_contain_string(c: &mut Criterion) {
    let mut group = c.benchmark_group("attrs_contain_string");

    // Different scenarios for searching
    let scenarios = vec![
        ("hit_first", 0, true),   // Key/value at first position
        ("hit_middle", 10, true), // Key/value in middle
        ("hit_last", 19, true),   // Key/value at last position
        ("miss", 0, false),       // Key exists but value doesn't match
    ];

    for (scenario_name, target_attr_idx, should_match) in scenarios {
        let num_rows = 1000;
        let attrs_per_row = 20;

        let attrs_array = generate_synthetic_attrs(num_rows, attrs_per_row, 1, true, true, true);

        let key = ColumnarValue::Scalar(ScalarValue::Utf8(Some(format!(
            "attr_{}_row_0",
            target_attr_idx
        ))));
        let value = if should_match {
            ColumnarValue::Scalar(ScalarValue::Utf8(Some("value_0".to_string())))
        } else {
            ColumnarValue::Scalar(ScalarValue::Utf8(Some("nonexistent".to_string())))
        };

        group.throughput(Throughput::Elements(num_rows as u64));

        group.bench_with_input(
            BenchmarkId::new("scenario", scenario_name),
            &(attrs_array, key, value),
            |b, (attrs, k, v)| {
                let args = vec![ColumnarValue::Array(attrs.clone()), k.clone(), v.clone()];
                b.iter(|| {
                    let result = attrs_contain_string(black_box(&args));
                    black_box(result)
                });
            },
        );
    }

    group.finish();
}

/// Benchmark attrs_contain_string with different data types
fn bench_attrs_contain_string_types(c: &mut Criterion) {
    let mut group = c.benchmark_group("attrs_contain_string_types");

    let num_rows = 1000;
    let attrs_per_row = 20;

    // Test different value types
    let type_scenarios = vec![
        ("string", 0, "value_0"),
        ("int", 1, "42"),
        ("double", 2, "3.14"),
        ("bool", 3, "true"),
    ];

    for (type_name, attr_idx, expected_value) in type_scenarios {
        let attrs_array = generate_synthetic_attrs(num_rows, attrs_per_row, 1, true, true, true);

        let key =
            ColumnarValue::Scalar(ScalarValue::Utf8(Some(format!("attr_{}_row_0", attr_idx))));
        let value = ColumnarValue::Scalar(ScalarValue::Utf8(Some(expected_value.to_string())));

        group.throughput(Throughput::Elements(num_rows as u64));

        group.bench_with_input(
            BenchmarkId::new("type", type_name),
            &(attrs_array, key, value),
            |b, (attrs, k, v)| {
                let args = vec![ColumnarValue::Array(attrs.clone()), k.clone(), v.clone()];
                b.iter(|| {
                    let result = attrs_contain_string(black_box(&args));
                    black_box(result)
                });
            },
        );
    }

    group.finish();
}

/// Benchmark attrs_contain_string scalability with different row counts
fn bench_attrs_contain_string_scalability(c: &mut Criterion) {
    let mut group = c.benchmark_group("attrs_contain_string_scalability");

    let row_counts = vec![10, 100, 1000, 5000, 10000];
    let attrs_per_row = 15;

    for num_rows in row_counts {
        let attrs_array = generate_synthetic_attrs(num_rows, attrs_per_row, 1, true, true, true);

        let key = ColumnarValue::Scalar(ScalarValue::Utf8(Some("attr_5_row_0".to_string())));
        let value = ColumnarValue::Scalar(ScalarValue::Utf8(Some("value_0".to_string())));

        group.throughput(Throughput::Elements(num_rows as u64));

        group.bench_with_input(
            BenchmarkId::from_parameter(num_rows),
            &(attrs_array, key, value),
            |b, (attrs, k, v)| {
                let args = vec![ColumnarValue::Array(attrs.clone()), k.clone(), v.clone()];
                b.iter(|| {
                    let result = attrs_contain_string(black_box(&args));
                    black_box(result)
                });
            },
        );
    }

    group.finish();
}

/// Benchmark attrs_contain_string with different attribute counts per row
fn bench_attrs_contain_string_attr_count(c: &mut Criterion) {
    let mut group = c.benchmark_group("attrs_contain_string_attr_count");

    let num_rows = 1000;
    let attr_counts = vec![5, 10, 20, 50, 100];

    for attrs_per_row in attr_counts {
        let attrs_array = generate_synthetic_attrs(num_rows, attrs_per_row, 1, true, true, true);

        // Search for an attribute in the middle
        let target_attr = attrs_per_row / 2;
        let key = ColumnarValue::Scalar(ScalarValue::Utf8(Some(format!(
            "attr_{}_row_0",
            target_attr
        ))));
        let value = ColumnarValue::Scalar(ScalarValue::Utf8(Some("value_0".to_string())));

        group.throughput(Throughput::Elements(num_rows as u64));

        group.bench_with_input(
            BenchmarkId::from_parameter(attrs_per_row),
            &(attrs_array, key, value),
            |b, (attrs, k, v)| {
                let args = vec![ColumnarValue::Array(attrs.clone()), k.clone(), v.clone()];
                b.iter(|| {
                    let result = attrs_contain_string(black_box(&args));
                    black_box(result)
                });
            },
        );
    }

    group.finish();
}

/// Benchmark attrs_contain_string with multi-valued attributes
fn bench_attrs_contain_string_multi_values(c: &mut Criterion) {
    let mut group = c.benchmark_group("attrs_contain_string_multi_values");

    let num_rows = 1000;
    let attrs_per_row = 20;
    let value_counts = vec![1, 3, 5, 10, 20];

    for values_per_attr in value_counts {
        let attrs_array =
            generate_synthetic_attrs(num_rows, attrs_per_row, values_per_attr, true, true, true);

        let key = ColumnarValue::Scalar(ScalarValue::Utf8(Some("attr_5_row_0".to_string())));
        // Search for a value in the middle of the value list
        let target_value = values_per_attr / 2;
        let value =
            ColumnarValue::Scalar(ScalarValue::Utf8(Some(format!("value_{}", target_value))));

        group.throughput(Throughput::Elements(num_rows as u64));

        group.bench_with_input(
            BenchmarkId::from_parameter(values_per_attr),
            &(attrs_array, key, value),
            |b, (attrs, k, v)| {
                let args = vec![ColumnarValue::Array(attrs.clone()), k.clone(), v.clone()];
                b.iter(|| {
                    let result = attrs_contain_string(black_box(&args));
                    black_box(result)
                });
            },
        );
    }

    group.finish();
}

criterion_group! {
    name = benches;
    config = Criterion::default()
        .measurement_time(std::time::Duration::from_secs(10))
        .sample_size(100);
    targets = bench_attrs_to_map,
              bench_attrs_to_map_scalability,
              bench_attrs_to_map_attr_count,
              bench_attrs_contain_string,
              bench_attrs_contain_string_types,
              bench_attrs_contain_string_scalability,
              bench_attrs_contain_string_attr_count,
              bench_attrs_contain_string_multi_values
}
criterion_main!(benches);
