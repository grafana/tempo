package tracediff

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash"
	"maps"
	"math"
	"sort"

	"github.com/grafana/tempo/pkg/tempopb"
)

const (
	signalIncreased      = "increased"
	signalDecreased      = "decreased"
	signalUnchanged      = "unchanged"
	patternStateAdded    = "added"
	patternStateRemoved  = "removed"
	patternStateModified = "modified"

	// patternCap bounds the patterns section. Overflow is disclosed in
	// patternsTruncated instead of silently dropped: a bounded section must
	// never be the only carrier of a fact class, and truncation without
	// disclosure lies by omission.
	patternCap = 20
	// patternSampleSpans bounds the sample span refs carried per pattern.
	patternSampleSpans = 3
	// Aggregate drift significance: a service counts as drifted when the
	// absolute per-service span-work delta is at least
	// max(driftMinNanos, 5% of base work). This is what lets the
	// summary attribute systemic sub-tolerance drift (every span slightly
	// slower) that the per-span patch tolerance filters out entirely.
	driftMinNanos      = int64(1_000_000)
	driftRelativeScale = int64(20)
)

// SummaryResult is the trace-summary-v0-native document: a compact
// triage/localization summary of a trace diff. The envelope, signals, and
// per-service work deltas are computed from the normalized traces — not from
// the patch — because whole-trace facts (the latency envelope, the topology
// signature, sub-tolerance drift) are not derivable from the change list.
// Matcher-derived counts complement them: net-zero churn on unambiguous spans
// cancels out of aggregates but stays visible in counts.
type SummaryResult struct {
	Version  string       `json:"version"`
	Base     TraceSummary `json:"base"`
	Compare  TraceSummary `json:"compare"`
	Signals  Signals      `json:"signals"`
	Trust    TrustSummary `json:"trust"`
	Stats    SummaryStats `json:"stats"`
	Warnings []Warning    `json:"warnings,omitempty"`
	// ChangedServices contains all services with matcher changes, significant
	// span-work drift, or structural changes.
	ChangedServices []string `json:"changedServices"`
	// Services contains rollups for services with matcher changes or significant
	// span-work drift. Structure-only services remain in ChangedServices.
	Services []ServiceRollup `json:"services"`
	// Patterns groups identical span changes into exemplars, capped at
	// patternCap with explicit truncation disclosure.
	Patterns          []Pattern          `json:"patterns"`
	PatternsTruncated *PatternsTruncated `json:"patternsTruncated,omitempty"`
}

// Signals are factual direction labels. They intentionally avoid declaring that
// a change is good or bad; callers and LLMs can interpret the directions in
// context.
type Signals struct {
	TraceLatency string `json:"traceLatency"`
	SpanWork     string `json:"spanWork"`
	Errors       string `json:"errors"`
	SpanCount    string `json:"spanCount"`
	Structure    string `json:"structure"`
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
	// SpanWorkMs is the sum of every span's individual duration, which can exceed
	// DurationMs when spans run concurrently. Invalid spans contribute zero.
	SpanWorkMs int64 `json:"spanWorkMs"`
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
	TraceLatencyDeltaMs int64 `json:"traceLatencyDeltaMs"`
	SpanWorkDeltaMs     int64 `json:"spanWorkDeltaMs"`
	SpanCountDelta      int   `json:"spanCountDelta"`
	ErrorSpanDelta      int   `json:"errorSpanDelta"`
	MatchedSpans        int   `json:"matchedSpans"`
	ModifiedSpans       int   `json:"modifiedSpans"`
	AddedSpans          int   `json:"addedSpans"`
	RemovedSpans        int   `json:"removedSpans"`
	FieldChanges        int   `json:"fieldChanges"`
	AttributeChanges    int   `json:"attributeChanges"`
}

// ServiceRollup summarizes one changed service. SpanWorkDeltaMs is computed
// from raw nanoseconds before final millisecond conversion; count columns come
// from the matcher so unambiguous net-zero churn remains visible.
type ServiceRollup struct {
	Name            string `json:"name"`
	SpanWorkDeltaMs int64  `json:"spanWorkDeltaMs"`
	Modified        int    `json:"modified"`
	Added           int    `json:"added"`
	Removed         int    `json:"removed"`
	NewErrors       int    `json:"newErrors"`
	ResolvedErrors  int    `json:"resolvedErrors"`
}

