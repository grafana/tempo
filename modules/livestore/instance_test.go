package livestore

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	util_log "github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/pkg/util/test"
)

func instanceWithPushLimits(t *testing.T, maxBytesPerTrace int, maxLiveTraces int) (*instance, *LiveStore) {
	instance, ls := defaultInstance(t)
	limits, err := overrides.NewOverrides(overrides.Config{
		Defaults: overrides.Overrides{
			Global: overrides.GlobalOverrides{
				MaxBytesPerTrace: maxBytesPerTrace,
			},
			Ingestion: overrides.IngestionOverrides{
				MaxLocalTracesPerUser: maxLiveTraces,
			},
		},
	}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)
	instance.overrides = limits

	return instance, ls
}

func pushTrace(ctx context.Context, t *testing.T, instance *instance, tr *tempopb.Trace, id []byte) {
	b, err := tr.Marshal()
	require.NoError(t, err)
	req := &tempopb.PushBytesRequest{
		Traces: []tempopb.PreallocBytes{{Slice: b}},
		Ids:    [][]byte{id},
	}
	instance.pushBytes(ctx, time.Now(), req)
}

// TestInstanceLimits verifies MaxBytesPerTrace and MaxLocalTracesPerUser enforcement in livestore.
func TestInstanceLimits(t *testing.T) {
	const batches = 20
	// Configure limits: allow up to ~1.5x small trace, and max 4 live traces
	maxTraces := 4

	batch1 := test.MakeTrace(batches, test.ValidTraceID(nil))
	batch2 := test.MakeTrace(batches, test.ValidTraceID(nil))
	maxBytes := batch1.Size() + batch2.Size()/2 // set limit between 1 and 2 batches so pushing both batches to a single trace exceeds limit

	// bytes - succeeds: push two different traces under size limit
	t.Run("bytes - succeeds", func(t *testing.T) {
		instance, ls := instanceWithPushLimits(t, maxBytes, maxTraces)
		// two different traces with different ids
		id1 := test.ValidTraceID(nil)
		id2 := test.ValidTraceID(nil)
		pushTrace(t.Context(), t, instance, batch1, id1)
		pushTrace(t.Context(), t, instance, batch2, id2)
		require.Equal(t, uint64(2), instance.liveTraces.Len())

		err := services.StopAndAwaitTerminated(t.Context(), ls)
		require.NoError(t, err)
	})

	// bytes - one fails: second push of the same trace exceeds MaxBytesPerTrace
	t.Run("bytes - one fails", func(t *testing.T) {
		instance, ls := instanceWithPushLimits(t, maxBytes, maxTraces)

		id := test.ValidTraceID(nil)
		// First push fits
		pushTrace(t.Context(), t, instance, batch1, id)
		// Second push with same id will exceed combined size (> maxBytes)
		pushTrace(t.Context(), t, instance, batch2, id)
		// Only one live trace stored, and accumulated size should be <= maxBytes
		require.Equal(t, uint64(1), instance.liveTraces.Len())
		require.LessOrEqual(t, instance.liveTraces.Size(), uint64(maxBytes))

		err := services.StopAndAwaitTerminated(t.Context(), ls)
		require.NoError(t, err)
	})

	// bytes - second push fails even after cutIdleTraces
	t.Run("bytes - second push fails even after cutIdleTraces", func(t *testing.T) {
		instance, ls := instanceWithPushLimits(t, maxBytes, maxTraces)

		id := test.ValidTraceID(nil)
		// First push fits
		pushTrace(t.Context(), t, instance, batch1, id)

		// cut idle traces but we retain the too large trace in traceSizes
		err := instance.cutIdleTraces(t.Context(), true)
		require.NoError(t, err)

		// Second push with same id will fail b/c we are still tracking in traceSizes
		pushTrace(t.Context(), t, instance, batch2, id)
		require.Equal(t, uint64(0), instance.liveTraces.Len())
		require.Equal(t, instance.liveTraces.Size(), uint64(0))

		err = services.StopAndAwaitTerminated(t.Context(), ls)
		require.NoError(t, err)
	})

	// bytes - second push succeeds after cutIdleTraces and 2x cutBlocks
	t.Run("bytes - second push succeeds after cutting head block 2x", func(t *testing.T) {
		instance, ls := instanceWithPushLimits(t, maxBytes, maxTraces)

		id := test.ValidTraceID(nil)
		// First push fits
		pushTrace(t.Context(), t, instance, batch1, id)

		// cut idle traces but we retain the too large trace in traceSizes
		err := instance.cutIdleTraces(t.Context(), true)
		require.NoError(t, err)
		blockID, err := instance.cutBlocks(t.Context(), true) // this won't clear the trace b/c the trace must not be seen for 2 head block cuts to be fully removed from live traces
		require.NoError(t, err)
		err = instance.completeBlock(t.Context(), blockID)
		require.NoError(t, err)

		// push a second trace so cutIdle/cutBlocks goes through
		secondID := test.ValidTraceID(nil)
		pushTrace(t.Context(), t, instance, batch1, secondID)

		err = instance.cutIdleTraces(t.Context(), true)
		require.NoError(t, err)
		blockID, err = instance.cutBlocks(t.Context(), true) // this will clear the trace b/c the trace has not been seen for 2 head block cuts
		require.NoError(t, err)
		err = instance.completeBlock(t.Context(), blockID)
		require.NoError(t, err)

		// Second push with same id will succeed b/c we have gone through one block flush cycles w/o seeing it
		pushTrace(t.Context(), t, instance, batch1, id)
		require.Equal(t, uint64(1), instance.liveTraces.Len())
		require.LessOrEqual(t, instance.liveTraces.Size(), uint64(maxBytes))

		err = services.StopAndAwaitTerminated(t.Context(), ls)
		require.NoError(t, err)
	})

	// max traces - too many: only first 4 unique traces are accepted
	t.Run("max traces - too many", func(t *testing.T) {
		instance, ls := instanceWithPushLimits(t, maxBytes, maxTraces)

		for range 10 {
			id := test.ValidTraceID(nil)
			ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(1*time.Second)) // Time out after 1s, push should be immediate
			t.Cleanup(cancel)
			pushTrace(ctx, t, instance, test.MakeTrace(1, id), id)
		}
		require.Equal(t, uint64(4), instance.liveTraces.Len())

		err := services.StopAndAwaitTerminated(t.Context(), ls)
		require.NoError(t, err)
	})
}

