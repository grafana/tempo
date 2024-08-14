package vparquet4

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/traceqlmetrics"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/pkg/util/traceidboundary"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

func TestOne(t *testing.T) {
	wantTr := fullyPopulatedTestTrace(nil)
	b := makeBackendBlockWithTraces(t, []*Trace{wantTr})
	ctx := context.Background()
	q := `{ resource.region != nil && resource.service.name = "bar" }`
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
	traceIDText := util.TraceIDToHexString(wantTraceID)

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
		{"Intrinsic: trace:id", traceql.MustExtractFetchSpansRequestWithMetadata(`{ trace:id = "` + traceIDText + `" }`)},
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
		// Events
		{"event:name", traceql.MustExtractFetchSpansRequestWithMetadata(`{event:name = "e1"}`)},
		{"event:timeSinceStart", traceql.MustExtractFetchSpansRequestWithMetadata(`{event:timeSinceStart > 2ms}`)},
		{"event.message", traceql.MustExtractFetchSpansRequestWithMetadata(`{event.message =~ "exception"}`)},
		// Links
		{"link:spanID", traceql.MustExtractFetchSpansRequestWithMetadata(`{link:spanID = "1234567890abcdef"}`)},
		{"link:traceID", traceql.MustExtractFetchSpansRequestWithMetadata(`{link:traceID = "1234567890abcdef1234567890abcdef"}`)},
		{"link.opentracing.ref_type", traceql.MustExtractFetchSpansRequestWithMetadata(`{link.opentracing.ref_type = "child-of"}`)},
		// Instrumentation Scope
		{"scope:name", traceql.MustExtractFetchSpansRequestWithMetadata(`{scope:name = "scope-1"}`)},
		{"scope:version", traceql.MustExtractFetchSpansRequestWithMetadata(`{scope:version = "version-1"}`)},
		{"scope.attr-str", traceql.MustExtractFetchSpansRequestWithMetadata(`{scope.scope-attr-str = "scope-attr-1"}`)},
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
		{"Intrinsic: event:name", traceql.MustExtractFetchSpansRequestWithMetadata(`{event:name = "x2"}`)},
		{"Intrinsic: link:spanID", traceql.MustExtractFetchSpansRequestWithMetadata(`{link:spanID = "ffffffffffffffff"}`)},
		{"Intrinsic: link:traceID", traceql.MustExtractFetchSpansRequestWithMetadata(`{link:traceID = "ffffffffffffffffffffffffffffffff"}`)},
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

func TestBackendBlockSearchTraceQLEvents(t *testing.T) {
	numTraces := 50
	traces := make([]*Trace, 0, numTraces)
	wantTraceIdx := rand.Intn(numTraces)
	wantTraceID := test.ValidTraceID(nil)

	for i := 0; i < numTraces; i++ {
		if i == wantTraceIdx {
			// this trace has one span with two identical events
			traces = append(traces, fullyPopulatedTestTrace(wantTraceID))
			continue
		}

		id := test.ValidTraceID(nil)
		tr, _ := traceToParquet(&backend.BlockMeta{}, id, test.MakeTrace(1, id), nil)
		traces = append(traces, tr)
	}

	b := makeBackendBlockWithTraces(t, traces)
	ctx := context.Background()

	requests := []string{
		`{event.message =~ "exception"}`,
		`{event:name = "e1"}`,
		`{event:timeSinceStart > 2ms}`,
	}

	for _, request := range requests {
		t.Run(request, func(t *testing.T) {
			req := traceql.MustExtractFetchSpansRequestWithMetadata(request)
			if req.SecondPass == nil {
				req.SecondPass = func(s *traceql.Spanset) ([]*traceql.Spanset, error) { return []*traceql.Spanset{s}, nil }
				req.SecondPassConditions = traceql.SearchMetaConditions()
			}

			resp, err := b.Fetch(ctx, req, common.DefaultSearchOptions())
			require.NoError(t, err, "search request:%v", req)

			found := false
			count := 0
			for {
				spanSet, err := resp.Results.Next(ctx)
				require.NoError(t, err, "search request:%v", req)
				if spanSet == nil {
					break
				}
				found = bytes.Equal(spanSet.TraceID, wantTraceID)
				if found {
					count++
				}
			}
			require.True(t, found, "search request:%v", req)
			// two events in the same span should still return just one span
			require.Equal(t, 1, count, "search request:%v", req)
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
	linkTraceID, _ := util.HexStringToTraceID("1234567890abcdef1234567890abcdef")
	linkSpanID, _ := util.HexStringToSpanID("1234567890abcdef")

	links := []Link{
		{
			TraceID:                linkTraceID,
			SpanID:                 linkSpanID,
			TraceState:             "state",
			DroppedAttributesCount: 3,
			Attrs: []Attribute{
				attr("opentracing.ref_type", "child-of"),
			},
		},
	}

	mixedArrayAttrValue := "{\"arrayValue\":{\"values\":[{\"stringValue\":\"value-one\"},{\"intValue\":\"100\"}]}}"
	kvListValue := "{\"kvlistValue\":{\"values\":[{\"key\":\"key-one\",\"value\":{\"stringValue\":\"value-one\"}},{\"key\":\"key-two\",\"value\":{\"stringValue\":\"value-two\"}}]}}"

	return &Trace{
		TraceID:           test.ValidTraceID(id),
		TraceIDText:       util.TraceIDToHexString(id),
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
						{Key: "unsupported-mixed-array", ValueUnsupported: &mixedArrayAttrValue, IsArray: false},
						{Key: "unsupported-kv-list", ValueUnsupported: &kvListValue, IsArray: false},
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
						Scope: InstrumentationScope{
							Name:                   "scope-1",
							Version:                "version-1",
							DroppedAttributesCount: 1,
							Attrs: []Attribute{
								attr("scope-attr-str", "scope-attr-1"),
								attr("scope-attr-int", 101),
								attr("scope-attr-float", 3.14),
								attr("scope-attr-bool", true),
							},
						},
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
									{Key: "unsupported-mixed-array", ValueUnsupported: &mixedArrayAttrValue, IsArray: false},
									{Key: "unsupported-kv-list", ValueUnsupported: &kvListValue, IsArray: false},
								},
								Events: []Event{
									{
										TimeSinceStartNano: 3 * 1000 * 1000, // 3ms
										Name:               "e1",
										Attrs: []Attribute{
											attr("event-attr-key-1", "event-value-1"),
											attr("event-attr-key-2", "event-value-2"),
											attr("message", "exception"),
										},
									},
									{TimeSinceStartNano: 2, Name: "e2", Attrs: []Attribute{}},
									{
										TimeSinceStartNano: 3 * 1000 * 1000, // 3ms
										Name:               "e1",
										Attrs: []Attribute{
											attr("event-attr-key-1", "event-value-1"),
											attr("event-attr-key-2", "event-value-2"),
											attr("message", "exception"),
										},
									},
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
						Scope: InstrumentationScope{
							Name:    "scope-2",
							Version: "version-2",
							Attrs: []Attribute{
								attr("scope-attr-str", "scope-attr-2"),
							},
						},
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

func TestBackendBlockSelectAll(t *testing.T) {
	var (
		ctx          = context.Background()
		numTraces    = 250
		traces       = make([]*Trace, 0, numTraces)
		wantTraceIdx = rand.Intn(numTraces)
		wantTraceID  = test.ValidTraceID(nil)
		wantTrace    = fullyPopulatedTestTrace(wantTraceID)
		dc           = test.MakeDedicatedColumns()
		dcm          = dedicatedColumnsToColumnMapping(dc)
	)

	// TODO - This strips unsupported attributes types for now. Revisit when
	// add support for arrays/kvlists in the fetch layer.
	trimForSelectAll(wantTrace)

	for i := 0; i < numTraces; i++ {
		if i == wantTraceIdx {
			traces = append(traces, wantTrace)
			continue
		}

		id := test.ValidTraceID(nil)
		tr, _ := traceToParquet(&backend.BlockMeta{}, id, test.MakeTrace(1, id), nil)
		traces = append(traces, tr)
	}

	b := makeBackendBlockWithTraces(t, traces)

	_, _, _, req, err := traceql.NewEngine().Compile("{}")
	require.NoError(t, err)
	req.SecondPass = func(inSS *traceql.Spanset) ([]*traceql.Spanset, error) { return []*traceql.Spanset{inSS}, nil }
	req.SecondPassSelectAll = true

	resp, err := b.Fetch(ctx, *req, common.DefaultSearchOptions())
	require.NoError(t, err)
	defer resp.Results.Close()

	// This is a dump of all spans in the fully-populated test trace
	wantSS := flattenForSelectAll(wantTrace, dcm)

	for {
		// Seek to our desired trace
		ss, err := resp.Results.Next(ctx)
		require.NoError(t, err)
		if ss == nil {
			break
		}
		if !bytes.Equal(ss.TraceID, wantTraceID) {
			continue
		}

		// Cleanup found data for comparison
		// equal will fail on the rownum mismatches. this is an internal detail to the
		// fetch layer. just wipe them out here
		ss.ReleaseFn = nil
		ss.ServiceStats = nil
		for _, sp := range ss.Spans {
			s := sp.(*span)
			s.cbSpanset = nil
			s.cbSpansetFinal = false
			s.rowNum = parquetquery.RowNumber{}
			s.startTimeUnixNanos = 0 // selectall doesn't imply start time
			sortAttrs(s.traceAttrs)
			sortAttrs(s.resourceAttrs)
			sortAttrs(s.spanAttrs)
			sortAttrs(s.scopeAttrs)
		}

		require.Equal(t, wantSS, ss)
	}
}

func sortAttrs(attrs []attrVal) {
	sort.SliceStable(attrs, func(i, j int) bool {
		is := attrs[i].a.String()
		js := attrs[j].a.String()
		if is == js {
			// Compare by value
			return attrs[i].s.String() < attrs[j].s.String()
		}
		return is < js
	})
}

func trimArrayAttrs(in []Attribute) []Attribute {
	out := []Attribute{}
	for _, a := range in {
		if a.IsArray || a.ValueUnsupported != nil {
			continue
		}
		out = append(out, a)
	}
	return out
}

func trimForSelectAll(tr *Trace) {
	for i, rs := range tr.ResourceSpans {
		tr.ResourceSpans[i].Resource.Attrs = trimArrayAttrs(rs.Resource.Attrs)
		for j, ss := range rs.ScopeSpans {
			for k, s := range ss.Spans {
				tr.ResourceSpans[i].ScopeSpans[j].Spans[k].Attrs = trimArrayAttrs(s.Attrs)
			}
		}
	}
}

func flattenForSelectAll(tr *Trace, dcm dedicatedColumnMapping) *traceql.Spanset {
	var traceAttrs []attrVal
	newSS := &traceql.Spanset{
		RootServiceName: tr.RootServiceName,
		RootSpanName:    tr.RootSpanName,
		TraceID:         tr.TraceID,
		DurationNanos:   tr.DurationNano,
	}
	traceAttrs = append(traceAttrs, attrVal{traceql.IntrinsicTraceIDAttribute, traceql.NewStaticString(tr.TraceIDText)})
	traceAttrs = append(traceAttrs, attrVal{traceql.IntrinsicTraceDurationAttribute, traceql.NewStaticDuration(time.Duration(tr.DurationNano))})
	traceAttrs = append(traceAttrs, attrVal{traceql.IntrinsicTraceRootServiceAttribute, traceql.NewStaticString(tr.RootServiceName)})
	traceAttrs = append(traceAttrs, attrVal{traceql.IntrinsicTraceRootSpanAttribute, traceql.NewStaticString(tr.RootSpanName)})
	sortAttrs(traceAttrs)

	for _, rs := range tr.ResourceSpans {
		var rsAttrs []attrVal
		rsAttrs = append(rsAttrs, attrVal{traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, LabelServiceName), traceql.NewStaticString(rs.Resource.ServiceName)})
		rsAttrs = append(rsAttrs, attrVal{traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, LabelCluster), traceql.NewStaticString(*rs.Resource.Cluster)})
		rsAttrs = append(rsAttrs, attrVal{traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, LabelNamespace), traceql.NewStaticString(*rs.Resource.Namespace)})
		rsAttrs = append(rsAttrs, attrVal{traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, LabelPod), traceql.NewStaticString(*rs.Resource.Pod)})
		rsAttrs = append(rsAttrs, attrVal{traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, LabelContainer), traceql.NewStaticString(*rs.Resource.Container)})
		rsAttrs = append(rsAttrs, attrVal{traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, LabelK8sClusterName), traceql.NewStaticString(*rs.Resource.K8sClusterName)})
		rsAttrs = append(rsAttrs, attrVal{traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, LabelK8sNamespaceName), traceql.NewStaticString(*rs.Resource.K8sNamespaceName)})
		rsAttrs = append(rsAttrs, attrVal{traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, LabelK8sPodName), traceql.NewStaticString(*rs.Resource.K8sPodName)})
		rsAttrs = append(rsAttrs, attrVal{traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, LabelK8sContainerName), traceql.NewStaticString(*rs.Resource.K8sContainerName)})

		for _, a := range parquetToProtoAttrs(rs.Resource.Attrs) {
			if arr := a.Value.GetArrayValue(); arr != nil {
				for _, v := range arr.Values {
					rsAttrs = append(rsAttrs, attrVal{traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, a.Key), traceql.StaticFromAnyValue(v)})
				}
				continue
			}
			rsAttrs = append(rsAttrs, attrVal{traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, a.Key), traceql.StaticFromAnyValue(a.Value)})
		}

		dcm.forEach(func(attr string, column dedicatedColumn) {
			if strings.Contains(column.ColumnPath, "Resource") {
				v := column.readValue(&rs.Resource.DedicatedAttributes)
				if v == nil {
					return
				}
				a := traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, attr)
				s := traceql.StaticFromAnyValue(v)
				rsAttrs = append(rsAttrs, attrVal{a, s})
			}
		})

		sortAttrs(rsAttrs)

		for _, ss := range rs.ScopeSpans {
			var scopeAttrs []attrVal
			scopeAttrs = append(scopeAttrs, attrVal{traceql.IntrinsicScopeNameAttribute, traceql.NewStaticString(ss.Scope.Name)})
			scopeAttrs = append(scopeAttrs, attrVal{traceql.IntrinsicScopeVersionAttribute, traceql.NewStaticString(ss.Scope.Version)})
			for _, a := range parquetToProtoAttrs(ss.Scope.Attrs) {
				if arr := a.Value.GetArrayValue(); arr != nil {
					for _, v := range arr.Values {
						scopeAttrs = append(scopeAttrs, attrVal{traceql.NewScopedAttribute(traceql.AttributeScopeInstrumentation, false, a.Key), traceql.StaticFromAnyValue(v)})
					}
					continue
				}
				scopeAttrs = append(scopeAttrs, attrVal{traceql.NewScopedAttribute(traceql.AttributeScopeInstrumentation, false, a.Key), traceql.StaticFromAnyValue(a.Value)})
			}
			sortAttrs(scopeAttrs)

			for _, s := range ss.Spans {

				newS := &span{}
				// newS.id = s.SpanID  SpanID isn't implied by SelectAll
				// newS.startTimeUnixNanos = s.StartTimeUnixNano Span StartTime isn't implied by selectAll
				newS.durationNanos = s.DurationNano
				newS.setTraceAttrs(traceAttrs)
				newS.setResourceAttrs(rsAttrs)
				newS.setScopeAttrs(scopeAttrs)
				newS.addSpanAttr(traceql.IntrinsicDurationAttribute, traceql.NewStaticDuration(time.Duration(s.DurationNano)))
				newS.addSpanAttr(traceql.IntrinsicKindAttribute, traceql.NewStaticKind(otlpKindToTraceqlKind(uint64(s.Kind))))
				newS.addSpanAttr(traceql.IntrinsicNameAttribute, traceql.NewStaticString(s.Name))
				newS.addSpanAttr(traceql.IntrinsicStatusAttribute, traceql.NewStaticStatus(otlpStatusToTraceqlStatus(uint64(s.StatusCode))))
				newS.addSpanAttr(traceql.IntrinsicStatusMessageAttribute, traceql.NewStaticString(s.StatusMessage))
				if s.HttpStatusCode != nil {
					newS.addSpanAttr(traceql.NewScopedAttribute(traceql.AttributeScopeSpan, false, LabelHTTPStatusCode), traceql.NewStaticInt(int(*s.HttpStatusCode)))
				}
				if s.HttpMethod != nil {
					newS.addSpanAttr(traceql.NewScopedAttribute(traceql.AttributeScopeSpan, false, LabelHTTPMethod), traceql.NewStaticString(*s.HttpMethod))
				}
				if s.HttpUrl != nil {
					newS.addSpanAttr(traceql.NewScopedAttribute(traceql.AttributeScopeSpan, false, LabelHTTPUrl), traceql.NewStaticString(*s.HttpUrl))
				}

				dcm.forEach(func(attr string, column dedicatedColumn) {
					if strings.Contains(column.ColumnPath, "Span") {
						v := column.readValue(&s.DedicatedAttributes)
						if v == nil {
							return
						}
						a := traceql.NewScopedAttribute(traceql.AttributeScopeSpan, false, attr)
						s := traceql.StaticFromAnyValue(v)
						newS.addSpanAttr(a, s)
					}
				})

				for _, a := range parquetToProtoAttrs(s.Attrs) {
					if arr := a.Value.GetArrayValue(); arr != nil {
						for _, v := range arr.Values {
							newS.addSpanAttr(traceql.NewScopedAttribute(traceql.AttributeScopeSpan, false, a.Key), traceql.StaticFromAnyValue(v))
						}
						continue
					}
					newS.addSpanAttr(traceql.NewScopedAttribute(traceql.AttributeScopeSpan, false, a.Key), traceql.StaticFromAnyValue(a.Value))
				}

				sortAttrs(newS.spanAttrs)
				newSS.Spans = append(newSS.Spans, newS)
			}
		}
	}
	return newSS
}

