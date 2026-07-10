package vparquet5

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// BenchmarkTraceIDProjection measures the actual wire bytes read while
// enumerating a real block's TraceID column, to re-derive DESIGN.md's
// "~2.8 MiB compressed per 200k IDs" reconstruction-sizing assumption
// (modules/bloomgateway/DESIGN.md § Reconstruction) from measurement
// instead of trusting it: TraceID carries tag `parquet:""` -- no dict, no
// compression codec (schema.go) -- so the design doc's figure may be
// optimistic.
//
// Skips cleanly if no real block is configured. Set VP5_BENCH_BLOCKID,
// VP5_BENCH_PATH, and optionally VP5_BENCH_TENANTID (see blockForBenchmarks
// in block_traceql_test.go) to run against a real on-disk block and get a
// number.
func BenchmarkTraceIDProjection(b *testing.B) {
	if _, ok := os.LookupEnv("VP5_BENCH_BLOCKID"); !ok {
		b.Skip("VP5_BENCH_BLOCKID not set; see blockForBenchmarks in block_traceql_test.go for how to point this benchmark at a real block")
	}
	if _, ok := os.LookupEnv("VP5_BENCH_PATH"); !ok {
		b.Skip("VP5_BENCH_PATH not set; see blockForBenchmarks in block_traceql_test.go for how to point this benchmark at a real block")
	}

	ctx := context.Background()
	block := blockForBenchmarks(b)

	b.ResetTimer()

	var (
		totalBytes uint64
		totalIDs   int
	)

	for i := 0; i < b.N; i++ {
		iter, err := block.openTraceIDReader(ctx)
		require.NoError(b, err)

		for {
			_, err := iter.Next(ctx)
			if errors.Is(err, io.EOF) {
				break
			}
			require.NoError(b, err)
			totalIDs++
		}

		totalBytes += iter.BytesRead()
		iter.Close()
	}

	b.SetBytes(int64(totalBytes) / int64(b.N))
	b.ReportMetric(float64(totalBytes)/float64(b.N), "bytes/op")
	if totalIDs > 0 {
		b.ReportMetric(float64(totalBytes)/float64(totalIDs), "bytes/trace-id")
	}
}
