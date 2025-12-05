//! Integration tests for vParquet4 reader using real test data

use arrow::array::Array;
use vparquet4::reader::{ReaderConfig, VParquet4Reader, VParquet4ReaderTrait};
use vparquet4::schema::field_paths::trace;

/// Path to the test data file (relative to workspace root)
fn test_data_path() -> std::path::PathBuf {
    // Get the workspace root (2 levels up from the crate)
    let manifest_dir = std::env::var("CARGO_MANIFEST_DIR").unwrap();
    let crate_dir = std::path::PathBuf::from(manifest_dir);
    let workspace_root = crate_dir.parent().unwrap().parent().unwrap();
    workspace_root.join("tempodb/encoding/vparquet4/test-data/single-tenant/b27b0e53-66a0-4505-afd6-434ae3cd4a10/data.parquet")
}

#[test]
fn test_open_and_validate_schema() {
    // Open the test file with schema validation enabled
    let reader = VParquet4Reader::open(test_data_path(), ReaderConfig::default())
        .expect("Failed to open test file");

    // Verify we can read metadata
    assert!(reader.num_row_groups() > 0, "Should have at least one row group");
    assert!(reader.num_rows() > 0, "Should have at least one row");

    println!("File has {} row groups", reader.num_row_groups());
    println!("File has {} total rows", reader.num_rows());
}

#[test]
fn test_read_trace_count() {
    // Open the test file
    let mut reader = VParquet4Reader::open(test_data_path(), ReaderConfig::default())
        .expect("Failed to open test file");

    // According to meta.json, this file contains 134 traces
    let expected_trace_count = 134;

    // Count traces by reading all row groups
    let mut total_traces = 0;
    for i in 0..reader.num_row_groups() {
        let batch = reader.read_row_group(i).expect("Failed to read row group");
        total_traces += batch.num_rows();
    }

    println!("Found {} traces (expected {})", total_traces, expected_trace_count);
    assert_eq!(
        total_traces, expected_trace_count,
        "Should have exactly {} traces",
        expected_trace_count
    );
}

#[test]
fn test_read_trace_ids() {
    // Open the test file
    let mut reader = VParquet4Reader::open(test_data_path(), ReaderConfig::default())
        .expect("Failed to open test file");

    // Read the first row group
    let batch = reader.read_row_group(0).expect("Failed to read row group");

    // Find the TraceID column
    let schema = batch.schema();
    let trace_id_idx = schema
        .index_of(trace::TRACE_ID)
        .expect("Should have TraceID column");

    let trace_id_array = batch
        .column(trace_id_idx)
        .as_any()
        .downcast_ref::<arrow::array::BinaryArray>()
        .expect("TraceID should be BinaryArray");

    // Verify TraceIDs are 16 bytes each
    for i in 0..trace_id_array.len() {
        let trace_id = trace_id_array.value(i);
        assert_eq!(
            trace_id.len(),
            16,
            "TraceID should be exactly 16 bytes, got {}",
            trace_id.len()
        );
    }

    println!("Verified {} trace IDs", trace_id_array.len());
}

#[test]
fn test_read_timestamps() {
    // Open the test file
    let mut reader = VParquet4Reader::open(test_data_path(), ReaderConfig::default())
        .expect("Failed to open test file");

    // Read the first row group
    let batch = reader.read_row_group(0).expect("Failed to read row group");

    // Find timestamp columns
    let schema = batch.schema();
    let start_time_idx = schema
        .index_of(trace::START_TIME_UNIX_NANO)
        .expect("Should have StartTimeUnixNano column");
    let end_time_idx = schema
        .index_of(trace::END_TIME_UNIX_NANO)
        .expect("Should have EndTimeUnixNano column");
    let duration_idx = schema
        .index_of(trace::DURATION_NANO)
        .expect("Should have DurationNano column");

    let start_times = batch
        .column(start_time_idx)
        .as_any()
        .downcast_ref::<arrow::array::UInt64Array>()
        .expect("StartTimeUnixNano should be UInt64Array");

    let end_times = batch
        .column(end_time_idx)
        .as_any()
        .downcast_ref::<arrow::array::UInt64Array>()
        .expect("EndTimeUnixNano should be UInt64Array");

    let durations = batch
        .column(duration_idx)
        .as_any()
        .downcast_ref::<arrow::array::UInt64Array>()
        .expect("DurationNano should be UInt64Array");

    // Verify timestamps are sensible
    for i in 0..start_times.len() {
        let start = start_times.value(i);
        let end = end_times.value(i);
        let duration = durations.value(i);

        assert!(start > 0, "Start time should be positive");
        assert!(end >= start, "End time should be >= start time");
        assert_eq!(
            end - start,
            duration,
            "Duration should equal end - start"
        );
    }

    println!("Verified {} timestamp records", start_times.len());
}

#[test]
fn test_read_root_metadata() {
    // Open the test file
    let mut reader = VParquet4Reader::open(test_data_path(), ReaderConfig::default())
        .expect("Failed to open test file");

    // Read the first row group
    let batch = reader.read_row_group(0).expect("Failed to read row group");

    // Find root metadata columns
    let schema = batch.schema();
    let root_service_idx = schema
        .index_of(trace::ROOT_SERVICE_NAME)
        .expect("Should have RootServiceName column");
    let root_span_idx = schema
        .index_of(trace::ROOT_SPAN_NAME)
        .expect("Should have RootSpanName column");

    let root_services = batch
        .column(root_service_idx)
        .as_any()
        .downcast_ref::<arrow::array::StringArray>()
        .expect("RootServiceName should be StringArray");

    let root_spans = batch
        .column(root_span_idx)
        .as_any()
        .downcast_ref::<arrow::array::StringArray>()
        .expect("RootSpanName should be StringArray");

    // Print some examples
    println!("Sample root metadata:");
    for i in 0..std::cmp::min(5, root_services.len()) {
        let service = root_services.value(i);
        let span = root_spans.value(i);
        println!("  Trace {}: service='{}', span='{}'", i, service, span);
    }
}

#[test]
fn test_streaming_reader() {
    // Open the test file
    let reader = VParquet4Reader::open(test_data_path(), ReaderConfig::default())
        .expect("Failed to open test file");

    // Convert to streaming reader
    let mut stream = reader.into_stream().expect("Failed to create stream");

    // Read all batches
    let mut total_rows = 0;
    let mut batch_count = 0;
    while let Some(batch) = stream.next() {
        let batch = batch.expect("Failed to read batch");
        total_rows += batch.num_rows();
        batch_count += 1;
    }

    println!("Read {} batches with {} total rows", batch_count, total_rows);
    assert_eq!(total_rows, 134, "Should have 134 total rows");
}

#[test]
fn test_metadata_access() {
    // Open the test file
    let reader = VParquet4Reader::open(test_data_path(), ReaderConfig::default())
        .expect("Failed to open test file");

    let metadata = reader.metadata();

    println!("Parquet metadata:");
    println!("  Version: {:?}", metadata.file_metadata().version());
    println!("  Created by: {:?}", metadata.file_metadata().created_by());
    println!("  Num row groups: {}", metadata.num_row_groups());
    println!("  Schema: {:?}", metadata.file_metadata().schema());

    // Verify row group statistics
    for (i, rg) in metadata.row_groups().iter().enumerate() {
        println!("  Row group {}: {} rows, {} bytes", i, rg.num_rows(), rg.total_byte_size());
    }
}
