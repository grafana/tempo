package tracediff

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/grafana/tempo/pkg/tempopb"
)

var (
	ErrNilTrace          = errors.New("trace is nil")
	ErrUnsupportedFormat = errors.New("unsupported trace diff format")
)

// Diff compares base and compare and returns the requested diff format.
func Diff(base, compare *tempopb.Trace, format Format) (*Result, error) {
	if format != FormatTracePatchV0 {
		return nil, fmt.Errorf("%q: %w", format, ErrUnsupportedFormat)
	}
	result, _, _, err := diffTraceInputs(base, compare)
	return result, err
}

func diffTraceInputs(base, compare *tempopb.Trace) (*Result, normalizedTrace, normalizedTrace, error) {
	if base == nil {
		return nil, normalizedTrace{}, normalizedTrace{}, fmt.Errorf("base: %w", ErrNilTrace)
	}
	if compare == nil {
		return nil, normalizedTrace{}, normalizedTrace{}, fmt.Errorf("compare: %w", ErrNilTrace)
	}

	baseTrace, baseWarnings := normalizeTraceForSide(base, "base")
	compareTrace, compareWarnings := normalizeTraceForSide(compare, "compare")
	matched, modified, added, removed := computeDiff(baseTrace, compareTrace)
	warnings := mergeWarnings(baseWarnings, compareWarnings)
	if ambiguousGroups := ambiguousSpanGroupCount(baseTrace, compareTrace); ambiguousGroups > 0 {
		warnings = append(warnings, Warning{
			Code:    WarningAmbiguousSpanMatch,
			Message: fmt.Sprintf("%d duplicate logical span group(s) may match ambiguously; matching minimizes changes, but instance-level transitions may not identify the same physical operation", ambiguousGroups),
		})
	}

	return &Result{
		Version: VersionTracePatchV0,
		Base:    baseTrace.meta,
		Compare: compareTrace.meta,
		Stats: Stats{
			SpanCountA:       baseTrace.meta.SpanCount,
			SpanCountB:       compareTrace.meta.SpanCount,
			MatchedSpans:     matched,
			ModifiedSpans:    len(modified),
			AddedSpans:       len(added),
			RemovedSpans:     len(removed),
			FieldChanges:     countChanges(modified, TargetField),
			AttributeChanges: countChanges(modified, TargetAttribute),
		},
		Modified: modified,
		Added:    added,
		Removed:  removed,
		Warnings: warnings,
	}, baseTrace, compareTrace, nil
}

func computeDiff(base, compare normalizedTrace) (int, []ModifiedSpan, []SpanChange, []SpanChange) {
	baseSpanByCompareSpanID := matchSpans(base, compare)
	matchedBaseSpanIDs := make(map[string]struct{}, len(baseSpanByCompareSpanID))

	modified := make([]ModifiedSpan, 0)
	added := make([]SpanChange, 0)
	removed := make([]SpanChange, 0)

	for _, span := range compare.spans {
		baseSpan, ok := baseSpanByCompareSpanID[span.spanID]
		if !ok {
			added = append(added, SpanChange{Target: addedSpanTarget(span.snapshot.Path), Span: span.snapshot})
			continue
		}

		matchedBaseSpanIDs[baseSpan.spanID] = struct{}{}
		changes := fieldChanges(baseSpan, span)
		changes = append(changes, attributeChanges(baseSpan, span)...)
		if len(changes) > 0 {
			modified = append(modified, ModifiedSpan{Span: span.ref, Changes: changes})
		}
	}

	for _, span := range base.spans {
		if _, ok := matchedBaseSpanIDs[span.spanID]; ok {
			continue
		}
		removed = append(removed, SpanChange{
			Target: SpanTarget{Type: TargetSpan, Path: span.snapshot.Path},
			Span:   span.snapshot,
		})
	}

	return len(matchedBaseSpanIDs), modified, added, removed
}

// matchSpans decides which base span, if any, matches each compare span.
func matchSpans(base, compare normalizedTrace) map[string]normalizedSpan {
	baseSpanByCompareSpanID := make(map[string]normalizedSpan)
	baseByID := bucketByIdentity(base)
	compareByID := bucketByIdentity(compare)
	for identity, compareCandidates := range compareByID {
		for compareSpanID, baseSpan := range matchSpanGroup(baseByID[identity], compareCandidates) {
			baseSpanByCompareSpanID[compareSpanID] = baseSpan
		}
	}
	return baseSpanByCompareSpanID
}

