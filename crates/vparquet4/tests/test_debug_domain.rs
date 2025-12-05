//! Debug test to understand the Arrow data structure

use vparquet4::{ReaderConfig, VParquet4Reader, VParquet4ReaderTrait};

const TEST_DATA_PATH: &str =
    "../../tempodb/encoding/vparquet4/test-data/single-tenant/b27b0e53-66a0-4505-afd6-434ae3cd4a10/data.parquet";

#[test]
fn debug_batch_structure() {
    let mut reader = VParquet4Reader::open(TEST_DATA_PATH, ReaderConfig::default())
        .expect("Failed to open test data file");

    let batch = reader
        .read_row_group(0)
        .expect("Failed to read row group");

    println!("Batch schema:");
    println!("{}", batch.schema());

    println!("\nBatch has {} rows", batch.num_rows());
    println!("Batch has {} columns", batch.num_columns());

    println!("\nColumn names:");
    for field in batch.schema().fields() {
        println!("  - {} ({})", field.name(), field.data_type());
    }

    // Try to find ResourceSpans column
    if let Some(rs_col) = batch.column_by_name("rs") {
        println!("\nFound 'rs' column");
        println!("Type: {:?}", rs_col.data_type());
        println!("Null count: {}", rs_col.null_count());
        println!("Length: {}", rs_col.len());
    } else {
        println!("\nNo 'rs' column found");
    }

    if let Some(rs_col) = batch.column_by_name("ResourceSpans") {
        println!("\nFound 'ResourceSpans' column");
        println!("Type: {:?}", rs_col.data_type());
    } else {
        println!("\nNo 'ResourceSpans' column found");
    }
}
