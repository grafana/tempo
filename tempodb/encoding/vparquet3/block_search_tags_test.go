package vparquet3

import (
	"context"
	"fmt"
	"path"
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackendBlockSearchTags(t *testing.T) {
	traces, _, resourceAttrVals, spanAttrVals := makeTraces()
	block := makeBackendBlockWithTraces(t, traces)

	testVals := func(scope traceql.AttributeScope, attrs map[string]string) {
		foundAttrs := map[string]struct{}{}
		cb := func(s string) {
			foundAttrs[s] = struct{}{}
		}

		ctx := context.Background()
		err := block.SearchTags(ctx, scope, cb, common.DefaultSearchOptions())
		require.NoError(t, err)

		// test that all attrs are in found attrs
		for k := range attrs {
			_, ok := foundAttrs[k]
			require.True(t, ok, "attr: %s, scope: %s", k, scope)
			delete(foundAttrs, k)
		}
		// if our scope is specific, we can also assert that SearchTags returned only exactly what we expected
		if scope != traceql.AttributeScopeNone {
			require.Len(t, foundAttrs, 0, "scope: %s", scope)
		}
	}

	testVals(traceql.AttributeScopeNone, resourceAttrVals)
	testVals(traceql.AttributeScopeResource, resourceAttrVals)
	testVals(traceql.AttributeScopeNone, spanAttrVals)
	testVals(traceql.AttributeScopeSpan, spanAttrVals)
}

func TestBackendBlockSearchTagValues(t *testing.T) {
	traces, intrinsics, resourceAttrs, spanAttrs := makeTraces()
	block := makeBackendBlockWithTraces(t, traces)

	// concat all attrs and test
	attrs := map[string]string{}
	for k, v := range intrinsics {
		attrs[k] = v
	}
	for k, v := range resourceAttrs {
		attrs[k] = v
	}
	for k, v := range spanAttrs {
		attrs[k] = v
	}

	ctx := context.Background()
	for tag, val := range attrs {
		wasCalled := false
		cb := func(s string) {
			wasCalled = true
			assert.Equal(t, val, s, tag)
		}

		err := block.SearchTagValues(ctx, tag, cb, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.True(t, wasCalled, tag)
	}
}

func TestBackendBlockSearchTagValuesV2(t *testing.T) {
	block := makeBackendBlockWithTraces(t, []*Trace{fullyPopulatedTestTrace(common.ID{0})})

	testCases := []struct {
		tag  traceql.Attribute
		vals []traceql.Static
	}{
		// Intrinsic
		{traceql.MustParseIdentifier("name"), []traceql.Static{
			traceql.NewStaticString("hello"),
			traceql.NewStaticString("world"),
		}},
		{traceql.MustParseIdentifier("rootName"), []traceql.Static{
			traceql.NewStaticString("RootSpan"),
		}},
		{traceql.MustParseIdentifier("rootServiceName"), []traceql.Static{
			traceql.NewStaticString("RootService"),
		}},

		// Attribute that conflicts with intrinsic
		{traceql.MustParseIdentifier(".name"), []traceql.Static{
			traceql.NewStaticString("Bob"),
		}},

		// Mixed types
		{traceql.MustParseIdentifier(".http.status_code"), []traceql.Static{
			traceql.NewStaticInt(500),
			traceql.NewStaticString("500ouch"),
		}},

		// Trace-level special
		{traceql.NewAttribute("root.name"), []traceql.Static{
			traceql.NewStaticString("RootSpan"),
		}},

		// Resource only, mixed well-known column and generic key/value
		{traceql.MustParseIdentifier("resource.service.name"), []traceql.Static{
			traceql.NewStaticString("myservice"),
			traceql.NewStaticString("service2"),
			traceql.NewStaticInt(123),
		}},

		// Span only
		{traceql.MustParseIdentifier("span.service.name"), []traceql.Static{
			traceql.NewStaticString("spanservicename"),
		}},

		// Float column
		{traceql.MustParseIdentifier(".float"), []traceql.Static{
			traceql.NewStaticFloat(456.78),
		}},

		// Attr present at both resource and span level
		{traceql.MustParseIdentifier(".foo"), []traceql.Static{
			traceql.NewStaticString("abc"),
			traceql.NewStaticString("def"),
		}},

		// Dedicated resource attributes
		{traceql.MustParseIdentifier(".dedicated.resource.3"), []traceql.Static{
			traceql.NewStaticString("dedicated-resource-attr-value-3"),
		}},
		{traceql.MustParseIdentifier("resource.dedicated.resource.2"), []traceql.Static{
			traceql.NewStaticString("dedicated-resource-attr-value-2"),
		}},

		// Dedicated span attributes
		{traceql.MustParseIdentifier(".dedicated.span.1"), []traceql.Static{
			traceql.NewStaticString("dedicated-span-attr-value-1"),
		}},
		{traceql.MustParseIdentifier("span.dedicated.span.2"), []traceql.Static{
			traceql.NewStaticString("dedicated-span-attr-value-2"),
		}},
	}

	ctx := context.Background()
	for _, tc := range testCases {

		var got []traceql.Static
		cb := func(v traceql.Static) bool {
			got = append(got, v)
			return false
		}

		err := block.SearchTagValuesV2(ctx, tc.tag, cb, common.DefaultSearchOptions())
		require.NoError(t, err, tc.tag)
		require.Equal(t, tc.vals, got, "tag=%v", tc.tag)
	}
}

func BenchmarkBackendBlockSearchTags(b *testing.B) {
	ctx := context.TODO()
	tenantID := "1"
	blockID := uuid.MustParse("3685ee3d-cbbf-4f36-bf28-93447a19dea6")

	r, _, _, err := local.New(&local.Config{
		Path: path.Join("/Users/marty/src/tmp/"),
	})
	require.NoError(b, err)

	rr := backend.NewReader(r)
	meta, err := rr.BlockMeta(ctx, blockID, tenantID)
	require.NoError(b, err)

	block := newBackendBlock(meta, rr)
	opts := common.DefaultSearchOptions()
	d := util.NewDistinctStringCollector(1_000_000)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := block.SearchTags(ctx, traceql.AttributeScopeNone, d.Collect, opts)
		require.NoError(b, err)
	}
}

func BenchmarkBackendBlockSearchTagValues(b *testing.B) {
	testCases := []struct {
		tag   string
		query string
	}{
		{
			tag:   "span.http.url",
			query: "{}",
		},
		{
			tag:   "span.http.url",
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
	}

	ctx := context.TODO()
	tenantID := "1"
	//blockID := uuid.MustParse("3685ee3d-cbbf-4f36-bf28-93447a19dea6")
	blockID := uuid.MustParse("0008e57d-069d-4510-a001-b9433b2da08c")

	r, _, _, err := local.New(&local.Config{
		//Path: path.Join("/Users/marty/src/tmp/"),
		Path: path.Join("/Users/mapno/workspace/testblock"),
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

			autocompleteReq := traceql.AutocompleteRequest{
				Conditions: req.Conditions,
				TagName:    "http.url",
			}
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				err := block.FetchTagValues(ctx, autocompleteReq, distinctValues.Collect, opts)
				//err := block.SearchTagValues(ctx, tc, d.Collect, opts)
				require.NoError(b, err)
			}
		})
	}
}
