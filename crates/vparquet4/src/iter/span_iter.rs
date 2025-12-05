//! Iterator for flattened spans in vParquet4 files
//!
//! This module provides [`SpanIterator`] which flattens the nested trace structure
//! and yields individual spans with their associated context (resource, scope, trace ID).

use crate::domain::{InstrumentationScope, Resource, Span};
use crate::error::Result;
use crate::iter::trace_iter::{Trace, TraceIterator};

/// A span with its associated context
///
/// Unlike the raw [`Span`] type, this includes the trace ID and references to the
/// resource and instrumentation scope that the span belongs to.
#[derive(Debug, Clone)]
pub struct SpanWithContext {
    /// The trace ID this span belongs to
    pub trace_id: Vec<u8>,

    /// The span itself
    pub span: Span,

    /// The resource this span is associated with (if any)
    pub resource: Option<Resource>,

    /// The instrumentation scope this span is associated with (if any)
    pub scope: Option<InstrumentationScope>,
}

impl SpanWithContext {
    /// Returns the trace ID as a hex string
    pub fn trace_id_hex(&self) -> String {
        hex::encode(&self.trace_id)
    }

    /// Returns the span ID as a hex string
    pub fn span_id_hex(&self) -> String {
        hex::encode(&self.span.span_id)
    }

    /// Returns the parent span ID as a hex string (empty if no parent)
    pub fn parent_span_id_hex(&self) -> String {
        if self.span.parent_span_id.is_empty() {
            String::new()
        } else {
            hex::encode(&self.span.parent_span_id)
        }
    }

    /// Returns the service name from the resource attributes, if present
    pub fn service_name(&self) -> Option<String> {
        self.resource.as_ref().and_then(|resource| {
            resource
                .attributes
                .iter()
                .find(|kv| kv.key == "service.name")
                .and_then(|kv| kv.value.as_ref())
                .and_then(|v| {
                    if let Some(crate::domain::otlp::common::v1::any_value::Value::StringValue(s)) =
                        &v.value
                    {
                        Some(s.clone())
                    } else {
                        None
                    }
                })
        })
    }

    /// Returns the instrumentation scope name, if present
    pub fn scope_name(&self) -> Option<String> {
        self.scope
            .as_ref()
            .map(|scope| scope.name.clone())
            .filter(|name| !name.is_empty())
    }
}

/// Iterator over flattened spans in a vParquet4 file
///
/// This iterator flattens the nested trace structure (Trace -> ResourceSpans -> ScopeSpans -> Span)
/// and yields individual [`SpanWithContext`] objects that include the trace ID and context.
///
/// # Example
///
/// ```rust,no_run
/// use vparquet4::{VParquet4Reader, ReaderConfig};
/// use vparquet4::iter::{TraceIterator, SpanIterator};
///
/// # fn main() -> Result<(), Box<dyn std::error::Error>> {
/// let reader = VParquet4Reader::open("path/to/data.parquet", ReaderConfig::default())?;
/// let trace_iter = TraceIterator::new(reader)?;
/// let span_iter = SpanIterator::new(trace_iter);
///
/// for span_ctx in span_iter {
///     let span_ctx = span_ctx?;
///     println!("Trace: {}, Span: {}, Name: {}",
///         span_ctx.trace_id_hex(),
///         span_ctx.span_id_hex(),
///         span_ctx.span.name
///     );
/// }
/// # Ok(())
/// # }
/// ```
pub struct SpanIterator {
    /// Underlying trace iterator
    trace_iter: TraceIterator,

    /// Current trace being processed
    current_trace: Option<Trace>,

    /// Flattened list of spans from the current trace
    current_spans: Vec<SpanWithContext>,

    /// Current position in the spans list
    current_span_index: usize,
}

impl SpanIterator {
    /// Creates a new SpanIterator from a TraceIterator
    pub fn new(trace_iter: TraceIterator) -> Self {
        Self {
            trace_iter,
            current_trace: None,
            current_spans: Vec::new(),
            current_span_index: 0,
        }
    }

    /// Loads the next trace and flattens its spans
    fn load_next_trace(&mut self) -> Result<bool> {
        match self.trace_iter.next() {
            Some(Ok(trace)) => {
                self.current_trace = Some(trace.clone());
                self.current_spans = Self::flatten_trace_spans(&trace);
                self.current_span_index = 0;
                Ok(true)
            }
            Some(Err(e)) => Err(e),
            None => Ok(false),
        }
    }

