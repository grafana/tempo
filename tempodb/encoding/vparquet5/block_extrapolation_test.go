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
		matched      []*span
		multipliers  []float64
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
var _ = (*v1.Span)(nil)
var _ = (*tempopb.QueryRangeRequest)(nil)