func BenchmarkBackendBlockTraceQL(b *testing.B) {
	testCases := []struct {
		name  string
		query string
	}{
		// span
		{"spanAttValMatch", "{ span.component = `net/http` }"},
		{"spanAttValNoMatch", "{ span.bloom = `does-not-exit-6c2408325a45` }"},
		{"spanAttIntrinsicMatch", "{ name = `/cortex.Ingester/Push` }"},
		{"spanAttIntrinsicNoMatch", "{ name = `does-not-exit-6c2408325a45` }"},

		// resource
		{"resourceAttValMatch", "{ resource.opencensus.exporterversion = `Jaeger-Go-2.30.0` }"},
		{"resourceAttValNoMatch", "{ resource.module.path = `does-not-exit-6c2408325a45` }"},
		{"resourceAttIntrinsicMatch", "{ resource.service.name = `tempo-gateway` }"},
		{"resourceAttIntrinsicMatch", "{ resource.service.name = `does-not-exit-6c2408325a45` }"},

		// trace
		{"traceOrMatch", "{ rootServiceName = `tempo-gateway` && (status = error || span.http.status_code = 500)}"},
		{"traceOrNoMatch", "{ rootServiceName = `doesntexist` && (status = error || span.http.status_code = 500)}"},

		// mixed
		{"mixedValNoMatch", "{ .bloom = `does-not-exit-6c2408325a45` }"},
		{"mixedValMixedMatchAnd", "{ resource.foo = `bar` && name = `gcs.ReadRange` }"},
		{"mixedValMixedMatchOr", "{ resource.foo = `bar` || name = `gcs.ReadRange` }"},

		{"count", "{ } | count() > 1"},
		{"struct", "{ resource.service.name != `loki-querier` } >> { resource.service.name = `loki-gateway` && status = error }"},
		{"||", "{ resource.service.name = `loki-querier` } || { resource.service.name = `loki-gateway` }"},
		{"mixed", `{resource.namespace!="" && resource.service.name="cortex-gateway" && duration>50ms && resource.cluster=~"prod.*"}`},
		{"complex", `{resource.cluster=~"prod.*" && resource.namespace = "tempo-prod" && resource.container="query-frontend" && name = "HTTP GET - tempo_api_v2_search_tags" && span.http.status_code = 200 && duration > 1s}`},
	}

	ctx := context.TODO()
	tenantID := "1"
	// blockID := uuid.MustParse("06ebd383-8d4e-4289-b0e9-cf2197d611d5")
	blockID := uuid.MustParse("0008e57d-069d-4510-a001-b9433b2da08c")

	r, _, _, err := local.New(&local.Config{
		// Path: path.Join("/Users/marty/src/tmp"),
		Path: path.Join("/Users/mapno/workspace/testblock"),
	})
	require.NoError(b, err)

	rr := backend.NewReader(r)
	meta, err := rr.BlockMeta(ctx, blockID, tenantID)
	require.NoError(b, err)

	opts := common.DefaultSearchOptions()
	opts.StartPage = 3
	opts.TotalPages = 2

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

func BenchmarkBackendBlockQueryRange(b *testing.B) {
	testCases := []string{
		"{} | rate()",
		"{} | rate() by (name)",
		"{} | rate() by (resource.service.name)",
		"{} | rate() by (span.http.url)", // High cardinality attribute
		"{resource.service.name=`loki-ingester`} | rate()",
		"{status=error} | rate()",
	}

	var (
		ctx      = context.TODO()
		e        = traceql.NewEngine()
		tenantID = "1"
		// blockID  = uuid.MustParse("06ebd383-8d4e-4289-b0e9-cf2197d611d5")
		blockID = uuid.MustParse("0008e57d-069d-4510-a001-b9433b2da08c")
		path    = "/Users/mapno/workspace/testblock"
		// path = "/Users/marty/src/tmp"
	)

	r, _, _, err := local.New(&local.Config{
		Path: path,
	})
	require.NoError(b, err)

	rr := backend.NewReader(r)
	meta, err := rr.BlockMeta(ctx, blockID, tenantID)
	require.NoError(b, err)
	require.Equal(b, VersionString, meta.Version)

	opts := common.DefaultSearchOptions()
	opts.TotalPages = 10
	block := newBackendBlock(meta, rr)
	_, _, err = block.openForSearch(ctx, opts)
	require.NoError(b, err)

	f := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
		return block.Fetch(ctx, req, opts)
	})

	for _, tc := range testCases {
		b.Run(tc, func(b *testing.B) {
			for _, minutes := range []int{5, 7} {
				b.Run(strconv.Itoa(minutes), func(b *testing.B) {
					st := meta.StartTime
					end := st.Add(time.Duration(minutes) * time.Minute)

					if end.After(meta.EndTime) {
						b.SkipNow()
						return
					}

					req := &tempopb.QueryRangeRequest{
						Query: tc,
						Step:  uint64(time.Minute),
						Start: uint64(st.UnixNano()),
						End:   uint64(end.UnixNano()),
					}

					eval, err := e.CompileMetricsQueryRange(req, false, 2, 0, false)
					require.NoError(b, err)

					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						err := eval.Do(ctx, f, uint64(block.meta.StartTime.UnixNano()), uint64(block.meta.EndTime.UnixNano()))
						require.NoError(b, err)
					}

					bytes, spansTotal, _ := eval.Metrics()
					b.ReportMetric(float64(bytes)/float64(b.N)/1024.0/1024.0, "MB_IO/op")
					b.ReportMetric(float64(spansTotal)/float64(b.N), "spans/op")
					b.ReportMetric(float64(spansTotal)/b.Elapsed().Seconds(), "spans/s")
				})
			}
		})
	}
}

