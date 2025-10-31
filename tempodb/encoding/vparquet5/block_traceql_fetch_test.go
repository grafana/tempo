package vparquet5

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/require"
)

func TestStuff(t *testing.T) {
	numTraces := 1
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

	block := makeBackendBlockWithTraces(t, traces)
	ctx := context.Background()
	e := traceql.NewEngine()
	f := block.FetcherFor(common.DefaultSearchOptions()).SpanFetcher()

	req := &tempopb.QueryRangeRequest{
		Query:     "{} | count_over_time() by (resource.service.name)",
		Step:      uint64(time.Minute),
		Start:     1,
		End:       uint64(time.Hour.Nanoseconds()),
		MaxSeries: 1000,
	}

	eval, err := e.CompileMetricsQueryRange(req, 0, 0, false)
	require.NoError(t, err)

	err = eval.DoSpansOnly(ctx, f, 0, 0, int(req.MaxSeries))
	require.NoError(t, err)

	_ = eval.Results()

	fmt.Println(eval.Results())
}

func TestBackendBlockSearchFetchSpansOnly(t *testing.T) {
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
		// Arrays
		{"resource.str-array", traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.str-array = "value-three"}`)},
		{"resource.int-array", traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.int-array = 11}`)},
		{"span.str-array", traceql.MustExtractFetchSpansRequestWithMetadata(`{span.str-array = "value-two"}`)},
		{"span.int-array", traceql.MustExtractFetchSpansRequestWithMetadata(`{span.int-array = 222}`)},
		// Events
		{"event:name", traceql.MustExtractFetchSpansRequestWithMetadata(`{event:name = "e1"}`)},
		{"event:timeSinceStart", traceql.MustExtractFetchSpansRequestWithMetadata(`{event:timeSinceStart > 2ms}`)},
		{"event.message", traceql.MustExtractFetchSpansRequestWithMetadata(`{event.message =~ "exception"}`)},
		// Links
		{"link:spanID", traceql.MustExtractFetchSpansRequestWithMetadata(`{link:spanID = "1234567890abcdef"}`)},
		{"link:traceID", traceql.MustExtractFetchSpansRequestWithMetadata(`{link:traceID = "1234567890abcdef1234567890abcdef"}`)},
		{"link.opentracing.ref_type", traceql.MustExtractFetchSpansRequestWithMetadata(`{link.opentracing.ref_type = "child-of"}`)},
		// Instrumentation Scope
		{"instrumentation:name", traceql.MustExtractFetchSpansRequestWithMetadata(`{instrumentation:name = "scope-1"}`)},
		{"instrumentation:version", traceql.MustExtractFetchSpansRequestWithMetadata(`{instrumentation:version = "version-1"}`)},
		{"instrumentation.attr-str", traceql.MustExtractFetchSpansRequestWithMetadata(`{instrumentation.scope-attr-str = "scope-attr-1"}`)},
		// Operations containing nil
		{".foo != nil", traceql.MustExtractFetchSpansRequestWithMetadata(`{.foo != nil}`)},
		{"nil != .foo", traceql.MustExtractFetchSpansRequestWithMetadata(`{nil != .foo}`)},
		{"span.http.status_code != nil", traceql.MustExtractFetchSpansRequestWithMetadata(`{span.http.status_code != nil}`)},
		{"nil != span.http.status_code", traceql.MustExtractFetchSpansRequestWithMetadata(`{nil != span.http.status_code}`)},
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
			parse(t, `{.`+LabelHTTPStatusCode+` = 500}`),
			parse(t, `{.`+LabelHTTPStatusCode+` > 500}`),
		)},
		{
			"Mix of duration with other conditions", makeReq(
				parse(t, `{`+LabelName+` = "hello"}`),   // Match
				parse(t, `{`+LabelDuration+` < 100s }`), // No match
			),
		},
		// Edge cases
		{"Almost conflicts with intrinsic but still works", traceql.MustExtractFetchSpansRequestWithMetadata(`{.name = "Bob"}`)},
		{"service.name doesn't match type of dedicated column", traceql.MustExtractFetchSpansRequestWithMetadata(`{resource.` + LabelServiceName + ` = 123}`)},
		{"service.name present on span", traceql.MustExtractFetchSpansRequestWithMetadata(`{.` + LabelServiceName + ` = "spanservicename"}`)},
		{`.foo = "def"`, traceql.MustExtractFetchSpansRequestWithMetadata(`{.foo = "def"}`)},
		{".bool && true", traceql.MustExtractFetchSpansRequestWithMetadata(`{.bool && true}`)},
		{"false || .bool", traceql.MustExtractFetchSpansRequestWithMetadata(`{false || .bool}`)},
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

			resp, err := b.FetchSpansOnly(ctx, req, common.DefaultSearchOptions())
			require.NoError(t, err, "search request:%v", req)

			found := false
			for {
				span, err := resp.Results.Next(ctx)
				require.NoError(t, err, "search request:%v", req)
				if span == nil {
					break
				}
				traceID, ok := span.AttributeFor(traceql.IntrinsicTraceIDAttribute)
				if !ok {
					continue
				}
				traceIDString := traceID.EncodeToString(false)
				// fmt.Println("got:", traceIDString, "want:", traceIDText)
				found = (traceIDString == traceIDText)
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

			resp, err := b.FetchSpansOnly(ctx, req, common.DefaultSearchOptions())
			require.NoError(t, err, "search request:%v", req)

			for {
				span, err := resp.Results.Next(ctx)
				require.NoError(t, err, "search request:%v", req)
				if span == nil {
					break
				}
				traceID, ok := span.AttributeFor(traceql.IntrinsicTraceIDAttribute)
				if !ok {
					continue
				}
				traceIDString := traceID.EncodeToString(false)
				require.NotEqual(t, traceIDText, traceIDString, "search request:%v", req)
			}
		})
	}
}

