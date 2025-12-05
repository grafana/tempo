//! TraceQL-style benchmarks for vParquet4
//!
//! Replicates the Go benchmark `BenchmarkBackendBlockTraceQL` from
//! `tempodb/encoding/vparquet4/block_traceql_test.go:1448`
//!
//! ## Environment Variables
//! - `BENCH_BLOCKID`: Block UUID to benchmark against (required)
//! - `BENCH_PATH`: Path to backend storage (required)
//! - `BENCH_TENANTID`: Tenant ID (default: "single-tenant")
//!
//! ## Example Usage
//! ```bash
//! export BENCH_BLOCKID=b27b0e53-66a0-4505-afd6-434ae3cd4a10
//! export BENCH_PATH=tempodb/encoding/vparquet4/test-data
//! export BENCH_TENANTID=single-tenant
//! cargo bench --bench traceql_benchmark
//! ```

use criterion::{criterion_group, criterion_main, BenchmarkId, Criterion};
use std::env;
use std::path::PathBuf;
use vparquet4::domain::otlp::trace::v1::status::StatusCode;
use vparquet4::iter::{SpanWithContext, TraceIterator};
use vparquet4::reader::{ReaderConfig, VParquet4Reader, VParquet4ReaderTrait};

/// Load benchmark configuration from environment variables
fn load_benchmark_config() -> Result<PathBuf, String> {
    let block_id = env::var("BENCH_BLOCKID")
        .map_err(|_| "BENCH_BLOCKID not set. Set it to the block UUID to benchmark.".to_string())?;

    let bench_path = env::var("BENCH_PATH")
        .map_err(|_| "BENCH_PATH not set. Set it to the root of the backend storage.".to_string())?;

    let tenant_id = env::var("BENCH_TENANTID")
        .unwrap_or_else(|_| "single-tenant".to_string());

    let data_path = PathBuf::from(bench_path)
        .join(tenant_id)
        .join(block_id)
        .join("data.parquet");

    if !data_path.exists() {
        return Err(format!("Block data file not found at: {}", data_path.display()));
    }

    Ok(data_path)
}

/// Predicate types for TraceQL-style queries
#[derive(Debug, Clone)]
enum Predicate {
    // Span attribute predicates
    SpanAttrEqual { key: String, value: String },
    SpanAttrRegex { key: String, pattern: String },

    // Span intrinsic predicates
    SpanNameEqual(String),

    // Resource attribute predicates
    ResourceAttrEqual { key: String, value: String },

    // Trace-level predicates
    RootServiceName(String),
    StatusError,
    SpanHttpStatusCode(i64),

    // Logical operators
    And(Vec<Predicate>),
    Or(Vec<Predicate>),
}

impl Predicate {
    /// Evaluate predicate against a span with context
    fn matches(&self, span_ctx: &SpanWithContext) -> bool {
        match self {
            Predicate::SpanAttrEqual { key, value } => {
                span_ctx.span.attributes.iter().any(|attr| {
                    attr.key == *key &&
                    attr.value.as_ref().and_then(|v| v.value.as_ref()).map_or(false, |val| {
                        matches!(val,
                            vparquet4::domain::otlp::common::v1::any_value::Value::StringValue(s)
                            if s == value
                        )
                    })
                })
            }

            Predicate::SpanAttrRegex { key, pattern } => {
                span_ctx.span.attributes.iter().any(|attr| {
                    attr.key == *key &&
                    attr.value.as_ref().and_then(|v| v.value.as_ref()).map_or(false, |val| {
                        if let vparquet4::domain::otlp::common::v1::any_value::Value::StringValue(s) = val {
                            s.contains(pattern) // Simplified regex matching
                        } else {
                            false
                        }
                    })
                })
            }

            Predicate::SpanNameEqual(name) => span_ctx.span.name == *name,

            Predicate::ResourceAttrEqual { key, value } => {
                span_ctx.resource.as_ref().map_or(false, |res| {
                    res.attributes.iter().any(|attr| {
                        attr.key == *key &&
                        attr.value.as_ref().and_then(|v| v.value.as_ref()).map_or(false, |val| {
                            matches!(val,
                                vparquet4::domain::otlp::common::v1::any_value::Value::StringValue(s)
                                if s == value
                            )
                        })
                    })
                })
            }

            Predicate::RootServiceName(name) => {
                // For root service name, check if this is a root span and matches service.name
                if span_ctx.span.parent_span_id.is_empty() || span_ctx.span.parent_span_id.iter().all(|&b| b == 0) {
                    span_ctx.resource.as_ref().map_or(false, |res| {
                        res.attributes.iter().any(|attr| {
                            attr.key == "service.name" &&
                            attr.value.as_ref().and_then(|v| v.value.as_ref()).map_or(false, |val| {
                                matches!(val,
                                    vparquet4::domain::otlp::common::v1::any_value::Value::StringValue(s)
                                    if s == name
                                )
                            })
                        })
                    })
                } else {
                    false
                }
            }

            Predicate::StatusError => {
                span_ctx.span.status.as_ref().map_or(false, |status| {
                    status.code == StatusCode::Error as i32
                })
            }

            Predicate::SpanHttpStatusCode(code) => {
                span_ctx.span.attributes.iter().any(|attr| {
                    attr.key == "http.status_code" &&
                    attr.value.as_ref().and_then(|v| v.value.as_ref()).map_or(false, |val| {
                        matches!(val,
                            vparquet4::domain::otlp::common::v1::any_value::Value::IntValue(i)
                            if *i == *code
                        )
                    })
                })
            }

            Predicate::And(preds) => preds.iter().all(|p| p.matches(span_ctx)),

            Predicate::Or(preds) => preds.iter().any(|p| p.matches(span_ctx)),
        }
    }
}

