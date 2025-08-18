package livestore

import (
	"context"
	"path"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestLiveStoreQueryRange(t *testing.T) {
	// init configuration
	var (
		tempDir = t.TempDir()
		tenant  = "TestLiveStoreQueryRange"
	)

	cfg := Config{}
	cfg.TimeOverlapCutoff = 0.5
	cfg.ConcurrentBlocks = 10
	cfg.CompleteBlockTimeout = 5 * time.Minute

	// Create WAL
	walCfg := &wal.Config{
		Filepath: path.Join(tempDir, "wal"),
		Version:  encoding.DefaultEncoding().Version(),
	}
	w, err := wal.New(walCfg)
	require.NoError(t, err)
	defer func() {
		// WAL doesn't have a shutdown method, just clean up the temp directory
		_ = w.Clear()
	}()

	mover, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)
	// Create instance
	instance, err := newInstance(tenant, cfg, w, mover, log.NewNopLogger())
	require.NoError(t, err)

	// Create test spans
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

	// Create traces from spans
	trace1 := &tempopb.Trace{
		ResourceSpans: []*v1.ResourceSpans{
			{
				ScopeSpans: []*v1.ScopeSpans{
					{
						Spans: []*v1.Span{sp},
					},
				},
			},
		},
	}

	trace2 := &tempopb.Trace{
		ResourceSpans: []*v1.ResourceSpans{
			{
				ScopeSpans: []*v1.ScopeSpans{
					{
						Spans: []*v1.Span{sp2},
					},
				},
			},
		},
	}

	// Marshal traces to bytes
	trace1Bytes, err := trace1.Marshal()
	require.NoError(t, err)

	trace2Bytes, err := trace2.Marshal()
	require.NoError(t, err)

	// Create trace IDs
	traceID1 := test.ValidTraceID(nil)
	traceID2 := test.ValidTraceID(nil)

	// Push traces using pushBytes
	pushReq := &tempopb.PushBytesRequest{
		Traces: []tempopb.PreallocBytes{
			{Slice: trace1Bytes},
			{Slice: trace2Bytes},
		},
		Ids: [][]byte{traceID1, traceID2},
	}

	instance.pushBytes(now, pushReq)

	// Force block creation by cutting traces and blocks
	err = instance.cutIdleTraces(true)
	require.NoError(t, err)

	blockID, err := instance.cutBlocks(true)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, blockID)

	// Complete the block
	ctx := context.Background()
	err = instance.completeBlock(ctx, blockID)
	require.NoError(t, err)

	// Wait a bit to ensure block is ready
	time.Sleep(100 * time.Millisecond)

	// Get the completed block for testing
	instance.blocksMtx.RLock()
	var block *ingester.LocalBlock
	for _, b := range instance.completeBlocks {
		block = b
		break
	}
	instance.blocksMtx.RUnlock()

	require.NotNil(t, block, "block should have been created and completed")

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

			err = instance.QueryRange(ctx, req, rawEval, jobEval)
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
