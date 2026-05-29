package generator

import (
	"context"
	"encoding/binary"
	"flag"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/gogo/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	promdto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prometheus/model/exemplar"
	promhistogram "github.com/prometheus/prometheus/model/histogram"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/metadata"
	prometheus_storage "github.com/prometheus/prometheus/storage"

	"github.com/grafana/tempo/modules/generator/processor"
	generator_storage "github.com/grafana/tempo/modules/generator/storage"
	"github.com/grafana/tempo/modules/overrides/histograms"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resource_v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

func BenchmarkPushSpansConfigurations(b *testing.B) {
	for _, tc := range []struct {
		name      string
		overrides func(*mockOverrides)
		request   *tempopb.PushSpansRequest
	}{
		{
			name: "spanmetrics_default",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}}
			},
			request: benchmarkGeneratorRequest(4, false),
		},
		{
			name: "spanmetrics_target_info",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}}
				o.spanMetricsEnableTargetInfo = boolPtr(true)
				o.spanMetricsTargetInfoExcludedDimensions = []string{"excluded"}
			},
			request: benchmarkGeneratorRequest(4, false),
		},
		{
			name: "spanmetrics_target_info_all_filtered",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}}
				o.spanMetricsEnableTargetInfo = boolPtr(true)
				o.spanMetricsTargetInfoExcludedDimensions = []string{"excluded"}
				o.spanMetricsFilterPolicies = []filterconfig.FilterPolicy{{
					Exclude: &filterconfig.PolicyMatch{
						MatchType: filterconfig.Strict,
						Attributes: []filterconfig.MatchPolicyAttribute{{
							Key:   "span.http.method",
							Value: "GET",
						}},
					},
				}}
			},
			request: benchmarkGeneratorRequest(4, false),
		},
		{
			name: "spanmetrics_dimensions",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}}
				o.spanMetricsDimensions = []string{"k8s.cluster.name", "k8s.namespace.name", "http.method", "http.status_code"}
				o.spanMetricsDimensionMappings = []sharedconfig.DimensionMappings{
					{Name: "route_key", SourceLabel: []string{"http.method", "http.route", "http.status_code"}, Join: ":"},
				}
			},
			request: benchmarkGeneratorRequest(4, false),
		},
		{
			name: "servicegraphs_default",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.ServiceGraphsName: {}}
			},
			request: benchmarkGeneratorRequest(2, true),
		},
		{
			name: "servicegraphs_span_multiplier",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.ServiceGraphsName: {}}
				o.serviceGraphsSpanMultiplierKey = "sample.rate"
			},
			request: benchmarkGeneratorServiceGraphRequest(100, true),
		},
		{
			name: "spanmetrics_prod_7dims_target_info_filters",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}}
				benchmarkGeneratorProdOverrides(o)
			},
			request: benchmarkGeneratorProdRequest(4, false),
		},
		{
			name: "servicegraphs_prod_7dims_prefix",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
			},
			request: benchmarkGeneratorProdRequest(2, true),
		},
		{
			name: "combined_target_info",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				o.spanMetricsEnableTargetInfo = boolPtr(true)
				o.spanMetricsTargetInfoExcludedDimensions = []string{"excluded"}
			},
			request: benchmarkGeneratorRequest(4, true),
		},
		{
			name: "combined_native_histograms",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				o.spanMetricsEnableTargetInfo = boolPtr(true)
				o.spanMetricsTargetInfoExcludedDimensions = []string{"excluded"}
				o.nativeHistograms = histograms.HistogramMethodBoth
			},
			request: benchmarkGeneratorRequest(4, true),
		},
		{
			name: "combined_prod_7dims_target_info_servicegraphs_prefix_filters",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
			},
			request: benchmarkGeneratorProdRequest(4, true),
		},
		{
			name: "combined_prod_7dims_native_histograms",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
				o.nativeHistograms = histograms.HistogramMethodBoth
			},
			request: benchmarkGeneratorProdRequest(4, true),
		},
		{
			name: "combined_prod_7dims_native_only_histograms",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
				o.nativeHistograms = histograms.HistogramMethodNative
			},
			request: benchmarkGeneratorProdRequest(4, true),
		},
	} {
		b.Run(tc.name, func(b *testing.B) {
			inst := benchmarkGeneratorInstance(b, tc.overrides)
			defer inst.shutdown()

			ctx := context.Background()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				inst.pushSpans(ctx, tc.request)
			}
		})
	}
}

func BenchmarkPushSpansProductionCardinality(b *testing.B) {
	for _, tc := range []struct {
		name       string
		overrides  func(*mockOverrides)
		seed       func(context.Context, *instance)
		request    *tempopb.PushSpansRequest
		requireEnv string
	}{
		{
			name: "spanmetrics_prod_7dims_100k_series_steady",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}}
				benchmarkGeneratorProdOverrides(o)
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalitySpanmetrics(ctx, inst, benchmarkGeneratorProdSpanmetrics100kResources)
			},
			request: benchmarkGeneratorProdHighCardinalitySpanmetricsRequest(0, benchmarkGeneratorHighCardinalityTimedResources),
		},
		{
			name: "servicegraphs_prod_7dims_100k_series_steady",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdServiceGraphs100kEdges)
			},
			request: benchmarkGeneratorProdHighCardinalityServiceGraphRequest(0, benchmarkGeneratorHighCardinalityTimedEdges),
		},
		{
			name: "combined_prod_7dims_100k_series_steady",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombined100kEdges)
			},
			request: benchmarkGeneratorProdHighCardinalityServiceGraphRequest(0, benchmarkGeneratorHighCardinalityTimedEdges),
		},
		{
			name: "combined_prod_7dims_100k_series_native_steady",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
				o.nativeHistograms = histograms.HistogramMethodBoth
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombinedNative100kEdges)
			},
			request: benchmarkGeneratorProdHighCardinalityServiceGraphRequest(0, benchmarkGeneratorHighCardinalityTimedEdges),
		},
		{
			name: "combined_prod_7dims_100k_series_native_only_steady",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
				o.nativeHistograms = histograms.HistogramMethodNative
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombinedNativeOnly100kEdges)
			},
			request: benchmarkGeneratorProdHighCardinalityServiceGraphRequest(0, benchmarkGeneratorHighCardinalityTimedEdges),
		},
		{
			name: "combined_prod_7dims_1m_series_steady",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombined1MEdges)
			},
			request:    benchmarkGeneratorProdHighCardinalityServiceGraphRequest(0, benchmarkGeneratorHighCardinalityTimedEdges),
			requireEnv: "TEMPO_GENERATOR_BENCH_1M",
		},
	} {
		b.Run(tc.name, func(b *testing.B) {
			if tc.requireEnv != "" && os.Getenv(tc.requireEnv) == "" {
				b.Skipf("set %s=1 to run this high-cardinality benchmark", tc.requireEnv)
			}

			inst := benchmarkGeneratorInstance(b, tc.overrides)
			defer inst.shutdown()

			ctx := context.Background()
			tc.seed(ctx, inst)
			runtime.GC()

			activeSeries := benchmarkGeneratorActiveSeries(b)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				inst.pushSpans(ctx, tc.request)
			}
			b.ReportMetric(activeSeries, "active_series")
		})
	}
}

