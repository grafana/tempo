package frontend

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"time"

	"github.com/go-kit/log" //nolint:all deprecated
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grafana/tempo/v3/tempodb"
	"github.com/grafana/tempo/v3/tempodb/backend"
)

const (
	scannedBytesResultWithin   = "within"
	scannedBytesResultRejected = "rejected"
	// max_duration rejected the query. Still logged, to show what max_scanned_bytes alone would allow.
	scannedBytesResultRejectedMaxDuration = "rejected_max_duration"
)

var (
	scannedBytesLimitDerivedRange = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metricNamespace,
		Name:      "query_frontend_scanned_bytes_limit_derived_range_seconds",
		Help:      "Query range the scanned bytes budget allows, before min_duration and max_duration apply, for queries within max_duration that are over the budget.",
		Buckets: []float64{
			time.Hour.Seconds(),
			(6 * time.Hour).Seconds(),
			(24 * time.Hour).Seconds(),
			(2 * 24 * time.Hour).Seconds(),
			(3 * 24 * time.Hour).Seconds(),
			(5 * 24 * time.Hour).Seconds(),
			(7 * 24 * time.Hour).Seconds(),
			(14 * 24 * time.Hour).Seconds(),
			(30 * 24 * time.Hour).Seconds(),
			(60 * 24 * time.Hour).Seconds(),
			(90 * 24 * time.Hour).Seconds(),
		},
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: 1 * time.Hour,
	}, []string{metricLabelTenant, metricLabelOp})

	scannedBytesLimitTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricNamespace,
		Name:      "query_frontend_scanned_bytes_limit_total",
		Help:      "Queries evaluated against the scanned bytes limit by result.",
	}, []string{metricLabelTenant, metricLabelOp, metricLabelResult})
)

// rangeLimiter checks a query range against the tenant's max_duration, min_duration and max_scanned_bytes.
type rangeLimiter interface {
	Limit(ctx context.Context, orgID string, start, end time.Time) error
}

// rangeLimits are a tenant's query range limits, with overrides applied.
type rangeLimits struct {
	minDuration     time.Duration
	maxDuration     time.Duration
	maxScannedBytes uint64
}

// allowed raises derived to min_duration and caps it at max_duration.
func (l rangeLimits) allowed(derived time.Duration) time.Duration {
	allowed := max(derived, l.minDuration)
	if l.maxDuration != 0 {
		allowed = min(allowed, l.maxDuration)
	}
	return allowed
}

// scannedBytesLimit is what the search and metrics rangeLimiters share.
type scannedBytesLimit struct {
	op string
	// kind names the query in the scanned bytes error, such as "metrics query" or "search".
	kind     string
	reader   tempodb.Reader
	filterFn func(m *backend.BlockMeta) bool
	// maxErr keeps each op's upstream max_duration error.
	maxErr func(l rangeLimits, start, end time.Time) error
	logger log.Logger
}

// limit rejects a range over max_duration, allows one within min_duration, and checks the rest against max_scanned_bytes.
// The caller sets the span, since only it knows which check decides the query.
func (s scannedBytesLimit) limit(ctx context.Context, tenantID string, l rangeLimits, start, end, cutoff time.Time) (*scannedBytesCheck, error) {
	requested := end.Sub(start)
	if l.maxDuration != 0 && requested > l.maxDuration {
		if l.maxScannedBytes == 0 {
			return nil, s.maxErr(l, start, end)
		}
		c := s.check(tenantID, l, start, end, cutoff)
		c.result = scannedBytesResultRejectedMaxDuration
		recordScannedBytesLimit(ctx, s.logger, c)
		return &c, s.maxErr(l, start, end)
	}
	if requested <= l.minDuration || l.maxScannedBytes == 0 {
		return nil, nil
	}

	c := s.check(tenantID, l, start, end, cutoff)
	recordScannedBytesLimit(ctx, s.logger, c)
	return &c, c.err(s.kind)
}

// scannedBytesCheck is a query range checked against max_scanned_bytes.
type scannedBytesCheck struct {
	rangeLimits
	op, tenantID   string
	requested      time.Duration
	estimatedBytes uint64
	// fits is true when the blocks in the range total max_scanned_bytes or less.
	fits bool
	// derived is the range max_scanned_bytes allows, before min_duration and max_duration. Only set when !fits.
	derived time.Duration
	allowed time.Duration
	result  string
}

