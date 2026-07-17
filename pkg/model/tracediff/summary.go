package tracediff

import (
	"maps"
	"math"
	"sort"

	"github.com/grafana/tempo/pkg/tempopb"
)

const (
	signalIncreased = "increased"
	signalDecreased = "decreased"
	signalUnchanged = "unchanged"
	// Aggregate drift significance: a service counts as drifted when the
	// absolute per-service sum-of-span-durations delta is at least
	// max(driftMinNanos, 5% of the base sum). This is what lets the
	// summary attribute systemic sub-tolerance drift (every span slightly
	// slower) that the per-span patch tolerance filters out entirely.
	driftMinNanos      = int64(1_000_000)
	driftRelativeScale = int64(20)
)

// SummaryResult is the trace-summary-v0-native document: a compact
// triage/localization summary of a trace diff. The envelope, signals, and
// per-service summed-duration deltas are computed from the normalized traces — not from
// the patch — because whole-trace facts (the latency envelope, the topology
// signature, sub-tolerance drift) are not derivable from the change list.
// Matcher-derived counts complement them: net-zero churn that cancels out of
// aggregates stays visible in counts.
type SummaryResult struct {
	Version  string       `json:"version"`
	Base     TraceSummary `json:"base"`
	Compare  TraceSummary `json:"compare"`
	Signals  Signals      `json:"signals"`
	Trust    TrustSummary `json:"trust"`
	Stats    SummaryStats `json:"stats"`
	Warnings []Warning    `json:"warnings,omitempty"`
	// ChangedServices contains all services with matcher changes, significant
	// sum-of-span-durations drift, or structural changes.
	ChangedServices []string `json:"changedServices"`
	// Services contains rollups for services with matcher changes or significant
	// sum-of-span-durations drift. Structure-only services remain in ChangedServices.
	Services []ServiceRollup `json:"services"`
}

// Signals are factual direction labels. They intentionally avoid declaring that
// a change is good or bad; callers and LLMs can interpret the directions in
// context.
type Signals struct {
	TraceLatency    string `json:"traceLatency"`
	SumSpanDuration string `json:"sumSpanDuration"`
	Errors          string `json:"errors"`
	SpanCount       string `json:"spanCount"`
	Structure       string `json:"structure"`
}

// TraceSummary describes one side of the comparison.
type TraceSummary struct {
	TraceID        string `json:"traceId"`
	RootService    string `json:"rootService,omitempty"`
	RootName       string `json:"rootName,omitempty"`
	SpanCount      int    `json:"spanCount"`
	ErrorSpanCount int    `json:"errorSpanCount"`
	// DurationMs is the trace wall-clock envelope: the latest span end minus
	// the earliest span start across spans with valid durations (start > 0 and
	// end >= start). It is a proxy for end-to-end latency and is 0 when no span
	// has a valid duration.
	DurationMs int64 `json:"durationMs"`
	// SumSpanDurationMs is the sum of every span's individual duration, which can exceed
	// DurationMs when spans run concurrently. Invalid spans contribute zero.
	SumSpanDurationMs int64 `json:"sumSpanDurationMs"`
}

// TrustSummary reports span matching coverage. It does not measure whether
// duplicate logical spans were paired unambiguously.
type TrustSummary struct {
	// MatchedSpanRatio is matched spans over the larger trace's span count, so it
	// falls when either side has many unmatched spans.
	MatchedSpanRatio float64 `json:"matchedSpanRatio"`
	// StructureOverlap is matched spans over the smaller trace's span count. It is
	// a containment ratio (1.0 when the smaller trace is fully contained in the
	// larger), not a symmetric overlap, so read it together with MatchedSpanRatio.
	StructureOverlap float64 `json:"structureOverlap"`
}

// SummaryStats holds aggregate deltas between the two traces.
type SummaryStats struct {
	// TraceLatencyDeltaMs is the difference in the wall-clock envelope (see
	// TraceSummary.DurationMs), not the difference in any single root span.
	TraceLatencyDeltaMs    int64 `json:"traceLatencyDeltaMs"`
	SumSpanDurationDeltaMs int64 `json:"sumSpanDurationDeltaMs"`
	SpanCountDelta         int   `json:"spanCountDelta"`
	ErrorSpanDelta         int   `json:"errorSpanDelta"`
	MatchedSpans           int   `json:"matchedSpans"`
	ModifiedSpans          int   `json:"modifiedSpans"`
	AddedSpans             int   `json:"addedSpans"`
	RemovedSpans           int   `json:"removedSpans"`
	FieldChanges           int   `json:"fieldChanges"`
	AttributeChanges       int   `json:"attributeChanges"`
}

