package combiner

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/api"
	modeltrace "github.com/grafana/tempo/v3/pkg/model/trace"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/v3/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
)

func TestServiceGraphEdgesTraceQueryResponses(t *testing.T) {
	for _, tc := range []struct {
		name       string
		v2, filter bool
		want       int
	}{
		{"v1 preserves original edge records", false, false, 2},
		{"v2 preserves original edge records", true, false, 2},
		{"v2 filtering scrubs hidden endpoint metadata", true, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id := func(value byte) []byte { out := make([]byte, 8); out[7] = value; return out }
			tr := &tempopb.Trace{}
			for i, service := range []string{"A", "gateway", "B"} {
				tr.ResourceSpans = append(tr.ResourceSpans, &tracev1.ResourceSpans{
					Resource: &resourcev1.Resource{Attributes: []*commonv1.KeyValue{tempopb.MakeKeyValueStringPtr("service.name", service)}},
					ScopeSpans: []*tracev1.ScopeSpans{{Spans: []*tracev1.Span{{SpanId: id(byte(i + 1)), ParentSpanId: func() []byte {
						if i == 0 {
							return nil
						}
						return id(byte(i))
					}(), Name: service, Kind: tracev1.Span_SPAN_KIND_INTERNAL}}}},
				})
				if i > 0 {
					tr.ServiceGraphEdges = append(tr.ServiceGraphEdges, &tempopb.ServiceGraphEdge{ParentSpanId: id(byte(i)), ChildSpanId: id(byte(i + 1)), ParentService: []string{"A", "gateway"}[i-1], ChildService: service, ParentKind: 1, ChildKind: 1})
				}
			}
			var c interface {
				AddResponse(PipelineResponse) error
				HTTPFinal(context.Context) (*http.Response, error)
			}
			if tc.v2 {
				opts := TraceByIDV2Options{}
				if tc.filter {
					opts.TraceFilter = dropByNameFilter{drop: "gateway"}
				}
				c = NewTraceByIDV2(0, api.MarshallingFormatJSON, nil, opts)
			} else {
				c = NewTraceByID(0, api.MarshallingFormatJSON, nil, log.NewNopLogger())
			}
			response := toHTTPProtoResponse(t, &tempopb.TraceByIDResponse{Trace: tr, Metrics: &tempopb.TraceByIDMetrics{}}, http.StatusOK)
			response.HTTPResponse().Header = http.Header{api.HeaderContentType: []string{string(api.MarshallingFormatProtobuf)}}
			require.NoError(t, c.AddResponse(response))
			res, err := c.HTTPFinal(context.Background())
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, res.StatusCode)
			body, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			require.NoError(t, res.Body.Close())
			var decoded map[string]any
			require.NoError(t, json.Unmarshal(body, &decoded))
			if tc.v2 {
				decoded = decoded["trace"].(map[string]any)
			}
			records, ok := decoded["serviceGraphEdges"].([]any)
			require.True(t, ok, string(body))
			require.Len(t, records, tc.want)
			if tc.filter {
				edge := records[0].(map[string]any)
				require.Equal(t, true, edge["parentMissing"])
				require.Empty(t, edge["parentService"])
				require.Equal(t, "B", edge["childService"])
			} else {
				first, second := records[0].(map[string]any), records[1].(map[string]any)
				require.Equal(t, "A", first["parentService"])
				require.Equal(t, "gateway", first["childService"])
				require.Equal(t, "gateway", second["parentService"])
				require.Equal(t, "B", second["childService"])
			}
		})
	}
}

func TestServiceGraphEdgesFollowSharedSpanIDDeduplication(t *testing.T) {
	id := func(value byte) []byte { out := make([]byte, 8); out[7] = value; return out }
	tr := &tempopb.Trace{}
	for _, node := range []struct {
		service    string
		id, parent byte
		kind       tracev1.Span_SpanKind
	}{
		{"A", 1, 0, tracev1.Span_SPAN_KIND_INTERNAL},
		{"gateway", 2, 1, tracev1.Span_SPAN_KIND_CLIENT},
		{"B", 2, 1, tracev1.Span_SPAN_KIND_SERVER},
	} {
		var parent []byte
		if node.parent != 0 {
			parent = id(node.parent)
		}
		tr.ResourceSpans = append(tr.ResourceSpans, &tracev1.ResourceSpans{
			Resource:   &resourcev1.Resource{Attributes: []*commonv1.KeyValue{tempopb.MakeKeyValueStringPtr("service.name", node.service)}},
			ScopeSpans: []*tracev1.ScopeSpans{{Spans: []*tracev1.Span{{SpanId: id(node.id), ParentSpanId: parent, Kind: node.kind}}}},
		})
		if node.parent != 0 {
			tr.ServiceGraphEdges = append(tr.ServiceGraphEdges, &tempopb.ServiceGraphEdge{ParentSpanId: id(node.parent), ChildSpanId: id(node.id), ParentService: "A", ChildService: node.service, ChildKind: int32(node.kind), ParentKind: 1})
		}
	}
	tr = deduplicateTrace(context.Background(), tr)
	modeltrace.ReconcileServiceGraphEdges(tr)
	require.Len(t, tr.ServiceGraphEdges, 2)
	client, server := tr.ServiceGraphEdges[0], tr.ServiceGraphEdges[1]
	require.Equal(t, "gateway", client.ChildService)
	require.Equal(t, "gateway", server.ParentService)
	require.Equal(t, "B", server.ChildService)
	require.Equal(t, client.ChildSpanId, server.ParentSpanId)
	require.NotEqual(t, client.ChildSpanId, server.ChildSpanId)
}
