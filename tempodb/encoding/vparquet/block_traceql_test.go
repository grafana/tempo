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

	searchesThatMatch := []traceql.FetchSpansRequest{
		{}, // Empty request
		makeReq(makeCond(LabelName, traceql.OperationEq, "hello")),                        // Intrinsic: name
		makeReq(makeCond(LabelDuration, traceql.OperationEq, uint64(100*time.Second))),    // Intrinsic: duration
		makeReq(makeCond(LabelDuration, traceql.OperationGT, uint64(99*time.Second))),     // Intrinsic: duration
		makeReq(makeCond(LabelDuration, traceql.OperationLT, uint64(101*time.Second))),    // Intrinsic: duration
		makeReq(makeCond("resource."+LabelServiceName, traceql.OperationEq, "myservice")), // Well-known attribute: service.name
		makeReq(makeCond("."+LabelHTTPStatusCode, traceql.OperationEq, int64(500))),       // Well-known attribute: http.status_code int
		makeReq(makeCond("."+LabelHTTPStatusCode, traceql.OperationGT, int64(200))),       // Well-known attribute: http.status_code int
		makeReq(makeCond(".float", traceql.OperationGT, 456.7)),                           // Float
		makeReq(makeCond(".float", traceql.OperationLT, 456.781)),                         // Float
		makeReq(makeCond(".bool", traceql.OperationEq, false)),                            // Bool
		makeReq(makeCond(".foo", traceql.OperationIn, "def", "xyz")),                      // String IN
		makeReq(makeCond(".foo", traceql.OperationIn, "xyz", "def")),                      // String IN Same as above but reversed order
		makeReq(makeCond(".foo", traceql.OperationRegexIn, "d.*")),                        // Regex IN
		makeReq(makeCond("resource.foo", traceql.OperationEq, "abc")),                     // Resource-level only
		makeReq(makeCond("span.foo", traceql.OperationEq, "def")),                         // Span-level only
		makeReq(makeCond(".foo", traceql.OperationNone)),                                  // Projection only
		makeReq(
			// Matches either condition
			makeCond(".foo", traceql.OperationEq, "baz"),
			makeCond("."+LabelHTTPStatusCode, traceql.OperationGT, int64(100)),
		),
		makeReq(
			// Same as above but reversed order
			makeCond("."+LabelHTTPStatusCode, traceql.OperationGT, int64(100)),
			makeCond(".foo", traceql.OperationEq, "baz"),
		),
		makeReq(
			// Same attribute with mixed types
			makeCond(".foo", traceql.OperationGT, int64(100)),
			makeCond(".foo", traceql.OperationEq, "def"),
		),
		makeReq(
			// Multiple conditions on same well-known attribute, matches either
			makeCond("."+LabelHTTPStatusCode, traceql.OperationEq, int64(500)),
			makeCond("."+LabelHTTPStatusCode, traceql.OperationGT, int64(500)),
		),
		// Edge cases
		makeReq(makeCond(".name", traceql.OperationEq, "Bob")),                           // Almost conflicts with intrinsic but still works
		makeReq(makeCond("resource."+LabelServiceName, traceql.OperationEq, int64(123))), // service.name doesn't match type of dedicated column
		makeReq(makeCond("."+LabelServiceName, traceql.OperationEq, "spanservicename")),  // service.name present on span
		makeReq(makeCond("."+LabelHTTPStatusCode, traceql.OperationEq, "500ouch")),       // http.status_code doesn't match type of dedicated column
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
		makeReq(makeCond(".foo", traceql.OperationEq, "abc")),                        // This should not return results because the span has overridden this attribute to "def".
		makeReq(makeCond(".foo", traceql.OperationIn, "abc", "xyz")),                 // Same as above but additional test value
		makeReq(makeCond(".foo", traceql.OperationRegexIn, "xyz.*")),                 // Regex IN
		makeReq(makeCond(".span.bool", traceql.OperationEq, true)),                   // Bool not match
		makeReq(makeCond(LabelName, traceql.OperationEq, "nothello")),                // Well-known attribute: name not match
		makeReq(makeCond("."+LabelServiceName, traceql.OperationEq, "notmyservice")), // Well-known attribute: service.name not match
		makeReq(makeCond("."+LabelHTTPStatusCode, traceql.OperationEq, int64(200))),  // Well-known attribute: http.status_code not match
		makeReq(makeCond("."+LabelHTTPStatusCode, traceql.OperationGT, int64(600))),  // Well-known attribute: http.status_code not match
		makeReq(
			// Matches neither condition
			makeCond(".foo", traceql.OperationEq, "xyz"),
			makeCond("."+LabelHTTPStatusCode, traceql.OperationEq, 1000),
		),
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
				makeCond("span.foo", traceql.OperationEq, "bar"),      // matches resource but not span
				makeCond("span.bar", traceql.OperationEq, int64(123)), // matches
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
				makeCond("resource.foo", traceql.OperationEq, "abc"), // matches resource but not span
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
				makeCond(".foo", traceql.OperationEq, "xyz"),                       // doesn't match anything
				makeCond("."+LabelHTTPStatusCode, traceql.OperationEq, int64(500)), // matches span
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
				makeCond(".foo", traceql.OperationNone),                  // String
				makeCond("."+LabelHTTPStatusCode, traceql.OperationNone), // Int
				makeCond(".float", traceql.OperationNone),                // Float
				makeCond(".bool", traceql.OperationNone),                 // bool
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

		{
			// Empty request returns 1 spanset with all spans
			traceql.FetchSpansRequest{},
			makeSpansets(
				makeSpanset(
					wantTr.TraceID,
					traceql.Span{
						ID:                 wantTr.ResourceSpans[0].InstrumentationLibrarySpans[0].Spans[0].ID,
						StartTimeUnixNanos: wantTr.ResourceSpans[0].InstrumentationLibrarySpans[0].Spans[0].StartUnixNanos,
						EndtimeUnixNanos:   wantTr.ResourceSpans[0].InstrumentationLibrarySpans[0].Spans[0].EndUnixNanos,
						Attributes:         map[string]interface{}{},
					},
					traceql.Span{
						ID:                 wantTr.ResourceSpans[1].InstrumentationLibrarySpans[0].Spans[0].ID,
						StartTimeUnixNanos: wantTr.ResourceSpans[1].InstrumentationLibrarySpans[0].Spans[0].StartUnixNanos,
						EndtimeUnixNanos:   wantTr.ResourceSpans[1].InstrumentationLibrarySpans[0].Spans[0].EndUnixNanos,
						Attributes:         map[string]interface{}{},
					},
				),
			),
		},

		{
			// 2nd span only
			makeReq(makeCond(LabelName, traceql.OperationEq, "world")),
			makeSpansets(
				makeSpanset(
					wantTr.TraceID,
					traceql.Span{
						ID:                 wantTr.ResourceSpans[1].InstrumentationLibrarySpans[0].Spans[0].ID,
						StartTimeUnixNanos: wantTr.ResourceSpans[1].InstrumentationLibrarySpans[0].Spans[0].StartUnixNanos,
						EndtimeUnixNanos:   wantTr.ResourceSpans[1].InstrumentationLibrarySpans[0].Spans[0].EndUnixNanos,
						Attributes: map[string]interface{}{
							LabelName: "world",
						},
					},
				),
			),
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

func makeReq(conditions ...traceql.Condition) traceql.FetchSpansRequest {
	return traceql.FetchSpansRequest{
		Conditions: conditions,
	}
}

func makeCond(k string, op traceql.Operation, v ...interface{}) traceql.Condition {
	return traceql.Condition{Selector: k, Operation: op, Operands: v}
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
						{Key: LabelServiceName, ValueInt: intPtr(123)}, // Different type than dedicated column
					},
				},
				InstrumentationLibrarySpans: []ILS{
					{
						Spans: []Span{
							{
								ID:             []byte("spanid"),
								Name:           "hello",
								StartUnixNanos: uint64(100 * time.Second),
								EndUnixNanos:   uint64(200 * time.Second),
								HttpMethod:     strPtr("get"),
								HttpUrl:        strPtr("url/hello/world"),
								HttpStatusCode: intPtr(500),
								ParentSpanID:   []byte{},
								StatusCode:     int(v1.Status_STATUS_CODE_ERROR),
								Attrs: []Attribute{
									{Key: "foo", Value: strPtr("def")},
									{Key: "bar", ValueInt: intPtr(123)},
									{Key: "float", ValueDouble: fltPtr(456.78)},
									{Key: "bool", ValueBool: boolPtr(false)},

									// Edge-cases
									{Key: LabelName, Value: strPtr("Bob")},                    // Conflicts with intrinsic but still looked up by .name
									{Key: LabelServiceName, Value: strPtr("spanservicename")}, // Overrides resource-level dedicated column
									{Key: LabelHTTPStatusCode, Value: strPtr("500ouch")},      // Different type than dedicated column
								},
							},
						},
					},
				},
			},
			{
				Resource: Resource{
					ServiceName: "service2",
				},
				InstrumentationLibrarySpans: []ILS{
					{
						Spans: []Span{
							{
								ID:   []byte("spanid2"),
								Name: "world",
							},
						},
					},
				},
			},
		},
	}
}
