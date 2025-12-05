//! Integration tests for column projection

use vparquet4::projection::{ProjectionBuilder, ProjectionMode};
use vparquet4::reader::{ReaderConfig, VParquet4Reader, VParquet4ReaderTrait};
use vparquet4::schema::field_paths::{resource, span, trace};

/// Path to the test data file (relative to workspace root)
fn test_data_path() -> std::path::PathBuf {
    let manifest_dir = std::env::var("CARGO_MANIFEST_DIR").unwrap();
    let crate_dir = std::path::PathBuf::from(manifest_dir);
    let workspace_root = crate_dir.parent().unwrap().parent().unwrap();
    workspace_root.join("tempodb/encoding/vparquet4/test-data/single-tenant/b27b0e53-66a0-4505-afd6-434ae3cd4a10/data.parquet")
}

#[test]
fn test_trace_summary_only_projection() {
    // Open the test file
    let reader = VParquet4Reader::open(test_data_path(), ReaderConfig::default())
        .expect("Failed to open test file");

    // Get the schema
    let arrow_schema = reader.arrow_schema().expect("Failed to get schema");
    let parquet_schema = reader.metadata().file_metadata().schema_descr();

    // Build trace summary projection
    let projection = ProjectionBuilder::trace_summary_only();
    let mask = projection.build(&arrow_schema, parquet_schema).expect("Failed to build projection");

    println!("Trace summary projection mode: {:?}", projection.mode());
    println!("Selected columns: {:?}", projection.columns());

    // Verify the projection was created
    assert!(mask.is_some(), "Should create a projection mask");
    assert_eq!(projection.mode(), ProjectionMode::TraceSummaryOnly);

    // Verify expected columns are included
    assert!(projection.columns().contains(trace::TRACE_ID));
    assert!(projection.columns().contains(trace::START_TIME_UNIX_NANO));
    assert!(projection.columns().contains(trace::END_TIME_UNIX_NANO));
    assert!(projection.columns().contains(trace::ROOT_SERVICE_NAME));
    assert!(projection.columns().contains(trace::ROOT_SPAN_NAME));

    // Create a new reader with the projection
    let mut reader = VParquet4Reader::open(test_data_path(), ReaderConfig::default())
        .expect("Failed to open test file");

    if let Some(mask) = mask {
        reader = reader.with_projection(mask);
    }

    // Read a row group with the projection
    let batch = reader.read_row_group(0).expect("Failed to read row group");

    println!("Read batch with {} columns and {} rows",
             batch.num_columns(), batch.num_rows());

    // Verify we have the trace ID column
    let schema = batch.schema();
    assert!(schema.index_of(trace::TRACE_ID).is_ok(),
            "Should have TraceID column");

    // Count how many columns we got vs total schema
    let total_fields = arrow_schema.fields().len();
    let projected_fields = batch.num_columns();

    println!("Projected {}/{} columns", projected_fields, total_fields);

    // Trace summary should be significantly fewer columns than the full schema
    assert!(projected_fields < total_fields,
            "Projected columns should be less than total");
}

#[test]
fn test_spans_without_attrs_projection() {
    // Open the test file
    let reader = VParquet4Reader::open(test_data_path(), ReaderConfig::default())
        .expect("Failed to open test file");

    let arrow_schema = reader.arrow_schema().expect("Failed to get schema");
    let parquet_schema = reader.metadata().file_metadata().schema_descr();

    // Build spans without attrs projection
    let projection = ProjectionBuilder::spans_without_attrs();
    let mask = projection.build(&arrow_schema, parquet_schema).expect("Failed to build projection");

    println!("Spans without attrs projection mode: {:?}", projection.mode());

    assert!(mask.is_some(), "Should create a projection mask");
    assert_eq!(projection.mode(), ProjectionMode::SpansWithoutAttrs);

    // Verify trace columns and ResourceSpans are included
    assert!(projection.columns().contains(trace::TRACE_ID));
    assert!(projection.columns().contains(trace::RESOURCE_SPANS));

    // Note: Individual span fields like SpanID, Name are nested inside ResourceSpans
    // and cannot be selected individually at the top level. The entire "rs" column
    // must be included to access spans.

    // Read with projection
    let mut reader = VParquet4Reader::open(test_data_path(), ReaderConfig::default())
        .expect("Failed to open test file");

    if let Some(mask) = mask {
        reader = reader.with_projection(mask);
    }

    let batch = reader.read_row_group(0).expect("Failed to read row group");

    println!("Read batch with {} columns and {} rows",
             batch.num_columns(), batch.num_rows());

    // Verify we have the ResourceSpans column (contains all span data)
    let schema = batch.schema();
    assert!(schema.index_of(trace::RESOURCE_SPANS).is_ok(),
            "Should have ResourceSpans (rs) column");

    // This projection includes all 9 top-level fields, same as full spans
    // The difference would be in how we parse the nested data (Phase 3)
    let total_fields = arrow_schema.fields().len();
    let projected_fields = batch.num_columns();

    println!("Projected {}/{} columns", projected_fields, total_fields);
    // Since we include all 9 top-level fields, this is the same as full spans
    assert_eq!(projected_fields, total_fields,
            "Spans without attrs includes all top-level fields (attributes filtered in Phase 3)");
}

