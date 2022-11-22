package vparquet

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

func TestBackendBlockSearchTraceQL(t *testing.T) {
	wantTr := fullyPopulatedTestTrace(nil)
	b := makeBackendBlockWithTraces(t, []*Trace{wantTr})
	ctx := context.Background()

	searchesThatMatch := []traceql.FetchSpansRequest{
		{}, // Empty request
		{
			// Time range
			StartTimeUnixNanos: uint64(101 * time.Second),
			EndTimeUnixNanos:   uint64(102 * time.Second),
		},
		// Intrinsics
		makeReq(parse(t, `{`+LabelName+` = "hello"}`)),
		makeReq(parse(t, `{`+LabelDuration+` =  100s}`)),
		makeReq(parse(t, `{`+LabelDuration+` >  99s}`)),
		makeReq(parse(t, `{`+LabelDuration+` >= 100s}`)),
		makeReq(parse(t, `{`+LabelDuration+` <  101s}`)),
		makeReq(parse(t, `{`+LabelDuration+` <= 100s}`)),
		makeReq(parse(t, `{`+LabelDuration+` <= 100s}`)),
		makeReq(parse(t, `{`+LabelStatus+` = error}`)),
		makeReq(parse(t, `{`+LabelStatus+` = 2}`)),
		// Resource well-known attributes
		makeReq(parse(t, `{.`+LabelServiceName+` = "spanservicename"}`)), // Overridden at span
		makeReq(parse(t, `{.`+LabelCluster+` = "cluster"}`)),
		makeReq(parse(t, `{.`+LabelNamespace+` = "namespace"}`)),
		makeReq(parse(t, `{.`+LabelPod+` = "pod"}`)),
		makeReq(parse(t, `{.`+LabelContainer+` = "container"}`)),
		makeReq(parse(t, `{.`+LabelK8sNamespaceName+` = "k8snamespace"}`)),
		makeReq(parse(t, `{.`+LabelK8sClusterName+` = "k8scluster"}`)),
		makeReq(parse(t, `{.`+LabelK8sPodName+` = "k8spod"}`)),
		makeReq(parse(t, `{.`+LabelK8sContainerName+` = "k8scontainer"}`)),
		makeReq(parse(t, `{resource.`+LabelServiceName+` = "myservice"}`)),
		makeReq(parse(t, `{resource.`+LabelCluster+` = "cluster"}`)),
		makeReq(parse(t, `{resource.`+LabelNamespace+` = "namespace"}`)),
		makeReq(parse(t, `{resource.`+LabelPod+` = "pod"}`)),
		makeReq(parse(t, `{resource.`+LabelContainer+` = "container"}`)),
		makeReq(parse(t, `{resource.`+LabelK8sNamespaceName+` = "k8snamespace"}`)),
		makeReq(parse(t, `{resource.`+LabelK8sClusterName+` = "k8scluster"}`)),
		makeReq(parse(t, `{resource.`+LabelK8sPodName+` = "k8spod"}`)),
		makeReq(parse(t, `{resource.`+LabelK8sContainerName+` = "k8scontainer"}`)),
		// Span well-known attributes
		makeReq(parse(t, `{.`+LabelHTTPStatusCode+` = 500}`)),
		makeReq(parse(t, `{.`+LabelHTTPMethod+` = "get"}`)),
		makeReq(parse(t, `{.`+LabelHTTPUrl+` = "url/hello/world"}`)),
		makeReq(parse(t, `{span.`+LabelHTTPStatusCode+` = 500}`)),
		makeReq(parse(t, `{span.`+LabelHTTPMethod+` = "get"}`)),
		makeReq(parse(t, `{span.`+LabelHTTPUrl+` = "url/hello/world"}`)),
		// Basic data types and operations
		makeReq(parse(t, `{.float = 456.78}`)),      // Float ==
		makeReq(parse(t, `{.float != 456.79}`)),     // Float !=
		makeReq(parse(t, `{.float > 456.7}`)),       // Float >
		makeReq(parse(t, `{.float >= 456.78}`)),     // Float >=
		makeReq(parse(t, `{.float < 456.781}`)),     // Float <
		makeReq(parse(t, `{.bool = false}`)),        // Bool ==
		makeReq(parse(t, `{.bool != true}`)),        // Bool !=
		makeReq(parse(t, `{.bar = 123}`)),           // Int ==
		makeReq(parse(t, `{.bar != 124}`)),          // Int !=
		makeReq(parse(t, `{.bar > 122}`)),           // Int >
		makeReq(parse(t, `{.bar >= 123}`)),          // Int >=
		makeReq(parse(t, `{.bar < 124}`)),           // Int <
		makeReq(parse(t, `{.bar <= 123}`)),          // Int <=
		makeReq(parse(t, `{.foo = "def"}`)),         // String ==
		makeReq(parse(t, `{.foo != "deg"}`)),        // String !=
		makeReq(parse(t, `{.foo =~ "d.*"}`)),        // String Regex
		makeReq(parse(t, `{resource.foo = "abc"}`)), // Resource-level only
		makeReq(parse(t, `{span.foo = "def"}`)),     // Span-level only
		makeReq(parse(t, `{.foo}`)),                 // Projection only
		makeReq(
			// Matches either condition
			parse(t, `{.foo = "baz"}`),
			parse(t, `{.`+LabelHTTPStatusCode+` > 100}`),
		),
		makeReq(
			// Same as above but reversed order
			parse(t, `{.`+LabelHTTPStatusCode+` > 100}`),
			parse(t, `{.foo = "baz"}`),
		),
		makeReq(
			// Same attribute with mixed types
			parse(t, `{.foo > 100}`),
			parse(t, `{.foo = "def"}`),
		),
		makeReq(
			// Multiple conditions on same well-known attribute, matches either
			parse(t, `{.`+LabelHTTPStatusCode+` = 500}`),
			parse(t, `{.`+LabelHTTPStatusCode+` > 500}`),
		),
		makeReq(
			// Mix of duration with other conditions
			parse(t, `{`+LabelName+` = "hello"}`),   // Match
			parse(t, `{`+LabelDuration+` < 100s }`), // No match
		),

		// Edge cases
		makeReq(parse(t, `{.name = "Bob"}`)),                             // Almost conflicts with intrinsic but still works
		makeReq(parse(t, `{resource.`+LabelServiceName+` = 123}`)),       // service.name doesn't match type of dedicated column
		makeReq(parse(t, `{.`+LabelServiceName+` = "spanservicename"}`)), // service.name present on span
		makeReq(parse(t, `{.`+LabelHTTPStatusCode+` = "500ouch"}`)),      // http.status_code doesn't match type of dedicated column
		makeReq(parse(t, `{.foo = "def"}`)),
	}

	for _, req := range searchesThatMatch {
		resp, err := b.Fetch(ctx, req, common.SearchOptions{})
		require.NoError(t, err, "search request:", req)

		spanSet, err := resp.Results.Next(ctx)
		require.NoError(t, err, "search request:", req)
		require.NotNil(t, spanSet, "search request:", req)
		require.Equal(t, wantTr.TraceID, spanSet.TraceID, "search request:", req)
		require.Equal(t, []byte("spanid"), spanSet.Spans[0].ID, "search request:", req)
	}

	searchesThatDontMatch := []traceql.FetchSpansRequest{
		// TODO - Should the below query return data or not?  It does match the resource
		// makeReq(parse(t, `{.foo = "abc"}`)),                           // This should not return results because the span has overridden this attribute to "def".
		makeReq(parse(t, `{.foo =~ "xyz.*"}`)),                        // Regex IN
		makeReq(parse(t, `{span.bool = true}`)),                       // Bool not match
		makeReq(parse(t, `{`+LabelDuration+` >  100s}`)),              // Intrinsic: duration
		makeReq(parse(t, `{`+LabelStatus+` = ok}`)),                   // Intrinsic: status
		makeReq(parse(t, `{`+LabelName+` = "nothello"}`)),             // Intrinsic: name
		makeReq(parse(t, `{.`+LabelServiceName+` = "notmyservice"}`)), // Well-known attribute: service.name not match
		makeReq(parse(t, `{.`+LabelHTTPStatusCode+` = 200}`)),         // Well-known attribute: http.status_code not match
		makeReq(parse(t, `{.`+LabelHTTPStatusCode+` > 600}`)),         // Well-known attribute: http.status_code not match
		makeReq(
			// Matches neither condition
			parse(t, `{.foo = "xyz"}`),
			parse(t, `{.`+LabelHTTPStatusCode+" = 1000}"),
		),
		{
			// Outside time range
			StartTimeUnixNanos: uint64(300 * time.Second),
			EndTimeUnixNanos:   uint64(400 * time.Second),
		},
		{
			// Matches some conditions but not all
			// Mix of span-level columns
			AllConditions: true,
			Conditions: []traceql.Condition{
				parse(t, `{span.foo = "baz"}`),                   // no match
				parse(t, `{span.`+LabelHTTPStatusCode+` > 100}`), // match
				parse(t, `{name = "hello"}`),                     // match
			},
		},
		{
			// Matches some conditions but not all
			// Only span generic attr lookups
			AllConditions: true,
			Conditions: []traceql.Condition{
				parse(t, `{span.foo = "baz"}`), // no match
				parse(t, `{span.bar = 123}`),   // match
			},
		},
		{
			// Matches some conditions but not all
			// Mix of span and resource columns
			AllConditions: true,
			Conditions: []traceql.Condition{
				parse(t, `{resource.cluster = "cluster"}`),     // match
				parse(t, `{resource.namespace = "namespace"}`), // match
				parse(t, `{span.foo = "baz"}`),                 // no match
			},
		},
		{
			// Matches some conditions but not all
			// Mix of resource columns
			AllConditions: true,
			Conditions: []traceql.Condition{
				parse(t, `{resource.cluster = "notcluster"}`),  // no match
				parse(t, `{resource.namespace = "namespace"}`), // match
				parse(t, `{resource.foo = "abc"}`),             // match
			},
		},
		{
			// Matches some conditions but not all
			// Only resource generic attr lookups
			AllConditions: true,
			Conditions: []traceql.Condition{
				parse(t, `{resource.foo = "abc"}`), // match
				parse(t, `{resource.bar = 123}`),   // no match
			},
		},
		{
			// Mix of duration with other conditions
			AllConditions: true,
			Conditions: []traceql.Condition{
				parse(t, `{`+LabelName+` = "nothello"}`), // No match
				parse(t, `{`+LabelDuration+` = 100s }`),  // Match
			},
		},
	}

	for _, req := range searchesThatDontMatch {
		resp, err := b.Fetch(ctx, req, common.SearchOptions{})
		require.NoError(t, err, "search request:", req)

		spanSet, err := resp.Results.Next(ctx)
		require.NoError(t, err, "search request:", req)
		require.Nil(t, spanSet, "search request:", req)
	}
}