    /// Flattens all spans from a trace into a vector with context
    fn flatten_trace_spans(trace: &Trace) -> Vec<SpanWithContext> {
        let mut spans = Vec::new();

        for resource_spans in &trace.resource_spans {
            let resource = resource_spans.resource.clone();

            for scope_spans in &resource_spans.scope_spans {
                let scope = scope_spans.scope.clone();

                for span in &scope_spans.spans {
                    spans.push(SpanWithContext {
                        trace_id: trace.trace_id.clone(),
                        span: span.clone(),
                        resource: resource.clone(),
                        scope: scope.clone(),
                    });
                }
            }
        }

        spans
    }
}

impl Iterator for SpanIterator {
    type Item = Result<SpanWithContext>;

    fn next(&mut self) -> Option<Self::Item> {
        loop {
            // If we have spans from the current trace, yield them
            if self.current_span_index < self.current_spans.len() {
                let span = self.current_spans[self.current_span_index].clone();
                self.current_span_index += 1;
                return Some(Ok(span));
            }

            // No more spans in current trace, load next trace
            match self.load_next_trace() {
                Ok(true) => continue,  // Successfully loaded a trace, continue loop
                Ok(false) => return None, // No more traces
                Err(e) => return Some(Err(e)),
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::domain::{
        otlp::common::v1::{any_value, AnyValue, KeyValue},
        ResourceSpans, ScopeSpans, Span,
    };

    fn create_test_trace() -> Trace {
        let resource = Some(crate::domain::Resource {
            attributes: vec![KeyValue {
                key: "service.name".to_string(),
                value: Some(AnyValue {
                    value: Some(any_value::Value::StringValue("test-service".to_string())),
                }),
            }],
            dropped_attributes_count: 0,
        });

        let scope = Some(InstrumentationScope {
            name: "test-scope".to_string(),
            version: "1.0".to_string(),
            attributes: vec![],
            dropped_attributes_count: 0,
        });

        let spans = vec![
            Span {
                span_id: vec![1, 2, 3],
                name: "span1".to_string(),
                start_time_unix_nano: 1000,
                end_time_unix_nano: 2000,
                ..Default::default()
            },
            Span {
                span_id: vec![4, 5, 6],
                name: "span2".to_string(),
                start_time_unix_nano: 1500,
                end_time_unix_nano: 3000,
                ..Default::default()
            },
        ];

        Trace {
            trace_id: vec![0xAA, 0xBB, 0xCC],
            resource_spans: vec![ResourceSpans {
                resource: resource.clone(),
                scope_spans: vec![ScopeSpans {
                    scope,
                    spans,
                    schema_url: String::new(),
                }],
                schema_url: String::new(),
            }],
            start_time_unix_nano: 1000,
            end_time_unix_nano: 3000,
        }
    }

    #[test]
    fn test_flatten_trace_spans() {
        let trace = create_test_trace();
        let spans = SpanIterator::flatten_trace_spans(&trace);

        assert_eq!(spans.len(), 2);
        assert_eq!(spans[0].span.name, "span1");
        assert_eq!(spans[1].span.name, "span2");
    }

    #[test]
    fn test_span_with_context_trace_id() {
        let trace = create_test_trace();
        let spans = SpanIterator::flatten_trace_spans(&trace);

        assert_eq!(spans[0].trace_id_hex(), "aabbcc");
    }

    #[test]
    fn test_span_with_context_service_name() {
        let trace = create_test_trace();
        let spans = SpanIterator::flatten_trace_spans(&trace);

        assert_eq!(
            spans[0].service_name(),
            Some("test-service".to_string())
        );
    }

    #[test]
    fn test_span_with_context_scope_name() {
        let trace = create_test_trace();
        let spans = SpanIterator::flatten_trace_spans(&trace);

        assert_eq!(spans[0].scope_name(), Some("test-scope".to_string()));
    }

    #[test]
    fn test_span_id_hex() {
        let trace = create_test_trace();
        let spans = SpanIterator::flatten_trace_spans(&trace);

        assert_eq!(spans[0].span_id_hex(), "010203");
        assert_eq!(spans[1].span_id_hex(), "040506");
    }
}