// PatternSpan is the logical identity a pattern groups by.
type PatternSpan struct {
	Service string `json:"service"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
}

// Pattern is an exemplar of one repeated change shape: identical changes on
// spans with the same logical identity collapse into a single entry with a
// count, a bounded set of sample refs, and the exemplar's rendered changes.
// Duration changes group by direction, not exact values, so per-span jitter
// does not defeat the compression.
type Pattern struct {
	State       string      `json:"state"`
	Span        PatternSpan `json:"span"`
	Count       int         `json:"count"`
	SampleSpans []SpanRef   `json:"sampleSpans"`
	// Changes is the exemplar span's rendered change list; durations are
	// rendered in milliseconds for readability.
	Changes         []Change            `json:"changes,omitempty"`
	DurationDeltaMs *DurationDeltaStats `json:"durationDeltaMs,omitempty"`
}

// DurationDeltaStats summarizes the duration deltas across all spans grouped
// into one pattern.
type DurationDeltaStats struct {
	Min   int64 `json:"min"`
	Max   int64 `json:"max"`
	Total int64 `json:"total"`
}

// PatternsTruncated discloses how much the pattern cap dropped.
type PatternsTruncated struct {
	Patterns int `json:"patterns"`
	Spans    int `json:"spans"`
}

// fieldDurationMs is the rendered Target.Name for duration changes inside
// patterns (the patch itself always uses FieldDurationNanos).
const fieldDurationMs = "duration_ms"

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
	result.Stats.SpanWorkDeltaMs = nanosToMillis(compareDurations.workNanos - baseDurations.workNanos)
	result.Signals = summarySignals(result.Stats, len(structureChanged) > 0)
	result.ChangedServices, result.Services = buildServiceSections(
		patch,
		serviceWorkNanos(baseTrace),
		serviceWorkNanos(compareTrace),
		structureChanged,
	)
	result.Patterns, result.PatternsTruncated = buildPatterns(patch)
	result.Warnings = patch.Warnings
	return result, nil
}

type traceDurationAggregate struct {
	envelopeNanos uint64
	workNanos     int64
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
		durations.workNanos += span.snapshot.DurationNanos
		if isErrorStatus(span.snapshot.Status) {
			summary.ErrorSpanCount++
		}
	}
	if haveStart && lastEnd >= firstStart {
		durations.envelopeNanos = lastEnd - firstStart
	}
	summary.SpanWorkMs = nanosToMillis(durations.workNanos)
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
		TraceLatency: signalFromDelta(stats.TraceLatencyDeltaMs),
		SpanWork:     signalFromDelta(stats.SpanWorkDeltaMs),
		Errors:       signalFromDelta(int64(stats.ErrorSpanDelta)),
		SpanCount:    signalFromDelta(int64(stats.SpanCountDelta)),
		Structure:    structure,
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

// serviceWorkNanos sums span durations per service over a whole normalized
// trace. Unlike the patch, this sees every span, so systemic drift below the
// per-span tolerance still accumulates here.
func serviceWorkNanos(trace normalizedTrace) map[string]int64 {
	work := make(map[string]int64)
	for _, span := range trace.spans {
		work[serviceOrUnknown(span.ref.Service)] += span.snapshot.DurationNanos
	}
	return work
}

type rankedServiceRollup struct {
	rollup ServiceRollup
}

// buildServiceSections follows the native reference contract: changedServices
// is complete, while service rollups contain matcher churn and significant
// trace-derived drift. Structure-only changes remain in changedServices.
func buildServiceSections(patch *Result, baseWork, compareWork map[string]int64, structureChanged map[string]struct{}) ([]string, []ServiceRollup) {
	counts := matcherRollupCounts(patch)

	services := make(map[string]struct{}, len(baseWork)+len(compareWork))
	for service := range baseWork {
		services[service] = struct{}{}
	}
	for service := range compareWork {
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
		deltaNanos := compareWork[service] - baseWork[service]
		count := counts[service]
		drifted := driftSignificant(baseWork[service], deltaNanos)
		if drifted {
			changed[service] = struct{}{}
		} else if count == nil {
			continue
		}
		rollup := ServiceRollup{Name: service}
		rollup.SpanWorkDeltaMs = nanosToMillis(deltaNanos)
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
		if absInt64(a.SpanWorkDeltaMs) != absInt64(b.SpanWorkDeltaMs) {
			return absInt64(a.SpanWorkDeltaMs) > absInt64(b.SpanWorkDeltaMs)
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
// transitions on unambiguous spans preserve simultaneous churn even when the
// net error delta is 0. Ambiguous duplicate groups use minimum-change multiset
// matching and carry a warning instead of inventing instance identity.
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

type patternInput struct {
	state   string
	ref     SpanRef
	changes []Change
}

type patternAccumulator struct {
	input           patternInput
	count           int
	samples         []SpanRef
	durationSet     bool
	durationMinMs   int64
	durationMaxMs   int64
	durationTotalNs int64
}

// buildPatterns follows the native reference: collect exact pattern counts,
// rank by frequency, and cap only the rendered pattern section.
func buildPatterns(patch *Result) ([]Pattern, *PatternsTruncated) {
	accumulators := make(map[[sha256.Size]byte]*patternAccumulator)
	order := make([][sha256.Size]byte, 0, len(patch.Modified)+len(patch.Added)+len(patch.Removed))
	forEachPatternInput(patch, func(input patternInput) {
		key := patternKey(input)
		acc := accumulators[key]
		if acc == nil {
			acc = &patternAccumulator{input: input, samples: make([]SpanRef, 0, patternSampleSpans)}
			accumulators[key] = acc
			order = append(order, key)
		}
		acc.count++
		if len(acc.samples) < patternSampleSpans {
			acc.samples = append(acc.samples, input.ref)
		}
		if delta, ok := patternDurationDeltaNanos(input); ok {
			deltaMs := nanosToMillis(delta)
			if !acc.durationSet {
				acc.durationSet = true
				acc.durationMinMs = deltaMs
				acc.durationMaxMs = deltaMs
			}
			acc.durationMinMs = min(acc.durationMinMs, deltaMs)
			acc.durationMaxMs = max(acc.durationMaxMs, deltaMs)
			acc.durationTotalNs += delta
		}
	})

	ranked := make([]*patternAccumulator, 0, len(order))
	for _, key := range order {
		ranked = append(ranked, accumulators[key])
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		a, b := ranked[i], ranked[j]
		if a.count != b.count {
			return a.count > b.count
		}
		if a.input.state != b.input.state {
			return a.input.state < b.input.state
		}
		if a.input.ref.Service != b.input.ref.Service {
			return a.input.ref.Service < b.input.ref.Service
		}
		if a.input.ref.Name != b.input.ref.Name {
			return a.input.ref.Name < b.input.ref.Name
		}
		return false
	})

	var patternTruncated *PatternsTruncated
	if len(ranked) > patternCap {
		patternTruncated = &PatternsTruncated{Patterns: len(ranked) - patternCap}
		for _, acc := range ranked[patternCap:] {
			patternTruncated.Spans += acc.count
		}
		ranked = ranked[:patternCap]
	}

	patterns := make([]Pattern, 0, len(ranked))
	for _, acc := range ranked {
		pattern := Pattern{
			State: acc.input.state,
			Span: PatternSpan{
				Service: acc.input.ref.Service,
				Name:    acc.input.ref.Name,
				Kind:    acc.input.ref.Kind,
			},
			Count:       acc.count,
			SampleSpans: append([]SpanRef(nil), acc.samples...),
			Changes:     renderPatternChanges(acc.input.changes),
		}
		if acc.durationSet {
			pattern.DurationDeltaMs = &DurationDeltaStats{
				Min:   acc.durationMinMs,
				Max:   acc.durationMaxMs,
				Total: nanosToMillis(acc.durationTotalNs),
			}
		}
		patterns = append(patterns, pattern)
	}
	return patterns, patternTruncated
}

func forEachPatternInput(patch *Result, visit func(patternInput)) {
	for _, modified := range patch.Modified {
		visit(patternInput{state: patternStateModified, ref: modified.Span, changes: modified.Changes})
	}
	for _, added := range patch.Added {
		visit(patternInput{state: patternStateAdded, ref: spanRefFromSnapshot(added.Span)})
	}
	for _, removed := range patch.Removed {
		visit(patternInput{state: patternStateRemoved, ref: spanRefFromSnapshot(removed.Span)})
	}
}

// patternKey uses a length-prefixed typed hash, so legal control characters in
// names or values cannot merge otherwise distinct patterns.
func patternKey(input patternInput) [sha256.Size]byte {
	hasher := sha256.New()
	hashString(hasher, input.state)
	hashString(hasher, input.ref.Service)
	hashString(hasher, input.ref.Name)
	hashString(hasher, input.ref.Kind)
	hashUint64(hasher, uint64(len(input.changes)))
	for _, change := range input.changes {
		hashString(hasher, "change")
		hashString(hasher, string(change.Op))
		if isDurationChange(change) {
			hashString(hasher, fieldDurationMs)
			hashString(hasher, durationChangeDirection(change))
			continue
		}
		hashString(hasher, string(change.Target.Type))
		hashString(hasher, change.Target.Name)
		hashString(hasher, change.Target.Scope)
		hashString(hasher, change.Target.Key)
		hashValue(hasher, change.Before)
		hashValue(hasher, change.After)
	}
	var key [sha256.Size]byte
	copy(key[:], hasher.Sum(nil))
	return key
}

func hashString(hasher hash.Hash, value string) {
	hashUint64(hasher, uint64(len(value)))
	_, _ = hasher.Write([]byte(value))
}

func hashUint64(hasher hash.Hash, value uint64) {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	_, _ = hasher.Write(encoded[:])
}

func hashValue(hasher hash.Hash, value any) {
	switch typed := value.(type) {
	case nil:
		hashString(hasher, "nil")
	case string:
		hashString(hasher, "string")
		hashString(hasher, typed)
	case bool:
		hashString(hasher, fmt.Sprintf("bool:%t", typed))
	case int64:
		hashString(hasher, "int64")
		var encoded [8]byte
		binary.BigEndian.PutUint64(encoded[:], uint64(typed))
		_, _ = hasher.Write(encoded[:])
	case float64:
		hashString(hasher, "float64")
		var encoded [8]byte
		binary.BigEndian.PutUint64(encoded[:], math.Float64bits(typed))
		_, _ = hasher.Write(encoded[:])
	case []byte:
		hashString(hasher, "bytes")
		hashUint64(hasher, uint64(len(typed)))
		_, _ = hasher.Write(typed)
	case []any:
		hashString(hasher, "array")
		hashUint64(hasher, uint64(len(typed)))
		for _, item := range typed {
			hashValue(hasher, item)
		}
	case map[string]any:
		hashString(hasher, "map")
		hashUint64(hasher, uint64(len(typed)))
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			hashString(hasher, key)
			hashValue(hasher, typed[key])
		}
	default:
		hashString(hasher, fmt.Sprintf("%T:%v", value, value))
	}
}

// renderPatternChanges converts duration values to milliseconds.
func renderPatternChanges(changes []Change) []Change {
	if len(changes) == 0 {
		return nil
	}
	out := make([]Change, 0, len(changes))
	for _, change := range changes {
		rendered := change
		if isDurationChange(change) {
			rendered.Target.Name = fieldDurationMs
			rendered.Before = renderDurationMs(change.Before)
			rendered.After = renderDurationMs(change.After)
		}
		out = append(out, rendered)
	}
	return out
}

func renderDurationMs(value any) any {
	if nanos, ok := value.(int64); ok {
		return nanosToMillis(nanos)
	}
	return value
}

func isDurationChange(change Change) bool {
	return change.Target.Type == TargetField && change.Target.Name == FieldDurationNanos
}

// durationChangeDeltaNanos returns the raw delta of a duration change. ok is
// false when either side is not a valid duration.
func durationChangeDeltaNanos(change Change) (int64, bool) {
	if !isDurationChange(change) {
		return 0, false
	}
	before, beforeOK := change.Before.(int64)
	after, afterOK := change.After.(int64)
	if !beforeOK || !afterOK {
		return 0, false
	}
	return after - before, true
}

func durationChangeDirection(change Change) string {
	before, beforeOK := change.Before.(int64)
	after, afterOK := change.After.(int64)
	if !beforeOK || !afterOK {
		return "invalid"
	}
	return signalFromDelta(nanosToMillis(after - before))
}

func patternDurationDeltaNanos(input patternInput) (int64, bool) {
	for _, change := range input.changes {
		if delta, ok := durationChangeDeltaNanos(change); ok {
			return delta, true
		}
	}
	return 0, false
}

func serviceOrUnknown(service string) string {
	if service == "" {
		return unknownServiceName
	}
	return service
}

const unknownServiceName = "<unknown>"

func spanRefFromSnapshot(snapshot SpanSnapshot) SpanRef {
	return SpanRef{Path: snapshot.Path, Service: snapshot.Service, Name: snapshot.Name, Kind: snapshot.Kind}
}

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
