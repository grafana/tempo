package frontend

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/tempodb/backend"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// extractTenant extracts tenant ID from request context and returns HTTP error response if extraction fails
func extractTenant(req *http.Request, logger log.Logger) (string, *http.Response) {
	tenant, err := user.ExtractOrgID(req.Context())
	if err != nil {
		level.Error(logger).Log("msg", "failed to extract tenant id", "err", err)
		return "", &http.Response{
			StatusCode: http.StatusBadRequest,
			Status:     http.StatusText(http.StatusBadRequest),
			Body:       io.NopCloser(strings.NewReader(err.Error())),
		}
	}
	return tenant, nil
}

func acceptAllBlocks(_ *backend.BlockMeta) bool { return true }

// setQueryShapeSpanAttrs stamps the query-shape attributes on the given span.
// Called from each sharder after starting its span.
func setQueryShapeSpanAttrs(span trace.Span, qs pipeline.QueryShape) {
	span.SetAttributes(
		attribute.String("query_type", qs.Type),
		attribute.Int("query_weight", qs.Weight),
		attribute.Int("query_sub_queries", qs.SubQueries),
		attribute.Int("query_conditions", qs.Conditions),
		attribute.Int("query_regex_conditions", qs.RegexConditions),
		attribute.Bool("query_has_or", qs.HasOr),
		attribute.Bool("query_needs_full_trace", qs.NeedsFullTrace),
		attribute.Bool("query_select_all", qs.SelectAll),
	)
}

// recordResult logs the response fields and mirrors them as span attributes.
//
//nolint:revive // logger first to match logWithShape's calling convention: recordResult(level.Info(logger), ctx, ...)
func recordResult(logger log.Logger, ctx context.Context, fields ...any) {
	logWithShape(logger, ctx, fields...)
	setSpanAttrsWithShape(ctx, fields...)
}

// setSpanAttrsWithShape mirrors response log fields plus query-shape fields as
// attributes on the span in ctx, if one is recording.
func setSpanAttrsWithShape(ctx context.Context, fields ...any) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	attrs := make([]attribute.KeyValue, 0, len(fields)/2)
	for i := 0; i+1 < len(fields); i += 2 {
		key, ok := fields[i].(string)
		if !ok || key == "msg" || key == "traceID" { // log field keys that are redundant on a span
			continue
		}
		if attr, ok := spanAttr(key, fields[i+1]); ok {
			attrs = append(attrs, attr)
		}
	}
	span.SetAttributes(attrs...)

	if qs, ok := pipeline.QueryShapeFromContext(ctx); ok {
		setQueryShapeSpanAttrs(span, qs)
	}
}

// spanAttr converts one log field value to a span attribute. ok=false for nil values.
func spanAttr(key string, val any) (attribute.KeyValue, bool) {
	switch v := val.(type) {
	case nil:
		return attribute.KeyValue{}, false
	case string:
		return attribute.String(key, v), true
	case bool:
		return attribute.Bool(key, v), true
	case int:
		return attribute.Int(key, v), true
	case int64:
		return attribute.Int64(key, v), true
	case uint32:
		return attribute.Int64(key, int64(v)), true
	case uint64:
		// OTel attributes don't support uint64; cast to int64 when safe, otherwise fall back to string.
		if v > ^uint64(0)>>1 {
			return attribute.String(key, fmt.Sprint(v)), true
		}
		return attribute.Int64(key, int64(v)), true //nolint:gosec // G115
	case float64:
		return attribute.Float64(key, v), true
	case error:
		return attribute.String(key, v.Error()), true
	default:
		return attribute.String(key, fmt.Sprint(v)), true
	}
}

// logWithShape emits a per-query response log line with query-shape fields
// appended when a shape is stamped on the context. The caller picks the level
// by wrapping the logger, e.g. logWithShape(level.Info(logger), ctx, ...).
//
//nolint:revive // logger comes first so callers can write logWithShape(level.Info(logger), ctx, ...).
func logWithShape(logger log.Logger, ctx context.Context, fields ...any) {
	qs, ok := pipeline.QueryShapeFromContext(ctx)
	if !ok {
		_ = logger.Log(fields...)
		return
	}
	out := make([]any, 0, len(fields)+16)
	out = append(out, fields...)
	out = append(out,
		"query_type", qs.Type,
		"query_weight", qs.Weight,
		"query_sub_queries", qs.SubQueries,
		"query_conditions", qs.Conditions,
		"query_regex_conditions", qs.RegexConditions,
		"query_has_or", qs.HasOr,
		"query_needs_full_trace", qs.NeedsFullTrace,
		"query_select_all", qs.SelectAll,
	)
	_ = logger.Log(out...)
}
