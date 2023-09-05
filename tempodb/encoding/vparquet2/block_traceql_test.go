package vparquet2

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"path"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/traceqlmetrics"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

func TestOne(t *testing.T) {
	wantTr := fullyPopulatedTestTrace(nil)
	b := makeBackendBlockWithTraces(t, []*Trace{wantTr})
	ctx := context.Background()
	req := traceql.MustExtractFetchSpansRequestWithMetadata(`{ span.foo = "bar" || duration > 1s }`)

	req.StartTimeUnixNanos = uint64(1000 * time.Second)
	req.EndTimeUnixNanos = uint64(1001 * time.Second)

	resp, err := b.Fetch(ctx, req, common.DefaultSearchOptions())
	require.NoError(t, err, "search request:", req)

	spanSet, err := resp.Results.Next(ctx)
	require.NoError(t, err, "search request:", req)

	fmt.Println("-----------")
	fmt.Println(resp.Results.(*spansetIterator).iter)
	fmt.Println("-----------")
	fmt.Println(spanSet)
}

func TestBackendBlockSearchTraceQL(t *testing.T) {
	numTraces := 250
	traces := make([]*Trace, 0, numTraces)
	wantTraceIdx := rand.Intn(numTraces)
	wantTraceID := test.ValidTraceID(nil)
	for i := 0; i < numTraces; i++ {
		if i == wantTraceIdx {
			traces = append(traces, fullyPopulatedTestTrace(wantTraceID))
			continue
		}

		id := test.ValidTraceID(nil)
		tr := traceToParquet(id, test.MakeTrace(1, id), nil)
		traces = append(traces, tr)
	}

	b := makeBackendBlockWithTraces(t, traces)
	ctx := context.Background()

	searchesThatMatch := []traceql.FetchSpansRequest{
		{}, // Empty request
		{
			// Time range inside trace
			StartTimeUnixNanos: uint64(1100 * time.Second),
			EndTimeUnixNanos:   uint64(1200 * time.Second),
		},
		{
			// Time range overlap start
			StartTimeUnixNanos: uint64(900 * time.Second),
			EndTimeUnixNanos:   uint64(1100 * time.Second),
		},
		{
			// Time range overlap end
			StartTimeUnixNanos: uint64(1900 * time.Second),
			EndTimeUnixNanos:   uint64(2100 * time.Second),
		},
		// Intrinsics
		traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelName + ` = "hello"}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelDuration + ` =  100s}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelDuration + ` >  99s}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelDuration + ` >= 100s}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelDuration + ` <  101s}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelDuration + ` <= 100s}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelDuration + ` <= 100s}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelStatus + ` = error}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelStatus + ` = 2}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{` + "statusMessage" + ` = "STATUS_CODE_ERROR"}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelKind + ` = client }`),
		// Resource well-known attributes
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelServiceName + ` = "spanservicename"}`), // Overridden at span
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelCluster + ` = "cluster"}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelNamespace + ` = "namespace"}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelPod + ` = "pod"}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelContainer + ` = "container"}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelK8sNamespaceName + ` = "k8snamespace"}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelK8sClusterName + ` = "k8scluster"}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelK8sPodName + ` = "k8spod"}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelK8sContainerName + ` = "k8scontainer"}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelServiceName + ` = "myservice"}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelCluster + ` = "cluster"}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelNamespace + ` = "namespace"}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelPod + ` = "pod"}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelContainer + ` = "container"}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelK8sNamespaceName + ` = "k8snamespace"}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelK8sClusterName + ` = "k8scluster"}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelK8sPodName + ` = "k8spod"}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelK8sContainerName + ` = "k8scontainer"}`),
		// Comparing strings

		traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelServiceName + ` > "myservic"}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelServiceName + ` >= "myservic"}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelServiceName + ` < "myservice1"}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelServiceName + ` <= "myservice1"}`),
		// Span well-known attributes
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelHTTPStatusCode + ` = 500}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelHTTPMethod + ` = "get"}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelHTTPUrl + ` = "url/hello/world"}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{span.` + LabelHTTPStatusCode + ` = 500}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{span.` + LabelHTTPMethod + ` = "get"}`),
		traceql.MustExtractFetchSpansRequestWithMetadata(`{span.` + LabelHTTPUrl + ` = "url/hello/world"}`),
		// Basic data types and operations
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.float = 456.78}`),      // Float ==
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.float != 456.79}`),     // Float !=
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.float > 456.7}`),       // Float >
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.float >= 456.78}`),     // Float >=
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.float < 456.781}`),     // Float <
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.bool = false}`),        // Bool ==
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.bool != true}`),        // Bool !=
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.bar = 123}`),           // Int ==
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.bar != 124}`),          // Int !=
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.bar > 122}`),           // Int >
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.bar >= 123}`),          // Int >=
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.bar < 124}`),           // Int <
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.bar <= 123}`),          // Int <=
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.foo = "def"}`),         // String ==
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.foo != "deg"}`),        // String !=
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.foo =~ "d.*"}`),        // String Regex
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.foo !~ "x.*"}`),        // String Not Regex
		traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.foo = "abc"}`), // Resource-level only
		traceql.MustExtractFetchSpansRequestWithMetadata(`{span.foo = "def"}`),     // Span-level only
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.foo}`),                 // Projection only

		// existence
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.foo != nil}`), // Exist

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
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.name = "Bob"}`),                                 // Almost conflicts with intrinsic but still works
		traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelServiceName + ` = 123}`),       // service.name doesn't match type of dedicated column
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelServiceName + ` = "spanservicename"}`), // service.name present on span
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelHTTPStatusCode + ` = "500ouch"}`),      // http.status_code doesn't match type of dedicated column
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.foo = "def"}`),
		{
			// Range at unscoped
			AllConditions: true,
			Conditions: []traceql.Condition{
				parse(t, `{.`+LabelHTTPStatusCode+` >= 500}`),
				parse(t, `{.`+LabelHTTPStatusCode+` <= 600}`),
			},
		},
		{
			// Range at span scope
			AllConditions: true,
			Conditions: []traceql.Condition{
				parse(t, `{span.`+LabelHTTPStatusCode+` >= 500}`),
				parse(t, `{span.`+LabelHTTPStatusCode+` <= 600}`),
			},
		},
		{
			// Range at resource scope
			AllConditions: true,
			Conditions: []traceql.Condition{
				parse(t, `{resource.`+LabelServiceName+` >= 122}`),
				parse(t, `{resource.`+LabelServiceName+` <= 124}`),
			},
		},
	}

	for _, req := range searchesThatMatch {
		if req.SecondPass == nil {
			req.SecondPass = func(s *traceql.Spanset) ([]*traceql.Spanset, error) { return []*traceql.Spanset{s}, nil }
			req.SecondPassConditions = traceql.SearchMetaConditions()
		}

		resp, err := b.Fetch(ctx, req, common.DefaultSearchOptions())
		require.NoError(t, err, "search request:%v", req)

		found := false
		for {
			spanSet, err := resp.Results.Next(ctx)
			require.NoError(t, err, "search request:%v", req)
			if spanSet == nil {
				break
			}
			found = bytes.Equal(spanSet.TraceID, wantTraceID)
			if found {
				break
			}
		}
		require.True(t, found, "search request:%v", req)
	}

	searchesThatDontMatch := []traceql.FetchSpansRequest{
		// TODO - Should the below query return data or not?  It does match the resource
		// makeReq(parse(t, `{.foo = "abc"}`)),                           // This should not return results because the span has overridden this attribute to "def".
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.foo =~ "xyz.*"}`),                                     // Regex IN
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.foo !~ ".*"}`),                                        // String Not Regex
		traceql.MustExtractFetchSpansRequestWithMetadata(`{span.bool = true}`),                                    // Bool not match
		traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelDuration + ` >  100s}`),                       // Intrinsic: duration
		traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelStatus + ` = ok}`),                            // Intrinsic: status
		traceql.MustExtractFetchSpansRequestWithMetadata(`{` + "statusMessage" + ` = "abc"}`),                     // Intrinsic: statusMessage
		traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelName + ` = "nothello"}`),                      // Intrinsic: name
		traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelKind + ` = producer }`),                       // Intrinsic: kind
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelServiceName + ` = "notmyservice"}`),          // Well-known attribute: service.name not match
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelHTTPStatusCode + ` = 200}`),                  // Well-known attribute: http.status_code not match
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelHTTPStatusCode + ` > 600}`),                  // Well-known attribute: http.status_code not match
		traceql.MustExtractFetchSpansRequestWithMetadata(`{.foo = "xyz" || .` + LabelHTTPStatusCode + " = 1000}"), // Matches neither condition
		{
			// Time range after trace
			StartTimeUnixNanos: uint64(3000 * time.Second),
			EndTimeUnixNanos:   uint64(4000 * time.Second),
		},
		{
			// Time range before trace
			StartTimeUnixNanos: uint64(600 * time.Second),
			EndTimeUnixNanos:   uint64(700 * time.Second),
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
		if req.SecondPass == nil {
			req.SecondPass = func(s *traceql.Spanset) ([]*traceql.Spanset, error) { return []*traceql.Spanset{s}, nil }
			req.SecondPassConditions = traceql.SearchMetaConditions()
		}

		resp, err := b.Fetch(ctx, req, common.DefaultSearchOptions())
		require.NoError(t, err, "search request:", req)

		for {
			spanSet, err := resp.Results.Next(ctx)
			require.NoError(t, err, "search request:", req)
			if spanSet == nil {
				break
			}
			require.NotEqual(t, wantTraceID, spanSet.TraceID, "search request:", req)
		}
	}
}