/// Execute a predicate query and return (spans_matched, bytes_read)
fn execute_query(data_path: &PathBuf, predicate: &Predicate) -> Result<(usize, u64), String> {
    let reader = VParquet4Reader::open(data_path, ReaderConfig::default())
        .map_err(|e| format!("Failed to open file: {}", e))?;

    // Calculate bytes read from row groups
    let metadata = reader.metadata();
    let mut bytes_read = 0u64;
    for rg_meta in metadata.row_groups() {
        bytes_read += rg_meta.total_byte_size() as u64;
    }

    let mut spans_matched = 0usize;

    // Iterate through all traces and count matching spans
    let trace_iter = TraceIterator::new(reader)
        .map_err(|e| format!("Failed to create trace iterator: {}", e))?;

    for trace_result in trace_iter {
        let trace = trace_result.map_err(|e| format!("Failed to read trace: {}", e))?;

        // Iterate through all spans in the trace
        for resource_span in &trace.resource_spans {
            for scope_span in &resource_span.scope_spans {
                for span in &scope_span.spans {
                    let span_ctx = SpanWithContext {
                        trace_id: trace.trace_id.clone(),
                        resource: resource_span.resource.clone(),
                        scope: scope_span.scope.clone(),
                        span: span.clone(),
                    };

                    if predicate.matches(&span_ctx) {
                        spans_matched += 1;
                    }
                }
            }
        }
    }

    Ok((spans_matched, bytes_read))
}

/// Benchmark results for summary reporting
#[derive(Debug)]
struct QueryMetrics {
    name: String,
    spans_matched: usize,
    bytes_read: u64,
}

