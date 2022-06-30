package tempodb

import (
	"context"
	"path"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v2 "github.com/grafana/tempo/tempodb/encoding/v2"
	"github.com/grafana/tempo/tempodb/encoding/vparquet"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/stretchr/testify/require"
)

func TestSearchCompleteBlock(t *testing.T) {
	for _, v := range []string{v2.VersionString, vparquet.VersionString} {
		t.Run(v, func(t *testing.T) {
			testSearchCompleteBlock(t, v)
		})
	}
}

func testSearchCompleteBlock(t *testing.T, blockVersion string) {
	tempDir := t.TempDir()
	ctx := context.Background()

	r, w, c, err := New(&Config{
		Backend: "local",
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &common.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              .01,
			BloomShardSizeBytes:  100_000,
			Version:              blockVersion,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			Filepath:       path.Join(tempDir, "wal"),
			IngestionSlack: time.Since(time.Time{}),
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	require.NoError(t, err)

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})

	r.EnablePolling(&mockJobSharder{})
	rw := r.(*readerWriter)

	id, wantTr, wantMeta := fullyPopulatedSearchTrace()

	// Write to wal
	wal := w.WAL()
	head, err := wal.NewBlock(uuid.New(), testTenantID, model.CurrentEncoding)
	require.NoError(t, err)
	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)
	b1, err := dec.PrepareForWrite(wantTr, 1000, 1001)
	require.NoError(t, err)
	b2, err := dec.ToObject([][]byte{b1})
	require.NoError(t, err)
	err = head.Append(id, b2, 1000, 1001)
	require.NoError(t, err, "unexpected error writing req")

	// Complete block
	block, err := w.CompleteBlock(head, &mockCombiner{})
	require.NoError(t, err)
	meta := block.BlockMeta()

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
			MinDurationMs: 999,
			MaxDurationMs: 1001,
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
			Start: 1001,
			End:   1002,
		},

		// Well-known resource attributes
		makeReq("service.name", "service"),
		makeReq("cluster", "cluster"),
		makeReq("namespace", "namespace"),
		makeReq("pod", "pod"),
		makeReq("container", "container"),
		makeReq("k8s.cluster.name", "k8scluster"),
		makeReq("k8s.namespace.name", "k8snamespace"),
		makeReq("k8s.pod.name", "k8spod"),
		makeReq("k8s.container.name", "k8scontainer"),

		// Well-known span attributes
		makeReq("name", "ell"),
		makeReq("http.method", "get"),
		makeReq("http.url", "hello"),
		makeReq("http.status_code", "500"),
		makeReq("status.code", "error"),

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

	for _, req := range searchesThatMatch {
		res, err := r.Search(ctx, meta, req, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.Equal(t, 1, len(res.Traces), "search request: %+v", req)
		require.Equal(t, wantMeta, res.Traces[0], "search request:", req)
	}

	// Excludes
	searchesThatDontMatch := []*tempopb.SearchRequest{
		{
			MinDurationMs: 1001,
		},
		{
			MaxDurationMs: 999,
		},
		{
			Start: 100,
			End:   200,
		},

		// Well-known resource attributes
		makeReq("service.name", "foo"),
		makeReq("cluster", "foo"),
		makeReq("namespace", "foo"),
		makeReq("pod", "foo"),
		makeReq("container", "foo"),

		// Well-known span attributes
		makeReq("http.method", "post"),
		makeReq("http.url", "asdf"),
		makeReq("http.status_code", "200"),
		makeReq("status.code", "ok"),

		// Span attributes
		makeReq("foo", "baz"),
	}
	for _, req := range searchesThatDontMatch {
		res, err := rw.Search(ctx, meta, req, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.Empty(t, res.Traces, "search request:", req)
	}

}

// This is a fully-populated trace that we search for every condition
func fullyPopulatedSearchTrace() (common.ID, *tempopb.Trace, *tempopb.TraceSearchMetadata) {
	stringKV := func(k, v string) *v1_common.KeyValue {
		return &v1_common.KeyValue{
			Key:   k,
			Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: v}},
		}
	}

	intKV := func(k string, v int) *v1_common.KeyValue {
		return &v1_common.KeyValue{
			Key:   k,
			Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_IntValue{IntValue: int64(v)}},
		}
	}
	id := test.ValidTraceID(nil)
	tr := &tempopb.Trace{
		Batches: []*v1.ResourceSpans{
			{
				Resource: &v1_resource.Resource{
					Attributes: []*v1_common.KeyValue{
						stringKV("service.name", "myservice"),
						stringKV("cluster", "cluster"),
						stringKV("namespace", "namespace"),
						stringKV("pod", "pod"),
						stringKV("container", "container"),
						stringKV("k8s.cluster.name", "k8scluster"),
						stringKV("k8s.namespace.name", "k8snamespace"),
						stringKV("k8s.pod.name", "k8spod"),
						stringKV("k8s.container.name", "k8scontainer"),
						stringKV("bat", "baz"),
					},
				},
				InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
					{
						Spans: []*v1.Span{
							{
								TraceId:           id,
								Name:              "hello",
								SpanId:            []byte{1, 2, 3},
								ParentSpanId:      []byte{4, 5, 6},
								StartTimeUnixNano: uint64(1000 * time.Second),
								EndTimeUnixNano:   uint64(1001 * time.Second),
								Status: &v1.Status{
									Code: v1.Status_STATUS_CODE_ERROR,
								},
								Attributes: []*v1_common.KeyValue{
									stringKV("http.method", "get"),
									stringKV("http.url", "url/hello/world"),
									intKV("http.status_code", 500),
									stringKV("foo", "bar"),
								},
							},
						},
					},
				},
			},
			{
				Resource: &v1_resource.Resource{
					Attributes: []*v1_common.KeyValue{
						stringKV("service.name", "RootService"),
					},
				},
				InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
					{
						Spans: []*v1.Span{
							{
								TraceId:           id,
								Name:              "RootSpan",
								StartTimeUnixNano: uint64(1000 * time.Second),
								EndTimeUnixNano:   uint64(1001 * time.Second),
								Status:            &v1.Status{},
							},
						},
					},
				},
			},
		},
	}

	expected := &tempopb.TraceSearchMetadata{
		TraceID:           util.TraceIDToHexString(id),
		StartTimeUnixNano: uint64(1000 * time.Second),
		DurationMs:        1000,
		RootServiceName:   "RootService",
		RootTraceName:     "RootSpan",
	}

	return id, tr, expected
}
