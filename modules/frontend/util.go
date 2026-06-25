package frontend

import (
	"context"
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
		attribute.String("queryType", qs.Type),
		attribute.Int("queryWeight", qs.Weight),
		attribute.Int("querySubQueries", qs.SubQueries),
		attribute.Int("queryConditions", qs.Conditions),
		attribute.Int("queryRegexConditions", qs.RegexConditions),
		attribute.Bool("queryHasOr", qs.HasOr),
		attribute.Bool("queryNeedsFullTrace", qs.NeedsFullTrace),
		attribute.Bool("querySelectAll", qs.SelectAll),
	)
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
