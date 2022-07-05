package trace

import (
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
)

func stringKV(k, v string) *v1_common.KeyValue {
	return &v1_common.KeyValue{
		Key:   k,
		Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: v}},
	}
}

func intKV(k string, v int) *v1_common.KeyValue {
	return &v1_common.KeyValue{
		Key:   k,
		Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_IntValue{IntValue: int64(v)}},
	}
}

// Helper function to make a tag search
func makeReq(k, v string) *tempopb.SearchRequest {
	return &tempopb.SearchRequest{
		Tags: map[string]string{
			k: v,
		},
	}
}

// This is a fully-populated trace that we search for every condition
func SearchTestSuite() (
	id []byte,
	tr *tempopb.Trace,
	start, end uint32,
	expected *tempopb.TraceSearchMetadata,
	searchesThatMatch []*tempopb.SearchRequest,
	searchesThatDontMatch []*tempopb.SearchRequest) {

	id = test.ValidTraceID(nil)

	start = 1000
	end = 1001

	tr = &tempopb.Trace{
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

	expected = &tempopb.TraceSearchMetadata{
		TraceID:           util.TraceIDToHexString(id),
		StartTimeUnixNano: uint64(1000 * time.Second),
		DurationMs:        1000,
		RootServiceName:   "RootService",
		RootTraceName:     "RootSpan",
	}

	// Matches
	searchesThatMatch = []*tempopb.SearchRequest{
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

	// Excludes
	searchesThatDontMatch = []*tempopb.SearchRequest{
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

	return
}
