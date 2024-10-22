package vparquet4

import (
	"context"
	"fmt"
	"path"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/collector"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/require"
)

func TestFetchTagNames(t *testing.T) {
	testCases := []struct {
		name                          string
		query                         string
		expectedSpanValues            []string
		expectedResourceValues        []string
		expectedEventValues           []string
		expectedLinkValues            []string
		expectedInstrumentationValues []string
	}{
		{
			name:  "no query - fall back to old search",
			query: "{}",
			expectedSpanValues: []string{
				"generic-01-01",
				"generic-01-02",
				"generic-02-01",
				"span-same",
			},
			expectedResourceValues:        []string{"generic-01", "generic-02", "resource-same"},
			expectedEventValues:           []string{"event-generic-01-01", "event-generic-02-01"},
			expectedLinkValues:            []string{"link-generic-01-01", "link-generic-02-01"},
			expectedInstrumentationValues: []string{"scope-attr-str-1", "scope-attr-str-2"},
		},
		{
			name:                          "matches nothing",
			query:                         "{span.generic-01-01=`bar`}",
			expectedSpanValues:            []string{},
			expectedResourceValues:        []string{},
			expectedEventValues:           []string{},
			expectedLinkValues:            []string{},
			expectedInstrumentationValues: []string{},
		},
		// span
		{
			name:                          "intrinsic span",
			query:                         "{statusMessage=`msg-01-01`}",
			expectedSpanValues:            []string{"generic-01-01", "span-same"},
			expectedResourceValues:        []string{"generic-01", "resource-same"},
			expectedEventValues:           []string{"event-generic-01-01"},
			expectedLinkValues:            []string{"link-generic-01-01"},
			expectedInstrumentationValues: []string{"scope-attr-str-1"},
		},
		{
			name:                          "well known span",
			query:                         "{span.http.method=`method-01-01`}",
			expectedSpanValues:            []string{"generic-01-01", "span-same"},
			expectedResourceValues:        []string{"generic-01", "resource-same"},
			expectedEventValues:           []string{"event-generic-01-01"},
			expectedLinkValues:            []string{"link-generic-01-01"},
			expectedInstrumentationValues: []string{"scope-attr-str-1"},
		},
		{
			name:                          "generic span",
			query:                         "{span.generic-01-01=`foo`}",
			expectedSpanValues:            []string{"generic-01-01", "span-same"},
			expectedResourceValues:        []string{"generic-01", "resource-same"},
			expectedEventValues:           []string{"event-generic-01-01"},
			expectedLinkValues:            []string{"link-generic-01-01"},
			expectedInstrumentationValues: []string{"scope-attr-str-1"},
		},
		{
			name:                          "match two spans",
			query:                         "{span.span-same=`foo`}",
			expectedSpanValues:            []string{"generic-01-01", "span-same", "generic-02-01"},
			expectedResourceValues:        []string{"generic-01", "resource-same", "generic-02"},
			expectedEventValues:           []string{"event-generic-01-01", "event-generic-02-01"},
			expectedLinkValues:            []string{"link-generic-01-01", "link-generic-02-01"},
			expectedInstrumentationValues: []string{"scope-attr-str-1", "scope-attr-str-2"},
		},
		// resource
		{
			name:                          "well known resource",
			query:                         "{resource.cluster=`cluster-01`}",
			expectedSpanValues:            []string{"generic-01-01", "generic-01-02", "span-same"},
			expectedResourceValues:        []string{"generic-01", "resource-same"},
			expectedEventValues:           []string{"event-generic-01-01"},
			expectedLinkValues:            []string{"link-generic-01-01"},
			expectedInstrumentationValues: []string{"scope-attr-str-1"},
		},
		{
			name:                          "generic resource",
			query:                         "{resource.generic-01=`bar`}",
			expectedSpanValues:            []string{"generic-01-01", "generic-01-02", "span-same"},
			expectedResourceValues:        []string{"generic-01", "resource-same"},
			expectedEventValues:           []string{"event-generic-01-01"},
			expectedLinkValues:            []string{"link-generic-01-01"},
			expectedInstrumentationValues: []string{"scope-attr-str-1"},
		},
		{
			name:                          "match two resources",
			query:                         "{resource.resource-same=`foo`}",
			expectedSpanValues:            []string{"generic-01-01", "generic-01-02", "span-same", "generic-02-01"},
			expectedResourceValues:        []string{"generic-01", "resource-same", "generic-02"},
			expectedEventValues:           []string{"event-generic-01-01", "event-generic-02-01"},
			expectedLinkValues:            []string{"link-generic-01-01", "link-generic-02-01"},
			expectedInstrumentationValues: []string{"scope-attr-str-1", "scope-attr-str-2"},
		},
		// trace level match
		{
			name:                          "trace",
			query:                         "{rootName=`root` }",
			expectedSpanValues:            []string{"generic-01-01", "generic-01-02", "span-same", "generic-02-01"},
			expectedResourceValues:        []string{"generic-01", "resource-same", "generic-02"},
			expectedEventValues:           []string{"event-generic-01-01", "event-generic-02-01"},
			expectedLinkValues:            []string{"link-generic-01-01", "link-generic-02-01"},
			expectedInstrumentationValues: []string{"scope-attr-str-1", "scope-attr-str-2"},
		},
	}

	strPtr := func(s string) *string { return &s }
	tr := &Trace{
		TraceID:         test.ValidTraceID(nil),
		RootServiceName: "tr",
		RootSpanName:    "root",
		ResourceSpans: []ResourceSpans{
			{
				Resource: Resource{
					ServiceName: "svc-01",
					Cluster:     strPtr("cluster-01"), // well known
					Attrs: []Attribute{
						{Key: "generic-01", Value: []string{"bar"}}, // generic
						{Key: "resource-same", Value: []string{"foo"}},
					},
					DedicatedAttributes: DedicatedAttributes{
						String01: strPtr("dedicated-01"),
					},
				},
				ScopeSpans: []ScopeSpans{
					{
						Scope: InstrumentationScope{
							Name:                   "scope-1",
							Version:                "version-1",
							DroppedAttributesCount: 1,
							Attrs: []Attribute{
								attr("scope-attr-str-1", "scope-attr-1"),
							},
						},
						Spans: []Span{
							{
								SpanID:        []byte("0101"),
								Name:          "span-01-01",
								HttpMethod:    strPtr("method-01-01"), // well known
								StatusMessage: "msg-01-01",            // intrinsic
								Attrs: []Attribute{
									{Key: "generic-01-01", Value: []string{"foo"}}, // generic
									{Key: "span-same", Value: []string{"foo"}},     // generic
								},
								DedicatedAttributes: DedicatedAttributes{
									String01: strPtr("dedicated-01-01"),
								},
								Events: []Event{
									{
										Name: "event-01-01",
										Attrs: []Attribute{
											{Key: "event-generic-01-01", Value: []string{"foo"}},
										},
									},
								},
								Links: []Link{
									{
										SpanID: []byte("0101"),
										Attrs: []Attribute{
											{Key: "link-generic-01-01", Value: []string{"foo"}},
										},
									},
								},
							},
							{
								SpanID:        []byte("0102"),
								Name:          "span-01-02",
								HttpMethod:    strPtr("method-01-02"), // well known
								StatusMessage: "msg-01-02",            // intrinsic
								Attrs: []Attribute{
									{Key: "generic-01-02", Value: []string{"foo"}}, // generic
								},
								DedicatedAttributes: DedicatedAttributes{
									String01: strPtr("dedicated-01-02"),
								},
							},
						},
					},
				},
			},
			{
				Resource: Resource{
					ServiceName: "svc-02",
					Cluster:     strPtr("cluster-02"), // well known
					Attrs: []Attribute{
						{Key: "generic-02", Value: []string{"bar"}}, // generic
						{Key: "resource-same", Value: []string{"foo"}},
					},
					DedicatedAttributes: DedicatedAttributes{
						String01: strPtr("dedicated-02"),
					},
				},
				ScopeSpans: []ScopeSpans{
					{
						Scope: InstrumentationScope{
							Name:                   "scope-2",
							Version:                "version-2",
							DroppedAttributesCount: 1,
							Attrs: []Attribute{
								attr("scope-attr-str-2", "scope-attr-2"),
							},
						},
						Spans: []Span{
							{
								SpanID:        []byte("0201"),
								Name:          "span-02-01",
								HttpMethod:    strPtr("method-02-01"), // well known
								StatusMessage: "msg-02-01",            // intrinsic
								Attrs: []Attribute{
									{Key: "generic-02-01", Value: []string{"foo"}}, // generic
									{Key: "span-same", Value: []string{"foo"}},     // generic
								},
								DedicatedAttributes: DedicatedAttributes{
									String01: strPtr("dedicated-02-01"),
								},
								Events: []Event{
									{
										Name: "event-02-01",
										Attrs: []Attribute{
											{Key: "event-generic-02-01", Value: []string{"foo"}},
										},
									},
								},
								Links: []Link{
									{
										SpanID: []byte("0102"),
										Attrs: []Attribute{
											{Key: "link-generic-02-01", Value: []string{"foo"}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ctx := context.TODO()
	block := makeBackendBlockWithTraces(t, []*Trace{tr})

	opts := common.DefaultSearchOptions()

	for _, tc := range testCases {
		for _, scope := range []traceql.AttributeScope{
			traceql.AttributeScopeSpan,
			traceql.AttributeScopeResource,
			traceql.AttributeScopeNone,
			traceql.AttributeScopeEvent,
			traceql.AttributeScopeLink,
			traceql.AttributeScopeInstrumentation,
		} {
			expectedSpanValues := tc.expectedSpanValues
			expectedResourceValues := tc.expectedResourceValues
			expectedEventValues := tc.expectedEventValues
			expectedLinkValues := tc.expectedLinkValues
			expectedInstrumentationValues := tc.expectedInstrumentationValues

			// add dedicated and well known columns to expected values. the code currently does not
			// attempt to perfectly filter these, but instead adds them to the return if any values are present
			dedicatedSpanValues := []string{"dedicated.span.1"}
			dedicatedResourceValues := []string{"dedicated.resource.1"}

			wellKnownSpanValues := []string{"http.method"}
			wellKnownResourceValues := []string{"cluster", "service.name"}

			expectedValues := map[string][]string{}
			if scope == traceql.AttributeScopeSpan || scope == traceql.AttributeScopeNone {
				expectedValues["span"] = append(expectedValues["span"], expectedSpanValues...)
				expectedValues["span"] = append(expectedValues["span"], wellKnownSpanValues...)
				expectedValues["span"] = append(expectedValues["span"], dedicatedSpanValues...)
			}
			if scope == traceql.AttributeScopeResource || scope == traceql.AttributeScopeNone {
				expectedValues["resource"] = append(expectedValues["resource"], expectedResourceValues...)
				expectedValues["resource"] = append(expectedValues["resource"], wellKnownResourceValues...)
				expectedValues["resource"] = append(expectedValues["resource"], dedicatedResourceValues...)
			}
			if scope == traceql.AttributeScopeEvent || scope == traceql.AttributeScopeNone {
				if len(expectedEventValues) > 0 {
					expectedValues["event"] = append(expectedValues["event"], expectedEventValues...)
				}
			}
			if scope == traceql.AttributeScopeLink || scope == traceql.AttributeScopeNone {
				if len(expectedLinkValues) > 0 {
					expectedValues["link"] = append(expectedValues["link"], expectedLinkValues...)
				}
			}

			if scope == traceql.AttributeScopeInstrumentation || scope == traceql.AttributeScopeNone {
				if len(expectedInstrumentationValues) > 0 {
					expectedValues["instrumentation"] = append(expectedValues["instrumentation"], expectedInstrumentationValues...)
				}
			}

			t.Run(fmt.Sprintf("query: %s %s-%s", tc.name, tc.query, scope), func(t *testing.T) {
				distinctAttrNames := collector.NewScopedDistinctString(0)
				req, err := traceql.ExtractFetchSpansRequest(tc.query)
				require.NoError(t, err)

				// Build autocomplete request
				autocompleteReq := traceql.FetchTagsRequest{
					Conditions: req.Conditions,
					Scope:      scope,
				}
				mc := collector.NewMetricsCollector()

				err = block.FetchTagNames(ctx, autocompleteReq, func(t string, scope traceql.AttributeScope) bool {
					distinctAttrNames.Collect(scope.String(), t)
					return false
				}, mc.Add, opts)
				require.NoError(t, err)
				// test that callback is recording bytes read
				require.Greater(t, mc.TotalValue(), uint64(100))

				actualValues := distinctAttrNames.Strings()

				require.Equal(t, len(expectedValues), len(actualValues))
				for k := range expectedValues {
					actual := actualValues[k]
					sort.Strings(actual)
					expected := expectedValues[k]
					sort.Strings(expected)

					require.Equal(t, expected, actual, "scope: %s", k)
				}
			})
		}
	}
}

func TestFetchTagValues(t *testing.T) {
	testCases := []struct {
		name           string
		tag, query     string
		expectedValues []tempopb.TagValue
	}{
		{
			name:  "intrinsic with no query - match",
			tag:   "name",
			query: "{}",
			expectedValues: []tempopb.TagValue{
				stringTagValue("hello"),
				stringTagValue("world"),
			},
		},
		{
			name:           "intrinsic with resource attribute - match",
			tag:            "name",
			query:          `{resource.namespace="namespace"}`,
			expectedValues: []tempopb.TagValue{stringTagValue("hello")},
		},
		{
			name:           "intrinsic with span attribute - match",
			tag:            "name",
			query:          `{span.foo="def"}`,
			expectedValues: []tempopb.TagValue{stringTagValue("hello")},
		},
		{
			name:           "intrinsic with span attribute and resource attribute - match",
			tag:            "name",
			query:          `{span.foo="def" && resource.namespace="namespace"}`,
			expectedValues: []tempopb.TagValue{stringTagValue("hello")},
		},
		{
			name:           "intrinsic with intrinsic attribute - match",
			tag:            "name",
			query:          `{kind=client}`,
			expectedValues: []tempopb.TagValue{stringTagValue("hello")},
		},
		{
			name:           "intrinsic with resource attribute - no match",
			tag:            "name",
			query:          `{resource.namespace="namespace3"}`,
			expectedValues: []tempopb.TagValue{},
		},
		{
			name:           "intrinsic with span attribute - no match",
			tag:            "name",
			query:          `{span.foo="jkl"}`,
			expectedValues: []tempopb.TagValue{},
		},
		{
			name:           "intrinsic with span attribute and resource attribute - no match",
			tag:            "name",
			query:          `{span.foo="jkl" && resource.namespace="namespace3"}`,
			expectedValues: []tempopb.TagValue{},
		},
		{
			name:           "intrinsic with intrinsic attribute - no match",
			tag:            "name",
			query:          `{kind=internal}`,
			expectedValues: []tempopb.TagValue{},
		},
		{
			name:           "resource attribute with no query - match",
			tag:            "resource.service.name",
			query:          `{}`,
			expectedValues: []tempopb.TagValue{stringTagValue("myservice"), stringTagValue("service2")},
		},
		{
			name:           "resource attribute with resource attribute - match",
			tag:            "resource.service.name",
			query:          `{resource.namespace="namespace"}`,
			expectedValues: []tempopb.TagValue{stringTagValue("myservice")},
		},
		{
			name:           "resource attribute with span attribute - match",
			tag:            "resource.service.name",
			query:          `{span.foo="def"}`,
			expectedValues: []tempopb.TagValue{stringTagValue("myservice")},
		},
		{
			name:           "resource attribute with span attribute and resource attribute - match",
			tag:            "resource.service.name",
			query:          `{span.foo="def" && resource.namespace="namespace"}`,
			expectedValues: []tempopb.TagValue{stringTagValue("myservice")},
		},
		{
			name:           "resource attribute with intrinsic attribute - match",
			tag:            "resource.service.name",
			query:          `{kind=client}`,
			expectedValues: []tempopb.TagValue{stringTagValue("myservice")},
		},
		{
			name:           "resource attribute with resource attribute - no match",
			tag:            "resource.service.name",
			query:          `{resource.namespace="namespace3"}`,
			expectedValues: []tempopb.TagValue{},
		},
		{
			name:           "resource attribute with span attribute - no match",
			tag:            "resource.service.name",
			query:          `{span.foo="jkl"}`,
			expectedValues: []tempopb.TagValue{},
		},
		{
			name:           "resource attribute with span attribute and resource attribute - no match",
			tag:            "resource.service.name",
			query:          `{span.foo="jkl" && resource.namespace="namespace3"}`,
			expectedValues: []tempopb.TagValue{},
		},
		{
			name:           "resource attribute with intrinsic attribute - no match",
			tag:            "resource.service.name",
			query:          `{kind=internal}`,
			expectedValues: []tempopb.TagValue{},
		},
		{
			name:           "span attribute with no query - match",
			tag:            "span.foo",
			query:          `{}`,
			expectedValues: []tempopb.TagValue{stringTagValue("def"), stringTagValue("ghi")},
		},
		{
			name:           "span attribute with resource attribute - match",
			tag:            "span.foo",
			query:          `{resource.namespace="namespace"}`,
			expectedValues: []tempopb.TagValue{stringTagValue("def")},
		},
		{
			name:           "span attribute with span attribute - match",
			tag:            "span.foo",
			query:          `{span.bar=123}`,
			expectedValues: []tempopb.TagValue{stringTagValue("def")},
		},
		{
			name:           "span attribute with span attribute and resource attribute - match",
			tag:            "span.foo",
			query:          `{span.bool=false && resource.namespace="namespace"}`,
			expectedValues: []tempopb.TagValue{stringTagValue("def")},
		},
		{
			name:           "span attribute with intrinsic attribute - match",
			tag:            "span.foo",
			query:          `{kind=client}`,
			expectedValues: []tempopb.TagValue{stringTagValue("def")},
		},
		{
			name:           "span attribute with resource attribute - no match",
			tag:            "span.foo",
			query:          `{resource.namespace="namespace3"}`,
			expectedValues: []tempopb.TagValue{},
		},
		{
			name:           "span attribute with span attribute - no match",
			tag:            "span.foo",
			query:          `{span.foo="jkl"}`,
			expectedValues: []tempopb.TagValue{},
		},
		{
			name:           "span attribute with span attribute and resource attribute - no match",
			tag:            "span.foo",
			query:          `{span.foo="jkl" && resource.namespace="namespace3"}`,
			expectedValues: []tempopb.TagValue{},
		},
		{
			name:           "span attribute with intrinsic attribute - no match",
			tag:            "span.foo",
			query:          `{kind=internal}`,
			expectedValues: []tempopb.TagValue{},
		},
		{
			name:           "trace intrinsic attribute with no query - match",
			tag:            "rootName",
			query:          `{}`,
			expectedValues: []tempopb.TagValue{stringTagValue("RootSpan")},
		},
		{
			name:           "trace intrinsic attribute with resource attribute - match",
			tag:            "rootName",
			query:          `{resource.namespace="namespace"}`,
			expectedValues: []tempopb.TagValue{stringTagValue("RootSpan")},
		},
		{
			name:           "trace intrinsic attribute with span attribute - match",
			tag:            "rootName",
			query:          `{span.foo="def"}`,
			expectedValues: []tempopb.TagValue{stringTagValue("RootSpan")},
		},
		{
			name:           "trace intrinsic attribute with span attribute and resource attribute - match",
			tag:            "rootName",
			query:          `{span.foo="def" && resource.namespace="namespace"}`,
			expectedValues: []tempopb.TagValue{stringTagValue("RootSpan")},
		},
		{
			name:           "trace intrinsic attribute with intrinsic attribute - match",
			tag:            "rootName",
			query:          `{kind=client}`,
			expectedValues: []tempopb.TagValue{stringTagValue("RootSpan")},
		},
		{
			name:           "trace intrinsic attribute with resource attribute - no match",
			tag:            "rootName",
			query:          `{resource.namespace="namespace3"}`,
			expectedValues: []tempopb.TagValue{},
		},
		{
			name:           "trace intrinsic attribute with span attribute - no match",
			tag:            "rootName",
			query:          `{span.foo="jkl"}`,
			expectedValues: []tempopb.TagValue{},
		},
		{
			name:           "trace intrinsic attribute with span attribute and resource attribute - no match",
			tag:            "rootName",
			query:          `{span.foo="jkl" && resource.namespace="namespace3"}`,
			expectedValues: []tempopb.TagValue{},
		},
		{
			name:           "trace intrinsic attribute with intrinsic attribute - no match",
			tag:            "rootName",
			query:          `{kind=internal}`,
			expectedValues: []tempopb.TagValue{},
		},
		{
			name:           "unscoped attribute - not supported",
			tag:            ".service.name",
			query:          `{ .namespace="namespace"}`,
			expectedValues: []tempopb.TagValue{intTagValue(123), intTagValue(1234), stringTagValue("myservice"), stringTagValue("service2"), stringTagValue("spanservicename"), stringTagValue("spanservicename2")},
		},
		{
			name:  "query with wrong op types - conditions are ignored",
			tag:   "status",
			query: `{resource.service.name="myservice" && span.http.status_code=server && resource.namespace=server}`,
			expectedValues: []tempopb.TagValue{
				{Type: "keyword", Value: "error"},
			},
		},
		{
			name:  "event attribute - match",
			tag:   "event.message",
			query: `{resource.service.name="myservice"}`,
			expectedValues: []tempopb.TagValue{
				stringTagValue("exception"),
			},
		},
		{
			name:  "link attribute - match",
			tag:   "link.opentracing.ref_type",
			query: `{span.bar=123}`,
			expectedValues: []tempopb.TagValue{
				stringTagValue("child-of"),
			},
		},
	}

	ctx := context.TODO()
	block := makeBackendBlockWithTraces(t, []*Trace{fullyPopulatedTestTrace(common.ID{0})})

	opts := common.DefaultSearchOptions()

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("tag: %s, query: %s", tc.tag, tc.query), func(t *testing.T) {
			distinctValues := collector.NewDistinctValue[tempopb.TagValue](1_000_000, func(v tempopb.TagValue) int { return len(v.Type) + len(v.Value) })
			req, err := traceql.ExtractFetchSpansRequest(tc.query)
			require.NoError(t, err)

			tag, err := traceql.ParseIdentifier(tc.tag)
			require.NoError(t, err)

			// Build autocomplete request
			autocompleteReq := traceql.FetchTagValuesRequest{
				Conditions: req.Conditions,
				TagName:    tag,
			}

			tagAtrr, err := traceql.ParseIdentifier(tc.tag)
			require.NoError(t, err)

			autocompleteReq.Conditions = append(autocompleteReq.Conditions, traceql.Condition{
				Attribute: tagAtrr,
				Op:        traceql.OpNone,
			})
			mc := collector.NewMetricsCollector()

			err = block.FetchTagValues(ctx, autocompleteReq, traceql.MakeCollectTagValueFunc(distinctValues.Collect), mc.Add, opts)
			require.NoError(t, err)
			// test that callback is recording bytes read
			require.Greater(t, mc.TotalValue(), uint64(100))

			expectedValues := tc.expectedValues
			actualValues := distinctValues.Values()
			sort.Slice(expectedValues, func(i, j int) bool { return tc.expectedValues[i].Value < tc.expectedValues[j].Value })
			sort.Slice(actualValues, func(i, j int) bool { return actualValues[i].Value < actualValues[j].Value })
			require.Equal(t, expectedValues, actualValues)
		})
	}
}

func stringTagValue(v string) tempopb.TagValue { return tempopb.TagValue{Type: "string", Value: v} }
func intTagValue(v int64) tempopb.TagValue {
	return tempopb.TagValue{Type: "int", Value: fmt.Sprintf("%d", v)}
}

func BenchmarkFetchTagValues(b *testing.B) {
	testCases := []struct {
		tag   string
		query string
	}{
		{
			tag:   "span.http.url", // well known column
			query: `{resource.namespace="tempo-ops"}`,
		},
		{
			tag:   "span.component", // normal column
			query: `{resource.namespace="tempo-ops"}`,
		},
		{
			tag:   "span.http.url",
			query: `{resource.namespace="tempo-ops" && span.http.status_code=200}`,
		},
		{
			tag:   "resource.namespace",
			query: `{span.http.status_code=200}`,
		},
		// pathologic cases
		/*
			{
				tag:   "resource.k8s.node.name",
				query: `{span.http.method="GET"}`,
			},
			{
				tag:   "span.sampler.type",
				query: `{span.http.method="GET"}`,
			},
			{
				tag:   "span.sampler.type",
				query: `{resource.k8s.node.name>"aaa"}`,
			},
			{
				tag:   "resource.k8s.node.name",
				query: `{span.sampler.type>"aaa"}`,
			},
		*/
	}

	ctx := context.TODO()
	tenantID := "1"
	// blockID := uuid.MustParse("3685ee3d-cbbf-4f36-bf28-93447a19dea6")
	blockID := uuid.MustParse("00145f38-6058-4e57-b1ba-334db8edce23")

	r, _, _, err := local.New(&local.Config{
		// Path: path.Join("/Users/marty/src/tmp/"),
		Path: path.Join("/Users/joe/testblock"),
	})
	require.NoError(b, err)

	rr := backend.NewReader(r)
	meta, err := rr.BlockMeta(ctx, blockID, tenantID)
	require.NoError(b, err)

	block := newBackendBlock(meta, rr)
	opts := common.DefaultSearchOptions()

	for _, tc := range testCases {
		b.Run(fmt.Sprintf("tag: %s, query: %s", tc.tag, tc.query), func(b *testing.B) {
			distinctValues := collector.NewDistinctValue[tempopb.TagValue](1_000_000, func(v tempopb.TagValue) int { return len(v.Type) + len(v.Value) })
			req, err := traceql.ExtractFetchSpansRequest(tc.query)
			require.NoError(b, err)

			tag, err := traceql.ParseIdentifier(tc.tag)
			require.NoError(b, err)

			// FetchTagValues expects the tag to be in the conditions with OpNone otherwise it will
			// fall back to the old tag search
			req.Conditions = append(req.Conditions, traceql.Condition{
				Attribute: tag,
			})

			autocompleteReq := traceql.FetchTagValuesRequest{
				Conditions: req.Conditions,
				TagName:    tag,
			}
			mc := collector.NewMetricsCollector()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				err := block.FetchTagValues(ctx, autocompleteReq, traceql.MakeCollectTagValueFunc(distinctValues.Collect), mc.Add, opts)
				require.NoError(b, err)
			}
		})
	}
}

func BenchmarkFetchTags(b *testing.B) {
	testCases := []struct {
		query string
	}{
		{
			query: `{resource.namespace="tempo-ops"}`, // well known/dedicated column
		},
		{
			query: `{resource.k8s.node.name>"h"}`, // generic attribute
		},
		{
			query: `{span.http.status_code=200}`, // well known/dedicated column
		},
		{
			query: `{nestedSetParent=-1}`, // generic attribute
		},
		{
			query: `{rootName="Memcache.Put"}`, // trace level
		},
		// pathological cases
		/*
			{
				query: `{resource.k8s.node.name>"aaa"}`, // generic attribute
			},
			{
				query: `{span.http.method="GET"}`, // well known/dedicated column
			},
			{
				query: `{span.sampler.type>"aaa"}`, // generic attribute
			},
		*/
	}

	ctx := context.TODO()
	tenantID := "1"
	// blockID := uuid.MustParse("3685ee3d-cbbf-4f36-bf28-93447a19dea6")
	blockID := uuid.MustParse("00145f38-6058-4e57-b1ba-334db8edce23")

	r, _, _, err := local.New(&local.Config{
		// Path: path.Join("/Users/marty/src/tmp/"),
		Path: path.Join("/Users/joe/testblock"),
	})
	require.NoError(b, err)

	rr := backend.NewReader(r)
	meta, err := rr.BlockMeta(ctx, blockID, tenantID)
	require.NoError(b, err)

	block := newBackendBlock(meta, rr)
	opts := common.DefaultSearchOptions()

	for _, tc := range testCases {
		for _, scope := range []traceql.AttributeScope{traceql.AttributeScopeSpan, traceql.AttributeScopeResource, traceql.AttributeScopeNone} {
			b.Run(fmt.Sprintf("query: %s %s", tc.query, scope), func(b *testing.B) {
				distinctStrings := collector.NewScopedDistinctString(1_000_000)
				req, err := traceql.ExtractFetchSpansRequest(tc.query)
				require.NoError(b, err)

				autocompleteReq := traceql.FetchTagsRequest{
					Conditions: req.Conditions,
					Scope:      scope,
				}
				mc := collector.NewMetricsCollector()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					err := block.FetchTagNames(ctx, autocompleteReq, func(t string, scope traceql.AttributeScope) bool {
						distinctStrings.Collect(scope.String(), t)
						return false
					}, mc.Add, opts)
					require.NoError(b, err)
				}
			})
		}
	}
}
