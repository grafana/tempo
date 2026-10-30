package vparquet5

import (
	"bytes"
	"context"
	"math/rand"
	"sort"
	"testing"
	"time"

	pq "github.com/grafana/tempo/v3/pkg/parquetquery"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	v1_common "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/v3/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/v3/pkg/traceql"
	"github.com/grafana/tempo/v3/pkg/util"
	"github.com/grafana/tempo/v3/pkg/util/test"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
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
			defer resp.Results.Close()

			found := false
			for {
				span, err := resp.Results.Next(ctx)
				require.NoError(t, err, "search request:%v", req)
				if span == nil {
					break
				}

				// Ensure that every attribute returned is present in the list of conditions.
				span.AllAttributesFunc(func(a traceql.Attribute, _ traceql.Static) {
					if !req.HasAttribute(a) {
						t.Errorf("attribute %v not found in conditions", a)
					}
				})

				traceID, ok := span.AttributeFor(traceql.IntrinsicTraceIDAttribute)
				if !ok {
					continue
				}
				traceIDString := traceID.EncodeToString(false)
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
			defer resp.Results.Close()

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

	_, _, eval, req, err := traceql.Compile("{}")
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

// spanTimeTestBlock has traces every 15s over an hour, each with spans up to 5m after
// the trace start, so traces straddle the edges of any window inside the hour.
func spanTimeTestBlock(t *testing.T) (*backendBlock, time.Time) {
	start := time.Now().Add(-2 * time.Hour).Truncate(time.Hour)
	offsets := []time.Duration{0, 30 * time.Second, time.Minute, 2 * time.Minute, 5 * time.Minute}
	names := []string{"a", "b", "c"}

	traces := make([]*Trace, 0, 240)
	for i := range 240 {
		traceStart := start.Add(time.Duration(i) * 15 * time.Second)
		spans := make([]*v1.Span, 0, len(offsets))
		for j, off := range offsets {
			s := &v1.Span{
				SpanId:            test.ValidTraceID(nil)[:8],
				Name:              names[(i+j)%len(names)],
				StartTimeUnixNano: uint64(traceStart.Add(off).UnixNano()),
				EndTimeUnixNano:   uint64(traceStart.Add(off + time.Duration(j+1)*time.Second).UnixNano()),
			}
			if (i+j)%4 == 0 {
				s.Status = &v1.Status{Code: v1.Status_STATUS_CODE_ERROR}
			}
			spans = append(spans, s)
		}
		id := test.ValidTraceID(nil)
		pb := &tempopb.Trace{ResourceSpans: []*v1.ResourceSpans{{
			Resource:   &v1_resource.Resource{Attributes: []*v1_common.KeyValue{{Key: "service.name", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "svc"}}}}},
			ScopeSpans: []*v1.ScopeSpans{{Spans: spans}},
		}}}
		tr, _ := traceToParquet(&backend.BlockMeta{}, id, pb, nil)
		traces = append(traces, tr)
	}
	sort.Slice(traces, func(i, j int) bool { return bytes.Compare(traces[i].TraceID, traces[j].TraceID) < 0 })
	return makeBackendBlockWithTraces(t, traces), start
}

func TestMetricsSpanTimePushdownSameResults(t *testing.T) {
	block, start := spanTimeTestBlock(t)
	opts := common.DefaultSearchOptions()
	fetcher := traceql.NewSpansetFetcherWrapperBoth(
		func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			return block.Fetch(ctx, req, opts)
		},
		func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansOnlyResponse, error) {
			return block.FetchSpans(ctx, req, opts)
		},
	)
	blockStart, blockEnd := uint64(start.UnixNano()), uint64(start.Add(66*time.Minute).UnixNano())

	queries := []string{
		`{} | count_over_time()`,
		`{ status = error } | rate()`,
		`{ name = "a" || name = "b" } | count_over_time() by (name)`,
		`{ resource.service.name = "svc" && name != "c" } | count_over_time() by (status)`,
		`{} | quantile_over_time(duration, .5, .9)`,
		`{} >> { status = error } | count_over_time()`,
	}
	windows := []struct {
		name       string
		start, end time.Time
	}{
		{"whole block", start, start.Add(66 * time.Minute)},
		{"aligned 10m", start.Add(20 * time.Minute), start.Add(30 * time.Minute)},
		{"unaligned 10m", start.Add(20*time.Minute + 7*time.Second), start.Add(30*time.Minute + 3*time.Second)},
	}

	// The span-only fetch pushes span start down; the full-trace fetch never does.
	run := func(t *testing.T, query string, st, end time.Time, spanOnly bool) traceql.SeriesSet {
		req := &tempopb.QueryRangeRequest{Query: query, Step: uint64(time.Minute), Start: uint64(st.UnixNano()), End: uint64(end.UnixNano()), MaxSeries: 1000}
		eval, err := traceql.NewEngine().CompileMetricsQueryRange(req, traceql.WithTimeOverlapCutoff(0.2), traceql.WithSpanOnlyFetch(spanOnly))
		require.NoError(t, err)
		require.NoError(t, eval.Do(t.Context(), fetcher, blockStart, blockEnd, int(req.MaxSeries)))
		return eval.Results()
	}

	for _, w := range windows {
		for _, q := range queries {
			t.Run(w.name+"/"+q, func(t *testing.T) {
				off := run(t, q, w.start, w.end, false)
				on := run(t, q, w.start, w.end, true)
				require.NotEmpty(t, off)
				require.Equal(t, len(off), len(on))
				for k, s := range off {
					require.Contains(t, on, k)
					require.Equal(t, s.Values, on[k].Values)
				}
			})
		}
	}
}

func TestFetchSpansSpanStartTimeBounds(t *testing.T) {
	block, start := spanTimeTestBlock(t)
	winStart, winEnd := start.Add(20*time.Minute), start.Add(30*time.Minute)

	for _, tc := range []struct {
		name      string
		precision time.Duration // 0 reads the nanosecond column, 60s the rounded one
		slack     time.Duration
	}{
		{"nanoseconds", 0, 0},
		{"rounded 60s", time.Minute, 2 * time.Minute},
	} {
		t.Run(tc.name, func(t *testing.T) {
			count := func(bounded bool) (n int) {
				req := traceql.FetchSpansRequest{
					Conditions:    []traceql.Condition{{Attribute: traceql.IntrinsicSpanStartTimeAttribute, Precision: tc.precision}},
					AllConditions: true,
				}
				if bounded {
					req.SpanStartTimeUnixNanos = uint64(winStart.UnixNano())
					req.SpanEndTimeUnixNanos = uint64(winEnd.UnixNano())
				}
				resp, err := block.FetchSpans(t.Context(), req, common.DefaultSearchOptions())
				require.NoError(t, err)
				defer resp.Results.Close()
				for {
					s, err := resp.Results.Next(t.Context())
					require.NoError(t, err)
					if s == nil {
						return n
					}
					n++
					if bounded {
						st := s.StartTimeUnixNanos()
						require.GreaterOrEqual(t, st, uint64(winStart.Add(-tc.slack).UnixNano()))
						require.LessOrEqual(t, st, uint64(winEnd.Add(tc.slack).UnixNano()))
					}
				}
			}
			all, bounded := count(false), count(true)
			require.Equal(t, 240*5, all)
			require.Greater(t, bounded, 0)
			require.Less(t, bounded, all/2)
		})
	}
}
