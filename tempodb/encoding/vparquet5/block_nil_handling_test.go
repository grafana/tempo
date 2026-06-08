package vparquet5

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

func TestNilHandlingBlockSearchTraceQL(t *testing.T) {
	var (
		ctx          = context.Background()
		wantTraceID1 = test.ValidTraceID(nil)
		wantTraceID2 = test.ValidTraceID(nil)
		dc           = []backend.DedicatedColumn{
			{Scope: "resource", Name: "dc_present", Type: "string"},
			{Scope: "resource", Name: "dc_nil", Type: "string"},
			{Scope: "span", Name: "dc_present", Type: "string"},
			{Scope: "span", Name: "dc_nil", Type: "string"},
			{Scope: "event", Name: "dc_present", Type: "string"},
			{Scope: "event", Name: "dc_nil", Type: "string"},
		}
	)

	traces := []*Trace{
		{
			TraceID: wantTraceID1,
			ResourceSpans: []ResourceSpans{
				{
					Resource: Resource{
						Attrs: []Attribute{
							attr("foo", "bar"),
						},
						DedicatedAttributes: DedicatedAttributes{
							String01: []string{"dc_present"},
						},
					},
					ScopeSpans: []ScopeSpans{
						{
							Scope: InstrumentationScope{
								Attrs: []Attribute{
									attr("foo", "bar"),
								},
							},
							Spans: []Span{
								{
									SpanID: []byte{1, 23},
									Attrs: []Attribute{
										attr("foo", "bar"),
									},
									DedicatedAttributes: DedicatedAttributes{
										String01: []string{"dc_present"},
									},
									Events: []Event{
										{
											Attrs: []Attribute{
												attr("foo", "bar"),
											},
											DedicatedAttributes: DedicatedAttributes{
												String01: []string{"dc_present"},
											},
										},
									},
									Links: []Link{
										{
											Attrs: []Attribute{
												attr("foo", "bar"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			TraceID:      wantTraceID2,
			RootSpanName: "empty",
			ResourceSpans: []ResourceSpans{
				{
					Resource: Resource{},
					ScopeSpans: []ScopeSpans{
						{
							Spans: []Span{
								{
									Name: "empty",
								},
							},
						},
					},
				},
			},
		},
	}

	searchesThatMatch := []struct {
		wantTraceID []byte
		query       string
	}{
		{wantTraceID1, `{resource.dc_nil = nil}`},
		{wantTraceID1, `{resource.bar = nil}`},
		{wantTraceID1, `{span.dc_nil = nil}`},
		{wantTraceID1, `{span.bar = nil}`},
		{wantTraceID1, `{instrumentation.bar = nil}`},
		{wantTraceID1, `{event.dc_nil = nil}`},
		{wantTraceID1, `{event.bar = nil}`},
		{wantTraceID1, `{link.bar = nil}`},
		{wantTraceID1, `{resource.foo != nil}`},
		{wantTraceID1, `{resource.dc_present != nil}`},
		{wantTraceID1, `{span.foo != nil}`},
		{wantTraceID1, `{span.dc_present != nil}`},
		{wantTraceID1, `{instrumentation.foo != nil}`},
		{wantTraceID1, `{event.foo != nil}`},
		{wantTraceID1, `{event.dc_present != nil}`},
		{wantTraceID1, `{link.foo != nil}`},

		{wantTraceID2, `{trace:rootName = "empty" && span.bar = nil}`},
		{wantTraceID2, `{trace:rootName = "empty" && span.dc_nil= nil}`},
		{wantTraceID2, `{span:name = "empty" && span.bar = nil}`},
		{wantTraceID2, `{span:name = "empty" && span.dc_nil = nil}`},
	}

	searchesThatDontMatch := []struct {
		dontWantTraceID []byte
		query           string
	}{
		// = nil on attributes that exist
		{wantTraceID1, `{resource.foo = nil}`},
		{wantTraceID1, `{resource.dc_present = nil}`},
		{wantTraceID1, `{span.foo = nil}`},
		{wantTraceID1, `{span.dc_present = nil}`},
		{wantTraceID1, `{instrumentation.foo = nil}`},
		{wantTraceID1, `{event.dc_present = nil}`},
		{wantTraceID1, `{link.foo = nil}`},
		// != nil on attributes that do not exist
		{wantTraceID1, `{resource.baz != nil}`},
		{wantTraceID1, `{span.baz != nil}`},
		{wantTraceID1, `{instrumentation.baz != nil}`},
		{wantTraceID1, `{event.baz != nil}`},
		{wantTraceID1, `{link.baz != nil}`},

		{wantTraceID2, `{event.foo = nil}`}, // Doesn't match because this span has no events.
		{wantTraceID2, `{link.foo = nil}`},  // Doesn't match because this span has no links.
	}

	b := makeBackendBlockWithTracesWithDedicatedColumns(t, traces, dc)

	for _, tc := range searchesThatMatch {
		t.Run("match/"+tc.query, func(t *testing.T) {
			req := traceql.MustExtractFetchSpansRequestWithMetadata(tc.query)
			if req.SecondPass == nil {
				req.SecondPass = func(s *traceql.Spanset) ([]*traceql.Spanset, error) { return []*traceql.Spanset{s}, nil }
				req.SecondPassConditions = traceql.SearchMetaConditions()
			}

			resp, err := b.Fetch(ctx, req, common.DefaultSearchOptions())
			require.NoError(t, err, "query: %s", tc)

			found := false
			for {
				spanSet, err := resp.Results.Next(ctx)
				require.NoError(t, err, "query: %s", tc)
				if spanSet == nil {
					break
				}
				if bytes.Equal(spanSet.TraceID, tc.wantTraceID) {
					found = true
					break
				}
			}
			require.True(t, found, "query: %s", tc)
		})
	}

	for _, tc := range searchesThatDontMatch {
		t.Run("no_match/"+tc.query, func(t *testing.T) {
			req := traceql.MustExtractFetchSpansRequestWithMetadata(tc.query)
			if req.SecondPass == nil {
				req.SecondPass = func(s *traceql.Spanset) ([]*traceql.Spanset, error) { return []*traceql.Spanset{s}, nil }
				req.SecondPassConditions = traceql.SearchMetaConditions()
			}

			resp, err := b.Fetch(ctx, req, common.DefaultSearchOptions())
			require.NoError(t, err, "query: %s", tc.query)

			for {
				spanSet, err := resp.Results.Next(ctx)
				require.NoError(t, err, "query: %s", tc.query)
				if spanSet == nil {
					break
				}
				require.NotEqual(t, tc.dontWantTraceID, spanSet.TraceID, "query: %s", tc.query)
			}
		})
	}
}
