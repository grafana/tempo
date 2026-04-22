package util

import (
	"context"
	"math/rand"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/jaegertracing/jaeger-idl/proto-gen/api_v2"
	thrift "github.com/jaegertracing/jaeger-idl/thrift-gen/jaeger"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/jaeger"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/pdata/ptrace"
	mnoop "go.opentelemetry.io/otel/metric/noop"
	tnoop "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	xScopeOrgIDHeader   = "x-scope-orgid"
	authorizationHeader = "authorization"
)

func MakeThriftBatch() *thrift.Batch {
	return MakeThriftBatchWithSpanCount(1)
}

func MakeThriftBatchWithSpanCount(n int) *thrift.Batch {
	return MakeThriftBatchWithSpanCountAttributeAndName(n, "my operation", "", "y", "xx", "x")
}

func MakeThriftBatchWithSpanCountAttributeAndName(n int, name, resourceValue, spanValue, resourceTag, spanTag string) *thrift.Batch {
	var spans []*thrift.Span

	traceIDLow := rand.Int63()  // nolint:gosec // G404: Use of weak random number generator
	traceIDHigh := rand.Int63() // nolint:gosec // G404: Use of weak random number generator
	for range n {
		spans = append(spans, &thrift.Span{
			TraceIdLow:    traceIDLow,
			TraceIdHigh:   traceIDHigh,
			SpanId:        rand.Int63(), // nolint:gosec // G404: Use of weak random number generator
			ParentSpanId:  0,
			OperationName: name,
			References:    nil,
			Flags:         0,
			StartTime:     time.Now().Add(-3*time.Second).UnixNano() / 1000, // microsecconds
			Duration:      1,
			Tags: []*thrift.Tag{
				{
					Key:  spanTag,
					VStr: &spanValue,
				},
			},
			Logs: nil,
		})
	}

	return &thrift.Batch{
		Process: &thrift.Process{
			ServiceName: "my-service",
			Tags: []*thrift.Tag{
				{
					Key:   resourceTag,
					VType: thrift.TagType_STRING,
					VStr:  &resourceValue,
				},
			},
		},
		Spans: spans,
	}
}

func (h *TempoHarness) WriteTraceInfo(traceInfo *util.TraceInfo, tenant string) error {
	endpoint := h.Services[ServiceDistributor].Endpoint(4317)
	exporter, err := NewJaegerToOTLPExporterWithAuth(endpoint, tenant, "", false)
	if err != nil {
		return err
	}
	return traceInfo.EmitAllBatches(exporter)
}

func (h *TempoHarness) WriteJaegerBatch(batch *thrift.Batch, tenant string) error {
	endpoint := h.Services[ServiceDistributor].Endpoint(4317)
	exporter, err := NewJaegerToOTLPExporterWithAuth(endpoint, tenant, "", false)
	if err != nil {
		return err
	}
	return exporter.EmitBatch(context.Background(), batch)
}

func (h *TempoHarness) WriteTempoProtoTraces(traces *tempopb.Trace, tenant string) error {
	b, err := traces.Marshal()
	if err != nil {
		return err
	}

	// unmarshal into otlp proto
	otlpTraces, err := (&ptrace.ProtoUnmarshaler{}).UnmarshalTraces(b)
	if err != nil {
		return err
	}

	endpoint := h.Services[ServiceDistributor].Endpoint(4317)
	exporter, err := newOtelGRPCExporterWithAuth(endpoint, tenant, "", false)
	if err != nil {
		return err
	}
	if err := exporter.ConsumeTraces(context.Background(), otlpTraces); err != nil {
		return err
	}
	return exporter.Shutdown(context.Background())
}

