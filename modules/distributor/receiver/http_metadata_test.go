package receiver

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/apache/thrift/lib/go/thrift"
	dslog "github.com/grafana/dskit/log"
	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/user"
	"github.com/jaegertracing/jaeger-idl/thrift-gen/jaeger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/grafana/tempo/v3/modules/generator"
)

func TestShim_HTTPMetadata(t *testing.T) {
	jaegerBody, err := thrift.NewTSerializer().Write(context.Background(), &jaeger.Batch{
		Process: &jaeger.Process{ServiceName: "test"},
		Spans: []*jaeger.Span{{
			TraceIdLow:    1,
			SpanId:        2,
			OperationName: "test",
			StartTime:     1,
			Duration:      1,
		}},
	})
	require.NoError(t, err)

	tests := []struct {
		name        string
		receiverCfg map[string]interface{}
		url         string
		contentType string
		body        []byte
	}{
		{
			name: "zipkin",
			receiverCfg: map[string]interface{}{
				"zipkin": map[string]interface{}{"endpoint": "127.0.0.1:9411"},
			},
			url:         "http://127.0.0.1:9411/api/v2/spans",
			contentType: "application/json",
			body:        []byte(`[{"traceId":"0000000000000001","id":"0000000000000002","name":"test","timestamp":1,"duration":1}]`),
		},
		{
			name: "jaeger thrift HTTP",
			receiverCfg: map[string]interface{}{
				"jaeger": map[string]interface{}{
					"protocols": map[string]interface{}{
						"thrift_http": map[string]interface{}{"endpoint": "127.0.0.1:14268"},
					},
				},
			},
			url:         "http://127.0.0.1:14268/api/traces",
			contentType: "application/x-thrift",
			body:        jaegerBody,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			metadata := make(chan client.Metadata, 1)
			middleware := MiddlewareFunc(func(next consumer.Traces) consumer.Traces {
				return ConsumeTracesFunc(func(ctx context.Context, traces ptrace.Traces) error {
					metadata <- client.FromContext(ctx).Metadata
					return next.ConsumeTraces(ctx, traces)
				})
			})
			shim, err := New(tc.receiverCfg, &erroringPusher{}, middleware, 0, dslog.Level{}, prometheus.NewRegistry())
			require.NoError(t, err)
			require.NoError(t, services.StartAndAwaitRunning(context.Background(), shim))
			t.Cleanup(func() {
				err := services.StopAndAwaitTerminated(context.Background(), shim)
				if !errors.Is(err, context.Canceled) {
					require.NoError(t, err)
				}
			})

			req, err := http.NewRequest(http.MethodPost, tc.url, bytes.NewReader(tc.body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", tc.contentType)
			req.Header.Set(user.OrgIDHeaderName, "test-tenant")
			req.Header.Set(generator.NoGenerateMetricsContextKey, "true")

			httpClient := &http.Client{Timeout: 5 * time.Second}
			resp, err := httpClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, http.StatusAccepted, resp.StatusCode)

			select {
			case got := <-metadata:
				require.Equal(t, []string{"test-tenant"}, got.Get(user.OrgIDHeaderName))
				require.Equal(t, []string{"true"}, got.Get(generator.NoGenerateMetricsContextKey))
			case <-time.After(5 * time.Second):
				t.Fatal("receiver did not forward HTTP metadata")
			}
		})
	}
}
