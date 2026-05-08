package spanmetrics

import (
	"context"
	"encoding/binary"
	"strconv"
	"testing"

	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resource_v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/prometheus/client_golang/prometheus"
)

func BenchmarkSpanMetricsPushSpans(b *testing.B) {
	for _, tc := range []struct {
		name    string
		tune    func(*Config)
		request *tempopb.PushSpansRequest
	}{
		{
			name:    "default",
			request: benchmarkSpanMetricsRequest(4, 100, true),
		},
		{
			name: "target_info",
			tune: func(cfg *Config) {
				cfg.EnableTargetInfo = true
				cfg.TargetInfoExcludedDimensions = []string{"excluded"}
			},
			request: benchmarkSpanMetricsRequest(4, 100, true),
		},
		{
			name: "target_info_all_filtered",
			tune: func(cfg *Config) {
				cfg.EnableTargetInfo = true
				cfg.TargetInfoExcludedDimensions = []string{"excluded"}
				cfg.FilterPolicies = []filterconfig.FilterPolicy{{
					Exclude: &filterconfig.PolicyMatch{
						MatchType: filterconfig.Strict,
						Attributes: []filterconfig.MatchPolicyAttribute{{
							Key:   "span.http.method",
							Value: "GET",
						}},
					},
				}}
			},
			request: benchmarkSpanMetricsRequest(4, 100, true),
		},
		{
			name: "dimensions_and_mappings",
			tune: func(cfg *Config) {
				cfg.EnableTargetInfo = false
				cfg.Dimensions = []string{
					"k8s.cluster.name",
					"k8s.namespace.name",
					"http.method",
					"http.status_code",
					"span.attr.1",
					"span.attr.2",
				}
				cfg.DimensionMappings = []sharedconfig.DimensionMappings{
					{Name: "route_key", SourceLabel: []string{"http.method", "http.route", "http.status_code"}, Join: ":"},
					{Name: "pod_key", SourceLabel: []string{"k8s.namespace.name", "k8s.pod.name"}, Join: "/"},
				}
			},
			request: benchmarkSpanMetricsRequest(4, 100, true),
		},
		{
			name: "count_only",
			tune: func(cfg *Config) {
				cfg.Subprocessors[Latency] = false
				cfg.Subprocessors[Size] = false
			},
			request: benchmarkSpanMetricsRequest(4, 100, false),
		},
	} {
		b.Run(tc.name, func(b *testing.B) {
			cfg := Config{}
			cfg.RegisterFlagsAndApplyDefaults("", nil)
			if tc.tune != nil {
				tc.tune(&cfg)
			}

			p, err := New(
				cfg,
				registry.NewTestRegistry(),
				prometheus.NewCounter(prometheus.CounterOpts{}),
				prometheus.NewCounter(prometheus.CounterOpts{}),
			)
			if err != nil {
				b.Fatal(err)
			}
			defer p.Shutdown(context.Background())

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.PushSpans(context.Background(), tc.request)
			}
		})
	}
}

func benchmarkSpanMetricsRequest(resources int, spansPerResource int, sameTracePerResource bool) *tempopb.PushSpansRequest {
	req := &tempopb.PushSpansRequest{
		Batches: make([]*trace_v1.ResourceSpans, 0, resources),
	}
	for r := 0; r < resources; r++ {
		rs := &trace_v1.ResourceSpans{
			Resource: &resource_v1.Resource{
				Attributes: []*common_v1.KeyValue{
					benchmarkStringAttr("service.name", "svc-"+strconv.Itoa(r)),
					benchmarkStringAttr("service.namespace", "ns-"+strconv.Itoa(r%2)),
					benchmarkStringAttr("service.instance.id", "instance-"+strconv.Itoa(r)),
					benchmarkStringAttr("k8s.cluster.name", "cluster-a"),
					benchmarkStringAttr("k8s.namespace.name", "namespace-"+strconv.Itoa(r%4)),
					benchmarkStringAttr("k8s.pod.name", "pod-"+strconv.Itoa(r)),
					benchmarkStringAttr("k8s.node.name", "node-"+strconv.Itoa(r%8)),
					benchmarkStringAttr("telemetry.sdk.language", "go"),
					benchmarkStringAttr("telemetry.sdk.version", "1.0.0"),
					benchmarkStringAttr("excluded", "drop-me"),
				},
			},
			ScopeSpans: []*trace_v1.ScopeSpans{{}},
		}
		for s := 0; s < spansPerResource; s++ {
			traceID := benchmarkTraceID(r)
			if !sameTracePerResource {
				traceID = benchmarkTraceID(r*spansPerResource + s)
			}
			rs.ScopeSpans[0].Spans = append(rs.ScopeSpans[0].Spans, &trace_v1.Span{
				TraceId:           traceID,
				SpanId:            benchmarkSpanID(r*spansPerResource + s + 1),
				Name:              "GET /api/:id",
				Kind:              trace_v1.Span_SPAN_KIND_SERVER,
				StartTimeUnixNano: uint64(1_700_000_000_000_000_000 + s),
				EndTimeUnixNano:   uint64(1_700_000_000_100_000_000 + s),
				Status:            &trace_v1.Status{Code: trace_v1.Status_STATUS_CODE_OK},
				Attributes: []*common_v1.KeyValue{
					benchmarkStringAttr("http.method", "GET"),
					benchmarkStringAttr("http.route", "/api/:id"),
					benchmarkStringAttr("http.status_code", "200"),
					benchmarkStringAttr("span.attr.1", "one"),
					benchmarkStringAttr("span.attr.2", "two"),
				},
			})
		}
		req.Batches = append(req.Batches, rs)
	}
	return req
}

func benchmarkTraceID(i int) []byte {
	var id [16]byte
	binary.BigEndian.PutUint64(id[8:], uint64(i+1))
	return id[:]
}

func benchmarkSpanID(i int) []byte {
	var id [8]byte
	binary.BigEndian.PutUint64(id[:], uint64(i+1))
	return id[:]
}

func benchmarkStringAttr(key, value string) *common_v1.KeyValue {
	return &common_v1.KeyValue{
		Key: key,
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{
			StringValue: value,
		}},
	}
}
