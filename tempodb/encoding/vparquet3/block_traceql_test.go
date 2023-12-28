package vparquet3

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
	q := `{ traceDuration > 1s }`
	req := traceql.MustExtractFetchSpansRequestWithMetadata(q)

	req.StartTimeUnixNanos = uint64(1000 * time.Second)
	req.EndTimeUnixNanos = uint64(1001 * time.Second)

	resp, err := b.Fetch(ctx, req, common.DefaultSearchOptions())
	require.NoError(t, err, "search request:", req)

	spanSet, err := resp.Results.Next(ctx)
	require.NoError(t, err, "search request:", req)

	fmt.Println(q)
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
		tr, _ := traceToParquet(&backend.BlockMeta{}, id, test.MakeTrace(1, id), nil)
		traces = append(traces, tr)
	}

	b := makeBackendBlockWithTraces(t, traces)
	ctx := context.Background()

	searchesThatMatch := []struct {
		name string
		req  traceql.FetchSpansRequest
	}{
		{"empty request", traceql.FetchSpansRequest{}},
		{
			"Time range inside trace",
			traceql.FetchSpansRequest{
				StartTimeUnixNanos: uint64(1100 * time.Second),
				EndTimeUnixNanos:   uint64(1200 * time.Second),
			},
		},
		{
			"Time range overlap start",
			traceql.FetchSpansRequest{
				StartTimeUnixNanos: uint64(900 * time.Second),
				EndTimeUnixNanos:   uint64(1100 * time.Second),
			},
		},
		{
			"Time range overlap end",
			traceql.FetchSpansRequest{
				StartTimeUnixNanos: uint64(1900 * time.Second),
				EndTimeUnixNanos:   uint64(2100 * time.Second),
			},
		},
		// Intrinsics
		{"Intrinsic: name", traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelName + ` = "hello"}`)},
		{"Intrinsic: duration = 100s", traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelDuration + ` = 100s}`)},
		{"Intrinsic: duration > 99s", traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelDuration + ` > 99s}`)},
		{"Intrinsic: duration >= 100s", traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelDuration + ` >= 100s}`)},
		{"Intrinsic: duration < 101s", traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelDuration + ` < 101s}`)},
		{"Intrinsic: duration <= 100s", traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelDuration + ` <= 100s}`)},
		{"Intrinsic: status = error", traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelStatus + ` = error}`)},
		{"Intrinsic: status = 2", traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelStatus + ` = 2}`)},
		{"Intrinsic: statusMessage = STATUS_CODE_ERROR", traceql.MustExtractFetchSpansRequestWithMetadata(`{` + "statusMessage" + ` = "STATUS_CODE_ERROR"}`)},
		{"Intrinsic: kind = client", traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelKind + ` = client }`)},
		// Resource well-known attributes
		{".service.name", traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelServiceName + ` = "spanservicename"}`)}, // Overridden at span},
		{".cluster", traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelCluster + ` = "cluster"}`)},
		{".namespace", traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelNamespace + ` = "namespace"}`)},
		{".pod", traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelPod + ` = "pod"}`)},
		{".container", traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelContainer + ` = "container"}`)},
		{".k8s.namespace.name", traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelK8sNamespaceName + ` = "k8snamespace"}`)},
		{".k8s.cluster.name", traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelK8sClusterName + ` = "k8scluster"}`)},
		{".k8s.pod.name", traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelK8sPodName + ` = "k8spod"}`)},
		{".k8s.container.name", traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelK8sContainerName + ` = "k8scontainer"}`)},
		{"resource.service.name", traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelServiceName + ` = "myservice"}`)},
		{"resource.cluster", traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelCluster + ` = "cluster"}`)},
		{"resource.namespace", traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelNamespace + ` = "namespace"}`)},
		{"resource.pod", traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelPod + ` = "pod"}`)},
		{"resource.container", traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelContainer + ` = "container"}`)},
		{"resource.k8s.namespace.name", traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelK8sNamespaceName + ` = "k8snamespace"}`)},
		{"resource.k8s.cluster.name", traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelK8sClusterName + ` = "k8scluster"}`)},
		{"resource.k8s.pod.name", traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelK8sPodName + ` = "k8spod"}`)},
		{"resource.k8s.container.name", traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelK8sContainerName + ` = "k8scontainer"}`)},
		// Resource dedicated attributes
		{"resource.dedicated.resource.3", traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.dedicated.resource.3 = "dedicated-resource-attr-value-3"}`)},
		{"resource.dedicated.resource.5", traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.dedicated.resource.5 = "dedicated-resource-attr-value-5"}`)},
		// Comparing strings
		{"resource.service.name > myservice", traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelServiceName + ` > "myservic"}`)},
		{"resource.service.name >= myservice", traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelServiceName + ` >= "myservic"}`)},
		{"resource.service.name < myservice1", traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelServiceName + ` < "myservice1"}`)},
		{"resource.service.name <= myservice1", traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelServiceName + ` <= "myservice1"}`)},
		// Span well-known attributes
		{".http.status_code", traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelHTTPStatusCode + ` = 500}`)},
		{".http.method", traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelHTTPMethod + ` = "get"}`)},
		{".http.url", traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelHTTPUrl + ` = "url/hello/world"}`)},
		{"span.http.status_code", traceql.MustExtractFetchSpansRequestWithMetadata(`{span.` + LabelHTTPStatusCode + ` = 500}`)},
		{"span.http.method", traceql.MustExtractFetchSpansRequestWithMetadata(`{span.` + LabelHTTPMethod + ` = "get"}`)},
		{"span.http.url", traceql.MustExtractFetchSpansRequestWithMetadata(`{span.` + LabelHTTPUrl + ` = "url/hello/world"}`)},
		// Span dedicated attributes
		{"span.dedicated.span.2", traceql.MustExtractFetchSpansRequestWithMetadata(`{span.dedicated.span.2 = "dedicated-span-attr-value-2"}`)},
		{"span.dedicated.span.4", traceql.MustExtractFetchSpansRequestWithMetadata(`{span.dedicated.span.4 = "dedicated-span-attr-value-4"}`)},
		// Basic data types and operations
		{".float = 456.78", traceql.MustExtractFetchSpansRequestWithMetadata(`{.float = 456.78}`)},             // Float ==
		{".float != 456.79", traceql.MustExtractFetchSpansRequestWithMetadata(`{.float != 456.79}`)},           // Float !=
		{".float > 456.7", traceql.MustExtractFetchSpansRequestWithMetadata(`{.float > 456.7}`)},               // Float >
		{".float >= 456.78", traceql.MustExtractFetchSpansRequestWithMetadata(`{.float >= 456.78}`)},           // Float >=
		{".float < 456.781", traceql.MustExtractFetchSpansRequestWithMetadata(`{.float < 456.781}`)},           // Float <
		{".bool = false", traceql.MustExtractFetchSpansRequestWithMetadata(`{.bool = false}`)},                 // Bool ==
		{".bool != true", traceql.MustExtractFetchSpansRequestWithMetadata(`{.bool != true}`)},                 // Bool !=
		{".bar = 123", traceql.MustExtractFetchSpansRequestWithMetadata(`{.bar = 123}`)},                       // Int ==
		{".bar != 124", traceql.MustExtractFetchSpansRequestWithMetadata(`{.bar != 124}`)},                     // Int !=
		{".bar > 122", traceql.MustExtractFetchSpansRequestWithMetadata(`{.bar > 122}`)},                       // Int >
		{".bar >= 123", traceql.MustExtractFetchSpansRequestWithMetadata(`{.bar >= 123}`)},                     // Int >=
		{".bar < 124", traceql.MustExtractFetchSpansRequestWithMetadata(`{.bar < 124}`)},                       // Int <
		{".bar <= 123", traceql.MustExtractFetchSpansRequestWithMetadata(`{.bar <= 123}`)},                     // Int <=
		{".foo = \"def\"", traceql.MustExtractFetchSpansRequestWithMetadata(`{.foo = "def"}`)},                 // String ==
		{".foo != \"deg\"", traceql.MustExtractFetchSpansRequestWithMetadata(`{.foo != "deg"}`)},               // String !=
		{".foo =~ \"d.*\"", traceql.MustExtractFetchSpansRequestWithMetadata(`{.foo =~ "d.*"}`)},               // String Regex
		{".foo !~ \"x.*\"", traceql.MustExtractFetchSpansRequestWithMetadata(`{.foo !~ "x.*"}`)},               // String Not Regex
		{"resource.foo = \"abc\"", traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.foo = "abc"}`)}, // Resource-level only
		{"span.foo = \"def\"", traceql.MustExtractFetchSpansRequestWithMetadata(`{span.foo = "def"}`)},         // Span-level only
		{".foo", traceql.MustExtractFetchSpansRequestWithMetadata(`{.foo}`)},                                   // Projection only
		{"Matches either condition", makeReq(
			parse(t, `{.foo = "baz"}`),
			parse(t, `{.`+LabelHTTPStatusCode+` > 100}`),
		)},
		{"Same as above but reversed order", makeReq(
			parse(t, `{.`+LabelHTTPStatusCode+` > 100}`),
			parse(t, `{.foo = "baz"}`),
		)},
		{"Same attribute with mixed types", makeReq(
			parse(t, `{.foo > 100}`),
			parse(t, `{.foo = "def"}`),
		)},
		{"Multiple conditions on same well-known attribute, matches either", makeReq(
			//
			parse(t, `{.`+LabelHTTPStatusCode+` = 500}`),
			parse(t, `{.`+LabelHTTPStatusCode+` > 500}`),
		)},
		{
			"Mix of duration with other conditions", makeReq(
				//
				parse(t, `{`+LabelName+` = "hello"}`),   // Match
				parse(t, `{`+LabelDuration+` < 100s }`), // No match
			),
		},
		// Edge cases
		{"Almost conflicts with intrinsic but still works", traceql.MustExtractFetchSpansRequestWithMetadata(`{.name = "Bob"}`)},
		{"service.name doesn't match type of dedicated column", traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelServiceName + ` = 123}`)},
		{"service.name present on span", traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelServiceName + ` = "spanservicename"}`)},
		{"http.status_code doesn't match type of dedicated column", traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelHTTPStatusCode + ` = "500ouch"}`)},
		{`.foo = "def"`, traceql.MustExtractFetchSpansRequestWithMetadata(`{.foo = "def"}`)},
		{
			name: "Range at unscoped",
			req: traceql.FetchSpansRequest{
				AllConditions: true,
				Conditions: []traceql.Condition{
					parse(t, `{.`+LabelHTTPStatusCode+` >= 500}`),
					parse(t, `{.`+LabelHTTPStatusCode+` <= 600}`),
				},
			},
		},
		{
			name: "Range at span scope",
			req: traceql.FetchSpansRequest{
				AllConditions: true,
				Conditions: []traceql.Condition{
					parse(t, `{span.`+LabelHTTPStatusCode+` >= 500}`),
					parse(t, `{span.`+LabelHTTPStatusCode+` <= 600}`),
				},
			},
		},
		{
			name: "Range at resource scope",
			req: traceql.FetchSpansRequest{
				AllConditions: true,
				Conditions: []traceql.Condition{
					parse(t, `{resource.`+LabelServiceName+` >= 122}`),
					parse(t, `{resource.`+LabelServiceName+` <= 124}`),
				},
			},
		},
	}

	for _, tc := range searchesThatMatch {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.req
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
		})
	}

	searchesThatDontMatch := []struct {
		name string
		req  traceql.FetchSpansRequest
	}{
		// TODO - Should the below query return data or not?  It does match the resource
		// makeReq(parse(t, `{.foo = "abc"}`)),                           // This should not return results because the span has overridden this attribute to "def".
		{"Regex IN", traceql.MustExtractFetchSpansRequestWithMetadata(`{.foo =~ "xyz.*"}`)},
		{"String Not Regex", traceql.MustExtractFetchSpansRequestWithMetadata(`{.foo !~ ".*"}`)},
		{"Bool not match", traceql.MustExtractFetchSpansRequestWithMetadata(`{span.bool = true && name = "hello"}`)}, // name = "hello" only matches the first span
		{"Intrinsic: duration", traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelDuration + ` >  1000s}`)},
		{"Intrinsic: status", traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelStatus + ` = unset}`)},
		{"Intrinsic: statusMessage", traceql.MustExtractFetchSpansRequestWithMetadata(`{` + "statusMessage" + ` = "abc"}`)},
		{"Intrinsic: name", traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelName + ` = "nothello"}`)},
		{"Intrinsic: kind", traceql.MustExtractFetchSpansRequestWithMetadata(`{` + LabelKind + ` = producer }`)},
		{"Well-known attribute: service.name not match", traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelServiceName + ` = "notmyservice"}`)},
		{"Well-known attribute: http.status_code not match", traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelHTTPStatusCode + ` = 200}`)},
		{"Well-known attribute: http.status_code not match", traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelHTTPStatusCode + ` > 600}`)},
		{"Matches neither condition", traceql.MustExtractFetchSpansRequestWithMetadata(`{.foo = "xyz" || .` + LabelHTTPStatusCode + " = 1000}")},
		{"Resource dedicated attributes does not match", traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.dedicated.resource.3 = "dedicated-resource-attr-value-4"}`)},
		{"Resource dedicated attributes does not match", traceql.MustExtractFetchSpansRequestWithMetadata(`{span.dedicated.span.2 = "dedicated-span-attr-value-5"}`)},
		{
			name: "Time range after trace",
			req: traceql.FetchSpansRequest{
				StartTimeUnixNanos: uint64(20000 * time.Second),
				EndTimeUnixNanos:   uint64(30000 * time.Second),
			},
		},
		{
			name: "Time range before trace",
			req: traceql.FetchSpansRequest{
				StartTimeUnixNanos: uint64(600 * time.Second),
				EndTimeUnixNanos:   uint64(700 * time.Second),
			},
		},
		{
			name: "Matches some conditions but not all. Mix of span-level columns",
			req: traceql.FetchSpansRequest{
				AllConditions: true,
				Conditions: []traceql.Condition{
					parse(t, `{span.foo = "baz"}`),                   // no match
					parse(t, `{span.`+LabelHTTPStatusCode+` > 100}`), // match
					parse(t, `{name = "hello"}`),                     // match
				},
			},
		},
		{
			name: "Matches some conditions but not all. Only span generic attr lookups",
			req: traceql.FetchSpansRequest{
				AllConditions: true,
				Conditions: []traceql.Condition{
					parse(t, `{span.foo = "baz"}`), // no match
					parse(t, `{span.bar = 123}`),   // match
				},
			},
		},
		{
			name: "Matches some conditions but not all. Mix of span and resource columns",
			req: traceql.FetchSpansRequest{
				AllConditions: true,
				Conditions: []traceql.Condition{
					parse(t, `{resource.cluster = "cluster"}`),     // match
					parse(t, `{resource.namespace = "namespace"}`), // match
					parse(t, `{span.foo = "baz"}`),                 // no match
				},
			},
		},
		{
			name: "Matches some conditions but not all. Mix of resource columns",
			req: traceql.FetchSpansRequest{
				AllConditions: true,
				Conditions: []traceql.Condition{
					parse(t, `{resource.cluster = "notcluster"}`),  // no match
					parse(t, `{resource.namespace = "namespace"}`), // match
					parse(t, `{resource.foo = "abc"}`),             // match
				},
			},
		},
		{
			name: "Matches some conditions but not all. Only resource generic attr lookups",
			req: traceql.FetchSpansRequest{
				AllConditions: true,
				Conditions: []traceql.Condition{
					parse(t, `{resource.foo = "abc"}`), // match
					parse(t, `{resource.bar = 123}`),   // no match
				},
			},
		},
		{
			name: "Mix of duration with other conditions",
			req: traceql.FetchSpansRequest{
				AllConditions: true,
				Conditions: []traceql.Condition{
					parse(t, `{`+LabelName+` = "nothello"}`), // No match
					parse(t, `{`+LabelDuration+` = 100s }`),  // Match
				},
			},
		},
	}

	for _, tc := range searchesThatDontMatch {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.req
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
		})
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
	links := []Link{
		{
			TraceID:                []byte{0x01},
			SpanID:                 []byte{0x02},
			TraceState:             "state",
			DroppedAttributesCount: 3,
			Attrs: []Attribute{
				attr("link-attr-key-1", "link-value-1"),
			},
		},
	}

	mixedArrayAttrValue := "{\"arrayValue\":{\"values\":[{\"stringValue\":\"value-one\"},{\"intValue\":\"100\"}]}}"
	kvListValue := "{\"kvlistValue\":{\"values\":[{\"key\":\"key-one\",\"value\":{\"stringValue\":\"value-one\"}},{\"key\":\"key-two\",\"value\":{\"stringValue\":\"value-two\"}}]}}"

	return &Trace{
		TraceID:           test.ValidTraceID(id),
		StartTimeUnixNano: uint64(1000 * time.Second),
		EndTimeUnixNano:   uint64(2000 * time.Second),
		DurationNano:      uint64((100 * time.Millisecond).Nanoseconds()),
		RootServiceName:   "RootService",
		RootSpanName:      "RootSpan",
		ServiceStats: map[string]ServiceStats{
			"myservice": {
				SpanCount:  1,
				ErrorCount: 0,
			},
			"service2": {
				SpanCount:  1,
				ErrorCount: 0,
			},
		},
		ResourceSpans: []ResourceSpans{
			{
				Resource: Resource{
					ServiceName:      "myservice",
					Cluster:          ptr("cluster"),
					Namespace:        ptr("namespace"),
					Pod:              ptr("pod"),
					Container:        ptr("container"),
					K8sClusterName:   ptr("k8scluster"),
					K8sNamespaceName: ptr("k8snamespace"),
					K8sPodName:       ptr("k8spod"),
					K8sContainerName: ptr("k8scontainer"),
					Attrs: []Attribute{
						attr("foo", "abc"),
						attr("str-array", []string{"value-one", "value-two"}),
						attr(LabelServiceName, 123), // Different type than dedicated column
						// Unsupported attributes
						{Key: "unsupported-mixed-array", ValueDropped: mixedArrayAttrValue, ValueType: attrTypeNotSupported},
						{Key: "unsupported-kv-list", ValueDropped: kvListValue, ValueType: attrTypeNotSupported},
					},
					DroppedAttributesCount: 22,
					DedicatedAttributes: DedicatedAttributes{
						String01: ptr("dedicated-resource-attr-value-1"),
						String02: ptr("dedicated-resource-attr-value-2"),
						String03: ptr("dedicated-resource-attr-value-3"),
						String04: ptr("dedicated-resource-attr-value-4"),
						String05: ptr("dedicated-resource-attr-value-5"),
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
								HttpMethod:             ptr("get"),
								HttpUrl:                ptr("url/hello/world"),
								HttpStatusCode:         ptr(int64(500)),
								ParentSpanID:           []byte{},
								StatusCode:             int(v1.Status_STATUS_CODE_ERROR),
								StatusMessage:          v1.Status_STATUS_CODE_ERROR.String(),
								TraceState:             "tracestate",
								Kind:                   int(v1.Span_SPAN_KIND_CLIENT),
								DroppedAttributesCount: 42,
								DroppedEventsCount:     43,
								Attrs: []Attribute{
									attr("foo", "def"),
									attr("bar", 123),
									attr("float", 456.78),
									attr("bool", false),
									attr("string-array", []string{"value-one"}),
									attr("int-array", []int64{11, 22}),
									attr("double-array", []float64{1.1, 2.2, 3.3}),
									attr("bool-array", []bool{true, false, true, false}),
									// Edge-cases
									attr(LabelName, "Bob"),                    // Conflicts with intrinsic but still looked up by .name
									attr(LabelServiceName, "spanservicename"), // Overrides resource-level dedicated column
									attr(LabelHTTPStatusCode, "500ouch"),      // Different type than dedicated column
									// Unsupported attributes
									{Key: "unsupported-mixed-array", ValueDropped: mixedArrayAttrValue, ValueType: attrTypeNotSupported},
									{Key: "unsupported-kv-list", ValueDropped: kvListValue, ValueType: attrTypeNotSupported},
								},
								Events: []Event{
									{
										TimeSinceStartNano: 1,
										Name:               "e1",
										Attrs: []Attribute{
											attr("event-attr-key-1", "event-value-1"),
											attr("event-attr-key-2", "event-value-2"),
										},
									},
									{TimeSinceStartNano: 2, Name: "e2", Attrs: []Attribute{}},
								},
								Links: links,
								DedicatedAttributes: DedicatedAttributes{
									String01: ptr("dedicated-span-attr-value-1"),
									String02: ptr("dedicated-span-attr-value-2"),
									String03: ptr("dedicated-span-attr-value-3"),
									String04: ptr("dedicated-span-attr-value-4"),
									String05: ptr("dedicated-span-attr-value-5"),
								},
							},
						},
					},
				},
			},
			{
				Resource: Resource{
					ServiceName:      "service2",
					Cluster:          ptr("cluster2"),
					Namespace:        ptr("namespace2"),
					Pod:              ptr("pod2"),
					Container:        ptr("container2"),
					K8sClusterName:   ptr("k8scluster2"),
					K8sNamespaceName: ptr("k8snamespace2"),
					K8sPodName:       ptr("k8spod2"),
					K8sContainerName: ptr("k8scontainer2"),
					Attrs: []Attribute{
						attr("foo", "abc2"),
						attr(LabelServiceName, 1234), // Different type than dedicated column
					},
					DedicatedAttributes: DedicatedAttributes{
						String01: ptr("dedicated-resource-attr-value-6"),
						String02: ptr("dedicated-resource-attr-value-7"),
						String03: ptr("dedicated-resource-attr-value-8"),
						String04: ptr("dedicated-resource-attr-value-9"),
						String05: ptr("dedicated-resource-attr-value-10"),
					},
				},
				ScopeSpans: []ScopeSpans{
					{
						Spans: []Span{
							{
								SpanID:                 []byte("spanid2"),
								Name:                   "world",
								StartTimeUnixNano:      uint64(200 * time.Second),
								DurationNano:           uint64(200 * time.Second),
								HttpMethod:             ptr("PUT"),
								HttpUrl:                ptr("url/hello/world/2"),
								HttpStatusCode:         ptr(int64(501)),
								StatusCode:             int(v1.Status_STATUS_CODE_OK),
								StatusMessage:          v1.Status_STATUS_CODE_OK.String(),
								TraceState:             "tracestate2",
								Kind:                   int(v1.Span_SPAN_KIND_SERVER),
								DroppedAttributesCount: 45,
								DroppedEventsCount:     46,
								Attrs: []Attribute{
									attr("foo", "ghi"),
									attr("bar", 1234),
									attr("float", 456.789),
									attr("bool", true),
									// Edge-cases
									attr(LabelName, "Bob2"),                    // Conflicts with intrinsic but still looked up by .name
									attr(LabelServiceName, "spanservicename2"), // Overrides resource-level dedicated column
									attr(LabelHTTPStatusCode, "500ouch2"),      // Different type than dedicated column
								},
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
		{"spanAttValNoMatch", "{ span.bloom = `bar` }"},
		{"spanAttIntrinsicNoMatch", "{ name = `asdfasdf` }"},

		// resource
		{"resourceAttValNoMatch", "{ resource.module.path = `bar` }"},
		{"resourceAttIntrinsicMatch", "{ resource.service.name = `tempo-query-frontend` }"},

		// mixed
		{"mixedValNoMatch", "{ .bloom = `bar` }"},
		{"mixedValMixedMatchAnd", "{ resource.foo = `bar` && name = `gcs.ReadRange` }"},
		{"mixedValMixedMatchOr", "{ resource.foo = `bar` || name = `gcs.ReadRange` }"},

		{"count", "{ } | count() > 1"},
		{"struct", "{ resource.service.name != `loki-querier` } >> { resource.service.name = `loki-querier` && status = error }"},
		{"||", "{ resource.service.name = `loki-querier` } || { resource.service.name = `loki-ingester` }"},
		{"mixed", `{resource.namespace!="" && resource.service.name="loki-distributor" && duration>2s && resource.cluster=~"prod.*"}`},
	}

	ctx := context.TODO()
	tenantID := "1"
	blockID := uuid.MustParse("00000c2f-8133-4a60-a62a-7748bd146938")
	// blockID := uuid.MustParse("06ebd383-8d4e-4289-b0e9-cf2197d611d5")

	r, _, _, err := local.New(&local.Config{
		Path: path.Join("/home/joe/testblock/"),
		// Path: path.Join("/Users/marty/src/tmp"),
	})
	require.NoError(b, err)

	rr := backend.NewReader(r)
	meta, err := rr.BlockMeta(ctx, blockID, tenantID)
	require.NoError(b, err)

	opts := common.DefaultSearchOptions()
	opts.StartPage = 10
	opts.TotalPages = 1

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
	blockID := uuid.MustParse("06ebd383-8d4e-4289-b0e9-cf2197d611d5")

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

func ptr[T any](v T) *T {
	return &v
}

func attr(key string, val any) Attribute {
	switch val := val.(type) {
	case string:
		return Attribute{Key: key, Value: []string{val}, ValueType: attrTypeString}
	case []string:
		return Attribute{Key: key, Value: val, ValueType: attrTypeStringArray}
	case int:
		return Attribute{Key: key, ValueInt: []int64{int64(val)}, ValueType: attrTypeInt}
	case []int64:
		return Attribute{Key: key, ValueInt: val, ValueType: attrTypeIntArray}
	case float64:
		return Attribute{Key: key, ValueDouble: []float64{val}, ValueType: attrTypeDouble}
	case []float64:
		return Attribute{Key: key, ValueDouble: val, ValueType: attrTypeDoubleArray}
	case bool:
		return Attribute{Key: key, ValueBool: []bool{val}, ValueType: attrTypeBool}
	case []bool:
		return Attribute{Key: key, ValueBool: val, ValueType: attrTypeBoolArray}
	default:
		panic(fmt.Sprintf("type %T not supported for attribute '%s'", val, key))
	}
}
