package tracediff

import (
	"fmt"
	"math"
	"slices"
	"sort"

	"github.com/grafana/tempo/pkg/tempopb"
)

const (
	// DefaultSummaryTopN bounds the ranked and grouped sections when TopN is unset.
	DefaultSummaryTopN = 10
)

// SummaryFormat selects a trace-summary-v0 output shape.
type SummaryFormat string

const (
	SummaryFormatAggregate SummaryFormat = VersionTraceSummaryV0Aggregate
	SummaryFormatRanked    SummaryFormat = VersionTraceSummaryV0Ranked
	SummaryFormatGrouped   SummaryFormat = VersionTraceSummaryV0Grouped
)

// SummaryOptions configures a Summarize call.
type SummaryOptions struct {
	// Format selects the summary representation; an empty value defaults to ranked.
	Format SummaryFormat
	// TopN bounds the ranked/grouped sections; a value <= 0 defaults to DefaultSummaryTopN.
	TopN int
}

// SummaryResult is the trace-summary-v0 document: a side-by-side comparison of two
// traces plus neutral direction signals, trust signals, aggregate stats, ranked
// per-span changes, and per-service groups.
type SummaryResult struct {
	Version    string       `json:"version"`
	Base       TraceSummary `json:"base"`
	Compare    TraceSummary `json:"compare"`
	Signals    Signals      `json:"signals"`
	Trust      TrustSummary `json:"trust"`
	Stats      SummaryStats `json:"stats"`
	TopChanges *TopChanges  `json:"topChanges,omitempty"`
	Groups     []Group      `json:"groups,omitempty"`
	Warnings   []Warning    `json:"warnings,omitempty"`
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
	// DurationMs is the trace wall-clock envelope: the latest span end minus the
	// earliest (non-zero) span start. It is a proxy for end-to-end latency and is
	// 0 when no span carries a usable start time.
	DurationMs int64 `json:"durationMs"`
	// SpanWorkMs is the sum of every span's individual duration, which can exceed
	// DurationMs when spans run concurrently.
	SpanWorkMs int64 `json:"spanWorkMs"`
}

// TrustSummary reports how confidently the two traces could be aligned; low values
// mean the diff matched few spans and the comparison should be read with caution.
type TrustSummary struct {
	// MatchedSpanRatio is matched spans over the larger trace's span count, so it
	// falls when either side has many unmatched spans.
	MatchedSpanRatio float64 `json:"matchedSpanRatio"`
	// StructureOverlap is matched spans over the smaller trace's span count. It is
	// a containment ratio (1.0 when the smaller trace is fully contained in the
	// larger), not a symmetric overlap, so read it together with MatchedSpanRatio.
	StructureOverlap float64 `json:"structureOverlap"`
}

// SummaryStats holds the aggregate, factual deltas between the two traces.
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

// TopChanges holds the most significant per-span changes, each section already
// sorted and truncated to TopN.
type TopChanges struct {
	Regressions  []ChangeSummary          `json:"regressions,omitempty"`
	Improvements []ChangeSummary          `json:"improvements,omitempty"`
	Structural   []ChangeSummary          `json:"structural,omitempty"`
	Status       []ChangeSummary          `json:"status,omitempty"`
	Attributes   []AttributeChangeSummary `json:"attributes,omitempty"`
}

// ChangeSummary describes a single span's change. State is one of "modified",
// "added", or "removed"; the status fields are populated for status transitions
// and for added/removed error spans.
type ChangeSummary struct {
	Span                 SpanRef `json:"span"`
	State                string  `json:"state"`
	DurationDeltaMs      int64   `json:"durationDeltaMs,omitempty"`
	StatusBefore         string  `json:"statusBefore,omitempty"`
	StatusAfter          string  `json:"statusAfter,omitempty"`
	AttributeChangeCount int     `json:"attributeChangeCount,omitempty"`
}

