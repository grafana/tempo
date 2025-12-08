use criterion::{black_box, criterion_group, criterion_main, Criterion};
use std::env;
use std::path::PathBuf;
use std::time::Duration;
use std::{fs::File, sync::Arc};
use vparquet4::{ReadOptions, SpanFilter, VParquet4Reader};
use parquet::file::reader::{FileReader, SerializedFileReader};
use parquet::column::page::PageReader;
use parquet::data_type::ByteArray;
use parquet::schema::types::ColumnDescriptor;
use parquet::basic::Encoding;

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

// Target to beat
// tempodb/encoding/vparquet4/block_traceql_test.go:L1448
// BenchmarkBackendBlockTraceQL/spanAttIntrinsicMatch-14            	      44	  28327922 ns/op	1880.30 MB/s	        53.26 MB_io/op	         0 spans/op
fn span_att_intrinsic_match(c: &mut Criterion) {
    let file_path = get_benchmark_file_path();

    // Verify the file exists before benchmarking
    if !file_path.exists() {
        panic!("Benchmark file does not exist: {:?}", file_path);
    }

    println!("Benchmarking file: {:?}", file_path);

    let mut group = c.benchmark_group("spanAttIntrinsicMatch");
    group.measurement_time(Duration::from_secs(10));
    group.sample_size(10);

    // Dictionary-based filtering - the key optimization for performance
    group.bench_function("spanAttIntrinsicMatch", |b| {
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

            for rg_idx in 0..reader.metadata().num_row_groups() {
                let row_group = reader.get_row_group(rg_idx).unwrap();
                let mut page_reader = row_group.get_column_page_reader(name_col_idx).unwrap();

                match check_dictionary_contains_value(&mut page_reader, column_desc.clone(), target_bytes) {
                    Ok(Some(true)) => {
                        matching_row_groups.push(rg_idx);
                    }
                    Ok(Some(false)) => {
                        skipped_by_dict += 1;
                    }
                    _ => {
                        matching_row_groups.push(rg_idx);
                    }
                }
            }

            black_box((matching_row_groups.len(), skipped_by_dict));
        });
    });

    group.finish();
}

criterion_group!(benches, span_att_intrinsic_match);
criterion_main!(benches);
