package e2e

import (
	"context"
	crand "crypto/rand"
	"testing"

	"github.com/grafana/dskit/user"
	"github.com/grafana/e2e"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/jaegerexporter"
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
	"go.opentelemetry.io/collector/pdata/ptrace"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	util "github.com/grafana/tempo/integration"
	"github.com/grafana/tempo/pkg/httpclient"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
)

const (
	configAllInOneLocal = "config-all-in-one-local.yaml"
)

func TestReceivers(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configAllInOneLocal, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	testReceivers := []struct {
		name     string
		factory  exporter.Factory
		config   func(exporter.Factory, string) component.Config
		endpoint string
	}{
		{
			"jaeger gRPC",
			jaegerexporter.NewFactory(),
			func(factory exporter.Factory, endpoint string) component.Config {
				exporterCfg := factory.CreateDefaultConfig()
				jaegerCfg := exporterCfg.(*jaegerexporter.Config)
				jaegerCfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
					Endpoint: endpoint,
					TLSSetting: configtls.TLSClientSetting{
						Insecure: true,
					},
				}
				return jaegerCfg
			},
			tempo.Endpoint(14250),
		},
		{
			"otlp gRPC",
			otlpexporter.NewFactory(),
			func(factory exporter.Factory, endpoint string) component.Config {
				exporterCfg := factory.CreateDefaultConfig()
				otlpCfg := exporterCfg.(*otlpexporter.Config)
				otlpCfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
					Endpoint: endpoint,
					TLSSetting: configtls.TLSClientSetting{
						Insecure: true,
					},
				}
				return otlpCfg
			},
			tempo.Endpoint(4317),
		},
		{
			"zipkin",
			zipkinexporter.NewFactory(),
			func(factory exporter.Factory, endpoint string) component.Config {
				exporterCfg := factory.CreateDefaultConfig()
				zipkinCfg := exporterCfg.(*zipkinexporter.Config)
				zipkinCfg.HTTPClientSettings = confighttp.HTTPClientSettings{
					Endpoint: endpoint,
					TLSSetting: configtls.TLSClientSetting{
						Insecure: true,
					},
				}
				zipkinCfg.Format = "json"
				return zipkinCfg
			},
			"http://" + tempo.Endpoint(9411),
		},
	}

	for _, tc := range testReceivers {
		t.Run(tc.name, func(t *testing.T) {
			// create exporter
			logger, _ := zap.NewDevelopment()
			exporter, err := tc.factory.CreateTracesExporter(
				context.Background(),
				exporter.CreateSettings{
					TelemetrySettings: component.TelemetrySettings{
						Logger:         logger,
						TracerProvider: tracenoop.NewTracerProvider(),
						MeterProvider:  metricnoop.NewMeterProvider(),
					},
					BuildInfo: component.NewDefaultBuildInfo(),
				},
				tc.config(tc.factory, tc.endpoint),
			)
			require.NoError(t, err)

			err = exporter.Start(context.Background(), componenttest.NewNopHost())
			require.NoError(t, err)

			// make request
			traceID := make([]byte, 16)
			_, err = crand.Read(traceID)
			require.NoError(t, err)
			req := test.MakeTrace(20, traceID)

			// zipkin doesn't support events and will 400 if you attempt to push one. just strip
			// all events from the trace here
			if tc.name == "zipkin" {
				for _, b := range req.Batches {
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
			err = exporter.ConsumeTraces(ctx, traces)
			require.NoError(t, err)

			// shutdown to ensure traces are flushed
			require.NoError(t, exporter.Shutdown(context.Background()))

			// query for the trace
			client := httpclient.New("http://"+tempo.Endpoint(3200), tempoUtil.FakeTenantID)
			trace, err := client.QueryTrace(tempoUtil.TraceIDToHexString(traceID))
			require.NoError(t, err)

			// just compare spanCount because otel flattens all ILS into one
			assert.Equal(t, spanCount(req), spanCount(trace))
		})
	}
}
