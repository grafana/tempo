package frontend

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grafana/tempo/v3/tempodb"
	"github.com/grafana/tempo/v3/tempodb/backend"
)

const (
	queryAllowed             = "allowed"
	queryRejectedBytes       = "rejected_scanned_bytes"
	queryRejectedMaxDuration = "rejected_max_duration"
)

var (
	queryRangeSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:                       metricNamespace,
		Name:                            "query_frontend_query_range_seconds",
		Help:                            "Requested range of search and metrics queries, by result.",
		Buckets:                         prometheus.ExponentialBuckets(time.Minute.Seconds(), 4, 10), // 1m to ~182d
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: 1 * time.Hour,
	}, []string{metricLabelOp, metricLabelResult})

	queryRangeEstimatedBytes = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:                       metricNamespace,
		Name:                            "query_frontend_query_range_estimated_bytes",
		Help:                            "Estimated backend bytes of search and metrics queries checked against max_scanned_bytes, by result.",
		Buckets:                         prometheus.ExponentialBuckets(1024*1024, 4, 12), // 1MiB to 4TiB
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: 1 * time.Hour,
	}, []string{metricLabelOp, metricLabelResult})

	queryRangeQueries = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricNamespace,
		Name:      "query_frontend_query_range_queries_total",
		Help:      "Search and metrics queries checked against the range limits, by result and whether their bytes were estimated.",
	}, []string{metricLabelOp, metricLabelResult, "bytes_estimated"})

	searchRangeLimitMetrics  = newRangeLimitMetrics(searchOp)
	metricsRangeLimitMetrics = newRangeLimitMetrics(metricsOp)
)

// rangeLimitMetrics are a range limiter's metrics with op set.
type rangeLimitMetrics struct {
	requestedRange prometheus.ObserverVec
	estimatedBytes prometheus.ObserverVec
	queries        *prometheus.CounterVec
}

func newRangeLimitMetrics(op string) rangeLimitMetrics {
	l := prometheus.Labels{metricLabelOp: op}
	return rangeLimitMetrics{
		requestedRange: queryRangeSeconds.MustCurryWith(l),
		estimatedBytes: queryRangeEstimatedBytes.MustCurryWith(l),
		queries:        queryRangeQueries.MustCurryWith(l),
	}
}

// rangeLimits are a tenant's query range limits, with overrides applied.
type rangeLimits struct {
	minDuration     time.Duration
	maxDuration     time.Duration
	maxScannedBytes uint64
}

// rangeLimiter checks a tenant's query range against its max_duration, min_duration and max_scanned_bytes.
type rangeLimiter interface {
	TenantLimits(tenantID string) rangeLimits
	// CheckRange returns the result, and the estimate when the blocks were read, else nil.
	CheckRange(tenantID string, l rangeLimits, start, end time.Time) (string, *checkResult)
}

// scannedBytesLimit is what the search and metrics rangeLimiters share.
type scannedBytesLimit struct {
	reader   tempodb.Reader
	filterFn func(m *backend.BlockMeta) bool
	metrics  rangeLimitMetrics
}

// checkTenantRange always rejects a range over max_duration, always allows one within min_duration,
// and checks the rest against max_scanned_bytes.
func (s scannedBytesLimit) checkTenantRange(tenantID string, l rangeLimits, start, end, backendAfter time.Time) (string, *checkResult) {
	result, est := s.decideTenantRange(tenantID, l, start, end, backendAfter)
	s.metrics.requestedRange.WithLabelValues(result).Observe(end.Sub(start).Seconds())

	s.metrics.queries.WithLabelValues(result, strconv.FormatBool(est != nil)).Inc()
	if est != nil {
		s.metrics.estimatedBytes.WithLabelValues(result).Observe(float64(est.estimatedBytes))
	}
	return result, est
}

func (s scannedBytesLimit) decideTenantRange(tenantID string, l rangeLimits, start, end, backendAfter time.Time) (string, *checkResult) {
	requested := end.Sub(start)

	if l.maxDuration != 0 && requested > l.maxDuration {
		if l.maxScannedBytes == 0 {
			return queryRejectedMaxDuration, nil
		}
		// Still estimate, to record the bytes for operators.
		return queryRejectedMaxDuration, &checkResult{estimatedBytes: s.estimateBackendBytes(s.reader.BlockMetas(tenantID), start, end, backendAfter)}
	}

	// under min_duration, always allowed
	if requested <= l.minDuration || l.maxScannedBytes == 0 {
		return queryAllowed, nil
	}

	// Read once, since each read copies the blocklist under its lock.
	metas := s.reader.BlockMetas(tenantID)
	est := checkResult{estimatedBytes: s.estimateBackendBytes(metas, start, end, backendAfter)}
	if est.estimatedBytes <= l.maxScannedBytes {
		return queryAllowed, &est
	}

	est.estimatedAllowedRange = s.estimateAllowedRange(metas, l, start, end, backendAfter)
	return queryRejectedBytes, &est
}

// checkResult is a tenant's estimate against max_scanned_bytes.
type checkResult struct {
	estimatedBytes uint64
	// estimatedAllowedRange is the range the error reports. Only set when the bytes check reject the query.
	estimatedAllowedRange time.Duration
}

// backendEnd caps end at backendAfter, since only the live-stores serve data after it.
func backendEnd(end, backendAfter time.Time) time.Time {
	if backendAfter.Before(end) {
		return backendAfter
	}
	return end
}

// estimateBackendBytes sums the backend blocks in range, without the sort in blockMetasForSearch, since most queries fit.
func (s scannedBytesLimit) estimateBackendBytes(metas []*backend.BlockMeta, start, end, backendAfter time.Time) uint64 {
	bEnd := backendEnd(end, backendAfter)
	// TODO: live-store data after backendAfter isn't counted, since its size needs a new live-store RPC.
	if !bEnd.After(start) {
		return 0
	}
	var bytes uint64
	for _, m := range metas {
		if !m.StartTime.After(bEnd) && !m.EndTime.Before(start) && s.filterFn(m) {
			bytes += m.Size_
		}
	}
	return bytes
}

// estimateAllowedRange is the longest range ending at end whose blocks fit max_scanned_bytes,
// raised to min_duration and capped at max_duration.
func (s scannedBytesLimit) estimateAllowedRange(metas []*backend.BlockMeta, l rangeLimits, start, end, backendAfter time.Time) time.Duration {
	bEnd := backendEnd(end, backendAfter)

	// The live-store range after backendAfter isn't counted, so it always fits.
	r := end.Sub(bEnd)
	var bytes uint64
	for _, m := range blockMetasForSearch(metas, start, bEnd, s.filterFn) {
		bytes += m.Size_
		if bytes > l.maxScannedBytes {
			// Inclusive overlap, so start 1ns after the block.
			r = max(r, end.Sub(m.EndTime)-time.Nanosecond)
			break
		}
	}
	// raise to min_duration
	r = max(r, l.minDuration)
	// and cap to max_duration
	if l.maxDuration != 0 {
		r = min(r, l.maxDuration)
	}

	return r
}
