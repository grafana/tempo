package generator

import (
	"context"
	"encoding/binary"
	"flag"
	"os"
	"runtime"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/modules/generator/storage"
	"github.com/grafana/tempo/modules/overrides/histograms"
	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resource_v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/prometheus/model/histogram"
	"github.com/prometheus/prometheus/model/labels"
	prometheus_storage "github.com/prometheus/prometheus/storage"
)

// The BenchmarkInstancePushSpans* suites measure deterministic, processor-only
// pushSpans cost: fixed fixtures pushed into an instance backed by a noop
// storage, so WAL writes never enter the measurement. Complementary suites in
// this package: BenchmarkPushSpans (WAL-inclusive cost with randomized
// batches) and BenchmarkCollect (registry collection cost).

// BenchmarkInstancePushSpansConfigurations compares the per-tenant override
// matrix on a small fixed workload. Rows push different workloads (the
// servicegraphs rows push 200 spans per op, the spanmetrics and combined rows
// 400, with different batch shapes), so compare a row against the same row on
// another branch, or use the reported ns/span for rough cross-row comparison.
// The service-graph based rows repeatedly update one small edge set across two
// resources (hot-series updates with prod-shaped label extraction); series-map
// pressure at production cardinality is covered by
// BenchmarkInstancePushSpansProductionCardinality.
func BenchmarkInstancePushSpansConfigurations(b *testing.B) {
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
			request: benchmarkGeneratorSpanmetricsRequest(benchmarkGeneratorResource, benchmarkGeneratorSpan),
		},
		{
			name: "spanmetrics_target_info",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}}
				o.spanMetricsEnableTargetInfo = boolPtr(true)
				o.spanMetricsTargetInfoExcludedDimensions = []string{"excluded"}
			},
			request: benchmarkGeneratorSpanmetricsRequest(benchmarkGeneratorResource, benchmarkGeneratorSpan),
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
			request: benchmarkGeneratorSpanmetricsRequest(benchmarkGeneratorResource, benchmarkGeneratorSpan),
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
			request: benchmarkGeneratorSpanmetricsRequest(benchmarkGeneratorResource, benchmarkGeneratorSpan),
		},
		{
			name: "servicegraphs_default",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.ServiceGraphsName: {}}
			},
			request: benchmarkGeneratorServiceGraphRequest(100, false, benchmarkGeneratorResource, benchmarkGeneratorSpan),
		},
		{
			name: "servicegraphs_span_multiplier",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.ServiceGraphsName: {}}
				o.serviceGraphsSpanMultiplierKey = "sample.rate"
			},
			request: benchmarkGeneratorServiceGraphRequest(100, true, benchmarkGeneratorResource, benchmarkGeneratorSpan),
		},
		{
			name: "spanmetrics_prod_7dims_target_info_filters",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}}
				benchmarkGeneratorProdOverrides(o)
			},
			request: benchmarkGeneratorSpanmetricsRequest(benchmarkGeneratorProdResource, benchmarkGeneratorProdSpan),
		},
		{
			name: "servicegraphs_prod_7dims_prefix",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
			},
			request: benchmarkGeneratorServiceGraphRequest(100, false, benchmarkGeneratorProdResource, benchmarkGeneratorProdSpan),
		},
		{
			name: "combined_target_info",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				o.spanMetricsEnableTargetInfo = boolPtr(true)
				o.spanMetricsTargetInfoExcludedDimensions = []string{"excluded"}
			},
			request: benchmarkGeneratorServiceGraphRequest(200, false, benchmarkGeneratorResource, benchmarkGeneratorSpan),
		},
		{
			name: "combined_native_histograms",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				o.spanMetricsEnableTargetInfo = boolPtr(true)
				o.spanMetricsTargetInfoExcludedDimensions = []string{"excluded"}
				o.nativeHistograms = histograms.HistogramMethodBoth
			},
			request: benchmarkGeneratorServiceGraphRequest(200, false, benchmarkGeneratorResource, benchmarkGeneratorSpan),
		},
		{
			name: "combined_prod_7dims_target_info_servicegraphs_prefix_filters",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
			},
			request: benchmarkGeneratorServiceGraphRequest(200, false, benchmarkGeneratorProdResource, benchmarkGeneratorProdSpan),
		},
		{
			name: "combined_prod_7dims_native_histograms",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
				o.nativeHistograms = histograms.HistogramMethodBoth
			},
			request: benchmarkGeneratorServiceGraphRequest(200, false, benchmarkGeneratorProdResource, benchmarkGeneratorProdSpan),
		},
	} {
		b.Run(tc.name, func(b *testing.B) {
			inst, _ := benchmarkGeneratorInstance(b, tc.overrides)

			ctx := context.Background()
			spansPerOp := benchmarkGeneratorSpanCount(tc.request)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				inst.pushSpans(ctx, tc.request)
			}
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(spansPerOp), "ns/span")
		})
	}
}