// TestBackendBlockQueryRange is the `TestOne` of metric queries.
// It's skipped because it depends on a local block, like benchmarks
//
// You also need to manually print the iterator in `backendBlock.Fetch`,
// because there is no access to the iterator in the test. Sad.
func TestBackendBlockQueryRange(t *testing.T) {
	if os.Getenv("debug") != "1" {
		t.Skip()
	}

	testCases := []string{
		"{} | rate()",
		"{} | rate() by (name)",
		"{} | rate() by (resource.service.name)",
		"{} | rate() by (span.http.url)", // High cardinality attribute
		"{resource.service.name=`tempo-ingester`} | rate()",
		"{status=unset} | rate()",
	}

	const (
		tenantID  = "1"
		queryHint = "with(exemplars=true)"
	)

	var (
		ctx     = context.TODO()
		e       = traceql.NewEngine()
		opts    = common.DefaultSearchOptions()
		blockID = uuid.MustParse("0008e57d-069d-4510-a001-b9433b2da08c")
		path    = path.Join("/Users/mapno/workspace/testblock")
	)

	r, _, _, err := local.New(&local.Config{
		Path: path,
	})
	require.NoError(t, err)

	rr := backend.NewReader(r)
	meta, err := rr.BlockMeta(ctx, blockID, tenantID)
	require.NoError(t, err)
	require.Equal(t, VersionString, meta.Version)

	block := newBackendBlock(meta, rr)
	opts.TotalPages = 10
	_, _, err = block.openForSearch(ctx, opts)
	require.NoError(t, err)

	f := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
		return block.Fetch(ctx, req, opts)
	})

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			st := meta.StartTime
			end := st.Add(time.Duration(5) * time.Minute)

			if end.After(meta.EndTime) {
				t.SkipNow()
				return
			}

			req := &tempopb.QueryRangeRequest{
				Query: fmt.Sprintf("%s %s", tc, queryHint),
				Step:  uint64(time.Minute),
				Start: uint64(st.UnixNano()),
				End:   uint64(end.UnixNano()),
			}

			eval, err := e.CompileMetricsQueryRange(req, false, 1, 0, false)
			require.NoError(t, err)

			require.NoError(t, eval.Do(ctx, f, uint64(block.meta.StartTime.UnixNano()), uint64(block.meta.EndTime.UnixNano())))

			ss := eval.Results()
			require.NotNil(t, ss)

			for _, s := range ss {
				if s.Exemplars != nil && len(s.Exemplars) > 0 {
					fmt.Println("series", s.Labels)
					fmt.Println("Exemplars", s.Exemplars)
				}
			}
		})
	}
}

