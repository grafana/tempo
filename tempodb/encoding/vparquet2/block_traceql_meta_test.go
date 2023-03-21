package vparquet2

import (
	"context"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/require"
)

func TestBackendBlockSearchFetchMetaData(t *testing.T) {
	wantTr := fullyPopulatedTestTrace(nil)
	b := makeBackendBlockWithTraces(t, []*Trace{wantTr})
	ctx := context.Background()

	// Helper functions to make requests

	makeSpansets := func(sets ...*traceql.Spanset) []*traceql.Spanset {
		return sets
	}

	makeSpanset := func(traceID []byte, rootSpanName, rootServiceName string, startTimeUnixNano, durationNanos uint64, spans ...traceql.Span) *traceql.Spanset {
		return &traceql.Spanset{
			TraceID:            traceID,
			RootSpanName:       rootSpanName,
			RootServiceName:    rootServiceName,
			StartTimeUnixNanos: startTimeUnixNano,
			DurationNanos:      durationNanos,
			Spans:              spans,
		}
	}

	testCases := []struct {
		req             traceql.FetchSpansRequest
		expectedResults []*traceql.Spanset
	}{
		{
			// Empty request returns 1 spanset with all spans
			traceql.FetchSpansRequest{},
			makeSpansets(
				makeSpanset(
					wantTr.TraceID,
					wantTr.RootSpanName,
					wantTr.RootServiceName,
					wantTr.StartTimeUnixNano,
					wantTr.DurationNano,
					&span{
						id:                 wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].ID,
						startTimeUnixNanos: wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].StartTimeUnixNano,
						endtimeUnixNanos:   wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].EndTimeUnixNano,
						attributes:         map[traceql.Attribute]traceql.Static{},
					},
					&span{
						id:                 wantTr.ResourceSpans[1].ScopeSpans[0].Spans[0].ID,
						startTimeUnixNanos: wantTr.ResourceSpans[1].ScopeSpans[0].Spans[0].StartTimeUnixNano,
						endtimeUnixNanos:   wantTr.ResourceSpans[1].ScopeSpans[0].Spans[0].EndTimeUnixNano,
						attributes:         map[traceql.Attribute]traceql.Static{},
					},
				),
			),
		},
		{
			// Span attributes lookup
			// Only matches 1 condition. Returns span but only attributes that matched
			makeReq(
				parse(t, `{span.foo = "bar"}`), // matches resource but not span
				parse(t, `{span.bar = 123}`),   // matches
			),
			makeSpansets(
				makeSpanset(
					wantTr.TraceID,
					wantTr.RootSpanName,
					wantTr.RootServiceName,
					wantTr.StartTimeUnixNano,
					wantTr.DurationNano,
					&span{
						id:                 wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].ID,
						startTimeUnixNanos: wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].StartTimeUnixNano,
						endtimeUnixNanos:   wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].EndTimeUnixNano,
						attributes: map[traceql.Attribute]traceql.Static{
							// foo not returned because the span didn't match it
							traceql.NewScopedAttribute(traceql.AttributeScopeSpan, false, "bar"): traceql.NewStaticInt(123),
						},
					},
				),
			),
		},

		{
			// Resource attributes lookup
			makeReq(
				parse(t, `{resource.foo = "abc"}`), // matches resource but not span
			),
			makeSpansets(
				makeSpanset(
					wantTr.TraceID,
					wantTr.RootSpanName,
					wantTr.RootServiceName,
					wantTr.StartTimeUnixNano,
					wantTr.DurationNano,
					&span{
						id:                 wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].ID,
						startTimeUnixNanos: wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].StartTimeUnixNano,
						endtimeUnixNanos:   wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].EndTimeUnixNano,
						attributes: map[traceql.Attribute]traceql.Static{
							// Foo matched on resource.
							// TODO - This seems misleading since the span has foo=<something else>
							//        but for this query we never even looked at span attribute columns.
							newResAttr("foo"): traceql.NewStaticString("abc"),
						},
					},
				),
			),
		},

		{
			// Multiple attributes, only 1 matches and is returned
			makeReq(
				parse(t, `{.foo = "xyz"}`),                   // doesn't match anything
				parse(t, `{.`+LabelHTTPStatusCode+` = 500}`), // matches span
			),
			makeSpansets(
				makeSpanset(
					wantTr.TraceID,
					wantTr.RootSpanName,
					wantTr.RootServiceName,
					wantTr.StartTimeUnixNano,
					wantTr.DurationNano,
					&span{
						id:                 wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].ID,
						startTimeUnixNanos: wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].StartTimeUnixNano,
						endtimeUnixNanos:   wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].EndTimeUnixNano,
						attributes: map[traceql.Attribute]traceql.Static{
							newSpanAttr(LabelHTTPStatusCode): traceql.NewStaticInt(500), // This is the only attribute that matched anything
						},
					},
				),
			),
		},

		{
			// Project attributes of all types
			makeReq(
				parse(t, `{.foo }`),                    // String
				parse(t, `{.`+LabelHTTPStatusCode+`}`), // Int
				parse(t, `{.float }`),                  // Float
				parse(t, `{.bool }`),                   // bool
			),
			makeSpansets(
				makeSpanset(
					wantTr.TraceID,
					wantTr.RootSpanName,
					wantTr.RootServiceName,
					wantTr.StartTimeUnixNano,
					wantTr.DurationNano,
					&span{
						id:                 wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].ID,
						startTimeUnixNanos: wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].StartTimeUnixNano,
						endtimeUnixNanos:   wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].EndTimeUnixNano,
						attributes: map[traceql.Attribute]traceql.Static{
							newResAttr("foo"):                traceql.NewStaticString("abc"), // Both are returned
							newSpanAttr("foo"):               traceql.NewStaticString("def"), // Both are returned
							newSpanAttr(LabelHTTPStatusCode): traceql.NewStaticInt(500),
							newSpanAttr("float"):             traceql.NewStaticFloat(456.78),
							newSpanAttr("bool"):              traceql.NewStaticBool(false),
						},
					},
				),
			),
		},

		{
			// doesn't match anything
			makeReq(parse(t, `{.xyz = "xyz"}`)),
			nil,
		},

		{
			// Intrinsics. 2nd span only
			makeReq(
				parse(t, `{ name = "world" }`),
				parse(t, `{ status = unset }`),
			),
			makeSpansets(
				makeSpanset(
					wantTr.TraceID,
					wantTr.RootSpanName,
					wantTr.RootServiceName,
					wantTr.StartTimeUnixNano,
					wantTr.DurationNano,
					&span{
						id:                 wantTr.ResourceSpans[1].ScopeSpans[0].Spans[0].ID,
						startTimeUnixNanos: wantTr.ResourceSpans[1].ScopeSpans[0].Spans[0].StartTimeUnixNano,
						endtimeUnixNanos:   wantTr.ResourceSpans[1].ScopeSpans[0].Spans[0].EndTimeUnixNano,
						attributes: map[traceql.Attribute]traceql.Static{
							traceql.NewIntrinsic(traceql.IntrinsicName):   traceql.NewStaticString("world"),
							traceql.NewIntrinsic(traceql.IntrinsicStatus): traceql.NewStaticStatus(traceql.StatusUnset),
						},
					},
				),
			),
		},
		{
			// Intrinsic duration with no filtering
			traceql.FetchSpansRequest{Conditions: []traceql.Condition{{Attribute: traceql.NewIntrinsic(traceql.IntrinsicDuration)}}},
			makeSpansets(
				makeSpanset(
					wantTr.TraceID,
					wantTr.RootSpanName,
					wantTr.RootServiceName,
					wantTr.StartTimeUnixNano,
					wantTr.DurationNano,
					&span{
						id:                 wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].ID,
						startTimeUnixNanos: wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].StartTimeUnixNano,
						endtimeUnixNanos:   wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].EndTimeUnixNano,
						attributes: map[traceql.Attribute]traceql.Static{
							traceql.NewIntrinsic(traceql.IntrinsicDuration): traceql.NewStaticDuration(100 * time.Second),
						},
					},
					&span{
						id:                 wantTr.ResourceSpans[1].ScopeSpans[0].Spans[0].ID,
						startTimeUnixNanos: wantTr.ResourceSpans[1].ScopeSpans[0].Spans[0].StartTimeUnixNano,
						endtimeUnixNanos:   wantTr.ResourceSpans[1].ScopeSpans[0].Spans[0].EndTimeUnixNano,
						attributes: map[traceql.Attribute]traceql.Static{
							traceql.NewIntrinsic(traceql.IntrinsicDuration): traceql.NewStaticDuration(0 * time.Second),
						},
					},
				),
			),
		},
	}

	for _, tc := range testCases {
		req := tc.req
		resp, err := b.Fetch(ctx, req, common.DefaultSearchOptions())
		require.NoError(t, err, "search request:", req)

		// Turn iterator into slice
		var ss []*traceql.Spanset
		for {
			spanSet, err := resp.Results.Next(ctx)
			require.NoError(t, err)
			if spanSet == nil {
				break
			}
			ss = append(ss, spanSet)
		}

		// equal will fail on the rownum mismatches. this is an internal detail to the
		// fetch layer. just wipe them out here
		for _, s := range ss {
			for _, sp := range s.Spans {
				sp.(*span).rowNum = parquetquery.RowNumber{}
			}
		}

		require.Equal(t, tc.expectedResults, ss, "search request:", req)
	}
}