// TestTraceTooLargeLogContainsInsight verifies that the "trace too large" log line contains insight=true
func TestTraceTooLargeLogContainsInsight(t *testing.T) {
	id := test.ValidTraceID(nil)
	trace := test.MakeTrace(1, id)
	instance, ls := instanceWithPushLimits(t, trace.Size(), 4)

	// Replace maxTraceLogger to capture log output
	var logBuf bytes.Buffer
	instance.maxTraceLogger = util_log.NewRateLimitedLogger(maxTraceLogLinesPerSecond, log.NewLogfmtLogger(&logBuf))

	pushTrace(t.Context(), t, instance, trace, id)
	pushTrace(t.Context(), t, instance, trace, id) // second push exceeds limit

	assert.Contains(t, logBuf.String(), "insight=true")

	require.NoError(t, services.StopAndAwaitTerminated(t.Context(), ls))
}

func TestInstanceNoLimits(t *testing.T) {
	instance, ls := instanceWithPushLimits(t, 0, 0) // no limits by default

	for range 100 {
		id := test.ValidTraceID(nil)
		pushTrace(t.Context(), t, instance, test.MakeTrace(1, id), id)
	}

	assert.Equal(t, uint64(100), instance.liveTraces.Len())
	assert.GreaterOrEqual(t, instance.liveTraces.Size(), uint64(1000))

	err := services.StopAndAwaitTerminated(t.Context(), ls)
	require.NoError(t, err)
}

