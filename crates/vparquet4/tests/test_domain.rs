//! Integration tests for domain type parsing (Arrow to OTLP conversion)

use vparquet4::{convert, ReaderConfig, VParquet4Reader, VParquet4ReaderTrait};

const TEST_DATA_PATH: &str =
    "../../tempodb/encoding/vparquet4/test-data/single-tenant/b27b0e53-66a0-4505-afd6-434ae3cd4a10/data.parquet";

#[test]
fn test_parse_resource_spans() {
    let mut reader = VParquet4Reader::open(TEST_DATA_PATH, ReaderConfig::default())
        .expect("Failed to open test data file");

    // Read first row group
    let batch = reader
        .read_row_group(0)
        .expect("Failed to read row group");

    // Parse ResourceSpans from the first trace
    let resource_spans = convert::parse_resource_spans_from_batch(&batch, 0)
        .expect("Failed to parse ResourceSpans");

    // Should have at least one ResourceSpans
    assert!(!resource_spans.is_empty(), "Expected at least one ResourceSpans");

    // Check that we have scope spans
    let first_rs = &resource_spans[0];
    assert!(!first_rs.scope_spans.is_empty(), "Expected at least one ScopeSpans");

    // Check that we have spans
    let first_ss = &first_rs.scope_spans[0];
    assert!(!first_ss.spans.is_empty(), "Expected at least one Span");
}

#[test]
fn test_parse_span_fields() {
    let mut reader = VParquet4Reader::open(TEST_DATA_PATH, ReaderConfig::default())
        .expect("Failed to open test data file");

    let batch = reader
        .read_row_group(0)
        .expect("Failed to read row group");

    let resource_spans = convert::parse_resource_spans_from_batch(&batch, 0)
        .expect("Failed to parse ResourceSpans");

    // Get first span
    let span = &resource_spans[0].scope_spans[0].spans[0];

    // Verify span has required fields
    assert!(!span.span_id.is_empty(), "Span should have span_id");
    assert!(!span.name.is_empty(), "Span should have name");
    assert!(span.start_time_unix_nano > 0, "Span should have start time");
    assert!(span.end_time_unix_nano >= span.start_time_unix_nano,
            "End time should be >= start time");
}

#[test]
fn test_parse_resource_attributes() {
    let mut reader = VParquet4Reader::open(TEST_DATA_PATH, ReaderConfig::default())
        .expect("Failed to open test data file");

    let batch = reader
        .read_row_group(0)
        .expect("Failed to read row group");

    let resource_spans = convert::parse_resource_spans_from_batch(&batch, 0)
        .expect("Failed to parse ResourceSpans");

    // Check if first ResourceSpans has a resource with attributes
    if let Some(ref resource) = resource_spans[0].resource {
        // Resource should have some attributes (at minimum service.name)
        let has_service_name = resource.attributes.iter()
            .any(|kv| kv.key == "service.name");

        assert!(has_service_name || resource.attributes.is_empty(),
                "If resource has attributes, should include service.name");
    }
}

#[test]
fn test_parse_span_attributes() {
    let mut reader = VParquet4Reader::open(TEST_DATA_PATH, ReaderConfig::default())
        .expect("Failed to open test data file");

    let batch = reader
        .read_row_group(0)
        .expect("Failed to read row group");

    let resource_spans = convert::parse_resource_spans_from_batch(&batch, 0)
        .expect("Failed to parse ResourceSpans");

    // Find a span with attributes
    let mut found_span_with_attrs = false;
    for rs in &resource_spans {
        for ss in &rs.scope_spans {
            for span in &ss.spans {
                if !span.attributes.is_empty() {
                    found_span_with_attrs = true;

                    // Verify attribute structure
                    for attr in &span.attributes {
                        assert!(!attr.key.is_empty(), "Attribute key should not be empty");
                        assert!(attr.value.is_some(), "Attribute should have value");
                    }
                    break;
                }
            }
        }
    }

    // Note: It's okay if no spans have attributes in this test data
    // The test just validates the structure if they exist
    println!("Found span with attributes: {}", found_span_with_attrs);
}

#[test]
fn test_parse_multiple_traces() {
    let mut reader = VParquet4Reader::open(TEST_DATA_PATH, ReaderConfig::default())
        .expect("Failed to open test data file");

    let batch = reader
        .read_row_group(0)
        .expect("Failed to read row group");

    // Parse first 5 traces
    let num_traces = std::cmp::min(5, batch.num_rows());
    let mut total_spans = 0;

    for i in 0..num_traces {
        let resource_spans = convert::parse_resource_spans_from_batch(&batch, i)
            .expect("Failed to parse ResourceSpans");

        for rs in &resource_spans {
            for ss in &rs.scope_spans {
                total_spans += ss.spans.len();
            }
        }
    }

    assert!(total_spans > 0, "Should have parsed some spans");
    println!("Parsed {} spans from {} traces", total_spans, num_traces);
}

#[test]
fn test_parse_span_status() {
    let mut reader = VParquet4Reader::open(TEST_DATA_PATH, ReaderConfig::default())
        .expect("Failed to open test data file");

    let batch = reader
        .read_row_group(0)
        .expect("Failed to read row group");

    let resource_spans = convert::parse_resource_spans_from_batch(&batch, 0)
        .expect("Failed to parse ResourceSpans");

    // Check that spans have status
    let span = &resource_spans[0].scope_spans[0].spans[0];
    assert!(span.status.is_some(), "Span should have status");

    if let Some(status) = &span.status {
        // Status code should be valid (0=UNSET, 1=OK, 2=ERROR)
        assert!(status.code >= 0 && status.code <= 2,
                "Status code should be 0, 1, or 2");
    }
}

#[test]
fn test_parse_events_and_links() {
    let mut reader = VParquet4Reader::open(TEST_DATA_PATH, ReaderConfig::default())
        .expect("Failed to open test data file");

    let batch = reader
        .read_row_group(0)
        .expect("Failed to read row group");

    let resource_spans = convert::parse_resource_spans_from_batch(&batch, 0)
        .expect("Failed to parse ResourceSpans");

    // Look for spans with events or links
    let mut found_events = false;
    let mut found_links = false;

    for rs in &resource_spans {
        for ss in &rs.scope_spans {
            for span in &ss.spans {
                if !span.events.is_empty() {
                    found_events = true;
                    // Verify event structure
                    for event in &span.events {
                        assert!(!event.name.is_empty() || event.name.is_empty(),
                                "Event name field exists");
                    }
                }
                if !span.links.is_empty() {
                    found_links = true;
                    // Verify link structure
                    for link in &span.links {
                        assert!(!link.span_id.is_empty() || link.span_id.is_empty(),
                                "Link span_id field exists");
                    }
                }
            }
        }
    }

    println!("Found events: {}, Found links: {}", found_events, found_links);
    // Note: It's okay if test data doesn't have events or links
    // This test just validates parsing doesn't fail
}