func BenchmarkDecodePushSpansProductionCardinality(b *testing.B) {
	for _, tc := range []struct {
		name      string
		decoder   func() ingest.GeneratorCodec
		payload   func(testing.TB, *tempopb.PushSpansRequest) []byte
		overrides func(*mockOverrides)
		seed      func(context.Context, *instance)
		request   *tempopb.PushSpansRequest
	}{
		{
			name:    "combined_prod_7dims_100k_series_otlp_decode_push",
			decoder: func() ingest.GeneratorCodec { return ingest.NewOTLPDecoder() },
			payload: benchmarkGeneratorOTLPPayload,
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombined100kEdges)
			},
			request: benchmarkGeneratorProdHighCardinalityServiceGraphRequest(0, benchmarkGeneratorHighCardinalityTimedEdges),
		},
		{
			name:    "combined_prod_7dims_100k_series_push_bytes_decode_push",
			decoder: func() ingest.GeneratorCodec { return ingest.NewPushBytesDecoder() },
			payload: benchmarkGeneratorPushBytesPayload,
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombined100kEdges)
			},
			request: benchmarkGeneratorProdHighCardinalityServiceGraphRequest(0, benchmarkGeneratorHighCardinalityTimedEdges),
		},
		{
			name:    "combined_prod_7dims_native_only_100k_series_otlp_decode_push",
			decoder: func() ingest.GeneratorCodec { return ingest.NewOTLPDecoder() },
			payload: benchmarkGeneratorOTLPPayload,
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
				o.nativeHistograms = histograms.HistogramMethodNative
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombinedNativeOnly100kEdges)
			},
			request: benchmarkGeneratorProdHighCardinalityServiceGraphRequest(0, benchmarkGeneratorHighCardinalityTimedEdges),
		},
		{
			name:    "combined_prod_7dims_native_only_100k_series_push_bytes_decode_push",
			decoder: func() ingest.GeneratorCodec { return ingest.NewPushBytesDecoder() },
			payload: benchmarkGeneratorPushBytesPayload,
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
				o.nativeHistograms = histograms.HistogramMethodNative
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombinedNativeOnly100kEdges)
			},
			request: benchmarkGeneratorProdHighCardinalityServiceGraphRequest(0, benchmarkGeneratorHighCardinalityTimedEdges),
		},
	} {
		b.Run(tc.name, func(b *testing.B) {
			inst := benchmarkGeneratorInstance(b, tc.overrides)
			defer inst.shutdown()

			ctx := context.Background()
			tc.seed(ctx, inst)
			payload := tc.payload(b, tc.request)
			decoder := tc.decoder()
			runtime.GC()

			activeSeries := benchmarkGeneratorActiveSeries(b)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				iterator, err := decoder.Decode(payload)
				if err != nil {
					b.Fatal(err)
				}
				for req, err := range iterator {
					if err != nil {
						b.Fatal(err)
					}
					inst.pushSpans(ctx, req)
				}
			}
			b.ReportMetric(activeSeries, "active_series")
		})
	}
}

func BenchmarkPushSpansProductionChurn(b *testing.B) {
	if os.Getenv(benchmarkGeneratorChurnEnv) == "" {
		b.Skipf("set %s=1 to run this high-cardinality churn benchmark", benchmarkGeneratorChurnEnv)
	}

	for _, tc := range []struct {
		name      string
		overrides func(*mockOverrides)
		seed      func(context.Context, *instance)
		request   func(int) *tempopb.PushSpansRequest
	}{
		{
			name: "spanmetrics_prod_7dims_100k_series_churn",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}}
				benchmarkGeneratorProdOverrides(o)
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalitySpanmetrics(ctx, inst, benchmarkGeneratorProdSpanmetrics100kResources)
			},
			request: func(iter int) *tempopb.PushSpansRequest {
				start := benchmarkGeneratorProdSpanmetrics100kResources + iter*benchmarkGeneratorHighCardinalityTimedResources
				return benchmarkGeneratorProdHighCardinalitySpanmetricsRequest(start, benchmarkGeneratorHighCardinalityTimedResources)
			},
		},
		{
			name: "servicegraphs_prod_7dims_100k_series_churn",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdServiceGraphs100kEdges)
			},
			request: func(iter int) *tempopb.PushSpansRequest {
				start := benchmarkGeneratorProdServiceGraphs100kEdges + iter*benchmarkGeneratorHighCardinalityTimedEdges
				return benchmarkGeneratorProdHighCardinalityServiceGraphRequest(start, benchmarkGeneratorHighCardinalityTimedEdges)
			},
		},
		{
			name: "combined_prod_7dims_100k_series_churn",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombined100kEdges)
			},
			request: func(iter int) *tempopb.PushSpansRequest {
				start := benchmarkGeneratorProdCombined100kEdges + iter*benchmarkGeneratorHighCardinalityTimedEdges
				return benchmarkGeneratorProdHighCardinalityServiceGraphRequest(start, benchmarkGeneratorHighCardinalityTimedEdges)
			},
		},
		{
			name: "combined_prod_7dims_native_100k_series_churn",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
				o.nativeHistograms = histograms.HistogramMethodBoth
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombinedNative100kEdges)
			},
			request: func(iter int) *tempopb.PushSpansRequest {
				start := benchmarkGeneratorProdCombinedNative100kEdges + iter*benchmarkGeneratorHighCardinalityTimedEdges
				return benchmarkGeneratorProdHighCardinalityServiceGraphRequest(start, benchmarkGeneratorHighCardinalityTimedEdges)
			},
		},
		{
			name: "combined_prod_7dims_native_only_100k_series_churn",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
				o.nativeHistograms = histograms.HistogramMethodNative
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombinedNativeOnly100kEdges)
			},
			request: func(iter int) *tempopb.PushSpansRequest {
				start := benchmarkGeneratorProdCombinedNativeOnly100kEdges + iter*benchmarkGeneratorHighCardinalityTimedEdges
				return benchmarkGeneratorProdHighCardinalityServiceGraphRequest(start, benchmarkGeneratorHighCardinalityTimedEdges)
			},
		},
	} {
		b.Run(tc.name, func(b *testing.B) {
			inst := benchmarkGeneratorInstance(b, tc.overrides)
			defer inst.shutdown()

			ctx := context.Background()
			tc.seed(ctx, inst)
			runtime.GC()

			initialActiveSeries := benchmarkGeneratorActiveSeries(b)
			b.SetBytes(int64(proto.Size(tc.request(0))))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				request := tc.request(i)
				b.StartTimer()

				inst.pushSpans(ctx, request)
			}
			b.StopTimer()
			b.ReportMetric(initialActiveSeries, "initial_active_series")
			b.ReportMetric(benchmarkGeneratorActiveSeries(b), "final_active_series")
		})
	}
}