func TestTraceIDShardingQuality(t *testing.T) {
	// Use debug=1 go test -v -run=TestTraceIDShardingQuality
	if os.Getenv("debug") != "1" {
		t.Skip()
	}

	var (
		ctx      = context.TODO()
		opts     = common.DefaultSearchOptions()
		tenantID = "1"
		// blockID  = uuid.MustParse("06ebd383-8d4e-4289-b0e9-cf2197d611d5")
		blockID = uuid.MustParse("18364616-f80d-45a6-b2a3-cb63e203edff")
		path    = "/Users/marty/src/tmp/"
	)

	r, _, _, err := local.New(&local.Config{
		Path: path,
	})
	require.NoError(t, err)

	rr := backend.NewReader(r)
	meta, err := rr.BlockMeta(ctx, blockID, tenantID)
	require.NoError(t, err)
	require.Equal(t, VersionString, meta.Version)

	block := newBackendBlock(meta, rr)
	pf, _, err := block.openForSearch(ctx, opts)
	require.NoError(t, err)

	fetchReq := traceql.FetchSpansRequest{
		AllConditions: true,
		Conditions: []traceql.Condition{
			{Attribute: traceql.IntrinsicTraceIDAttribute},
		},
	}

	f := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
		return block.Fetch(ctx, req, opts)
	})

	summarizeCounts := func(prefix string, cs []int) {
		count := len(cs)
		sum := 0
		l := math.MaxInt
		h := math.MinInt
		for _, v := range cs {
			sum += v
			l = min(l, v)
			h = max(h, v)
		}
		fmt.Printf("Shards:%d %s: Min:%d Max:%d Total:%d Avg:%.1f Quality:%.1f %%\n",
			count, prefix, l, h, sum, float64(sum)/float64(count), float64(l)/float64(h)*100.0)
	}

	shardsToTest := []int{10, 100, 1000}

	for _, shards := range shardsToTest {
		t.Run(strconv.Itoa(shards), func(t *testing.T) {
			var (
				rgCounts = make([]int, shards)
				trCounts = make([]int, shards)
				pairs    = make([][]traceidboundary.Boundary, shards)
				funcs    = make([]func([]byte) bool, shards)
			)

			for s := 1; s <= shards; s++ {
				pairs[s-1], _ = traceidboundary.Pairs(uint32(s), uint32(shards))
				funcs[s-1], _ = traceidboundary.Funcs(uint32(s), uint32(shards))

				rgs, err := block.rowGroupsForShard(ctx, pf, *meta, uint32(s), uint32(shards))
				require.NoError(t, err)
				rgCounts[s-1] = len(rgs)
			}

			resp, err := f.Fetch(ctx, fetchReq)
			require.NoError(t, err)
			defer resp.Results.Close()

			for {
				ss, err := resp.Results.Next(ctx)
				require.NoError(t, err)

				if ss == nil {
					break
				}

				// Match the trace ID against every shard
				matched := []int{}
				for i := 0; i < shards; i++ {
					if funcs[i](ss.TraceID) {
						trCounts[i]++
						matched = append(matched, i)
					}
				}

				// Check for missing or overlapping ranges
				if len(matched) > 1 {
					fmt.Printf("TraceID %X matched %d shards\n", ss.TraceID, len(matched))
					for _, s := range matched {
						for i, b := range pairs[s] {
							fmt.Printf("  Bucket %d %d: %X %X\n", s, i, b.Min, b.Max)
						}
					}
					panic("trace matched multiple shards")
				} else if len(matched) == 0 {
					fmt.Println("TraceID not matched by any shard:", ss.TraceID)
					panic("trace id not matched by any shard")
				}

				ss.Release()
			}

			summarizeCounts("Traces", trCounts)
			summarizeCounts("RowGroups", rgCounts)
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}

func attr(key string, val any) Attribute {
	switch val := val.(type) {
	case string:
		return Attribute{Key: key, Value: []string{val}, IsArray: false}
	case []string:
		return Attribute{Key: key, Value: val, IsArray: true}
	case int:
		return Attribute{Key: key, ValueInt: []int64{int64(val)}, IsArray: false}
	case []int64:
		return Attribute{Key: key, ValueInt: val, IsArray: true}
	case float64:
		return Attribute{Key: key, ValueDouble: []float64{val}, IsArray: false}
	case []float64:
		return Attribute{Key: key, ValueDouble: val, IsArray: true}
	case bool:
		return Attribute{Key: key, ValueBool: []bool{val}, IsArray: false}
	case []bool:
		return Attribute{Key: key, ValueBool: val, IsArray: true}
	default:
		panic(fmt.Sprintf("type %T not supported for attribute '%s'", val, key))
	}
}

func TestDescendantOf(t *testing.T) {
	ancestor1 := &span{nestedSetLeft: 3, nestedSetRight: 8}
	descendant1a := &span{nestedSetLeft: 4, nestedSetRight: 5}
	descendant1b := &span{nestedSetLeft: 6, nestedSetRight: 7}

	ancestor2 := &span{nestedSetLeft: 11, nestedSetRight: 19}
	descendant2a := &span{nestedSetLeft: 12, nestedSetRight: 13}
	descendant2b := &span{nestedSetLeft: 14, nestedSetRight: 17}
	descendant2bb := &span{nestedSetLeft: 15, nestedSetRight: 16}

	// adding disconnected spans that purposefully show up before, between and
	// after the above trees
	disconnectedBefore := &span{nestedSetLeft: 1, nestedSetRight: 2}
	disconnectedBetween := &span{nestedSetLeft: 9, nestedSetRight: 10}
	disconnectedAfter := &span{nestedSetLeft: 20, nestedSetRight: 21}

	allDisconnected := []traceql.Span{disconnectedBefore, disconnectedBetween, disconnectedAfter}

	tcs := []struct {
		name        string
		lhs         []traceql.Span
		rhs         []traceql.Span
		falseForAll bool // !<< or !>>
		invert      bool // <<
		union       bool // &>> or &<<
		expected    []traceql.Span
	}{
		{
			name:     "empty",
			lhs:      []traceql.Span{},
			rhs:      []traceql.Span{},
			expected: nil,
		},
		// >>
		{
			name:     "descendant: basic",
			lhs:      []traceql.Span{ancestor1},
			rhs:      []traceql.Span{descendant1a, descendant1b, descendant2a, descendant2b},
			expected: []traceql.Span{descendant1a, descendant1b},
		},
		{
			name:     "descendant: multiple matching trees",
			lhs:      []traceql.Span{ancestor1, ancestor2},
			rhs:      []traceql.Span{descendant1a, descendant1b, descendant2a, descendant2b},
			expected: []traceql.Span{descendant1a, descendant1b, descendant2a, descendant2b},
		},
		{
			name:     "descendant: all",
			lhs:      []traceql.Span{ancestor1, ancestor2, descendant1a, descendant1b, descendant2a, descendant2b, descendant2bb},
			rhs:      []traceql.Span{ancestor1, ancestor2, descendant1a, descendant1b, descendant2a, descendant2b, descendant2bb},
			expected: []traceql.Span{descendant1a, descendant1b, descendant2a, descendant2b, descendant2bb},
		},
		{
			name:     "descendant: don't match self",
			lhs:      []traceql.Span{ancestor1},
			rhs:      []traceql.Span{ancestor1},
			expected: nil,
		},
		// <<
		{
			name:     "ancestor: basic",
			lhs:      []traceql.Span{descendant1a, descendant1b, descendant2a, descendant2b},
			rhs:      []traceql.Span{ancestor1},
			invert:   true,
			expected: []traceql.Span{ancestor1},
		},
		{
			name:     "ancestor: multiple matching trees",
			lhs:      []traceql.Span{descendant1a, descendant1b, descendant2a, descendant2b},
			rhs:      []traceql.Span{ancestor1, ancestor2},
			invert:   true,
			expected: []traceql.Span{ancestor1, ancestor2},
		},
		{
			name:     "ancestor: all",
			lhs:      []traceql.Span{ancestor1, ancestor2, descendant1a, descendant1b, descendant2a, descendant2b, descendant2bb},
			rhs:      []traceql.Span{ancestor1, ancestor2, descendant1a, descendant1b, descendant2a, descendant2b, descendant2bb},
			invert:   true,
			expected: []traceql.Span{ancestor1, ancestor2, descendant2b},
		},
		{
			name:     "ancestor: don't match self",
			lhs:      []traceql.Span{ancestor1},
			rhs:      []traceql.Span{ancestor1},
			invert:   true,
			expected: nil,
		},
		// !>>
		{
			name:        "!descendant: basic",
			lhs:         []traceql.Span{ancestor1},
			rhs:         []traceql.Span{descendant1a, descendant1b, descendant2a, descendant2b},
			falseForAll: true,
			expected:    []traceql.Span{descendant2a, descendant2b},
		},
		{
			name:        "!descendant: multiple matching trees",
			lhs:         []traceql.Span{ancestor1, ancestor2},
			rhs:         []traceql.Span{descendant1a, descendant1b, descendant2a, descendant2b, disconnectedBefore, disconnectedBetween, disconnectedAfter},
			falseForAll: true,
			expected:    []traceql.Span{disconnectedBefore, disconnectedBetween, disconnectedAfter},
		},
		{
			name:        "!descendant: all",
			lhs:         []traceql.Span{ancestor1, ancestor2, descendant1a, descendant1b, descendant2a, descendant2b, descendant2bb},
			rhs:         []traceql.Span{ancestor1, ancestor2, descendant1a, descendant1b, descendant2a, descendant2b, descendant2bb},
			falseForAll: true,
			expected:    []traceql.Span{ancestor1, ancestor2},
		},
		{
			name:        "!descendant: match self",
			lhs:         []traceql.Span{ancestor1},
			rhs:         []traceql.Span{ancestor1},
			falseForAll: true,
			expected:    []traceql.Span{ancestor1},
		},
		// !<<
		{
			name:        "!ancestor: basic",
			lhs:         []traceql.Span{descendant1a, descendant1b, descendant2a, descendant2b},
			rhs:         []traceql.Span{ancestor1, disconnectedBefore, disconnectedBetween, disconnectedAfter},
			invert:      true,
			falseForAll: true,
			expected:    []traceql.Span{disconnectedBefore, disconnectedBetween, disconnectedAfter},
		},
		{
			name:        "!ancestor: multiple matching trees",
			lhs:         []traceql.Span{descendant1a, descendant1b, descendant2a, descendant2b},
			rhs:         []traceql.Span{ancestor1, ancestor2, disconnectedBefore, disconnectedBetween, disconnectedAfter},
			invert:      true,
			falseForAll: true,
			expected:    []traceql.Span{disconnectedBefore, disconnectedBetween, disconnectedAfter},
		},
		{
			name:        "!ancestor: all",
			lhs:         []traceql.Span{ancestor1, ancestor2, descendant1a, descendant1b, descendant2a, descendant2b, descendant2bb},
			rhs:         []traceql.Span{ancestor1, ancestor2, descendant1a, descendant1b, descendant2a, descendant2b, descendant2bb},
			invert:      true,
			falseForAll: true,
			expected:    []traceql.Span{descendant1a, descendant1b, descendant2a, descendant2bb},
		},
		{
			name:        "!ancestor: match self",
			lhs:         []traceql.Span{ancestor1},
			rhs:         []traceql.Span{ancestor1},
			invert:      true,
			falseForAll: true,
			expected:    []traceql.Span{ancestor1},
		},
		// &>>
		{
			name:     "&descendant: basic",
			lhs:      []traceql.Span{ancestor1},
			rhs:      []traceql.Span{descendant1a, descendant1b, descendant2a, descendant2b},
			expected: []traceql.Span{descendant1a, ancestor1, descendant1b},
			union:    true,
		},
		{
			name:     "&descendant: multiple matching trees",
			lhs:      []traceql.Span{ancestor1, ancestor2},
			rhs:      []traceql.Span{descendant1a, descendant1b, descendant2a, descendant2b},
			expected: []traceql.Span{descendant1a, ancestor1, descendant1b, descendant2a, ancestor2, descendant2b},
			union:    true,
		},
		{
			name:     "&descendant: all",
			lhs:      []traceql.Span{ancestor1, ancestor2, descendant1a, descendant1b, descendant2a, descendant2b, descendant2bb},
			rhs:      []traceql.Span{ancestor1, ancestor2, descendant1a, descendant1b, descendant2a, descendant2b, descendant2bb},
			expected: []traceql.Span{descendant1a, ancestor1, descendant1b, descendant2a, ancestor2, descendant2b, descendant2bb},
			union:    true,
		},
		{
			name:     "&descendant: multi-tier",
			lhs:      []traceql.Span{ancestor2, descendant2b, descendant2bb},
			rhs:      []traceql.Span{ancestor2, descendant2bb},
			union:    true,
			expected: []traceql.Span{descendant2bb, ancestor2, descendant2b},
		},
		{
			name:     "&descendant: don't match self",
			lhs:      []traceql.Span{ancestor1},
			rhs:      []traceql.Span{ancestor1},
			expected: nil,
			union:    true,
		},
		// |<<
		{
			name:     "&ancestor: basic",
			lhs:      []traceql.Span{descendant1a, descendant1b, descendant2a, descendant2b},
			rhs:      []traceql.Span{ancestor1},
			invert:   true,
			union:    true,
			expected: []traceql.Span{descendant1a, ancestor1, descendant1b},
		},
		{
			name:     "&ancestor: multiple matching trees",
			lhs:      []traceql.Span{descendant1a, descendant1b, descendant2a, descendant2b},
			rhs:      []traceql.Span{ancestor1, ancestor2},
			invert:   true,
			union:    true,
			expected: []traceql.Span{descendant1a, ancestor1, descendant1b, descendant2a, ancestor2, descendant2b},
		},
		{
			name:     "&ancestor: multi-tier",
			lhs:      []traceql.Span{ancestor2, descendant2b, descendant2bb},
			rhs:      []traceql.Span{ancestor2, descendant2bb},
			invert:   true,
			union:    true,
			expected: []traceql.Span{descendant2b, descendant2bb, ancestor2},
		},
		{
			name:     "&ancestor: all",
			lhs:      []traceql.Span{ancestor1, ancestor2, descendant1a, descendant1b, descendant2a, descendant2b, descendant2bb},
			rhs:      []traceql.Span{ancestor1, ancestor2, descendant1a, descendant1b, descendant2a, descendant2b, descendant2bb},
			invert:   true,
			union:    true,
			expected: []traceql.Span{descendant1a, ancestor1, descendant1b, descendant2a, ancestor2, descendant2b, descendant2bb},
		},
		{
			name:     "&ancestor: don't match self",
			lhs:      []traceql.Span{ancestor1},
			rhs:      []traceql.Span{ancestor1},
			invert:   true,
			union:    true,
			expected: nil,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			s := &span{}

			actual := s.DescendantOf(tc.lhs, tc.rhs, tc.falseForAll, tc.invert, tc.union, nil)
			require.Equal(t, tc.expected, actual)
		})

		// if !falseForAll we can safe insert disconnected spans and get the same results
		if !tc.falseForAll {
			t.Run(tc.name+"-disconnected", func(t *testing.T) {
				s := &span{}

				lhs := append(tc.lhs, allDisconnected...)
				rhs := append(tc.rhs, allDisconnected...)

				actual := s.DescendantOf(lhs, rhs, tc.falseForAll, tc.invert, tc.union, nil)
				require.Equal(t, tc.expected, actual)
			})
		}
	}
}

func TestChildOf(t *testing.T) {
	root := &span{nestedSetLeft: 1, nestedSetParent: -1}

	parent1 := &span{nestedSetLeft: 2, nestedSetParent: 1}
	child1a := &span{nestedSetLeft: 5, nestedSetParent: 2}
	child1aa := &span{nestedSetLeft: 6, nestedSetParent: 5}
	child1b := &span{nestedSetLeft: 10, nestedSetParent: 2}

	parent2 := &span{nestedSetLeft: 15, nestedSetParent: 1}
	child2a := &span{nestedSetLeft: 20, nestedSetParent: 15}
	child2b := &span{nestedSetLeft: 25, nestedSetParent: 15}
	child2bb := &span{nestedSetLeft: 26, nestedSetParent: 25}

	disconnectedBefore := &span{nestedSetLeft: 4, nestedSetParent: 3}
	disconnectedBetween := &span{nestedSetLeft: 12, nestedSetParent: 11}
	disconnectedAfter := &span{nestedSetLeft: 30, nestedSetRight: 29}

	allDisconnected := []traceql.Span{disconnectedBefore, disconnectedBetween, disconnectedAfter}

	tcs := []struct {
		name        string
		lhs         []traceql.Span
		rhs         []traceql.Span
		falseForAll bool // !< or !>
		invert      bool // <
		union       bool // &< or &>
		expected    []traceql.Span
	}{
		{
			name:     "empty",
			lhs:      []traceql.Span{},
			rhs:      []traceql.Span{},
			expected: nil,
		},
		// >
		{
			name:     "child: basic",
			lhs:      []traceql.Span{parent1},
			rhs:      []traceql.Span{child1a, child1b, child2a, child2b},
			expected: []traceql.Span{child1a, child1b},
		},
		{
			name:     "child: multiple matching trees",
			lhs:      []traceql.Span{parent1, parent2},
			rhs:      []traceql.Span{child1a, child1b, child2a, child2b},
			expected: []traceql.Span{child1a, child1b, child2a, child2b},
		},
		{
			name:     "child: all",
			lhs:      []traceql.Span{root, parent1, parent2, child1a, child1aa, child1b, child2a, child2b, child2bb},
			rhs:      []traceql.Span{root, parent1, parent2, child1a, child1aa, child1b, child2a, child2b, child2bb},
			expected: []traceql.Span{parent1, parent2, child1a, child1aa, child1b, child2a, child2b, child2bb},
		},
		{
			name:     "child: don't match self",
			lhs:      []traceql.Span{parent1},
			rhs:      []traceql.Span{parent1},
			expected: nil,
		},
		// <
		{
			name:     "parent: basic",
			lhs:      []traceql.Span{child1a, child1b, child2a, child2b},
			rhs:      []traceql.Span{parent1},
			invert:   true,
			expected: []traceql.Span{parent1},
		},
		{
			name:     "parent: multiple matching trees",
			lhs:      []traceql.Span{child1a, child1b, child2a, child2b},
			rhs:      []traceql.Span{parent1, parent2},
			invert:   true,
			expected: []traceql.Span{parent1, parent2},
		},
		{
			name:     "parent: all",
			lhs:      []traceql.Span{root, parent1, parent2, child1a, child1aa, child1b, child2a, child2b, child2bb},
			rhs:      []traceql.Span{root, parent1, parent2, child1a, child1aa, child1b, child2a, child2b, child2bb},
			invert:   true,
			expected: []traceql.Span{root, parent1, parent2, child1a, child2b},
		},
		{
			name:     "parent: don't match self",
			lhs:      []traceql.Span{parent1},
			rhs:      []traceql.Span{parent1},
			invert:   true,
			expected: nil,
		},
		// !>
		{
			name:        "!child: basic",
			lhs:         []traceql.Span{parent1},
			rhs:         []traceql.Span{child1a, child1b, child2a, child2b},
			falseForAll: true,
			expected:    []traceql.Span{child2a, child2b},
		},
		{
			name:        "!child: multiple matching trees",
			lhs:         []traceql.Span{parent1, parent2},
			rhs:         []traceql.Span{child1a, child1b, child2a, child2b, disconnectedBefore, disconnectedBetween, disconnectedAfter},
			falseForAll: true,
			expected:    []traceql.Span{disconnectedBefore, disconnectedBetween, disconnectedAfter},
		},
		{
			name:        "!child: match self",
			lhs:         []traceql.Span{parent1},
			rhs:         []traceql.Span{parent1},
			falseForAll: true,
			expected:    []traceql.Span{parent1},
		},
		// !<
		{
			name:        "!parent: basic",
			lhs:         []traceql.Span{child1a, child1b, child2a, child2b},
			rhs:         []traceql.Span{parent1, disconnectedBefore, disconnectedBetween, disconnectedAfter},
			invert:      true,
			falseForAll: true,
			expected:    []traceql.Span{disconnectedBefore, disconnectedBetween, disconnectedAfter},
		},
		{
			name:        "!parent: multiple matching trees",
			lhs:         []traceql.Span{child1a, child1b, child2a, child2b},
			rhs:         []traceql.Span{parent1, parent2, disconnectedBefore, disconnectedBetween, disconnectedAfter},
			invert:      true,
			falseForAll: true,
			expected:    []traceql.Span{disconnectedBefore, disconnectedBetween, disconnectedAfter},
		},
		{
			name:        "!parent: match self",
			lhs:         []traceql.Span{parent1},
			rhs:         []traceql.Span{parent1},
			invert:      true,
			falseForAll: true,
			expected:    []traceql.Span{parent1},
		},
		// &>
		{
			name:     "&child: basic",
			lhs:      []traceql.Span{parent1},
			rhs:      []traceql.Span{child1a, child1b, child2a, child2b},
			expected: []traceql.Span{child1a, child1b, parent1},
			union:    true,
		},
		{
			name:     "&child: multiple matching trees",
			lhs:      []traceql.Span{parent1, parent2},
			rhs:      []traceql.Span{child1a, child1b, child2a, child2b},
			expected: []traceql.Span{child1a, child1b, parent1, child2a, child2b, parent2},
			union:    true,
		},
		{
			name:     "&child: don't match self",
			lhs:      []traceql.Span{parent1},
			rhs:      []traceql.Span{parent1},
			expected: nil,
			union:    true,
		},
		// |<
		{
			name:     "&parent: basic",
			lhs:      []traceql.Span{child1a, child1b, child2a, child2b},
			rhs:      []traceql.Span{parent1},
			invert:   true,
			expected: []traceql.Span{child1a, child1b, parent1},
			union:    true,
		},
		{
			name:     "&parent: multiple matching trees",
			lhs:      []traceql.Span{child1a, child1b, child2a, child2b},
			rhs:      []traceql.Span{parent1, parent2},
			invert:   true,
			expected: []traceql.Span{child1a, child1b, parent1, child2a, child2b, parent2},
			union:    true,
		},
		{
			name:     "&parent: don't match self",
			lhs:      []traceql.Span{parent1},
			rhs:      []traceql.Span{parent1},
			invert:   true,
			expected: nil,
			union:    true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			s := &span{}

			actual := s.ChildOf(tc.lhs, tc.rhs, tc.falseForAll, tc.invert, tc.union, nil)
			require.Equal(t, tc.expected, actual)
		})

		// if !falseForAll we can safe insert disconnected spans and get the same results
		if !tc.falseForAll {
			t.Run(tc.name+"-disconnected", func(t *testing.T) {
				s := &span{}

				lhs := append(tc.lhs, allDisconnected...)
				rhs := append(tc.rhs, allDisconnected...)

				actual := s.ChildOf(lhs, rhs, tc.falseForAll, tc.invert, tc.union, nil)
				require.Equal(t, tc.expected, actual)
			})
		}
	}
}

