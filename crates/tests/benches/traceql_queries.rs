use criterion::{criterion_group, criterion_main, BenchmarkId, Criterion};
use std::path::PathBuf;
use std::time::Duration;
use tests::{execute_query, setup_context_with_file};
use tokio::runtime::Runtime;
use traceql::traceql_to_sql_string;
use anyhow;

/// Benchmark test case
struct BenchCase {
    name: &'static str,
    traceql: &'static str,
}

/// Test cases matching the Go benchmark queries
fn get_test_cases() -> Vec<BenchCase> {
    vec![
        // Span attribute matching
        BenchCase {
            name: "spanAttValMatch",
            traceql: "{ span.component = `net/http` }",
        },
        BenchCase {
            name: "spanAttValMatchFew",
            traceql: "{ span.component =~ `database/sql` }",
        },
        BenchCase {
            name: "spanAttValNoMatch",
            traceql: "{ span.bloom = `does-not-exit-6c2408325a45` }",
        },
        BenchCase {
            name: "spanAttIntrinsicMatch",
            traceql: "{ name = `distributor.ConsumeTraces` }",
        },
        BenchCase {
            name: "spanAttIntrinsicMatchFew",
            traceql: "{ name = `grpcutils.Authenticate` }",
        },
        BenchCase {
            name: "spanAttIntrinsicNoMatch",
            traceql: "{ name = `does-not-exit-6c2408325a45` }",
        },
        // Resource attribute matching
        BenchCase {
            name: "resourceAttValMatch",
            traceql: "{ resource.opencensus.exporterversion = `Jaeger-Go-2.30.0` }",
        },
        BenchCase {
            name: "resourceAttValNoMatch",
            traceql: "{ resource.module.path = `does-not-exit-6c2408325a45` }",
        },
        BenchCase {
            name: "resourceAttIntrinsicMatch",
            traceql: "{ resource.service.name = `tempo-gateway` }",
        },
        BenchCase {
            name: "resourceAttIntrinsicNoMatch",
            traceql: "{ resource.service.name = `does-not-exit-6c2408325a45` }",
        },
        // Trace-level queries with OR
        BenchCase {
            name: "traceOrMatch",
            traceql: "{ rootServiceName = `tempo-distributor` && (status = error || span.http.status_code = 500)}",
        },
        BenchCase {
            name: "traceOrMatchFew",
            traceql: "{ rootServiceName = `faro-collector` && (status = error || span.http.status_code = 500)}",
        },
        BenchCase {
            name: "traceOrNoMatch",
            traceql: "{ rootServiceName = `doesntexist` && (status = error || span.http.status_code = 500)}",
        },
        // Mixed queries
        BenchCase {
            name: "mixedValNoMatch",
            traceql: "{ .bloom = `does-not-exit-6c2408325a45` }",
        },
        BenchCase {
            name: "mixedValMixedMatchAnd",
            traceql: "{ resource.k8s.cluster.name =~ `prod.*` && name = `gcs.ReadRange` }",
        },
        BenchCase {
            name: "mixedValMixedMatchOr",
            traceql: "{ resource.foo = `bar` || name = `gcs.ReadRange` }",
        },
        BenchCase {
            name: "mixed",
            traceql: r#"{resource.namespace!="" && resource.service.name="cortex-gateway" && duration>50ms && resource.cluster=~"prod.*"}"#,
        },
        // Complex queries
        BenchCase {
            name: "complex",
            traceql: r#"{resource.k8s.cluster.name =~ "prod.*" && resource.k8s.namespace.name = "hosted-grafana" && resource.k8s.container.name="hosted-grafana-gateway" && name = "httpclient/grafana" && span.http.status_code = 200 && duration > 20ms}"#,
        },
    ]
}

