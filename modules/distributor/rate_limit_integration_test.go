package distributor

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	dslog "github.com/grafana/dskit/log"
	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/grafana/tempo/v3/modules/distributor/receiver"
	"github.com/grafana/tempo/v3/modules/overrides"
)

// TestOversizedBatchOTLPStatus exercises the real distributor and OTel receivers.
// Even with retry information enabled, an oversized batch must fail permanently.
func TestOversizedBatchOTLPStatus(t *testing.T) {
	retryInfoEnabled := true
	d := prepare(t, overrides.Config{Defaults: overrides.Overrides{
		Ingestion: overrides.IngestionOverrides{
			RateStrategy:     overrides.LocalIngestionRateStrategy,
			RateLimitBytes:   500,
			BurstSizeBytes:   500,
			RetryInfoEnabled: &retryInfoEnabled,
		},
	}}, nil)
	d.cfg.RetryAfterOnResourceExhausted = 5 * time.Second
	receiverCfg := map[string]interface{}{
		"otlp": map[string]interface{}{
			"protocols": map[string]interface{}{"http": nil, "grpc": nil},
		},
	}
	shim, err := receiver.New(receiverCfg, d, receiver.FakeTenantMiddleware(),
		d.cfg.RetryAfterOnResourceExhausted, dslog.Level{}, prometheus.NewPedanticRegistry())
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	require.NoError(t, services.StartAndAwaitRunning(ctx, shim))
	t.Cleanup(func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		err := services.StopAndAwaitTerminated(stopCtx, shim)
		if !errors.Is(err, context.Canceled) {
			require.NoError(t, err)
		}
	})

	traces := ptrace.NewTraces()
	traces.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName(strings.Repeat("x", 501))
	request := ptraceotlp.NewExportRequestFromTraces(traces)

	t.Run("HTTP", func(t *testing.T) {
		body, err := request.MarshalProto()
		require.NoError(t, err)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			"http://127.0.0.1:4318/v1/traces", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-protobuf")
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		t.Logf("oversized batch -> HTTP %d", resp.StatusCode)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("gRPC", func(t *testing.T) {
		conn, err := grpc.NewClient("127.0.0.1:4317", grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()
		_, err = ptraceotlp.NewGRPCClient(conn).Export(ctx, request)
		require.Error(t, err)
		st := status.Convert(err)
		t.Logf("oversized batch -> gRPC %s, details: %v", st.Code(), st.Details())
		require.Equal(t, codes.InvalidArgument, st.Code())
		require.Empty(t, st.Details(), "permanent failures must not advertise retries")
	})
}
