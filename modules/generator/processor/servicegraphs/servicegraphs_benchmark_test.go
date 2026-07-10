package servicegraphs

import (
	"context"
	"encoding/binary"
	"strconv"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resource_v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/prometheus/client_golang/prometheus"
)

// BenchmarkServiceGraphsPushSpansSteadyState measures ingestion after edge
// pools and registry series have been warmed outside the timed loop.
func BenchmarkServiceGraphsPushSpansSteadyState(b *testing.B) {
	for _, tc := range []struct {
		name           string
		tune           func(*Config)
		spanMultiplier bool
		histogramMode  registry.HistogramMode
		edgeKind       benchmarkServiceGraphEdgeKind
	}{
		{name: "default"},
		{
			name: "dimensions",
			tune: func(cfg *Config) {
				cfg.Dimensions = []string{"beast", "god"}
			},
		},
		{
			name: "client_server_prefix",
			tune: func(cfg *Config) {
				cfg.Dimensions = []string{"beast", "god"}
				cfg.EnableClientServerPrefix = true
			},
		},
		{
			name: "span_multiplier",
			tune: func(cfg *Config) {
				cfg.SpanMultiplierKey = "sample.rate"
			},
			spanMultiplier: true,
		},
		{name: "native/database", histogramMode: registry.HistogramModeNative, edgeKind: benchmarkServiceGraphDatabaseEdge},
		{name: "both/database", histogramMode: registry.HistogramModeBoth, edgeKind: benchmarkServiceGraphDatabaseEdge},
		{name: "native/messaging", histogramMode: registry.HistogramModeNative, edgeKind: benchmarkServiceGraphMessagingEdge},
		{name: "both/messaging", histogramMode: registry.HistogramModeBoth, edgeKind: benchmarkServiceGraphMessagingEdge},
		{name: "native/virtual", histogramMode: registry.HistogramModeNative, edgeKind: benchmarkServiceGraphVirtualEdge},
		{name: "both/virtual", histogramMode: registry.HistogramModeBoth, edgeKind: benchmarkServiceGraphVirtualEdge},
	} {
		b.Run(tc.name, func(b *testing.B) {
			cfg := Config{}
			cfg.RegisterFlagsAndApplyDefaults("", nil)
			cfg.HistogramBuckets = []float64{0.04, 0.1, 1}
			cfg.HistogramOverride = tc.histogramMode
			cfg.Workers = 0
			if tc.edgeKind == benchmarkServiceGraphMessagingEdge {
				cfg.EnableMessagingSystemLatencyHistogram = true
			}
			if tc.edgeKind == benchmarkServiceGraphVirtualEdge {
				cfg.Wait = -time.Nanosecond
			}
			if tc.tune != nil {
				tc.tune(&cfg)
			}

			registryCfg := &registry.Config{}
			registryCfg.RegisterFlagsAndApplyDefaults("", nil)
			managedRegistry := registry.New(
				registryCfg,
				serviceGraphRegistryOverrides{
					nativeHistogramBucketFactor:     1.1,
					nativeHistogramMaxBucketNumber:  100,
					nativeHistogramMinResetDuration: 15 * time.Minute,
				},
				"bench",
				&serviceGraphCapturingAppender{},
				log.NewNopLogger(),
				serviceGraphNoopLimiter{},
			)
			defer managedRegistry.Close()

			p, err := New(
				cfg,
				"bench",
				managedRegistry,
				log.NewNopLogger(),
				prometheus.NewCounter(prometheus.CounterOpts{}),
				prometheus.NewCounter(prometheus.CounterOpts{}),
			)
			if err != nil {
				b.Fatal(err)
			}
			defer p.Shutdown(context.Background())

			req := benchmarkServiceGraphRequestForKind(100, tc.spanMultiplier, tc.edgeKind)
			p.PushSpans(context.Background(), req)
			if tc.edgeKind == benchmarkServiceGraphVirtualEdge {
				p.(*Processor).store.Expire()
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.PushSpans(context.Background(), req)
				if tc.edgeKind == benchmarkServiceGraphVirtualEdge {
					p.(*Processor).store.Expire()
				}
			}
		})
	}
}

type benchmarkServiceGraphEdgeKind string

const (
	benchmarkServiceGraphDatabaseEdge  benchmarkServiceGraphEdgeKind = "database"
	benchmarkServiceGraphMessagingEdge benchmarkServiceGraphEdgeKind = "messaging"
	benchmarkServiceGraphVirtualEdge   benchmarkServiceGraphEdgeKind = "virtual"
)