/// Construct the parquet file path from environment variables (same as Go benchmark)
/// - BENCH_BLOCKID: GUID of the block (e.g., "030c8c4f-9d47-4916-aadc-26b90b1d2bc4")
/// - BENCH_PATH: Root path to backend (e.g., "/path/to/tempo/storage")
/// - BENCH_TENANTID: Tenant ID (defaults to "1")
///
/// Returns path: <BENCH_PATH>/<BENCH_TENANTID>/<BENCH_BLOCKID>/data.parquet
fn get_bench_file_path() -> anyhow::Result<String> {
    let block_id = std::env::var("BENCH_BLOCKID").map_err(|_| {
        anyhow::anyhow!(
            "BENCH_BLOCKID is not set. These benchmarks are designed to run against a block on local disk. \
            Set BENCH_BLOCKID to the guid of the block to run benchmarks against. \
            e.g. `export BENCH_BLOCKID=030c8c4f-9d47-4916-aadc-26b90b1d2bc4`"
        )
    })?;

    let bench_path = std::env::var("BENCH_PATH").map_err(|_| {
        anyhow::anyhow!(
            "BENCH_PATH is not set. These benchmarks are designed to run against a block on local disk. \
            Set BENCH_PATH to the root of the backend such that the block to benchmark is at \
            <BENCH_PATH>/<BENCH_TENANTID>/<BENCH_BLOCKID>."
        )
    })?;

    let tenant_id = std::env::var("BENCH_TENANTID").unwrap_or_else(|_| "1".to_string());

    // Construct the path: <BENCH_PATH>/<BENCH_TENANTID>/<BENCH_BLOCKID>/data.parquet
    let file_path = PathBuf::from(bench_path)
        .join(&tenant_id)
        .join(&block_id)
        .join("data.parquet");

    Ok(file_path.to_string_lossy().to_string())
}

fn bench_traceql_queries(c: &mut Criterion) {
    let rt = Runtime::new().unwrap();

    // Setup context once
    let file_path = get_bench_file_path().unwrap_or_else(|e| {
        panic!("Failed to get bench file path: {}. \
               Make sure BENCH_BLOCKID, BENCH_PATH, and optionally BENCH_TENANTID environment variables are set.", e)
    });

    let ctx = rt.block_on(async {
        setup_context_with_file(&file_path).await.unwrap_or_else(|e| {
            panic!("Failed to setup context: {}", e)
        })
    });

    let mut group = c.benchmark_group("traceql");

    // Set measurement time to get more stable results
    group.measurement_time(Duration::from_secs(10));
    group.sample_size(10);

    // Run benchmarks for each query
    for case in get_test_cases() {
        // Convert TraceQL to SQL
        let sql = match traceql_to_sql_string(case.traceql) {
            Ok(sql) => {
                eprintln!("\n{}: {}", case.name, sql);
                sql
            },
            Err(e) => {
                eprintln!("Skipping {}: Failed to convert TraceQL to SQL: {}", case.name, e);
                continue;
            }
        };

        group.bench_with_input(
            BenchmarkId::new("query", case.name),
            &sql,
            |b, sql_query| {
                b.to_async(&rt).iter(|| async {
                    execute_query(&ctx, sql_query).await.ok()
                });
            },
        );
    }

    group.finish();
}

/// Benchmark with custom output formatting
fn bench_with_metrics(c: &mut Criterion) {
    let rt = Runtime::new().unwrap();

    // Setup context
    let file_path = get_bench_file_path().unwrap_or_else(|e| {
        panic!("Failed to get bench file path: {}. \
               Make sure BENCH_BLOCKID, BENCH_PATH, and optionally BENCH_TENANTID environment variables are set.", e)
    });

    let ctx = rt.block_on(async {
        setup_context_with_file(&file_path).await.unwrap_or_else(|e| {
            panic!("Failed to setup context: {}", e)
        })
    });

    let mut group = c.benchmark_group("traceql_detailed");
    group.measurement_time(Duration::from_secs(10));
    group.sample_size(10);

    // Run a subset of queries with detailed metrics
    let important_queries = vec![
        BenchCase {
            name: "spanAttValMatch",
            traceql: "{ span.component = `net/http` }",
        },
        BenchCase {
            name: "resourceAttIntrinsicMatch",
            traceql: "{ resource.service.name = `tempo-gateway` }",
        },
        BenchCase {
            name: "complex",
            traceql: r#"{resource.k8s.cluster.name =~ "prod.*" && resource.k8s.namespace.name = "hosted-grafana" && resource.k8s.container.name="hosted-grafana-gateway" && name = "httpclient/grafana" && span.http.status_code = 200 && duration > 20ms}"#,
        },
    ];

    for case in important_queries {
        let sql = match traceql_to_sql_string(case.traceql) {
            Ok(sql) => sql,
            Err(e) => {
                eprintln!("Skipping {}: {}", case.name, e);
                continue;
            }
        };

        // Warm-up run to get metrics
        let warmup_rows = rt.block_on(async {
            execute_query(&ctx, &sql).await.unwrap_or(0)
        });

        println!("\n{}: {} rows returned", case.name, warmup_rows);

        group.bench_with_input(
            BenchmarkId::new("query", case.name),
            &sql,
            |b, sql_query| {
                b.to_async(&rt).iter(|| async {
                    execute_query(&ctx, sql_query).await.ok()
                });
            },
        );
    }

    group.finish();
}

criterion_group!(benches, bench_traceql_queries, bench_with_metrics);
criterion_main!(benches);