// matchSpanGroup pairs a logical-identity bucket in phases. Matching all
// unchanged spans before changed spans makes the result independent of sibling
// order while parent-first phases preserve branch identity where it is known.
func matchSpanGroup(baseCandidates, compareCandidates []normalizedSpan) map[string]normalizedSpan {
	matchCount := min(len(baseCandidates), len(compareCandidates))
	matches := make(map[string]normalizedSpan, matchCount)
	usedBaseSpanIDs := make(map[string]struct{}, matchCount)

	pair := func(eligible func(normalizedSpan, normalizedSpan) bool) {
		for _, compareSpan := range compareCandidates {
			if _, ok := matches[compareSpan.spanID]; ok {
				continue
			}
			for _, baseSpan := range baseCandidates {
				if _, ok := usedBaseSpanIDs[baseSpan.spanID]; ok || !eligible(baseSpan, compareSpan) {
					continue
				}
				matches[compareSpan.spanID] = baseSpan
				usedBaseSpanIDs[baseSpan.spanID] = struct{}{}
				break
			}
		}
	}

	pair(func(base, compare normalizedSpan) bool {
		return sameParentIdentity(base, compare) && spansEquivalentForMatching(base, compare)
	})
	pair(func(base, compare normalizedSpan) bool {
		return sameParentIdentity(base, compare) && base.snapshot.Status == compare.snapshot.Status
	})
	pair(sameParentIdentity)
	pair(spansEquivalentForMatching)
	pair(func(base, compare normalizedSpan) bool {
		return base.snapshot.Status == compare.snapshot.Status
	})
	pair(func(normalizedSpan, normalizedSpan) bool { return true })
	return matches
}

func sameParentIdentity(a, b normalizedSpan) bool {
	return a.hasParent == b.hasParent && (!a.hasParent || a.parentIdentity == b.parentIdentity)
}

func spansEquivalentForMatching(base, compare normalizedSpan) bool {
	baseDuration, compareDuration := durationChangeValues(base, compare)
	if durationChanged(baseDuration, compareDuration) || base.snapshot.Status != compare.snapshot.Status {
		return false
	}
	if len(base.spanAttrs) != len(compare.spanAttrs) {
		return false
	}
	for key, before := range base.spanAttrs {
		after, ok := compare.spanAttrs[key]
		if !ok || attributeChanged(key, before, after) {
			return false
		}
	}
	return true
}

// bucketByIdentity groups spans by their flat identity.
func bucketByIdentity(trace normalizedTrace) map[spanLogicalKey][]normalizedSpan {
	out := make(map[spanLogicalKey][]normalizedSpan, len(trace.spans))
	for _, s := range trace.spans {
		id := s.identity()
		out[id] = append(out[id], s)
	}
	return out
}

type spanParentIdentity struct {
	identity  spanLogicalKey
	hasParent bool
}

// ambiguousSpanGroupCount counts logical identities whose spans cannot be
// paired uniquely using parent identity. Exact-content matching still avoids
// spurious changes where possible, but consumers must not treat the remaining
// pairings as certain instance-to-instance transitions.
func ambiguousSpanGroupCount(base, compare normalizedTrace) int {
	baseCounts := spanParentCounts(base)
	compareCounts := spanParentCounts(compare)

	identities := make(map[spanLogicalKey]struct{}, len(baseCounts)+len(compareCounts))
	for identity := range baseCounts {
		identities[identity] = struct{}{}
	}
	for identity := range compareCounts {
		identities[identity] = struct{}{}
	}

	ambiguous := 0
	for identity := range identities {
		baseParents := baseCounts[identity]
		compareParents := compareCounts[identity]
		remainingBase := 0
		remainingCompare := 0
		isAmbiguous := false
		for parent, baseCount := range baseParents {
			compareCount := compareParents[parent]
			paired := min(baseCount, compareCount)
			if paired > 0 && (baseCount > 1 || compareCount > 1) {
				isAmbiguous = true
			}
			remainingBase += baseCount - paired
			remainingCompare += compareCount - paired
		}
		for parent, compareCount := range compareParents {
			if _, ok := baseParents[parent]; !ok {
				remainingCompare += compareCount
			}
		}
		if remainingBase > 0 && remainingCompare > 0 && (remainingBase > 1 || remainingCompare > 1) {
			isAmbiguous = true
		}
		if isAmbiguous {
			ambiguous++
		}
	}
	return ambiguous
}

func spanParentCounts(trace normalizedTrace) map[spanLogicalKey]map[spanParentIdentity]int {
	counts := make(map[spanLogicalKey]map[spanParentIdentity]int)
	for _, span := range trace.spans {
		identity := span.identity()
		parents := counts[identity]
		if parents == nil {
			parents = make(map[spanParentIdentity]int)
			counts[identity] = parents
		}
		parents[spanParentIdentity{identity: span.parentIdentity, hasParent: span.hasParent}]++
	}
	return counts
}

func mergeWarnings(warningGroups ...[]Warning) []Warning {
	out := make([]Warning, 0)
	seen := make(map[string]struct{})
	for _, group := range warningGroups {
		for _, warning := range group {
			key := warningDedupKey(warning)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, warning)
		}
	}
	return out
}

func warningDedupKey(warning Warning) string {
	switch warning.Code {
	case WarningInvalidDuration, WarningZeroSpanTrace, WarningDuplicateSpanID:
		// The message carries the side; keying on it keeps both sides visible.
		return warning.Code + "\x00" + warning.Message
	default:
		return warning.Code
	}
}

