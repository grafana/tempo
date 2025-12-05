//! Debug test to see actual schema of test data

use vparquet4::reader::{ReaderConfig, VParquet4Reader};

/// Path to the test data file (relative to workspace root)
fn test_data_path() -> std::path::PathBuf {
    let manifest_dir = std::env::var("CARGO_MANIFEST_DIR").unwrap();
    let crate_dir = std::path::PathBuf::from(manifest_dir);
    let workspace_root = crate_dir.parent().unwrap().parent().unwrap();
    workspace_root.join("tempodb/encoding/vparquet4/test-data/single-tenant/b27b0e53-66a0-4505-afd6-434ae3cd4a10/data.parquet")
}

#[test]
fn debug_actual_schema() {
    // Open without validation
    let mut config = ReaderConfig::default();
    config.validate_schema = false;

    let reader = VParquet4Reader::open(test_data_path(), config)
        .expect("Failed to open test file");

    let schema = reader.arrow_schema().expect("Failed to get schema");

    println!("Actual schema has {} fields:", schema.fields().len());
    for field in schema.fields() {
        println!("  - {}: {:?} (nullable: {})", field.name(), field.data_type(), field.is_nullable());
    }
}
