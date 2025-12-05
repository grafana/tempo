//! Basic read benchmarks for vParquet4
//!
//! More comprehensive TraceQL-style benchmarks will be added in Phase 5

use criterion::{black_box, criterion_group, criterion_main, Criterion};
use vparquet4::reader::{ReaderConfig, VParquet4Reader, VParquet4ReaderTrait};

const TEST_DATA_PATH: &str =
    "tempodb/encoding/vparquet4/test-data/single-tenant/b27b0e53-66a0-4505-afd6-434ae3cd4a10/data.parquet";

fn benchmark_open_file(c: &mut Criterion) {
    c.bench_function("open_vparquet4_file", |b| {
        b.iter(|| {
            let reader = VParquet4Reader::open(
                black_box(TEST_DATA_PATH),
                ReaderConfig::default()
            );
            black_box(reader)
        });
    });
}

fn benchmark_read_first_row_group(c: &mut Criterion) {
    c.bench_function("read_first_row_group", |b| {
        b.iter(|| {
            let mut reader = VParquet4Reader::open(TEST_DATA_PATH, ReaderConfig::default())
                .expect("Failed to open file");
            let batch = reader.read_row_group(0).expect("Failed to read row group");
            black_box(batch)
        });
    });
}

fn benchmark_read_all_traces(c: &mut Criterion) {
    c.bench_function("read_all_traces", |b| {
        b.iter(|| {
            let mut reader = VParquet4Reader::open(TEST_DATA_PATH, ReaderConfig::default())
                .expect("Failed to open file");

            let mut total_rows = 0;
            for i in 0..reader.num_row_groups() {
                let batch = reader.read_row_group(i).expect("Failed to read row group");
                total_rows += batch.num_rows();
            }
            black_box(total_rows)
        });
    });
}

criterion_group!(benches, benchmark_open_file, benchmark_read_first_row_group, benchmark_read_all_traces);
criterion_main!(benches);