// AttributeChangeSummary describes a single span attribute that was added,
// removed, or modified.
type AttributeChangeSummary struct {
	Span   SpanRef   `json:"span"`
	Key    string    `json:"key"`
	Op     Operation `json:"op"`
	Before any       `json:"before,omitempty"`
	After  any       `json:"after,omitempty"`
}

// Group rolls up the changes for a single service. StatusChanges counts modified
// spans whose status changed plus added/removed spans that are already errors.
type Group struct {
	ServiceName           string `json:"serviceName"`
	SpanWorkDeltaMs       int64  `json:"spanWorkDeltaMs"`
	AddedSpans            int    `json:"addedSpans"`
	RemovedSpans          int    `json:"removedSpans"`
	ModifiedSpans         int    `json:"modifiedSpans"`
	StatusChanges         int    `json:"statusChanges"`
	NewErrorSpans         int    `json:"newErrorSpans,omitempty"`
	ResolvedErrorSpans    int    `json:"resolvedErrorSpans,omitempty"`
	AttributeChangedSpans int    `json:"attributeChangedSpans"`
}

// Summarize compares base and compare and returns a higher-level, human-oriented
// summary built on top of the trace-patch-v0 diff. TopN bounds the number of
// entries in the ranked and grouped sections.
func Summarize(base, compare *tempopb.Trace, options SummaryOptions) (*SummaryResult, error) {
	format := options.Format
	if format == "" {
		format = SummaryFormatRanked
	}
	switch format {
	case SummaryFormatAggregate, SummaryFormatRanked, SummaryFormatGrouped:
	default:
		return nil, fmt.Errorf("%q: %w", format, ErrUnsupportedFormat)
	}

	topN := options.TopN
	if topN <= 0 {
		topN = DefaultSummaryTopN
	}

	patch, baseTrace, compareTrace, err := diffTraceInputs(base, compare)
	if err != nil {
		return nil, err
	}

	baseSummary := summarizeNormalizedTrace(baseTrace)
	compareSummary := summarizeNormalizedTrace(compareTrace)

	result := &SummaryResult{
		Version: string(format),
		Base:    baseSummary,
		Compare: compareSummary,
		Trust: TrustSummary{
			MatchedSpanRatio: matchedSpanRatio(patch.Stats.SpanCountA, patch.Stats.SpanCountB, patch.Stats.MatchedSpans),
			StructureOverlap: structureOverlap(patch.Stats.SpanCountA, patch.Stats.SpanCountB, patch.Stats.MatchedSpans),
		},
		Stats: SummaryStats{
			TraceLatencyDeltaMs: compareSummary.DurationMs - baseSummary.DurationMs,
			SpanWorkDeltaMs:     compareSummary.SpanWorkMs - baseSummary.SpanWorkMs,
			SpanCountDelta:      patch.Stats.SpanCountB - patch.Stats.SpanCountA,
			ErrorSpanDelta:      compareSummary.ErrorSpanCount - baseSummary.ErrorSpanCount,
			MatchedSpans:        patch.Stats.MatchedSpans,
			ModifiedSpans:       patch.Stats.ModifiedSpans,
			AddedSpans:          patch.Stats.AddedSpans,
			RemovedSpans:        patch.Stats.RemovedSpans,
			FieldChanges:        patch.Stats.FieldChanges,
			AttributeChanges:    patch.Stats.AttributeChanges,
		},
		Warnings: patch.Warnings,
	}
	result.Signals = summarySignals(result.Stats, topologyChanged(baseTrace, compareTrace))
	switch format {
	case SummaryFormatRanked:
		result.TopChanges = buildTopChanges(patch, topN)
	case SummaryFormatGrouped:
		result.Groups = buildGroups(patch, topN)
	}
	return result, nil
}

