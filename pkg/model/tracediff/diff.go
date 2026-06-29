package tracediff

import (
	"errors"
	"fmt"
	"sort"

	"github.com/grafana/tempo/pkg/tempopb"
)

var (
	ErrNilTrace          = errors.New("trace is nil")
	ErrUnsupportedFormat = errors.New("unsupported trace diff format")
)

// Diff compares base and compare and returns the requested diff format.
func Diff(base, compare *tempopb.Trace, format Format) (*Result, error) {
	if base == nil {
		return nil, fmt.Errorf("base: %w", ErrNilTrace)
	}
	if compare == nil {
		return nil, fmt.Errorf("compare: %w", ErrNilTrace)
	}
	if format != FormatTracePatchV0 {
		return nil, fmt.Errorf("%q: %w", format, ErrUnsupportedFormat)
	}

	baseTrace, baseWarnings := normalizeTrace(base)
	compareTrace, compareWarnings := normalizeTrace(compare)
	matched, modified, added, removed := computeDiff(baseTrace, compareTrace)
	warnings := mergeWarnings(baseWarnings, compareWarnings)

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
	}, nil
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
	usedBaseSpanIDs := make(map[string]struct{})
	baseByID := bucketByIdentity(base)

	for _, compareSpan := range compare.spans {
		baseSpan, ok := findMatch(baseByID[compareSpan.identity()], compareSpan, usedBaseSpanIDs)
		if !ok {
			continue
		}
		baseSpanByCompareSpanID[compareSpan.spanID] = baseSpan
		usedBaseSpanIDs[baseSpan.spanID] = struct{}{}
	}
	return baseSpanByCompareSpanID
}

// findMatch receives base spans with the same identity as compareSpan. Parent
// identity is only a duplicate tie-breaker: if it cannot disambiguate, the first
// unused candidate preserves the ancestor-insert behavior.
func findMatch(baseCandidates []normalizedSpan, compareSpan normalizedSpan, usedBaseSpanIDs map[string]struct{}) (normalizedSpan, bool) {
	var fallback normalizedSpan
	fallbackSet := false
	for _, baseSpan := range baseCandidates {
		if _, ok := usedBaseSpanIDs[baseSpan.spanID]; ok {
			continue
		}
		// Both spans share the same parent identity
		if sameParentIdentity(baseSpan, compareSpan) {
			return baseSpan, true
		}
		// If not we choose the first one as a fallback
		if !fallbackSet {
			fallback = baseSpan
			fallbackSet = true
		}
	}
	return fallback, fallbackSet
}

func sameParentIdentity(a, b normalizedSpan) bool {
	return a.hasParent && b.hasParent && a.parentIdentity == b.parentIdentity
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

func mergeWarnings(warningGroups ...[]Warning) []Warning {
	out := make([]Warning, 0)
	seen := make(map[string]struct{})
	for _, group := range warningGroups {
		for _, warning := range group {
			if _, ok := seen[warning.Code]; ok {
				continue
			}
			seen[warning.Code] = struct{}{}
			out = append(out, warning)
		}
	}
	return out
}

func fieldChanges(base, compare normalizedSpan) []Change {
	changes := make([]Change, 0, 2)
	if !numericClose(float64(base.snapshot.DurationNanos), float64(compare.snapshot.DurationNanos), durationRelTol, durationAbsTolNano) {
		changes = append(changes, Change{
			Op:     OperationModify,
			Target: Target{Type: TargetField, Name: FieldDurationNanos},
			Before: base.snapshot.DurationNanos,
			After:  compare.snapshot.DurationNanos,
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
			changes = append(changes, Change{Op: OperationAdd, Target: target, Before: nil, After: after})
		case !inCompare:
			changes = append(changes, Change{Op: OperationRemove, Target: target, Before: before, After: nil})
		case attributeChanged(key, before, after):
			changes = append(changes, Change{Op: OperationModify, Target: target, Before: before, After: after})
		}
	}
	return changes
}

// attributeChanged reports whether before and after differ. Allow-listed numeric
// magnitude attributes use a relative tolerance when both values are numeric;
// everything else is compared exactly.
func attributeChanged(key string, before, after any) bool {
	if isNumericFuzzyAttribute(key) {
		if a, aok := numericValue(before); aok {
			if b, bok := numericValue(after); bok {
				return !numericClose(a, b, attrRelTol, attrAbsTol)
			}
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
	case int64:
		bv, ok := b.(int64)
		return ok && av == bv
	case float64:
		bv, ok := b.(float64)
		return ok && av == bv
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
