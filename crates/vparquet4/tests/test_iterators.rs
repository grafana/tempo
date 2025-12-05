//! Integration tests for trace and span iterators

use vparquet4::iter::{SpanIterator, TraceIterator};
use vparquet4::{ReaderConfig, VParquet4Reader};

const TEST_DATA_PATH: &str =
    "../../tempodb/encoding/vparquet4/test-data/single-tenant/b27b0e53-66a0-4505-afd6-434ae3cd4a10/data.parquet";

#[test]
fn test_trace_iterator_basic() {
    let reader = VParquet4Reader::open(TEST_DATA_PATH, ReaderConfig::default())
        .expect("Failed to open test data file");

    let trace_iter = TraceIterator::new(reader).expect("Failed to create TraceIterator");

    let mut trace_count = 0;
    let mut total_spans = 0;

    for result in trace_iter {
        let trace = result.expect("Failed to read trace");
        trace_count += 1;
        total_spans += trace.total_spans();

        // Verify trace has required fields
        assert!(!trace.trace_id.is_empty(), "Trace should have trace_id");
        assert_eq!(trace.trace_id.len(), 16, "Trace ID should be 16 bytes");
        assert!(!trace.resource_spans.is_empty(), "Trace should have resource spans");
    }

    // According to meta.json, test file has 134 traces
    assert_eq!(
        trace_count, 134,
        "Expected 134 traces from test data"
    );
    assert!(total_spans > 0, "Should have found some spans");

    println!(
        "Processed {} traces with {} total spans",
        trace_count, total_spans
    );
}

#[test]
fn test_trace_iterator_trace_id_format() {
    let reader = VParquet4Reader::open(TEST_DATA_PATH, ReaderConfig::default())
        .expect("Failed to open test data file");

    let mut trace_iter = TraceIterator::new(reader).expect("Failed to create TraceIterator");

    if let Some(Ok(trace)) = trace_iter.next() {
        let trace_id_hex = trace.trace_id_hex();

        // Hex string should be 32 characters (16 bytes * 2)
        assert_eq!(
            trace_id_hex.len(),
            32,
            "Trace ID hex string should be 32 characters"
        );

        // Should be valid hex
        assert!(
            trace_id_hex.chars().all(|c| c.is_ascii_hexdigit()),
            "Trace ID hex should contain only hex digits"
        );

        println!("Sample trace ID: {}", trace_id_hex);
    }
}

#[test]
fn test_trace_iterator_time_range() {
    let reader = VParquet4Reader::open(TEST_DATA_PATH, ReaderConfig::default())
        .expect("Failed to open test data file");

    let trace_iter = TraceIterator::new(reader).expect("Failed to create TraceIterator");

    for result in trace_iter.take(10) {
        let trace = result.expect("Failed to read trace");

        // Verify time range is valid
        if trace.total_spans() > 0 {
            assert!(
                trace.start_time_unix_nano > 0,
                "Trace should have valid start time"
            );
            assert!(
                trace.end_time_unix_nano >= trace.start_time_unix_nano,
                "End time should be >= start time"
            );

            let duration = trace.duration_nanos();
            assert!(
                duration >= 0,
                "Duration should be non-negative"
            );
        }
    }
}

#[test]
fn test_span_iterator_basic() {
    let reader = VParquet4Reader::open(TEST_DATA_PATH, ReaderConfig::default())
        .expect("Failed to open test data file");

    let trace_iter = TraceIterator::new(reader).expect("Failed to create TraceIterator");
    let span_iter = SpanIterator::new(trace_iter);

    let mut span_count = 0;
    let mut unique_traces = std::collections::HashSet::new();

    for result in span_iter {
        let span_ctx = result.expect("Failed to read span");
        span_count += 1;

        // Verify span has required fields
        assert!(!span_ctx.trace_id.is_empty(), "Span should have trace_id");
        assert!(!span_ctx.span.span_id.is_empty(), "Span should have span_id");
        assert!(!span_ctx.span.name.is_empty(), "Span should have name");

        unique_traces.insert(span_ctx.trace_id_hex());
    }

    assert!(span_count > 0, "Should have found some spans");
    assert_eq!(
        unique_traces.len(),
        134,
        "Should have spans from 134 unique traces"
    );

    println!(
        "Processed {} spans from {} unique traces",
        span_count,
        unique_traces.len()
    );
}

