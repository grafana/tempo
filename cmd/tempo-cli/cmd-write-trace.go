package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var appName = "tempo-cli"

type writeTraceCmd struct {
	tracingOptions

	logger log.Logger

	SpanCount int64 `arg:"" help:"The number of spans to send in the trace"`
}

func (cmd *writeTraceCmd) Run(_ *globalOptions) error {
	cmd.logger = newLogger()

	shutdownTracer, err := cmd.installOpenTelemetryTracer()
	if err != nil {
		return errors.Wrap(err, "error initializing tracer")
	}
	defer shutdownTracer()

	tracer := otel.Tracer(appName + ".main")

	n := int64(0)

	ctx, wrapSpan := tracer.Start(context.Background(), "two")
	defer wrapSpan.End()

	for n < cmd.SpanCount {
		opts := trace.WithSpanKind(trace.SpanKindUnspecified)
		_, span := tracer.Start(ctx, fmt.Sprintf("span %d", n), opts)
		span.SetAttributes(attribute.KeyValue{Key: "itteration", Value: attribute.Int64Value(n)})
		span.SetAttributes(attribute.KeyValue{Key: "base64TraceID", Value: attribute.StringValue(base64.StdEncoding.EncodeToString([]byte(span.SpanContext().TraceID().String())))})
		n++
		span.End()
	}

	level.Info(cmd.logger).Log("msg", "trace complete", "traceID", wrapSpan.SpanContext().TraceID().String())

	return nil
}

func (cmd *writeTraceCmd) installOpenTelemetryTracer() (func(), error) {
	if cmd.OTELEndpoint == "" {
		return func() {}, nil
	}

	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(appName),
		),
		resource.WithHost(),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize trace resuorce")
	}

	conn, err := grpc.DialContext(ctx, cmd.OTELEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial otel grpc")
	}

	options := []otlptracegrpc.Option{otlptracegrpc.WithGRPCConn(conn)}

	if cmd.TenantID != "" {
		options = append(options,
			otlptracegrpc.WithHeaders(map[string]string{"X-Scope-OrgID": cmd.TenantID}))
	}

	traceExporter, err := otlptracegrpc.New(ctx, options...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to creat trace exporter")
	}

	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	otel.SetTracerProvider(tracerProvider)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})

	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tracerProvider.Shutdown(ctx); err != nil {
			level.Error(cmd.logger).Log("msg", "failed to shutdown tracer", "err", err)
			os.Exit(1)
		}
	}

	return shutdown, nil
}
