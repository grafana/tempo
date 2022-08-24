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

	wantTr := fullyPopulatedTestTrace()
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

	// scratch pad thinking
	// { span.foo=bar     } // returns spans with foo=bar					<-- this is handled by iterating resources with no filters
	// { .foo=bar         } // returns spans with foo=bar or foo missing    <-- how to do this?  hmm
	// { resource.foo=bar } // returns all spans							<-- this is handled by iterating spans with no filters
	// { .foo=bar || .foo=baz } // returns spans with foo in bar/baz or missing

	searchesThatMatch := []traceql.FetchSpansRequest{
		{}, // Empty request
		makeReq(LabelName, traceql.OperationEq, "hello"),            // Well-known attribute: name
		makeReq(LabelServiceName, traceql.OperationEq, "myservice"), // Well-known attribute: service.name
		makeReq(LabelHTTPStatusCode, traceql.OperationEq, 500),      // Well-known attribute: http.status_code int
		makeReq(LabelHTTPStatusCode, traceql.OperationGT, 200),      // Well-known attribute: http.status_code int
		makeReq(".float", traceql.OperationGT, 456.7),               // Float
		makeReq(".float", traceql.OperationLT, 456.781),             // Float
		makeReq(".bool", traceql.OperationEq, false),                // Bool
		makeReq(".foo", traceql.OperationIn, "def", "xyz"),          // String IN
		makeReq(".foo", traceql.OperationIn, "xyz", "def"),          // String IN Same as above but reversed order
		makeReq(".foo", traceql.OperationRegexIn, "d.*"),            // Regex IN
		makeReq(".resource.foo", traceql.OperationEq, "abc"),        // Resource-level only
		makeReq(".span.foo", traceql.OperationEq, "def"),            // Span-level only
		makeReq(".foo", traceql.OperationNone),                      // Projection only
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
		makeReq(".foo", traceql.OperationEq, "abc"),        // This should not return results because the span has overridden this attribute to "def".
		makeReq(".foo", traceql.OperationIn, "abc", "xyz"), // Same as above but additional test value
		makeReq(".foo", traceql.OperationRegexIn, "xyz.*"), // Regex IN
		makeReq(".span.bool", traceql.OperationEq, true),
		makeReq(LabelName, traceql.OperationEq, "nothello"),
		makeReq(LabelServiceName, traceql.OperationEq, "notmyservice"),
		makeReq(LabelHTTPStatusCode, traceql.OperationEq, int64(200)),
		makeReq(LabelHTTPStatusCode, traceql.OperationGT, int64(600)),
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

func TestBackendBlockSearchTraceQLResults(t *testing.T) {
	wantTr := fullyPopulatedTestTrace()
	b := makeBackendBlockWithTraces(t, []*Trace{wantTr})
	ctx := context.TODO()

	// Helper functions to make requests

	makeReq := func(conditions ...traceql.Condition) traceql.FetchSpansRequest {
		return traceql.FetchSpansRequest{
			Conditions: conditions,
		}
	}

	makeCond := func(k string, op traceql.Operation, v ...interface{}) traceql.Condition {
		return traceql.Condition{Selector: k, Operation: op, Operands: v}
	}

	makeSpansets := func(sets ...traceql.Spanset) []traceql.Spanset {
		return sets
	}

	makeSpanset := func(traceID []byte, spans ...traceql.Span) traceql.Spanset {
		return traceql.Spanset{TraceID: traceID, Spans: spans}
	}

	testCases := []struct {
		req             traceql.FetchSpansRequest
		expectedResults []traceql.Spanset
	}{
		{
			// Span attributes lookup
			// Only matches 1 condition. Returns span but only attributes that matched
			makeReq(
				makeCond(".span.foo", traceql.OperationEq, "bar"), // matches resource but not span
				makeCond(".span.bar", traceql.OperationEq, 123),   // matches
			),
			makeSpansets(
				makeSpanset(
					wantTr.TraceID,
					traceql.Span{
						ID:                 wantTr.ResourceSpans[0].InstrumentationLibrarySpans[0].Spans[0].ID,
						StartTimeUnixNanos: wantTr.ResourceSpans[0].InstrumentationLibrarySpans[0].Spans[0].StartUnixNanos,
						EndtimeUnixNanos:   wantTr.ResourceSpans[0].InstrumentationLibrarySpans[0].Spans[0].EndUnixNanos,
						Attributes: map[string]interface{}{
							// foo not returned because the span didn't match it
							"bar": int64(123),
						},
					},
				),
			),
		},

		{
			// Resource attributes lookup
			makeReq(
				makeCond(".resource.foo", traceql.OperationEq, "abc"), // matches resource but not span
			),
			makeSpansets(
				makeSpanset(
					wantTr.TraceID,
					traceql.Span{
						ID:                 wantTr.ResourceSpans[0].InstrumentationLibrarySpans[0].Spans[0].ID,
						StartTimeUnixNanos: wantTr.ResourceSpans[0].InstrumentationLibrarySpans[0].Spans[0].StartUnixNanos,
						EndtimeUnixNanos:   wantTr.ResourceSpans[0].InstrumentationLibrarySpans[0].Spans[0].EndUnixNanos,
						Attributes: map[string]interface{}{
							// Foo matched on resource.
							// TODO - This seems misleading since the span has foo=<something else>
							//        but for this query we never even looked at span attribute columns.
							"foo": "abc",
						},
					},
				),
			),
		},

		{
			// Multiple attributes, only 1 matches and is returned
			makeReq(
				makeCond(".foo", traceql.OperationEq, "xyz"),            // doesn't match anything
				makeCond(LabelHTTPStatusCode, traceql.OperationEq, 500), // matches span
			),
			makeSpansets(
				makeSpanset(
					wantTr.TraceID,
					traceql.Span{
						ID:                 wantTr.ResourceSpans[0].InstrumentationLibrarySpans[0].Spans[0].ID,
						StartTimeUnixNanos: wantTr.ResourceSpans[0].InstrumentationLibrarySpans[0].Spans[0].StartUnixNanos,
						EndtimeUnixNanos:   wantTr.ResourceSpans[0].InstrumentationLibrarySpans[0].Spans[0].EndUnixNanos,
						Attributes: map[string]interface{}{
							LabelHTTPStatusCode: int64(500), // This is the only attribute that matched anything
						},
					},
				),
			),
		},

		{
			// Project attributes of all types
			makeReq(
				makeCond(".foo", traceql.OperationNone),              // String
				makeCond(LabelHTTPStatusCode, traceql.OperationNone), // Int
				makeCond(".float", traceql.OperationNone),            // Float
				makeCond(".bool", traceql.OperationNone),             // bool
			),
			makeSpansets(
				makeSpanset(
					wantTr.TraceID,
					traceql.Span{
						ID:                 wantTr.ResourceSpans[0].InstrumentationLibrarySpans[0].Spans[0].ID,
						StartTimeUnixNanos: wantTr.ResourceSpans[0].InstrumentationLibrarySpans[0].Spans[0].StartUnixNanos,
						EndtimeUnixNanos:   wantTr.ResourceSpans[0].InstrumentationLibrarySpans[0].Spans[0].EndUnixNanos,
						Attributes: map[string]interface{}{
							"foo":               "def",
							LabelHTTPStatusCode: int64(500),
							"float":             456.78,
							"bool":              false,
						},
					},
				),
			),
		},

		{
			// doesn't match anything
			makeReq(makeCond(".xyz", traceql.OperationEq, "xyz")),
			nil,
		},
	}

	for _, tc := range testCases {
		req := tc.req
		resp, err := b.Fetch(ctx, req)
		require.NoError(t, err, "search request:", req)

		// Turn iterator into slice
		var actualResults []traceql.Spanset
		for {
			spanSet, err := resp.Results.Next(ctx)
			require.NoError(t, err)
			if spanSet == nil {
				break
			}
			actualResults = append(actualResults, *spanSet)
		}
		require.Equal(t, tc.expectedResults, actualResults, "search request:", req)
	}
}

func fullyPopulatedTestTrace() *Trace {
	// Helper functions to make pointers
	strPtr := func(s string) *string { return &s }
	intPtr := func(i int64) *int64 { return &i }
	fltPtr := func(f float64) *float64 { return &f }
	boolPtr := func(b bool) *bool { return &b }

	return &Trace{
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
									{Key: "bar", ValueInt: intPtr(123)},
									{Key: "float", ValueDouble: fltPtr(456.78)},
									{Key: "bool", ValueBool: boolPtr(false)},
								},
							},
						},
					},
				},
			},
		},
	}
}
