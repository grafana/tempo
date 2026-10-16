package vparquet5

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	v1_common "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/v3/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/v3/pkg/traceql"
	"github.com/grafana/tempo/v3/pkg/util/test"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
)

// TestMetricsSkipMissingAttribute: a span without the attribute matches neither
// != chains nor = chains. A single-condition metrics query skips the engine's second
// pass, so the storage predicate alone decides what is counted.
func TestMetricsSkipMissingAttribute(t *testing.T) {
	const attr = "dedicated.span.1" // a dedicated column, so a missing value is a null
	start := time.Now().Add(-time.Minute)

	var spans []*v1.Span
	for i := range 5 {
		s := &v1.Span{
			SpanId:            test.ValidTraceID(nil)[:8],
			Name:              "span",
			StartTimeUnixNano: uint64(start.UnixNano()) + uint64(i),
			EndTimeUnixNano:   uint64(start.Add(time.Second).UnixNano()),
		}
		if i < 3 {
			s.Attributes = []*v1_common.KeyValue{{
				Key:   attr,
				Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "b"}},
			}}
		}
		spans = append(spans, s)
	}
	id := test.ValidTraceID(nil)
	pb := &tempopb.Trace{ResourceSpans: []*v1.ResourceSpans{{
		Resource:   &v1_resource.Resource{Attributes: []*v1_common.KeyValue{{Key: "service.name", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "svc"}}}}},
		ScopeSpans: []*v1.ScopeSpans{{Spans: spans}},
	}}}
	tr, _ := traceToParquet(&backend.BlockMeta{DedicatedColumns: test.MakeDedicatedColumns()}, id, pb, nil)
	block := makeBackendBlockWithTraces(t, []*Trace{tr})

	opts := common.DefaultSearchOptions()
	fetcher := traceql.NewSpansetFetcherWrapperBoth(
		func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			return block.Fetch(ctx, req, opts)
		},
		func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansOnlyResponse, error) {
			return block.FetchSpans(ctx, req, opts)
		},
	)

	st := uint64(start.Add(-10 * time.Second).UnixNano())
	end := uint64(start.Add(10 * time.Second).UnixNano())
	for _, tc := range []struct {
		query string
		want  float64
	}{
		{`{ span.dedicated.span.1 != "a" && span.dedicated.span.1 != "c" } | count_over_time()`, 3},
		{`{ span.dedicated.span.1 = "b" || span.dedicated.span.1 = "" } | count_over_time()`, 3},
	} {
		t.Run(tc.query, func(t *testing.T) {
			req := &tempopb.QueryRangeRequest{Query: tc.query, Step: uint64(time.Minute), Start: st, End: end, MaxSeries: 1000}
			eval, err := traceql.NewEngine().CompileMetricsQueryRange(req)
			require.NoError(t, err)
			require.NoError(t, eval.Do(context.Background(), fetcher, st, end, int(req.MaxSeries)))

			var total float64
			for _, s := range eval.Results() {
				for _, v := range s.Values {
					total += v
				}
			}
			require.Equal(t, tc.want, total)
		})
	}
}
