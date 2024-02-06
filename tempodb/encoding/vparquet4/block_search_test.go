package vparquet4

import (
	"context"
	"math/rand"
	"path"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

func TestBackendBlockSearch(t *testing.T) {
	// Helper functions to make pointers
	strPtr := func(s string) *string { return &s }
	intPtr := func(i int64) *int64 { return &i }

	// Trace
	// This is a fully-populated trace that we search for every condition
	wantTr := &Trace{
		TraceID:           test.ValidTraceID(nil),
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
						{Key: "bat", Value: strPtr("baz")},
					},
					DedicatedAttributes: DedicatedAttributes{
						String01: strPtr("dedicated-resource-attr-value-1"),
						String02: strPtr("dedicated-resource-attr-value-2"),
						String03: strPtr("dedicated-resource-attr-value-3"),
						String04: strPtr("dedicated-resource-attr-value-4"),
						String05: strPtr("dedicated-resource-attr-value-5"),
					},
				},
				ScopeSpans: []ScopeSpans{
					{
						Spans: []Span{
							{
								Name:           "hello",
								HttpMethod:     strPtr("get"),
								HttpUrl:        strPtr("url/hello/world"),
								HttpStatusCode: intPtr(500),
								SpanID:         []byte{},
								ParentSpanID:   []byte{},
								StatusCode:     int(v1.Status_STATUS_CODE_ERROR),
								Attrs: []Attribute{
									{Key: "foo", Value: strPtr("bar")},
								},
								DedicatedAttributes: DedicatedAttributes{
									String01: strPtr("dedicated-span-attr-value-1"),
									String02: strPtr("dedicated-span-attr-value-2"),
									String03: strPtr("dedicated-span-attr-value-3"),
									String04: strPtr("dedicated-span-attr-value-4"),
									String05: strPtr("dedicated-span-attr-value-5"),
								},
							},
						},
					},
				},
			},
		},
	}

	// make a bunch of traces and include our wantTr above
	total := 1000
	insertAt := rand.Intn(total)
	allTraces := make([]*Trace, 0, total)
	for i := 0; i < total; i++ {
		if i == insertAt {
			allTraces = append(allTraces, wantTr)
			continue
		}

		id := test.ValidTraceID(nil)
		pbTrace := test.MakeTrace(10, id)
		pqTrace, _ := traceToParquet(&backend.BlockMeta{}, id, pbTrace, nil)
		allTraces = append(allTraces, pqTrace)
	}

	b := makeBackendBlockWithTraces(t, allTraces)
	ctx := context.TODO()

	// Helper function to make a tag search
	makeReq := func(k, v string) *tempopb.SearchRequest {
		return &tempopb.SearchRequest{
			Tags: map[string]string{
				k: v,
			},
		}
	}

	// Matches
	searchesThatMatch := []*tempopb.SearchRequest{
		{
			// Empty request
		},
		{
			MinDurationMs: 99,
			MaxDurationMs: 101,
		},
		{
			Start: 1000,
			End:   2000,
		},
		{
			// Overlaps start
			Start: 999,
			End:   1001,
		},
		{
			// Overlaps end
			Start: 1999,
			End:   2001,
		},

		// Well-known resource attributes
		makeReq(LabelServiceName, "service"),
		makeReq(LabelCluster, "cluster"),
		makeReq(LabelNamespace, "namespace"),
		makeReq(LabelPod, "pod"),
		makeReq(LabelContainer, "container"),
		makeReq(LabelK8sClusterName, "k8scluster"),
		makeReq(LabelK8sNamespaceName, "k8snamespace"),
		makeReq(LabelK8sPodName, "k8spod"),
		makeReq(LabelK8sContainerName, "k8scontainer"),

		// Dedicated resource attributes
		makeReq("dedicated.resource.3", "dedicated-resource-attr-value-3"),

		// Well-known span attributes
		makeReq(LabelName, "ell"),
		makeReq(LabelHTTPMethod, "get"),
		makeReq(LabelHTTPUrl, "hello"),
		makeReq(LabelHTTPStatusCode, "500"),
		makeReq(LabelStatusCode, StatusCodeError),

		// Dedicated span attributes
		makeReq("dedicated.span.4", "dedicated-span-attr-value-4"),

		// Span attributes
		makeReq("foo", "bar"),
		// Resource attributes
		makeReq("bat", "baz"),

		// Multiple
		{
			Tags: map[string]string{
				"service.name": "service",
				"http.method":  "get",
				"foo":          "bar",
			},
		},
	}
	expected := &tempopb.TraceSearchMetadata{
		TraceID:           util.TraceIDToHexString(wantTr.TraceID),
		StartTimeUnixNano: wantTr.StartTimeUnixNano,
		DurationMs:        uint32(wantTr.DurationNano / uint64(time.Millisecond)),
		RootServiceName:   wantTr.RootServiceName,
		RootTraceName:     wantTr.RootSpanName,
	}

	findInResults := func(id string, res []*tempopb.TraceSearchMetadata) *tempopb.TraceSearchMetadata {
		for _, r := range res {
			if r.TraceID == id {
				return r
			}
		}
		return nil
	}

	for _, req := range searchesThatMatch {
		res, err := b.Search(ctx, req, common.DefaultSearchOptions())
		require.NoError(t, err)

		meta := findInResults(expected.TraceID, res.Traces)
		require.NotNil(t, meta, "search request:", req)
		require.Equal(t, expected, meta, "search request:", req)
	}

	// Excludes
	searchesThatDontMatch := []*tempopb.SearchRequest{
		{
			MinDurationMs: 101,
		},
		{
			MaxDurationMs: 99,
		},
		{
			Start: 100,
			End:   200,
		},

		// Well-known resource attributes
		makeReq(LabelServiceName, "foo"),
		makeReq(LabelCluster, "foo"),
		makeReq(LabelNamespace, "foo"),
		makeReq(LabelPod, "foo"),
		makeReq(LabelContainer, "foo"),

		// Dedicated resource attributes
		makeReq("dedicated.resource.3", "dedicated-resource-attr-value-1"),

		// Well-known span attributes
		makeReq(LabelHTTPMethod, "post"),
		makeReq(LabelHTTPUrl, "asdf"),
		makeReq(LabelHTTPStatusCode, "200"),
		makeReq(LabelStatusCode, StatusCodeOK),

		// Dedicated span attributes
		makeReq("dedicated.span.4", "dedicated-span-attr-value-5"),

		// Span attributes
		makeReq("foo", "baz"),

		// Multiple
		{
			Tags: map[string]string{
				"http.status_code": "500",
				"service.name":     "asdf",
			},
		},
	}
	for _, req := range searchesThatDontMatch {
		res, err := b.Search(ctx, req, common.DefaultSearchOptions())
		require.NoError(t, err)
		meta := findInResults(expected.TraceID, res.Traces)
		require.Nil(t, meta, req)
	}
}