/// Benchmark suite matching Go's BenchmarkBackendBlockTraceQL
fn benchmark_traceql_queries(c: &mut Criterion) {
    let data_path = match load_benchmark_config() {
        Ok(path) => path,
        Err(e) => {
            eprintln!("Skipping TraceQL benchmarks: {}", e);
            eprintln!("To run these benchmarks, set BENCH_BLOCKID and BENCH_PATH environment variables.");
            return;
        }
    };

    // Verify the file can be opened before running benchmarks
    if let Err(e) = VParquet4Reader::open(&data_path, ReaderConfig::default()) {
        eprintln!("Failed to open benchmark data file: {}", e);
        return;
    }

    println!("\nValidating queries and gathering metrics...\n");

    let test_cases = vec![
        // Span attribute predicates
        (
            "spanAttValMatch",
            Predicate::SpanAttrEqual {
                key: "component".to_string(),
                value: "net/http".to_string(),
            },
        ),
        (
            "spanAttValMatchFew",
            Predicate::SpanAttrRegex {
                key: "component".to_string(),
                pattern: "database/sql".to_string(),
            },
        ),
        (
            "spanAttValNoMatch",
            Predicate::SpanAttrEqual {
                key: "bloom".to_string(),
                value: "does-not-exit-6c2408325a45".to_string(),
            },
        ),

        // Span intrinsic predicates
        (
            "spanAttIntrinsicMatch",
            Predicate::SpanNameEqual("distributor.ConsumeTraces".to_string()),
        ),
        (
            "spanAttIntrinsicMatchFew",
            Predicate::SpanNameEqual("grpcutils.Authenticate".to_string()),
        ),
        (
            "spanAttIntrinsicNoMatch",
            Predicate::SpanNameEqual("does-not-exit-6c2408325a45".to_string()),
        ),

        // Resource attribute predicates
        (
            "resourceAttValMatch",
            Predicate::ResourceAttrEqual {
                key: "opencensus.exporterversion".to_string(),
                value: "Jaeger-Go-2.30.0".to_string(),
            },
        ),
        (
            "resourceAttValNoMatch",
            Predicate::ResourceAttrEqual {
                key: "module.path".to_string(),
                value: "does-not-exit-6c2408325a45".to_string(),
            },
        ),
        (
            "resourceAttIntrinsicMatch",
            Predicate::ResourceAttrEqual {
                key: "service.name".to_string(),
                value: "tempo-gateway".to_string(),
            },
        ),
        (
            "resourceAttIntrinsicNoMatch",
            Predicate::ResourceAttrEqual {
                key: "service.name".to_string(),
                value: "does-not-exit-6c2408325a45".to_string(),
            },
        ),

        // Trace-level predicates (OR queries)
        (
            "traceOrMatch",
            Predicate::And(vec![
                Predicate::RootServiceName("tempo-distributor".to_string()),
                Predicate::Or(vec![
                    Predicate::StatusError,
                    Predicate::SpanHttpStatusCode(500),
                ]),
            ]),
        ),
        (
            "traceOrMatchFew",
            Predicate::And(vec![
                Predicate::RootServiceName("faro-collector".to_string()),
                Predicate::Or(vec![
                    Predicate::StatusError,
                    Predicate::SpanHttpStatusCode(500),
                ]),
            ]),
        ),
        (
            "traceOrNoMatch",
            Predicate::And(vec![
                Predicate::RootServiceName("doesntexist".to_string()),
                Predicate::Or(vec![
                    Predicate::StatusError,
                    Predicate::SpanHttpStatusCode(500),
                ]),
            ]),
        ),
    ];

    // Validate all queries and collect metrics before benchmarking
    let mut query_metrics = Vec::new();
    for (name, predicate) in &test_cases {
        println!("Running {}", name);
        let (spans_matched, bytes_read) = match execute_query(&data_path, predicate) {
            Ok(result) => result,
            Err(e) => {
                eprintln!("Failed to validate query '{}': {}", name, e);
                continue;
            }
        };

        query_metrics.push(QueryMetrics {
            name: name.to_string(),
            spans_matched,
            bytes_read,
        });
    }

    // Print summary of all query metrics
    println!("Query validation complete. Metrics:");
    println!("{:<30} {:>15} {:>15}", "Query", "Spans Matched", "MB Read");
    println!("{:-<62}", "");
    for metrics in &query_metrics {
        let mb_io = metrics.bytes_read as f64 / 1_000_000.0;
        println!(
            "{:<30} {:>15} {:>15.2}",
            metrics.name, metrics.spans_matched, mb_io
        );
    }
    println!("\nStarting benchmark iterations...\n");

    let mut group = c.benchmark_group("traceql");

    // Configure sample size for more stable results
    group.sample_size(50);

    for (i, (name, predicate)) in test_cases.iter().enumerate() {
        if i >= query_metrics.len() {
            continue;
        }

        let bytes_read = query_metrics[i].bytes_read;

        // Set throughput so Criterion reports MB/s
        group.throughput(criterion::Throughput::Bytes(bytes_read));

        group.bench_with_input(BenchmarkId::new("query", name), predicate, |b, pred| {
            b.iter_custom(|iters| {
                let mut total_duration = std::time::Duration::ZERO;

                for _ in 0..iters {
                    let start = std::time::Instant::now();
                    let _ = execute_query(&data_path, pred);
                    total_duration += start.elapsed();
                }

                total_duration
            });
        });
    }

    group.finish();
}

criterion_group!(benches, benchmark_traceql_queries);
criterion_main!(benches);
