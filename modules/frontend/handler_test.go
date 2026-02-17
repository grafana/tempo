package frontend

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestFormatRequestHeaders(t *testing.T) {
	h := http.Header{}
	h.Add("X-Header-To-Log", "i should be logged!")
	h.Add("X-Header-To-Not-Log", "i shouldn't be logged!")

	fields := formatRequestHeaders(&h, []string{"X-Header-To-Log", "X-Header-Not-Present"})

	expected := []interface{}{
		"header_x_header_to_log",
		"i should be logged!",
	}

	require.Equal(t, expected, fields)
}

func TestSetRequestSpanAttributes(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	ctx, span := tp.Tracer("test").Start(context.Background(), "root")
	setRequestSpanAttributes(span, "tenant-a", "grafana-assistant")
	span.End()

	spans := sr.Ended()
	require.Len(t, spans, 1)
	require.Contains(t, spans[0].Attributes(), attribute.String("orgID", "tenant-a"))
	require.Contains(t, spans[0].Attributes(), attribute.String("pluginID", "grafana-assistant"))

	_ = tp.Shutdown(ctx)
}
