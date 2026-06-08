package telemetry

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func generateTracerProvider(ctx context.Context, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	if !hasOTELConfig(os.Environ()) {
		return nil, nil
	}
	traceExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to set up trace exporter: %w", err)
	}
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traceExporter,
			sdktrace.WithBatchTimeout(time.Second*5)),
	)
	return tracerProvider, nil
}

func hasOTELConfig(env []string) bool {
	for _, envName := range env {
		if strings.HasPrefix(envName, "OTEL_") {
			return true
		}
	}
	return false
}

func generatePropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func Setup(ctx context.Context, res *resource.Resource) (func(context.Context) error, error) {
	var shutdownFuncs []func(context.Context) error
	var err error

	shutdown := func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	handleError := func(e error) {
		err = errors.Join(e, shutdown(ctx))
	}

	finalRes, err := resource.Merge(resource.Default(), res)
	if err != nil {
		return nil, err
	}

	prop := generatePropagator()
	otel.SetTextMapPropagator(prop)

	tracerProvider, err := generateTracerProvider(ctx, finalRes)
	if err != nil {
		handleError(err)
		return shutdown, err
	}
	if tracerProvider != nil {
		shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
		otel.SetTracerProvider(tracerProvider)
	}

	return shutdown, err
}

func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

func FailSpanWithError(span trace.Span, err error) {
	if err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, "")
}

func SucceedSpan(span trace.Span) {
	span.SetStatus(codes.Ok, "")
}

func InjectIntoEnv(ctx context.Context, env []string) []string {
	carrier := make(propagation.MapCarrier)
	prop := otel.GetTextMapPropagator()
	prop.Inject(ctx, &carrier)
	env = append(env, fmt.Sprintf("BAGGAGE=%s", carrier.Get("baggage")))
	env = append(env, fmt.Sprintf("TRACEPARENT=%s", carrier.Get("traceparent")))
	env = append(env, fmt.Sprintf("TRACESTATE=%s", carrier.Get("tracestate")))
	return env
}

func LoadEnvironmentCarrier() propagation.TextMapCarrier {
	carrier := make(propagation.MapCarrier)
	carrier.Set("baggage", os.Getenv("BAGGAGE"))
	carrier.Set("traceparent", os.Getenv("TRACEPARENT"))
	carrier.Set("tracestate", os.Getenv("TRACESTATE"))
	return carrier
}
