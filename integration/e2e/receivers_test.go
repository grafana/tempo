package e2e

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	cortex_e2e "github.com/cortexproject/cortex/integration/e2e"
	util "github.com/grafana/tempo/integration"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/jaegerexporter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weaveworks/common/user"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/model/otlp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func TestReceivers(t *testing.T) {
	testReceivers := []struct {
		name    string
		factory component.ExporterFactory
		config  func(component.ExporterFactory, string) config.Exporter
		port    int
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
				jaegerCfg.RetrySettings = exporterhelper.RetrySettings{
					Enabled: false,
				}
				return jaegerCfg
			},
			14250,
		},
	}

	for _, tc := range testReceivers {
		t.Run(tc.name, func(t *testing.T) {
			s, err := cortex_e2e.NewScenario("tempo_e2e")
			require.NoError(t, err)
			defer s.Close()

			require.NoError(t, util.CopyFileToSharedDir(s, configCompression, "config.yaml"))
			tempo := util.NewTempoAllInOne()
			require.NoError(t, s.StartAndWaitReady(tempo))

			// Create exporter
			exporter, err := tc.factory.CreateTracesExporter(
				context.Background(),
				component.ExporterCreateSettings{
					TelemetrySettings: component.TelemetrySettings{
						Logger:         zap.NewNop(),
						TracerProvider: trace.NewNoopTracerProvider(),
						MeterProvider:  metric.NewNoopMeterProvider(),
					},
					BuildInfo: component.NewDefaultBuildInfo(),
				},
				tc.config(tc.factory, tempo.Endpoint(tc.port)),
			)
			require.NoError(t, err)

			err = exporter.Start(context.Background(), componenttest.NewNopHost())
			require.NoError(t, err)

			traceID := make([]byte, 16)
			_, err = rand.Read(traceID)
			require.NoError(t, err)

			req := test.MakeRequest(20, traceID)
			b, err := req.Marshal()
			require.NoError(t, err)

			traces, err := otlp.NewProtobufTracesUnmarshaler().UnmarshalTraces(b)
			require.NoError(t, err)
			require.NotNil(t, traces)

			ctx := user.InjectOrgID(context.Background(), tempoUtil.FakeTenantID)
			ctx, err = user.InjectIntoGRPCRequest(ctx)
			require.NoError(t, err)

			err = exporter.ConsumeTraces(ctx, traces)
			require.NoError(t, err)

			client := tempoUtil.NewClient("http://"+tempo.Endpoint(3200), "")
			trace, err := client.QueryTrace(tempoUtil.TraceIDToHexString(traceID))
			require.NoError(t, err)

			fmt.Printf("%#v\n", trace.Batches[0])
			fmt.Printf("%#v\n", req.Batch)

			assert.True(t, equalTraces(&tempopb.Trace{Batches: []*v1_trace.ResourceSpans{req.Batch}}, trace))
		})
	}

}