func makeReq(conditions ...traceql.Condition) traceql.FetchSpansRequest {
	return traceql.FetchSpansRequest{
		Conditions: conditions,
		SecondPass: func(s *traceql.Spanset) ([]*traceql.Spanset, error) {
			return []*traceql.Spanset{s}, nil
		},
		SecondPassConditions: traceql.SearchMetaConditions(),
	}
}

func parse(t *testing.T, q string) traceql.Condition {
	req, err := traceql.ExtractFetchSpansRequest(q)
	require.NoError(t, err, "query:", q)

	return req.Conditions[0]
}

func fullyPopulatedTestTrace(id common.ID) *Trace {
	// Helper functions to make pointers
	strPtr := func(s string) *string { return &s }
	intPtr := func(i int64) *int64 { return &i }
	fltPtr := func(f float64) *float64 { return &f }
	boolPtr := func(b bool) *bool { return &b }

	links := tempopb.LinkSlice{
		Links: []*v1.Span_Link{
			{
				TraceId:                []byte{0x01},
				SpanId:                 []byte{0x02},
				TraceState:             "state",
				DroppedAttributesCount: 3,
				Attributes: []*v1_common.KeyValue{
					{
						Key:   "key",
						Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "value"}},
					},
				},
			},
		},
	}
	linkBytes := make([]byte, links.Size())
	_, err := links.MarshalTo(linkBytes)
	if err != nil {
		panic("failed to marshal links")
	}

	return &Trace{
		TraceID:           test.ValidTraceID(id),
		StartTimeUnixNano: uint64(1000 * time.Second),
		EndTimeUnixNano:   uint64(2000 * time.Second),
		DurationNano:      uint64((100 * time.Millisecond).Nanoseconds()),
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
				ScopeSpans: []ScopeSpans{
					{
						Spans: []Span{
							{
								SpanID:                 []byte("spanid"),
								Name:                   "hello",
								StartTimeUnixNano:      uint64(100 * time.Second),
								DurationNano:           uint64(100 * time.Second),
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
								Links: linkBytes,
							},
						},
					},
				},
			},
			{
				Resource: Resource{
					ServiceName: "service2",
				},
				ScopeSpans: []ScopeSpans{
					{
						Spans: []Span{
							{
								SpanID: []byte("spanid2"),
								Name:   "world",
							},
						},
					},
				},
			},
		},
	}
}