func summarizeNormalizedTrace(trace normalizedTrace) TraceSummary {
	summary := TraceSummary{TraceID: trace.meta.TraceID, SpanCount: trace.meta.SpanCount}
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
		// durationNanos returns 0 for unset starts, so they add no fabricated work.
		summary.SpanWorkMs += nanosToMillis(span.snapshot.DurationNanos)
		if isErrorStatus(span.snapshot.Status) {
			summary.ErrorSpanCount++
		}
	}
	if haveStart && lastEnd >= firstStart {
		summary.DurationMs = int64((lastEnd - firstStart) / 1_000_000)
	}
	if len(trace.spans) > 0 {
		summary.RootService = trace.spans[0].ref.Service
		summary.RootName = trace.spans[0].ref.Name
	}
	return summary
}

func topologyChanged(base, compare normalizedTrace) bool {
	matches := matchSpans(base, compare)
	for _, compareSpan := range compare.spans {
		baseSpan, ok := matches[compareSpan.spanID]
		if !ok {
			continue
		}
		if baseSpan.hasParent != compareSpan.hasParent || baseSpan.parentIdentity != compareSpan.parentIdentity {
			return true
		}
	}
	return false
}

func summarySignals(stats SummaryStats, topologyChanged bool) Signals {
	structure := "unchanged"
	if stats.AddedSpans > 0 || stats.RemovedSpans > 0 || topologyChanged {
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
		return "increased"
	case delta < 0:
		return "decreased"
	default:
		return "unchanged"
	}
}

func buildTopChanges(patch *Result, topN int) *TopChanges {
	changes := &TopChanges{}
	for _, modified := range patch.Modified {
		spanChange := summarizeModifiedSpan(modified)
		if spanChange.DurationDeltaMs > 0 {
			changes.Regressions = append(changes.Regressions, spanChange)
		}
		if spanChange.DurationDeltaMs < 0 {
			changes.Improvements = append(changes.Improvements, spanChange)
		}
		if spanChange.StatusBefore != "" || spanChange.StatusAfter != "" {
			changes.Status = append(changes.Status, spanChange)
		}
		for _, change := range modified.Changes {
			if change.Target.Type != TargetAttribute {
				continue
			}
			changes.Attributes = append(changes.Attributes, AttributeChangeSummary{
				Span:   modified.Span,
				Key:    change.Target.Key,
				Op:     change.Op,
				Before: sanitizeJSONValue(change.Before),
				After:  sanitizeJSONValue(change.After),
			})
		}
	}
	for _, added := range patch.Added {
		change := ChangeSummary{Span: spanRefFromSnapshot(added.Span), State: "added", DurationDeltaMs: nanosToMillis(added.Span.DurationNanos)}
		// Surface a newly added error span so its severity is visible alongside the
		// structural change; a non-error status would only add noise here.
		if isErrorStatus(added.Span.Status) {
			change.StatusAfter = added.Span.Status
			changes.Status = append(changes.Status, change)
		}
		changes.Structural = append(changes.Structural, change)
	}
	for _, removed := range patch.Removed {
		change := ChangeSummary{Span: spanRefFromSnapshot(removed.Span), State: "removed", DurationDeltaMs: -nanosToMillis(removed.Span.DurationNanos)}
		if isErrorStatus(removed.Span.Status) {
			change.StatusBefore = removed.Span.Status
			changes.Status = append(changes.Status, change)
		}
		changes.Structural = append(changes.Structural, change)
	}

	sortChangeSummaries(changes.Regressions, false)
	sortChangeSummaries(changes.Improvements, true)
	sortChangeSummariesByAbsDuration(changes.Structural)
	sortStatusChanges(changes.Status)
	sortAttributeChanges(changes.Attributes)

	changes.Regressions = limitChangeSummaries(changes.Regressions, topN)
	changes.Improvements = limitChangeSummaries(changes.Improvements, topN)
	changes.Structural = limitChangeSummaries(changes.Structural, topN)
	changes.Status = limitChangeSummaries(changes.Status, topN)
	changes.Attributes = limitAttributeChanges(changes.Attributes, topN)
	return changes
}