func BenchmarkCollectMetricsProductionCardinality(b *testing.B) {
	for _, tc := range []struct {
		name      string
		overrides func(*mockOverrides)
		seed      func(context.Context, *instance)
	}{
		{
			name: "combined_prod_7dims_100k_series",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombined100kEdges)
			},
		},
		{
			name: "combined_prod_7dims_native_100k_series",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
				o.nativeHistograms = histograms.HistogramMethodBoth
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombinedNative100kEdges)
			},
		},
		{
			name: "combined_prod_7dims_native_only_100k_series",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
				o.nativeHistograms = histograms.HistogramMethodNative
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombinedNativeOnly100kEdges)
			},
		},
	} {
		b.Run(tc.name, func(b *testing.B) {
			inst := benchmarkGeneratorInstance(b, tc.overrides)
			defer inst.shutdown()

			ctx := context.Background()
			tc.seed(ctx, inst)
			inst.registry.CollectMetrics(ctx)
			runtime.GC()

			activeSeries := benchmarkGeneratorActiveSeries(b)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				inst.registry.CollectMetrics(ctx)
			}
			b.ReportMetric(activeSeries, "active_series")
		})
	}
}

func BenchmarkCollectMetricsProductionCardinalityWithRefs(b *testing.B) {
	for _, tc := range []struct {
		name      string
		overrides func(*mockOverrides)
		seed      func(context.Context, *instance)
	}{
		{
			name: "combined_prod_7dims_native_100k_series",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
				o.nativeHistograms = histograms.HistogramMethodBoth
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombinedNative100kEdges)
			},
		},
		{
			name: "combined_prod_7dims_native_only_100k_series",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
				o.nativeHistograms = histograms.HistogramMethodNative
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombinedNativeOnly100kEdges)
			},
		},
	} {
		b.Run(tc.name, func(b *testing.B) {
			storage := newBenchmarkRefStorage()
			inst := benchmarkGeneratorInstanceWithStorage(b, tc.overrides, storage)
			defer inst.shutdown()

			ctx := context.Background()
			tc.seed(ctx, inst)
			inst.registry.CollectMetrics(ctx)
			runtime.GC()

			activeSeries := benchmarkGeneratorActiveSeries(b)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				inst.registry.CollectMetrics(ctx)
			}
			b.ReportMetric(activeSeries, "active_series")
		})
	}
}

func BenchmarkCollectMetricsProductionCardinalityWithRefsAndExemplars(b *testing.B) {
	for _, tc := range []struct {
		name      string
		overrides func(*mockOverrides)
		seed      func(context.Context, *instance)
		request   *tempopb.PushSpansRequest
	}{
		{
			name: "combined_prod_7dims_native_100k_series",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
				o.nativeHistograms = histograms.HistogramMethodBoth
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombinedNative100kEdges)
			},
			request: benchmarkGeneratorProdHighCardinalityServiceGraphRequest(0, benchmarkGeneratorHighCardinalityTimedEdges),
		},
		{
			name: "combined_prod_7dims_native_only_100k_series",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
				o.nativeHistograms = histograms.HistogramMethodNative
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombinedNativeOnly100kEdges)
			},
			request: benchmarkGeneratorProdHighCardinalityServiceGraphRequest(0, benchmarkGeneratorHighCardinalityTimedEdges),
		},
	} {
		b.Run(tc.name, func(b *testing.B) {
			storage := newBenchmarkRefStorage()
			inst := benchmarkGeneratorInstanceWithStorage(b, tc.overrides, storage)
			defer inst.shutdown()

			ctx := context.Background()
			tc.seed(ctx, inst)
			inst.registry.CollectMetrics(ctx)
			runtime.GC()

			activeSeries := benchmarkGeneratorActiveSeries(b)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				inst.pushSpans(ctx, tc.request)
				b.StartTimer()
				inst.registry.CollectMetrics(ctx)
			}
			b.ReportMetric(activeSeries, "active_series")
		})
	}
}

func BenchmarkCollectMetricsProductionCardinalityRealStorage(b *testing.B) {
	const collectionsPerOp = 10

	for _, tc := range []struct {
		name      string
		overrides func(*mockOverrides)
		seed      func(context.Context, *instance)
	}{
		{
			name: "combined_prod_7dims_native_100k_series",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
				o.nativeHistograms = histograms.HistogramMethodBoth
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombinedNative100kEdges)
			},
		},
		{
			name: "combined_prod_7dims_native_only_100k_series",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
				o.nativeHistograms = histograms.HistogramMethodNative
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombinedNativeOnly100kEdges)
			},
		},
	} {
		b.Run(tc.name, func(b *testing.B) {
			inst := benchmarkGeneratorInstanceWithRealStorage(b, tc.overrides)
			defer inst.shutdown()

			ctx := context.Background()
			tc.seed(ctx, inst)
			inst.registry.CollectMetrics(ctx)
			runtime.GC()

			activeSeries := benchmarkGeneratorActiveSeries(b)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for j := 0; j < collectionsPerOp; j++ {
					inst.registry.CollectMetrics(ctx)
					if i+1 < b.N || j+1 < collectionsPerOp {
						b.StopTimer()
						time.Sleep(time.Millisecond)
						b.StartTimer()
					}
				}
			}
			b.ReportMetric(activeSeries, "active_series")
			b.ReportMetric(collectionsPerOp, "collections/op")
		})
	}
}