// BenchmarkInstancePushSpansProductionCardinality measures steady-state
// pushSpans cost on instances pre-seeded to production-scale active series
// counts. Seeding creates the series before the timer starts; the timed
// request updates a strict subset of the seeded label sets, so no new series
// are created while timing.
func BenchmarkInstancePushSpansProductionCardinality(b *testing.B) {
	for _, tc := range []struct {
		name       string
		overrides  func(*mockOverrides)
		seed       func(context.Context, *instance)
		wantSeries int64
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
			wantSeries: benchmarkGeneratorProdSpanmetrics100kResources * 20,
			request:    benchmarkGeneratorProdHighCardinalitySpanmetricsRequest(0, benchmarkGeneratorHighCardinalityTimedResources),
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
			wantSeries: benchmarkGeneratorProdServiceGraphs100kEdges * 35,
			request:    benchmarkGeneratorProdHighCardinalityServiceGraphRequest(0, benchmarkGeneratorHighCardinalityTimedEdges),
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
			wantSeries: benchmarkGeneratorProdCombined100kEdges * 75,
			request:    benchmarkGeneratorProdHighCardinalityServiceGraphRequest(0, benchmarkGeneratorHighCardinalityTimedEdges),
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
			wantSeries: benchmarkGeneratorProdCombinedNative100kEdges * 79,
			request:    benchmarkGeneratorProdHighCardinalityServiceGraphRequest(0, benchmarkGeneratorHighCardinalityTimedEdges),
		},
		{
			name: "combined_prod_7dims_500k_series_native_steady",
			overrides: func(o *mockOverrides) {
				o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
				benchmarkGeneratorProdOverrides(o)
				o.nativeHistograms = histograms.HistogramMethodBoth
			},
			seed: func(ctx context.Context, inst *instance) {
				benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombinedNative500kEdges)
			},
			wantSeries: benchmarkGeneratorProdCombinedNative500kEdges * 79,
			request:    benchmarkGeneratorProdHighCardinalityServiceGraphRequest(0, benchmarkGeneratorHighCardinalityTimedEdges),
			requireEnv: "TEMPO_GENERATOR_BENCH_1M",
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
			wantSeries: benchmarkGeneratorProdCombined1MEdges * 75,
			request:    benchmarkGeneratorProdHighCardinalityServiceGraphRequest(0, benchmarkGeneratorHighCardinalityTimedEdges),
			requireEnv: "TEMPO_GENERATOR_BENCH_1M",
		},
	} {
		b.Run(tc.name, func(b *testing.B) {
			if tc.requireEnv != "" && os.Getenv(tc.requireEnv) == "" {
				b.Skipf("set %s to a non-empty value to run this high-cardinality benchmark", tc.requireEnv)
			}

			inst, st := benchmarkGeneratorInstance(b, tc.overrides)

			ctx := context.Background()
			benchmarkGeneratorSeed(ctx, b, inst, st, tc.seed, tc.wantSeries)
			spansPerOp := benchmarkGeneratorSpanCount(tc.request)
			runtime.GC()

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				inst.pushSpans(ctx, tc.request)
			}
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(spansPerOp), "ns/span")
		})
	}
}

