package vparquet5

import (
	"context"
	"os"
	"testing"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/require"
)

func BenchmarkQueryRangeSpansOnly(b *testing.B) {
	testCases := []string{
		"{} | rate()",
	}

	// For sampler debugging
	log.Logger = kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stderr))

	e := traceql.NewEngine()
	ctx := context.TODO()
	opts := common.DefaultSearchOptions()

	block := blockForBenchmarks(b)

	f := traceql.NewSpanFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansOnlyResponse, error) {
		return block.FetchSpansOnly(ctx, req, opts)
	})

	for _, tc := range testCases {
		b.Run(tc, func(b *testing.B) {
			st := uint64(block.meta.StartTime.UnixNano())
			end := uint64(block.meta.EndTime.UnixNano())

			req := &tempopb.QueryRangeRequest{
				Query:     tc,
				Step:      uint64(time.Minute),
				Start:     st,
				End:       end,
				MaxSeries: 1000,
			}

			eval, err := e.CompileMetricsQueryRange(req, 0, 0, false)
			require.NoError(b, err)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err := eval.DoSpansOnly(ctx, f, st, end, int(req.MaxSeries))
				require.NoError(b, err)
			}

			_ = eval.Results()

			bytes, spansTotal, _ := eval.Metrics()
			b.ReportMetric(float64(bytes)/float64(b.N)/1024.0/1024.0, "MB_IO/op")
			b.ReportMetric(float64(spansTotal)/float64(b.N), "spans/op")
			b.ReportMetric(float64(spansTotal)/b.Elapsed().Seconds(), "spans/s")
		})
	}
}