func TestSiblingOf(t *testing.T) {
	sibling1a := &span{nestedSetParent: 2}
	sibling1b := &span{nestedSetParent: 2}
	sibling2a := &span{nestedSetParent: 4}
	sibling2b := &span{nestedSetParent: 4}
	sibling2c := &span{nestedSetParent: 4}

	disconnectedBefore := &span{nestedSetParent: 1}
	disconnectedBetween := &span{nestedSetParent: 3}
	disconnectedAfter := &span{nestedSetParent: 5}

	allDisconnected := []traceql.Span{disconnectedBefore, disconnectedBetween, disconnectedAfter}

	tcs := []struct {
		name        string
		lhs         []traceql.Span
		rhs         []traceql.Span
		falseForAll bool // !~ or !~
		union       bool
		expected    []traceql.Span
	}{
		{
			name:     "empty",
			lhs:      []traceql.Span{},
			rhs:      []traceql.Span{},
			expected: nil,
		},
		// ~
		{
			name:     "sibling: basic",
			lhs:      []traceql.Span{sibling1a},
			rhs:      []traceql.Span{sibling1b, sibling2a, sibling2b},
			expected: []traceql.Span{sibling1b},
		},
		{
			name:     "sibling: multiple matching trees",
			lhs:      []traceql.Span{sibling1a, sibling1b, sibling2a, sibling2b},
			rhs:      []traceql.Span{sibling1a, sibling1b, sibling2a, sibling2b},
			expected: []traceql.Span{sibling1a, sibling1b, sibling2a, sibling2b},
		},
		{
			name:     "sibling: match self",
			lhs:      []traceql.Span{sibling1a},
			rhs:      []traceql.Span{sibling1a},
			expected: nil,
		},
		// !~
		{
			name:        "!sibling: basic",
			lhs:         []traceql.Span{sibling1a},
			rhs:         []traceql.Span{sibling1b, sibling2a, sibling2b, disconnectedAfter, disconnectedBefore, disconnectedBetween},
			falseForAll: true,
			expected:    []traceql.Span{sibling2a, sibling2b, disconnectedAfter, disconnectedBefore, disconnectedBetween},
		},
		{
			name:        "!sibling: multiple matching trees",
			lhs:         []traceql.Span{sibling1a, sibling1b, sibling2a, sibling2b},
			rhs:         []traceql.Span{sibling1b, sibling2a, sibling2b, disconnectedAfter, disconnectedBefore, disconnectedBetween},
			falseForAll: true,
			expected:    []traceql.Span{disconnectedAfter, disconnectedBefore, disconnectedBetween},
		},
		{
			name:        "!sibling: match self",
			lhs:         []traceql.Span{sibling1a},
			rhs:         []traceql.Span{sibling1a},
			falseForAll: true,
			expected:    []traceql.Span{sibling1a},
		},
		// &~
		{
			name:     "&sibling: basic",
			lhs:      []traceql.Span{sibling1a},
			rhs:      []traceql.Span{sibling1b, sibling2a, sibling2b},
			union:    true,
			expected: []traceql.Span{sibling1b, sibling1a},
		},
		{
			name:     "&sibling: multiple left",
			lhs:      []traceql.Span{sibling2a, sibling2b},
			rhs:      []traceql.Span{sibling2c},
			union:    true,
			expected: []traceql.Span{sibling2a, sibling2b, sibling2c},
		},
		{
			name:     "&sibling: multiple matching trees",
			lhs:      []traceql.Span{sibling1a, sibling1b, sibling2a, sibling2b},
			rhs:      []traceql.Span{sibling1b, sibling2a, sibling2b},
			union:    true,
			expected: []traceql.Span{sibling1a, sibling1b, sibling2a, sibling2b},
		},
		{
			name:     "&sibling: match self",
			lhs:      []traceql.Span{sibling1a},
			rhs:      []traceql.Span{sibling1a},
			union:    true,
			expected: nil,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			s := &span{}

			actual := s.SiblingOf(tc.lhs, tc.rhs, tc.falseForAll, tc.union, nil)
			require.Equal(t, tc.expected, actual)
		})

		// if !falseForAll we can safe insert disconnected spans and get the same results
		if !tc.falseForAll {
			t.Run(tc.name+"-disconnected-lhs", func(t *testing.T) {
				s := &span{}

				lhs := append(tc.lhs, allDisconnected...)

				actual := s.SiblingOf(lhs, tc.rhs, tc.falseForAll, tc.union, nil)
				require.Equal(t, tc.expected, actual)
			})

			t.Run(tc.name+"-disconnected-rhs", func(t *testing.T) {
				s := &span{}

				rhs := append(tc.rhs, allDisconnected...)

				actual := s.SiblingOf(tc.lhs, rhs, tc.falseForAll, tc.union, nil)
				require.Equal(t, tc.expected, actual)
			})
		}
	}
}