func newOtelGRPCExporterWithAuth(endpoint, orgID, basicAuthToken string, useTLS bool) (exporter.Traces, error) {
	factory := otlpexporter.NewFactory()
	exporterCfg := factory.CreateDefaultConfig()
	otlpCfg := exporterCfg.(*otlpexporter.Config)

	// Configure headers for authentication (gRPC metadata format)
	var headers configopaque.MapList
	if orgID != "" {
		headers.Set(xScopeOrgIDHeader, configopaque.String(orgID))
	}
	if basicAuthToken != "" {
		headers.Set(authorizationHeader, configopaque.String("Basic "+basicAuthToken))
	}

	otlpCfg.ClientConfig = configgrpc.ClientConfig{
		Endpoint: endpoint,
		TLS: configtls.ClientConfig{
			Insecure: !useTLS,
		},
		Headers: headers,
	}

	// Disable retries to get immediate error feedback
	otlpCfg.RetryConfig.Enabled = false
	// Disable queueing
	otlpCfg.QueueConfig = configoptional.None[exporterhelper.QueueBatchConfig]()
	// beef up the timeout to 30 seconds to avoid flakes
	otlpCfg.TimeoutConfig.Timeout = 30 * time.Second

	logger, _ := zap.NewDevelopment()
	te, err := factory.CreateTraces(
		context.Background(),
		exporter.Settings{
			ID: component.MustNewID(factory.Type().String()),
			TelemetrySettings: component.TelemetrySettings{
				Logger:         logger,
				TracerProvider: tnoop.NewTracerProvider(),
				MeterProvider:  mnoop.NewMeterProvider(),
			},
			BuildInfo: component.NewDefaultBuildInfo(),
		},
		otlpCfg,
	)
	if err != nil {
		return nil, err
	}
	err = te.Start(context.Background(), componenttest.NewNopHost())
	if err != nil {
		return nil, err
	}
	return te, nil
}

type JaegerToOTLPExporter struct {
	exporter exporter.Traces
}

func NewJaegerToOTLPExporter(endpoint string) (*JaegerToOTLPExporter, error) {
	return NewJaegerToOTLPExporterWithAuth(endpoint, "", "", false)
}

func NewJaegerToOTLPExporterWithAuth(endpoint, orgID, basicAuthToken string, useTLS bool) (*JaegerToOTLPExporter, error) {
	exp, err := newOtelGRPCExporterWithAuth(endpoint, orgID, basicAuthToken, useTLS)
	if err != nil {
		return nil, err
	}
	return &JaegerToOTLPExporter{exporter: exp}, nil
}

// EmitBatch converts a Jaeger Thrift batch to OpenTelemetry traces formats
// and forwards them to the configured OTLP endpoint.
func (c *JaegerToOTLPExporter) EmitBatch(ctx context.Context, b *thrift.Batch) error {
	traces, err := jaeger.ThriftToTraces(b)
	if err != nil {
		return err
	}
	return c.exporter.ConsumeTraces(ctx, traces)
}

// JaegerGRPCExporter wraps a gRPC client that sends traces to the Jaeger gRPC receiver (port 14250)
type JaegerGRPCExporter struct {
	client    *grpc.ClientConn
	collector api_v2.CollectorServiceClient
}

// NewJaegerGRPCExporter creates a new Jaeger gRPC exporter that sends traces to the Jaeger gRPC receiver
func NewJaegerGRPCExporter(endpoint string) (*JaegerGRPCExporter, error) {
	conn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &JaegerGRPCExporter{
		client:    conn,
		collector: api_v2.NewCollectorServiceClient(conn),
	}, nil
}

// Start implements component.Component
func (e *JaegerGRPCExporter) Start(_ context.Context, _ component.Host) error {
	return nil
}

// Shutdown implements component.Component
func (e *JaegerGRPCExporter) Shutdown(_ context.Context) error {
	return e.client.Close()
}

// ConsumeTraces converts OTLP traces to Jaeger format and sends them via gRPC
func (e *JaegerGRPCExporter) ConsumeTraces(ctx context.Context, traces ptrace.Traces) error {
	batches := jaeger.ProtoFromTraces(traces)

	for _, batch := range batches {
		_, err := e.collector.PostSpans(ctx, &api_v2.PostSpansRequest{
			Batch: *batch,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// Capabilities implements consumer.Traces
func (e *JaegerGRPCExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}