func makeBackendBlockWithTraces(t *testing.T, trs []*Trace) *backendBlock {
	rawR, rawW, _, err := local.New(&local.Config{
		Path: t.TempDir(),
	})
	require.NoError(t, err)

	r := backend.NewReader(rawR)
	w := backend.NewWriter(rawW)
	ctx := context.Background()

	cfg := &common.BlockConfig{
		BloomFP:             0.01,
		BloomShardSizeBytes: 100 * 1024,
	}

	meta := backend.NewBlockMeta("fake", uuid.New(), VersionString, backend.EncNone, "")
	meta.TotalObjects = 1
	meta.DedicatedColumns = test.MakeDedicatedColumns()

	s := newStreamingBlock(ctx, cfg, meta, r, w, tempo_io.NewBufferedWriter)

	for i, tr := range trs {
		err = s.Add(tr, 0, 0)
		require.NoError(t, err)
		if i%100 == 0 {
			_, err := s.Flush()
			require.NoError(t, err)
		}
	}

	_, err = s.Complete()
	require.NoError(t, err)

	b := newBackendBlock(s.meta, r)

	return b
}

func makeTraces() ([]*Trace, map[string]string, map[string]string, map[string]string) {
	traces := []*Trace{}
	intrinsicVals := map[string]string{}
	resourceAttrVals := map[string]string{}
	spanAttrVals := map[string]string{}

	ptr := func(s string) *string { return &s }

	resourceAttrVals[LabelCluster] = "cluster"
	resourceAttrVals[LabelServiceName] = "servicename"
	resourceAttrVals[LabelNamespace] = "ns"
	resourceAttrVals[LabelPod] = "pod"
	resourceAttrVals[LabelContainer] = "con"
	resourceAttrVals[LabelK8sClusterName] = "kclust"
	resourceAttrVals[LabelK8sNamespaceName] = "kns"
	resourceAttrVals[LabelK8sPodName] = "kpod"
	resourceAttrVals[LabelK8sContainerName] = "k8scon"

	dedicatedResourceAttrs := DedicatedAttributes{
		String01: ptr("dedicated-resource-attr-value-1"),
		String02: ptr("dedicated-resource-attr-value-2"),
		String03: ptr("dedicated-resource-attr-value-3"),
		String04: ptr("dedicated-resource-attr-value-4"),
		String05: ptr("dedicated-resource-attr-value-5"),
	}
	resourceAttrVals["dedicated.resource.1"] = *dedicatedResourceAttrs.String01
	resourceAttrVals["dedicated.resource.2"] = *dedicatedResourceAttrs.String02
	resourceAttrVals["dedicated.resource.3"] = *dedicatedResourceAttrs.String03
	resourceAttrVals["dedicated.resource.4"] = *dedicatedResourceAttrs.String04
	resourceAttrVals["dedicated.resource.5"] = *dedicatedResourceAttrs.String05

	intrinsicVals[LabelName] = "span"
	// todo: the below 3 are not supported in traceql and should be removed when support for tags based search is removed
	intrinsicVals[LabelRootServiceName] = "rootsvc"
	intrinsicVals[LabelStatusCode] = "2"
	intrinsicVals[LabelRootSpanName] = "rootspan"

	spanAttrVals[LabelHTTPMethod] = "method"
	spanAttrVals[LabelHTTPUrl] = "url"
	spanAttrVals[LabelHTTPStatusCode] = "404"

	dedicatedSpanAttrs := DedicatedAttributes{
		String01: ptr("dedicated-span-attr-value-1"),
		String02: ptr("dedicated-span-attr-value-2"),
		String03: ptr("dedicated-span-attr-value-3"),
		String04: ptr("dedicated-span-attr-value-4"),
		String05: ptr("dedicated-span-attr-value-5"),
	}
	spanAttrVals["dedicated.span.1"] = *dedicatedSpanAttrs.String01
	spanAttrVals["dedicated.span.2"] = *dedicatedSpanAttrs.String02
	spanAttrVals["dedicated.span.3"] = *dedicatedSpanAttrs.String03
	spanAttrVals["dedicated.span.4"] = *dedicatedSpanAttrs.String04
	spanAttrVals["dedicated.span.5"] = *dedicatedSpanAttrs.String05

	for i := 0; i < 10; i++ {
		tr := &Trace{
			RootServiceName: "rootsvc",
			RootSpanName:    "rootspan",
		}

		for j := 0; j < 3; j++ {
			key := test.RandomString()
			val := test.RandomString()
			resourceAttrVals[key] = val

			rs := ResourceSpans{
				Resource: Resource{
					ServiceName:      "servicename",
					Cluster:          ptr("cluster"),
					Namespace:        ptr("ns"),
					Pod:              ptr("pod"),
					Container:        ptr("con"),
					K8sClusterName:   ptr("kclust"),
					K8sNamespaceName: ptr("kns"),
					K8sPodName:       ptr("kpod"),
					K8sContainerName: ptr("k8scon"),
					Attrs: []Attribute{
						{
							Key:   key,
							Value: &val,
						},
					},
					DedicatedAttributes: dedicatedResourceAttrs,
				},
				ScopeSpans: []ScopeSpans{
					{},
				},
			}
			tr.ResourceSpans = append(tr.ResourceSpans, rs)

			for k := 0; k < 10; k++ {
				key := test.RandomString()
				val := test.RandomString()
				spanAttrVals[key] = val

				sts := int64(404)
				span := Span{
					Name:           "span",
					HttpMethod:     ptr("method"),
					HttpUrl:        ptr("url"),
					HttpStatusCode: &sts,
					StatusCode:     2,
					Attrs: []Attribute{
						{
							Key:   key,
							Value: &val,
						},
					},
					DedicatedAttributes: dedicatedSpanAttrs,
				}

				rs.ScopeSpans[0].Spans = append(rs.ScopeSpans[0].Spans, span)
			}

		}

		traces = append(traces, tr)
	}

	return traces, intrinsicVals, resourceAttrVals, spanAttrVals
}

func BenchmarkBackendBlockSearchTraces(b *testing.B) {
	testCases := []struct {
		name string
		tags map[string]string
	}{
		{"noMatch", map[string]string{"foo": "bar"}},
		{"partialMatch", map[string]string{"foo": "bar", "component": "gRPC"}},
		{"service.name", map[string]string{"service.name": "a"}},
	}

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
	opts.StartPage = 10
	opts.TotalPages = 10

	for _, tc := range testCases {

		req := &tempopb.SearchRequest{
			Tags:  tc.tags,
			Limit: 20,
		}

		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			bytesRead := 0
			for i := 0; i < b.N; i++ {
				resp, err := block.Search(ctx, req, opts)
				require.NoError(b, err)
				bytesRead += int(resp.Metrics.InspectedBytes)
			}
			b.SetBytes(int64(bytesRead) / int64(b.N))
			b.ReportMetric(float64(bytesRead)/float64(b.N), "bytes/op")
		})
	}
}
