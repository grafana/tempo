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
	"github.com/grafana/dskit/tenant"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/grafana/tempo/v3/modules/overrides"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/tempodb"
	"github.com/grafana/tempo/v3/tempodb/backend"
)

const (
	metricLabelOutcome = "outcome"

	scannedBytesOutcomeWithin      = "within"
	scannedBytesOutcomeRejected    = "rejected"
	scannedBytesOutcomeRaisedToMin = "raised_to_min"
	// scannedBytesOutcomeRejectedMaxDuration marks queries over max_duration, evaluated only for the what-if log.
	scannedBytesOutcomeRejectedMaxDuration = "rejected_max_duration"
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
		Help:      "Queries evaluated against the scanned bytes limit by outcome.",
	}, []string{metricLabelTenant, metricLabelOp, metricLabelOutcome})
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
	// scannedBytes estimates the bytes a query scans in a block. Returning 0 leaves the block out.
	scannedBytes(m *backend.BlockMeta) uint64
}

type searchScannedBytesLimit struct {
	cfg       SearchSharderConfig
	overrides overrides.Interface
}

func (searchScannedBytesLimit) op() string { return searchOp }

func (l searchScannedBytesLimit) limits(tenantID string) rangeLimits {
	return rangeLimits{
		minDuration:     cmp.Or(l.overrides.MinSearchDuration(tenantID), l.cfg.MinDuration),
		maxDuration:     cmp.Or(l.overrides.MaxSearchDuration(tenantID), l.cfg.MaxDuration),
		maxScannedBytes: cmp.Or(l.overrides.MaxSearchScannedBytes(tenantID), l.cfg.MaxScannedBytes),
	}
}

func (searchScannedBytesLimit) scannedBytes(m *backend.BlockMeta) uint64 { return m.Size_ }

// metricsScannedBytesLimit checks metrics queries in the handlers, before traceql.AlignRequest widens the range.
type metricsScannedBytesLimit struct {
	cfg       QueryRangeSharderConfig
	overrides overrides.Interface
	reader    tempodb.Reader
	logger    log.Logger
	// multiTenant is multi_tenant_queries_enabled. Without it a later middleware rejects a|b, so it is not split here.
	multiTenant bool
}

func newMetricsScannedBytesLimit(reader tempodb.Reader, o overrides.Interface, cfg QueryRangeSharderConfig, multiTenant bool, logger log.Logger) *metricsScannedBytesLimit {
	return &metricsScannedBytesLimit{cfg: cfg, overrides: o, reader: reader, logger: logger, multiTenant: multiTenant}
}

func (*metricsScannedBytesLimit) op() string { return metricsOp }

func (l *metricsScannedBytesLimit) limits(tenantID string) rangeLimits {
	return rangeLimits{
		minDuration:     cmp.Or(l.overrides.MinMetricsDuration(tenantID), l.cfg.MinDuration),
		maxDuration:     metricsMaxDuration(l.overrides, tenantID, l.cfg.MaxDuration),
		maxScannedBytes: cmp.Or(l.overrides.MaxMetricsScannedBytes(tenantID), l.cfg.MaxScannedBytes),
	}
}

// scannedBytes counts only RF1 blocks, the only ones metrics queries read.
func (*metricsScannedBytesLimit) scannedBytes(m *backend.BlockMeta) uint64 {
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
	if len(tenants) > 1 && !l.multiTenant {
		return maxErr
	}
	span := trace.SpanFromContext(ctx)
	start, end := unixNanoTime(req.Start), unixNanoTime(req.End)
	cutoff := time.Now().Add(-l.cfg.QueryBackendAfter)

	rejection := maxErr
	for _, tenantID := range tenants {
		// TODO: the sharder walks the blocklist again. See "Optimization (future)" in the design doc.
		c := checkScannedBytes(l, tenantID, start, end, cutoff, func() []*backend.BlockMeta { return l.reader.BlockMetas(tenantID) })
		if maxErr != nil {
			// The raw org ID's max_duration rejected the request, even if this tenant's own allows it.
			c.outcome = scannedBytesOutcomeRejectedMaxDuration
		}
		c.record(span, l.logger)
		switch {
		// With the limit off, validateMetricsQueryMaxDuration stays the only max_duration check.
		case rejection != nil || c.maxScannedBytes == 0:
		case c.outcome == scannedBytesOutcomeRejectedMaxDuration:
			rejection = fmt.Errorf("%w of %s", errMetricsMaxDuration, c.maxDuration)
		default:
			rejection = c.err("metrics query")
		}
	}
	return rejection
}

// rangeCheck is one tenant's range checked against its rangeLimits.
type rangeCheck struct {
	rangeLimits
	op, tenantID string
	requested    time.Duration
	outcome      string
	// evaluated is false when nothing is recorded: the limit is off, max_duration is negative, or there is no range.
	evaluated bool
	// fits is true when the blocks in the requested range are within max_scanned_bytes.
	fits bool
	// derived is the range max_scanned_bytes allows, before min_duration and max_duration. Only set when !fits.
	derived      time.Duration
	allowed      time.Duration
	scannedBytes uint64
}