func BenchmarkPushCollectCycleProduction(b *testing.B) {
	for _, tc := range []struct {
		name      string
		overrides func(*mockOverrides)
		seed      func(context.Context, *instance)
		request   *tempopb.PushSpansRequest
	}{
		{
			name: "combined_prod_7dims_100k_series_10_pushes_per_collect",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombined100kEdges)
			},
			request: benchmarkGeneratorProdHighCardinalityServiceGraphRequest(0, benchmarkGeneratorHighCardinalityTimedEdges),
		},
		{
			name: "combined_prod_7dims_native_100k_series_10_pushes_per_collect",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
				o.nativeHistograms = histograms.HistogramMethodBoth
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombinedNative100kEdges)
			},
			request: benchmarkGeneratorProdHighCardinalityServiceGraphRequest(0, benchmarkGeneratorHighCardinalityTimedEdges),
		},
		{
			name: "combined_prod_7dims_native_only_100k_series_10_pushes_per_collect",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
				o.nativeHistograms = histograms.HistogramMethodNative
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombinedNativeOnly100kEdges)
			},
			request: benchmarkGeneratorProdHighCardinalityServiceGraphRequest(0, benchmarkGeneratorHighCardinalityTimedEdges),
		},
	} {
		b.Run(tc.name, func(b *testing.B) {
			inst := benchmarkGeneratorInstance(b, tc.overrides)
			defer inst.shutdown()

			ctx := context.Background()
			tc.seed(ctx, inst)
			inst.pushSpans(ctx, tc.request)
			inst.registry.CollectMetrics(ctx)
			runtime.GC()

			activeSeries := benchmarkGeneratorActiveSeries(b)
			b.SetBytes(int64(proto.Size(tc.request) * benchmarkGeneratorPushesPerCollect))
			b.ReportMetric(float64(benchmarkGeneratorPushesPerCollect), "pushes/op")
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for j := 0; j < benchmarkGeneratorPushesPerCollect; j++ {
					inst.pushSpans(ctx, tc.request)
				}
				inst.registry.CollectMetrics(ctx)
			}
			b.ReportMetric(activeSeries, "active_series")
		})
	}
}

func BenchmarkPushCollectCycleProductionChurn(b *testing.B) {
	if os.Getenv(benchmarkGeneratorChurnEnv) == "" {
		b.Skipf("set %s=1 to run this high-cardinality churn benchmark", benchmarkGeneratorChurnEnv)
	}

	for _, tc := range []struct {
		name      string
		overrides func(*mockOverrides)
		seed      func(context.Context, *instance)
		request   func(int) *tempopb.PushSpansRequest
	}{
		{
			name: "combined_prod_7dims_100k_series_10_pushes_per_collect_churn",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombined100kEdges)
			},
			request: func(iter int) *tempopb.PushSpansRequest {
				start := benchmarkGeneratorProdCombined100kEdges + iter*benchmarkGeneratorHighCardinalityTimedEdges
				return benchmarkGeneratorProdHighCardinalityServiceGraphRequest(start, benchmarkGeneratorHighCardinalityTimedEdges)
			},
		},
		{
			name: "combined_prod_7dims_native_100k_series_10_pushes_per_collect_churn",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
				o.nativeHistograms = histograms.HistogramMethodBoth
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombinedNative100kEdges)
			},
			request: func(iter int) *tempopb.PushSpansRequest {
				start := benchmarkGeneratorProdCombinedNative100kEdges + iter*benchmarkGeneratorHighCardinalityTimedEdges
				return benchmarkGeneratorProdHighCardinalityServiceGraphRequest(start, benchmarkGeneratorHighCardinalityTimedEdges)
			},
		},
		{
			name: "combined_prod_7dims_native_only_100k_series_10_pushes_per_collect_churn",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
				o.nativeHistograms = histograms.HistogramMethodNative
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombinedNativeOnly100kEdges)
			},
			request: func(iter int) *tempopb.PushSpansRequest {
				start := benchmarkGeneratorProdCombinedNativeOnly100kEdges + iter*benchmarkGeneratorHighCardinalityTimedEdges
				return benchmarkGeneratorProdHighCardinalityServiceGraphRequest(start, benchmarkGeneratorHighCardinalityTimedEdges)
			},
		},
	} {
		b.Run(tc.name, func(b *testing.B) {
			storage := newBenchmarkRefStorage()
			inst := benchmarkGeneratorInstanceWithStorage(b, tc.overrides, storage)
			defer inst.shutdown()

			ctx := context.Background()
			tc.seed(ctx, inst)
			inst.registry.CollectMetrics(ctx)
			runtime.GC()

			initialActiveSeries := benchmarkGeneratorActiveSeries(b)
			b.SetBytes(int64(proto.Size(tc.request(0)) * benchmarkGeneratorPushesPerCollect))
			b.ReportMetric(float64(benchmarkGeneratorPushesPerCollect), "pushes/op")
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for j := 0; j < benchmarkGeneratorPushesPerCollect; j++ {
					b.StopTimer()
					request := tc.request(i*benchmarkGeneratorPushesPerCollect + j)
					b.StartTimer()

					inst.pushSpans(ctx, request)
				}
				inst.registry.CollectMetrics(ctx)
			}
			b.StopTimer()
			b.ReportMetric(initialActiveSeries, "initial_active_series")
			b.ReportMetric(benchmarkGeneratorActiveSeries(b), "final_active_series")
		})
	}
}

func BenchmarkRetainedMemoryProductionCardinality(b *testing.B) {
	for _, tc := range []struct {
		name      string
		overrides func(*mockOverrides)
		seed      func(context.Context, *instance)
	}{
		{
			name: "combined_prod_7dims_100k_series",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombined100kEdges)
			},
		},
		{
			name: "combined_prod_7dims_native_100k_series",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
				o.nativeHistograms = histograms.HistogramMethodBoth
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombinedNative100kEdges)
			},
		},
		{
			name: "combined_prod_7dims_native_only_100k_series",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
				o.nativeHistograms = histograms.HistogramMethodNative
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombinedNativeOnly100kEdges)
			},
		},
	} {
		b.Run(tc.name, func(b *testing.B) {
			var heapBytesPerSeries float64
			var objectsPerSeries float64
			var activeSeriesTotal float64

			ctx := context.Background()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				runtime.GC()

				var before runtime.MemStats
				runtime.ReadMemStats(&before)

				inst := benchmarkGeneratorInstance(b, tc.overrides)
				tc.seed(ctx, inst)
				inst.registry.CollectMetrics(ctx)
				activeSeries := benchmarkGeneratorActiveSeries(b)
				runtime.GC()

				var after runtime.MemStats
				runtime.ReadMemStats(&after)

				heapBytesPerSeries += float64(benchmarkGeneratorMemStatsDelta(after.HeapAlloc, before.HeapAlloc)) / activeSeries
				objectsPerSeries += float64(benchmarkGeneratorMemStatsDelta(after.HeapObjects, before.HeapObjects)) / activeSeries
				activeSeriesTotal += activeSeries

				inst.shutdown()
				b.StartTimer()
			}

			if b.N > 0 {
				b.ReportMetric(activeSeriesTotal/float64(b.N), "active_series")
				b.ReportMetric(heapBytesPerSeries/float64(b.N), "heap_bytes/active_series")
				b.ReportMetric(objectsPerSeries/float64(b.N), "objects/active_series")
			}
		})
	}
}

func benchmarkGeneratorMemStatsDelta(after, before uint64) uint64 {
	if after < before {
		return 0
	}
	return after - before
}

