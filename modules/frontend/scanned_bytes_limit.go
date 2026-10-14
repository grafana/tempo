package frontend

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/go-kit/log" //nolint:all deprecated
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grafana/tempo/v3/tempodb/backend"
)

const (
	scannedBytesResultWithin      = "within"
	scannedBytesResultRejected    = "rejected"
	scannedBytesResultRaisedToMin = "raised_to_min"
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

// rangeLimits are one tenant's range limits for one query type, with overrides applied.
type rangeLimits struct {
	minDuration     time.Duration
	maxDuration     time.Duration
	maxScannedBytes uint64
}

// scannedBytesLimit is what differs between the search and metrics scanned bytes limits.
type scannedBytesLimit interface {
	// op labels the metrics and the log line.
	op() string
	limits(tenantID string) rangeLimits
	// estimatedBytes estimates the bytes a query scans in a block. Returning 0 leaves the block out.
	estimatedBytes(m *backend.BlockMeta) uint64
}

// rangeCheck is one tenant's range checked against its rangeLimits.
type rangeCheck struct {
	rangeLimits
	op, tenantID string
	requested    time.Duration
	result       string
	// evaluated is false when nothing is recorded: the limit is off, max_duration is negative, or there is no range.
	evaluated bool
	// fits is true when the blocks in the requested range are within max_scanned_bytes.
	fits bool
	// derived is the range max_scanned_bytes allows, before min_duration and max_duration. Only set when !fits.
	derived        time.Duration
	allowed        time.Duration
	estimatedBytes uint64
}

// checkScannedBytes allows the longest range ending at end whose blocks fit max_scanned_bytes, raised to
// min_duration and capped at max_duration.
func checkScannedBytes(l scannedBytesLimit, tenantID string, start, end, cutoff time.Time, metas func() []*backend.BlockMeta) rangeCheck {
	c := rangeCheck{
		rangeLimits: l.limits(tenantID),
		op:          l.op(),
		tenantID:    tenantID,
		requested:   end.Sub(start),
		result:      scannedBytesResultWithin,
		fits:        true,
	}
	overMax := c.maxDuration != 0 && c.requested > c.maxDuration
	if overMax {
		c.result = scannedBytesResultRejectedMaxDuration
	}
	// A negative max_duration rejects everything and would pollute the log. A zero start has no backend range.
	if c.maxScannedBytes == 0 || c.maxDuration < 0 || start.UnixNano() <= 0 || !end.After(start) {
		return c
	}
	c.evaluated = true

	c.derived, c.estimatedBytes, c.fits = rangeWithinBudget(metas, l.estimatedBytes, start, end, cutoff, c.maxScannedBytes)
	if c.fits {
		c.allowed = c.maxDuration
		return c
	}
	c.allowed = max(c.derived, c.minDuration)
	if c.maxDuration > 0 {
		c.allowed = min(c.allowed, c.maxDuration)
	}
	if overMax {
		return c
	}
	switch {
	case c.requested > c.allowed:
		c.result = scannedBytesResultRejected
	case c.allowed > c.derived:
		c.result = scannedBytesResultRaisedToMin
	}
	return c
}

// err is the rejection error for a query of kind, such as "metrics query" or "search".
func (c rangeCheck) err(kind string) error {
	if c.result != scannedBytesResultRejected {
		return nil
	}
	// Rounding in opposite directions keeps the two values distinct and the allowed one safe to retry.
	return fmt.Errorf("%s time range of %s exceeds the maximum allowed range of %s for tenant %s (limited by scanned bytes)",
		kind, ceilSeconds(c.requested), c.allowed.Truncate(time.Second), c.tenantID)
}

// ceilSeconds rounds d up to whole seconds, or down where rounding up would overflow.
func ceilSeconds(d time.Duration) time.Duration {
	t := d.Truncate(time.Second)
	if t < d && t <= math.MaxInt64-time.Second {
		t += time.Second
	}
	return t
}

// recordScannedBytesLimit counts an evaluated check and logs it like the query response, with the fields mirrored on the span.
func recordScannedBytesLimit(ctx context.Context, logger log.Logger, c rangeCheck) {
	if !c.evaluated {
		return
	}
	scannedBytesLimitTotal.WithLabelValues(c.tenantID, c.op, c.result).Inc()
	// Over-max queries are left out, so the histogram reflects what the limit decides today.
	// The raw range carries the calibration signal, since min_duration and max_duration can pin allowed.
	if !c.fits && c.result != scannedBytesResultRejectedMaxDuration {
		scannedBytesLimitDerivedRange.WithLabelValues(c.tenantID, c.op).Observe(c.derived.Seconds())
	}

	//nolint:goconst // logfmt keys match the other query log lines.
	fields := []any{
		"msg", "scanned bytes limit",
		"tenant", c.tenantID,
		"op", c.op,
		// Rounded like the rejection error: requested up, limits down.
		"requested_seconds", int64(ceilSeconds(c.requested) / time.Second),
		"estimated_scanned_bytes", c.estimatedBytes,
	}
	if c.fits {
		fields = append(fields, "fits", true)
	} else {
		fields = append(fields, "derived_seconds", int64(c.derived/time.Second))
	}
	fields = append(fields,
		"min_seconds", int64(c.minDuration/time.Second),
		"max_seconds", int64(c.maxDuration/time.Second),
		"result", c.result,
		// Whether max_scanned_bytes and min_duration alone allow the query, to size a max_duration raise.
		"passes_scanned_bytes", c.fits || c.requested <= max(c.derived, c.minDuration),
	)
	recordResult(level.Info(logger), ctx, nil, fields...)
}

// rangeWithinBudget returns the longest range ending at end whose blocks fit maxScannedBytes, adding blocks newest
// first, and the estimated bytes of the whole range. fits is true when every block in the range fits.
func rangeWithinBudget(metas func() []*backend.BlockMeta, estimate func(m *backend.BlockMeta) uint64, start, end, cutoff time.Time, maxScannedBytes uint64) (derived time.Duration, scanned uint64, fits bool) {
	// The range after the cutoff reaches no backend block, so it is free.
	backendEnd := end
	if backendEnd.After(cutoff) {
		backendEnd = cutoff
	}
	if !backendEnd.After(start) {
		return 0, 0, true
	}
	size := func(m *backend.BlockMeta) uint64 {
		if m.StartTime.After(backendEnd) || m.EndTime.Before(start) {
			return 0
		}
		return estimate(m)
	}
	all := metas()

	// Sum first so queries within budget pay no allocation or sort.
	n := 0
	for _, m := range all {
		if b := size(m); b > 0 {
			scanned += b
			n++
		}
	}
	if scanned <= maxScannedBytes {
		return 0, scanned, true
	}

	// Value pairs keep the sort in cache instead of chasing meta pointers.
	type block struct {
		end  int64
		size uint64
	}
	blocks := make([]block, 0, n)
	for _, m := range all {
		if b := size(m); b > 0 {
			blocks = append(blocks, block{end: m.EndTime.UnixNano(), size: b})
		}
	}
	slices.SortFunc(blocks, func(a, b block) int {
		return cmp.Compare(b.end, a.end)
	})

	var running uint64
	for _, b := range blocks {
		running += b.size
		if running > maxScannedBytes {
			// Overlap is inclusive, so the range must start 1ns after the block's end.
			return max(end.Sub(time.Unix(0, b.end))-time.Nanosecond, end.Sub(backendEnd), 0), scanned, false
		}
	}
	return 0, scanned, true
}

// unixNanoTime converts a request timestamp without wrapping past math.MaxInt64.
func unixNanoTime(ns uint64) time.Time {
	return time.Unix(0, int64(min(ns, math.MaxInt64)))
}