func summarizeModifiedSpan(modified ModifiedSpan) ChangeSummary {
	summary := ChangeSummary{Span: modified.Span, State: "modified"}
	for _, change := range modified.Changes {
		switch {
		case change.Target.Type == TargetField && change.Target.Name == FieldDurationNanos:
			before, beforeOK := change.Before.(int64)
			after, afterOK := change.After.(int64)
			if beforeOK && afterOK {
				summary.DurationDeltaMs = nanosToMillis(after - before)
			}
		case change.Target.Type == TargetField && change.Target.Name == FieldStatus:
			summary.StatusBefore, _ = change.Before.(string)
			summary.StatusAfter, _ = change.After.(string)
		case change.Target.Type == TargetAttribute:
			summary.AttributeChangeCount++
		}
	}
	return summary
}

func buildGroups(patch *Result, topN int) []Group {
	groupsByService := map[string]*Group{}
	for _, modified := range patch.Modified {
		group := groupForService(groupsByService, modified.Span.Service)
		group.ModifiedSpans++
		changeSummary := summarizeModifiedSpan(modified)
		group.SpanWorkDeltaMs += changeSummary.DurationDeltaMs
		if changeSummary.StatusBefore != "" || changeSummary.StatusAfter != "" {
			group.StatusChanges++
			if statusBecameError(changeSummary) {
				group.NewErrorSpans++
			}
			if statusRecoveredFromError(changeSummary) {
				group.ResolvedErrorSpans++
			}
		}
		if changeSummary.AttributeChangeCount > 0 {
			group.AttributeChangedSpans++
		}
	}
	for _, added := range patch.Added {
		group := groupForService(groupsByService, added.Span.Service)
		group.AddedSpans++
		group.SpanWorkDeltaMs += nanosToMillis(added.Span.DurationNanos)
		if isErrorStatus(added.Span.Status) {
			group.StatusChanges++
			group.NewErrorSpans++
		}
	}
	for _, removed := range patch.Removed {
		group := groupForService(groupsByService, removed.Span.Service)
		group.RemovedSpans++
		group.SpanWorkDeltaMs -= nanosToMillis(removed.Span.DurationNanos)
		if isErrorStatus(removed.Span.Status) {
			group.StatusChanges++
			group.ResolvedErrorSpans++
		}
	}

	groups := make([]Group, 0, len(groupsByService))
	for _, group := range groupsByService {
		groups = append(groups, *group)
	}
	sort.SliceStable(groups, func(i, j int) bool {
		a, b := groups[i], groups[j]
		if absInt64(a.SpanWorkDeltaMs) != absInt64(b.SpanWorkDeltaMs) {
			return absInt64(a.SpanWorkDeltaMs) > absInt64(b.SpanWorkDeltaMs)
		}
		// Break span-work ties by significance so new errors are not truncated below
		// recoveries or services that merely sort earlier alphabetically.
		if a.NewErrorSpans != b.NewErrorSpans {
			return a.NewErrorSpans > b.NewErrorSpans
		}
		if a.ResolvedErrorSpans != b.ResolvedErrorSpans {
			return a.ResolvedErrorSpans > b.ResolvedErrorSpans
		}
		if a.StatusChanges != b.StatusChanges {
			return a.StatusChanges > b.StatusChanges
		}
		if ca, cb := groupChangeCount(a), groupChangeCount(b); ca != cb {
			return ca > cb
		}
		return a.ServiceName < b.ServiceName
	})
	if len(groups) > topN {
		return groups[:topN]
	}
	return groups
}

// groupChangeCount totals the structural and field-level changes attributed to a
// service group, used as a tie-breaker when ranking groups for top-N truncation.
func groupChangeCount(g Group) int {
	return g.AddedSpans + g.RemovedSpans + g.ModifiedSpans + g.AttributeChangedSpans
}

// unknownServiceName labels spans with no resolvable service.name. The angle
// brackets keep it from colliding with a real service literally named "unknown".
const unknownServiceName = "<unknown>"

func groupForService(groups map[string]*Group, service string) *Group {
	if service == "" {
		service = unknownServiceName
	}
	group := groups[service]
	if group == nil {
		group = &Group{ServiceName: service}
		groups[service] = group
	}
	return group
}

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