const (
	// The 100k cases seed enough metric label sets to create roughly 100k active
	// time series before timing steady-state updates.
	benchmarkGeneratorProdSpanmetrics100kResources    = 5000
	benchmarkGeneratorProdServiceGraphs100kEdges      = 2858
	benchmarkGeneratorProdCombined100kEdges           = 1334
	benchmarkGeneratorProdCombinedNative100kEdges     = 1266
	benchmarkGeneratorProdCombinedNativeOnly100kEdges = 9500
	benchmarkGeneratorProdCombined1MEdges             = 13334
	benchmarkGeneratorHighCardinalityTimedResources   = 400
	benchmarkGeneratorHighCardinalityTimedEdges       = 200
	benchmarkGeneratorHighCardinalitySeedResourceSize = 500
	benchmarkGeneratorHighCardinalitySeedEdgeSize     = 250
	benchmarkGeneratorPushesPerCollect                = 10
	benchmarkGeneratorTenant                          = "bench-tenant"
	benchmarkGeneratorChurnEnv                        = "TEMPO_GENERATOR_BENCH_CHURN"
)

var benchmarkGeneratorProdDimensions = []string{
	"cloud.availability_zone",
	"cloud.region",
	"deployment.environment",
	"k8s.cluster.name",
	"k8s.namespace.name",
	"service.namespace",
	"service.version",
}

var benchmarkGeneratorProdBuckets = []float64{
	0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10,
}

var benchmarkGeneratorProdTargetInfoExcludedDimensions = []string{
	"k8s.pod.start_time",
	"os.description",
	"os.type",
	"process.command_args",
	"process.executable.path",
	"process.pid",
	"process.runtime.description",
	"process.runtime.name",
	"process.runtime.version",
}

var benchmarkGeneratorProdPeerAttributes = []string{
	"peer.service",
	"db.namespace",
	"db.name",
	"db.system",
	"db.system.name",
	"messaging.system",
	"db.url",
	"server.address",
	"net.peer.name",
}

var benchmarkGeneratorProdFilterPolicies = []filterconfig.FilterPolicy{
	{
		Include: &filterconfig.PolicyMatch{
			MatchType: filterconfig.Regex,
			Attributes: []filterconfig.MatchPolicyAttribute{{
				Key:   "kind",
				Value: "SPAN_KIND_(SERVER|CONSUMER|CLIENT|PRODUCER)",
			}},
		},
	},
	{
		Exclude: &filterconfig.PolicyMatch{
			MatchType: filterconfig.Strict,
			Attributes: []filterconfig.MatchPolicyAttribute{
				{Key: "kind", Value: "SPAN_KIND_SERVER"},
				{Key: "span.url.path", Value: "/healthz"},
			},
		},
	},
	{
		Exclude: &filterconfig.PolicyMatch{
			MatchType: filterconfig.Strict,
			Attributes: []filterconfig.MatchPolicyAttribute{
				{Key: "kind", Value: "SPAN_KIND_SERVER"},
				{Key: "span.url.path", Value: "/health"},
			},
		},
	},
	{
		Exclude: &filterconfig.PolicyMatch{
			MatchType: filterconfig.Strict,
			Attributes: []filterconfig.MatchPolicyAttribute{{
				Key:   "resource.span.metrics.skip",
				Value: true,
			}},
		},
	},
}

func benchmarkGeneratorProdOverrides(o *mockOverrides) {
	o.nativeHistogramBucketFactor = 1.1
	o.nativeHistogramMaxBucketNumber = 16
	o.nativeHistogramMinResetDuration = 15 * time.Minute

	o.spanMetricsDimensions = benchmarkGeneratorProdDimensions
	o.spanMetricsEnableTargetInfo = boolPtr(true)
	o.spanMetricsTargetInfoExcludedDimensions = benchmarkGeneratorProdTargetInfoExcludedDimensions
	o.spanMetricsHistogramBuckets = benchmarkGeneratorProdBuckets
	o.spanMetricsFilterPolicies = benchmarkGeneratorProdFilterPolicies

	o.serviceGraphsDimensions = benchmarkGeneratorProdDimensions
	o.serviceGraphsEnableClientServerPrefix = true
	o.serviceGraphsPeerAttributes = benchmarkGeneratorProdPeerAttributes
	o.serviceGraphsHistogramBuckets = benchmarkGeneratorProdBuckets
}

func benchmarkGeneratorInstance(b *testing.B, tune func(*mockOverrides)) *instance {
	return benchmarkGeneratorInstanceWithStorage(b, tune, &noopStorage{})
}

func benchmarkGeneratorInstanceWithStorage(b *testing.B, tune func(*mockOverrides), storage generator_storage.Storage) *instance {
	b.Helper()

	cfg := &Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.MetricsIngestionSlack = 365 * 24 * time.Hour

	o := &mockOverrides{}
	o.ingestionSlack = 365 * 24 * time.Hour
	tune(o)

	inst, err := newInstance(cfg, benchmarkGeneratorTenant, o, storage, log.NewNopLogger())
	if err != nil {
		b.Fatal(err)
	}
	return inst
}

func benchmarkGeneratorInstanceWithRealStorage(b *testing.B, tune func(*mockOverrides)) *instance {
	b.Helper()

	cfg := &Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.MetricsIngestionSlack = 365 * 24 * time.Hour

	o := &mockOverrides{}
	o.ingestionSlack = 365 * 24 * time.Hour
	tune(o)

	storageCfg := &generator_storage.Config{}
	storageCfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	storageCfg.Path = b.TempDir()

	var wal generator_storage.Storage
	var err error
	benchmarkWithDiscardedStdout(b, func() {
		wal, err = generator_storage.New(storageCfg, o, benchmarkGeneratorTenant, prometheus.NewRegistry(), log.NewNopLogger())
	})
	if err != nil {
		b.Fatal(err)
	}

	inst, err := newInstance(cfg, benchmarkGeneratorTenant, o, wal, log.NewNopLogger())
	if err != nil {
		_ = wal.Close()
		b.Fatal(err)
	}
	return inst
}

func benchmarkGeneratorOTLPPayload(tb testing.TB, req *tempopb.PushSpansRequest) []byte {
	tb.Helper()

	trace := tempopb.Trace{ResourceSpans: req.Batches}
	data, err := trace.Marshal()
	if err != nil {
		tb.Fatal(err)
	}
	return data
}

func benchmarkGeneratorPushBytesPayload(tb testing.TB, req *tempopb.PushSpansRequest) []byte {
	tb.Helper()

	traceBytes := benchmarkGeneratorOTLPPayload(tb, req)
	pushBytes := tempopb.PushBytesRequest{
		Traces: []tempopb.PreallocBytes{{Slice: traceBytes}},
		Ids:    [][]byte{{}},
	}
	data, err := pushBytes.Marshal()
	if err != nil {
		tb.Fatal(err)
	}
	return data
}

