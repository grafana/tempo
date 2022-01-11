package e2e

import (
	"context"
	"math/rand"
	"testing"

	cortex_e2e "github.com/cortexproject/cortex/integration/e2e"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/jaegerexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/zipkinexporter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weaveworks/common/user"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/model/otlp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	util "github.com/grafana/tempo/integration"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
)

const (
	configAllInOneLocal = "config-all-in-one-local.yaml"
)

func TestReceivers(t *testing.T) {
	s, err := cortex_e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configAllInOneLocal, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	testReceivers := []struct {
		name     string
		factory  component.ExporterFactory
		config   func(component.ExporterFactory, string) config.Exporter
		endpoint string
	}{
		{
			"jaeger gRPC",
			jaegerexporter.NewFactory(),
			func(factory component.ExporterFactory, endpoint string) config.Exporter {
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
			func(factory component.ExporterFactory, endpoint string) config.Exporter {
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
			func(factory component.ExporterFactory, endpoint string) config.Exporter {
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
				component.ExporterCreateSettings{
					TelemetrySettings: component.TelemetrySettings{
						Logger:         logger,
						TracerProvider: trace.NewNoopTracerProvider(),
						MeterProvider:  metric.NewNoopMeterProvider(),
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
			_, err = rand.Read(traceID)
			require.NoError(t, err)
			req := test.MakeTrace(20, traceID)
			b, err := req.Marshal()
			require.NoError(t, err)

			// unmarshal into otlp proto
			traces, err := otlp.NewProtobufTracesUnmarshaler().UnmarshalTraces(b)
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
			client := tempoUtil.NewClient("http://"+tempo.Endpoint(3200), tempoUtil.FakeTenantID)
			trace, err := client.QueryTrace(tempoUtil.TraceIDToHexString(traceID))
			require.NoError(t, err)

			// just compare spanCount because otel flattens all ILS into one
			assert.Equal(t, spanCount(req), spanCount(trace))
		})
	}
}