func TestStructuralSameSlice(t *testing.T) {
	root := &span{nestedSetLeft: 1, nestedSetRight: 10, nestedSetParent: -1}

	parent1 := &span{nestedSetLeft: 2, nestedSetRight: 9, nestedSetParent: 1}
	child1a := &span{nestedSetLeft: 3, nestedSetRight: 6, nestedSetParent: 2}
	child1aa := &span{nestedSetLeft: 4, nestedSetRight: 5, nestedSetParent: 3}
	child1b := &span{nestedSetLeft: 7, nestedSetRight: 8, nestedSetParent: 2}

	all := []traceql.Span{root, parent1, child1a, child1aa, child1b}

	expectedChildOf := []traceql.Span{parent1, child1a, child1aa, child1b}
	exepectedDescendantOf := []traceql.Span{child1b, child1aa, child1a, parent1}
	expectedSiblingOf := []traceql.Span{child1b, child1a}

	actualChildOf := child1a.ChildOf(all, all, false, false, false, nil)
	require.Equal(t, expectedChildOf, actualChildOf)

	actualDescendantOf := child1a.DescendantOf(all, all, false, false, false, nil)
	require.Equal(t, exepectedDescendantOf, actualDescendantOf)

	actualSiblingOf := child1a.SiblingOf(all, all, false, false, nil)
	require.Equal(t, expectedSiblingOf, actualSiblingOf)
}