func benchmarkWithDiscardedStdout(b *testing.B, f func()) {
	b.Helper()

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		f()
		return
	}
	defer devNull.Close()

	stdout := os.Stdout
	os.Stdout = devNull
	defer func() {
		os.Stdout = stdout
	}()

	f()
}

func benchmarkGeneratorActiveSeries(b *testing.B) float64 {
	b.Helper()

	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		b.Fatal(err)
	}
	for _, family := range metricFamilies {
		if family.GetName() != "tempo_metrics_generator_registry_active_series" {
			continue
		}
		for _, metric := range family.GetMetric() {
			if !benchmarkGeneratorHasLabel(metric, "tenant", benchmarkGeneratorTenant) {
				continue
			}
			if metric.GetGauge() == nil {
				b.Fatalf("active series metric has no gauge")
			}
			activeSeries := metric.GetGauge().GetValue()
			if activeSeries == 0 {
				b.Fatalf("active series metric is zero")
			}
			return activeSeries
		}
	}
	b.Fatalf("active series metric for %q not found", benchmarkGeneratorTenant)
	return 0
}

func benchmarkGeneratorHasLabel(metric *promdto.Metric, name, value string) bool {
	for _, label := range metric.GetLabel() {
		if label.GetName() == name && label.GetValue() == value {
			return true
		}
	}
	return false
}

type benchmarkRefStorage struct {
	refs    map[uint64]prometheus_storage.SeriesRef
	nextRef prometheus_storage.SeriesRef
}

var _ generator_storage.Storage = (*benchmarkRefStorage)(nil)

func newBenchmarkRefStorage() *benchmarkRefStorage {
	return &benchmarkRefStorage{
		refs:    make(map[uint64]prometheus_storage.SeriesRef),
		nextRef: 1,
	}
}

func (s *benchmarkRefStorage) Appender(context.Context) prometheus_storage.Appender {
	return &benchmarkRefAppender{storage: s}
}

func (s *benchmarkRefStorage) Close() error {
	return nil
}

type benchmarkRefAppender struct {
	storage *benchmarkRefStorage
}

var _ prometheus_storage.Appender = (*benchmarkRefAppender)(nil)

func (a *benchmarkRefAppender) Append(ref prometheus_storage.SeriesRef, l labels.Labels, _ int64, _ float64) (prometheus_storage.SeriesRef, error) {
	return a.refFor(ref, l), nil
}

func (a *benchmarkRefAppender) AppendExemplar(ref prometheus_storage.SeriesRef, _ labels.Labels, _ exemplar.Exemplar) (prometheus_storage.SeriesRef, error) {
	return ref, nil
}

func (a *benchmarkRefAppender) AppendHistogram(ref prometheus_storage.SeriesRef, l labels.Labels, _ int64, _ *promhistogram.Histogram, _ *promhistogram.FloatHistogram) (prometheus_storage.SeriesRef, error) {
	return a.refFor(ref, l), nil
}

func (a *benchmarkRefAppender) Commit() error {
	return nil
}

func (a *benchmarkRefAppender) Rollback() error {
	return nil
}

func (a *benchmarkRefAppender) SetOptions(_ *prometheus_storage.AppendOptions) {}

func (a *benchmarkRefAppender) UpdateMetadata(prometheus_storage.SeriesRef, labels.Labels, metadata.Metadata) (prometheus_storage.SeriesRef, error) {
	return 0, nil
}

func (a *benchmarkRefAppender) AppendCTZeroSample(ref prometheus_storage.SeriesRef, l labels.Labels, _, _ int64) (prometheus_storage.SeriesRef, error) {
	return a.refFor(ref, l), nil
}

func (a *benchmarkRefAppender) AppendSTZeroSample(ref prometheus_storage.SeriesRef, l labels.Labels, _, _ int64) (prometheus_storage.SeriesRef, error) {
	return a.refFor(ref, l), nil
}

func (a *benchmarkRefAppender) AppendHistogramCTZeroSample(ref prometheus_storage.SeriesRef, l labels.Labels, _, _ int64, _ *promhistogram.Histogram, _ *promhistogram.FloatHistogram) (prometheus_storage.SeriesRef, error) {
	return a.refFor(ref, l), nil
}

func (a *benchmarkRefAppender) AppendHistogramSTZeroSample(ref prometheus_storage.SeriesRef, l labels.Labels, _, _ int64, _ *promhistogram.Histogram, _ *promhistogram.FloatHistogram) (prometheus_storage.SeriesRef, error) {
	return a.refFor(ref, l), nil
}

func (a *benchmarkRefAppender) refFor(ref prometheus_storage.SeriesRef, l labels.Labels) prometheus_storage.SeriesRef {
	if ref != 0 {
		return ref
	}
	if l.IsEmpty() {
		return 0
	}

	hash := l.Hash()
	if ref, ok := a.storage.refs[hash]; ok {
		return ref
	}

	ref = a.storage.nextRef
	a.storage.nextRef++
	a.storage.refs[hash] = ref
	return ref
}

func benchmarkGeneratorRequest(resources int, includeServiceGraphPairs bool) *tempopb.PushSpansRequest {
	const spansPerResource = 100

	if includeServiceGraphPairs {
		return benchmarkGeneratorServiceGraphRequest(resources*spansPerResource/2, false)
	}

	req := &tempopb.PushSpansRequest{
		Batches: make([]*trace_v1.ResourceSpans, 0, resources),
	}
	now := uint64(time.Now().UnixNano())
	for r := 0; r < resources; r++ {
		rs := benchmarkGeneratorResource("svc-"+strconv.Itoa(r), r)
		for s := 0; s < spansPerResource; s++ {
			rs.ScopeSpans[0].Spans = append(rs.ScopeSpans[0].Spans, benchmarkGeneratorSpan(
				benchmarkGeneratorTraceID(r),
				benchmarkGeneratorSpanID(r*spansPerResource+s+1),
				nil,
				trace_v1.Span_SPAN_KIND_SERVER,
				now+uint64(s),
			))
		}
		req.Batches = append(req.Batches, rs)
	}
	return req
}