func BenchmarkBackendBlockTraceQL(b *testing.B) {
	testCases := []struct {
		name  string
		query string
	}{
		// span
		{"spanAttNameNoMatch", "{ span.foo = `bar` }"},
		{"spanAttValNoMatch", "{ span.bloom = `bar` }"},
		{"spanAttValMatch", "{ span.bloom > 0 }"},
		{"spanAttIntrinsicNoMatch", "{ name = `asdfasdf` }"},
		{"spanAttIntrinsicMatch", "{ name = `gcs.ReadRange` }"},
		{"spanAttIntrinsicRegexNoMatch", "{ name =~ `asdfasdf` }"},
		{"spanAttIntrinsicRegexMatch", "{ name =~ `gcs.ReadRange` }"},

		// resource
		{"resourceAttNameNoMatch", "{ resource.foo = `bar` }"},
		{"resourceAttValNoMatch", "{ resource.module.path = `bar` }"},
		{"resourceAttValMatch", "{ resource.os.type = `linux` }"},
		{"resourceAttIntrinsicNoMatch", "{ resource.service.name = `a` }"},
		{"resourceAttIntrinsicMatch", "{ resource.service.name = `tempo-query-frontend` }"},

		// mixed
		{"mixedNameNoMatch", "{ .foo = `bar` }"},
		{"mixedValNoMatch", "{ .bloom = `bar` }"},
		{"mixedValMixedMatchAnd", "{ resource.foo = `bar` && name = `gcs.ReadRange` }"},
		{"mixedValMixedMatchOr", "{ resource.foo = `bar` || name = `gcs.ReadRange` }"},
		{"mixedValBothMatch", "{ resource.service.name = `query-frontend` && name = `gcs.ReadRange` }"},
	}

	ctx := context.TODO()
	tenantID := "1"
	blockID := uuid.MustParse("000d37d0-1e66-4f4e-bbd4-f85c1deb6e5e")

	r, _, _, err := local.New(&local.Config{
		Path: path.Join("/home/joe/testblock/"),
	})
	require.NoError(b, err)

	rr := backend.NewReader(r)
	meta, err := rr.BlockMeta(ctx, blockID, tenantID)
	require.NoError(b, err)

	opts := common.DefaultSearchOptions()
	opts.StartPage = 10
	opts.TotalPages = 10

	block := newBackendBlock(meta, rr)
	_, _, err = block.openForSearch(ctx, opts)
	require.NoError(b, err)

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			bytesRead := 0

			for i := 0; i < b.N; i++ {
				e := traceql.NewEngine()

				resp, err := e.ExecuteSearch(ctx, &tempopb.SearchRequest{Query: tc.query}, traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
					return block.Fetch(ctx, req, opts)
				}))
				require.NoError(b, err)
				require.NotNil(b, resp)

				// Read first 20 results (if any)
				bytesRead += int(resp.Metrics.InspectedBytes)
			}
			b.SetBytes(int64(bytesRead) / int64(b.N))
			b.ReportMetric(float64(bytesRead)/float64(b.N)/1000.0/1000.0, "MB_io/op")
		})
	}
}