func BenchmarkDescendantOf(b *testing.B) {
	for _, count := range []int{10, 100, 1000, 10000} {
		b.Run(fmt.Sprintf("%d", count), func(b *testing.B) {
			totalSpans := count

			// create 1k s1 in a direct line
			s1 := randomTree(totalSpans)
			// copy the same slice to s2
			s2 := make([]traceql.Span, totalSpans)
			copy(s2, s1)

			for _, tc := range []struct {
				name        string
				falseForAll bool
				invert      bool
				union       bool
			}{
				{
					name: ">>",
				},
				{
					name:   "<<",
					invert: true,
				},
				{
					name:        "!>>",
					falseForAll: true,
				},
				{
					name:        "!<<",
					falseForAll: true,
					invert:      true,
				},
				{
					name:  "&>>",
					union: true,
				},
				{
					name:   "&<<",
					invert: true,
					union:  true,
				},
			} {
				b.Run(fmt.Sprintf("%s : %d", tc.name, count), func(b *testing.B) {
					s := &span{}

					shuffleSpans(s1)
					shuffleSpans(s2)
					b.ResetTimer()

					for i := 0; i < b.N; i++ {
						s.DescendantOf(s1, s1, tc.falseForAll, tc.invert, tc.union, nil)
					}
				})
			}
		})
	}
}

