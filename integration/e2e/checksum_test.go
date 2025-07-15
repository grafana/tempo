package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/grafana/dskit/user"
	"github.com/grafana/e2e"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"gopkg.in/yaml.v2"

	"github.com/grafana/tempo/cmd/tempo/app"
	"github.com/grafana/tempo/integration/e2e/backend"
	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/httpclient"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
)

const (
	configS3Checksum = "./config-s3-checksum.tmpl.yaml"
)

func TestS3Checksum(t *testing.T) {
	checksumTypes := []string{
		"None",
		"SHA256",
		"SHA1",
		"CRC32",
		"CRC32C",
		"CRC64NVME",
	}

	for _, checksumType := range checksumTypes {
		t.Run(checksumType, func(t *testing.T) {
			t.Logf("Setting up scenario for checksum type: %s", checksumType)
			s, err := e2e.NewScenario("tempo_e2e")
			require.NoError(t, err)
			defer s.Close()

			t.Log("Prepare run config")
			tmplConfig := map[string]any{
				"ChecksumType": checksumType,
			}
			config, err := util.CopyTemplateToSharedDir(s, configS3Checksum, "config.yaml", tmplConfig)
			require.NoError(t, err)

			// load final config
			var cfg app.Config
			buff, err := os.ReadFile(config)
			require.NoError(t, err)
			err = yaml.UnmarshalStrict(buff, &cfg)
			require.NoError(t, err)

			t.Log("Set up the backend")
			_, err = backend.New(s, cfg)
			require.NoError(t, err)

			t.Log("Start tempo")
			tempo := util.NewTempoAllInOne()
			require.NoError(t, s.StartAndWaitReady(tempo))

			// wait for backend to be ready
			time.Sleep(2 * time.Second)

			// create OTLP exporter
			exporter, err := util.NewOtelGRPCExporter(tempo.Endpoint(4317))
			require.NoError(t, err)

			err = exporter.Start(context.Background(), componenttest.NewNopHost())
			require.NoError(t, err)

			t.Log("Inject trace into tempo")
			// generate trace
			traceID := test.ValidTraceID(nil)
			req := test.MakeTrace(10, traceID)
			b, err := req.Marshal()
			require.NoError(t, err)

			traces, err := (&ptrace.ProtoUnmarshaler{}).UnmarshalTraces(b)
			require.NoError(t, err)
			require.NotNil(t, traces)

			ctx := user.InjectOrgID(context.Background(), tempoUtil.FakeTenantID)
			ctx, err = user.InjectIntoGRPCRequest(ctx)
			require.NoError(t, err)

			// send traces to tempo
			err = exporter.ConsumeTraces(ctx, traces)
			require.NoError(t, err)
			require.NoError(t, exporter.Shutdown(context.Background())) // shutdown to ensure traces are flushed

			// wait for trace to be flushed to backend
			util.CallFlush(t, tempo)
			time.Sleep(5 * time.Second)
			util.CallFlush(t, tempo)

			// to be sure the block is flushed
			require.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(1), "tempo_ingester_blocks_flushed_total"))

			t.Log("Query for the trace")
			client := httpclient.New("http://"+tempo.Endpoint(3200), tempoUtil.FakeTenantID)
			trace, err := client.QueryTrace(tempoUtil.TraceIDToHexString(traceID))
			require.NoError(t, err)

			// verify the trace was stored and retrieved correctly
			assert.Equal(t, util.SpanCount(req), util.SpanCount(trace))
			assert.NotNil(t, trace)
			assert.NotEmpty(t, trace.ResourceSpans)
		})
	}
}
