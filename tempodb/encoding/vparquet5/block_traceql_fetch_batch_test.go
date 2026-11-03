package vparquet5

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/pkg/traceql"
	"github.com/grafana/tempo/v3/pkg/util/test"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
)

// The fetch layer reuses one span per pass and keeps trace- and resource-level attributes
// across the spans of a trace or batch. These tests guard against that state leaking or
// growing from span to span: every span must carry each attribute once, and regrouping the
// same spans into larger batches must not change the results or the engine bytes.

var batchedMetricsQueries = []string{
	// Resource filter and resource by(): pass 1 resource attributes are copied into pass 2.
	`{ resource.res.generic = "r0" } | count_over_time() by (resource.dedicated.resource.1)`,
	`{ resource.dedicated.resource.1 = "d1" } | count_over_time() by (resource.res.generic)`,
	`{ resource.service.name != "" } | count_over_time() by (resource.res.generic, name)`,
	`{ resource.res.generic != "" } | quantile_over_time(duration, .9) by (resource.dedicated.resource.1)`,
	// Trace filter and resource by(): pass 1 trace attributes are copied into pass 2.
	`{ trace:rootService = "svc-0" } | count_over_time() by (resource.res.generic)`,
	// Other levels.
	`{ resource.res.generic != "" } | count_over_time() by (name)`,
	`{ span.span.generic = "s1" } | count_over_time() by (resource.res.generic)`,
	`{ status = error } | count_over_time() by (resource.dedicated.resource.1)`,
	`{ resource.res.generic != "" } | count_over_time()`,
}

func TestMetricsFetchSpansAttributesOncePerSpan(t *testing.T) {
	start := time.Now().Add(-time.Minute)

	batched := makeBackendBlockWithTraces(t, makeBatchedTraces(start, 3, 2, 50, true))
	unbatched := makeBackendBlockWithTraces(t, makeBatchedTraces(start, 3, 2, 50, false))

	for _, q := range batchedMetricsQueries {
		t.Run(q, func(t *testing.T) {
			wantSeries, wantBytes, wantSpans := runCheckedMetricsQuery(t, unbatched, q, start, false)
			require.NotZero(t, wantSpans, "query must match spans")

			gotSeries, gotBytes, gotSpans := runCheckedMetricsQuery(t, batched, q, start, false)
			require.Equal(t, wantSpans, gotSpans)
			require.Equal(t, wantSeries, gotSeries, "regrouping spans into batches changed the results")
			require.Equal(t, wantBytes, gotBytes, "regrouping spans into batches changed the engine bytes")

			legacySeries, _, _ := runCheckedMetricsQuery(t, batched, q, start, true)
			require.Equal(t, wantSeries, legacySeries, "spans-only fetch differs from the spanset fetch")
		})
	}
}

// runCheckedMetricsQuery runs a metrics query through the engine with engine bytes tracking,
// on the spanset fetch path if legacy is set and the spans-only path otherwise. It fails if the
// spans-only path yields a span with the same attribute more than once.
func runCheckedMetricsQuery(t *testing.T, b *backendBlock, query string, start time.Time, legacy bool) (map[string][]float64, int64, uint64) {
	t.Helper()

	var (
		ctx                   = context.Background()
		opts                  = common.DefaultSearchOptions()
		st                    = uint64(start.Add(-10 * time.Second).UnixNano())
		end                   = uint64(start.Add(10 * time.Second).UnixNano())
		fetches, spansFetches int
	)

	fetcher := traceql.NewSpansetFetcherWrapperBoth(
		func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			fetches++
			return b.Fetch(ctx, req, opts)
		},
		func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansOnlyResponse, error) {
			spansFetches++
			resp, err := b.FetchSpans(ctx, req, opts)
			if err != nil {
				return resp, err
			}
			resp.Results = &onceAttributesIterator{t: t, inner: resp.Results}
			return resp, nil
		},
	)

	req := &tempopb.QueryRangeRequest{
		Query:     query,
		Step:      uint64(time.Minute),
		Start:     st,
		End:       end,
		MaxSeries: 1000,
	}
	eval, err := traceql.NewEngine().CompileMetricsQueryRange(req, traceql.WithEngineBytesTracking(true), traceql.WithSpanOnlyFetch(!legacy))
	require.NoError(t, err)
	require.NoError(t, eval.Do(ctx, fetcher, st, end, int(req.MaxSeries)))
	if legacy {
		require.True(t, fetches > 0 && spansFetches == 0, "query must run on the spanset fetch path")
	} else {
		require.True(t, spansFetches > 0 && fetches == 0, "query must run on the spans-only fetch path")
	}

	series := map[string][]float64{}
	for _, s := range eval.Results() {
		series[s.Labels.String()] = s.Values
	}
	m := eval.Metrics()
	return series, m.AdditionalMetrics[tempopb.AdditionalMetricEngineBytes], m.SpansTotal
}

