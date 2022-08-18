package vparquet

import (
	"context"
	"testing"
	"time"

	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/require"
)

func TestBackendBlockSearchTraceQL(t *testing.T) {

	// Helper functions to make pointers
	strPtr := func(s string) *string { return &s }
	intPtr := func(i int64) *int64 { return &i }

	// This is a fully populated trace that we
	// search many different ways.
	wantTr := &Trace{
		TraceID:           test.ValidTraceID(nil),
		StartTimeUnixNano: uint64(1000 * time.Second),
		EndTimeUnixNano:   uint64(2000 * time.Second),
		DurationNanos:     uint64((100 * time.Millisecond).Nanoseconds()),
		RootServiceName:   "RootService",
		RootSpanName:      "RootSpan",
		ResourceSpans: []ResourceSpans{
			{
				Resource: Resource{
					ServiceName:      "myservice",
					Cluster:          strPtr("cluster"),
					Namespace:        strPtr("namespace"),
					Pod:              strPtr("pod"),
					Container:        strPtr("container"),
					K8sClusterName:   strPtr("k8scluster"),
					K8sNamespaceName: strPtr("k8snamespace"),
					K8sPodName:       strPtr("k8spod"),
					K8sContainerName: strPtr("k8scontainer"),
					Attrs: []Attribute{
						{Key: "foo", Value: strPtr("abc")},
					},
				},
				InstrumentationLibrarySpans: []ILS{
					{
						Spans: []Span{
							{
								Name:           "hello",
								StartUnixNanos: uint64(100 * time.Second),
								EndUnixNanos:   uint64(200 * time.Second),
								HttpMethod:     strPtr("get"),
								HttpUrl:        strPtr("url/hello/world"),
								HttpStatusCode: intPtr(500),
								ID:             []byte("spanid"),
								ParentSpanID:   []byte{},
								StatusCode:     int(v1.Status_STATUS_CODE_ERROR),
								Attrs: []Attribute{
									{Key: "foo", Value: strPtr("def")},
								},
							},
						},
					},
				},
			},
		},
	}

	b := makeBackendBlockWithTraces(t, []*Trace{wantTr})
	ctx := context.TODO()

	// Helper function to make a basic search
	makeReq := func(k string, op traceql.Operation, v ...interface{}) traceql.FetchSpansRequest {
		return traceql.FetchSpansRequest{
			Conditions: []traceql.Condition{
				{
					Selector:  k,
					Operation: op,
					Operands:  v,
				},
			},
		}
	}

	searchesThatMatch := []traceql.FetchSpansRequest{
		makeReq(LabelName, traceql.OperationEq, "hello"),
		makeReq(LabelServiceName, traceql.OperationEq, "myservice"),
		makeReq(LabelHTTPStatusCode, traceql.OperationEq, int64(500)),
		makeReq(LabelHTTPStatusCode, traceql.OperationGT, int64(200)),
		makeReq(".foo", traceql.OperationEq, "abc"),
		makeReq(".foo", traceql.OperationEq, "def"),
		makeReq(".foo", traceql.OperationIn, "abc", "xyz"), // Matches either condition
		makeReq(".foo", traceql.OperationIn, "xyz", "abc"), // Same as above but reversed order
		makeReq(".resource.foo", traceql.OperationEq, "abc"),
		makeReq(".span.foo", traceql.OperationEq, "def"),
		makeReq(".foo", traceql.OperationNone), // Here we are only projecting the value up to higher logic
		{
			// Matches either condition
			Conditions: []traceql.Condition{
				{Selector: ".foo", Operation: traceql.OperationEq, Operands: []interface{}{"baz"}},
				{Selector: LabelHTTPStatusCode, Operation: traceql.OperationGT, Operands: []interface{}{100}},
			},
		},
		{
			// Same as above but reversed order
			Conditions: []traceql.Condition{
				{Selector: LabelHTTPStatusCode, Operation: traceql.OperationGT, Operands: []interface{}{100}},
				{Selector: ".foo", Operation: traceql.OperationEq, Operands: []interface{}{"baz"}},
			},
		},
	}

	for _, req := range searchesThatMatch {
		resp, err := b.Fetch(ctx, req)
		require.NoError(t, err, "search request:", req)

		spanSet, err := resp.Results.Next(ctx)
		require.NoError(t, err, "search request:", req)
		require.NotNil(t, spanSet, "search request:", req)
		require.Equal(t, wantTr.TraceID, spanSet.TraceID, "search request:", req)
		require.Equal(t, []byte("spanid"), spanSet.Spans[0].ID, "search request:", req)
	}

	searchesThatDontMatch := []traceql.FetchSpansRequest{
		makeReq(LabelName, traceql.OperationEq, "nothello"),
		makeReq(LabelServiceName, traceql.OperationEq, "notmyservice"),
		makeReq(LabelHTTPStatusCode, traceql.OperationEq, int64(200)),
		makeReq(LabelHTTPStatusCode, traceql.OperationGT, int64(600)),
		makeReq(".foo", traceql.OperationEq, "xyz"),
		{
			// Matches neither condition
			Conditions: []traceql.Condition{
				{Selector: ".foo", Operation: traceql.OperationEq, Operands: []interface{}{"xyz"}},
				{Selector: LabelHTTPStatusCode, Operation: traceql.OperationEq, Operands: []interface{}{1000}},
			},
		},
	}

	for _, req := range searchesThatDontMatch {
		resp, err := b.Fetch(ctx, req)
		require.NoError(t, err, "search request:", req)

		spanSet, err := resp.Results.Next(ctx)
		require.NoError(t, err, "search request:", req)
		require.Nil(t, spanSet, "search request:", req)
	}
}
