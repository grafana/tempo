package deployments

import (
	"context"
	crand "crypto/rand"
	"testing"

	"github.com/grafana/dskit/user"
	"github.com/grafana/e2e"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/zipkinexporter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.opentelemetry.io/collector/pdata/ptrace"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"github.com/grafana/tempo/integration/util"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
)

func TestReceivers(t *testing.T) {
	util.WithTempoHarness(t, util.TestHarnessConfig{}, func(h *util.TempoHarness) {
		testReceivers := []struct {
			name           string
			createExporter func() (exporter.Traces, error)
		}{
			{
				name: "otlp gRPC",
				createExporter: func() (exporter.Traces, error) {
					factory := otlpexporter.NewFactory()
					cfg := factory.CreateDefaultConfig().(*otlpexporter.Config)
					cfg.ClientConfig = configgrpc.ClientConfig{
						Endpoint: h.Services[util.ServiceDistributor].Endpoint(4317),
						TLS: configtls.ClientConfig{
							Insecure: true,
						},
					}

					logger, _ := zap.NewDevelopment()
					return factory.CreateTraces(
						context.Background(),
						exporter.Settings{
							ID: component.NewID(factory.Type()),
							TelemetrySettings: component.TelemetrySettings{
								Logger:         logger,
								TracerProvider: tracenoop.NewTracerProvider(),
								MeterProvider:  metricnoop.NewMeterProvider(),
							},
							BuildInfo: component.NewDefaultBuildInfo(),
						},
						cfg,
					)
				},
			},
			{
				name: "otlp HTTP",
				createExporter: func() (exporter.Traces, error) {
					factory := otlphttpexporter.NewFactory()
					cfg := factory.CreateDefaultConfig().(*otlphttpexporter.Config)
					cfg.ClientConfig = confighttp.ClientConfig{
						Endpoint: "http://" + h.Services[util.ServiceDistributor].Endpoint(4318),
						TLS: configtls.ClientConfig{
							Insecure: true,
						},
					}

					logger, _ := zap.NewDevelopment()
					return factory.CreateTraces(
						context.Background(),
						exporter.Settings{
							ID: component.NewID(factory.Type()),
							TelemetrySettings: component.TelemetrySettings{
								Logger:         logger,
								TracerProvider: tracenoop.NewTracerProvider(),
								MeterProvider:  metricnoop.NewMeterProvider(),
							},
							BuildInfo: component.NewDefaultBuildInfo(),
						},
						cfg,
					)
				},
			},
			{
				name: "zipkin",
				createExporter: func() (exporter.Traces, error) {
					factory := zipkinexporter.NewFactory()
					cfg := factory.CreateDefaultConfig().(*zipkinexporter.Config)
					cfg.ClientConfig = confighttp.ClientConfig{
						Endpoint: "http://" + h.Services[util.ServiceDistributor].Endpoint(9411),
						TLS: configtls.ClientConfig{
							Insecure: true,
						},
					}
					cfg.Format = "json"

					logger, _ := zap.NewDevelopment()
					return factory.CreateTraces(
						context.Background(),
						exporter.Settings{
							ID: component.NewID(factory.Type()),
							TelemetrySettings: component.TelemetrySettings{
								Logger:         logger,
								TracerProvider: tracenoop.NewTracerProvider(),
								MeterProvider:  metricnoop.NewMeterProvider(),
							},
							BuildInfo: component.NewDefaultBuildInfo(),
						},
						cfg,
					)
				},
			},
			{
				name: "jaeger gRPC",
				createExporter: func() (exporter.Traces, error) {
					return util.NewJaegerGRPCExporter(h.Services[util.ServiceDistributor].Endpoint(14250))
				},
			},
		}

		for i, tc := range testReceivers {
			t.Run(tc.name, func(t *testing.T) {
				// create exporter
				exp, err := tc.createExporter()
				require.NoError(t, err)

				err = exp.Start(context.Background(), componenttest.NewNopHost())
				require.NoError(t, err)

				// make request
				traceID := make([]byte, 16)
				_, err = crand.Read(traceID)
				require.NoError(t, err)
				req := test.MakeTrace(20, traceID)

				// zipkin doesn't support events and will 400 if you attempt to push one. just strip
				// all events from the trace here
				if tc.name == "zipkin" {
					for _, b := range req.ResourceSpans {
						for _, ss := range b.ScopeSpans {
							for _, s := range ss.Spans {
								s.Events = nil
							}
						}
					}
				}

				b, err := req.Marshal()
				require.NoError(t, err)

				// unmarshal into otlp proto
				traces, err := (&ptrace.ProtoUnmarshaler{}).UnmarshalTraces(b)
				require.NoError(t, err)
				require.NotNil(t, traces)

				ctx := user.InjectOrgID(context.Background(), tempoUtil.FakeTenantID)
				ctx, err = user.InjectIntoGRPCRequest(ctx)
				require.NoError(t, err)

				// send traces to tempo
				err = exp.ConsumeTraces(ctx, traces)
				require.NoError(t, err)

				// shutdown to ensure traces are flushed
				require.NoError(t, exp.Shutdown(context.Background()))

				expectedTraces := i + 1
				require.NoError(t, h.Services[util.ServiceLiveStoreZoneA].WaitSumMetricsWithOptions(e2e.Equals(float64(expectedTraces)), []string{"tempo_live_store_traces_created_total"}, e2e.WaitMissingMetrics))

				// query for the trace
				trace, err := h.HTTPClient.QueryTrace(tempoUtil.TraceIDToHexString(traceID))
				require.NoError(t, err)

				// just compare spanCount because otel flattens all ILS into one
				assert.Equal(t, util.SpanCount(req), util.SpanCount(trace))
			})
		}
	})
}