func BenchmarkQueryRangeSpansOnly(b *testing.B) {
	testCases := []string{
		/*"{} | rate()",
		"{} | rate() with(sample=true)",
		"{} | rate() by (span.http.status_code)",
		"{} | rate() by (resource.service.name)",
		"{} | rate() by (span.http.url)", // High cardinality attribute
		"{resource.service.name=`loki-ingester`} | rate()",
		"{span.http.host != `` && span.http.flavor=`2`} | rate() by (span.http.flavor)", // Multiple conditions
		"{status=error} | rate()",
		"{} | quantile_over_time(duration, .99, .9, .5)",
		"{} | quantile_over_time(duration, .99) by (span.http.status_code)",
		"{} | histogram_over_time(duration)",
		"{} | avg_over_time(duration) by (span.http.status_code)",
		"{} | max_over_time(duration) by (span.http.status_code)",
		"{} | min_over_time(duration) by (span.http.status_code)",*/
		"{ name != nil } | compare({status=error})",
	}

	// For sampler debugging
	log.Logger = kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stderr))

	e := traceql.NewEngine()
	ctx := context.TODO()
	opts := common.DefaultSearchOptions()

	block := blockForBenchmarks(b)

	f := block.FetcherFor(opts).SpanFetcher()

	for _, tc := range testCases {
		b.Run(tc, func(b *testing.B) {
			st := uint64(block.meta.StartTime.UnixNano())
			end := uint64(block.meta.EndTime.UnixNano())

			req := &tempopb.QueryRangeRequest{
				Query:     tc,
				Step:      uint64(time.Minute),
				Start:     st,
				End:       end,
				MaxSeries: 1000,
			}

			eval, err := e.CompileMetricsQueryRange(req, 0, 0, false)
			require.NoError(b, err)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err := eval.DoSpansOnly(ctx, f, st, end, int(req.MaxSeries))
				require.NoError(b, err)
			}

			_ = eval.Results()

			bytes, spansTotal, _ := eval.Metrics()
			b.ReportMetric(float64(bytes)/float64(b.N)/1024.0/1024.0, "MB_IO/op")
			b.ReportMetric(float64(spansTotal)/float64(b.N), "spans/op")
			b.ReportMetric(float64(spansTotal)/b.Elapsed().Seconds(), "spans/s")
		})
	}
}
