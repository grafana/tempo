package frontend

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-kit/log" //nolint:all deprecated
	"github.com/grafana/dskit/tenant"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/v3/modules/overrides"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/tempodb"
)

var (
	errEndMustBeGreaterThanStart = errors.New("end must be greater than start")
	errMetricsMaxDuration        = errors.New("metrics query time range exceeds the maximum allowed duration")
)

// validateQueryRangeReq must run before traceql.AlignRequest, which inflates
// the range by up to 3 steps and would cause valid requests to fail the cap.
func validateQueryRangeReq(ctx context.Context, cfg Config, rl rangeLimiter, req *tempopb.QueryRangeRequest) error {
	if req.Start >= req.End {
		return errEndMustBeGreaterThanStart
	}
	orgID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return err
	}
	if err := rl.Limit(ctx, orgID, unixNanoTime(req.Start), unixNanoTime(req.End)); err != nil {
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

// metricsRangeLimiter checks metrics queries in the handlers, before traceql.AlignRequest widens the range.
type metricsRangeLimiter struct {
	scannedBytesLimit
	cfg       Config
	overrides overrides.Interface
}

var _ rangeLimiter = (*metricsRangeLimiter)(nil)

func newMetricsRangeLimiter(cfg Config, o overrides.Interface, reader tempodb.Reader, logger log.Logger) *metricsRangeLimiter {
	return &metricsRangeLimiter{
		scannedBytesLimit: scannedBytesLimit{
			op:       metricsOp,
			kind:     "metrics query",
			reader:   reader,
			filterFn: isRF1Block,
			maxErr: func(l rangeLimits, _, _ time.Time) error {
				return fmt.Errorf("%w of %s", errMetricsMaxDuration, l.maxDuration)
			},
			logger: logger,
		},
		cfg:       cfg,
		overrides: o,
	}
}

// Limit checks every tenant of the request against its own limits, and returns the rejection with the shortest retry range.
func (l *metricsRangeLimiter) Limit(ctx context.Context, orgID string, start, end time.Time) error {
	// Runs before the multi-tenant split, so each tenant gets its own limits.
	tenants, err := tenant.TenantIDsFromOrgID(orgID)
	if err != nil {
		return err
	}
	// Without multi-tenant queries the tenant validator rejects a|b later, so there is no tenant to check here.
	if len(tenants) > 1 && !l.cfg.MultiTenantQueriesEnabled {
		return nil
	}
	// NOTE - alignment scans up to 4 steps past this range, so a large step can exceed the budget, as with max_duration.
	// NOTE - the sharder reads the clock again, so its cutoff is later by the handler-to-sharder latency.
	cutoff := time.Now().Add(-l.cfg.Metrics.Sharder.QueryBackendAfter)

	var (
		rejection error
		retry     time.Duration
		decided   *scannedBytesCheck
	)
	for _, tenantID := range tenants {
		limits := l.rangeLimits(tenantID)
		// TODO: the sharder lists this tenant's blocks again. Passing these on would walk the blocklist once.
		c, err := l.limit(ctx, tenantID, limits, start, end, cutoff)
		if err != nil {
			r := limits.maxDuration
			if c != nil && c.result == scannedBytesResultRejected {
				r = c.allowed
			}
			// Only the shortest retry range passes every tenant.
			if rejection == nil || r < retry {
				rejection, retry, decided = err, r, c
			}
			continue
		}
		// Tenants share span keys, so the span shows only the check behind the final result.
		if rejection == nil && decided == nil {
			decided = c
		}
	}
	if decided != nil {
		setSpanAttrsWithShape(ctx, decided.fields()...)
	}
	return rejection
}

// rangeLimits returns the metrics range limits for this tenant. A non-zero override wins over the config.
func (l *metricsRangeLimiter) rangeLimits(tenantID string) rangeLimits {
	return rangeLimits{
		minDuration:     cmp.Or(l.overrides.MinMetricsDuration(tenantID), l.cfg.Metrics.Sharder.MinDuration),
		maxDuration:     cmp.Or(l.overrides.MaxMetricsDuration(tenantID), l.cfg.Metrics.Sharder.MaxDuration),
		maxScannedBytes: cmp.Or(l.overrides.MaxMetricsScannedBytes(tenantID), l.cfg.Metrics.Sharder.MaxScannedBytes),
	}
}
