package localblocks

import (
	"flag"
	"path"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/encoding/vparquet4"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/stretchr/testify/require"
)

func (m *mockOverrides) BlockRetentionForTenant(_ string) time.Duration     { return 0 }
func (m *mockOverrides) CompactionDisabledForTenant(_ string) bool          { return false }
func (m *mockOverrides) MaxBytesPerTraceForTenant(_ string) int             { return 0 }
func (m *mockOverrides) MaxCompactionRangeForTenant(_ string) time.Duration { return 0 }

func TestProcessor(t *testing.T) {
	// init configuration
	var (
		tempDir      = t.TempDir()
		blockVersion = vparquet4.VersionString
	)

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.FlushCheckPeriod = 10 * time.Millisecond
	cfg.TraceIdlePeriod = 10 * time.Millisecond
	cfg.MaxBlockDuration = 10 * time.Millisecond
	cfg.FlushToStorage = true

	tenant := "TestProcessor"

	_, w, _, err := tempodb.New(&tempodb.Config{
		Backend: backend.Local,
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &common.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              .01,
			BloomShardSizeBytes:  100_000,
			Version:              blockVersion,
			IndexPageSizeBytes:   1000,
			RowGroupSizeBytes:    10000,
		},
		WAL: &wal.Config{
			Filepath:       path.Join(tempDir, "wal"),
			IngestionSlack: time.Since(time.Time{}),
		},
		Search: &tempodb.SearchConfig{
			ChunkSizeBytes:  1_000_000,
			ReadBufferCount: 8, ReadBufferSizeBytes: 4 * 1024 * 1024,
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	// init components
	wal := w.WAL()

	processor, err := New(cfg, tenant, wal, w, &mockOverrides{})
	require.NoError(t, err)
	defer processor.Shutdown(t.Context())

	// push spanes
	now := time.Now()

	duration := time.Millisecond
	span1Start := now.Add(-10 * time.Second)
	span2Start := now.Add(-time.Second + time.Millisecond)

	sp := test.MakeSpan(test.ValidTraceID(nil))
	sp.StartTimeUnixNano = uint64(span1Start.UnixNano())
	sp.EndTimeUnixNano = uint64(span1Start.Add(duration).UnixNano())
	sp.Kind = v1.Span_SPAN_KIND_SERVER

	sp2 := test.MakeSpan(test.ValidTraceID(nil))
	sp2.StartTimeUnixNano = uint64(span2Start.UnixNano())
	sp2.EndTimeUnixNano = uint64(span2Start.Add(duration).UnixNano())
	sp2.Kind = v1.Span_SPAN_KIND_SERVER

	processor.PushSpans(t.Context(), &tempopb.PushSpansRequest{
		Batches: []*v1.ResourceSpans{
			{
				ScopeSpans: []*v1.ScopeSpans{
					{
						Spans: []*v1.Span{
							sp,
						},
					},
				},
			},
			{
				ScopeSpans: []*v1.ScopeSpans{
					{
						Spans: []*v1.Span{
							sp2,
						},
					},
				},
			},
		},
	})

	// wait for flush and check if block was created
	time.Sleep(500 * time.Millisecond)
	var block common.BackendBlock
	func() {
		processor.blocksMtx.Lock() // to avoid race condition with running processor
		defer processor.blocksMtx.Unlock()
		require.Equal(t, len(processor.completeBlocks), 1, "block was not flushed or flushed in multiple blocks")
		for _, b := range processor.completeBlocks {
			block = b
			break
		}
	}()

	type testCase struct {
		name              string
		req               *tempopb.QueryRangeRequest
		expectedSpans     int
		expectedExemplars int
	}

	for _, tc := range []testCase{
		{
			// -------------- SP1 ------- SP2 ---------
			// ---------^------------^-----------------
			// ------- START ------ END ---------------
			name: "first trace",
			req: &tempopb.QueryRangeRequest{
				Query:     "{} | count_over_time()",
				Start:     uint64(span1Start.Add(-1 * time.Second).UnixNano()),
				End:       uint64(span1Start.Add(duration + time.Second).UnixNano()),
				Step:      uint64(time.Second),
				Exemplars: 2,
			},
			expectedSpans:     1,
			expectedExemplars: 1,
		},
		{
			// -------------- SP1 ------- SP2 ---------
			// ----------------------^------------^----
			// ------------------- START ------- END --
			name: "second trace",
			req: &tempopb.QueryRangeRequest{
				Query:     "{} | count_over_time()",
				Start:     uint64(span2Start.Add(-1 * time.Second).UnixNano()),
				End:       uint64(now.UnixNano()),
				Step:      uint64(time.Second),
				Exemplars: 2,
			},
			expectedSpans:     1,
			expectedExemplars: 1,
		},
		{
			// -------------- SP1 ------- SP2 -----------
			// ---------------^-------------------^------
			// ------------- START ------------- END ----
			name: "start of block included",
			req: &tempopb.QueryRangeRequest{
				Query:     "{} | count_over_time()",
				Start:     uint64(block.BlockMeta().StartTime.UnixNano()),
				End:       uint64(now.UnixNano()),
				Step:      uint64(time.Second),
				Exemplars: 2,
			},
			expectedSpans:     2,
			expectedExemplars: 2,
		},
		{
			// -------------- SP1 ------- SP2 ------------------
			// ----------------------------^-------------^------
			// ------------------------- START -------- END ----
			name: "end of block included",
			req: &tempopb.QueryRangeRequest{
				Query:     "{} | count_over_time()",
				Start:     uint64(block.BlockMeta().EndTime.UnixNano()),
				End:       uint64(now.UnixNano()),
				Step:      uint64(time.Second),
				Exemplars: 2,
			},
			expectedSpans:     0, // TODO: possible bug
			expectedExemplars: 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.req
			req.Start, req.End, req.Step = traceql.TrimToBlockOverlap(req.Start, req.End, req.Step, block.BlockMeta().StartTime, block.BlockMeta().EndTime)

			e := traceql.NewEngine()
			rawEval, err := e.CompileMetricsQueryRange(req, int(req.Exemplars), 0, false)
			require.NoError(t, err)
			jobEval, err := traceql.NewEngine().CompileMetricsQueryRangeNonRaw(req, traceql.AggregateModeSum)
			require.NoError(t, err)

			err = processor.QueryRange(t.Context(), req, rawEval, jobEval)
			require.NoError(t, err)
			results := jobEval.Results()

			require.Equal(t, 1, len(results))
			for _, ts := range results {
				var sum float64
				for _, val := range ts.Values {
					sum += val
				}
				require.InDelta(t, tc.expectedSpans, sum, 0.000001)
				require.Equal(t, tc.expectedExemplars, len(ts.Exemplars))
			}
		})
	}
}