func fieldChanges(base, compare normalizedSpan) []Change {
	changes := make([]Change, 0, 2)
	baseDuration, compareDuration := durationChangeValues(base, compare)
	if durationChanged(baseDuration, compareDuration) {
		changes = append(changes, Change{
			Op:     OperationModify,
			Target: Target{Type: TargetField, Name: FieldDurationNanos},
			Before: baseDuration,
			After:  compareDuration,
		})
	}
	if base.snapshot.Status != compare.snapshot.Status {
		changes = append(changes, Change{
			Op:     OperationModify,
			Target: Target{Type: TargetField, Name: FieldStatus},
			Before: base.snapshot.Status,
			After:  compare.snapshot.Status,
		})
	}
	return changes
}

func durationChangeValues(base, compare normalizedSpan) (any, any) {
	if !base.durationValid && !compare.durationValid {
		return nil, nil
	}
	var before any
	if base.durationValid {
		before = base.snapshot.DurationNanos
	}
	var after any
	if compare.durationValid {
		after = compare.snapshot.DurationNanos
	}
	return before, after
}

func durationChanged(before, after any) bool {
	beforeNanos, beforeOK := before.(int64)
	afterNanos, afterOK := after.(int64)
	if beforeOK && afterOK {
		return !numericClose(float64(beforeNanos), float64(afterNanos), durationRelTolerance, durationAbsTolerance)
	}
	return !valuesEqual(before, after)
}

func attributeChanges(base, compare normalizedSpan) []Change {
	changes := make([]Change, 0)

	baseAttrs := base.spanAttrs
	compareAttrs := compare.spanAttrs
	for _, key := range sortedAttributeKeys(baseAttrs, compareAttrs) {
		before, inBase := baseAttrs[key]
		after, inCompare := compareAttrs[key]
		target := Target{Type: TargetAttribute, Scope: ScopeSpan, Key: key}

		switch {
		case !inBase:
			changes = append(changes, Change{Op: OperationAdd, Target: target, Before: nil, After: sanitizeJSONValue(after)})
		case !inCompare:
			changes = append(changes, Change{Op: OperationRemove, Target: target, Before: sanitizeJSONValue(before), After: nil})
		case attributeChanged(key, before, after):
			changes = append(changes, Change{Op: OperationModify, Target: target, Before: sanitizeJSONValue(before), After: sanitizeJSONValue(after)})
		}
	}
	return changes
}

// attributeChanged reports whether before and after differ. Allow-listed numeric
// magnitude attributes use a relative tolerance when both values are numeric;
// everything else is compared exactly.
func attributeChanged(key string, before, after any) bool {
	if isNumericFuzzyAttribute(key) {
		a, aOK := numericValue(before)
		b, bOK := numericValue(after)
		if aOK && bOK {
			return !numericClose(a, b, attrRelTolerance, attrAbsTolerance)
		}
	}
	return !valuesEqual(before, after)
}

func valuesEqual(a, b any) bool {
	switch av := a.(type) {
	case nil:
		return b == nil
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case int64, float64:
		// if both a and b are int64 do exact comparison, otherwise use numericValue() to convert to float64
		if ai, ok := a.(int64); ok {
			if bi, ok := b.(int64); ok {
				return ai == bi
			}
		}
		af, aOK := numericValue(a)
		bf, bOK := numericValue(b)
		if !aOK || !bOK {
			return false
		}
		// NaN != NaN under ==, but two attributes that are both NaN should be
		// treated as unchanged rather than reported as a spurious modification.
		if math.IsNaN(af) && math.IsNaN(bf) {
			return true
		}
		return af == bf
	case []byte:
		bv, ok := b.([]byte)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if av[i] != bv[i] {
				return false
			}
		}
		return true
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !valuesEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for key, val := range av {
			other, ok := bv[key]
			if !ok || !valuesEqual(val, other) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func sortedAttributeKeys(base, compare map[string]any) []string {
	seen := make(map[string]struct{}, len(base)+len(compare))
	keys := make([]string, 0, len(base)+len(compare))
	for key := range base {
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	for key := range compare {
		if _, ok := seen[key]; ok {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func countChanges(modified []ModifiedSpan, targetType TargetType) int {
	var count int
	for _, span := range modified {
		for _, change := range span.Changes {
			if change.Target.Type == targetType {
				count++
			}
		}
	}
	return count
}

func addedSpanTarget(path []int) SpanTarget {
	index := 0
	if len(path) > 0 {
		index = path[len(path)-1]
	}
	return SpanTarget{
		Type:       TargetSpan,
		ParentPath: parentPath(path),
		Index:      &index,
	}
}

func parentPath(path []int) []int {
	if len(path) == 0 {
		return nil
	}
	out := make([]int, len(path)-1)
	copy(out, path[:len(path)-1])
	return out
}