// BenchmarkInstancePushSpansConcurrent measures steady-state pushSpans cost
// under concurrent pushers into a single seeded instance, approximating the
// production Kafka consumer (ingest_concurrency, default 16). It catches
// changes that win serially but lose under processor and per-series lock
// contention.
func BenchmarkInstancePushSpansConcurrent(b *testing.B) {
	inst, st := benchmarkGeneratorInstance(b, func(o *mockOverrides) {
		o.processors = map[string]struct{}{processor.SpanMetricsName: {}, processor.ServiceGraphsName: {}}
		benchmarkGeneratorProdOverrides(o)
	})

	ctx := context.Background()
	benchmarkGeneratorSeed(ctx, b, inst, st, func(ctx context.Context, inst *instance) {
		benchmarkGeneratorSeedHighCardinalityServiceGraph(ctx, inst, benchmarkGeneratorProdCombined100kEdges)
	}, benchmarkGeneratorProdCombined100kEdges*75)
	spansPerOp := benchmarkGeneratorHighCardinalityTimedEdges * 2

	// pushSpans mutates the request while preprocessing, so every goroutine
	// needs its own copy, built outside the timed region. All goroutines update
	// the same seeded label sets, exercising contention on the hot series.
	requests := make(chan *tempopb.PushSpansRequest, runtime.GOMAXPROCS(0))
	for i := 0; i < cap(requests); i++ {
		requests <- benchmarkGeneratorProdHighCardinalityServiceGraphRequest(0, benchmarkGeneratorHighCardinalityTimedEdges)
	}
	runtime.GC()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var req *tempopb.PushSpansRequest
		select {
		case req = <-requests:
		default:
			// Only reached with b.SetParallelism, which this benchmark does not use.
			req = benchmarkGeneratorProdHighCardinalityServiceGraphRequest(0, benchmarkGeneratorHighCardinalityTimedEdges)
		}
		for pb.Next() {
			inst.pushSpans(ctx, req)
		}
	})
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(spansPerOp), "ns/span")
}

const (
	// The 100k cases seed enough metric label sets to create roughly 100k
	// active time series before timing steady-state updates. Per-unit active
	// series with the prod overrides (classic histograms: sum + count +
	// 14 buckets + +Inf = 17 series each):
	//
	//	spanmetrics resource:  latency 17 + calls 1 + size 1 + target_info 1          = 20
	//	servicegraphs edge:    client latency 17 + server latency 17 + request 1      = 35
	//	combined edge:         servicegraphs 35 + spanmetrics 20 x 2 resources        = 75
	//	combined native edge:  native adds 1 per histogram: (35 + 2) + (20 + 1) x 2   = 79
	//
	// If bucket counts, subprocessors, or target_info gating change, these
	// constants and the suite names go stale; benchmarkGeneratorSeed asserts
	// the seeded series count after seeding to catch that.
	benchmarkGeneratorProdSpanmetrics100kResources    = 5000  // x 20 series = 100_000
	benchmarkGeneratorProdServiceGraphs100kEdges      = 2858  // x 35 series = 100_030
	benchmarkGeneratorProdCombined100kEdges           = 1334  // x 75 series = 100_050
	benchmarkGeneratorProdCombinedNative100kEdges     = 1266  // x 79 series = 100_014
	benchmarkGeneratorProdCombinedNative500kEdges     = 6329  // x 79 series = 499_991
	benchmarkGeneratorProdCombined1MEdges             = 13334 // x 75 series = 1_000_050
	benchmarkGeneratorHighCardinalityTimedResources   = 400
	benchmarkGeneratorHighCardinalityTimedEdges       = 200
	benchmarkGeneratorHighCardinalitySeedResourceSize = 500
	benchmarkGeneratorHighCardinalitySeedEdgeSize     = 250

	benchmarkGeneratorTenant = "bench-tenant"
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

// Policy keys use the production config syntax: `span.` and `resource.` scope
// prefixes are parsed as TraceQL and stripped before matching raw attribute
// names. The fixtures therefore carry `url.path` (span) and
// `span.metrics.skip` (resource) so the policies resolve against them and the
// value-compare path is exercised.
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

func benchmarkGeneratorInstance(b *testing.B, tune func(*mockOverrides)) (*instance, *benchmarkGeneratorStorage) {
	b.Helper()

	cfg := &Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})

	// A year of ingestion slack keeps requests reused across iterations from
	// being dropped as stale, and a one-hour collection interval keeps registry
	// collection jobs from firing inside the timed window at long -benchtime
	// values (collection cost is covered by BenchmarkCollect).
	o := &mockOverrides{
		ingestionSlack:     365 * 24 * time.Hour,
		collectionInterval: time.Hour,
	}
	tune(o)

	st := &benchmarkGeneratorStorage{}
	inst, err := newInstance(cfg, benchmarkGeneratorTenant, o, st, log.NewNopLogger())
	if err != nil {
		b.Fatal(err)
	}
	// Cleanup runs after each trial's timer has stopped, keeping shutdown out
	// of the measured window.
	b.Cleanup(inst.shutdown)
	return inst, st
}

