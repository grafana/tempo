package api

import (
	"context"
	"testing"
	"time"

	"github.com/grafana/e2e"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/pkg/tempopb"
)

func TestLinkTraversal(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configAllInOneLocal, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	ctx := context.Background()
	cancelfunc, tp, err := setupTracerProvider(ctx, tempo, t)
	require.NoError(t, err)
	defer cancelfunc()

	tracer := tp.Tracer("test-tracer")

	gatewayCtx, gatewaySpan := tracer.Start(context.Background(), "gateway",
		oteltrace.WithAttributes(
			attribute.String("service.name", "gateway"),
			attribute.String("name", "gateway"),
		),
	)
	time.Sleep(10 * time.Millisecond)
	gatewaySpanCtx := gatewaySpan.SpanContext()
	gatewaySpan.End()
	require.NoError(t, tp.ForceFlush(gatewayCtx))

	backendCtx, backendSpan := tracer.Start(context.Background(), "backend",
		oteltrace.WithAttributes(
			attribute.String("service.name", "backend"),
			attribute.String("name", "backend"),
		),
		oteltrace.WithLinks(oteltrace.Link{
			SpanContext: gatewaySpanCtx,
		}),
	)
	time.Sleep(10 * time.Millisecond)
	backendSpanCtx := backendSpan.SpanContext()
	backendSpan.End()
	require.NoError(t, tp.ForceFlush(backendCtx))

	databaseCtx, databaseSpan := tracer.Start(context.Background(), "database",
		oteltrace.WithAttributes(
			attribute.String("service.name", "database"),
			attribute.String("name", "database"),
		),

		oteltrace.WithLinks(oteltrace.Link{
			SpanContext: backendSpanCtx,
		}),
	)
	time.Sleep(20 * time.Millisecond)
	databaseSpan.End()
	require.NoError(t, tp.ForceFlush(databaseCtx))

	require.NoError(t, tp.ForceFlush(ctx))

	waitForSearchBackend(t, tempo, 3)

	testCases := []struct {
		name        string
		query       string
		expectError bool
		minTraces   int
		description string
	}{
		{
			name:        "link to operator - two hops",
			query:       `{name="database"} &->> {name="backend"}`,
			expectError: false,
			minTraces:   2,
			description: "Should find database and backend traces linked together",
		},
		{
			name:        "link to operator - three hops",
			query:       `{name="database"} &->> {name="backend"} &->> {name="gateway"}`,
			expectError: false,
			minTraces:   3,
			description: "Should find all three traces linked in chain",
		},
		{
			name:        "link from operator - two hops",
			query:       `{name="gateway"} &<<- {name="backend"}`,
			expectError: false,
			minTraces:   2,
			description: "Should find gateway and backend traces linked together (reverse direction)",
		},
		{
			name:        "link from operator - three hops",
			query:       `{name="gateway"} &<<- {name="backend"} &<<- {name="database"}`,
			expectError: false,
			minTraces:   3,
			description: "Should find all three traces linked in chain (reverse direction)",
		},
		{
			name:        "non-union link to",
			query:       `{name="database"} ->> {name="backend"}`,
			expectError: false,
			minTraces:   1,
			description: "Non-union operator should work but may return fewer results",
		},
		{
			name:        "non-union link from",
			query:       `{name="gateway"} <<- {name="backend"}`,
			expectError: false,
			minTraces:   1,
			description: "Non-union operator should work but may return fewer results",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Execute search query
			resp, err := executeSearchQuery(tempo.Endpoint(tempoPort), tc.query)

			if tc.expectError {
				require.Error(t, err, "Expected error for query: %s", tc.query)
				return
			}

			require.NoError(t, err, "Query failed: %s - %s", tc.query, tc.description)
			require.NotNil(t, resp, "Response should not be nil")

			// Verify we got results
			require.GreaterOrEqual(t, len(resp.Traces), tc.minTraces,
				"Expected at least %d traces for query: %s - %s", tc.minTraces, tc.query, tc.description)

			// Verify metrics are present
			require.NotNil(t, resp.Metrics, "Metrics should be present")
		})
	}
}

// executeSearchQuery executes a TraceQL search query against Tempo
func executeSearchQuery(endpoint, query string) (*tempopb.SearchResponse, error) {
	client := httpclient.New("http://"+endpoint, "")
	return client.SearchTraceQL(query)
}

func setupTracerProvider(ctx context.Context, tempo *e2e.HTTPService, t *testing.T) (func(), *tracesdk.TracerProvider, error) {
	t.Helper()

	conn, err := grpc.NewClient(
		tempo.Endpoint(4317),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	// Create OTLP exporter
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	require.NoError(t, err)

	// Create resource
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("link-traversal-test"),
		semconv.ServiceVersion("0.0.0-test"),
		attribute.String("environment", "test"),
	)

	// Create tracer provider
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	cancelfn := func() {
		err := exporter.Shutdown(context.Background())
		require.NoError(t, err)
		err = tp.Shutdown(context.Background())
		require.NoError(t, err)
		err = conn.Close()
		require.NoError(t, err)
	}

	return cancelfn, tp, nil
}
