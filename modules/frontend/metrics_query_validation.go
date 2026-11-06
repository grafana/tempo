package frontend

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-kit/log" //nolint:all deprecated
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/tenant"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/v3/modules/overrides"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/tempodb"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var (
	errEndMustBeGreaterThanStart = errors.New("end must be greater than start")
	errMetricsMaxDuration        = errors.New("metrics query time range exceeds the maximum allowed duration")
)

// validateQueryRangeReq must run before traceql.AlignRequest, which inflates
// the range by up to 3 steps and would cause valid requests to fail the cap.
func validateQueryRangeReq(ctx context.Context, cfg Config, rl rangeLimiter, logger log.Logger, req *tempopb.QueryRangeRequest) error {
	if req.Start >= req.End {
		return errEndMustBeGreaterThanStart
	}
	orgID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return err
	}
	if err := checkMetricsRange(ctx, cfg, rl, logger, orgID, unixNanoTime(req.Start), unixNanoTime(req.End)); err != nil {
		return err
	}
	if cfg.Metrics.MaxIntervals != 0 && (req.Step == 0 || (req.End-req.Start)/req.Step > cfg.Metrics.MaxIntervals) {
		minimumStep := (req.End - req.Start) / cfg.Metrics.MaxIntervals
		return fmt.Errorf(
			"step of %s is too small, minimum step for given range is %s",
			time.Duration(req.Step).String(),
			time.Duration(minimumStep).String(),
		)
	}
	return nil
}

// checkMetricsRange checks every tenant of the request against its own limits, and returns every rejection.
func checkMetricsRange(ctx context.Context, cfg Config, rl rangeLimiter, logger log.Logger, orgID string, start, end time.Time) error {
	// Runs before the multi-tenant split, so each tenant gets its own limits.
	tenants, err := tenant.TenantIDsFromOrgID(orgID)
	if err != nil {
		return err
	}

	// Without multi-tenant queries the tenant validator rejects a|b later, so there is no tenant to check here.
	if len(tenants) > 1 && !cfg.MultiTenantQueriesEnabled {
		return nil
	}

	var errs []error
	for _, tenantID := range tenants {
		spanCtx, span := tracer.Start(ctx, "frontend.CheckRange", trace.WithAttributes(attribute.String("tenant", tenantID)))
		// TODO: the sharder lists this tenant's blocks again. Passing these on would walk the blocklist once.
		limits := rl.TenantLimits(tenantID)
		result, est := rl.CheckRange(tenantID, limits, start, end)
		if est != nil {
			//nolint:goconst // logfmt keys match the other query log lines.
			kv := []any{
				"msg", "scanned bytes limit",
				"tenant", tenantID,
				"op", metricsOp,
				"max_duration", int64(limits.maxDuration / time.Second),
				"min_duration", int64(limits.minDuration / time.Second),
				"max_scanned_bytes", limits.maxScannedBytes,
				// Rounded like the rejection error: requested up, limits down.
				"requested_seconds", int64(ceilSeconds(end.Sub(start)) / time.Second),
				"estimated_scanned_bytes", est.estimatedBytes,
				"estimated_range_seconds", int64(est.estimatedAllowedRange / time.Second),
				"result", result,
			}
			logWithShape(level.Info(logger), ctx, kv...)
			setSpanAttrsWithShape(spanCtx, kv...)
		}
		span.End()

		switch result {
		case queryRejectedMaxDuration:
			errs = append(errs, fmt.Errorf("%w of %s", errMetricsMaxDuration, limits.maxDuration))
		case queryRejectedBytes:
			// Rounding in opposite directions keeps the two values distinct. Under 1s it stays exact.
			allowed := cmp.Or(est.estimatedAllowedRange.Truncate(time.Second), est.estimatedAllowedRange)
			errs = append(errs, fmt.Errorf("metrics query time range of %s reads too much data, the allowed range for tenant %s is %s",
				ceilSeconds(end.Sub(start)), tenantID, allowed))
		}
	}
	return errors.Join(errs...)
}

// metricsRangeLimiter checks metrics queries in the handlers, before traceql.AlignRequest widens the range.
type metricsRangeLimiter struct {
	cfg       Config
	overrides overrides.Interface

	scannedBytesLimit
}

var _ rangeLimiter = (*metricsRangeLimiter)(nil)

func newMetricsRangeLimiter(cfg Config, o overrides.Interface, reader tempodb.Reader) *metricsRangeLimiter {
	return &metricsRangeLimiter{
		cfg:       cfg,
		overrides: o,

		op:       metricsOp,
		reader:   reader,
		filterFn: isRF1Block,

		metrics: metricsRangeLimitMetrics,
	}
}

func (l *metricsRangeLimiter) CheckRange(tenantID string, limits rangeLimits, start, end time.Time) (string, *checkResult) {
	// NOTE - AlignRequest, in the handler and again in the sharder, adds up to 3 steps before start and 1 after end,
	// so a large step can exceed max_scanned_bytes, but that's by design.
	backendAfter := time.Now().Add(-l.cfg.Metrics.Sharder.QueryBackendAfter)
	return l.checkTenantRange(tenantID, limits, start, end, backendAfter)
}

// TenantLimits returns the metrics range limits for this tenant. A non-zero override wins over the config.
func (l *metricsRangeLimiter) TenantLimits(tenantID string) rangeLimits {
	return rangeLimits{
		minDuration:     cmp.Or(l.overrides.MinMetricsDuration(tenantID), l.cfg.Metrics.Sharder.MinDuration),
		maxDuration:     cmp.Or(l.overrides.MaxMetricsDuration(tenantID), l.cfg.Metrics.Sharder.MaxDuration),
		maxScannedBytes: cmp.Or(l.overrides.MaxMetricsScannedBytes(tenantID), l.cfg.Metrics.Sharder.MaxScannedBytes),
	}
}
