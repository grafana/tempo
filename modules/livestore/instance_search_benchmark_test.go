package livestore

import (
	"context"
	"encoding/binary"
	"flag"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/tempodb/encoding/vparquet5"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func BenchmarkLiveStoreQueryRangeSpanOnlyWALContention(b *testing.B) {
	const (
		blockConcurrency = 10
		tracesPerBlock   = 8
		spansPerTrace    = 128
	)

	instance := newBenchmarkInstance(b, blockConcurrency)

	oldMaxProcs := runtime.GOMAXPROCS(blockConcurrency)
	b.Cleanup(func() {
		runtime.GOMAXPROCS(oldMaxProcs)
	})

	ctx := context.Background()
	now := time.Now()
	spanStart := uint64(now.Add(-30 * time.Second).UnixNano())
	for block := range blockConcurrency {
		headBlock := instance.blocks.Load().headBlock
		for traceNum := range tracesPerBlock {
			traceID := make([]byte, 16)
			binary.BigEndian.PutUint64(traceID[:8], uint64(block+1))
			binary.BigEndian.PutUint64(traceID[8:], uint64(traceNum+1))

			testTrace := benchmarkTrace(traceID, spansPerTrace, spanStart, block)
			trace.SortTrace(testTrace)
			err := headBlock.AppendTrace(
				traceID,
				testTrace,
				uint32(spanStart/uint64(time.Second)),
				uint32((spanStart+uint64(spansPerTrace)*uint64(time.Millisecond))/uint64(time.Second)),
				false,
			)
			require.NoError(b, err)
		}

		_, err := instance.cutBlocks(ctx, true)
		require.NoError(b, err)
	}
	require.Len(b, instance.blocks.Load().walBlocks, blockConcurrency)

	// Keep this query uncapped so it exercises batched iteration. Bounded
	// queries retain per-span advancement to stop storage exactly at the
	// max-series cutoff.
	req := &tempopb.QueryRangeRequest{
		Query: "{} | count_over_time() by (span.service)",
		Start: uint64(now.Add(-time.Minute).UnixNano()),
		End:   uint64(now.Add(time.Minute).UnixNano()),
		Step:  uint64(10 * time.Second),
	}

	// Prime filesystem caches and verify that the fixture exercises the WAL query path.
	response, err := instance.QueryRange(ctx, req)
	require.NoError(b, err)
	require.Len(b, response.Series, 4)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		response, err = instance.QueryRange(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
		if len(response.Series) == 0 {
			b.Fatal("expected QueryRange results")
		}
	}
}

func newBenchmarkInstance(b *testing.B, queryBlockConcurrency uint) *instance {
	blockEncoding := vparquet5.Encoding{}

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", flag.NewFlagSet("", flag.ContinueOnError))
	cfg.BlockConfig.RegisterFlagsAndApplyDefaults("", flag.NewFlagSet("", flag.ContinueOnError))
	cfg.BlockConfig.Version = blockEncoding.Version()
	cfg.Metrics.TimeOverlapCutoff = 0.5
	cfg.QueryBlockConcurrency = queryBlockConcurrency
	cfg.CompleteBlockTimeout = 5 * time.Minute

	walStore, err := wal.New(&wal.Config{
		Filepath: path.Join(b.TempDir(), "wal"),
		Version:  blockEncoding.Version(),
	})
	require.NoError(b, err)
	b.Cleanup(func() {
		require.NoError(b, walStore.Clear())
	})

	limits, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.NewRegistry())
	require.NoError(b, err)
	lifecycle, err := newCompleteBlockLifecycle(cfg, noopCompleteBlockFlusher{}, log.NewNopLogger())
	require.NoError(b, err)

	instance, err := newInstance("benchmark", cfg, walStore, blockEncoding, lifecycle, limits, log.NewNopLogger())
	require.NoError(b, err)
	return instance
}

func benchmarkTrace(traceID []byte, spanCount int, start uint64, block int) *tempopb.Trace {
	spans := make([]*tracev1.Span, spanCount)
	for i := range spans {
		spanID := make([]byte, 8)
		binary.BigEndian.PutUint64(spanID, uint64(i+1))
		spanStart := start + uint64(i)*uint64(time.Millisecond)
		spans[i] = &tracev1.Span{
			TraceId:           traceID,
			SpanId:            spanID,
			Name:              "benchmark",
			StartTimeUnixNano: spanStart,
			EndTimeUnixNano:   spanStart + uint64(time.Millisecond),
			Attributes: []*commonv1.KeyValue{{
				Key: "service",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: string(rune('a' + (block+i)%4))},
				},
			}},
		}
	}

	return &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{{
			ScopeSpans: []*tracev1.ScopeSpans{{Spans: spans}},
		}},
	}
}
