package vparquet5

import (
	"math/rand"
	"testing"

	pq "github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/require"
)

func TestSearchFetchSpansOnly(t *testing.T) {
	var (
		ctx          = t.Context()
		numTraces    = 250
		traces       = make([]*Trace, 0, numTraces)
		wantTraceIdx = rand.Intn(numTraces)
		wantTraceID  = test.ValidTraceID(nil)
		traceIDText  = util.TraceIDToHexString(wantTraceID)
	)

	for i := 0; i < numTraces; i++ {
		if i == wantTraceIdx {
			traces = append(traces, fullyPopulatedTestTrace(wantTraceID))
			continue
		}

		id := test.ValidTraceID(nil)
		tr, _ := traceToParquet(&backend.BlockMeta{}, id, test.MakeTrace(1, id), nil)
		traces = append(traces, tr)
	}

	b := makeBackendBlockWithTraces(t, traces)

	for _, tc := range searchesThatMatch(t, traceIDText) {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.req
			if req.SecondPass == nil {
				req.SecondPass = func(s *traceql.Spanset) ([]*traceql.Spanset, error) { return []*traceql.Spanset{s}, nil }
				req.SecondPassConditions = traceql.SearchMetaConditions()
			}

			resp, err := b.FetchSpans(ctx, req, common.DefaultSearchOptions())
			require.NoError(t, err, "search request:%v", req)

			found := false
			for {
				span, err := resp.Results.Next(ctx)
				require.NoError(t, err, "search request:%v", req)
				if span == nil {
					break
				}
				traceID, ok := span.AttributeFor(traceql.IntrinsicTraceIDAttribute)
				if !ok {
					continue
				}
				traceIDString := traceID.EncodeToString(false)
				// fmt.Println("got:", traceIDString, "want:", traceIDText)
				found = (traceIDString == traceIDText)
				if found {
					break
				}
			}
			require.True(t, found, "search request:%v", req)
		})
	}

	for _, tc := range searchesThatDontMatch(t) {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.req
			if req.SecondPass == nil {
				req.SecondPass = func(s *traceql.Spanset) ([]*traceql.Spanset, error) { return []*traceql.Spanset{s}, nil }
				req.SecondPassConditions = traceql.SearchMetaConditions()
			}

			resp, err := b.FetchSpans(ctx, req, common.DefaultSearchOptions())
			require.NoError(t, err, "search request:%v", req)

			for {
				span, err := resp.Results.Next(ctx)
				require.NoError(t, err, "search request:%v", req)
				if span == nil {
					break
				}
				traceID, ok := span.AttributeFor(traceql.IntrinsicTraceIDAttribute)
				if !ok {
					continue
				}
				traceIDString := traceID.EncodeToString(false)
				require.NotEqual(t, traceIDText, traceIDString, "search request:%v", req)
			}
		})
	}
}

func TestSelectAllFetchSpansOnly(t *testing.T) {
	var (
		ctx             = t.Context()
		numTraces       = 250
		traces          = make([]*Trace, 0, numTraces)
		wantTraceIdx    = rand.Intn(numTraces)
		wantTraceID     = test.ValidTraceID(nil)
		wantTraceIDText = util.TraceIDToHexString(wantTraceID)
		wantTrace       = fullyPopulatedTestTrace(wantTraceID)
		dc              = test.MakeDedicatedColumns()
		dcm             = dedicatedColumnsToColumnMapping(dc)
		opts            = common.DefaultSearchOptions()
	)

	// TODO - This strips unsupported attributes types for now. Revisit when
	// add support for arrays/kvlists in the fetch layer.
	trimForSelectAll(wantTrace)

	for i := 0; i < numTraces; i++ {
		if i == wantTraceIdx {
			traces = append(traces, wantTrace)
			continue
		}

		id := test.ValidTraceID(nil)
		tr, _ := traceToParquet(&backend.BlockMeta{}, id, test.MakeTrace(1, id), nil)
		traces = append(traces, tr)
	}

	b := makeBackendBlockWithTraces(t, traces)

	_, eval, _, _, req, err := traceql.Compile("{}")
	require.NoError(t, err)

	req.SecondPass = func(inSS *traceql.Spanset) ([]*traceql.Spanset, error) { return eval([]*traceql.Spanset{inSS}) }
	req.SecondPassSelectAll = true

	resp, err := b.FetchSpans(ctx, *req, opts)
	require.NoError(t, err)
	defer resp.Results.Close()

	// This is a dump of all spans in the fully-populated test trace
	// Since this fetch is spans only, we compare one by one.
	// Spans are returned in the same order as they were written.
	wantSS := flattenForSelectAll(wantTrace, dcm)
	found := false

	for {
		// Seek to our desired trace
		sp, err := resp.Results.Next(ctx)
		require.NoError(t, err)
		if sp == nil {
			break
		}
		tid, ok := sp.AttributeFor(traceql.IntrinsicTraceIDAttribute)
		require.True(t, ok, "trace id attribute missing")
		if tid.EncodeToString(false) != wantTraceIDText {
			continue
		}

		found = true

		// Cleanup found data for comparison
		// equal will fail on the rownum mismatches. this is an internal detail to the
		// fetch layer. just wipe them out here
		gotSp := sp.(*span)
		gotSp.cbSpanset = nil
		gotSp.cbSpansetFinal = false

		rn := gotSp.rowNum
		gotSp.rowNum = pq.RowNumber{}

		gotSp.startTimeUnixNanos = 0 // selectall doesn't imply start time
		sortAttrs(gotSp.traceAttrs)
		sortAttrs(gotSp.resourceAttrs)
		sortAttrs(gotSp.spanAttrs)
		sortAttrs(gotSp.instrumentationAttrs)

		// Pop next wanted span from the spanset.
		wantSp := wantSS.Spans[0].(*span)
		wantSS.Spans = wantSS.Spans[1:]

		require.Equal(t, wantSp, gotSp)

		// Restore row number because we are mucking with internal state.
		// This is special
		gotSp.rowNum = rn
	}
	require.True(t, found, "trace was found")
}