type onceAttributesIterator struct {
	t     *testing.T
	inner traceql.SpanIterator
}

func (i *onceAttributesIterator) Next(ctx context.Context) (traceql.Span, error) {
	s, err := i.inner.Next(ctx)
	if s == nil || err != nil {
		return s, err
	}

	seen := map[traceql.Attribute]int{}
	s.AllAttributesFunc(func(a traceql.Attribute, _ traceql.Static) {
		seen[a]++
	})
	for a, n := range seen {
		if n > 1 {
			i.t.Errorf("span has attribute %s %d times", a, n)
		}
	}
	return s, nil
}

func (i *onceAttributesIterator) Close() { i.inner.Close() }

// makeBatchedTraces returns traces of resources with spansPerResource spans each. With
// batched false, every span is written as its own batch with the same resource, so the
// two layouts hold the same spans and attributes.
func makeBatchedTraces(start time.Time, traces, resourcesPerTrace, spansPerResource int, batched bool) []*Trace {
	out := make([]*Trace, 0, traces)
	for ti := range traces {
		tr := &Trace{
			TraceID:           test.ValidTraceID(nil),
			StartTimeUnixNano: uint64(start.UnixNano()),
			EndTimeUnixNano:   uint64(start.Add(time.Second).UnixNano()),
			DurationNano:      uint64(time.Second),
			RootServiceName:   fmt.Sprintf("svc-%d", ti%2),
			RootSpanName:      "root",
		}
		for ri := range resourcesPerTrace {
			res := Resource{
				ServiceName:         fmt.Sprintf("svc-%d", (ti+ri)%2),
				Attrs:               []Attribute{attr("res.generic", fmt.Sprintf("r%d", ri))},
				DedicatedAttributes: DedicatedAttributes{String01: []string{fmt.Sprintf("d%d", (ti+ri)%2)}},
			}
			spans := make([]Span, 0, spansPerResource)
			for si := range spansPerResource {
				status := 0
				if si%5 == 0 {
					status = 2
				}
				spans = append(spans, Span{
					SpanID:            []byte(fmt.Sprintf("%02d%02d%04d", ti, ri, si)),
					Name:              fmt.Sprintf("op-%d", si%3),
					StartTimeUnixNano: uint64(start.Add(time.Duration(si) * time.Millisecond).UnixNano()),
					DurationNano:      uint64((si + 1) * int(time.Millisecond)),
					StatusCode:        status,
					Attrs:             []Attribute{attr("span.generic", fmt.Sprintf("s%d", si%2))},
				})
			}
			if batched {
				tr.ResourceSpans = append(tr.ResourceSpans, ResourceSpans{
					Resource:   res,
					ScopeSpans: []ScopeSpans{{Spans: spans, SpanCount: int32(len(spans))}},
				})
				continue
			}
			for _, s := range spans {
				tr.ResourceSpans = append(tr.ResourceSpans, ResourceSpans{
					Resource:   res,
					ScopeSpans: []ScopeSpans{{Spans: []Span{s}, SpanCount: 1}},
				})
			}
		}
		out = append(out, tr)
	}
	return out
}