// check counts the backend bytes in range. Blocks after cutoff are left out, since only the live-stores serve that range.
func (s scannedBytesLimit) check(tenantID string, l rangeLimits, start, end, cutoff time.Time) scannedBytesCheck {
	c := scannedBytesCheck{
		rangeLimits: l,
		op:          s.op,
		tenantID:    tenantID,
		requested:   end.Sub(start),
		fits:        true,
		result:      scannedBytesResultWithin,
	}
	backendEnd := end
	if cutoff.Before(end) {
		backendEnd = cutoff
	}
	// TODO: live-store data after the cutoff isn't counted, since its size needs a new live-store RPC.
	if !backendEnd.After(start) {
		return c
	}

	metas := s.reader.BlockMetas(tenantID)
	c.estimatedBytes = backendBytes(metas, s.filterFn, start, backendEnd)
	if c.estimatedBytes <= l.maxScannedBytes {
		return c
	}
	c.fits = false
	c.derived = derivedRange(metas, s.filterFn, l.maxScannedBytes, start, end, backendEnd)
	c.allowed = l.allowed(c.derived)
	// Over budget always rejects, since limit only gets here above min_duration and derived is below requested.
	c.result = scannedBytesResultRejected
	return c
}

// backendBytes sums the blocks overlapping [start, backendEnd], without the sort in blockMetasForSearch.
func backendBytes(metas []*backend.BlockMeta, filterFn func(m *backend.BlockMeta) bool, start, backendEnd time.Time) uint64 {
	var bytes uint64
	for _, m := range metas {
		if !m.StartTime.After(backendEnd) && !m.EndTime.Before(start) && filterFn(m) {
			bytes += m.Size_
		}
	}
	return bytes
}

// derivedRange is the longest range ending at end whose newest blocks fit budget.
func derivedRange(metas []*backend.BlockMeta, filterFn func(m *backend.BlockMeta) bool, budget uint64, start, end, backendEnd time.Time) time.Duration {
	var running uint64
	for _, m := range blockMetasForSearch(metas, start, backendEnd, filterFn) {
		running += m.Size_
		if running > budget {
			// Inclusive overlap, so start 1ns after the block, never before the cutoff.
			return max(end.Sub(m.EndTime)-time.Nanosecond, end.Sub(backendEnd))
		}
	}
	return 0
}

// err is the rejection error for a query of kind, such as "metrics query" or "search".
func (c scannedBytesCheck) err(kind string) error {
	if c.result != scannedBytesResultRejected {
		return nil
	}
	// No shorter range ending at the same end passes, so a "reduce to 0s" hint would mislead.
	if c.allowed == 0 {
		return fmt.Errorf("%s time range of %s exceeds the maximum allowed range for tenant %s (limited by scanned bytes), the newest data in the range is already over the limit",
			kind, ceilSeconds(c.requested), c.tenantID)
	}
	// Rounding in opposite directions keeps the two values distinct and the allowed one safe to retry. Under 1s it stays exact.
	allowed := cmp.Or(c.allowed.Truncate(time.Second), c.allowed)
	return fmt.Errorf("%s time range of %s exceeds the maximum allowed range of %s for tenant %s (limited by scanned bytes), reduce the time range to %s or less",
		kind, ceilSeconds(c.requested), allowed, c.tenantID, allowed)
}

// ceilSeconds rounds d up to whole seconds, or down where rounding up would overflow.
func ceilSeconds(d time.Duration) time.Duration {
	t := d.Truncate(time.Second)
	if t < d && t <= math.MaxInt64-time.Second {
		t += time.Second
	}
	return t
}

// recordScannedBytesLimit counts a check and logs it like the query response.
func recordScannedBytesLimit(ctx context.Context, logger log.Logger, c scannedBytesCheck) {
	scannedBytesLimitTotal.WithLabelValues(c.tenantID, c.op, c.result).Inc()
	// Over-max checks are left out, so the histogram shows what the limit decides.
	if !c.fits && c.result != scannedBytesResultRejectedMaxDuration {
		scannedBytesLimitDerivedRange.WithLabelValues(c.tenantID, c.op).Observe(c.derived.Seconds())
	}

	logWithShape(level.Info(logger), ctx, c.fields()...)
}

// fields are the check's log fields, which the span mirrors.
func (c scannedBytesCheck) fields() []any {
	//nolint:goconst // logfmt keys match the other query log lines.
	fields := []any{
		"msg", "scanned bytes limit",
		"tenant", c.tenantID,
		"op", c.op,
		// Rounded like the rejection error: requested up, limits down.
		"requested_seconds", int64(ceilSeconds(c.requested) / time.Second),
		"estimated_scanned_bytes", c.estimatedBytes,
		"max_scanned_bytes", c.maxScannedBytes,
	}
	if c.fits {
		fields = append(fields, "fits", true)
	} else {
		fields = append(fields, "derived_seconds", int64(c.derived/time.Second), "allowed_seconds", int64(c.allowed/time.Second))
	}
	fields = append(fields,
		"min_seconds", int64(c.minDuration/time.Second),
		"max_seconds", int64(c.maxDuration/time.Second),
		"result", c.result,
		// Whether max_scanned_bytes and min_duration alone allow the query, to size a max_duration raise.
		"passes_scanned_bytes", c.fits || c.requested <= max(c.derived, c.minDuration),
	)
	return fields
}

// unixNanoTime converts a request timestamp without wrapping past math.MaxInt64.
func unixNanoTime(ns uint64) time.Time {
	return time.Unix(0, int64(min(ns, math.MaxInt64)))
}
