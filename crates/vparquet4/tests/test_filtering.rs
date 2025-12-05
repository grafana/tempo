//! Integration tests for row group filtering

use vparquet4::filter::{RowGroupFilter, RowGroupFilterTrait, RowGroupStats};
use vparquet4::reader::{ReaderConfig, VParquet4Reader, VParquet4ReaderTrait};

/// Path to the test data file (relative to workspace root)
fn test_data_path() -> std::path::PathBuf {
    let manifest_dir = std::env::var("CARGO_MANIFEST_DIR").unwrap();
    let crate_dir = std::path::PathBuf::from(manifest_dir);
    let workspace_root = crate_dir.parent().unwrap().parent().unwrap();
    workspace_root.join("tempodb/encoding/vparquet4/test-data/single-tenant/b27b0e53-66a0-4505-afd6-434ae3cd4a10/data.parquet")
}

#[test]
fn test_extract_row_group_statistics() {
    // Open the test file
    let reader = VParquet4Reader::open(test_data_path(), ReaderConfig::default())
        .expect("Failed to open test file");

    let metadata = reader.metadata();

    // Extract statistics from each row group
    for (i, rg) in metadata.row_groups().iter().enumerate() {
        let stats = RowGroupStats::from_metadata(rg).expect("Failed to extract statistics");

        println!("Row group {} statistics:", i);
        println!("  Rows: {}", stats.num_rows);

        if let (Some(min_start), Some(max_start)) = (stats.min_start_time_ns, stats.max_start_time_ns) {
            println!("  Start time range: {} - {}", min_start, max_start);
        }

        if let (Some(min_end), Some(max_end)) = (stats.min_end_time_ns, stats.max_end_time_ns) {
            println!("  End time range: {} - {}", min_end, max_end);
        }

        if let (Some(ref min_id), Some(ref max_id)) = (stats.min_trace_id, stats.max_trace_id) {
            println!("  Trace ID range: {:02x?}... - {:02x?}...",
                     &min_id[..4.min(min_id.len())],
                     &max_id[..4.min(max_id.len())]);
        }

        // Verify basic invariants
        assert!(stats.num_rows > 0, "Row group should have rows");

        if let (Some(min_start), Some(max_start)) = (stats.min_start_time_ns, stats.max_start_time_ns) {
            assert!(min_start <= max_start, "Min start time should be <= max start time");
        }

        if let (Some(min_end), Some(max_end)) = (stats.min_end_time_ns, stats.max_end_time_ns) {
            assert!(min_end <= max_end, "Min end time should be <= max end time");
        }
    }

    assert!(metadata.num_row_groups() > 0, "Should have at least one row group");
}

#[test]
fn test_time_range_filter_include_all() {
    // Open the test file
    let reader = VParquet4Reader::open(test_data_path(), ReaderConfig::default())
        .expect("Failed to open test file");

    let metadata = reader.metadata();

    // Create a filter with a very wide time range that includes everything
    // According to meta.json: 2022-07-04T11:11:09Z to 2022-07-04T11:11:35Z
    // In nanoseconds: ~1656932269000000000 to ~1656932295000000000
    let filter = RowGroupFilter::new()
        .with_time_range(0, u64::MAX);

    // All row groups should be included
    let mut included_count = 0;
    for rg in metadata.row_groups() {
        if filter.should_include(rg).expect("Filter failed") {
            included_count += 1;
        }
    }

    println!("Time range filter (all): included {}/{} row groups",
             included_count, metadata.num_row_groups());
    assert_eq!(included_count, metadata.num_row_groups(),
               "All row groups should be included with wide time range");
}

#[test]
fn test_time_range_filter_exclude_all() {
    // Open the test file
    let reader = VParquet4Reader::open(test_data_path(), ReaderConfig::default())
        .expect("Failed to open test file");

    let metadata = reader.metadata();

    // Create a filter with a time range that excludes everything
    // Use a time range far in the past
    let filter = RowGroupFilter::new()
        .with_time_range(1000, 2000);

    // No row groups should be included
    let mut included_count = 0;
    for rg in metadata.row_groups() {
        if filter.should_include(rg).expect("Filter failed") {
            included_count += 1;
        }
    }

    println!("Time range filter (none): included {}/{} row groups",
             included_count, metadata.num_row_groups());
    assert_eq!(included_count, 0,
               "No row groups should be included with past time range");
}

