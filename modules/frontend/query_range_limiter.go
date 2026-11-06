package frontend

import (
	"cmp"
	"fmt"
	"math"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grafana/tempo/v3/tempodb"
	"github.com/grafana/tempo/v3/tempodb/backend"
)

const (
	queryRangeResultAllowed              = "allowed"
	queryRangeResultRejectedScannedBytes = "rejected_scanned_bytes"
	queryRangeResultRejectedMaxDuration  = "rejected_max_duration"
)

var (
	scannedBytesLimitBudgetRange = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:                       metricNamespace,
		Name:                            "query_frontend_scanned_bytes_limit_budget_range_seconds",
		Help:                            "Query range the scanned bytes budget allows, before min_duration and max_duration apply, for queries within max_duration that are over the budget.",
		Buckets:                         prometheus.ExponentialBuckets(time.Minute.Seconds(), 4, 10), // 1m to ~182d
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: 1 * time.Hour,
	}, []string{metricLabelTenant, metricLabelOp})

	requestQueryRange = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:                       metricNamespace,
		Name:                            "query_frontend_query_range_seconds",
		Help:                            "Requested range of search and metrics queries.",
		Buckets:                         prometheus.ExponentialBuckets(time.Minute.Seconds(), 4, 10), // 1m to ~182d
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: 1 * time.Hour,
	}, []string{metricLabelOp, metricLabelResult})

	scannedBytesLimitTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricNamespace,
		Name:      "query_frontend_scanned_bytes_limit_total",
		Help:      "Queries evaluated against the scanned bytes limit by result.",
	}, []string{metricLabelTenant, metricLabelOp, metricLabelResult})

	searchRangeLimitMetrics = rangeLimitMetrics{
		scannedBytes:      scannedBytesLimitTotal.MustCurryWith(prometheus.Labels{metricLabelOp: searchOp}),
		budgetRange:       scannedBytesLimitBudgetRange.MustCurryWith(prometheus.Labels{metricLabelOp: searchOp}),
		requestQueryRange: requestQueryRange.MustCurryWith(prometheus.Labels{metricLabelOp: searchOp}),
	}
	metricsRangeLimitMetrics = rangeLimitMetrics{
		scannedBytes:      scannedBytesLimitTotal.MustCurryWith(prometheus.Labels{metricLabelOp: metricsOp}),
		budgetRange:       scannedBytesLimitBudgetRange.MustCurryWith(prometheus.Labels{metricLabelOp: metricsOp}),
		requestQueryRange: requestQueryRange.MustCurryWith(prometheus.Labels{metricLabelOp: metricsOp}),
	}
)

// rangeLimitMetrics are a range limiter's metrics with op set.
type rangeLimitMetrics struct {
	scannedBytes      *prometheus.CounterVec
	budgetRange       prometheus.ObserverVec
	requestQueryRange prometheus.ObserverVec
}

// rangeLimiter checks a tenant's query range against its max_duration, min_duration and max_scanned_bytes.
// It has no side effects, so the caller records, logs and builds the error.
type rangeLimiter interface {
	TenantLimits(tenantID string) rangeLimits
	// CheckRange returns the result, and the estimate when the blocks were read, else nil.
	CheckRange(tenantID string, l rangeLimits, start, end time.Time) (string, *scannedBytesCheck)
}

// rangeLimits are a tenant's query range limits, with overrides applied.
type rangeLimits struct {
	minDuration     time.Duration
	maxDuration     time.Duration
	maxScannedBytes uint64
}

// scannedBytesLimit is what the search and metrics rangeLimiters share.
type scannedBytesLimit struct {
	op       string
	reader   tempodb.Reader
	filterFn func(m *backend.BlockMeta) bool
}