func TestInstanceBackpressure(t *testing.T) {
	instance, ls := defaultInstance(t)

	id1 := test.ValidTraceID(nil)
	pushTrace(t.Context(), t, instance, test.MakeTrace(1, id1), id1)

	instance.Cfg.MaxLiveTracesBytes = instance.liveTraces.Size() // Set max size to current live-traces size

	id2 := test.ValidTraceID(nil)

	// Use a channel to coordinate the blocking push operation
	pushComplete := make(chan struct{})
	go func() {
		defer close(pushComplete)
		// Second write will block waiting for the live traces to have room
		pushTrace(t.Context(), t, instance, test.MakeTrace(1, id2), id2)
	}()

	// Give goroutine time to start and block
	time.Sleep(10 * time.Millisecond)

	// First trace is found
	res, err := instance.FindByTraceID(t.Context(), id1, true)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Trace)
	require.Greater(t, res.Trace.Size(), 0)

	// Second is not (should be blocked)
	res, err = instance.FindByTraceID(t.Context(), id2, true)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Nil(t, res.Trace)

	// Free up space for the blocked push
	require.NoError(t, instance.cutIdleTraces(t.Context(), true))

	// Wait for push to complete with timeout
	select {
	case <-pushComplete:
		// Push completed successfully
	case <-time.After(1 * time.Second):
		t.Fatal("push operation did not complete within timeout")
	}

	// After cut, second trace is pushed to instance and can be found
	res, err = instance.FindByTraceID(t.Context(), id2, true)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Trace)
	require.Greater(t, res.Trace.Size(), 0)

	require.NoError(t, services.StopAndAwaitTerminated(t.Context(), ls))
}

// Realistic service definitions modeled on production Grafana Cloud traces.
// Each service carries its own resource attributes, instrumentation scopes,
// and typical span attribute shapes.
var benchServices = []struct {
	name       string
	namespace  string
	cluster    string
	sdkLang    string
	scopes     []string
	spanAttrs  []string // common attribute keys produced by this service
	hasEvents  bool
	eventNames []string
}{
	{
		name: "hggateway", namespace: "hosted-grafana", cluster: "prod-us-central-0",
		sdkLang: "go",
		scopes:  []string{"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho", "github.com/grafana/hosted-grafana/pkg/gateway"},
		spanAttrs: []string{
			"http.method", "http.status_code", "http.url", "http.host",
			"http.flavor", "http.client_ip", "gateway.route_name",
			"http.request.body.size", "http.response.body.size",
			"net.peer.ip", "net.peer.port",
		},
	},
	{
		name: "grafana", namespace: "hosted-grafana", cluster: "prod-us-central-0",
		sdkLang: "go",
		scopes: []string{
			"github.com/grafana/grafana/pkg/infra/tracing",
			"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux",
			"github.com/grafana/grafana/pkg/services/accesscontrol",
		},
		spanAttrs: []string{
			"http.method", "http.status_code", "http.url",
			"organization", "auth.mode", "auth.namespace",
		},
		hasEvents: true, eventNames: []string{"log"},
	},
	{
		name: "gme-querier", namespace: "metrics", cluster: "prod-us-central-0",
		sdkLang: "go",
		scopes:  []string{"pkg/querier", "pkg/streamingpromql", "dskit/tracing"},
		spanAttrs: []string{
			"rpc.method", "rpc.system.name", "rpc.response.status_code",
			"server.address", "server.port",
			"tenant_ids", "ingester_address", "ingester_zone",
			"query", "start", "end", "step_ms", "bytes",
		},
		hasEvents: true, eventNames: []string{"log", "using cache", "PostingsForMatchers returned"},
	},
	{
		name: "gme-query-frontend", namespace: "metrics", cluster: "prod-us-central-0",
		sdkLang: "go",
		scopes:  []string{"pkg/querymiddleware", "pkg/frontend/v2"},
		spanAttrs: []string{
			"rpc.method", "rpc.system.name", "rpc.response.status_code",
			"server.address", "server.port", "tenant_ids",
		},
	},
	{
		name: "cortex-gateway", namespace: "metrics", cluster: "prod-us-central-0",
		sdkLang: "go",
		scopes:  []string{"pkg/authentication/grafanacloud", "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"},
		spanAttrs: []string{
			"http.method", "http.status_code", "http.url",
			"AccessPolicyID", "TokenID",
		},
	},
}