// ServiceRollup summarizes one changed service. SumSpanDurationDeltaMs is computed
// from raw nanoseconds before final millisecond conversion; count columns come
// from the matcher so unambiguous net-zero churn remains visible.
type ServiceRollup struct {
	Name                   string `json:"name"`
	SumSpanDurationDeltaMs int64  `json:"sumSpanDurationDeltaMs"`
	Modified               int    `json:"modified"`
	Added                  int    `json:"added"`
	Removed                int    `json:"removed"`
	NewErrors              int    `json:"newErrors"`
	ResolvedErrors         int    `json:"resolvedErrors"`
}

// Summarize compares base and compare and returns the trace-summary-v0-native
// document built from the normalized traces plus the trace-patch-v0 diff.
func Summarize(base, compare *tempopb.Trace) (*SummaryResult, error) {
	patch, baseTrace, compareTrace, err := diffTraceInputs(base, compare)
	if err != nil {
		return nil, err
	}

	baseSummary, baseDurations := summarizeNormalizedTrace(baseTrace)
	compareSummary, compareDurations := summarizeNormalizedTrace(compareTrace)
	structureChanged := structureChangedServices(structureSignature(baseTrace), structureSignature(compareTrace))

	result := &SummaryResult{
		Version: VersionTraceSummaryV0Native,
		Base:    baseSummary,
		Compare: compareSummary,
		Trust: TrustSummary{
			MatchedSpanRatio: matchedSpanRatio(patch.Stats.SpanCountA, patch.Stats.SpanCountB, patch.Stats.MatchedSpans),
			StructureOverlap: structureOverlap(patch.Stats.SpanCountA, patch.Stats.SpanCountB, patch.Stats.MatchedSpans),
		},
		Stats: SummaryStats{
			SpanCountDelta:   patch.Stats.SpanCountB - patch.Stats.SpanCountA,
			ErrorSpanDelta:   compareSummary.ErrorSpanCount - baseSummary.ErrorSpanCount,
			MatchedSpans:     patch.Stats.MatchedSpans,
			ModifiedSpans:    patch.Stats.ModifiedSpans,
			AddedSpans:       patch.Stats.AddedSpans,
			RemovedSpans:     patch.Stats.RemovedSpans,
			FieldChanges:     patch.Stats.FieldChanges,
			AttributeChanges: patch.Stats.AttributeChanges,
		},
	}
	result.Stats.TraceLatencyDeltaMs = deltaUint64Millis(compareDurations.envelopeNanos, baseDurations.envelopeNanos)
	result.Stats.SumSpanDurationDeltaMs = nanosToMillis(compareDurations.sumSpanDurationNanos - baseDurations.sumSpanDurationNanos)
	result.Signals = summarySignals(result.Stats, len(structureChanged) > 0)
	result.ChangedServices, result.Services = buildServiceSections(
		patch,
		serviceSumSpanDurationNanos(baseTrace),
		serviceSumSpanDurationNanos(compareTrace),
		structureChanged,
	)
	result.Warnings = patch.Warnings
	return result, nil
}

type traceDurationAggregate struct {
	envelopeNanos        uint64
	sumSpanDurationNanos int64
}

// summarizeNormalizedTrace keeps nanoseconds until cross-trace deltas have been
// calculated. Invalid spans contribute zero duration, matching the native
// reference policy.
func summarizeNormalizedTrace(trace normalizedTrace) (TraceSummary, traceDurationAggregate) {
	summary := TraceSummary{TraceID: trace.meta.TraceID, SpanCount: trace.meta.SpanCount}
	var durations traceDurationAggregate
	var firstStart uint64
	var lastEnd uint64
	var haveStart bool
	for _, span := range trace.spans {
		// A start time of 0 is unset (OTLP start times are required and non-zero);
		// including it would collapse firstStart to the epoch and report a trace
		// duration of tens of thousands of years. Skip unset starts so a single
		// span with buggy instrumentation cannot poison the whole envelope.
		start := span.startUnixNano
		end := span.endUnixNano
		if span.durationValid && (!haveStart || start < firstStart) {
			firstStart = start
			haveStart = true
		}
		if span.durationValid && end > lastEnd {
			lastEnd = end
		}
		durations.sumSpanDurationNanos += span.snapshot.DurationNanos
		if isErrorStatus(span.snapshot.Status) {
			summary.ErrorSpanCount++
		}
	}
	if haveStart && lastEnd >= firstStart {
		durations.envelopeNanos = lastEnd - firstStart
	}
	summary.SumSpanDurationMs = nanosToMillis(durations.sumSpanDurationNanos)
	summary.DurationMs = int64(durations.envelopeNanos / 1_000_000)
	if len(trace.spans) > 0 {
		summary.RootService = trace.spans[0].ref.Service
		summary.RootName = trace.spans[0].ref.Name
	}
	return summary, durations
}