func benchmarkGeneratorServiceGraphRequest(edges int, spanMultiplier bool) *tempopb.PushSpansRequest {
	now := uint64(time.Now().UnixNano())
	client := benchmarkGeneratorResource("client", 0)
	server := benchmarkGeneratorResource("server", 1)
	for i := 0; i < edges; i++ {
		traceID := benchmarkGeneratorTraceID(i)
		clientSpanID := benchmarkGeneratorSpanID(i*2 + 1)
		client.ScopeSpans[0].Spans = append(client.ScopeSpans[0].Spans, benchmarkGeneratorSpan(
			traceID,
			clientSpanID,
			nil,
			trace_v1.Span_SPAN_KIND_CLIENT,
			now+uint64(i),
		))
		if spanMultiplier {
			client.ScopeSpans[0].Spans[len(client.ScopeSpans[0].Spans)-1].Attributes = append(
				client.ScopeSpans[0].Spans[len(client.ScopeSpans[0].Spans)-1].Attributes,
				benchmarkGeneratorDoubleAttr("sample.rate", 0.5),
			)
		}

		server.ScopeSpans[0].Spans = append(server.ScopeSpans[0].Spans, benchmarkGeneratorSpan(
			traceID,
			benchmarkGeneratorSpanID(i*2+2),
			clientSpanID,
			trace_v1.Span_SPAN_KIND_SERVER,
			now+uint64(i),
		))
		if spanMultiplier {
			server.ScopeSpans[0].Spans[len(server.ScopeSpans[0].Spans)-1].Attributes = append(
				server.ScopeSpans[0].Spans[len(server.ScopeSpans[0].Spans)-1].Attributes,
				benchmarkGeneratorDoubleAttr("sample.rate", 0.5),
			)
		}
	}
	return &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{client, server}}
}

func benchmarkGeneratorProdRequest(resources int, includeServiceGraphPairs bool) *tempopb.PushSpansRequest {
	const spansPerResource = 100

	if includeServiceGraphPairs {
		return benchmarkGeneratorProdServiceGraphRequest(resources * spansPerResource / 2)
	}

	req := &tempopb.PushSpansRequest{
		Batches: make([]*trace_v1.ResourceSpans, 0, resources),
	}
	now := uint64(time.Now().UnixNano())
	for r := 0; r < resources; r++ {
		rs := benchmarkGeneratorProdResource("svc-"+strconv.Itoa(r), r)
		for s := 0; s < spansPerResource; s++ {
			rs.ScopeSpans[0].Spans = append(rs.ScopeSpans[0].Spans, benchmarkGeneratorProdSpan(
				benchmarkGeneratorTraceID(r),
				benchmarkGeneratorSpanID(r*spansPerResource+s+1),
				nil,
				trace_v1.Span_SPAN_KIND_SERVER,
				now+uint64(s),
			))
		}
		req.Batches = append(req.Batches, rs)
	}
	return req
}

func benchmarkGeneratorProdServiceGraphRequest(edges int) *tempopb.PushSpansRequest {
	now := uint64(time.Now().UnixNano())
	client := benchmarkGeneratorProdResource("client", 0)
	server := benchmarkGeneratorProdResource("server", 1)
	for i := 0; i < edges; i++ {
		traceID := benchmarkGeneratorTraceID(i)
		clientSpanID := benchmarkGeneratorSpanID(i*2 + 1)
		client.ScopeSpans[0].Spans = append(client.ScopeSpans[0].Spans, benchmarkGeneratorProdSpan(
			traceID,
			clientSpanID,
			nil,
			trace_v1.Span_SPAN_KIND_CLIENT,
			now+uint64(i),
		))

		server.ScopeSpans[0].Spans = append(server.ScopeSpans[0].Spans, benchmarkGeneratorProdSpan(
			traceID,
			benchmarkGeneratorSpanID(i*2+2),
			clientSpanID,
			trace_v1.Span_SPAN_KIND_SERVER,
			now+uint64(i),
		))
	}
	return &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{client, server}}
}

func benchmarkGeneratorSeedHighCardinalitySpanmetrics(ctx context.Context, inst *instance, resources int) {
	for start := 0; start < resources; start += benchmarkGeneratorHighCardinalitySeedResourceSize {
		count := min(benchmarkGeneratorHighCardinalitySeedResourceSize, resources-start)
		inst.pushSpans(ctx, benchmarkGeneratorProdHighCardinalitySpanmetricsRequest(start, count))
	}
}

func benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx context.Context, inst *instance, edges int) {
	for start := 0; start < edges; start += benchmarkGeneratorHighCardinalitySeedEdgeSize {
		count := min(benchmarkGeneratorHighCardinalitySeedEdgeSize, edges-start)
		inst.pushSpans(ctx, benchmarkGeneratorProdHighCardinalityServiceGraphRequest(start, count))
	}
}

func benchmarkGeneratorProdHighCardinalitySpanmetricsRequest(startResource, resources int) *tempopb.PushSpansRequest {
	req := &tempopb.PushSpansRequest{
		Batches: make([]*trace_v1.ResourceSpans, 0, resources),
	}
	now := uint64(time.Now().UnixNano())
	for r := 0; r < resources; r++ {
		idx := startResource + r
		rs := benchmarkGeneratorProdResource("svc-"+strconv.Itoa(idx), idx)
		rs.ScopeSpans[0].Spans = append(rs.ScopeSpans[0].Spans, benchmarkGeneratorProdSpan(
			benchmarkGeneratorTraceID(idx),
			benchmarkGeneratorSpanID(idx+1),
			nil,
			trace_v1.Span_SPAN_KIND_SERVER,
			now+uint64(r),
		))
		req.Batches = append(req.Batches, rs)
	}
	return req
}

func benchmarkGeneratorProdHighCardinalityServiceGraphRequest(startEdge, edges int) *tempopb.PushSpansRequest {
	req := &tempopb.PushSpansRequest{
		Batches: make([]*trace_v1.ResourceSpans, 0, edges*2),
	}
	now := uint64(time.Now().UnixNano())
	for i := 0; i < edges; i++ {
		edge := startEdge + i
		traceID := benchmarkGeneratorTraceID(edge)
		clientSpanID := benchmarkGeneratorSpanID(edge*2 + 1)

		client := benchmarkGeneratorProdResource("client-"+strconv.Itoa(edge), edge*2)
		client.ScopeSpans[0].Spans = append(client.ScopeSpans[0].Spans, benchmarkGeneratorProdSpan(
			traceID,
			clientSpanID,
			nil,
			trace_v1.Span_SPAN_KIND_CLIENT,
			now+uint64(i),
		))
		req.Batches = append(req.Batches, client)

		server := benchmarkGeneratorProdResource("server-"+strconv.Itoa(edge), edge*2+1)
		server.ScopeSpans[0].Spans = append(server.ScopeSpans[0].Spans, benchmarkGeneratorProdSpan(
			traceID,
			benchmarkGeneratorSpanID(edge*2+2),
			clientSpanID,
			trace_v1.Span_SPAN_KIND_SERVER,
			now+uint64(i),
		))
		req.Batches = append(req.Batches, server)
	}
	return req
}