#[test]
fn test_span_iterator_trace_id_preservation() {
    let reader = VParquet4Reader::open(TEST_DATA_PATH, ReaderConfig::default())
        .expect("Failed to open test data file");

    let trace_iter = TraceIterator::new(reader).expect("Failed to create TraceIterator");
    let span_iter = SpanIterator::new(trace_iter);

    // Group spans by trace ID
    let mut spans_by_trace: std::collections::HashMap<String, Vec<_>> =
        std::collections::HashMap::new();

    for result in span_iter.take(500) {
        let span_ctx = result.expect("Failed to read span");
        spans_by_trace
            .entry(span_ctx.trace_id_hex())
            .or_insert_with(Vec::new)
            .push(span_ctx);
    }

    // Verify that all spans within a trace have the same trace ID
    for (trace_id, spans) in &spans_by_trace {
        for span in spans {
            assert_eq!(&span.trace_id_hex(), trace_id);
        }
    }

    println!(
        "Verified trace ID preservation across {} traces",
        spans_by_trace.len()
    );
}

#[test]
fn test_span_iterator_with_context() {
    let reader = VParquet4Reader::open(TEST_DATA_PATH, ReaderConfig::default())
        .expect("Failed to open test data file");

    let trace_iter = TraceIterator::new(reader).expect("Failed to create TraceIterator");
    let span_iter = SpanIterator::new(trace_iter);

    let mut spans_with_service = 0;
    let mut spans_with_scope = 0;

    for result in span_iter.take(100) {
        let span_ctx = result.expect("Failed to read span");

        // Check for service name
        if span_ctx.service_name().is_some() {
            spans_with_service += 1;
        }

        // Check for scope name
        if span_ctx.scope_name().is_some() {
            spans_with_scope += 1;
        }

        // Verify span ID format
        let span_id_hex = span_ctx.span_id_hex();
        assert!(
            span_id_hex.len() > 0,
            "Span ID hex should not be empty"
        );
    }

    println!(
        "Found {} spans with service name, {} with scope name",
        spans_with_service, spans_with_scope
    );
}

#[test]
fn test_span_iterator_span_fields() {
    let reader = VParquet4Reader::open(TEST_DATA_PATH, ReaderConfig::default())
        .expect("Failed to open test data file");

    let trace_iter = TraceIterator::new(reader).expect("Failed to create TraceIterator");
    let span_iter = SpanIterator::new(trace_iter);

    for result in span_iter.take(50) {
        let span_ctx = result.expect("Failed to read span");
        let span = &span_ctx.span;

        // Verify timestamps
        assert!(span.start_time_unix_nano > 0, "Span should have start time");
        assert!(
            span.end_time_unix_nano >= span.start_time_unix_nano,
            "End time should be >= start time"
        );

        // Verify span has a status
        assert!(span.status.is_some(), "Span should have status");

        // Verify span kind is valid (0-5)
        assert!(
            span.kind >= 0 && span.kind <= 5,
            "Span kind should be valid (0-5)"
        );
    }
}

#[test]
fn test_span_iterator_parent_relationships() {
    let reader = VParquet4Reader::open(TEST_DATA_PATH, ReaderConfig::default())
        .expect("Failed to open test data file");

    let trace_iter = TraceIterator::new(reader).expect("Failed to create TraceIterator");
    let span_iter = SpanIterator::new(trace_iter);

    let mut root_spans = 0;
    let mut child_spans = 0;

    for result in span_iter.take(200) {
        let span_ctx = result.expect("Failed to read span");

        if span_ctx.span.parent_span_id.is_empty() {
            root_spans += 1;
        } else {
            child_spans += 1;
            let parent_id_hex = span_ctx.parent_span_id_hex();
            assert!(!parent_id_hex.is_empty(), "Parent span ID hex should not be empty");
        }
    }

    assert!(root_spans > 0, "Should have found some root spans");
    assert!(child_spans > 0, "Should have found some child spans");

    println!(
        "Found {} root spans and {} child spans",
        root_spans, child_spans
    );
}

#[test]
fn test_trace_iterator_exhaustion() {
    let reader = VParquet4Reader::open(TEST_DATA_PATH, ReaderConfig::default())
        .expect("Failed to open test data file");

    let mut trace_iter = TraceIterator::new(reader).expect("Failed to create TraceIterator");

    // Read all traces
    let mut count = 0;
    while let Some(result) = trace_iter.next() {
        result.expect("Failed to read trace");
        count += 1;
    }

    // Iterator should be exhausted
    assert!(trace_iter.next().is_none(), "Iterator should be exhausted");

    // Verify we read the expected number
    assert_eq!(count, 134, "Should have read 134 traces");
}

#[test]
fn test_span_iterator_exhaustion() {
    let reader = VParquet4Reader::open(TEST_DATA_PATH, ReaderConfig::default())
        .expect("Failed to open test data file");

    let trace_iter = TraceIterator::new(reader).expect("Failed to create TraceIterator");
    let mut span_iter = SpanIterator::new(trace_iter);

    // Read all spans
    let mut count = 0;
    while let Some(result) = span_iter.next() {
        result.expect("Failed to read span");
        count += 1;
    }

    // Iterator should be exhausted
    assert!(span_iter.next().is_none(), "Iterator should be exhausted");

    assert!(count > 0, "Should have read some spans");
    println!("Read {} total spans", count);
}
