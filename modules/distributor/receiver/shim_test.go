package receiver

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	dslog "github.com/grafana/dskit/log"
	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/grafana/tempo/modules/generator"
	"github.com/grafana/tempo/pkg/tempopb"
)

// These tests use the OpenTelemetry Collector Exporters to validate the different protocols
func TestShim_integration(t *testing.T) {
	randomTraces := testdata.GenerateTraces(5)
	headers := map[string]configopaque.String{generator.NoGenerateMetricsContextKey: "true"}

	testCases := []struct {
		name              string
		receiverCfg       map[string]interface{}
		factory           exporter.Factory
		exporterCfg       component.Config
		expectedTransport string
	}{
		{
			name: "otlpexporter",
			receiverCfg: map[string]interface{}{
				"otlp": map[string]interface{}{
					"protocols": map[string]interface{}{
						"grpc": nil,
					},
				},
			},
			factory: otlpexporter.NewFactory(),
			exporterCfg: &otlpexporter.Config{
				ClientConfig: configgrpc.ClientConfig{
					Endpoint: "127.0.0.1:4317",
					TLS: configtls.ClientConfig{
						Insecure: true,
					},
					Headers: headers,
				},
			},
			expectedTransport: "grpc",
		},
		{
			name: "otlphttpexporter - JSON encoding",
			receiverCfg: map[string]interface{}{
				"otlp": map[string]interface{}{
					"protocols": map[string]interface{}{
						"http": nil,
					},
				},
			},
			factory: otlphttpexporter.NewFactory(),
			exporterCfg: &otlphttpexporter.Config{
				ClientConfig: confighttp.ClientConfig{
					Endpoint: "http://127.0.0.1:4318",
					Headers:  headers,
				},
				Encoding: otlphttpexporter.EncodingJSON,
			},
			expectedTransport: "http",
		},
		{
			name: "otlphttpexporter - proto encoding",
			receiverCfg: map[string]interface{}{
				"otlp": map[string]interface{}{
					"protocols": map[string]interface{}{
						"http": nil,
					},
				},
			},
			factory: otlphttpexporter.NewFactory(),
			exporterCfg: &otlphttpexporter.Config{
				ClientConfig: confighttp.ClientConfig{
					Endpoint: "http://127.0.0.1:4318",
					Headers:  headers,
				},
				Encoding: otlphttpexporter.EncodingProto,
			},
			expectedTransport: "http",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			pusher := &capturingPusher{t: t}
			reg := prometheus.NewPedanticRegistry()

			stopShim := runReceiverShim(t, testCase.receiverCfg, pusher, reg)
			defer stopShim()

			exporter, stopExporter := runOTelExporter(t, testCase.factory, testCase.exporterCfg)
			defer stopExporter()

			err := exporter.ConsumeTraces(context.Background(), randomTraces)
			assert.NoError(t, err)

			receivedTraces := pusher.GetAndClearTraces()
			// We should only have received one push request
			require.Len(t, receivedTraces, 1)

			assert.Equal(t, randomTraces, receivedTraces[0])

			count, err := testutil.GatherAndCount(reg, "tempo_receiver_accepted_spans", "tempo_receiver_refused_spans")
			assert.NoError(t, err)
			assert.Equal(t, 2, count)

			expected := `
			# HELP tempo_receiver_accepted_spans Number of spans successfully pushed into the pipeline.
			# TYPE tempo_receiver_accepted_spans counter
			tempo_receiver_accepted_spans{receiver="otlp/otlp_receiver", transport="<transport>"} 5
			# HELP tempo_receiver_refused_spans Number of spans that could not be pushed into the pipeline.
			# TYPE tempo_receiver_refused_spans counter
			tempo_receiver_refused_spans{receiver="otlp/otlp_receiver", transport="<transport>"} 0
			`
			expectedWithTransport := strings.ReplaceAll(expected, "<transport>", testCase.expectedTransport)

			err = testutil.GatherAndCompare(reg, strings.NewReader(expectedWithTransport), "tempo_receiver_accepted_spans", "tempo_receiver_refused_spans")
			assert.NoError(t, err)
		})
	}
}

