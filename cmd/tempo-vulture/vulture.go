package main

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Vulture is used to send traces to Tempo, and then read those traces back out to verify that the service is operating correctly.
type Vulture struct {
	completeChan chan completeTrace
	tracer       trace.Tracer
	writeBackoff time.Duration
	readBackoff  time.Duration
}

// NewVulture creates a new Vulture, or an error if any.
func NewVulture(writeBackoff, readBackoff time.Duration) (*Vulture, error) {
	ctx := context.Background()

	otelClient, err := newOtelGRPCClient(tempoPushURL)
	if err != nil {
		return nil, err
	}

	otelExporter, err := otlptrace.New(ctx, otelClient)
	if err != nil {
		return nil, err
	}

	bsp := sdktrace.NewSimpleSpanProcessor(otelExporter)
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(bsp))

	tracer := tp.Tracer("tempo-vulture")

	v := Vulture{
		completeChan: make(chan completeTrace, 100),
		writeBackoff: writeBackoff,
		readBackoff:  readBackoff,
		tracer:       tracer,
	}

	return &v, nil
}

func (v *Vulture) Start() (context.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())

	go v.generateShortSpans(ctx, v.writeBackoff)
	go v.generateLongSpans(ctx, v.writeBackoff)
	go v.validateSpans(ctx, v.readBackoff)

	return ctx, cancel
}

func (v *Vulture) Stop() {
	// defer func() { _ = tp.Shutdown(ctx) }()
}

// generateShortSpans is sued to create a trace that includes a single span.
func (v *Vulture) generateShortSpans(ctx context.Context, dur time.Duration) {
	ticker := time.NewTicker(dur)

	for {
		select {
		case <-ticker.C:
			spanCtx, span := v.tracer.Start(ctx, "write")
			logSpan(spanCtx, v.tracer, span)
			span.End()
			v.completeChan <- completeTrace{span.SpanContext().TraceID(), 1}
		}
	}

}

// generateLongSpans is used to create a trace that includes a number of spands
// generated over a somewhat random time duration before being sent to Tempo.
func (v *Vulture) generateLongSpans(ctx context.Context, dur time.Duration) {
	ticker := time.NewTicker(dur)

	var longSpan *trace.Span
	var longSpanCtx context.Context
	var longSpanCount int64

	highWaterMark := generateRandomInt(1, 25)

	log := logger.With(
		zap.Int64("high_water_mark", highWaterMark),
		zap.String("method", "long"),
	)

	for {
		select {
		case <-ticker.C:
			if longSpan == nil {
				c, s := v.tracer.Start(ctx, "longWrite")
				longSpan = &s
				longSpanCtx = c

				log = log.With(
					zap.String("traceID", s.SpanContext().TraceID().String()),
				)

				log.Info("started new long span")
			}

			span := *longSpan
			longSpanCount++

			// create a span for this itteration
			_, x := v.tracer.Start(longSpanCtx, fmt.Sprintf("itteration: %d", longSpanCount))
			x.End()

			if longSpanCount > highWaterMark {
				logSpan(longSpanCtx, v.tracer, span)
				span.End()
				traceID := span.SpanContext().TraceID()
				// the number of itterations +1 for the logSpan() call.
				v.completeChan <- completeTrace{traceID, int(longSpanCount) + 1}

				log.Info("finished long span")
				// reset
				longSpanCount = 0
				longSpan = nil

				highWaterMark = generateRandomInt(1, 50)

				log = logger.With(
					zap.Int64("high_water_mark", highWaterMark),
					zap.String("method", "long"),
				)
			}
		}
	}
}

func (v *Vulture) validateSpans(ctx context.Context, dur time.Duration) {
	ticker := time.NewTicker(dur)

	for {
		select {
		case now := <-ticker.C:
			time.Sleep(500 * time.Millisecond)

			readIds := 0
			idCount := len(v.completeChan)
			readCtx, span := v.tracer.Start(ctx, "read")

			for readIds <= idCount {
				readIds++

				completeTrace := <-v.completeChan
				_, idSpan := v.tracer.Start(readCtx, completeTrace.traceID.String())

				span.SetName(completeTrace.traceID.String())
				span.SetAttributes(attribute.String("time", now.String()))

				// query the trace
				metrics, err := queryTempoAndAnalyze(tempoQueryURL, completeTrace)
				if err != nil {
					metricErrorTotal.Inc()
				}

				metricTracesInspected.Add(float64(metrics.requested))
				metricTracesErrors.WithLabelValues("requestfailed").Add(float64(metrics.requestFailed))
				metricTracesErrors.WithLabelValues("notfound").Add(float64(metrics.notFound))
				metricTracesErrors.WithLabelValues("missingspans").Add(float64(metrics.missingSpans))
				metricTracesErrors.WithLabelValues("incorrectspancount").Add(float64(metrics.incorrectSpanCount))

				idSpan.End()
			}

			logSpan(readCtx, v.tracer, span)
			span.End()
			// the numebr of itterations +1 for the logSpan() call
			v.completeChan <- completeTrace{span.SpanContext().TraceID(), readIds + 1}
		}
	}
}
