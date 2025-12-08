use criterion::{black_box, criterion_group, criterion_main, Criterion};
use futures::StreamExt;
use std::env;
use std::path::PathBuf;
use std::time::Duration;
use tokio::runtime::Runtime;
use vparquet4::{ReadOptions, SpanFilter, VParquet4Reader};

fn get_benchmark_file_path() -> PathBuf {
    let block_id = env::var("BENCH_BLOCKID").expect(
        "BENCH_BLOCKID is not set. Set BENCH_BLOCKID to the guid of the block to benchmark. \
         e.g. `export BENCH_BLOCKID=030c8c4f-9d47-4916-aadc-26b90b1d2bc4`",
    );

    let tenant_id = env::var("BENCH_TENANTID").unwrap_or_else(|_| "1".to_string());

    let bench_path = env::var("BENCH_PATH").expect(
        "BENCH_PATH is not set. Set BENCH_PATH to the root of the backend such that the block \
         to benchmark is at <BENCH_PATH>/<BENCH_TENANTID>/<BENCH_BLOCKID>.",
    );

    let mut path = PathBuf::from(bench_path);
    path.push(tenant_id);
    path.push(block_id);
    path.push("data.parquet");

    path
}

// Target to beat
// tempodb/encoding/vparquet4/block_traceql_test.go:L1448
// BenchmarkBackendBlockTraceQL/spanAttIntrinsicMatch-14            	      44	  28327922 ns/op	1880.30 MB/s	        53.26 MB_io/op	         0 spans/op
fn span_att_intrinsic_match(c: &mut Criterion) {
    let file_path = get_benchmark_file_path();

    // Verify the file exists before benchmarking
    if !file_path.exists() {
        panic!("Benchmark file does not exist: {:?}", file_path);
    }

    println!("Benchmarking file: {:?}", file_path);

    let rt = Runtime::new().unwrap();
    let mut group = c.benchmark_group("spanAttIntrinsicMatch");
    group.measurement_time(Duration::from_secs(10));
    group.sample_size(10);

    let target_name = "distributor.ConsumeTraces";

    // VParquet4Reader does dictionary-based filtering internally
    // during open() and read()
    let options = ReadOptions {
        filter: Some(SpanFilter::NameEquals(target_name.to_string())),
        batch_size: 4,
        parallelism: num_cpus::get(),
    };

    // Open reader ONCE, outside the benchmark loop
    let vp_reader = rt
        .block_on(async { VParquet4Reader::open(&file_path, options).await.unwrap() });

    // Dictionary-based filtering using VParquet4Reader
    group.bench_function("spanAttIntrinsicMatch", |b| {
        b.to_async(&rt).iter(|| async {
            // Only measure the read() operation
            let mut stream = vp_reader.read();

            let mut span_count = 0;
            while let Some(spanset_result) = stream.next().await {
                let spanset = spanset_result.unwrap();
                span_count += spanset.spans.len();
            }

            black_box(span_count)
        });
    });

    group.finish();
}

// Target: TraceQL '{ name = `grpcutils.Authenticate` }'
// Expected: 406329 spans, <28ms runtime
fn span_att_intrinsic_match_few(c: &mut Criterion) {
    let file_path = get_benchmark_file_path();

    if !file_path.exists() {
        panic!("Benchmark file does not exist: {:?}", file_path);
    }

    println!("Benchmarking file: {:?}", file_path);

    let rt = Runtime::new().unwrap();
    let mut group = c.benchmark_group("spanAttIntrinsicMatchFew");
    group.measurement_time(Duration::from_secs(10));
    group.sample_size(10);

    let target_name = "grpcutils.Authenticate";

    // The new async reader now does dictionary-based filtering internally
    // during open() and read(), so we just need to use it directly
    let options = ReadOptions {
        filter: Some(SpanFilter::NameEquals(target_name.to_string())),
        batch_size: 4,
        parallelism: num_cpus::get(),
    };

    // Open reader ONCE, outside the benchmark loop
    let vp_reader = rt
        .block_on(async { VParquet4Reader::open(&file_path, options).await.unwrap() });

    group.bench_function("spanAttIntrinsicMatchFew", |b| {
        b.to_async(&rt).iter(|| async {
            // Only measure the read() operation
            let mut stream = vp_reader.read();

            let mut span_count = 0;
            while let Some(spanset_result) = stream.next().await {
                let spanset = spanset_result.unwrap();
                span_count += spanset.spans.len();
            }

            // Assert we got the expected number of spans
            assert_eq!(span_count, 406329, "Expected 406329 spans, got {}", span_count);

            black_box(span_count)
        });
    });

    group.finish();
}

criterion_group!(benches, span_att_intrinsic_match, span_att_intrinsic_match_few);
criterion_main!(benches);
