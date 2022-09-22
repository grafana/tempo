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

// SearchTestSuite returns a set of search test cases that ensure
// search behavior is consistent across block types and modules.
// The return parameters are:
//   - trace ID
//   - trace - a fully-populated trace that is searched for every condition. If testing a
//     block format, then write this trace to the block.
//   - start, end - the unix second start/end times for the trace, i.e. slack-adjusted timestamps
//   - expected - The exact search result that should be returned for every matching request
//   - searchesThatMatch - List of search requests that are expected to match the trace
//   - searchesThatDontMatch - List of requests that don't match the trace
func SearchTestSuite() (
	id []byte,
	tr *tempopb.Trace,
	start, end uint32,
	expected *tempopb.TraceSearchMetadata,
	searchesThatMatch []*tempopb.SearchRequest,
	searchesThatDontMatch []*tempopb.SearchRequest,
	tagNames []string,
	tagValues map[string][]string,
) {

	id = test.ValidTraceID(nil)

	start = 1000
	end = 1001

	tr = &tempopb.Trace{
		Batches: []*v1.ResourceSpans{
			{
				Resource: &v1_resource.Resource{
					Attributes: []*v1_common.KeyValue{
						stringKV("service.name", "MyService"),
						stringKV("cluster", "MyCluster"),
						stringKV("namespace", "MyNamespace"),
						stringKV("pod", "MyPod"),
						stringKV("container", "MyContainer"),
						stringKV("k8s.cluster.name", "k8sCluster"),
						stringKV("k8s.namespace.name", "k8sNamespace"),
						stringKV("k8s.pod.name", "k8sPod"),
						stringKV("k8s.container.name", "k8sContainer"),
						stringKV("bat", "Baz"),
					},
				},
				ScopeSpans: []*v1.ScopeSpans{
					{
						Spans: []*v1.Span{
							{
								TraceId:           id,
								Name:              "MySpan",
								SpanId:            []byte{1, 2, 3},
								ParentSpanId:      []byte{4, 5, 6},
								StartTimeUnixNano: uint64(1000 * time.Second),
								EndTimeUnixNano:   uint64(1001 * time.Second),
								Status: &v1.Status{
									Code: v1.Status_STATUS_CODE_ERROR,
								},
								Attributes: []*v1_common.KeyValue{
									stringKV("http.method", "Get"),
									stringKV("http.url", "url/Hello/World"),
									intKV("http.status_code", 500),
									stringKV("foo", "Bar"),
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
				ScopeSpans: []*v1.ScopeSpans{
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
		makeReq("service.name", "Service"),
		makeReq("cluster", "Cluster"),
		makeReq("namespace", "Namespace"),
		makeReq("pod", "Pod"),
		makeReq("container", "Container"),
		makeReq("k8s.cluster.name", "k8sCluster"),
		makeReq("k8s.namespace.name", "k8sNamespace"),
		makeReq("k8s.pod.name", "k8sPod"),
		makeReq("k8s.container.name", "k8sContainer"),
		makeReq("root.service.name", "RootService"),
		makeReq("root.name", "RootSpan"),

		// Well-known span attributes
		makeReq("name", "Span"),
		makeReq("http.method", "Get"),
		makeReq("http.url", "Hello"),
		makeReq("http.status_code", "500"),
		makeReq("status.code", "error"),

		// Span attributes
		makeReq("foo", "Bar"),
		// Resource attributes
		makeReq("bat", "Baz"),

		// Multiple
		{
			Tags: map[string]string{
				"service.name": "Service",
				"http.method":  "Get",
				"foo":          "Bar",
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
		makeReq("service.name", "service"), // wrong case
		makeReq("cluster", "cluster"),      // wrong case
		makeReq("namespace", "namespace"),  // wrong case
		makeReq("pod", "pod"),              // wrong case
		makeReq("container", "container"),  // wrong case

		// Well-known span attributes
		makeReq("http.method", "post"),
		makeReq("http.url", "asdf"),
		makeReq("http.status_code", "200"),
		makeReq("status.code", "ok"),
		makeReq("root.service.name", "NotRootService"),
		makeReq("root.name", "NotRootSpan"),

		// Span attributes
		makeReq("foo", "baz"), // wrong case
	}

	tagNames = []string{
		"bat",
		"cluster",
		"container",
		"foo",
		"http.method",
		"http.status_code",
		"http.url",
		"k8s.cluster.name",
		"k8s.container.name",
		"k8s.namespace.name",
		"k8s.pod.name",
		"name",
		"namespace",
		"pod",
		"root.name",
		"root.service.name",
		"service.name",
		"status.code",
	}

	tagValues = map[string][]string{
		"bat":                {"Baz"},
		"cluster":            {"MyCluster"},
		"container":          {"MyContainer"},
		"foo":                {"Bar"},
		"http.method":        {"Get"},
		"http.status_code":   {"500"},
		"http.url":           {"url/Hello/World"},
		"k8s.cluster.name":   {"k8sCluster"},
		"k8s.container.name": {"k8sContainer"},
		"k8s.namespace.name": {"k8sNamespace"},
		"k8s.pod.name":       {"k8sPod"},
		"name":               {"MySpan", "RootSpan"},
		"namespace":          {"MyNamespace"},
		"pod":                {"MyPod"},
		"root.name":          {"RootSpan"},
		"root.service.name":  {"RootService"},
		"service.name":       {"MyService", "RootService"},
		"status.code":        {"0", "2"},
	}

	return
}