// BenchmarkBackendBlockGetMetrics This doesn't really belong here but I can't think of
// a better place that has access to all of the packages, especially the backend.
func BenchmarkBackendBlockGetMetrics(b *testing.B) {
	testCases := []struct {
		query   string
		groupby string
	}{
		//{"{ resource.service.name = `gme-ingester` }", "resource.cluster"},
		{"{}", "name"},
	}

	ctx := context.TODO()
	tenantID := "1"
	blockID := uuid.MustParse("2968a567-5873-4e4c-b3cb-21c106c6714b")

	r, _, _, err := local.New(&local.Config{
		Path: path.Join("/Users/marty/src/tmp/"),
	})
	require.NoError(b, err)

	rr := backend.NewReader(r)
	meta, err := rr.BlockMeta(ctx, blockID, tenantID)
	require.NoError(b, err)
	require.Equal(b, VersionString, meta.Version)

	opts := common.DefaultSearchOptions()
	opts.StartPage = 10
	opts.TotalPages = 10

	block := newBackendBlock(meta, rr)
	_, _, err = block.openForSearch(ctx, opts)
	require.NoError(b, err)

	for _, tc := range testCases {
		b.Run(tc.query+"/"+tc.groupby, func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				f := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
					return block.Fetch(ctx, req, opts)
				})

				r, err := traceqlmetrics.GetMetrics(ctx, tc.query, tc.groupby, 0, 0, 0, f)

				require.NoError(b, err)
				require.NotNil(b, r)
			}
		})
	}
}