// checkScannedBytes allows the longest range ending at end whose blocks fit max_scanned_bytes, raised to
// min_duration and capped at max_duration.
func checkScannedBytes(l scannedBytesLimit, tenantID string, start, end, cutoff time.Time, metas func() []*backend.BlockMeta) rangeCheck {
	c := rangeCheck{
		rangeLimits: l.limits(tenantID),
		op:          l.op(),
		tenantID:    tenantID,
		requested:   end.Sub(start),
		outcome:     scannedBytesOutcomeWithin,
		fits:        true,
	}
	overMax := c.maxDuration != 0 && c.requested > c.maxDuration
	if overMax {
		c.outcome = scannedBytesOutcomeRejectedMaxDuration
	}
	// A negative max_duration rejects everything and would pollute the what-if log. A zero start has no backend range.
	if c.maxScannedBytes == 0 || c.maxDuration < 0 || start.UnixNano() <= 0 || !end.After(start) {
		return c
	}
	c.evaluated = true

	backendEnd := end
	if backendEnd.After(cutoff) {
		backendEnd = cutoff
	}
	var blockEnd time.Time
	blockEnd, c.scannedBytes, c.fits = overflowBlockEnd(metas, l.scannedBytes, start, backendEnd, c.maxScannedBytes)
	c.allowed = c.maxDuration
	if c.fits {
		return c
	}

	// Overlap is inclusive, so the range must start 1ns after the block's end. The range after the cutoff is free.
	c.derived = max(end.Sub(blockEnd)-time.Nanosecond, end.Sub(backendEnd), 0)
	c.allowed = max(c.derived, c.minDuration)
	if c.maxDuration > 0 {
		c.allowed = min(c.allowed, c.maxDuration)
	}
	switch {
	case overMax:
	case c.requested > c.allowed:
		c.outcome = scannedBytesOutcomeRejected
	case c.allowed > c.derived:
		c.outcome = scannedBytesOutcomeRaisedToMin
	}
	return c
}

// err is the rejection error for a query of kind, such as "metrics query" or "search".
func (c rangeCheck) err(kind string) error {
	if c.outcome != scannedBytesOutcomeRejected {
		return nil
	}
	// Rounding in opposite directions keeps the two values distinct and the allowed one safe to retry.
	return fmt.Errorf("%s time range of %s exceeds the maximum allowed range of %s for tenant %s (limited by scanned bytes)",
		kind, (c.requested + time.Second - 1).Truncate(time.Second), c.allowed.Truncate(time.Second), c.tenantID)
}

// record emits the metrics, span attributes, and what-if log line for an evaluated check.
func (c rangeCheck) record(span trace.Span, logger log.Logger) {
	if !c.evaluated {
		return
	}
	scannedBytesLimitTotal.WithLabelValues(c.tenantID, c.op, c.outcome).Inc()
	// Over-max queries are left out, so the histogram reflects what the limit decides today.
	// The raw range carries the calibration signal, since min_duration and max_duration can pin allowed.
	if !c.fits && c.outcome != scannedBytesOutcomeRejectedMaxDuration {
		scannedBytesLimitDerivedRange.WithLabelValues(c.tenantID, c.op).Observe(c.derived.Seconds())
	}

	//nolint:goconst // logfmt keys match the other query log lines.
	kv := []any{
		"msg", "scanned bytes limit",
		"tenant", c.tenantID,
		"op", c.op,
		// Rounded like the rejection error: requested up, limits down.
		"requested_seconds", int64((c.requested + time.Second - 1) / time.Second),
		"scanned_bytes", c.scannedBytes,
	}
	if c.fits {
		kv = append(kv, "fits", true)
	} else {
		kv = append(kv, "derived_seconds", int64(c.derived/time.Second))
	}
	kv = append(kv,
		"min_seconds", int64(c.minDuration/time.Second),
		"max_seconds", int64(c.maxDuration/time.Second),
		"outcome", c.outcome,
		// Whether max_scanned_bytes and min_duration alone allow the query, to size a max_duration raise.
		"passes_scanned_bytes", c.fits || c.requested <= max(c.derived, c.minDuration),
	)
	level.Info(logger).Log(kv...)
	span.SetAttributes(
		attribute.String("scannedBytesLimitOutcome", c.outcome),
		attribute.Int64("scannedBytesLimitAllowedSeconds", int64(c.allowed/time.Second)),
	)
}

// overflowBlockEnd returns the EndTime of the block that pushes the estimated bytes in [start, backendEnd] over maxScannedBytes,
// adding blocks newest first, and the total bytes in the range. fits is true when every block in the range fits.
func overflowBlockEnd(metas func() []*backend.BlockMeta, estimate func(m *backend.BlockMeta) uint64, start, backendEnd time.Time, maxScannedBytes uint64) (blockEnd time.Time, scanned uint64, fits bool) {
	if !backendEnd.After(start) {
		return time.Time{}, 0, true
	}
	all := metas()
	overlaps := func(m *backend.BlockMeta) bool {
		return !m.StartTime.After(backendEnd) && !m.EndTime.Before(start)
	}

	// Sum first so queries within budget pay no allocation or sort.
	n := 0
	for _, m := range all {
		if overlaps(m) {
			if size := estimate(m); size > 0 {
				scanned += size
				n++
			}
		}
	}
	if scanned <= maxScannedBytes {
		return time.Time{}, scanned, true
	}

	// Value pairs keep the sort in cache instead of chasing meta pointers.
	type block struct {
		end  int64
		size uint64
	}
	blocks := make([]block, 0, n)
	for _, m := range all {
		if overlaps(m) {
			if size := estimate(m); size > 0 {
				blocks = append(blocks, block{end: m.EndTime.UnixNano(), size: size})
			}
		}
	}
	slices.SortFunc(blocks, func(a, b block) int {
		return cmp.Compare(b.end, a.end)
	})

	var running uint64
	for _, b := range blocks {
		running += b.size
		if running > maxScannedBytes {
			return time.Unix(0, b.end), scanned, false
		}
	}
	return time.Time{}, scanned, true
}

// unixNanoTime converts a request timestamp without wrapping past math.MaxInt64.
func unixNanoTime(ns uint64) time.Time {
	return time.Unix(0, int64(min(ns, math.MaxInt64)))
}
