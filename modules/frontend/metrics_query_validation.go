package frontend

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
)

var errEndMustBeGreaterThanStart = errors.New("end must be greater than start")

// validateQueryRangeReq must run before traceql.AlignRequest, which inflates
// the range by up to ~3*step and would cause valid requests to fail the cap.
func validateQueryRangeReq(ctx context.Context, cfg Config, o overrides.Interface, req *tempopb.QueryRangeRequest) error {
	if req.Start >= req.End {
		return errEndMustBeGreaterThanStart
	}
	if err := validateMetricsQueryMaxDuration(ctx, o, cfg.Metrics.Sharder.MaxDuration, req); err != nil {
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

func validateMetricsQueryMaxDuration(ctx context.Context, o overrides.Interface, defaultMaxDuration time.Duration, req *tempopb.QueryRangeRequest) error {
	tenantID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return err
	}
	maxDuration := defaultMaxDuration
	if override := o.MaxMetricsDuration(tenantID); override != 0 {
		maxDuration = override
	}
	if maxDuration == 0 {
		return nil
	}
	// Negative cap: fail closed (matches prior sharder behavior).
	// Unsigned compare: time.Duration is int64; a range > math.MaxInt64 ns
	// would otherwise wrap negative and silently bypass the cap.
	if maxDuration < 0 || req.End-req.Start > uint64(maxDuration) {
		return fmt.Errorf("metrics query time range exceeds the maximum allowed duration of %s", maxDuration)
	}
	return nil
}
