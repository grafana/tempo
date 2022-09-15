package vparquet

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
						{Key: "bat", Value: strPtr("baz")},
					},
				},
				InstrumentationLibrarySpans: []ILS{
					{
						Spans: []Span{
							{
								Name:           "hello",
								HttpMethod:     strPtr("get"),
								HttpUrl:        strPtr("url/hello/world"),
								HttpStatusCode: intPtr(500),
								ID:             []byte{},
								ParentSpanID:   []byte{},
								StatusCode:     int(v1.Status_STATUS_CODE_ERROR),
								Attrs: []Attribute{
									{Key: "foo", Value: strPtr("bar")},
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
		pqTrace := traceToParquet(id, pbTrace)
		allTraces = append(allTraces, &pqTrace)
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

		// Well-known span attributes
		makeReq(LabelName, "ell"),
		makeReq(LabelHTTPMethod, "get"),
		makeReq(LabelHTTPUrl, "hello"),
		makeReq(LabelHTTPStatusCode, "500"),
		makeReq(LabelStatusCode, StatusCodeError),

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
		DurationMs:        uint32(wantTr.DurationNanos / uint64(time.Millisecond)),
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
		res, err := b.Search(ctx, req, defaultSearchOptions())
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

		// Well-known span attributes
		makeReq(LabelHTTPMethod, "post"),
		makeReq(LabelHTTPUrl, "asdf"),
		makeReq(LabelHTTPStatusCode, "200"),
		makeReq(LabelStatusCode, StatusCodeOK),

		// Span attributes
		makeReq("foo", "baz"),
	}
	for _, req := range searchesThatDontMatch {
		res, err := b.Search(ctx, req, defaultSearchOptions())
		require.NoError(t, err)
		meta := findInResults(expected.TraceID, res.Traces)
		require.Nil(t, meta, req)
	}
}

func TestBackendBlockSearchTags(t *testing.T) {
	traces, attrs := makeTraces()
	block := makeBackendBlockWithTraces(t, traces)

	foundAttrs := map[string]struct{}{}

	cb := func(s string) {
		foundAttrs[s] = struct{}{}
	}

	ctx := context.Background()
	err := block.SearchTags(ctx, cb, defaultSearchOptions())
	require.NoError(t, err)

	// test that all attrs are in found attrs
	for k := range attrs {
		_, ok := foundAttrs[k]
		require.True(t, ok)
	}
}

func TestBackendBlockSearchTagValues(t *testing.T) {
	traces, attrs := makeTraces()
	block := makeBackendBlockWithTraces(t, traces)

	ctx := context.Background()
	for tag, val := range attrs {
		wasCalled := false
		cb := func(s string) {
			wasCalled = true
			assert.Equal(t, val, s, tag)
		}

		err := block.SearchTagValues(ctx, tag, cb, defaultSearchOptions())
		require.NoError(t, err)
		require.True(t, wasCalled, tag)
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

	s := newStreamingBlock(ctx, cfg, meta, r, w, tempo_io.NewBufferedWriter)

	for i, tr := range trs {
		s.Add(tr, 0, 0)
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

func defaultSearchOptions() common.SearchOptions {
	return common.SearchOptions{
		ChunkSizeBytes:  1_000_000,
		ReadBufferCount: 8,
		ReadBufferSize:  4 * 1024 * 1024,
	}
}

func makeTraces() ([]*Trace, map[string]string) {
	traces := []*Trace{}
	attrVals := make(map[string]string)

	ptr := func(s string) *string { return &s }

	attrVals[LabelCluster] = "cluster"
	attrVals[LabelServiceName] = "servicename"
	attrVals[LabelRootServiceName] = "rootsvc"
	attrVals[LabelNamespace] = "ns"
	attrVals[LabelPod] = "pod"
	attrVals[LabelContainer] = "con"
	attrVals[LabelK8sClusterName] = "kclust"
	attrVals[LabelK8sNamespaceName] = "kns"
	attrVals[LabelK8sPodName] = "kpod"
	attrVals[LabelK8sContainerName] = "k8scon"

	attrVals[LabelName] = "span"
	attrVals[LabelRootSpanName] = "rootspan"
	attrVals[LabelHTTPMethod] = "method"
	attrVals[LabelHTTPUrl] = "url"
	attrVals[LabelHTTPStatusCode] = "404"
	attrVals[LabelStatusCode] = "2"

	for i := 0; i < 10; i++ {
		tr := &Trace{
			RootServiceName: "rootsvc",
			RootSpanName:    "rootspan",
		}

		for j := 0; j < 3; j++ {
			key := test.RandomString()
			val := test.RandomString()
			attrVals[key] = val

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
				},
				InstrumentationLibrarySpans: []ILS{
					{},
				},
			}
			tr.ResourceSpans = append(tr.ResourceSpans, rs)

			for k := 0; k < 10; k++ {
				key := test.RandomString()
				val := test.RandomString()
				attrVals[key] = val

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
				}

				rs.InstrumentationLibrarySpans[0].Spans = append(rs.InstrumentationLibrarySpans[0].Spans, span)
			}

		}

		traces = append(traces, tr)
	}

	return traces, attrVals
}
