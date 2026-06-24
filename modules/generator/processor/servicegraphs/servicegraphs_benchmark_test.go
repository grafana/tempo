package servicegraphs

import (
	"context"
	"encoding/binary"
	"strconv"
	"testing"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resource_v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/prometheus/client_golang/prometheus"
)

func BenchmarkServiceGraphsPushSpansSynthetic(b *testing.B) {
	for _, tc := range []struct {
		name           string
		tune           func(*Config)
		spanMultiplier bool
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
	} {
		b.Run(tc.name, func(b *testing.B) {
			cfg := Config{}
			cfg.RegisterFlagsAndApplyDefaults("", nil)
			cfg.HistogramBuckets = []float64{0.04, 0.1, 1}
			if tc.tune != nil {
				tc.tune(&cfg)
			}

			p, err := New(
				cfg,
				"bench",
				registry.NewTestRegistry(),
				log.NewNopLogger(),
				prometheus.NewCounter(prometheus.CounterOpts{}),
				prometheus.NewCounter(prometheus.CounterOpts{}),
			)
			if err != nil {
				b.Fatal(err)
			}
			defer p.Shutdown(context.Background())

			req := benchmarkServiceGraphRequest(100, tc.spanMultiplier)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.PushSpans(context.Background(), req)
			}
		})
	}
}

func benchmarkServiceGraphRequest(edges int, spanMultiplier bool) *tempopb.PushSpansRequest {
	client := &trace_v1.ResourceSpans{
		Resource: &resource_v1.Resource{Attributes: []*common_v1.KeyValue{
			benchmarkSGStringAttr("service.name", "client"),
			benchmarkSGStringAttr("beast", "manticore"),
		}},
		ScopeSpans: []*trace_v1.ScopeSpans{{}},
	}
	server := &trace_v1.ResourceSpans{
		Resource: &resource_v1.Resource{Attributes: []*common_v1.KeyValue{
			benchmarkSGStringAttr("service.name", "server"),
			benchmarkSGStringAttr("god", "athena"),
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
				benchmarkSGStringAttr("peer.service", "server"),
				benchmarkSGStringAttr("beast", "client-"+strconv.Itoa(i%4)),
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
				benchmarkSGStringAttr("god", "server-"+strconv.Itoa(i%4)),
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
	binary.BigEndian.PutUint64(id[8:], uint64(i+1))
	return id[:]
}

func benchmarkSGSpanID(i int) []byte {
	var id [8]byte
	binary.BigEndian.PutUint64(id[:], uint64(i+1))
	return id[:]
}

func benchmarkSGStringAttr(key, value string) *common_v1.KeyValue {
	return &common_v1.KeyValue{
		Key: key,
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{
			StringValue: value,
		}},
	}
}

func benchmarkSGDoubleAttr(key string, value float64) *common_v1.KeyValue {
	return &common_v1.KeyValue{
		Key: key,
		Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_DoubleValue{
			DoubleValue: value,
		}},
	}
}