// checkTenantRange always rejects a range over max_duration, always allows one within min_duration,
// and checks the rest against max_scanned_bytes.
func (s scannedBytesLimit) checkTenantRange(tenantID string, l rangeLimits, start, end, backendAfter time.Time) (string, *scannedBytesCheck) {
	requested := end.Sub(start)
	if l.maxDuration != 0 && requested > l.maxDuration {
		if l.maxScannedBytes == 0 {
			return queryRangeResultRejectedMaxDuration, nil
		}
		// Still estimated, to show what max_scanned_bytes alone would allow.
		est := s.estimateBackendBytes(tenantID, l, start, end, backendAfter)
		return queryRangeResultRejectedMaxDuration, &est
	}
	if requested <= l.minDuration || l.maxScannedBytes == 0 {
		return queryRangeResultAllowed, nil
	}
	est := s.estimateBackendBytes(tenantID, l, start, end, backendAfter)
	if est.withinBudget {
		return queryRangeResultAllowed, &est
	}
	// Over budget always rejects, since this is only reached above min_duration and budgetRange is below requested.
	return queryRangeResultRejectedScannedBytes, &est
}

// recordRangeCheck counts a range check in the op's metrics. est is nil when the blocks were not read.
func recordRangeCheck(m rangeLimitMetrics, tenantID string, requested time.Duration, result string, est *scannedBytesCheck) {
	m.requestQueryRange.WithLabelValues(result).Observe(requested.Seconds())
	if est == nil {
		return
	}
	m.scannedBytes.WithLabelValues(tenantID, result).Inc()
	// Over-max checks are left out, so the histogram shows what the limit decides.
	if result == queryRangeResultRejectedScannedBytes {
		m.budgetRange.WithLabelValues(tenantID).Observe(est.budgetRange.Seconds())
	}
}

// scannedBytesErr is the error for a range over max_scanned_bytes.
func scannedBytesErr(op, tenantID string, requested, allowedRange time.Duration) error {
	// Rounding in opposite directions keeps the two values distinct. Under 1s it stays exact.
	allowed := cmp.Or(allowedRange.Truncate(time.Second), allowedRange)
	return fmt.Errorf("%s query time range of %s reads too much data, the allowed range for tenant %s is %s",
		op, ceilSeconds(requested), tenantID, allowed)
}

// scannedBytesCheck is a query range checked against max_scanned_bytes.
type scannedBytesCheck struct {
	estimatedBytes uint64
	// withinBudget is true when the blocks in the range total max_scanned_bytes or less.
	withinBudget bool

	// budgetRange is the range max_scanned_bytes allows, before min_duration and max_duration. Only set when !withinBudget.
	budgetRange time.Duration

	// allowedRange is budgetRange raised to min_duration and capped at max_duration, the range the error reports. Only set when !withinBudget.
	allowedRange time.Duration
}

// estimateBackendBytes counts the backend bytes in range. Blocks after backendAfter are left out, since only the live-stores serve that range.
func (s scannedBytesLimit) estimateBackendBytes(tenantID string, l rangeLimits, start, end, backendAfter time.Time) scannedBytesCheck {
	c := scannedBytesCheck{withinBudget: true}
	backendEnd := end
	if backendAfter.Before(end) {
		backendEnd = backendAfter
	}
	// TODO: live-store data after backendAfter isn't counted, since its size needs a new live-store RPC.
	if !backendEnd.After(start) {
		return c
	}

	// Newest first, so budgetRange is the longest range ending at end that fits.
	for _, m := range blockMetasForSearch(s.reader.BlockMetas(tenantID), start, backendEnd, s.filterFn) {
		c.estimatedBytes += m.Size_
		if c.withinBudget && c.estimatedBytes > l.maxScannedBytes {
			c.withinBudget = false
			// Inclusive overlap, so start 1ns after the block, never before backendAfter.
			c.budgetRange = max(end.Sub(m.EndTime)-time.Nanosecond, end.Sub(backendEnd))
		}
	}
	if c.withinBudget {
		return c
	}

	c.allowedRange = max(c.budgetRange, l.minDuration)
	if l.maxDuration != 0 {
		c.allowedRange = min(c.allowedRange, l.maxDuration)
	}
	return c
}

// ceilSeconds rounds d up to whole seconds, or down where rounding up would overflow.
func ceilSeconds(d time.Duration) time.Duration {
	t := d.Truncate(time.Second)
	if t < d && t <= math.MaxInt64-time.Second {
		t += time.Second
	}
	return t
}

// unixNanoTime converts a request timestamp without wrapping past math.MaxInt64.
func unixNanoTime(ns uint64) time.Time {
	return time.Unix(int64(ns/uint64(time.Second)), int64(ns%uint64(time.Second)))
}
