package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	tempoEndpoint = "localhost:4317"
)

func main() {
	log.Printf("Link Traversal Test Data Generator")
	log.Printf("Connecting to Tempo at: %s", tempoEndpoint)

	// Create OTLP exporter
	ctx := context.Background()
	exporter, err := createExporter(ctx)
	if err != nil {
		log.Fatalf("Failed to create exporter: %v", err)
	}
	defer func() {
		if err := exporter.Shutdown(ctx); err != nil {
			log.Printf("Failed to shutdown exporter: %v", err)
		}
	}()

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(newResource()),
	)
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Printf("Failed to shutdown tracer provider: %v", err)
		}
	}()
	otel.SetTracerProvider(tp)

	// Generate test data: database → backend → gateway
	generateLinkedTraces(ctx, tp)

	// Flush all traces
	if err := tp.ForceFlush(ctx); err != nil {
		log.Printf("Failed to flush traces: %v", err)
		return
	}

	log.Println("Visit http://localhost:3000/explore to see the traces")

	log.Println(
		"\nTest queries:",
		"\n {span.service.name=\"database\"} &->> {span.service.name=\"backend\"} &->> {span.service.name=\"gateway\"}",
		"\n {span.service.name=\"gateway\"} &<<- {span.service.name=\"backend\"} &<<- {span.service.name=\"database\"}",
	)
}

func createExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	conn, err := grpc.NewClient(
		tempoEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	return exporter, nil
}

func newResource() *resource.Resource {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("link-traversal-test"),
		semconv.ServiceVersion("1.0.0"),
		attribute.String("environment", "test"),
	)
}

func generateLinkedTraces(ctx context.Context, tp *sdktrace.TracerProvider) {
	tracer := tp.Tracer("link-traversal-test")

	log.Println("\nGenerating linked traces: database → backend → gateway")

	// Step 1: Create gateway span (terminal - no links to anyone)
	gatewayCtx, gatewaySpan := tracer.Start(ctx, "gateway-request",
		trace.WithAttributes(
			attribute.String("service.name", "gateway"),
			attribute.String("name", "gateway"),
			attribute.String("http.method", "GET"),
			attribute.String("http.url", "/api/users"),
		),
	)
	time.Sleep(10 * time.Millisecond)
	gatewaySpanCtx := gatewaySpan.SpanContext()
	gatewaySpan.End()
	log.Printf("  ✓ Gateway span: traceID=%s, spanID=%s",
		gatewaySpanCtx.TraceID().String(), gatewaySpanCtx.SpanID().String())

	tp.ForceFlush(gatewayCtx)

	// Step 2: Create backend span with link to gateway
	backendCtx, backendSpan := tracer.Start(context.Background(), "backend-process",
		trace.WithAttributes(
			attribute.String("service.name", "backend"),
			attribute.String("name", "backend"),
		),
		trace.WithLinks(trace.Link{
			SpanContext: gatewaySpanCtx,
		}),
	)
	time.Sleep(15 * time.Millisecond)
	backendSpanCtx := backendSpan.SpanContext()
	backendSpan.End()
	log.Printf("  ✓ Backend span: traceID=%s, spanID=%s → links to gateway",
		backendSpanCtx.TraceID().String(), backendSpanCtx.SpanID().String())

	tp.ForceFlush(backendCtx)

	// Step 3: Create database span with link to backend
	dbCtx, dbSpan := tracer.Start(context.Background(), "database-query",
		trace.WithAttributes(
			attribute.String("service.name", "database"),
			attribute.String("name", "database"),
		),
		trace.WithLinks(trace.Link{
			SpanContext: backendSpanCtx,
		}),
	)
	time.Sleep(20 * time.Millisecond)
	dbSpanCtx := dbSpan.SpanContext()
	dbSpan.End()
	log.Printf("  ✓ Database span: traceID=%s, spanID=%s → links to backend",
		dbSpanCtx.TraceID().String(), dbSpanCtx.SpanID().String())

	tp.ForceFlush(dbCtx)
}
