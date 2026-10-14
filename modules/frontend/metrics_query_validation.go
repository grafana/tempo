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
	"github.com/grafana/tempo/v3/tempodb/backend"
)

var (
	errEndMustBeGreaterThanStart = errors.New("end must be greater than start")
	errMetricsMaxDuration        = errors.New("metrics query time range exceeds the maximum allowed duration")
)

// validateQueryRangeReq must run before traceql.AlignRequest, which inflates
// the range by up to ~3*step and would cause valid requests to fail the cap.
func validateQueryRangeReq(ctx context.Context, cfg Config, o overrides.Interface, sl *metricsScannedBytesLimit, req *tempopb.QueryRangeRequest) error {
	if req.Start >= req.End {
		return errEndMustBeGreaterThanStart
	}
	// The scanned bytes check still logs a request max_duration rejects, to show what max_scanned_bytes alone would allow.
	maxErr := validateMetricsQueryMaxDuration(ctx, o, cfg.Metrics.Sharder.MaxDuration, req)
	if maxErr == nil && cfg.Metrics.MaxIntervals != 0 && (req.Step == 0 || (req.End-req.Start)/req.Step > cfg.Metrics.MaxIntervals) {
		minimumStep := (req.End - req.Start) / cfg.Metrics.MaxIntervals
		return fmt.Errorf(
			"step of %s is too small, minimum step for given range is %s",
			time.Duration(req.Step).String(),
			time.Duration(minimumStep).String(),
		)
	}
	// Last, so cheaper checks reject bad requests before the blocklist walk.
	return sl.check(ctx, req, maxErr)
}

func validateMetricsQueryMaxDuration(ctx context.Context, o overrides.Interface, defaultMaxDuration time.Duration, req *tempopb.QueryRangeRequest) error {
	tenantID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return err
	}
	maxDuration := metricsMaxDuration(o, tenantID, defaultMaxDuration)
	if maxDuration == 0 {
		return nil
	}
	// Negative cap: fail closed (matches prior sharder behavior).
	// Unsigned compare: time.Duration is int64; a range > math.MaxInt64 ns
	// would otherwise wrap negative and silently bypass the cap.
	if maxDuration < 0 || req.End-req.Start > uint64(maxDuration) {
		return fmt.Errorf("%w of %s", errMetricsMaxDuration, maxDuration)
	}
	return nil
}

// metricsMaxDuration resolves the metrics max_duration. The max_duration and scanned bytes checks share it to stay in sync.
func metricsMaxDuration(o overrides.Interface, tenantID string, defaultMaxDuration time.Duration) time.Duration {
	if d := o.MaxMetricsDuration(tenantID); d != 0 {
		return d
	}
	return defaultMaxDuration
}

// metricsScannedBytesLimit checks metrics queries in the handlers, before traceql.AlignRequest widens the range.
type metricsScannedBytesLimit struct {
	cfg       QueryRangeSharderConfig
	overrides overrides.Interface
	reader    tempodb.Reader
	logger    log.Logger
	// multiTenantQueriesEnabled is multi_tenant_queries_enabled. Only then does a|b name several tenants.
	multiTenantQueriesEnabled bool
}

func newMetricsScannedBytesLimit(reader tempodb.Reader, o overrides.Interface, cfg QueryRangeSharderConfig, multiTenantQueriesEnabled bool, logger log.Logger) *metricsScannedBytesLimit {
	return &metricsScannedBytesLimit{cfg: cfg, overrides: o, reader: reader, logger: logger, multiTenantQueriesEnabled: multiTenantQueriesEnabled}
}

func (*metricsScannedBytesLimit) op() string { return metricsOp }

func (l *metricsScannedBytesLimit) limits(tenantID string) rangeLimits {
	return rangeLimits{
		minDuration:     cmp.Or(l.overrides.MinMetricsDuration(tenantID), l.cfg.MinDuration),
		maxDuration:     metricsMaxDuration(l.overrides, tenantID, l.cfg.MaxDuration),
		maxScannedBytes: cmp.Or(l.overrides.MaxMetricsScannedBytes(tenantID), l.cfg.MaxScannedBytes),
	}
}

// estimatedBytes counts only RF1 blocks, the only ones metrics queries read.
func (*metricsScannedBytesLimit) estimatedBytes(m *backend.BlockMeta) uint64 {
	if !isMetricsGeneratorBlock(m) {
		return 0
	}
	return m.Size_
}

// check records every tenant of the request, then returns maxErr or the first tenant's rejection.
// maxErr is validateMetricsQueryMaxDuration's verdict on the raw org ID.
func (l *metricsScannedBytesLimit) check(ctx context.Context, req *tempopb.QueryRangeRequest, maxErr error) error {
	// Runs before the multi-tenant split, so each tenant gets its own limits.
	tenants, err := tenant.TenantIDs(ctx)
	if err != nil {
		return cmp.Or(maxErr, err)
	}
	// Without multi-tenant queries the tenant validator rejects a|b later, so there is no tenant to check here.
	if len(tenants) > 1 && !l.multiTenantQueriesEnabled {
		return maxErr
	}
	start, end := unixNanoTime(req.Start), unixNanoTime(req.End)
	cutoff := time.Now().Add(-l.cfg.QueryBackendAfter)

	var rejection error
	for _, tenantID := range tenants {
		// TODO: the sharder lists this tenant's blocks again. Passing these on would walk the blocklist once.
		c := checkScannedBytes(l, tenantID, start, end, cutoff, func() []*backend.BlockMeta { return l.reader.BlockMetas(tenantID) })
		if maxErr != nil {
			// The raw org ID's max_duration rejected the request, even if this tenant's own allows it.
			c.result = scannedBytesResultRejectedMaxDuration
		}
		recordScannedBytesLimit(ctx, l.logger, c)
		if rejection == nil {
			rejection = metricsRangeErr(c)
		}
	}
	return cmp.Or(maxErr, rejection)
}

// metricsRangeErr is c's error for a metrics query.
func metricsRangeErr(c rangeCheck) error {
	switch {
	// With the limit off, validateMetricsQueryMaxDuration stays the only max_duration check.
	case c.maxScannedBytes == 0:
		return nil
	case c.result == scannedBytesResultRejectedMaxDuration:
		return fmt.Errorf("%w of %s", errMetricsMaxDuration, c.maxDuration)
	default:
		return c.err("metrics query")
	}
}
