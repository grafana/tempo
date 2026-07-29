package vparquet5

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// TestExtrapolationFilterEndToEnd reproduces the docker-compose symptom via
// the real CompileMetricsQueryRange + Do flow. Two services: shop-backend
// (tracestate ot=th:8 -> multiplier 2) and tempo-vulture (no tracestate).
// `{resource.service.name="shop-backend"} | count_over_time() with(extrapolate=true)`
// must return exactly 2x the count of shop-backend spans only — not all spans.
func TestExtrapolationFilterEndToEnd(t *testing.T) {
	startTime := time.Now().Add(-1 * time.Minute)
	makeTrace := func(svc, traceState string, n int) *Trace {
		spans := make([]Span, n)
		for i := range spans {
			spans[i] = Span{
				SpanID:            []byte(svc + "/span00000000")[:12],
				Name:              svc,
				StartTimeUnixNano: uint64(startTime.UnixNano()) + uint64(i),
				DurationNano:      uint64(time.Second),
				TraceState:        traceState,
				Attrs:             []Attribute{attr("foo", "y")},
			}
		}
		return &Trace{
			TraceID: test.ValidTraceID(nil),
			ResourceSpans: []ResourceSpans{{
				Resource: Resource{
					ServiceName: svc,
					Attrs:       []Attribute{attr("foo", "x")},
				},
				ScopeSpans: []ScopeSpans{{
					SpanCount: int32(n),
					Spans:     spans,
				}},
			}},
		}
	}

	shop := makeTrace("shop-backend", "ot=th:8", 5)
	vulture := makeTrace("tempo-vulture", "", 10)

	block := makeBackendBlockWithTraces(t, []*Trace{shop, vulture})

	opts := common.DefaultSearchOptions()
	fetcher := traceql.NewSpansetFetcherWrapperBoth(
		func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			return block.Fetch(ctx, req, opts)
		},
		func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansOnlyResponse, error) {
			return block.FetchSpans(ctx, req, opts)
		},
	)

	ctx := context.Background()
	st := uint64(startTime.Add(-10 * time.Second).UnixNano())
	end := uint64(startTime.Add(10 * time.Second).UnixNano())

	cases := []struct {
		query string
		want  float64
		descr string
	}{
		{`{resource.service.name="shop-backend"} | count_over_time()`, 5, "naive shop"},
		{`{resource.service.name="shop-backend"} | count_over_time() with(extrapolate=true)`, 10, "extrap shop = 2x"},
		{`{resource.service.name="tempo-vulture"} | count_over_time()`, 10, "naive vulture"},
		{`{resource.service.name="tempo-vulture"} | count_over_time() with(extrapolate=true)`, 10, "extrap vulture = 1x (no tracestate)"},
		{`{} | count_over_time()`, 15, "naive all"},
		{`{} | count_over_time() with(extrapolate=true)`, 20, "extrap all = 5*2 + 10*1"},
	}

	for _, tc := range cases {
		t.Run(tc.descr, func(t *testing.T) {
			req := &tempopb.QueryRangeRequest{
				Query:     tc.query,
				Step:      uint64(time.Minute),
				Start:     st,
				End:       end,
				MaxSeries: 1000,
			}
			eng := traceql.NewEngine()
			eval, err := eng.CompileMetricsQueryRange(req)
			require.NoError(t, err)
			err = eval.Do(ctx, fetcher, st, end, int(req.MaxSeries))
			require.NoError(t, err)

			ss := eval.Results()
			var total float64
			for _, s := range ss {
				for _, v := range s.Values {
					total += v
				}
			}
			t.Logf("query=%q total=%v want=%v", tc.query, total, tc.want)
			require.Equal(t, tc.want, total, tc.descr)
		})
	}
}

