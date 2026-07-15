package frontend

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestExtractTenant(t *testing.T) {
	logger := log.NewNopLogger()

	t.Run("success case - tenant extracted from context", func(t *testing.T) {
		// Create request with tenant in context
		req, err := http.NewRequest("GET", "/api/traces/123", nil)
		require.NoError(t, err)

		// Inject tenant ID into context
		ctx := user.InjectOrgID(req.Context(), "test-tenant")
		req = req.WithContext(ctx)

		// Call extractTenant
		tenant, errResp := extractTenant(req, logger)

		// Verify success
		assert.Equal(t, "test-tenant", tenant)
		assert.Nil(t, errResp)
	})

	t.Run("success case - single-tenant mode", func(t *testing.T) {
		// Create request with single-tenant ID
		req, err := http.NewRequest("GET", "/api/search", nil)
		require.NoError(t, err)

		// Inject single-tenant ID (simulating fake auth middleware)
		ctx := user.InjectOrgID(req.Context(), "single-tenant")
		req = req.WithContext(ctx)

		// Call extractTenant
		tenant, errResp := extractTenant(req, logger)

		// Verify success
		assert.Equal(t, "single-tenant", tenant)
		assert.Nil(t, errResp)
	})

	t.Run("error case - no tenant in context", func(t *testing.T) {
		// Create request without tenant in context
		req, err := http.NewRequest("GET", "/api/traces/123", nil)
		require.NoError(t, err)

		// Call extractTenant
		tenant, errResp := extractTenant(req, logger)

		// Verify error response
		assert.Empty(t, tenant)
		require.NotNil(t, errResp)

		// Check HTTP response details
		assert.Equal(t, http.StatusBadRequest, errResp.StatusCode)
		assert.Equal(t, "Bad Request", errResp.Status)

		// Read response body
		bodyBytes, err := io.ReadAll(errResp.Body)
		require.NoError(t, err)
		bodyStr := string(bodyBytes)

		// Should contain error message about missing org ID
		assert.Contains(t, bodyStr, "no org id")
	})
}

func TestSpanAttr(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want attribute.KeyValue
		ok   bool
	}{
		{"nil", nil, attribute.KeyValue{}, false},
		{"string", "foo", attribute.String("k", "foo"), true},
		{"bool", true, attribute.Bool("k", true), true},
		{"int", 42, attribute.Int("k", 42), true},
		{"int64", int64(-7), attribute.Int64("k", -7), true},
		{"uint32", uint32(3600), attribute.Int64("k", 3600), true},
		{"uint64", uint64(123456), attribute.Int64("k", 123456), true},
		{"float64", 1.5, attribute.Float64("k", 1.5), true},
		{"error", errors.New("boom"), attribute.String("k", "boom"), true},
		{"fallback", time.Second, attribute.String("k", "1s"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := spanAttr("k", tt.val)
			require.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestSetSpanAttrsWithShape(t *testing.T) {
	attrsOf := func(rec *tracetest.SpanRecorder) map[attribute.Key]attribute.Value {
		ended := rec.Ended()
		require.Len(t, ended, 1)
		out := map[attribute.Key]attribute.Value{}
		for _, kv := range ended[0].Attributes() {
			out[kv.Key] = kv.Value
		}
		return out
	}

	t.Run("mirrors fields, skips nil values and redundant keys", func(t *testing.T) {
		rec := tracetest.NewSpanRecorder()
		tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
		ctx, span := tp.Tracer("test").Start(context.Background(), "test")

		setSpanAttrsWithShape(ctx,
			"msg", "search response",
			"traceID", "abcd",
			"tenant", "test-tenant",
			"duration_seconds", 1.5,
			"status_code", 200,
			"path", "/api/search",
			"query", "{}",
			"inspected_bytes", uint64(1024),
			"error", nil,
		)
		span.End()

		attrs := attrsOf(rec)
		assert.NotContains(t, attrs, attribute.Key("msg"))
		assert.NotContains(t, attrs, attribute.Key("traceID"))
		assert.NotContains(t, attrs, attribute.Key("tenant"))
		assert.NotContains(t, attrs, attribute.Key("duration_seconds"))
		assert.NotContains(t, attrs, attribute.Key("status_code"))
		assert.NotContains(t, attrs, attribute.Key("path"))
		assert.NotContains(t, attrs, attribute.Key("error"))
		assert.Equal(t, "{}", attrs["query"].AsString())
		assert.Equal(t, int64(1024), attrs["inspected_bytes"].AsInt64())
		// no shape stamped on ctx -> no shape attrs
		assert.NotContains(t, attrs, attribute.Key("query_type"))
	})

	t.Run("appends query-shape attrs when stamped on ctx", func(t *testing.T) {
		rec := tracetest.NewSpanRecorder()
		tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
		ctx, span := tp.Tracer("test").Start(context.Background(), "test")
		ctx = pipeline.WithQueryShapeCell(ctx)

		// stamp the shape the same way production does: via the weight middleware
		httpReq, err := http.NewRequest(http.MethodGet, "http://example.com/api/search?q="+url.QueryEscape("{ span.foo = `bar` }"), nil)
		require.NoError(t, err)
		httpReq = httpReq.WithContext(ctx)
		rt := pipeline.NewWeightRequestWare(pipeline.TraceQLSearch, pipeline.WeightsConfig{
			RequestWithWeights:   true,
			MaxTraceQLConditions: 4,
			MaxRegexConditions:   1,
		}).Wrap(pipeline.GetRoundTripperFunc())
		_, err = rt.RoundTrip(pipeline.NewHTTPRequest(httpReq))
		require.NoError(t, err)

		setSpanAttrsWithShape(ctx, "query", "{ span.foo = `bar` }")
		span.End()

		attrs := attrsOf(rec)
		assert.Equal(t, "{ span.foo = `bar` }", attrs["query"].AsString())
		assert.Equal(t, pipeline.QueryTypeSearch, attrs["query_type"].AsString())
		assert.GreaterOrEqual(t, attrs["query_weight"].AsInt64(), int64(1))
		assert.Equal(t, int64(1), attrs["query_sub_queries"].AsInt64())
		assert.Equal(t, int64(1), attrs["query_conditions"].AsInt64())
		assert.Equal(t, int64(0), attrs["query_regex_conditions"].AsInt64())
		assert.False(t, attrs["query_has_or"].AsBool())
		assert.False(t, attrs["query_needs_full_trace"].AsBool())
		assert.False(t, attrs["query_select_all"].AsBool())
	})

	t.Run("no-op without a recording span", func(_ *testing.T) {
		// must not panic with a noop span from a bare context
		setSpanAttrsWithShape(context.Background(), "query", "{}")
	})
}