// benchmarkGeneratorSeed seeds the instance and asserts the steady-state
// premise: no spans were discarded and the expected number of active series
// was created.
func benchmarkGeneratorSeed(ctx context.Context, b *testing.B, inst *instance, st *benchmarkGeneratorStorage, seed func(context.Context, *instance), wantSeries int64) {
	b.Helper()

	discarded := metricSpansDiscarded.WithLabelValues(benchmarkGeneratorTenant, reasonOutsideTimeRangeSlack, "all")
	discardedBefore := testutil.ToFloat64(discarded)

	seed(ctx, inst)

	if d := testutil.ToFloat64(discarded) - discardedBefore; d > 0 {
		b.Fatalf("seeding discarded %.0f spans as outside the ingestion slack, series were not seeded as intended", d)
	}
	seeded := benchmarkGeneratorActiveSeries(ctx, inst, st)
	if seeded != wantSeries {
		b.Fatalf("seeded %d active series, want %d; revisit the seed constants and their per-unit series math", seeded, wantSeries)
	}
}

// benchmarkGeneratorActiveSeries counts active series by collecting the
// registry twice: the first collection lets brand-new series append their
// extra zero samples, the second appends exactly one sample per active series.
func benchmarkGeneratorActiveSeries(ctx context.Context, inst *instance, st *benchmarkGeneratorStorage) int64 {
	inst.registry.CollectMetrics(ctx)
	st.samples.Store(0)
	inst.registry.CollectMetrics(ctx)
	return st.samples.Load()
}

func benchmarkGeneratorSpanCount(req *tempopb.PushSpansRequest) int {
	spans := 0
	for _, batch := range req.Batches {
		for _, ss := range batch.ScopeSpans {
			spans += len(ss.Spans)
		}
	}
	return spans
}

type (
	benchmarkGeneratorResourceFunc func(service string, idx int) *trace_v1.ResourceSpans
	benchmarkGeneratorSpanFunc     func(traceID, spanID, parentSpanID []byte, kind trace_v1.Span_SpanKind, start uint64) *trace_v1.Span
)

