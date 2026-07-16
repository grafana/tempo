package querier

import (
	"context"
	"encoding/hex"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/validation"
)

func startQuerierSpan(ctx context.Context, name, query string, attrs ...attribute.KeyValue) (context.Context, oteltrace.Span, string, error) {
	tenantID, err := validation.ExtractValidTenantID(ctx)
	if err != nil {
		return ctx, nil, "", err
	}

	ctx, span := tracer.Start(ctx, name)
	setQuerierSpanAttributes(span, tenantID, query, attrs...)
	return ctx, span, tenantID, nil
}

func finishQuerierSpan(span oteltrace.Span, err error, metrics any) {
	setQuerierSpanMetrics(span, metrics)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}

func setQuerierSpanAttributes(span oteltrace.Span, tenantID, query string, attrs ...attribute.KeyValue) {
	if tenantID != "" {
		span.SetAttributes(attribute.String("tenant", tenantID))
	}
	if query != "" {
		span.SetAttributes(attribute.String("query", query))
	}
	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
}

func startTraceByIDSpan(ctx context.Context, name string, req *tempopb.TraceByIDRequest, timeStart, timeEnd time.Time) (context.Context, oteltrace.Span, string, error) {
	attrs := []attribute.KeyValue{
		attribute.String("traceID", hex.EncodeToString(req.TraceID)),
		attribute.String("queryMode", req.QueryMode),
		attribute.String("blockStart", req.BlockStart),
		attribute.String("blockEnd", req.BlockEnd),
		attribute.Bool("allowPartialTrace", req.AllowPartialTrace),
		attribute.String("timeStart", timeStart.String()),
		attribute.String("timeEnd", timeEnd.String()),
	}
	if !timeStart.IsZero() {
		attrs = append(attrs, attribute.Int64("startUnixNanos", timeStart.UnixNano()))
	}
	if !timeEnd.IsZero() {
		attrs = append(attrs, attribute.Int64("endUnixNanos", timeEnd.UnixNano()))
	}
	if !timeStart.IsZero() && !timeEnd.IsZero() {
		attrs = append(attrs, attribute.Int64("rangeNanos", timeEnd.Sub(timeStart).Nanoseconds()))
	}

	return startQuerierSpan(ctx, name, "", attrs...)
}

func startSearchRequestSpan(ctx context.Context, name string, req *tempopb.SearchRequest) (context.Context, oteltrace.Span, string, error) {
	return startQuerierSpan(ctx, name, req.Query,
		attribute.Int64("startUnixSeconds", int64(req.Start)),
		attribute.Int64("endUnixSeconds", int64(req.End)),
		attribute.Int64("rangeSeconds", int64(req.End)-int64(req.Start)),
	)
}

func startTagsRequestSpan(ctx context.Context, name string, req *tempopb.SearchTagsRequest) (context.Context, oteltrace.Span, string, error) {
	return startQuerierSpan(ctx, name, req.Query,
		attribute.String("scope", req.Scope),
		attribute.Int64("startUnixSeconds", int64(req.Start)),
		attribute.Int64("endUnixSeconds", int64(req.End)),
		attribute.Int64("rangeSeconds", int64(req.End)-int64(req.Start)),
	)
}

func startTagValuesRequestSpan(ctx context.Context, name string, req *tempopb.SearchTagValuesRequest) (context.Context, oteltrace.Span, string, error) {
	return startQuerierSpan(ctx, name, req.Query,
		attribute.String("tagName", req.TagName),
		attribute.Int64("startUnixSeconds", int64(req.Start)),
		attribute.Int64("endUnixSeconds", int64(req.End)),
		attribute.Int64("rangeSeconds", int64(req.End)-int64(req.Start)),
	)
}

func startSearchBlockSpan(ctx context.Context, name string, req *tempopb.SearchBlockRequest) (context.Context, oteltrace.Span, string, error) {
	return startQuerierSpan(ctx, name, req.SearchReq.Query,
		attribute.Int64("startUnixSeconds", int64(req.SearchReq.Start)),
		attribute.Int64("endUnixSeconds", int64(req.SearchReq.End)),
		attribute.Int64("rangeSeconds", int64(req.SearchReq.End)-int64(req.SearchReq.Start)),
		attribute.String("blockID", req.BlockID),
		attribute.String("version", req.Version),
		attribute.Int64("startPage", int64(req.StartPage)),
		attribute.Int64("pagesToSearch", int64(req.PagesToSearch)),
		attribute.Int64("blockSize", int64(req.Size_)),
		attribute.Int64("footerSize", int64(req.FooterSize)),
		attribute.Int64("totalRecords", int64(req.TotalRecords)),
	)
}

func startTagsBlockSpan(ctx context.Context, name string, req *tempopb.SearchTagsBlockRequest) (context.Context, oteltrace.Span, string, error) {
	return startQuerierSpan(ctx, name, req.SearchReq.Query,
		attribute.String("scope", req.SearchReq.Scope),
		attribute.Int64("startUnixSeconds", int64(req.SearchReq.Start)),
		attribute.Int64("endUnixSeconds", int64(req.SearchReq.End)),
		attribute.Int64("rangeSeconds", int64(req.SearchReq.End)-int64(req.SearchReq.Start)),
		attribute.String("blockID", req.BlockID),
		attribute.String("version", req.Version),
		attribute.Int64("startPage", int64(req.StartPage)),
		attribute.Int64("pagesToSearch", int64(req.PagesToSearch)),
		attribute.Int64("blockSize", int64(req.Size_)),
		attribute.Int64("footerSize", int64(req.FooterSize)),
		attribute.Int64("totalRecords", int64(req.TotalRecords)),
	)
}

