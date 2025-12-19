package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestLinkTraversal tests cross-trace link traversal queries
// This test creates multiple traces with links between them and verifies
// that link traversal operators work correctly
func TestLinkTraversal(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configAllInOneLocal, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	// Create OTLP exporter
	ctx := context.Background()
	conn, err := grpc.NewClient(
		tempo.Endpoint(4317),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	require.NoError(t, err)
	defer exporter.Shutdown(ctx)

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("link-traversal-test"),
			semconv.ServiceVersion("test"),
		),
	)
	require.NoError(t, err)

	// Create tracer provider
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(res),
	)
	defer tp.Shutdown(ctx)

	tracer := tp.Tracer("test-tracer")

	// Create a chain of linked traces:
	// database -> backend -> gateway (database links TO backend, backend links TO gateway)
	// This means: database has a link pointing to backend's span, backend has a link pointing to gateway's span

	// Create Trace 3: gateway span (terminal - no outgoing links)
	gatewayCtx, span3 := tracer.Start(context.Background(), "gateway-request",
		oteltrace.WithAttributes(
			attribute.String("service.name", "gateway"),
			attribute.String("name", "gateway"),
		),
	)
	span3Ctx := span3.SpanContext()
	span3.End()
	tp.ForceFlush(gatewayCtx)

	// Create Trace 2: backend span with link to gateway
	backendCtx, span2 := tracer.Start(context.Background(), "backend-process",
		oteltrace.WithAttributes(
			attribute.String("service.name", "backend"),
			attribute.String("name", "backend"),
		),
		oteltrace.WithLinks(oteltrace.Link{
			SpanContext: span3Ctx,
		}),
	)
	span2Ctx := span2.SpanContext()
	span2.End()
	tp.ForceFlush(backendCtx)

	// Create Trace 1: database span with link to backend
	databaseCtx, span1 := tracer.Start(context.Background(), "database-query",
		oteltrace.WithAttributes(
			attribute.String("service.name", "database"),
			attribute.String("name", "database"),
		),
		oteltrace.WithLinks(oteltrace.Link{
			SpanContext: span2Ctx,
		}),
	)
	span1.End()
	tp.ForceFlush(databaseCtx)

	// Flush traces
	require.NoError(t, tp.ForceFlush(ctx))

	waitForSearchBackend(t, tempo, 1)

	// Wait for traces to be written
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(3), []string{"tempo_ingester_traces_created_total"}, e2e.WaitMissingMetrics))

	// Test link traversal queries
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

			t.Logf("Query: %s", tc.query)
			t.Logf("Description: %s", tc.description)
			t.Logf("Found %d traces", len(resp.Traces))
			t.Logf("Inspected %d traces, %d bytes", resp.Metrics.InspectedTraces, resp.Metrics.InspectedBytes)
		})
	}
}

// TestLinkTraversalWithConditions tests link traversal with additional conditions
func TestLinkTraversalWithConditions(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configAllInOneLocal, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	// Create OTLP exporter
	ctx := context.Background()
	conn, err := grpc.NewClient(
		tempo.Endpoint(4317),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	require.NoError(t, err)
	defer exporter.Shutdown(ctx)

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("link-traversal-conditions-test"),
			semconv.ServiceVersion("test"),
		),
	)
	require.NoError(t, err)

	// Create tracer provider
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(res),
	)
	defer tp.Shutdown(ctx)

	tracer := tp.Tracer("test-tracer")

	// Create traces with different statuses
	// database -> backend (database links TO backend)

	// Trace 2: failed backend call (terminal - no outgoing links)
	backendCtx, span2 := tracer.Start(context.Background(), "backend-error",
		oteltrace.WithAttributes(
			attribute.String("service.name", "backend"),
			attribute.String("name", "backend"),
			attribute.String("http.status_code", "500"),
		),
	)
	span2.SetStatus(codes.Error, "internal error")
	span2Ctx := span2.SpanContext()
	span2.End()
	tp.ForceFlush(backendCtx)

	// Trace 1: successful database query with link to backend
	databaseCtx, span1 := tracer.Start(context.Background(), "db-success",
		oteltrace.WithAttributes(
			attribute.String("service.name", "database"),
			attribute.String("name", "database"),
			attribute.String("db.operation", "select"),
		),
		oteltrace.WithLinks(oteltrace.Link{
			SpanContext: span2Ctx,
		}),
	)
	span1.SetStatus(codes.Ok, "success")
	span1.End()
	tp.ForceFlush(databaseCtx)

	// Flush traces
	require.NoError(t, tp.ForceFlush(ctx))

	waitForSearchBackend(t, tempo, 1)

	// Test queries with conditions
	testCases := []struct {
		name        string
		query       string
		expectError bool
		minTraces   int
	}{
		{
			name:        "link with status condition",
			query:       `{name="database" && status=ok} &->> {name="backend" && status=error}`,
			expectError: false,
			minTraces:   1,
		},
		{
			name:        "link with attribute condition",
			query:       `{name="database" && span.db.operation="select"} &->> {name="backend"}`,
			expectError: false,
			minTraces:   1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := executeSearchQuery(tempo.Endpoint(tempoPort), tc.query)

			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if resp != nil {
				require.GreaterOrEqual(t, len(resp.Traces), tc.minTraces)
				t.Logf("Query: %s - Found %d traces", tc.query, len(resp.Traces))
			}
		})
	}
}

// executeSearchQuery executes a TraceQL search query against Tempo
func executeSearchQuery(endpoint, query string) (*tempopb.SearchResponse, error) {
	// Build search URL
	searchURL := fmt.Sprintf("http://%s/api/search", endpoint)

	// Add query parameter
	params := url.Values{}
	params.Add("q", query)
	params.Add("limit", "100")

	fullURL := fmt.Sprintf("%s?%s", searchURL, params.Encode())

	// Execute request
	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var searchResp tempopb.SearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &searchResp, nil
}