// structureSigEntry is one row of a structure signature: a span's logical
// identity plus its parent linkage.
type structureSigEntry struct {
	identity       spanLogicalKey
	parentIdentity spanLogicalKey
	hasParent      bool
}

// structureSignature builds each service's multiset of signature rows from the
// normalized spans. Sibling order never enters a row, so the signature is
// order-insensitive by construction, while any reparenting rewrites the moved
// span's row.
func structureSignature(trace normalizedTrace) map[string]map[structureSigEntry]int {
	sig := make(map[string]map[structureSigEntry]int)
	for _, span := range trace.spans {
		service := serviceOrUnknown(span.ref.Service)
		rows := sig[service]
		if rows == nil {
			rows = make(map[structureSigEntry]int)
			sig[service] = rows
		}
		rows[structureSigEntry{
			identity:       span.identity(),
			parentIdentity: span.parentIdentity,
			hasParent:      span.hasParent,
		}]++
	}
	return sig
}

// structureChangedServices returns the services whose signature multiset
// differs between base and compare. The signature partitions the whole trace
// by service, so a non-empty result is also the whole-trace structure-changed
// signal, and a matched span's parent change always rewrites its row.
func structureChangedServices(base, compare map[string]map[structureSigEntry]int) map[string]struct{} {
	changed := make(map[string]struct{})
	for service, baseRows := range base {
		if !maps.Equal(baseRows, compare[service]) {
			changed[service] = struct{}{}
		}
	}
	for service := range compare {
		if _, ok := base[service]; !ok {
			changed[service] = struct{}{}
		}
	}
	return changed
}

func summarySignals(stats SummaryStats, structureChanged bool) Signals {
	structure := signalUnchanged
	if stats.AddedSpans > 0 || stats.RemovedSpans > 0 || structureChanged {
		structure = "changed"
	}
	return Signals{
		TraceLatency:    signalFromDelta(stats.TraceLatencyDeltaMs),
		SumSpanDuration: signalFromDelta(stats.SumSpanDurationDeltaMs),
		Errors:          signalFromDelta(int64(stats.ErrorSpanDelta)),
		SpanCount:       signalFromDelta(int64(stats.SpanCountDelta)),
		Structure:       structure,
	}
}

func signalFromDelta(delta int64) string {
	switch {
	case delta > 0:
		return signalIncreased
	case delta < 0:
		return signalDecreased
	default:
		return signalUnchanged
	}
}

// serviceSumSpanDurationNanos sums span durations per service over a whole normalized
// trace. Unlike the patch, this sees every span, so systemic drift below the
// per-span tolerance still accumulates here.
func serviceSumSpanDurationNanos(trace normalizedTrace) map[string]int64 {
	sums := make(map[string]int64)
	for _, span := range trace.spans {
		sums[serviceOrUnknown(span.ref.Service)] += span.snapshot.DurationNanos
	}
	return sums
}

type rankedServiceRollup struct {
	rollup ServiceRollup
}

// buildServiceSections follows the native reference contract: changedServices
// is complete, while service rollups contain matcher churn and significant
// trace-derived drift. Structure-only changes remain in changedServices.
func buildServiceSections(patch *Result, baseDurationSums, compareDurationSums map[string]int64, structureChanged map[string]struct{}) ([]string, []ServiceRollup) {
	counts := matcherRollupCounts(patch)

	services := make(map[string]struct{}, len(baseDurationSums)+len(compareDurationSums))
	for service := range baseDurationSums {
		services[service] = struct{}{}
	}
	for service := range compareDurationSums {
		services[service] = struct{}{}
	}

	changed := make(map[string]struct{}, len(counts)+len(structureChanged))
	for service := range counts {
		changed[service] = struct{}{}
	}
	for service := range structureChanged {
		changed[service] = struct{}{}
	}

	rollups := make([]rankedServiceRollup, 0, len(services))
	for service := range services {
		deltaNanos := compareDurationSums[service] - baseDurationSums[service]
		count := counts[service]
		drifted := driftSignificant(baseDurationSums[service], deltaNanos)
		if drifted {
			changed[service] = struct{}{}
		} else if count == nil {
			continue
		}
		rollup := ServiceRollup{Name: service}
		rollup.SumSpanDurationDeltaMs = nanosToMillis(deltaNanos)
		if count != nil {
			rollup.Modified = count.Modified
			rollup.Added = count.Added
			rollup.Removed = count.Removed
			rollup.NewErrors = count.NewErrors
			rollup.ResolvedErrors = count.ResolvedErrors
		}
		rollups = append(rollups, rankedServiceRollup{rollup: rollup})
	}
	sort.SliceStable(rollups, func(i, j int) bool {
		a, b := rollups[i].rollup, rollups[j].rollup
		if absInt64(a.SumSpanDurationDeltaMs) != absInt64(b.SumSpanDurationDeltaMs) {
			return absInt64(a.SumSpanDurationDeltaMs) > absInt64(b.SumSpanDurationDeltaMs)
		}
		if a.NewErrors != b.NewErrors {
			return a.NewErrors > b.NewErrors
		}
		return a.Name < b.Name
	})

	outputRollups := make([]ServiceRollup, 0, len(rollups))
	for _, ranked := range rollups {
		outputRollups = append(outputRollups, ranked.rollup)
	}
	changedServices := make([]string, 0, len(changed))
	for service := range changed {
		changedServices = append(changedServices, service)
	}
	sort.Strings(changedServices)
	return changedServices, outputRollups
}