func BenchmarkSiblingOf(b *testing.B) {
	for _, count := range []int{10, 100, 1000, 10000} {
		b.Run(fmt.Sprintf("%d", count), func(b *testing.B) {
			totalSpans := count

			// create 1k s1 with random siblings
			s1 := make([]traceql.Span, totalSpans)
			for i := 0; i < totalSpans; i++ {
				s1[i] = &span{nestedSetParent: rand.Int31n(10)}
			}
			// copy the same slice to s2
			s2 := make([]traceql.Span, totalSpans)
			copy(s2, s1)

			for _, tc := range []struct {
				name        string
				falseForAll bool
				union       bool
			}{
				{
					name: "~",
				},
				{
					name:        "!~",
					falseForAll: true,
				},
				{
					name:  "&~",
					union: true,
				},
			} {
				b.Run(fmt.Sprintf("%s : %d", tc.name, count), func(b *testing.B) {
					s := &span{}

					shuffleSpans(s1)
					shuffleSpans(s2)
					b.ResetTimer()

					for i := 0; i < b.N; i++ {
						s.SiblingOf(s1, s2, tc.falseForAll, tc.union, nil)
					}
				})
			}
		})
	}
}

func BenchmarkChildOf(b *testing.B) {
	for _, count := range []int{10, 100, 1000, 10000} {
		b.Run(fmt.Sprintf("%d", count), func(b *testing.B) {
			totalSpans := count

			// create 1k s1 in a direct line
			s1 := randomTree(totalSpans)
			// copy the same slice to s2
			s2 := make([]traceql.Span, totalSpans)
			copy(s2, s1)

			for _, tc := range []struct {
				name        string
				falseForAll bool
				invert      bool
				union       bool
			}{
				{
					name: ">",
				},
				{
					name:   "<",
					invert: true,
				},
				{
					name:        "!>",
					falseForAll: true,
				},
				{
					name:        "!<",
					falseForAll: true,
					invert:      true,
				},
				{
					name:  "&>",
					union: true,
				},
				{
					name:   "&<",
					invert: true,
					union:  true,
				},
			} {
				b.Run(fmt.Sprintf("%s : %d", tc.name, count), func(b *testing.B) {
					s := &span{}

					shuffleSpans(s1)
					shuffleSpans(s2)
					b.ResetTimer()

					for i := 0; i < b.N; i++ {
						s.ChildOf(s1, s1, tc.falseForAll, tc.invert, tc.union, nil)
					}
				})
			}
		})
	}
}

func shuffleSpans(spans []traceql.Span) {
	rand.Shuffle(len(spans), func(i, j int) {
		spans[i], spans[j] = spans[j], spans[i]
	})
}

func randomTree(N int) []traceql.Span {
	nodes := make([]traceql.Span, 0, N)

	// Helper function to recursively generate nodes
	var generateNodes func(parent int) int
	generateNodes = func(parent int) int {
		left := parent
		for N > 0 {
			// make sibling
			N--
			left++
			right := left + 1
			nodes = append(nodes, &span{
				nestedSetLeft:   int32(left),
				nestedSetRight:  int32(right),
				nestedSetParent: int32(parent),
			})
			left++

			if rand.Intn(3) > 1 {
				continue // keep making siblings
			}

			if rand.Intn(3) > 1 {
				break // stop making children
			}

			// descend and make children
			N--
			right = generateNodes(left)
			nodes = append(nodes, &span{
				nestedSetLeft:   int32(left),
				nestedSetRight:  int32(right),
				nestedSetParent: int32(parent),
			})
			left = right + 1
		}
		return left
	}

	// Start with root node
	generateNodes(1)

	return nodes
}