// httpMethods and statusCodes provide realistic value distributions.
var (
	httpMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	statusCodes = []int64{200, 200, 200, 200, 200, 200, 200, 201, 204, 301, 400, 404, 500} // skewed toward 200
	rpcMethods  = []string{"Query", "QueryStream", "MetricsForLabelMatchers", "LabelValues", "Series"}
)

func makeRealisticTrace(traceID []byte, numServices, spansPerService int) *tempopb.Trace { // nolint:unparam
	now := time.Now()
	trace := &tempopb.Trace{}

	for si := range numServices {
		svc := benchServices[si%len(benchServices)]

		resource := &v1_resource.Resource{
			Attributes: []*v1_common.KeyValue{
				{Key: "service.name", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: svc.name}}},
				{Key: "k8s.namespace.name", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: svc.namespace}}},
				{Key: "k8s.cluster.name", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: svc.cluster}}},
				{Key: "telemetry.sdk.language", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: svc.sdkLang}}},
				{Key: "telemetry.sdk.name", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "opentelemetry"}}},
				{Key: "telemetry.sdk.version", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "1.28.0"}}},
				{Key: "k8s.pod.name", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: fmt.Sprintf("%s-7f8b9c6d4-x%04d", svc.name, si)}}},
			},
		}

		scope := svc.scopes[rand.Intn(len(svc.scopes))] // nolint:gosec
		spans := make([]*v1_trace.Span, 0, spansPerService)
		for j := range spansPerService {
			startNano := uint64(now.Add(time.Duration(j) * time.Millisecond).UnixNano())
			endNano := startNano + uint64((1+rand.Intn(50))*int(time.Millisecond)) // nolint:gosec

			spanID := make([]byte, 8)
			_, _ = crand.Read(spanID)

			attrs := makeRealisticSpanAttrs(svc.spanAttrs)
			var events []*v1_trace.Span_Event
			if svc.hasEvents && rand.Intn(3) == 0 { // nolint:gosec
				eName := svc.eventNames[rand.Intn(len(svc.eventNames))] // nolint:gosec
				events = append(events, &v1_trace.Span_Event{
					TimeUnixNano: startNano + uint64(rand.Intn(int(time.Millisecond))), // nolint:gosec
					Name:         eName,
					Attributes: []*v1_common.KeyValue{
						{Key: "message", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "benchmark event data"}}},
					},
				})
			}

			spans = append(spans, &v1_trace.Span{
				TraceId:           traceID,
				SpanId:            spanID,
				Name:              fmt.Sprintf("%s.op%d", svc.name, j%5),
				Kind:              v1_trace.Span_SpanKind(1 + rand.Intn(4)), // nolint:gosec
				StartTimeUnixNano: startNano,
				EndTimeUnixNano:   endNano,
				Attributes:        attrs,
				Events:            events,
				Status:            &v1_trace.Status{Code: v1_trace.Status_STATUS_CODE_OK},
			})
		}

		trace.ResourceSpans = append(trace.ResourceSpans, &v1_trace.ResourceSpans{
			Resource: resource,
			ScopeSpans: []*v1_trace.ScopeSpans{
				{
					Scope: &v1_common.InstrumentationScope{Name: scope},
					Spans: spans,
				},
			},
		})
	}

	return trace
}

func makeRealisticSpanAttrs(keys []string) []*v1_common.KeyValue {
	attrs := make([]*v1_common.KeyValue, 0, len(keys))
	for _, k := range keys {
		var v *v1_common.AnyValue
		switch k {
		case "http.method", "http.request.method":
			v = &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: httpMethods[rand.Intn(len(httpMethods))]}} // nolint:gosec
		case "http.status_code", "http.response.status_code":
			v = &v1_common.AnyValue{Value: &v1_common.AnyValue_IntValue{IntValue: statusCodes[rand.Intn(len(statusCodes))]}} // nolint:gosec
		case "rpc.method":
			v = &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: rpcMethods[rand.Intn(len(rpcMethods))]}} // nolint:gosec
		case "rpc.system.name":
			v = &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "grpc"}}
		case "rpc.response.status_code":
			v = &v1_common.AnyValue{Value: &v1_common.AnyValue_IntValue{IntValue: 0}} // OK
		case "server.port":
			v = &v1_common.AnyValue{Value: &v1_common.AnyValue_IntValue{IntValue: int64(8080 + rand.Intn(5))}} // nolint:gosec
		case "server.address":
			v = &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: fmt.Sprintf("10.0.%d.%d", rand.Intn(256), rand.Intn(256))}} // nolint:gosec
		case "ingester_zone":
			zones := []string{"zone-a", "zone-b", "zone-c"}
			v = &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: zones[rand.Intn(len(zones))]}} // nolint:gosec
		default:
			v = &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: fmt.Sprintf("val-%d", rand.Intn(100))}} // nolint:gosec
		}
		attrs = append(attrs, &v1_common.KeyValue{Key: k, Value: v})
	}
	return attrs
}