func benchmarkGeneratorResource(service string, idx int) *trace_v1.ResourceSpans {
	return &trace_v1.ResourceSpans{
		Resource: &resource_v1.Resource{
			Attributes: []*common_v1.KeyValue{
				benchmarkGeneratorStringAttr("service.name", service),
				benchmarkGeneratorStringAttr("service.namespace", "ns-"+strconv.Itoa(idx%2)),
				benchmarkGeneratorStringAttr("service.instance.id", "instance-"+strconv.Itoa(idx)),
				benchmarkGeneratorStringAttr("k8s.cluster.name", "cluster-a"),
				benchmarkGeneratorStringAttr("k8s.namespace.name", "namespace-"+strconv.Itoa(idx%4)),
				benchmarkGeneratorStringAttr("k8s.pod.name", "pod-"+strconv.Itoa(idx)),
				benchmarkGeneratorStringAttr("k8s.node.name", "node-"+strconv.Itoa(idx%8)),
				benchmarkGeneratorStringAttr("telemetry.sdk.language", "go"),
				benchmarkGeneratorStringAttr("telemetry.sdk.version", "1.0.0"),
				benchmarkGeneratorStringAttr("excluded", "drop-me"),
			},
		},
		ScopeSpans: []*trace_v1.ScopeSpans{{}},
	}
}

func benchmarkGeneratorProdResource(service string, idx int) *trace_v1.ResourceSpans {
	return &trace_v1.ResourceSpans{
		Resource: &resource_v1.Resource{
			Attributes: []*common_v1.KeyValue{
				benchmarkGeneratorStringAttr("service.name", service),
				benchmarkGeneratorStringAttr("service.namespace", "ns-"+strconv.Itoa(idx%2)),
				benchmarkGeneratorStringAttr("service.instance.id", "instance-"+strconv.Itoa(idx)),
				benchmarkGeneratorStringAttr("service.version", "1."+strconv.Itoa(idx%10)+".0"),
				benchmarkGeneratorStringAttr("cloud.availability_zone", "zone-"+strconv.Itoa(idx%3)),
				benchmarkGeneratorStringAttr("cloud.region", "region-"+strconv.Itoa(idx%4)),
				benchmarkGeneratorStringAttr("deployment.environment", "prod"),
				benchmarkGeneratorStringAttr("k8s.cluster.name", "cluster-a"),
				benchmarkGeneratorStringAttr("k8s.namespace.name", "namespace-"+strconv.Itoa(idx%4)),
				benchmarkGeneratorStringAttr("k8s.pod.name", "pod-"+strconv.Itoa(idx)),
				benchmarkGeneratorStringAttr("k8s.node.name", "node-"+strconv.Itoa(idx%8)),
				benchmarkGeneratorStringAttr("telemetry.sdk.language", "go"),
				benchmarkGeneratorStringAttr("telemetry.sdk.version", "1.0.0"),
				benchmarkGeneratorStringAttr("k8s.pod.start_time", "2026-05-07T00:00:00Z"),
				benchmarkGeneratorStringAttr("os.description", "linux"),
				benchmarkGeneratorStringAttr("os.type", "linux"),
				benchmarkGeneratorStringAttr("process.command_args", "tempo"),
				benchmarkGeneratorStringAttr("process.executable.path", "/tempo"),
				benchmarkGeneratorStringAttr("process.pid", strconv.Itoa(1000+idx)),
				benchmarkGeneratorStringAttr("process.runtime.description", "go"),
				benchmarkGeneratorStringAttr("process.runtime.name", "go"),
				benchmarkGeneratorStringAttr("process.runtime.version", "go1.24"),
				benchmarkGeneratorBoolAttr("resource.span.metrics.skip", false),
			},
		},
		ScopeSpans: []*trace_v1.ScopeSpans{{}},
	}
}

func benchmarkGeneratorSpan(traceID []byte, spanID []byte, parentSpanID []byte, kind trace_v1.Span_SpanKind, start uint64) *trace_v1.Span {
	return &trace_v1.Span{
		TraceId:           traceID,
		SpanId:            spanID,
		ParentSpanId:      parentSpanID,
		Name:              "GET /api/:id",
		Kind:              kind,
		StartTimeUnixNano: start,
		EndTimeUnixNano:   start + uint64(50*time.Millisecond),
		Status:            &trace_v1.Status{Code: trace_v1.Status_STATUS_CODE_OK},
		Attributes: []*common_v1.KeyValue{
			benchmarkGeneratorStringAttr("http.method", "GET"),
			benchmarkGeneratorStringAttr("http.route", "/api/:id"),
			benchmarkGeneratorStringAttr("http.status_code", "200"),
			benchmarkGeneratorStringAttr("span.attr.1", "one"),
			benchmarkGeneratorStringAttr("span.attr.2", "two"),
			benchmarkGeneratorStringAttr("peer.service", "server"),
		},
	}
}

func benchmarkGeneratorProdSpan(traceID []byte, spanID []byte, parentSpanID []byte, kind trace_v1.Span_SpanKind, start uint64) *trace_v1.Span {
	return &trace_v1.Span{
		TraceId:           traceID,
		SpanId:            spanID,
		ParentSpanId:      parentSpanID,
		Name:              "GET /api/:id",
		Kind:              kind,
		StartTimeUnixNano: start,
		EndTimeUnixNano:   start + uint64(50*time.Millisecond),
		Status:            &trace_v1.Status{Code: trace_v1.Status_STATUS_CODE_OK},
		Attributes: []*common_v1.KeyValue{
			benchmarkGeneratorStringAttr("http.method", "GET"),
			benchmarkGeneratorStringAttr("http.route", "/api/:id"),
			benchmarkGeneratorStringAttr("http.status_code", "200"),
			benchmarkGeneratorStringAttr("span.url.path", "/api/:id"),
			benchmarkGeneratorStringAttr("span.attr.1", "one"),
			benchmarkGeneratorStringAttr("span.attr.2", "two"),
			benchmarkGeneratorStringAttr("peer.service", "server"),
			benchmarkGeneratorStringAttr("server.address", "server"),
			benchmarkGeneratorStringAttr("net.peer.name", "server"),
		},
	}
}

func benchmarkGeneratorTraceID(i int) []byte {
	var id [16]byte
	binary.BigEndian.PutUint64(id[8:], uint64(i+1))
	return id[:]
}

func benchmarkGeneratorSpanID(i int) []byte {
	var id [8]byte
	binary.BigEndian.PutUint64(id[:], uint64(i+1))
	return id[:]
}

func benchmarkGeneratorStringAttr(key, value string) *common_v1.KeyValue {
	return &common_v1.KeyValue{
		Key: key,
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{
			StringValue: value,
		}},
	}
}

func benchmarkGeneratorBoolAttr(key string, value bool) *common_v1.KeyValue {
	return &common_v1.KeyValue{
		Key: key,
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_BoolValue{
			BoolValue: value,
		}},
	}
}

func benchmarkGeneratorDoubleAttr(key string, value float64) *common_v1.KeyValue {
	return &common_v1.KeyValue{
		Key: key,
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_DoubleValue{
			DoubleValue: value,
		}},
	}
}
