package vparquet5

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/require"
)

// BenchmarkBackendBlockMetricsQuery runs an arbitrary metrics query_range against
// a block on local disk. It is opt-in: the query and the block location are
// supplied entirely via environment variables and the benchmark skips when they
// are not set, so it never runs as part of a normal `go test`/CI run and embeds
// no query or data in the source tree.
//
//	VP5_BENCH_PATH      backend root (block lives at <PATH>/<TENANTID>/<BLOCKID>)
//	VP5_BENCH_TENANTID  tenant id (default "1")
//	VP5_BENCH_BLOCKID   block guid
//	VP5_BENCH_QUERY     TraceQL metrics query to run
//	VP5_BENCH_STEP      step duration (default "1m")
//	VP5_BENCH_EXEMPLARS exemplars per series (default 0)
//
// Example:
//
//	VP5_BENCH_PATH=/path/to/blocks VP5_BENCH_TENANTID=1 \
//	VP5_BENCH_BLOCKID=<guid> VP5_BENCH_QUERY='{ name = "foo" } | rate()' \
//	go test ./tempodb/encoding/vparquet5/ -run '^$' -bench BenchmarkBackendBlockMetricsQuery -cpuprofile cpu.out
func BenchmarkBackendBlockMetricsQuery(b *testing.B) {
	query := os.Getenv("VP5_BENCH_QUERY")
	if query == "" || os.Getenv("VP5_BENCH_PATH") == "" || os.Getenv("VP5_BENCH_BLOCKID") == "" {
		b.Skip("set VP5_BENCH_QUERY, VP5_BENCH_PATH and VP5_BENCH_BLOCKID to run this benchmark")
	}

	var (
		e     = traceql.NewEngine()
		ctx   = context.Background()
		opts  = common.DefaultSearchOptions()
		block = blockForBenchmarks(b)
		st    = uint64(block.meta.StartTime.UnixNano())
		end   = uint64(block.meta.EndTime.UnixNano())
		f     = traceql.NewSpansetFetcherWrapperBoth(
			func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
				return block.Fetch(ctx, req, opts)
			},
			func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansOnlyResponse, error) {
				return block.FetchSpans(ctx, req, opts)
			},
		)
	)

	req := &tempopb.QueryRangeRequest{
		Query:     query,
		Step:      uint64(durationEnv("VP5_BENCH_STEP", time.Minute)),
		Start:     st,
		End:       end,
		MaxSeries: 1000,
		Exemplars: uint32(intEnv("VP5_BENCH_EXEMPLARS", 0)),
	}

	var em traceql.EvaluatorMetrics
	b.ResetTimer()
	for b.Loop() {
		eval, err := e.CompileMetricsQueryRange(req, traceql.WithUnsafeHints(true))
		require.NoError(b, err)
		require.NoError(b, eval.Do(ctx, f, st, end, int(req.MaxSeries)))
		_ = eval.Results()
		em = eval.Metrics()
	}
	// Report once after the loop: ReportMetric overwrites, so calling it per
	// iteration only adds overhead and keeps the last iteration's values anyway.
	b.ReportMetric(float64(em.Bytes)/1024.0/1024.0, "MB_io/op")
	b.ReportMetric(float64(em.SpansTotal), "spans/op")
}

// durationEnv reads a duration from name, falling back to def when unset, invalid,
// or non-positive (a negative step would overflow the uint64 request field).
func durationEnv(name string, def time.Duration) time.Duration {
	if v := os.Getenv(name); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return def
}

// intEnv reads a non-negative int from name, falling back to def when unset,
// invalid, or negative (a negative value would overflow the uint32 request field).
func intEnv(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return def
}