func sortChangeSummaries(changes []ChangeSummary, ascending bool) {
	sort.SliceStable(changes, func(i, j int) bool {
		if changes[i].DurationDeltaMs != changes[j].DurationDeltaMs {
			if ascending {
				return changes[i].DurationDeltaMs < changes[j].DurationDeltaMs
			}
			return changes[i].DurationDeltaMs > changes[j].DurationDeltaMs
		}
		return spanRefLess(changes[i].Span, changes[j].Span)
	})
}

func sortChangeSummariesByAbsDuration(changes []ChangeSummary) {
	sort.SliceStable(changes, func(i, j int) bool {
		if absInt64(changes[i].DurationDeltaMs) != absInt64(changes[j].DurationDeltaMs) {
			return absInt64(changes[i].DurationDeltaMs) > absInt64(changes[j].DurationDeltaMs)
		}
		return spanRefLess(changes[i].Span, changes[j].Span)
	})
}

// sortStatusChanges orders status transitions by severity first so the most
// important transitions (spans that became errors) survive top-N truncation,
// then by absolute duration delta and span ref for deterministic output.
func sortStatusChanges(changes []ChangeSummary) {
	sort.SliceStable(changes, func(i, j int) bool {
		si, sj := statusChangeSeverity(changes[i]), statusChangeSeverity(changes[j])
		if si != sj {
			return si > sj
		}
		if absInt64(changes[i].DurationDeltaMs) != absInt64(changes[j].DurationDeltaMs) {
			return absInt64(changes[i].DurationDeltaMs) > absInt64(changes[j].DurationDeltaMs)
		}
		return spanRefLess(changes[i].Span, changes[j].Span)
	})
}

// statusChangeSeverity ranks a status transition: becoming an error is the most
// important (a regression), recovering from an error is next, and transitions
// that involve neither error state are least important.
func statusChangeSeverity(change ChangeSummary) int {
	switch {
	case statusBecameError(change):
		return 2
	case statusRecoveredFromError(change):
		return 1
	default:
		return 0
	}
}

func statusBecameError(change ChangeSummary) bool {
	return isErrorStatus(change.StatusAfter) && !isErrorStatus(change.StatusBefore)
}

func statusRecoveredFromError(change ChangeSummary) bool {
	return isErrorStatus(change.StatusBefore) && !isErrorStatus(change.StatusAfter)
}

func isErrorStatus(status string) bool {
	return status == "error"
}

func sortAttributeChanges(changes []AttributeChangeSummary) {
	sort.SliceStable(changes, func(i, j int) bool {
		if spanRefLess(changes[i].Span, changes[j].Span) {
			return true
		}
		if spanRefLess(changes[j].Span, changes[i].Span) {
			return false
		}
		return changes[i].Key < changes[j].Key
	})
}

// spanRefLess orders span refs by service, name, kind, then path. Comparing the
// fields directly avoids allocating a formatted key on every comparison.
func spanRefLess(a, b SpanRef) bool {
	if a.Service != b.Service {
		return a.Service < b.Service
	}
	if a.Name != b.Name {
		return a.Name < b.Name
	}
	if a.Kind != b.Kind {
		return a.Kind < b.Kind
	}
	return slices.Compare(a.Path, b.Path) < 0
}

// sanitizeJSONValue replaces non-finite float64 values (NaN, +Inf, -Inf) with
// string placeholders so diff outputs can be encoded with encoding/json, which
// rejects non-finite floats. Arrays and key/value maps are sanitized
// recursively; all other values are returned unchanged.
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

func limitChangeSummaries(changes []ChangeSummary, limit int) []ChangeSummary {
	if len(changes) > limit {
		return changes[:limit]
	}
	return changes
}

func limitAttributeChanges(changes []AttributeChangeSummary, limit int) []AttributeChangeSummary {
	if len(changes) > limit {
		return changes[:limit]
	}
	return changes
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