func TestBackendBlockSearchTraceQLResults(t *testing.T) {
	wantTr := fullyPopulatedTestTrace(nil)
	b := makeBackendBlockWithTraces(t, []*Trace{wantTr})
	ctx := context.Background()

	// Helper functions to make requests

	makeSpansets := func(sets ...traceql.Spanset) []traceql.Spanset {
		return sets
	}

	makeSpanset := func(traceID []byte, rootSpanName, rootServiceName string, startTimeUnixNano, durationNanos uint64, spans ...traceql.Span) traceql.Spanset {
		return traceql.Spanset{
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
		expectedResults []traceql.Spanset
	}{
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
					wantTr.DurationNanos,
					traceql.Span{
						ID:                 wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].ID,
						StartTimeUnixNanos: wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].StartUnixNanos,
						EndtimeUnixNanos:   wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].EndUnixNanos,
						Attributes: map[traceql.Attribute]traceql.Static{
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
					wantTr.DurationNanos,
					traceql.Span{
						ID:                 wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].ID,
						StartTimeUnixNanos: wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].StartUnixNanos,
						EndtimeUnixNanos:   wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].EndUnixNanos,
						Attributes: map[traceql.Attribute]traceql.Static{
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
					wantTr.DurationNanos,
					traceql.Span{
						ID:                 wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].ID,
						StartTimeUnixNanos: wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].StartUnixNanos,
						EndtimeUnixNanos:   wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].EndUnixNanos,
						Attributes: map[traceql.Attribute]traceql.Static{
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
					wantTr.DurationNanos,
					traceql.Span{
						ID:                 wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].ID,
						StartTimeUnixNanos: wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].StartUnixNanos,
						EndtimeUnixNanos:   wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].EndUnixNanos,
						Attributes: map[traceql.Attribute]traceql.Static{
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
			// Empty request returns 1 spanset with all spans
			traceql.FetchSpansRequest{},
			makeSpansets(
				makeSpanset(
					wantTr.TraceID,
					wantTr.RootSpanName,
					wantTr.RootServiceName,
					wantTr.StartTimeUnixNano,
					wantTr.DurationNanos,
					traceql.Span{
						ID:                 wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].ID,
						StartTimeUnixNanos: wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].StartUnixNanos,
						EndtimeUnixNanos:   wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].EndUnixNanos,
						Attributes:         map[traceql.Attribute]traceql.Static{},
					},
					traceql.Span{
						ID:                 wantTr.ResourceSpans[1].ScopeSpans[0].Spans[0].ID,
						StartTimeUnixNanos: wantTr.ResourceSpans[1].ScopeSpans[0].Spans[0].StartUnixNanos,
						EndtimeUnixNanos:   wantTr.ResourceSpans[1].ScopeSpans[0].Spans[0].EndUnixNanos,
						Attributes:         map[traceql.Attribute]traceql.Static{},
					},
				),
			),
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
					wantTr.DurationNanos,
					traceql.Span{
						ID:                 wantTr.ResourceSpans[1].ScopeSpans[0].Spans[0].ID,
						StartTimeUnixNanos: wantTr.ResourceSpans[1].ScopeSpans[0].Spans[0].StartUnixNanos,
						EndtimeUnixNanos:   wantTr.ResourceSpans[1].ScopeSpans[0].Spans[0].EndUnixNanos,
						Attributes: map[traceql.Attribute]traceql.Static{
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
					wantTr.DurationNanos,
					traceql.Span{
						ID:                 wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].ID,
						StartTimeUnixNanos: wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].StartUnixNanos,
						EndtimeUnixNanos:   wantTr.ResourceSpans[0].ScopeSpans[0].Spans[0].EndUnixNanos,
						Attributes: map[traceql.Attribute]traceql.Static{
							traceql.NewIntrinsic(traceql.IntrinsicDuration): traceql.NewStaticDuration(100 * time.Second),
						},
					},
					traceql.Span{
						ID:                 wantTr.ResourceSpans[1].ScopeSpans[0].Spans[0].ID,
						StartTimeUnixNanos: wantTr.ResourceSpans[1].ScopeSpans[0].Spans[0].StartUnixNanos,
						EndtimeUnixNanos:   wantTr.ResourceSpans[1].ScopeSpans[0].Spans[0].EndUnixNanos,
						Attributes: map[traceql.Attribute]traceql.Static{
							traceql.NewIntrinsic(traceql.IntrinsicDuration): traceql.NewStaticDuration(0 * time.Second),
						},
					},
				),
			),
		},
	}

	for _, tc := range testCases {
		req := tc.req
		resp, err := b.Fetch(ctx, req, common.SearchOptions{})
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

func parse(t *testing.T, q string) traceql.Condition {

	cond, err := traceql.ExtractCondition(q)
	require.NoError(t, err, "query:", q)

	return cond
}

func fullyPopulatedTestTrace(id common.ID) *Trace {
	// Helper functions to make pointers
	strPtr := func(s string) *string { return &s }
	intPtr := func(i int64) *int64 { return &i }
	fltPtr := func(f float64) *float64 { return &f }
	boolPtr := func(b bool) *bool { return &b }

	return &Trace{
		TraceID:           test.ValidTraceID(id),
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
				ScopeSpans: []ScopeSpan{
					{
						Spans: []Span{
							{
								ID:             []byte("spanid"),
								Name:           "hello",
								StartUnixNanos: uint64(100 * time.Second),
								EndUnixNanos:   uint64(200 * time.Second),
								// DurationNanos:  uint64(100 * time.Second),
								HttpMethod:             strPtr("get"),
								HttpUrl:                strPtr("url/hello/world"),
								HttpStatusCode:         intPtr(500),
								ParentSpanID:           []byte{},
								StatusCode:             int(v1.Status_STATUS_CODE_ERROR),
								StatusMessage:          v1.Status_STATUS_CODE_ERROR.String(),
								TraceState:             "tracestate",
								Kind:                   int(v1.Span_SPAN_KIND_CLIENT),
								DroppedAttributesCount: 42,
								DroppedEventsCount:     43,
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
								Events: []Event{
									{TimeUnixNano: 1, Name: "e1", Attrs: []EventAttribute{
										{Key: "foo", Value: []byte("fake proto encoded data. i hope this never matters")},
										{Key: "bar", Value: []byte("fake proto encoded data. i hope this never matters")},
									}},
									{TimeUnixNano: 2, Name: "e2", Attrs: []EventAttribute{}},
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
				ScopeSpans: []ScopeSpan{
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
