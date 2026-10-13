package frontend

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/v3/modules/overrides"
	"github.com/grafana/tempo/v3/pkg/tempopb"
)

var errEndMustBeGreaterThanStart = errors.New("end must be greater than start")

// validateQueryRangeReq must run before traceql.AlignRequest, which inflates
// the range by up to ~3*step and would cause valid requests to fail the cap.
func validateQueryRangeReq(ctx context.Context, cfg Config, o overrides.Interface, sl *scannedBytesLimiter, req *tempopb.QueryRangeRequest) error {
	if req.Start >= req.End {
		return errEndMustBeGreaterThanStart
	}
	if err := validateMetricsQueryMaxDuration(ctx, o, cfg.Metrics.Sharder.MaxDuration, req); err != nil {
		// The max_duration error stands. The scanned bytes check only feeds the what-if log.
		sl.recordRejectedMaxDuration(ctx, req)
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
	// Last, so cheaper checks reject bad requests before the blocklist walk.
	return sl.check(ctx, req)
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
		return errMetricsMaxDuration(maxDuration)
	}
	return nil
}

func errMetricsMaxDuration(maxDuration time.Duration) error {
	return fmt.Errorf("metrics query time range exceeds the maximum allowed duration of %s", maxDuration)
}

// metricsMaxDuration resolves the metrics ceiling. The max_duration and scanned bytes checks share it to stay in sync.
func metricsMaxDuration(o overrides.Interface, tenantID string, defaultMaxDuration time.Duration) time.Duration {
	if d := o.MaxMetricsDuration(tenantID); d != 0 {
		return d
	}
	return defaultMaxDuration
}
