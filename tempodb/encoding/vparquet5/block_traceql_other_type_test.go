package vparquet5

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/grafana/tempo/v3/pkg/traceql"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
)

func TestFetchReportsDedicatedColumnValuesOfAnotherType(t *testing.T) {
	tests := []struct {
		name     string
		mixedPct int
		query    string
		selected string
		reported bool
	}{
		{name: "no other type", mixedPct: 0, query: `{ span.mixed != nil }`},
		{name: "no other type, select", mixedPct: 0, query: `{ }`, selected: "mixed"},
		{name: "exists", mixedPct: 50, query: `{ span.mixed != nil }`, reported: true},
		{name: "select", mixedPct: 50, query: `{ }`, selected: "mixed", reported: true},
		{name: "resource exists", mixedPct: 50, query: `{ resource.mixed != nil }`, reported: true},
		// A value condition reads only the dedicated column.
		{name: "value", mixedPct: 50, query: `{ span.mixed = "v-1" }`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			block := makeBackendBlockWithMixedTypes(t, tc.mixedPct)

			newReq := func() traceql.FetchSpansRequest {
				req := traceql.MustExtractFetchSpansRequestWithMetadata(tc.query)
				// The engine, not the request extraction, adds select() attributes to the second pass.
				if tc.selected != "" {
					req.SecondPassConditions = append(req.SecondPassConditions, traceql.Condition{Attribute: traceql.NewScopedAttribute(traceql.AttributeScopeSpan, false, tc.selected)})
				}
				req.SecondPass = func(s *traceql.Spanset) ([]*traceql.Spanset, error) { return []*traceql.Spanset{s}, nil }
				return req
			}
			for _, fetch := range []func(context.Context) traceql.CommonIterator[int]{
				func(ctx context.Context) traceql.CommonIterator[int] {
					resp, err := block.Fetch(ctx, newReq(), common.DefaultSearchOptions())
					require.NoError(t, err)
					return spansetCounter{resp.Results}
				},
				func(ctx context.Context) traceql.CommonIterator[int] {
					resp, err := block.FetchSpans(ctx, newReq(), common.DefaultSearchOptions())
					require.NoError(t, err)
					return spanCounter{resp.Results}
				},
			} {
				exporter := testSpans(t)
				ctx, caller := otel.Tracer("test").Start(t.Context(), "caller")
				iter := fetch(ctx)
				for {
					n, err := iter.Next(ctx)
					require.NoError(t, err)
					if n == 0 {
						break
					}
				}
				iter.Close()
				iter.Close()
				caller.End()

				var events []map[attribute.Key]attribute.Value
				for _, s := range exporter.GetSpans() {
					if s.Name != "caller" {
						continue
					}
					for _, e := range s.Events {
						if e.Name == "dedicated column values of another type" {
							attrs := map[attribute.Key]attribute.Value{}
							for _, kv := range e.Attributes {
								attrs[kv.Key] = kv.Value
							}
							events = append(events, attrs)
						}
					}
				}

				if !tc.reported {
					require.Empty(t, events)
					continue
				}
				require.Len(t, events, 1, "a second Close reported again")
				require.Equal(t, block.meta.TenantID, events[0]["tenantID"].AsString())
				require.Equal(t, block.meta.BlockID.String(), events[0]["blockID"].AsString())
				require.Positive(t, events[0]["values"].AsInt64())
			}
		})
	}
}

// spansetCounter and spanCounter return 0 when done, so both fetch APIs drain the same way.
type spansetCounter struct{ traceql.SpansetIterator }

func (c spansetCounter) Next(ctx context.Context) (int, error) {
	ss, err := c.SpansetIterator.Next(ctx)
	if ss == nil || err != nil {
		return 0, err
	}
	return 1, nil
}

type spanCounter struct{ traceql.SpanIterator }

func (c spanCounter) Next(ctx context.Context) (int, error) {
	s, err := c.SpanIterator.Next(ctx)
	if s == nil || err != nil {
		return 0, err
	}
	return 1, nil
}
