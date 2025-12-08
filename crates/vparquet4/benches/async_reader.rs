use criterion::{criterion_group, criterion_main, BenchmarkId, Criterion};
use futures::StreamExt;
use tokio::runtime::Runtime;
use vparquet4::{ReadOptions, SpanFilter, VParquet4Reader};

/// Get path to benchmark file
fn get_benchmark_file_path() -> String {
    std::env::var("VPARQUET4_BENCH_FILE").unwrap_or_else(|_| {
        "../../tempodb/encoding/vparquet4/test-data/single-tenant/b27b0e53-66a0-4505-afd6-434ae3cd4a10/data.parquet"
            .to_string()
    })
}

fn async_benchmarks(c: &mut Criterion) {
    let rt = Runtime::new().unwrap();
    let file_path = get_benchmark_file_path();

    // Check if file exists
    if !std::path::Path::new(&file_path).exists() {
        println!("Benchmark file not found at: {}", file_path);
        println!("Set VPARQUET4_BENCH_FILE environment variable to specify a different path");
        return;
    }

    let mut group = c.benchmark_group("vparquet4_async_reader");

    // Benchmark: open (metadata + dictionary loading)
    group.bench_function("open", |b| {
        b.to_async(&rt).iter(|| async {
            VParquet4Reader::open(&file_path, ReadOptions::default())
                .await
                .unwrap()
        });
    });

    // Benchmark: open + read all (no filter)
    group.bench_function("read_all", |b| {
        b.to_async(&rt).iter(|| async {
            let reader = VParquet4Reader::open(&file_path, ReadOptions::default())
                .await
                .unwrap();
            reader.read().count().await
        });
    });

    // Find a span name from the data for filter benchmarks
    let span_name = rt.block_on(async {
        let reader = VParquet4Reader::open(&file_path, ReadOptions::default())
            .await
            .ok()?;
        let mut stream = reader.read();
        while let Some(result) = stream.next().await {
            if let Ok(spanset) = result {
                if let Some(span) = spanset.spans.first() {
                    return Some(span.name.clone());
                }
            }
        }
        None
    });

    if let Some(name) = span_name {
        // Benchmark: read with filter
        group.bench_function("read_with_filter", |b| {
            let name = name.clone();
            b.to_async(&rt).iter(|| async {
                let options = ReadOptions {
                    filter: Some(SpanFilter::NameEquals(name.clone())),
                    ..Default::default()
                };
                let reader = VParquet4Reader::open(&file_path, options).await.unwrap();
                reader.read().count().await
            });
        });

        // Test parallelism scaling
        for p in [1, 2, 4, 8] {
            group.bench_with_input(BenchmarkId::new("parallelism", p), &p, |b, &p| {
                let name = name.clone();
                let file_path = file_path.clone();
                b.to_async(&rt).iter(move || {
                    let name = name.clone();
                    let file_path = file_path.clone();
                    async move {
                        let options = ReadOptions {
                            parallelism: p,
                            filter: Some(SpanFilter::NameEquals(name)),
                            ..Default::default()
                        };
                        let reader = VParquet4Reader::open(&file_path, options).await.unwrap();
                        reader.read().count().await
                    }
                });
            });
        }

        // Test batch size variations
        for batch_size in [1, 2, 4, 8] {
            group.bench_with_input(
                BenchmarkId::new("batch_size", batch_size),
                &batch_size,
                |b, &batch_size| {
                    let name = name.clone();
                    let file_path = file_path.clone();
                    b.to_async(&rt).iter(move || {
                        let name = name.clone();
                        let file_path = file_path.clone();
                        async move {
                            let options = ReadOptions {
                                batch_size,
                                filter: Some(SpanFilter::NameEquals(name)),
                                ..Default::default()
                            };
                            let reader = VParquet4Reader::open(&file_path, options).await.unwrap();
                            reader.read().count().await
                        }
                    });
                },
            );
        }
    }

    group.finish();
}

criterion_group!(benches, async_benchmarks);
criterion_main!(benches);