// TestExtrapolationFilterInteraction reproduces the docker-compose symptom:
// `{resource.service.name="shop-backend"} | rate() with(extrapolate=true)`
// returns all spans instead of only shop-backend spans.
func TestExtrapolationFilterInteraction(t *testing.T) {
	shopBackendID := test.ValidTraceID(nil)
	otherID := test.ValidTraceID(nil)

	shopBackend := &Trace{
		TraceID: shopBackendID,
		ResourceSpans: []ResourceSpans{{
			Resource: Resource{
				ServiceName: "shop-backend",
				Attrs:       []Attribute{attr("foo", "x")},
			},
			ScopeSpans: []ScopeSpans{{
				SpanCount: 2,
				Spans: []Span{
					{
						SpanID:            []byte("shopspan001"),
						Name:              "shop-a",
						StartTimeUnixNano: uint64(time.Now().UnixNano()),
						DurationNano:      uint64(time.Second),
						TraceState:        "ot=th:8",
						Attrs:             []Attribute{attr("foo", "y")},
					},
					{
						SpanID:            []byte("shopspan002"),
						Name:              "shop-b",
						StartTimeUnixNano: uint64(time.Now().UnixNano()),
						DurationNano:      uint64(time.Second),
						TraceState:        "ot=th:8",
						Attrs:             []Attribute{attr("foo", "y")},
					},
				},
			}},
		}},
	}

	other := &Trace{
		TraceID: otherID,
		ResourceSpans: []ResourceSpans{{
			Resource: Resource{
				ServiceName: "other-service",
				Attrs:       []Attribute{attr("foo", "x")},
			},
			ScopeSpans: []ScopeSpans{{
				SpanCount: 3,
				Spans: []Span{
					{
						SpanID:            []byte("other000001"),
						Name:              "other-a",
						StartTimeUnixNano: uint64(time.Now().UnixNano()),
						DurationNano:      uint64(time.Second),
						Attrs:             []Attribute{attr("foo", "y")},
					},
					{
						SpanID:            []byte("other000002"),
						Name:              "other-b",
						StartTimeUnixNano: uint64(time.Now().UnixNano()),
						DurationNano:      uint64(time.Second),
						Attrs:             []Attribute{attr("foo", "y")},
					},
					{
						SpanID:            []byte("other000003"),
						Name:              "other-c",
						StartTimeUnixNano: uint64(time.Now().UnixNano()),
						DurationNano:      uint64(time.Second),
						Attrs:             []Attribute{attr("foo", "y")},
					},
				},
			}},
		}},
	}

	block := makeBackendBlockWithTraces(t, []*Trace{shopBackend, other})
	ctx := context.Background()

	// Build the FetchSpansRequest the engine would build for the query
	// `{resource.service.name="shop-backend"} | rate() with(extrapolate=true)`
	// after extractConditions + applyExtrapolation + optimize().
	req := traceql.FetchSpansRequest{
		AllConditions: true,
		Conditions: []traceql.Condition{
			{
				Attribute: traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, "service.name"),
				Op:        traceql.OpEqual,
				Operands:  []traceql.Static{traceql.NewStaticString("shop-backend")},
			},
			{Attribute: traceql.IntrinsicSpanMultiplierAttribute},
			{Attribute: traceql.IntrinsicSpanStartTimeAttribute},
		},
	}

	resp, err := block.FetchSpans(ctx, req, common.DefaultSearchOptions())
	require.NoError(t, err)
	defer resp.Results.Close()

	var (
		matched     []*span
		multipliers []float64
	)
	for {
		s, err := resp.Results.Next(ctx)
		require.NoError(t, err)
		if s == nil {
			break
		}
		sp := s.(*span)
		// Clone the relevant fields because the iterator reuses the buffer.
		clone := *sp
		matched = append(matched, &clone)

		v, ok := sp.AttributeFor(traceql.IntrinsicSpanMultiplierAttribute)
		require.True(t, ok, "multiplier intrinsic should be surfaced when requested")
		multipliers = append(multipliers, v.Float())
	}

	t.Logf("matched %d spans", len(matched))
	for i, sp := range matched {
		var svc string
		for _, a := range sp.resourceAttrs {
			if a.a.Name == "service.name" {
				svc, _ = a.s.String(), true
			}
		}
		t.Logf("  %d: name=%s service=%s mult=%v",
			i, spanNameOf(sp), svc, multipliers[i])
	}

	require.Equal(t, 2, len(matched), "should match only the 2 shop-backend spans")
	for _, m := range multipliers {
		require.Equal(t, 2.0, m, "shop-backend spans carry ot=th:8 -> multiplier 2")
	}
}

func spanNameOf(s *span) string {
	for _, a := range s.spanAttrs {
		if a.a.Intrinsic == traceql.IntrinsicName {
			str, _ := a.s.String(), true
			return str
		}
	}
	return ""
}

// Keep import used.
var (
	_ = (*v1.Span)(nil)
	_ = (*tempopb.QueryRangeRequest)(nil)
)