#[test]
fn test_full_spans_projection() {
    // Open the test file
    let reader = VParquet4Reader::open(test_data_path(), ReaderConfig::default())
        .expect("Failed to open test file");

    let arrow_schema = reader.arrow_schema().expect("Failed to get schema");
    let parquet_schema = reader.metadata().file_metadata().schema_descr();

    // Build full spans projection (should be same as no projection)
    let projection = ProjectionBuilder::full_spans();
    let mask = projection.build(&arrow_schema, parquet_schema).expect("Failed to build projection");

    println!("Full spans projection mode: {:?}", projection.mode());

    // Full spans should not create a projection mask (read all columns)
    assert!(mask.is_none(), "Full spans should not create a projection mask");
    assert_eq!(projection.mode(), ProjectionMode::FullSpans);

    // Read without projection
    let mut reader = VParquet4Reader::open(test_data_path(), ReaderConfig::default())
        .expect("Failed to open test file");

    let batch = reader.read_row_group(0).expect("Failed to read row group");

    println!("Read batch with {} columns and {} rows",
             batch.num_columns(), batch.num_rows());

    // Should have all columns
    let total_fields = arrow_schema.fields().len();
    let projected_fields = batch.num_columns();

    println!("Read {}/{} columns", projected_fields, total_fields);
}

#[test]
fn test_custom_projection() {
    // Open the test file
    let reader = VParquet4Reader::open(test_data_path(), ReaderConfig::default())
        .expect("Failed to open test file");

    let arrow_schema = reader.arrow_schema().expect("Failed to get schema");
    let parquet_schema = reader.metadata().file_metadata().schema_descr();

    // Build custom projection with specific columns
    let projection = ProjectionBuilder::new()
        .add_column(trace::TRACE_ID)
        .add_column(trace::START_TIME_UNIX_NANO)
        .add_column(span::NAME);

    let mask = projection.build(&arrow_schema, parquet_schema).expect("Failed to build projection");

    println!("Custom projection columns: {:?}", projection.columns());

    assert!(mask.is_some(), "Should create a projection mask");
    assert_eq!(projection.columns().len(), 3);

    // Read with custom projection
    let mut reader = VParquet4Reader::open(test_data_path(), ReaderConfig::default())
        .expect("Failed to open test file");

    if let Some(mask) = mask {
        reader = reader.with_projection(mask);
    }

    let batch = reader.read_row_group(0).expect("Failed to read row group");

    println!("Read batch with {} columns and {} rows",
             batch.num_columns(), batch.num_rows());

    // Should have fewer columns
    let total_fields = arrow_schema.fields().len();
    let projected_fields = batch.num_columns();

    println!("Projected {}/{} columns", projected_fields, total_fields);
    assert!(projected_fields < total_fields,
            "Custom projection should result in fewer columns");
}

#[test]
fn test_empty_projection() {
    // Open the test file
    let reader = VParquet4Reader::open(test_data_path(), ReaderConfig::default())
        .expect("Failed to open test file");

    let arrow_schema = reader.arrow_schema().expect("Failed to get schema");
    let parquet_schema = reader.metadata().file_metadata().schema_descr();

    // Empty projection builder should read all columns
    let projection = ProjectionBuilder::new();
    let mask = projection.build(&arrow_schema, parquet_schema).expect("Failed to build projection");

    println!("Empty projection mode: {:?}", projection.mode());

    // Empty projection should not create a mask (read all)
    assert!(mask.is_none(), "Empty projection should not create a projection mask");
}

#[test]
fn test_projection_with_multiple_columns() {
    // Open the test file
    let reader = VParquet4Reader::open(test_data_path(), ReaderConfig::default())
        .expect("Failed to open test file");

    let arrow_schema = reader.arrow_schema().expect("Failed to get schema");
    let parquet_schema = reader.metadata().file_metadata().schema_descr();

    // Build projection with multiple columns using add_columns
    let columns = vec![
        trace::TRACE_ID.to_string(),
        trace::START_TIME_UNIX_NANO.to_string(),
        trace::END_TIME_UNIX_NANO.to_string(),
        span::SPAN_ID.to_string(),
        span::NAME.to_string(),
    ];

    let projection = ProjectionBuilder::new()
        .add_columns(columns.clone());

    let mask = projection.build(&arrow_schema, parquet_schema).expect("Failed to build projection");

    println!("Multi-column projection: {:?}", projection.columns());

    assert!(mask.is_some(), "Should create a projection mask");
    assert_eq!(projection.columns().len(), columns.len());

    // Verify all requested columns are in the projection
    for col in &columns {
        assert!(projection.columns().contains(col),
                "Should contain column {}", col);
    }
}