// benchmarkGeneratorSpanmetricsRequest builds 4 resource batches of 100 server
// spans each (400 spans per op), shaped by the given fixture constructors.
func benchmarkGeneratorSpanmetricsRequest(resource benchmarkGeneratorResourceFunc, span benchmarkGeneratorSpanFunc) *tempopb.PushSpansRequest {
	const (
		resources        = 4
		spansPerResource = 100
	)

	req := &tempopb.PushSpansRequest{
		Batches: make([]*trace_v1.ResourceSpans, 0, resources),
	}
	now := uint64(time.Now().UnixNano())
	for r := 0; r < resources; r++ {
		rs := resource("svc-"+strconv.Itoa(r), r)
		for s := 0; s < spansPerResource; s++ {
			rs.ScopeSpans[0].Spans = append(rs.ScopeSpans[0].Spans, span(
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

// benchmarkGeneratorServiceGraphRequest builds edges client/server span pairs
// across one client and one server resource, shaped by the given fixture
// constructors. Every edge completes within the request: the client batch
// precedes the server batch and the server spans reference the client span
// IDs.
func benchmarkGeneratorServiceGraphRequest(edges int, spanMultiplier bool, resource benchmarkGeneratorResourceFunc, span benchmarkGeneratorSpanFunc) *tempopb.PushSpansRequest {
	now := uint64(time.Now().UnixNano())
	client := resource("client", 0)
	server := resource("server", 1)
	for i := 0; i < edges; i++ {
		traceID := benchmarkGeneratorTraceID(i)
		clientSpanID := benchmarkGeneratorSpanID(i*2 + 1)

		clientSpan := span(traceID, clientSpanID, nil, trace_v1.Span_SPAN_KIND_CLIENT, now+uint64(i))
		serverSpan := span(traceID, benchmarkGeneratorSpanID(i*2+2), clientSpanID, trace_v1.Span_SPAN_KIND_SERVER, now+uint64(i))
		if spanMultiplier {
			clientSpan.Attributes = append(clientSpan.Attributes, benchmarkGeneratorDoubleAttr("sample.rate", 0.5))
			serverSpan.Attributes = append(serverSpan.Attributes, benchmarkGeneratorDoubleAttr("sample.rate", 0.5))
		}
		client.ScopeSpans[0].Spans = append(client.ScopeSpans[0].Spans, clientSpan)
		server.ScopeSpans[0].Spans = append(server.ScopeSpans[0].Spans, serverSpan)
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
	rs := &trace_v1.ResourceSpans{
		Resource: &resource_v1.Resource{
			Attributes: benchmarkGeneratorCommonResourceAttrs(service, idx),
		},
		ScopeSpans: []*trace_v1.ScopeSpans{{}},
	}
	rs.Resource.Attributes = append(rs.Resource.Attributes,
		test.MakeAttribute("excluded", "drop-me"),
	)
	return rs
}

func benchmarkGeneratorProdResource(service string, idx int) *trace_v1.ResourceSpans {
	rs := &trace_v1.ResourceSpans{
		Resource: &resource_v1.Resource{
			Attributes: benchmarkGeneratorCommonResourceAttrs(service, idx),
		},
		ScopeSpans: []*trace_v1.ScopeSpans{{}},
	}
	rs.Resource.Attributes = append(rs.Resource.Attributes,
		test.MakeAttribute("service.version", "1."+strconv.Itoa(idx%10)+".0"),
		test.MakeAttribute("cloud.availability_zone", "zone-"+strconv.Itoa(idx%3)),
		test.MakeAttribute("cloud.region", "region-"+strconv.Itoa(idx%4)),
		test.MakeAttribute("deployment.environment", "prod"),
		test.MakeAttribute("k8s.pod.start_time", "2026-05-07T00:00:00Z"),
		test.MakeAttribute("os.description", "linux"),
		test.MakeAttribute("os.type", "linux"),
		test.MakeAttribute("process.command_args", "tempo"),
		test.MakeAttribute("process.executable.path", "/tempo"),
		test.MakeAttribute("process.pid", strconv.Itoa(1000+idx)),
		test.MakeAttribute("process.runtime.description", "go"),
		test.MakeAttribute("process.runtime.name", "go"),
		test.MakeAttribute("process.runtime.version", "go1.24"),
		// Matched by the resource.span.metrics.skip exclude policy after its
		// resource. prefix is stripped; false keeps the spans included.
		benchmarkGeneratorBoolAttr("span.metrics.skip", false),
	)
	return rs
}

func benchmarkGeneratorCommonResourceAttrs(service string, idx int) []*common_v1.KeyValue {
	return []*common_v1.KeyValue{
		test.MakeAttribute("service.name", service),
		test.MakeAttribute("service.namespace", "ns-"+strconv.Itoa(idx%2)),
		test.MakeAttribute("service.instance.id", "instance-"+strconv.Itoa(idx)),
		test.MakeAttribute("k8s.cluster.name", "cluster-a"),
		test.MakeAttribute("k8s.namespace.name", "namespace-"+strconv.Itoa(idx%4)),
		test.MakeAttribute("k8s.pod.name", "pod-"+strconv.Itoa(idx)),
		test.MakeAttribute("k8s.node.name", "node-"+strconv.Itoa(idx%8)),
		test.MakeAttribute("telemetry.sdk.language", "go"),
		test.MakeAttribute("telemetry.sdk.version", "1.0.0"),
	}
}

func benchmarkGeneratorSpan(traceID, spanID, parentSpanID []byte, kind trace_v1.Span_SpanKind, start uint64) *trace_v1.Span {
	return &trace_v1.Span{
		TraceId:           traceID,
		SpanId:            spanID,
		ParentSpanId:      parentSpanID,
		Name:              "GET /api/:id",
		Kind:              kind,
		StartTimeUnixNano: start,
		EndTimeUnixNano:   start + uint64(benchmarkGeneratorSpanDuration(spanID)),
		Status:            &trace_v1.Status{Code: trace_v1.Status_STATUS_CODE_OK},
		Attributes: []*common_v1.KeyValue{
			test.MakeAttribute("http.method", "GET"),
			test.MakeAttribute("http.route", "/api/:id"),
			test.MakeAttribute("http.status_code", "200"),
			test.MakeAttribute("span.attr.1", "one"),
			test.MakeAttribute("span.attr.2", "two"),
			test.MakeAttribute("peer.service", "server"),
		},
	}
}

func benchmarkGeneratorProdSpan(traceID, spanID, parentSpanID []byte, kind trace_v1.Span_SpanKind, start uint64) *trace_v1.Span {
	span := benchmarkGeneratorSpan(traceID, spanID, parentSpanID, kind, start)
	span.Attributes = append(span.Attributes,
		// Matched by the span.url.path exclude policies after their span.
		// prefix is stripped; /api/:id keeps the spans included.
		test.MakeAttribute("url.path", "/api/:id"),
		test.MakeAttribute("server.address", "server"),
		test.MakeAttribute("net.peer.name", "server"),
	)
	return span
}

// benchmarkGeneratorSpanDurations spreads span durations across roughly three
// decades of the prod histogram buckets so latency observations exercise
// bucket search and native (sparse) bucket mapping instead of hitting a single
// hot bucket.
var benchmarkGeneratorSpanDurations = [...]time.Duration{
	2 * time.Millisecond,
	8 * time.Millisecond,
	30 * time.Millisecond,
	120 * time.Millisecond,
	500 * time.Millisecond,
	2 * time.Second,
}

func benchmarkGeneratorSpanDuration(spanID []byte) time.Duration {
	idx := binary.BigEndian.Uint64(spanID)
	return benchmarkGeneratorSpanDurations[idx%uint64(len(benchmarkGeneratorSpanDurations))]
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

// benchmarkGeneratorStorage is a noop storage whose appenders count appended
// samples, letting benchmarks assert seeded active series counts (one sample
// per active series per collection).
type benchmarkGeneratorStorage struct {
	samples atomic.Int64
}

var _ storage.Storage = (*benchmarkGeneratorStorage)(nil)

func (s *benchmarkGeneratorStorage) Appender(context.Context) prometheus_storage.Appender {
	return &benchmarkGeneratorCountingAppender{samples: &s.samples}
}

func (s *benchmarkGeneratorStorage) Close() error { return nil }

type benchmarkGeneratorCountingAppender struct {
	noopAppender
	samples *atomic.Int64
}

func (a *benchmarkGeneratorCountingAppender) Append(prometheus_storage.SeriesRef, labels.Labels, int64, float64) (prometheus_storage.SeriesRef, error) {
	a.samples.Add(1)
	return 0, nil
}

func (a *benchmarkGeneratorCountingAppender) AppendHistogram(prometheus_storage.SeriesRef, labels.Labels, int64, *histogram.Histogram, *histogram.FloatHistogram) (prometheus_storage.SeriesRef, error) {
	a.samples.Add(1)
	return 0, nil
}