// populateWALBlock pushes numTraces traces into the instance across numFlushes
// flush cycles (simulating production where cutIdleTraces runs ~12 times before
// cutBlocks cuts the head block). Each trace spans numServices services with
// spansPerService spans each.
func populateWALBlock(b *testing.B, inst *instance, numTraces, numServices, spansPerService, numFlushes int) uuid.UUID {
	b.Helper()
	ctx := context.Background()

	tracesPerFlush := numTraces / numFlushes
	remainder := numTraces % numFlushes

	for f := range numFlushes {
		count := tracesPerFlush
		if f < remainder {
			count++
		}
		for range count {
			id := test.ValidTraceID(nil)
			tr := makeRealisticTrace(id, numServices, spansPerService)
			traceBytes, err := tr.Marshal()
			require.NoError(b, err)

			req := &tempopb.PushBytesRequest{
				Traces: []tempopb.PreallocBytes{{Slice: traceBytes}},
				Ids:    [][]byte{id},
			}
			inst.pushBytes(ctx, time.Now(), req)
		}

		// Each flush cycle creates a new page in the WAL block
		err := inst.cutIdleTraces(ctx, true)
		require.NoError(b, err)
	}

	// Cut head block → WAL block
	blockID, err := inst.cutBlocks(ctx, true)
	require.NoError(b, err)
	require.NotEqual(b, uuid.Nil, blockID)

	return blockID
}

func BenchmarkCompleteBlock(b *testing.B) {
	// Parameters calibrated against production tempo_live_store_completion_size_bytes:
	//   p50 ~800KB, avg ~3.4MB, p90 ~6.6MB, p99 ~65MB
	// numFlushes=12 matches production (InstanceFlushPeriod=5s, MaxBlockDuration=1m)
	const numFlushes = 12

	benchmarks := []struct {
		numTraces       int
		numServices     int
		spansPerService int
	}{
		{300, 3, 10},    // p50: ~800KB block
		{1000, 4, 10},   // avg: ~3.4MB block
		{1500, 5, 100},  // p90+: ~40MB block
	}

	for _, bc := range benchmarks {
		totalSpans := bc.numTraces * bc.numServices * bc.spansPerService
		b.Run(fmt.Sprintf("traces=%d/svcs=%d/spans=%d/total=%d", bc.numTraces, bc.numServices, bc.spansPerService, totalSpans), func(b *testing.B) {
			inst, ls := defaultInstance(b)
			b.Cleanup(func() {
				_ = services.StopAndAwaitTerminated(context.Background(), ls)
			})

			ctx := context.Background()
			var lastBlockID uuid.UUID

			b.ResetTimer()
			for range b.N {
				b.StopTimer()
				blockID := populateWALBlock(b, inst, bc.numTraces, bc.numServices, bc.spansPerService, numFlushes)
				b.StartTimer()

				err := inst.completeBlock(ctx, blockID)
				require.NoError(b, err)
				lastBlockID = blockID
			}
			b.StopTimer()

			inst.blocksMtx.RLock()
			if cb, ok := inst.completeBlocks[lastBlockID]; ok {
				b.ReportMetric(float64(cb.BlockMeta().Size_), "block-bytes")
				b.ReportMetric(float64(cb.BlockMeta().TotalObjects), "traces")
			}
			inst.blocksMtx.RUnlock()
		})
	}
}
