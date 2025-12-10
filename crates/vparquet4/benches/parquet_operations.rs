use std::{env, fs::File, path::PathBuf, sync::Arc};

use arrow::array::{BooleanArray, ListArray, StringArray, StructArray};
use arrow::compute::kernels::cmp;
use criterion::{black_box, criterion_group, criterion_main, Criterion};
use parquet::arrow::arrow_reader::{
    ArrowPredicateFn, ArrowReaderMetadata, ParquetRecordBatchReaderBuilder, RowFilter,
};
use parquet::arrow::ProjectionMask;
use parquet::basic::Encoding;
use parquet::column::page::PageReader;
use parquet::data_type::ByteArray;
use parquet::file::reader::{FileReader, SerializedFileReader};
use parquet::schema::types::ColumnDescriptor;

fn get_benchmark_file_path() -> PathBuf {
    let block_id = env::var("BENCH_BLOCKID").expect(
        "BENCH_BLOCKID is not set. Set BENCH_BLOCKID to the guid of the block to benchmark. \
         e.g. `export BENCH_BLOCKID=030c8c4f-9d47-4916-aadc-26b90b1d2bc4`",
    );

    let tenant_id = env::var("BENCH_TENANTID").unwrap_or_else(|_| "1".to_string());

    let bench_path = env::var("BENCH_PATH").expect(
        "BENCH_PATH is not set. Set BENCH_PATH to the root of the backend such that the block \
         to benchmark is at <BENCH_PATH>/<BENCH_TENANTID>/<BENCH_BLOCKID>.",
    );

    let mut path = PathBuf::from(bench_path);
    path.push(tenant_id);
    path.push(block_id);
    path.push("data.parquet");

    path
}

/// Check if a target value exists in the dictionary using low-level parquet decoding
/// This is the KEY optimization that Go uses - checking the dictionary directly
fn check_dictionary_contains_value(
    page_reader: &mut Box<dyn PageReader>,
    column_desc: Arc<ColumnDescriptor>,
    target: &[u8],
) -> Result<Option<bool>, Box<dyn std::error::Error>> {
    use parquet::column::page::Page;
    use parquet::decoding::get_decoder;

    // Read through pages looking for dictionary page
    while let Some(page) = page_reader.get_next_page()? {
        match page {
            Page::DictionaryPage {
                buf,
                num_values,
                encoding,
                ..
            } => {
                // Found dictionary page! Decode it to check if target exists
                let mut decoder = get_decoder::<parquet::data_type::ByteArrayType>(
                    column_desc.clone(),
                    encoding,
                )?;

                // Create buffer to hold dictionary values
                let mut dict_buffer = vec![ByteArray::default(); num_values as usize];

                // Decode the dictionary
                decoder.set_data(buf, num_values as usize)?;
                let decoded_count = decoder.get(&mut dict_buffer)?;

                // Search for target in dictionary
                for i in 0..decoded_count {
                    if dict_buffer[i].data() == target {
                        // Target FOUND in dictionary - this row group might contain it
                        return Ok(Some(true));
                    }
                }

                // Target NOT found in dictionary - definitely not in this row group!
                return Ok(Some(false));
            }
            Page::DataPage { .. } | Page::DataPageV2 { .. } => {
                // Hit data pages without seeing dictionary - not dictionary encoded
                return Ok(None);
            }
        }
    }

    // No pages found
    Ok(None)
}

/// Helper to get dictionary info without decoding (for diagnostics)
fn get_dictionary_info(
    page_reader: &mut Box<dyn PageReader>,
) -> Result<Option<(Encoding, usize)>, Box<dyn std::error::Error>> {
    use parquet::column::page::Page;

    while let Some(page) = page_reader.get_next_page()? {
        match page {
            Page::DictionaryPage {
                num_values,
                encoding,
                ..
            } => {
                return Ok(Some((encoding, num_values as usize)));
            }
            Page::DataPage { .. } | Page::DataPageV2 { .. } => {
                return Ok(None);
            }
        }
    }

    Ok(None)
}

