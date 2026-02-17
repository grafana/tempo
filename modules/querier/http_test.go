package querier

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/grafana/tempo/pkg/api"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestSetRequestSpanAttributes(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	ctx, span := tp.Tracer("test").Start(context.Background(), "root")

	req := httptest.NewRequest("GET", "/api/search", nil)
	req.Header.Set(api.HeaderPluginID, "grafana-assistant")

	setRequestSpanAttributes(span, req)
	span.End()

	spans := sr.Ended()
	require.Len(t, spans, 1)
	require.Contains(t, spans[0].Attributes(), attribute.String("pluginID", "grafana-assistant"))

	_ = tp.Shutdown(ctx)
}