func benchmarkServiceGraphRequestForKind(edges int, spanMultiplier bool, kind benchmarkServiceGraphEdgeKind) *tempopb.PushSpansRequest {
	req := benchmarkServiceGraphRequest(edges, spanMultiplier)

	switch kind {
	case benchmarkServiceGraphDatabaseEdge:
		for _, span := range req.Batches[0].ScopeSpans[0].Spans {
			span.Attributes = append(span.Attributes, tempopb.MakeKeyValueStringPtr("db.namespace", "orders"))
		}
		req.Batches = req.Batches[:1]
	case benchmarkServiceGraphMessagingEdge:
		for _, span := range req.Batches[0].ScopeSpans[0].Spans {
			span.Kind = trace_v1.Span_SPAN_KIND_PRODUCER
		}
		for _, span := range req.Batches[1].ScopeSpans[0].Spans {
			span.Kind = trace_v1.Span_SPAN_KIND_CONSUMER
			span.StartTimeUnixNano += 50_000_000
			span.EndTimeUnixNano += 50_000_000
		}
	case benchmarkServiceGraphVirtualEdge:
		req.Batches = req.Batches[:1]
	}

	return req
}

func benchmarkServiceGraphRequest(edges int, spanMultiplier bool) *tempopb.PushSpansRequest {
	client := &trace_v1.ResourceSpans{
		Resource: &resource_v1.Resource{Attributes: []*common_v1.KeyValue{
			tempopb.MakeKeyValueStringPtr("service.name", "client"),
			tempopb.MakeKeyValueStringPtr("beast", "manticore"),
		}},
		ScopeSpans: []*trace_v1.ScopeSpans{{}},
	}
	server := &trace_v1.ResourceSpans{
		Resource: &resource_v1.Resource{Attributes: []*common_v1.KeyValue{
			tempopb.MakeKeyValueStringPtr("service.name", "server"),
			tempopb.MakeKeyValueStringPtr("god", "athena"),
		}},
		ScopeSpans: []*trace_v1.ScopeSpans{{}},
	}

	for i := 0; i < edges; i++ {
		traceID := benchmarkSGTraceID(i)
		clientSpanID := benchmarkSGSpanID(i*2 + 1)
		client.ScopeSpans[0].Spans = append(client.ScopeSpans[0].Spans, &trace_v1.Span{
			TraceId:           traceID,
			SpanId:            clientSpanID,
			Name:              "GET /api/:id",
			Kind:              trace_v1.Span_SPAN_KIND_CLIENT,
			StartTimeUnixNano: uint64(1_700_000_000_000_000_000 + i),
			EndTimeUnixNano:   uint64(1_700_000_000_050_000_000 + i),
			Status:            &trace_v1.Status{Code: trace_v1.Status_STATUS_CODE_OK},
			Attributes: []*common_v1.KeyValue{
				tempopb.MakeKeyValueStringPtr("peer.service", "server"),
				tempopb.MakeKeyValueStringPtr("beast", "client-"+strconv.Itoa(i%4)),
			},
		})
		if spanMultiplier {
			client.ScopeSpans[0].Spans[len(client.ScopeSpans[0].Spans)-1].Attributes = append(
				client.ScopeSpans[0].Spans[len(client.ScopeSpans[0].Spans)-1].Attributes,
				benchmarkSGDoubleAttr("sample.rate", 0.5),
			)
		}

		server.ScopeSpans[0].Spans = append(server.ScopeSpans[0].Spans, &trace_v1.Span{
			TraceId:           traceID,
			SpanId:            benchmarkSGSpanID(i*2 + 2),
			ParentSpanId:      clientSpanID,
			Name:              "GET /api/:id",
			Kind:              trace_v1.Span_SPAN_KIND_SERVER,
			StartTimeUnixNano: uint64(1_700_000_000_010_000_000 + i),
			EndTimeUnixNano:   uint64(1_700_000_000_040_000_000 + i),
			Status:            &trace_v1.Status{Code: trace_v1.Status_STATUS_CODE_OK},
			Attributes: []*common_v1.KeyValue{
				tempopb.MakeKeyValueStringPtr("god", "server-"+strconv.Itoa(i%4)),
			},
		})
		if spanMultiplier {
			server.ScopeSpans[0].Spans[len(server.ScopeSpans[0].Spans)-1].Attributes = append(
				server.ScopeSpans[0].Spans[len(server.ScopeSpans[0].Spans)-1].Attributes,
				benchmarkSGDoubleAttr("sample.rate", 0.5),
			)
		}
	}

	return &tempopb.PushSpansRequest{Batches: []*trace_v1.ResourceSpans{client, server}}
}

func benchmarkSGTraceID(i int) []byte {
	var id [16]byte
	binary.BigEndian.PutUint64(id[:8], 0x1020304050607080+uint64(i))
	binary.BigEndian.PutUint64(id[8:], uint64(i+1))
	return id[:]
}

func benchmarkSGSpanID(i int) []byte {
	var id [8]byte
	binary.BigEndian.PutUint64(id[:], uint64(i+1))
	return id[:]
}

func benchmarkSGDoubleAttr(key string, value float64) *common_v1.KeyValue {
	kv := tempopb.MakeKeyValueDouble(key, value)
	return &kv
}