func startTagValuesBlockSpan(ctx context.Context, name string, req *tempopb.SearchTagValuesBlockRequest) (context.Context, oteltrace.Span, string, error) {
	return startQuerierSpan(ctx, name, req.SearchReq.Query,
		attribute.String("tagName", req.SearchReq.TagName),
		attribute.Int64("startUnixSeconds", int64(req.SearchReq.Start)),
		attribute.Int64("endUnixSeconds", int64(req.SearchReq.End)),
		attribute.Int64("rangeSeconds", int64(req.SearchReq.End)-int64(req.SearchReq.Start)),
		attribute.String("blockID", req.BlockID),
		attribute.String("version", req.Version),
		attribute.Int64("startPage", int64(req.StartPage)),
		attribute.Int64("pagesToSearch", int64(req.PagesToSearch)),
		attribute.Int64("blockSize", int64(req.Size_)),
		attribute.Int64("footerSize", int64(req.FooterSize)),
		attribute.Int64("totalRecords", int64(req.TotalRecords)),
	)
}

func startQueryRangeSpan(ctx context.Context, name string, req *tempopb.QueryRangeRequest) (context.Context, oteltrace.Span, string, error) {
	return startQuerierSpan(ctx, name, req.Query,
		attribute.String("queryMode", req.QueryMode),
		attribute.Int64("startUnixNanos", int64(req.Start)),
		attribute.Int64("endUnixNanos", int64(req.End)),
		attribute.Int64("rangeNanos", int64(req.End)-int64(req.Start)),
		attribute.Int64("step", int64(req.Step)),
	)
}

func startQueryRangeBlockSpan(ctx context.Context, name string, req *tempopb.QueryRangeRequest) (context.Context, oteltrace.Span, string, error) {
	return startQuerierSpan(ctx, name, req.Query,
		attribute.String("queryMode", req.QueryMode),
		attribute.Int64("startUnixNanos", int64(req.Start)),
		attribute.Int64("endUnixNanos", int64(req.End)),
		attribute.Int64("rangeNanos", int64(req.End)-int64(req.Start)),
		attribute.Int64("step", int64(req.Step)),
		attribute.String("blockID", req.BlockID),
		attribute.String("version", req.Version),
		attribute.Int64("startPage", int64(req.StartPage)),
		attribute.Int64("pagesToSearch", int64(req.PagesToSearch)),
		attribute.Int64("blockSize", int64(req.Size_)),
		attribute.Int64("footerSize", int64(req.FooterSize)),
	)
}

func setQuerierSpanMetrics(span oteltrace.Span, metrics any) {
	if metrics == nil {
		return
	}
	switch m := metrics.(type) {
	case *tempopb.SearchMetrics:
		if m == nil {
			return
		}
		span.SetAttributes(
			attribute.Int64("inspectedBytes", int64(m.InspectedBytes)),
			attribute.Int64("inspectedTraces", int64(m.InspectedTraces)),
			attribute.Int64("inspectedSpans", int64(m.InspectedSpans)),
			attribute.Int64("backendReads", int64(m.BackendReads)),
			attribute.Int64("backendBytes", int64(m.BackendBytes)),
			attribute.Int64("totalBlocks", int64(m.TotalBlocks)),
			attribute.Int64("completedJobs", int64(m.CompletedJobs)),
			attribute.Int64("totalJobs", int64(m.TotalJobs)),
			attribute.Int64("totalBlockBytes", int64(m.TotalBlockBytes)),
		)
		for k, v := range m.AdditionalMetrics {
			span.SetAttributes(attribute.Int64("additionalMetrics."+k, v))
		}
	case *tempopb.MetadataMetrics:
		if m == nil {
			return
		}
		span.SetAttributes(
			attribute.Int64("inspectedBytes", int64(m.InspectedBytes)),
			attribute.Int64("backendReads", int64(m.BackendReads)),
			attribute.Int64("backendBytes", int64(m.BackendBytes)),
			attribute.Int64("totalBlocks", int64(m.TotalBlocks)),
			attribute.Int64("completedJobs", int64(m.CompletedJobs)),
			attribute.Int64("totalJobs", int64(m.TotalJobs)),
			attribute.Int64("totalBlockBytes", int64(m.TotalBlockBytes)),
		)
		for k, v := range m.AdditionalMetrics {
			span.SetAttributes(attribute.Int64("additionalMetrics."+k, v))
		}
	case *tempopb.TraceByIDMetrics:
		if m == nil {
			return
		}
		span.SetAttributes(
			attribute.Int64("inspectedBytes", int64(m.InspectedBytes)),
			attribute.Int64("backendReads", int64(m.BackendReads)),
			attribute.Int64("backendBytes", int64(m.BackendBytes)),
		)
		for k, v := range m.AdditionalMetrics {
			span.SetAttributes(attribute.Int64("additionalMetrics."+k, v))
		}
	default:
		span.AddEvent("unsupported querier span metrics type")
	}
}
