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

	scannedBytesOutcomeWithin   = "within"
	scannedBytesOutcomeRejected = "rejected"
	scannedBytesOutcomeFloored  = "floored"
	// scannedBytesOutcomeRejectedMaxDuration marks queries over max_duration, evaluated only for the what-if log.
	scannedBytesOutcomeRejectedMaxDuration = "rejected_max_duration"
)

var (
	scannedBytesLimitDerivedRange = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metricNamespace,
		Name:      "query_frontend_scanned_bytes_limit_derived_range_seconds",
		Help:      "Query range the scanned bytes budget allows, before the floor and ceiling, for queries within max_duration that are over the budget.",
		Buckets: []float64{
			time.Hour.Seconds(),
			(6 * time.Hour).Seconds(),
			(24 * time.Hour).Seconds(),
			(2 * 24 * time.Hour).Seconds(),
			(3 * 24 * time.Hour).Seconds(),
			(5 * 24 * time.Hour).Seconds(),
			(7 * 24 * time.Hour).Seconds(),
			(14 * 24 * time.Hour).Seconds(),
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

// scannedBytesConfig is the config side of one query type's range limits. Overrides replace non-zero values per tenant.
type scannedBytesConfig struct {
	maxDuration     time.Duration
	minDuration     time.Duration
	maxScannedBytes uint64
}

// scannedBytesQuery is one tenant's query for evaluateScannedBytesLimit.
type scannedBytesQuery struct {
	// op is metricsOp or searchOp. It picks the overrides and the blocks that count.
	op         string
	tenantID   string
	requested  time.Duration
	start, end time.Time
	// cutoff is where backend blocks stop and generator or ingester data begins.
	cutoff time.Time
	// metas returns the tenant blocklist in any order. It is only called when backend blocks can matter.
	metas func() []*backend.BlockMeta
}

// scannedBytesDecision is the outcome of evaluateScannedBytesLimit and what record needs.
type scannedBytesDecision struct {
	op, tenantID string
	outcome      string
	requested    time.Duration
	// evaluated is false when nothing is recorded: the limit is off, the ceiling is negative, or there is no range.
	evaluated bool
	// limited is true when the blocks in the requested range exceed max_scanned_bytes.
	limited bool
	// allowed is the effective max duration. It is the ceiling when the blocks fit.
	allowed time.Duration
	// derived is the range the budget allows before the floor and ceiling. Only set when limited.
	derived         time.Duration
	scannedBytes    uint64
	minDuration     time.Duration
	maxDuration     time.Duration
	maxScannedBytes uint64
}

// evaluateScannedBytesLimit resolves a tenant's limits for one query type and decides whether the requested range
// is allowed, over max_duration, or over the effective max duration that the scanned bytes, floor, and ceiling set.
func evaluateScannedBytesLimit(o overrides.Interface, cfg scannedBytesConfig, q scannedBytesQuery) scannedBytesDecision {
	d := scannedBytesDecision{
		op:              q.op,
		tenantID:        q.tenantID,
		outcome:         scannedBytesOutcomeWithin,
		requested:       q.requested,
		maxDuration:     cfg.maxDuration,
		minDuration:     cfg.minDuration,
		maxScannedBytes: cfg.maxScannedBytes,
	}
	filter := acceptAllBlocks
	switch q.op {
	case metricsOp:
		filter = isMetricsGeneratorBlock
		d.maxDuration = metricsMaxDuration(o, q.tenantID, cfg.maxDuration)
		d.minDuration = cmp.Or(o.MinMetricsDuration(q.tenantID), cfg.minDuration)
		d.maxScannedBytes = cmp.Or(o.MaxMetricsScannedBytes(q.tenantID), cfg.maxScannedBytes)
	case searchOp:
		d.maxDuration = cmp.Or(o.MaxSearchDuration(q.tenantID), cfg.maxDuration)
		d.minDuration = cmp.Or(o.MinSearchDuration(q.tenantID), cfg.minDuration)
		d.maxScannedBytes = cmp.Or(o.MaxSearchScannedBytes(q.tenantID), cfg.maxScannedBytes)
	}

	overMax := d.maxDuration != 0 && q.requested > d.maxDuration
	if overMax {
		d.outcome = scannedBytesOutcomeRejectedMaxDuration
	}
	// A negative ceiling fails closed and would pollute the what-if log. A zero start has no backend range.
	if d.maxScannedBytes == 0 || d.maxDuration < 0 || q.start.UnixNano() <= 0 || !q.end.After(q.start) {
		return d
	}
	d.evaluated = true

	backendEnd := q.end
	if backendEnd.After(q.cutoff) {
		backendEnd = q.cutoff
	}
	blockEnd, scanned, limited := overflowBlockEnd(q.metas, filter, q.start, backendEnd, d.maxScannedBytes)
	d.scannedBytes = scanned
	d.allowed = d.maxDuration
	if !limited {
		return d
	}

	// Overlap is inclusive, so the range must start 1ns after the block's end. The range after the cutoff is free.
	d.derived = max(q.end.Sub(blockEnd)-time.Nanosecond, q.end.Sub(backendEnd), 0)
	d.limited = true
	d.allowed = max(d.derived, d.minDuration)
	if d.maxDuration > 0 {
		d.allowed = min(d.allowed, d.maxDuration)
	}
	switch {
	case overMax:
	case q.requested > d.allowed:
		d.outcome = scannedBytesOutcomeRejected
	case d.allowed > d.derived:
		d.outcome = scannedBytesOutcomeFloored
	}
	return d
}

// passesScannedBytes reports whether the budget alone, with its floor and without the ceiling, allows the request.
func (d scannedBytesDecision) passesScannedBytes() bool {
	return !d.limited || d.requested <= max(d.derived, d.minDuration)
}

// scannedBytesErr is the rejection error for a query of kind, such as "metrics query" or "search".
func (d scannedBytesDecision) scannedBytesErr(kind string) error {
	if d.outcome != scannedBytesOutcomeRejected {
		return nil
	}
	// Rounding in opposite directions keeps the two values distinct and the allowed one safe to retry.
	return fmt.Errorf("%s time range of %s exceeds the maximum allowed range of %s for tenant %s (limited by scanned bytes)",
		kind, (d.requested + time.Second - 1).Truncate(time.Second), d.allowed.Truncate(time.Second), d.tenantID)
}

// record emits the metrics, span attributes, and what-if log line for an evaluated decision.
func (d scannedBytesDecision) record(span trace.Span, logger log.Logger) {
	if !d.evaluated {
		return
	}
	scannedBytesLimitTotal.WithLabelValues(d.tenantID, d.op, d.outcome).Inc()
	// Over-max queries are left out, so the histogram reflects what the limit decides today.
	if d.limited && d.outcome != scannedBytesOutcomeRejectedMaxDuration {
		// The raw range carries the calibration signal, since the floor and ceiling can pin allowed.
		scannedBytesLimitDerivedRange.WithLabelValues(d.tenantID, d.op).Observe(d.derived.Seconds())
	}

	// Whole seconds, rounded like the rejection error: requested up, limits down.
	requestedSeconds := int64((d.requested + time.Second - 1) / time.Second)
	//nolint:goconst // logfmt keys match the other query log lines.
	kv := []any{
		"msg", "scanned bytes limit",
		"tenant", d.tenantID,
		"op", d.op,
		"requested_seconds", requestedSeconds,
		"scanned_bytes", d.scannedBytes,
	}
	attrs := []attribute.KeyValue{
		attribute.String("scannedBytesLimitOutcome", d.outcome),
		attribute.Int64("scannedBytesLimitRequestedSeconds", requestedSeconds),
		attribute.Int64("scannedBytesLimitScannedBytes", int64(min(d.scannedBytes, math.MaxInt64))),
		attribute.Int64("scannedBytesLimitMinSeconds", int64(d.minDuration/time.Second)),
		attribute.Int64("scannedBytesLimitMaxSeconds", int64(d.maxDuration/time.Second)),
		attribute.Bool("scannedBytesLimitPassesScannedBytes", d.passesScannedBytes()),
	}
	if d.limited {
		kv = append(kv, "derived_seconds", int64(d.derived/time.Second))
		attrs = append(attrs, attribute.Int64("scannedBytesLimitDerivedRangeSeconds", int64(d.derived/time.Second)))
	} else {
		kv = append(kv, "fits", true)
		attrs = append(attrs, attribute.Bool("scannedBytesLimitFits", true))
	}
	if d.allowed > 0 {
		attrs = append(attrs, attribute.Int64("scannedBytesLimitMaxDurationSeconds", int64(d.allowed/time.Second)))
	}
	kv = append(kv,
		"min_seconds", int64(d.minDuration/time.Second),
		"max_seconds", int64(d.maxDuration/time.Second),
		"outcome", d.outcome,
		"passes_scanned_bytes", d.passesScannedBytes(),
	)
	level.Info(logger).Log(kv...)
	span.SetAttributes(attrs...)
}

// overflowBlockEnd returns the EndTime of the block that pushes the bytes in [start, backendEnd] over maxScannedBytes,
// adding blocks newest first, and the total bytes in the range. limited is false when every block in the range fits.
func overflowBlockEnd(metas func() []*backend.BlockMeta, filter func(m *backend.BlockMeta) bool, start, backendEnd time.Time, maxScannedBytes uint64) (blockEnd time.Time, scanned uint64, limited bool) {
	if !backendEnd.After(start) {
		return time.Time{}, 0, false
	}
	all := metas()
	overlaps := func(m *backend.BlockMeta) bool {
		return !m.StartTime.After(backendEnd) && !m.EndTime.Before(start) && filter(m)
	}

	// Sum first so queries within budget pay no allocation or sort.
	n := 0
	for _, m := range all {
		if overlaps(m) {
			scanned += m.Size_
			n++
		}
	}
	if scanned <= maxScannedBytes {
		return time.Time{}, scanned, false
	}

	// Value pairs keep the sort in cache instead of chasing meta pointers.
	type block struct {
		end  int64
		size uint64
	}
	blocks := make([]block, 0, n)
	for _, m := range all {
		if overlaps(m) {
			blocks = append(blocks, block{end: m.EndTime.UnixNano(), size: m.Size_})
		}
	}
	slices.SortFunc(blocks, func(a, b block) int {
		return cmp.Compare(b.end, a.end)
	})

	var running uint64
	for _, b := range blocks {
		running += b.size
		if running > maxScannedBytes {
			return time.Unix(0, b.end), scanned, true
		}
	}
	return time.Time{}, scanned, false
}

// scannedBytesLimiter runs the metrics scanned bytes check in the handlers, before traceql.AlignRequest widens the range.
type scannedBytesLimiter struct {
	reader    tempodb.Reader
	overrides overrides.Interface
	cfg       QueryRangeSharderConfig
	logger    log.Logger
}

func newScannedBytesLimiter(reader tempodb.Reader, o overrides.Interface, cfg QueryRangeSharderConfig, logger log.Logger) *scannedBytesLimiter {
	return &scannedBytesLimiter{reader: reader, overrides: o, cfg: cfg, logger: logger}
}

// check records every tenant of the request, then returns the first tenant's rejection.
func (l *scannedBytesLimiter) check(ctx context.Context, req *tempopb.QueryRangeRequest) error {
	decisions, err := l.decide(ctx, req)
	if err != nil {
		return err
	}
	span := trace.SpanFromContext(ctx)
	var rejection error
	for _, d := range decisions {
		d.record(span, l.logger)
		// The limit is off for this tenant, so validateMetricsQueryMaxDuration's raw org ID check stays the only ceiling.
		if rejection != nil || d.maxScannedBytes == 0 {
			continue
		}
		// validateMetricsQueryMaxDuration checks the raw org ID. With the limit on, a tenant's own ceiling is enforced here too.
		if d.outcome == scannedBytesOutcomeRejectedMaxDuration {
			rejection = errMetricsMaxDuration(d.maxDuration)
		} else {
			rejection = d.scannedBytesErr("metrics query")
		}
	}
	return rejection
}

// recordRejectedMaxDuration records a request validateMetricsQueryMaxDuration rejected, for the what-if log only.
func (l *scannedBytesLimiter) recordRejectedMaxDuration(ctx context.Context, req *tempopb.QueryRangeRequest) {
	decisions, err := l.decide(ctx, req)
	if err != nil {
		// No valid tenants means nothing to record, and the max_duration error already goes back to the caller.
		return
	}
	span := trace.SpanFromContext(ctx)
	for _, d := range decisions {
		// The raw org ID's ceiling rejected the request, even when this tenant's own ceiling allows it.
		d.outcome = scannedBytesOutcomeRejectedMaxDuration
		d.record(span, l.logger)
	}
}

// decide evaluates every tenant of the request. It runs before the multi-tenant split, so each tenant gets its own limits.
func (l *scannedBytesLimiter) decide(ctx context.Context, req *tempopb.QueryRangeRequest) ([]scannedBytesDecision, error) {
	if l == nil {
		return nil, nil
	}
	tenants, err := tenant.TenantIDs(ctx)
	if err != nil {
		return nil, err
	}
	cutoff := time.Now().Add(-l.cfg.QueryBackendAfter)

	decisions := make([]scannedBytesDecision, 0, len(tenants))
	for _, tenantID := range tenants {
		decisions = append(decisions, evaluateScannedBytesLimit(l.overrides, scannedBytesConfig{
			maxDuration:     l.cfg.MaxDuration,
			minDuration:     l.cfg.MinDuration,
			maxScannedBytes: l.cfg.MaxScannedBytes,
		}, scannedBytesQuery{
			op:        metricsOp,
			tenantID:  tenantID,
			requested: time.Duration(min(req.End-req.Start, math.MaxInt64)),
			start:     unixNanoTime(req.Start),
			end:       unixNanoTime(req.End),
			cutoff:    cutoff,
			// TODO: the sharder walks the blocklist again. See "Optimization (future)" in the design doc.
			metas: func() []*backend.BlockMeta { return l.reader.BlockMetas(tenantID) },
		}))
	}
	return decisions, nil
}

// unixNanoTime converts a request timestamp without wrapping past math.MaxInt64.
func unixNanoTime(ns uint64) time.Time {
	return time.Unix(0, int64(min(ns, math.MaxInt64)))
}
