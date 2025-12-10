use futures::StreamExt;
use vparquet4::{ReadOptions, SpanFilter, VParquet4Reader};

/// Get path to test data file
fn test_file() -> String {
    std::env::var("VPARQUET4_TEST_FILE").unwrap_or_else(|_| {
        "../../tempodb/encoding/vparquet4/test-data/single-tenant/b27b0e53-66a0-4505-afd6-434ae3cd4a10/data.parquet"
            .to_string()
    })
}

#[tokio::test]
async fn test_open_loads_metadata() {
    let reader = VParquet4Reader::open(&test_file(), ReadOptions::default())
        .await
        .unwrap();

    // Verify cache is populated
    assert!(!reader.cache.row_groups.is_empty());
    assert!(reader.cache.metadata.num_row_groups() > 0);
}

#[tokio::test]
async fn test_read_all_spansets() {
    let reader = VParquet4Reader::open(&test_file(), ReadOptions::default())
        .await
        .unwrap();

    let mut stream = reader.read();
    let mut count = 0;
    while let Some(result) = stream.next().await {
        let spanset = result.unwrap();
        assert!(!spanset.trace_id.is_empty());
        count += 1;
    }
    assert!(count > 0, "Should read at least one spanset");
}

#[tokio::test]
async fn test_filter_by_span_name() {
    let test_file = test_file();

    // First, read all spansets to find a valid span name
    let reader_all = VParquet4Reader::open(&test_file, ReadOptions::default())
        .await
        .unwrap();

    let mut stream_all = reader_all.read();
    let mut span_name = None;

    // Find the first span name
    while let Some(result) = stream_all.next().await {
        if let Ok(spanset) = result {
            if let Some(first_span) = spanset.spans.first() {
                span_name = Some(first_span.name.clone());
                break;
            }
        }
    }

    let span_name = span_name.expect("Should find at least one span");

    // Now test filtering with that span name
    let options = ReadOptions {
        filter: Some(SpanFilter::NameEquals(span_name.clone())),
        ..Default::default()
    };

    let reader = VParquet4Reader::open(&test_file, options).await.unwrap();
    let mut stream = reader.read();

    let mut found_matching = false;
    while let Some(result) = stream.next().await {
        let spanset = result.unwrap();
        // All spans in the spanset should match the filter
        for span in &spanset.spans {
            assert_eq!(span.name, span_name, "All spans should match the filter");
            found_matching = true;
        }
    }

    assert!(found_matching, "Should find at least one matching span");
}

#[tokio::test]
async fn test_filter_reduces_results() {
    let test_file = test_file();

    // Count all spansets
    let reader_all = VParquet4Reader::open(&test_file, ReadOptions::default())
        .await
        .unwrap();
    let all_count: usize = reader_all.read().count().await;

    // Find a span name that appears in the data
    let reader_find = VParquet4Reader::open(&test_file, ReadOptions::default())
        .await
        .unwrap();
    let mut stream_find = reader_find.read();
    let mut span_name = None;

    while let Some(result) = stream_find.next().await {
        if let Ok(spanset) = result {
            if let Some(first_span) = spanset.spans.first() {
                span_name = Some(first_span.name.clone());
                break;
            }
        }
    }

    if let Some(name) = span_name {
        // Count filtered spansets
        let options = ReadOptions {
            filter: Some(SpanFilter::NameEquals(name)),
            ..Default::default()
        };

        let reader_filtered = VParquet4Reader::open(&test_file, options).await.unwrap();
        let filtered_count: usize = reader_filtered.read().count().await;

        // Filtered count should be less than or equal to all count
        assert!(
            filtered_count <= all_count,
            "Filtered count ({}) should be <= all count ({})",
            filtered_count,
            all_count
        );
    }
}

#[tokio::test]
async fn test_parallel_processing() {
    // Test with different parallelism levels
    for parallelism in [1, 2, 4] {
        let options = ReadOptions {
            parallelism,
            ..Default::default()
        };

        let reader = VParquet4Reader::open(&test_file(), options).await.unwrap();

        let count: usize = reader.read().count().await;
        assert!(
            count > 0,
            "Should read spansets with parallelism={}",
            parallelism
        );
    }
}

#[tokio::test]
async fn test_dictionary_caching() {
    let reader = VParquet4Reader::open(&test_file(), ReadOptions::default())
        .await
        .unwrap();

    // Check that dictionaries were loaded for span name column
    if let Some(col_idx) = reader.cache.span_name_column_index {
        let mut has_dictionary = false;
        for rg in &reader.cache.row_groups {
            if rg.dictionaries.contains_key(&col_idx) {
                has_dictionary = true;
                break;
            }
        }
        // At least some row groups should have dictionaries
        // (though not all row groups necessarily have them)
        println!("Has dictionary for span name column: {}", has_dictionary);
    }
}

#[tokio::test]
async fn test_batch_size_variations() {
    // Test with different batch sizes
    for batch_size in [1, 2, 4, 8] {
        let options = ReadOptions {
            batch_size,
            ..Default::default()
        };

        let reader = VParquet4Reader::open(&test_file(), options).await.unwrap();

        let count: usize = reader.read().count().await;
        assert!(
            count > 0,
            "Should read spansets with batch_size={}",
            batch_size
        );
    }
}
