package frontend

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/grafana/dskit/tenant"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
)

func validateQueryRangeReq(ctx context.Context, cfg Config, o overrides.Interface, req *tempopb.QueryRangeRequest) error {
	if req.Start > req.End {
		return errors.New("end must be greater than start")
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
	tenants, err := tenant.TenantIDs(ctx)
	if err != nil {
		return err
	}

	rawDuration := time.Duration(req.End - req.Start)
	var maxDuration time.Duration

	for _, tenantID := range tenants {
		tenantMaxDuration := defaultMaxDuration
		if o != nil {
			if overrideMaxDuration := o.MaxMetricsDuration(tenantID); overrideMaxDuration != 0 {
				tenantMaxDuration = overrideMaxDuration
			}
		}

		if tenantMaxDuration == 0 {
			continue
		}
		if maxDuration == 0 || tenantMaxDuration < maxDuration {
			maxDuration = tenantMaxDuration
		}
	}

	if maxDuration != 0 && rawDuration > maxDuration {
		return fmt.Errorf("metrics query time range exceeds the maximum allowed duration of %s", maxDuration)
	}

	return nil
}