fn parquet_operations(c: &mut Criterion) {
    let file_path = get_benchmark_file_path();

    // Load ArrowReaderMetadata once for reuse
    let file = File::open(&file_path).unwrap();
    let metadata = ArrowReaderMetadata::load(&file, Default::default()).unwrap();

    c.bench_function("metadata", |b| {
        b.iter(|| {
            let file = File::open(&file_path).unwrap();
            let builder =
                ParquetRecordBatchReaderBuilder::new_with_metadata(file, metadata.clone());
            let metadata = builder.metadata().clone();
            black_box(metadata);
        });
    });

    c.bench_function("schema", |b| {
        b.iter(|| {
            let file = File::open(&file_path).unwrap();
            let builder =
                ParquetRecordBatchReaderBuilder::new_with_metadata(file, metadata.clone());
            let schema = builder.schema();
            black_box(schema);
        });
    });

    c.bench_function("readSpanName", |b| {
        b.iter(|| {
            let file = File::open(&file_path).unwrap();
            let builder =
                ParquetRecordBatchReaderBuilder::new_with_metadata(file, metadata.clone());

            // Create ProjectionMask to only read the span.name column
            let schema = builder.parquet_schema();
            let span_name_path = "rs.list.element.ss.list.element.Spans.list.element.Name";
            let name_idx = schema
                .columns()
                .iter()
                .position(|c| c.path().string() == span_name_path)
                .expect("span name column not found");

            let projection = ProjectionMask::leaves(schema, vec![name_idx]);
            let mut reader = builder.with_projection(projection).build().unwrap();

            while let Some(Ok(batch)) = reader.next() {
                black_box(batch);
            }
        });
    });

    c.bench_function("createMask", |b| {
        b.iter(|| {
            let file = File::open(&file_path).unwrap();
            let builder =
                ParquetRecordBatchReaderBuilder::new_with_metadata(file, metadata.clone());

            // Create ProjectionMask to only read the span.name column
            let schema = builder.parquet_schema();
            let span_name_path = "rs.list.element.ss.list.element.Spans.list.element.Name";
            let name_idx = schema
                .columns()
                .iter()
                .position(|c| c.path().string() == span_name_path)
                .expect("span name column not found");

            let projection = ProjectionMask::leaves(schema, vec![name_idx]);
            let mut reader = builder.with_projection(projection).build().unwrap();

            while let Some(Ok(batch)) = reader.next() {
                // Navigate: rs -> element (struct) -> ss -> element (struct) -> Spans -> element (struct) -> Name
                let rs_column = batch.column_by_name("rs").unwrap();
                let rs_list = rs_column.as_any().downcast_ref::<ListArray>().unwrap();
                let rs_values = rs_list.values();
                let rs_struct = rs_values.as_any().downcast_ref::<StructArray>().unwrap();

                // Get the "ss" field from rs struct
                let ss_column = rs_struct.column_by_name("ss").unwrap();
                let ss_list = ss_column.as_any().downcast_ref::<ListArray>().unwrap();
                let ss_values = ss_list.values();
                let ss_struct = ss_values.as_any().downcast_ref::<StructArray>().unwrap();

                // Get the "Spans" field from ss struct
                let spans_column = ss_struct.column_by_name("Spans").unwrap();
                let spans_list = spans_column.as_any().downcast_ref::<ListArray>().unwrap();
                let spans_values = spans_list.values();
                let spans_struct = spans_values.as_any().downcast_ref::<StructArray>().unwrap();

                // Get the "Name" field from spans struct
                let name_column = spans_struct.column_by_name("Name").unwrap();
                let name_array = name_column.as_any().downcast_ref::<StringArray>().unwrap();

                // Create comparison mask
                let scalar = StringArray::new_scalar("distributor.ConsumeTraces");
                let mask = cmp::eq(name_array, &scalar).unwrap();
                black_box(mask);
            }
        });
    });

    c.bench_function("useRowFilter", |b| {
        b.iter(|| {
            let file = File::open(&file_path).unwrap();
            let builder =
                ParquetRecordBatchReaderBuilder::new_with_metadata(file, metadata.clone());

            // Create ProjectionMask to only read the span.name column
            let schema = builder.parquet_schema();
            let span_name_path = "rs.list.element.ss.list.element.Spans.list.element.Name";
            let name_idx = schema
                .columns()
                .iter()
                .position(|c| c.path().string() == span_name_path)
                .expect("span name column not found");
            let projection = ProjectionMask::leaves(schema, vec![name_idx]);
            let projection2 = ProjectionMask::leaves(schema, vec![name_idx]);

            let predicate = ArrowPredicateFn::new(projection, |batch| {
                // The predicate receives a projected batch with only the name column
                // But the column structure still follows the nested schema
                // Navigate: rs -> element (struct) -> ss -> element (struct) -> Spans -> element (struct) -> Name
                let rs_column = batch.column_by_name("rs").ok_or_else(|| {
                    arrow::error::ArrowError::ComputeError("rs column not found".to_string())
                })?;
                let rs_list = rs_column
                    .as_any()
                    .downcast_ref::<ListArray>()
                    .ok_or_else(|| {
                        arrow::error::ArrowError::ComputeError(
                            "Failed to downcast rs to ListArray".to_string(),
                        )
                    })?;
                let rs_values = rs_list.values();
                let rs_struct = rs_values
                    .as_any()
                    .downcast_ref::<StructArray>()
                    .ok_or_else(|| {
                        arrow::error::ArrowError::ComputeError(
                            "Failed to downcast rs values to StructArray".to_string(),
                        )
                    })?;

                // Get the "ss" field from rs struct
                let ss_column = rs_struct.column_by_name("ss").ok_or_else(|| {
                    arrow::error::ArrowError::ComputeError("ss column not found".to_string())
                })?;
                let ss_list = ss_column
                    .as_any()
                    .downcast_ref::<ListArray>()
                    .ok_or_else(|| {
                        arrow::error::ArrowError::ComputeError(
                            "Failed to downcast ss to ListArray".to_string(),
                        )
                    })?;
                let ss_values = ss_list.values();
                let ss_struct = ss_values
                    .as_any()
                    .downcast_ref::<StructArray>()
                    .ok_or_else(|| {
                        arrow::error::ArrowError::ComputeError(
                            "Failed to downcast ss values to StructArray".to_string(),
                        )
                    })?;

                // Get the "Spans" field from ss struct
                let spans_column = ss_struct.column_by_name("Spans").ok_or_else(|| {
                    arrow::error::ArrowError::ComputeError("Spans column not found".to_string())
                })?;
                let spans_list = spans_column
                    .as_any()
                    .downcast_ref::<ListArray>()
                    .ok_or_else(|| {
                        arrow::error::ArrowError::ComputeError(
                            "Failed to downcast Spans to ListArray".to_string(),
                        )
                    })?;
                let spans_values = spans_list.values();
                let spans_struct = spans_values
                    .as_any()
                    .downcast_ref::<StructArray>()
                    .ok_or_else(|| {
                        arrow::error::ArrowError::ComputeError(
                            "Failed to downcast Spans values to StructArray".to_string(),
                        )
                    })?;

                // Get the "Name" field from spans struct
                let name_column = spans_struct.column_by_name("Name").ok_or_else(|| {
                    arrow::error::ArrowError::ComputeError("Name column not found".to_string())
                })?;
                let name_array = name_column
                    .as_any()
                    .downcast_ref::<StringArray>()
                    .ok_or_else(|| {
                        arrow::error::ArrowError::ComputeError(
                            "Failed to downcast Name to StringArray".to_string(),
                        )
                    })?;

                // Compare all span names to the target
                let scalar = StringArray::new_scalar("distributor.ConsumeTraces");
                let span_matches = cmp::eq(name_array, &scalar)?;

                // Now aggregate to row level: for each row, check if ANY of its spans match
                // We need to traverse back through the list offsets to map spans to rows
                let num_rows = batch.num_rows();
                let mut row_matches = vec![false; num_rows];

                for row_idx in 0..num_rows {
                    // Get rs list range for this row
                    let rs_start = rs_list.value_offsets()[row_idx] as usize;
                    let rs_end = rs_list.value_offsets()[row_idx + 1] as usize;

                    for rs_elem_idx in rs_start..rs_end {
                        // Get ss list range for this rs element
                        let ss_start = ss_list.value_offsets()[rs_elem_idx] as usize;
                        let ss_end = ss_list.value_offsets()[rs_elem_idx + 1] as usize;

                        for ss_elem_idx in ss_start..ss_end {
                            // Get Spans list range for this ss element
                            let spans_start = spans_list.value_offsets()[ss_elem_idx] as usize;
                            let spans_end = spans_list.value_offsets()[ss_elem_idx + 1] as usize;

                            // Check if any span in this range matches
                            for span_idx in spans_start..spans_end {
                                if span_matches.value(span_idx) {
                                    row_matches[row_idx] = true;
                                    break;
                                }
                            }
                            if row_matches[row_idx] {
                                break;
                            }
                        }
                        if row_matches[row_idx] {
                            break;
                        }
                    }
                }

                Ok(BooleanArray::from(row_matches))
            });
            let filter = RowFilter::new(vec![Box::new(predicate)]);

            let mut reader = builder
                .with_row_filter(filter)
                .with_projection(projection2)
                .build()
                .unwrap();

            while let Some(Ok(batch)) = reader.next() {
                // Navigate: rs -> element (struct) -> ss -> element (struct) -> Spans -> element (struct) -> Name
                black_box(batch);
            }
        });
    });

    // Alternative: Read projected data and filter in memory (likely faster for nested data)
    c.bench_function("readAndFilter", |b| {
        b.iter(|| {
            let file = File::open(&file_path).unwrap();
            let builder =
                ParquetRecordBatchReaderBuilder::new_with_metadata(file, metadata.clone());

            let schema = builder.parquet_schema();
            let span_name_path = "rs.list.element.ss.list.element.Spans.list.element.Name";
            let name_idx = schema
                .columns()
                .iter()
                .position(|c| c.path().string() == span_name_path)
                .expect("span name column not found");

            let projection = ProjectionMask::leaves(schema, vec![name_idx]);
            let mut reader = builder.with_projection(projection).build().unwrap();

            let mut matching_rows = 0;
            while let Some(Ok(batch)) = reader.next() {
                // Navigate to get the name array (same as createMask benchmark)
                let rs_column = batch.column_by_name("rs").unwrap();
                let rs_list = rs_column.as_any().downcast_ref::<ListArray>().unwrap();
                let rs_values = rs_list.values();
                let rs_struct = rs_values.as_any().downcast_ref::<StructArray>().unwrap();

                let ss_column = rs_struct.column_by_name("ss").unwrap();
                let ss_list = ss_column.as_any().downcast_ref::<ListArray>().unwrap();
                let ss_values = ss_list.values();
                let ss_struct = ss_values.as_any().downcast_ref::<StructArray>().unwrap();

                let spans_column = ss_struct.column_by_name("Spans").unwrap();
                let spans_list = spans_column.as_any().downcast_ref::<ListArray>().unwrap();
                let spans_values = spans_list.values();
                let spans_struct = spans_values.as_any().downcast_ref::<StructArray>().unwrap();

                let name_column = spans_struct.column_by_name("Name").unwrap();
                let name_array = name_column.as_any().downcast_ref::<StringArray>().unwrap();

                // Create comparison mask
                let scalar = StringArray::new_scalar("distributor.ConsumeTraces");
                let span_mask = cmp::eq(name_array, &scalar).unwrap();

                // Count matching spans (don't need to aggregate to rows for this benchmark)
                matching_rows += span_mask.true_count();
            }
            black_box(matching_rows);
        });
    });

    // Dictionary-based filtering - decodes and checks dictionary for target value
    c.bench_function("dictionaryLookup", |b| {
        b.iter(|| {
            let file = File::open(&file_path).unwrap();
            let reader = SerializedFileReader::new(file).unwrap();
            let target_name = "distributor.ConsumeTraces";
            let target_bytes = target_name.as_bytes();

            let schema = reader.metadata().file_metadata().schema_descr();
            let span_name_path = "rs.list.element.ss.list.element.Spans.list.element.Name";
            let name_col_idx = schema
                .columns()
                .iter()
                .position(|c| c.path().string() == span_name_path)
                .expect("Name column not found");

            let column_desc = schema.column(name_col_idx);

            let mut matching_row_groups = Vec::new();
            let mut skipped_by_dict = 0;
            let mut no_dict = 0;

            for rg_idx in 0..reader.metadata().num_row_groups() {
                let row_group = reader.get_row_group(rg_idx).unwrap();
                let mut page_reader = row_group.get_column_page_reader(name_col_idx).unwrap();

                match check_dictionary_contains_value(
                    &mut page_reader,
                    column_desc.clone(),
                    target_bytes,
                ) {
                    Ok(Some(true)) => {
                        // Target found in dictionary - might exist in this row group
                        matching_row_groups.push(rg_idx);
                    }
                    Ok(Some(false)) => {
                        // Target NOT in dictionary - skip this row group!
                        skipped_by_dict += 1;
                    }
                    Ok(None) => {
                        // No dictionary encoding - conservatively include
                        matching_row_groups.push(rg_idx);
                        no_dict += 1;
                    }
                    Err(_) => {
                        // Error - conservatively include
                        matching_row_groups.push(rg_idx);
                    }
                }
            }

            black_box((matching_row_groups.len(), skipped_by_dict, no_dict));
        });
    });

    // Metadata-based filtering using column chunk statistics and encoding info
    c.bench_function("metadataFilter", |b| {
        b.iter(|| {
            let file = File::open(&file_path).unwrap();
            let reader = SerializedFileReader::new(file).unwrap();
            let target_name = "distributor.ConsumeTraces";

            // Find the Name column index
            let schema = reader.metadata().file_metadata().schema_descr();
            let span_name_path = "rs.list.element.ss.list.element.Spans.list.element.Name";
            let name_col_idx = schema
                .columns()
                .iter()
                .position(|c| c.path().string() == span_name_path)
                .expect("Name column not found");

            let mut matching_row_groups = Vec::new();
            let mut encoding_info = Vec::new();

            // Check each row group's column chunk metadata
            for rg_idx in 0..reader.metadata().num_row_groups() {
                let row_group = reader.metadata().row_group(rg_idx);
                let col_chunk = row_group.column(name_col_idx);

                // Check encoding types to see if dictionary is used
                let encodings = col_chunk.encodings();
                let has_dict = encodings.contains(&Encoding::PLAIN_DICTIONARY)
                    || encodings.contains(&Encoding::RLE_DICTIONARY);
                encoding_info.push((rg_idx, has_dict, encodings.clone()));

                // Use statistics for filtering (min/max already being used)
                if let Some(stats) = col_chunk.statistics() {
                    let target_bytes = target_name.as_bytes();

                    // Skip if target is outside min/max range
                    if let (Some(min), Some(max)) = (stats.min_bytes_opt(), stats.max_bytes_opt()) {
                        if target_bytes < min || target_bytes > max {
                            continue; // Skip this row group
                        }
                    }

                    // Check null count - if all values are null, skip
                    if stats.null_count_opt() == Some(col_chunk.num_values() as u64) {
                        continue;
                    }
                }

                matching_row_groups.push(rg_idx);
            }

            assert_eq!(0, matching_row_groups.len());
            black_box((matching_row_groups, encoding_info));
        });
    });

    // Combined approach: Use metadata filtering + Arrow reader for matching row groups only
    c.bench_function("metadataThenRead", |b| {
        b.iter(|| {
            let target_name = "distributor.ConsumeTraces";

            // Step 1: Use metadata-based filtering to identify matching row groups
            let file = File::open(&file_path).unwrap();
            let reader = SerializedFileReader::new(file).unwrap();

            let schema = reader.metadata().file_metadata().schema_descr();
            let span_name_path = "rs.list.element.ss.list.element.Spans.list.element.Name";
            let name_col_idx = schema
                .columns()
                .iter()
                .position(|c| c.path().string() == span_name_path)
                .expect("Name column not found");

            let mut matching_row_groups = Vec::new();
            let target_bytes = target_name.as_bytes();

            // Use statistics to filter row groups
            for rg_idx in 0..reader.metadata().num_row_groups() {
                let row_group = reader.metadata().row_group(rg_idx);
                let col_chunk = row_group.column(name_col_idx);

                let mut should_read = true;

                if let Some(stats) = col_chunk.statistics() {
                    // Skip if target is outside min/max range
                    if let (Some(min), Some(max)) = (stats.min_bytes_opt(), stats.max_bytes_opt()) {
                        if target_bytes < min || target_bytes > max {
                            should_read = false;
                        }
                    }

                    // Skip if all values are null
                    if stats.null_count_opt() == Some(col_chunk.num_values() as u64) {
                        should_read = false;
                    }
                }

                if should_read {
                    matching_row_groups.push(rg_idx);
                }
            }

            drop(reader); // Close the file so we can reopen it

            // Step 2: Read only the matching row groups with Arrow reader
            if !matching_row_groups.is_empty() {
                let file = File::open(&file_path).unwrap();
                let builder =
                    ParquetRecordBatchReaderBuilder::new_with_metadata(file, metadata.clone());

                let schema = builder.parquet_schema();
                let name_idx = schema
                    .columns()
                    .iter()
                    .position(|c| c.path().string() == span_name_path)
                    .expect("span name column not found");

                let projection = ProjectionMask::leaves(schema, vec![name_idx]);
                let mut reader = builder
                    .with_row_groups(matching_row_groups)
                    .with_projection(projection)
                    .build()
                    .unwrap();

                let mut matching_spans = 0;
                while let Some(Ok(batch)) = reader.next() {
                    let rs_column = batch.column_by_name("rs").unwrap();
                    let rs_list = rs_column.as_any().downcast_ref::<ListArray>().unwrap();
                    let rs_values = rs_list.values();
                    let rs_struct = rs_values.as_any().downcast_ref::<StructArray>().unwrap();

                    let ss_column = rs_struct.column_by_name("ss").unwrap();
                    let ss_list = ss_column.as_any().downcast_ref::<ListArray>().unwrap();
                    let ss_values = ss_list.values();
                    let ss_struct = ss_values.as_any().downcast_ref::<StructArray>().unwrap();

                    let spans_column = ss_struct.column_by_name("Spans").unwrap();
                    let spans_list = spans_column.as_any().downcast_ref::<ListArray>().unwrap();
                    let spans_values = spans_list.values();
                    let spans_struct = spans_values.as_any().downcast_ref::<StructArray>().unwrap();

                    let name_column = spans_struct.column_by_name("Name").unwrap();
                    let name_array = name_column.as_any().downcast_ref::<StringArray>().unwrap();

                    let scalar = StringArray::new_scalar(target_name);
                    let span_mask = cmp::eq(name_array, &scalar).unwrap();
                    matching_spans += span_mask.true_count();
                }
                black_box(matching_spans);
            }
        });
    });
}

criterion_group!(benches, parquet_operations);
criterion_main!(benches);
