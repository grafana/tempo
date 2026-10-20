package tracing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestRecordErr(t *testing.T) {
	testCases := []struct {
		name       string
		err        error
		wantCode   codes.Code
		wantEvents int
	}{
		{
			name:       "nil error sets Ok",
			err:        nil,
			wantCode:   codes.Ok,
			wantEvents: 0,
		},
		{
			name:       "non-nil error sets Error and records an exception event",
			err:        errors.New("boom"),
			wantCode:   codes.Error,
			wantEvents: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			exporter := tracetest.NewInMemoryExporter()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
			defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

			_, span := tp.Tracer("test").Start(context.Background(), "op")
			RecordErr(span, tc.err)
			span.End()

			require.NoError(t, tp.ForceFlush(context.Background()))
			spans := exporter.GetSpans()
			require.Len(t, spans, 1)

			require.Equal(t, tc.wantCode, spans[0].Status.Code)
			require.Len(t, spans[0].Events, tc.wantEvents)
		})
	}
}

func TestEndSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	// doThing mirrors the intended call site: EndSpan is deferred while err
	// is still nil, and must observe the value err holds at return time, not
	// at the point the defer was registered.
	doThing := func() (err error) {
		_, span := tp.Tracer("test").Start(context.Background(), "op")
		defer EndSpan(span, &err)

		err = errors.New("boom")
		return err
	}

	require.EqualError(t, doThing(), "boom")
	require.NoError(t, tp.ForceFlush(context.Background()))

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	require.True(t, spans[0].EndTime.After(spans[0].StartTime), "span should have been ended")
	require.Equal(t, codes.Error, spans[0].Status.Code)
}