func runReceiverShim(t *testing.T, receiverCfg map[string]interface{}, pusher TracesPusher, reg prometheus.Registerer) func() {
	level := dslog.Level{}
	_ = level.Set("info")

	shim, err := New(receiverCfg, pusher, FakeTenantMiddleware(), 0, level, reg)
	require.NoError(t, err)

	err = services.StartAndAwaitRunning(context.Background(), shim)
	require.NoError(t, err)

	return func() {
		err := services.StopAndAwaitTerminated(context.Background(), shim)
		if errors.Is(err, context.Canceled) {
			return
		}
		assert.NoError(t, err)
	}
}

func runOTelExporter(t *testing.T, factory exporter.Factory, cfg component.Config) (exporter.Traces, func()) {
	exporter, err := factory.CreateTraces(
		context.Background(),
		exporter.Settings{
			ID: component.MustNewID(factory.Type().String()),
			TelemetrySettings: component.TelemetrySettings{
				Logger:         zap.NewNop(),
				TracerProvider: tracenoop.NewTracerProvider(),
				MeterProvider:  metricnoop.NewMeterProvider(),
			},
		},
		cfg,
	)
	require.NoError(t, err)

	err = exporter.Start(context.Background(), &mockHost{})
	require.NoError(t, err)

	return exporter, func() {
		err = exporter.Shutdown(context.Background())
		assert.NoError(t, err, "traces exporter shutting down failed")
	}
}

type mockHost struct{}

var _ component.Host = (*mockHost)(nil)

func (m *mockHost) GetFactory(component.Kind, component.Type) component.Factory {
	panic("implement me")
}

func (m *mockHost) GetExtensions() map[component.ID]component.Component {
	panic("implement me")
}

type capturingPusher struct {
	traces []ptrace.Traces
	t      *testing.T
}

func (p *capturingPusher) GetAndClearTraces() []ptrace.Traces {
	traces := p.traces
	p.traces = nil
	return traces
}

func (p *capturingPusher) PushTraces(ctx context.Context, t ptrace.Traces) (*tempopb.PushResponse, error) {
	p.traces = append(p.traces, t)

	// Ensure that headers from the exporter config are propagated.
	assert.True(p.t, generator.ExtractNoGenerateMetrics(ctx))

	return &tempopb.PushResponse{}, nil
}

// TestWrapRetryableError confirms that errors are wrapped as expected
func TestWrapRetryableError(t *testing.T) {
	// no wrapping b/c not a grpc error
	err := errors.New("test error")
	wrapped := wrapErrorIfRetryable(err, nil)
	require.Equal(t, err, wrapped)
	require.False(t, isRetryable(wrapped))

	// no wrapping b/c not a resource exhausted grpc error
	err = status.Error(codes.FailedPrecondition, "failed precondition")
	wrapped = wrapErrorIfRetryable(err, nil)
	require.Equal(t, err, wrapped)
	require.False(t, isRetryable(wrapped))

	// no wrapping b/c no configured duration
	err = status.Error(codes.ResourceExhausted, "res exhausted")
	wrapped = wrapErrorIfRetryable(err, nil)
	require.Equal(t, err, wrapped)
	require.False(t, isRetryable(wrapped))

	// wrapping b/c this is a resource exhausted grpc error
	err = status.Error(codes.ResourceExhausted, "res exhausted")
	wrapped = wrapErrorIfRetryable(err, durationpb.New(time.Second))
	require.NotEqual(t, err, wrapped)
	require.True(t, isRetryable(wrapped))
}

func isRetryable(err error) bool {
	st, ok := status.FromError(err)

	if !ok {
		return false
	}

	for _, detail := range st.Details() {
		if _, ok := detail.(*errdetails.RetryInfo); ok {
			return true
		}
	}
	return false
}
