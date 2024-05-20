package vparquet3

import (
	"context"
	"fmt"
	"path"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/require"
)

func TestFetchTagNames(t *testing.T) {
	testCases := []struct {
		name           string
		query          string
		expectedValues []string
	}{
		{
			name:  "no query - fall back to old search", // jpe - not working. not falling back to old search due to span start time cond?
			query: "{}",
			expectedValues: []string{
				"generic-01",
				"generic-01-01",
				"generic-01-02",
				"generic-02",
				"generic-02-01",
				"span-same",
				"resource-same",
			},
		},
		{
			name:           "matches nothing",
			query:          "{span.generic-01-01=`bar`}",
			expectedValues: []string{},
		},
		// span
		{
			name:           "intrinsic span",
			query:          "{statusMessage=`msg-01-01`}",
			expectedValues: []string{"generic-01", "generic-01-01", "span-same", "resource-same"},
		},
		{
			name:           "well known span",
			query:          "{span.http.method=`method-01-01`}",
			expectedValues: []string{"generic-01", "generic-01-01", "span-same", "resource-same"},
		},
		{
			name:           "generic span",
			query:          "{span.generic-01-01=`foo`}",
			expectedValues: []string{"generic-01", "generic-01-01", "span-same", "resource-same"},
		},
		{
			name:           "match two spans",
			query:          "{span.span-same=`foo`}",
			expectedValues: []string{"generic-01", "generic-01-01", "span-same", "resource-same", "generic-02", "generic-02-01"},
		},
		// resource
		{
			name:           "well known resource",
			query:          "{resource.cluster=`cluster-01`}",
			expectedValues: []string{"generic-01", "generic-01-01", "generic-01-02", "span-same", "resource-same"},
		},
		{
			name:           "generic resource",
			query:          "{resource.generic-01=`bar`}",
			expectedValues: []string{"generic-01", "generic-01-01", "generic-01-02", "span-same", "resource-same"},
		},
		{
			name:           "match two resources",
			query:          "{resource.resource-same=`foo`}",
			expectedValues: []string{"generic-01", "generic-01-01", "generic-01-02", "span-same", "resource-same", "generic-02", "generic-02-01"},
		},
		// trace level match
		{
			name:           "trace",
			query:          "{rootName=`root` }",
			expectedValues: []string{"generic-01", "generic-01-01", "generic-01-02", "span-same", "resource-same", "generic-02", "generic-02-01"},
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
						{Key: "generic-01", Value: strPtr("bar")}, // generic
						{Key: "resource-same", Value: strPtr("foo")},
					},
					DedicatedAttributes: DedicatedAttributes{
						String01: strPtr("dedicated-01"),
					},
				},
				ScopeSpans: []ScopeSpans{
					{
						Spans: []Span{
							{
								SpanID:        []byte("0101"),
								Name:          "span-01-01",
								HttpMethod:    strPtr("method-01-01"), // well known
								StatusMessage: "msg-01-01",            // intrinsic
								Attrs: []Attribute{
									{Key: "generic-01-01", Value: strPtr("foo")}, // generic
									{Key: "span-same", Value: strPtr("foo")},     // generic
								},
								DedicatedAttributes: DedicatedAttributes{
									String01: strPtr("dedicated-01-01"),
								},
							},
							{
								SpanID:        []byte("0102"),
								Name:          "span-01-02",
								HttpMethod:    strPtr("method-01-02"), // well known
								StatusMessage: "msg-01-02",            // intrinsic
								Attrs: []Attribute{
									{Key: "generic-01-02", Value: strPtr("foo")}, // generic
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
						{Key: "generic-02", Value: strPtr("bar")}, // generic
						{Key: "resource-same", Value: strPtr("foo")},
					},
					DedicatedAttributes: DedicatedAttributes{
						String01: strPtr("dedicated-02"),
					},
				},
				ScopeSpans: []ScopeSpans{
					{
						Spans: []Span{
							{
								SpanID:        []byte("0201"),
								Name:          "span-02-01",
								HttpMethod:    strPtr("method-02-01"), // well known
								StatusMessage: "msg-02-01",            // intrinsic
								Attrs: []Attribute{
									{Key: "generic-02-01", Value: strPtr("foo")}, // generic
									{Key: "span-same", Value: strPtr("foo")},     // generic
								},
								DedicatedAttributes: DedicatedAttributes{
									String01: strPtr("dedicated-02-01"),
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
		t.Run(fmt.Sprintf("query: %s", tc.query), func(t *testing.T) {
			distinctAttrNames := util.NewDistinctStringCollector(1_000_000)
			req, err := traceql.ExtractFetchSpansRequest(tc.query)
			require.NoError(t, err)

			// Build autocomplete request
			autocompleteReq := traceql.AutocompleteRequest{
				Conditions: req.Conditions,
				TagName:    traceql.Attribute{},
			}

			err = block.FetchTagNames(ctx, autocompleteReq, distinctAttrNames.Collect, opts)
			require.NoError(t, err)

			expectedValues := tc.expectedValues
			actualValues := distinctAttrNames.Strings()

			// add dedicated and well known columns to expected values. the code currently does not
			// attempt to perfectly filter these, but instead adds them to the return if any values are present
			expectedValues = append(expectedValues, "dedicated.span.1", "dedicated.resource.1")
			expectedValues = append(expectedValues, "cluster", "http.method", "service.name")

			sort.Strings(expectedValues)
			sort.Strings(actualValues)
			require.Equal(t, expectedValues, actualValues)
		})
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
	}

	ctx := context.TODO()
	block := makeBackendBlockWithTraces(t, []*Trace{fullyPopulatedTestTrace(common.ID{0})})

	opts := common.DefaultSearchOptions()

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("tag: %s, query: %s", tc.tag, tc.query), func(t *testing.T) {
			distinctValues := util.NewDistinctValueCollector[tempopb.TagValue](1_000_000, func(v tempopb.TagValue) int { return len(v.Type) + len(v.Value) })
			req, err := traceql.ExtractFetchSpansRequest(tc.query)
			require.NoError(t, err)

			tag, err := traceql.ParseIdentifier(tc.tag)
			require.NoError(t, err)

			// Build autocomplete request
			autocompleteReq := traceql.AutocompleteRequest{
				Conditions: req.Conditions,
				TagName:    tag,
			}

			tagAtrr, err := traceql.ParseIdentifier(tc.tag)
			require.NoError(t, err)

			// jpe - remove this if we move it into the FetchTagValues function
			autocompleteReq.Conditions = append(autocompleteReq.Conditions, traceql.Condition{
				Attribute: tagAtrr,
				Op:        traceql.OpNone,
			})

			err = block.FetchTagValues(ctx, autocompleteReq, traceql.MakeCollectTagValueFunc(distinctValues.Collect), opts)
			require.NoError(t, err)

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
			distinctValues := util.NewDistinctValueCollector[tempopb.TagValue](1_000_000, func(v tempopb.TagValue) int { return len(v.Type) + len(v.Value) })
			req, err := traceql.ExtractFetchSpansRequest(tc.query)
			require.NoError(b, err)

			tag, err := traceql.ParseIdentifier(tc.tag)
			require.NoError(b, err)

			// FetchTagValues expects the tag to be in the conditions with OpNone otherwise it will
			// fall back to the old tag search
			req.Conditions = append(req.Conditions, traceql.Condition{
				Attribute: tag,
			})

			autocompleteReq := traceql.AutocompleteRequest{
				Conditions: req.Conditions,
				TagName:    tag,
			}
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				err := block.FetchTagValues(ctx, autocompleteReq, traceql.MakeCollectTagValueFunc(distinctValues.Collect), opts)
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
			query: `{span.sampler.type="probabilistic"}`, // generic attribute
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
		b.Run(fmt.Sprintf("query: %s", tc.query), func(b *testing.B) {
			distinctStrings := util.NewDistinctStringCollector(1_000_000)
			req, err := traceql.ExtractFetchSpansRequest(tc.query)
			require.NoError(b, err)

			autocompleteReq := traceql.AutocompleteRequest{
				Conditions: req.Conditions,
				TagName:    traceql.Attribute{},
			}
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				err := block.FetchTagNames(ctx, autocompleteReq, distinctStrings.Collect, opts)
				require.NoError(b, err)
			}
		})
	}
}