#[test]
fn test_time_range_filter_partial() {
    // Open the test file
    let reader = VParquet4Reader::open(test_data_path(), ReaderConfig::default())
        .expect("Failed to open test file");

    let metadata = reader.metadata();

    // First, get the actual time range from the data
    let mut global_min = u64::MAX;
    let mut global_max = u64::MIN;

    for rg in metadata.row_groups() {
        let stats = RowGroupStats::from_metadata(rg).expect("Failed to extract statistics");
        if let (Some(min), Some(max)) = (stats.min_start_time_ns, stats.max_end_time_ns) {
            global_min = global_min.min(min);
            global_max = global_max.max(max);
        }
    }

    println!("Global time range: {} - {} (span: {}ns)",
             global_min, global_max, global_max - global_min);

    // Create a filter that covers the middle portion
    let range_span = global_max - global_min;
    let filter_min = global_min + range_span / 4;
    let filter_max = global_max - range_span / 4;

    let filter = RowGroupFilter::new()
        .with_time_range(filter_min, filter_max);

    // Count included row groups
    let mut included_count = 0;
    for rg in metadata.row_groups() {
        if filter.should_include(rg).expect("Filter failed") {
            included_count += 1;
        }
    }

    println!("Time range filter (partial {}-{}): included {}/{} row groups",
             filter_min, filter_max, included_count, metadata.num_row_groups());

    // We should include at least some but possibly not all row groups
    // (depending on how the data is distributed)
    assert!(included_count > 0, "Should include at least some row groups");
}

#[test]
fn test_trace_id_prefix_filter() {
    // Open the test file
    let reader = VParquet4Reader::open(test_data_path(), ReaderConfig::default())
        .expect("Failed to open test file");

    let metadata = reader.metadata();

    // Get the first row group's min trace ID as a prefix
    let first_rg = metadata.row_groups().get(0).expect("Should have at least one row group");
    let stats = RowGroupStats::from_metadata(first_rg).expect("Failed to extract statistics");

    if let Some(ref min_trace_id) = stats.min_trace_id {
        // Use the first 4 bytes as a prefix
        let prefix = min_trace_id[..4].to_vec();
        println!("Testing with trace ID prefix: {:02x?}", prefix);

        let filter = RowGroupFilter::new()
            .with_trace_id_prefix(prefix);

        // Count included row groups
        let mut included_count = 0;
        for rg in metadata.row_groups() {
            if filter.should_include(rg).expect("Filter failed") {
                included_count += 1;
            }
        }

        println!("Trace ID prefix filter: included {}/{} row groups",
                 included_count, metadata.num_row_groups());

        // At least the first row group should be included
        assert!(included_count > 0, "Should include at least one row group with matching prefix");
    } else {
        println!("Skipping trace ID prefix test - no trace ID statistics available");
    }
}

#[test]
fn test_combined_filters() {
    // Open the test file
    let reader = VParquet4Reader::open(test_data_path(), ReaderConfig::default())
        .expect("Failed to open test file");

    let metadata = reader.metadata();

    // Get time range and trace ID from first row group
    let first_rg = metadata.row_groups().get(0).expect("Should have at least one row group");
    let stats = RowGroupStats::from_metadata(first_rg).expect("Failed to extract statistics");

    // Create a filter with both time range and trace ID prefix
    let mut filter = RowGroupFilter::new();

    if let (Some(min_start), Some(max_end)) = (stats.min_start_time_ns, stats.max_end_time_ns) {
        // Use a time range that should include the first row group
        filter = filter.with_time_range(min_start, max_end + 1000000000);
    }

    if let Some(ref min_trace_id) = stats.min_trace_id {
        // Use first 2 bytes as prefix
        let prefix = min_trace_id[..2].to_vec();
        filter = filter.with_trace_id_prefix(prefix);
    }

    // Count included row groups
    let mut included_count = 0;
    for rg in metadata.row_groups() {
        if filter.should_include(rg).expect("Filter failed") {
            included_count += 1;
        }
    }

    println!("Combined filter: included {}/{} row groups",
             included_count, metadata.num_row_groups());

    // Should include at least the first row group
    assert!(included_count > 0, "Should include at least one row group with combined filters");
}

#[test]
fn test_no_filter() {
    // Open the test file
    let reader = VParquet4Reader::open(test_data_path(), ReaderConfig::default())
        .expect("Failed to open test file");

    let metadata = reader.metadata();

    // Empty filter should include everything
    let filter = RowGroupFilter::new();

    let mut included_count = 0;
    for rg in metadata.row_groups() {
        if filter.should_include(rg).expect("Filter failed") {
            included_count += 1;
        }
    }

    println!("No filter: included {}/{} row groups",
             included_count, metadata.num_row_groups());
    assert_eq!(included_count, metadata.num_row_groups(),
               "Empty filter should include all row groups");
}
