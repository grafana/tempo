use criterion::{black_box, criterion_group, criterion_main, Criterion};
use std::env;
use std::path::PathBuf;
use std::time::Duration;
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
// BenchmarkBackendBlockTraceQL/spanAttIntrinsicMatch-14            	      44	  28327922 ns/op	1880.30 MB/s	        53.26 MB_io/op	         0 spans/op
fn span_att_intrinsic_match(c: &mut Criterion) {
    let file_path = get_benchmark_file_path();

    // Verify the file exists before benchmarking
    if !file_path.exists() {
        panic!("Benchmark file does not exist: {:?}", file_path);
    }

    println!("Benchmarking file: {:?}", file_path);

    let mut group = c.benchmark_group("spanAttIntrinsicMatch");
    group.measurement_time(Duration::from_secs(10));
    group.sample_size(10);

    group.bench_function("spanAttIntrinsicMatch", |b| {
        b.iter(|| {
            let options = ReadOptions {
                start_row_group: 0,
                total_row_groups: 0, // Read all row groups
                filter: Some(SpanFilter::NameEquals(
                    "distributor.ConsumeTraces".to_string(),
                )),
            };

            let reader = VParquet4Reader::open(&file_path, options)
                .expect("Failed to open parquet file");

            let mut matches = 0;
            let mut total_spans = 0;

            for result in reader {
                let spanset = result.expect("Failed to read spanset");
                let span_count = spanset.spans.len();
                total_spans += span_count;
                if span_count > 0 {
                    matches += 1;
                }
            }

            black_box((matches, total_spans))
        });
    });
}

criterion_group!(benches, span_att_intrinsic_match);
criterion_main!(benches);
