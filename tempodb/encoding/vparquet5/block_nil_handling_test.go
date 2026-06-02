package vparquet5

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

func TestNilHandlingBlockSearchTraceQL(t *testing.T) {
	dc := []backend.DedicatedColumn{
		{Scope: "resource", Name: "dc_present", Type: "string"},
		{Scope: "resource", Name: "dc_nil", Type: "string"},
		{Scope: "span", Name: "dc_present", Type: "string"},
		{Scope: "span", Name: "dc_nil", Type: "string"},
		{Scope: "event", Name: "dc_present", Type: "string"},
		{Scope: "event", Name: "dc_nil", Type: "string"},
	}
	// beforeID := test.ValidTraceID(nil)
	wantTraceID := test.ValidTraceID(nil)
	// afterID := test.ValidTraceID(nil)

	// Minimal trace: every scope has foo=bar; baz is omitted and used in "= nil" queries.
	traces := []*Trace{
		// fullyPopulatedTestTrace(beforeID),
		{
			TraceID:     wantTraceID,
			TraceIDText: util.TraceIDToHexString(wantTraceID),
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
			RootSpanName: "no-events",
			ResourceSpans: []ResourceSpans{
				{
					Resource: Resource{},
					ScopeSpans: []ScopeSpans{
						{
							Spans: []Span{
								{},
							},
						},
					},
				},
			},
		},
		// fullyPopulatedTestTrace(afterID),
	}

	b := makeBackendBlockWithTracesWithDedicatedColumns(t, traces, dc)
	// b := makeBackendBlockWithTraces(t, traces)
	ctx := context.Background()

	searchesThatMatch := []string{
		`{resource.dc_nil = nil}`,
		`{resource.bar = nil}`,
		`{span.dc_nil = nil}`,
		`{span.baz = nil}`,
		`{instrumentation.baz = nil}`,
		`{event.dc_nil = nil}`,
		`{event.bar = nil}`,
		`{link.baz = nil}`,
		`{resource.foo != nil}`,
		`{resource.dc_present != nil}`,
		`{span.foo != nil}`,
		`{span.dc_present != nil}`,
		`{instrumentation.foo != nil}`,
		`{event.foo != nil}`,
		`{event.dc_present != nil}`,
		`{link.foo != nil}`,
	}

	searchesThatDontMatch := []string{
		// = nil on attributes that exist
		`{resource.foo = nil}`,
		`{resource.dc_present = nil}`,
		`{span.foo = nil}`,
		`{span.dc_present = nil}`,
		`{instrumentation.foo = nil}`,
		`{trace:rootName = "no-events" && event.foo = nil}`,
		`{event.dc_present = nil}`,
		`{link.foo = nil}`,
		// != nil on attributes that do not exist
		`{resource.baz != nil}`,
		`{span.baz != nil}`,
		`{instrumentation.baz != nil}`,
		`{event.baz != nil}`,
		`{link.baz != nil}`,
	}

	for _, q := range searchesThatMatch {
		t.Run("match/"+q, func(t *testing.T) {
			req := traceql.MustExtractFetchSpansRequestWithMetadata(q)
			if req.SecondPass == nil {
				req.SecondPass = func(s *traceql.Spanset) ([]*traceql.Spanset, error) { return []*traceql.Spanset{s}, nil }
				req.SecondPassConditions = traceql.SearchMetaConditions()
			}

			resp, err := b.Fetch(ctx, req, common.DefaultSearchOptions())
			require.NoError(t, err, "query: %s", q)

			found := false
			for {
				spanSet, err := resp.Results.Next(ctx)
				require.NoError(t, err, "query: %s", q)
				if spanSet == nil {
					break
				}
				if bytes.Equal(spanSet.TraceID, wantTraceID) {
					found = true
					break
				}
			}
			require.True(t, found, "query: %s", q)
		})
	}

	for _, q := range searchesThatDontMatch {
		t.Run("no_match/"+q, func(t *testing.T) {
			req := traceql.MustExtractFetchSpansRequestWithMetadata(q)
			if req.SecondPass == nil {
				req.SecondPass = func(s *traceql.Spanset) ([]*traceql.Spanset, error) { return []*traceql.Spanset{s}, nil }
				req.SecondPassConditions = traceql.SearchMetaConditions()
			}

			resp, err := b.Fetch(ctx, req, common.DefaultSearchOptions())
			require.NoError(t, err, "query: %s", q)

			for {
				spanSet, err := resp.Results.Next(ctx)
				require.NoError(t, err, "query: %s", q)
				if spanSet == nil {
					break
				}
				require.NotEqual(t, wantTraceID, spanSet.TraceID, "query: %s", q)
			}
		})
	}
}