// driftSignificant applies the aggregate drift threshold in nanoseconds, so a
// sub-millisecond jitter can never round up past the absolute floor.
func driftSignificant(baseNanos, deltaNanos int64) bool {
	absDelta := absInt64(deltaNanos)
	if absDelta < driftMinNanos {
		return false
	}
	relativeThreshold := baseNanos / driftRelativeScale
	if baseNanos%driftRelativeScale != 0 {
		relativeThreshold++
	}
	return absDelta >= relativeThreshold
}

// matcherRollupCounts extracts the per-service count columns from the patch.
// Status changes are never tolerance-filtered, so matcher-derived error
// transitions preserve simultaneous churn even when the net error delta is 0.
func matcherRollupCounts(patch *Result) map[string]*ServiceRollup {
	counts := map[string]*ServiceRollup{}
	get := func(service string) *ServiceRollup {
		service = serviceOrUnknown(service)
		count := counts[service]
		if count == nil {
			count = &ServiceRollup{Name: service}
			counts[service] = count
		}
		return count
	}
	for _, modified := range patch.Modified {
		count := get(modified.Span.Service)
		count.Modified++
		for _, change := range modified.Changes {
			if change.Target.Type != TargetField || change.Target.Name != FieldStatus {
				continue
			}
			before, _ := change.Before.(string)
			after, _ := change.After.(string)
			if isErrorStatus(after) && !isErrorStatus(before) {
				count.NewErrors++
			} else if isErrorStatus(before) && !isErrorStatus(after) {
				count.ResolvedErrors++
			}
		}
	}
	for _, added := range patch.Added {
		count := get(added.Span.Service)
		count.Added++
		if isErrorStatus(added.Span.Status) {
			count.NewErrors++
		}
	}
	for _, removed := range patch.Removed {
		count := get(removed.Span.Service)
		count.Removed++
		if isErrorStatus(removed.Span.Status) {
			count.ResolvedErrors++
		}
	}
	return counts
}

func serviceOrUnknown(service string) string {
	if service == "" {
		return unknownServiceName
	}
	return service
}

const unknownServiceName = "<unknown>"

// matchedSpanRatio divides matched spans by the larger span count, so it stays
// low whenever either trace has many unmatched spans.
func matchedSpanRatio(a, b, matched int) float64 {
	if a == 0 && b == 0 {
		return 1
	}
	denominator := max(a, b)
	if denominator == 0 {
		return 0
	}
	return float64(matched) / float64(denominator)
}

// structureOverlap divides matched spans by the smaller span count. This is a
// containment ratio: it reaches 1.0 when the smaller trace is fully contained in
// the larger one even if the larger trace has many extra spans. Pair it with
// matchedSpanRatio (which uses the larger count) to detect that asymmetry.
func structureOverlap(a, b, matched int) float64 {
	if a == 0 && b == 0 {
		return 1
	}
	denominator := min(a, b)
	if denominator == 0 {
		return 0
	}
	return float64(matched) / float64(denominator)
}

func isErrorStatus(status string) bool {
	return status == "error"
}

// sanitizeJSONValue replaces non-finite float64 values (NaN, +Inf, -Inf) with
// string placeholders so diff outputs remain JSON-encodable. Arrays and maps
// are sanitized recursively.
func sanitizeJSONValue(value any) any {
	switch v := value.(type) {
	case float64:
		switch {
		case math.IsNaN(v):
			return "NaN"
		case math.IsInf(v, 1):
			return "+Inf"
		case math.IsInf(v, -1):
			return "-Inf"
		default:
			return v
		}
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = sanitizeJSONValue(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = sanitizeJSONValue(item)
		}
		return out
	default:
		return v
	}
}

func absInt64(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

func nanosToMillis(value int64) int64 {
	return value / 1_000_000
}

func deltaUint64Millis(compare, base uint64) int64 {
	if compare >= base {
		return int64((compare - base) / 1_000_000)
	}
	return -int64((base - compare) / 1_000_000)
}
