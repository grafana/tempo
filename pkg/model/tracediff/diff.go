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

	baseTrace := normalizeTrace(base)
	compareTrace := normalizeTrace(compare)
	matched, modified, added, removed := diffSpanPresence(baseTrace, compareTrace)

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
		Warnings: []Warning{},
	}, nil
}

func diffSpanPresence(base, compare normalizedTrace) (int, []ModifiedSpan, []SpanChange, []SpanChange) {
	baseByKey := indexByMatchKey(base)
	compareByKey := indexByMatchKey(compare)

	var matched int
	modified := make([]ModifiedSpan, 0)
	added := make([]SpanChange, 0)
	removed := make([]SpanChange, 0)
	for _, span := range compare.spans {
		baseSpan, ok := baseByKey[span.matchKey]
		if ok {
			matched++
			changes := fieldChanges(baseSpan, span)
			changes = append(changes, attributeChanges(baseSpan, span)...)
			if len(changes) > 0 {
				modified = append(modified, ModifiedSpan{
					Span:    span.ref,
					Changes: changes,
				})
			}
			continue
		}
		added = append(added, SpanChange{
			Target: addedSpanTarget(span.snapshot.Path),
			Span:   span.snapshot,
		})
	}
	for _, span := range base.spans {
		if _, ok := compareByKey[span.matchKey]; ok {
			continue
		}
		removed = append(removed, SpanChange{
			Target: SpanTarget{Type: TargetSpan, Path: span.snapshot.Path},
			Span:   span.snapshot,
		})
	}
	return matched, modified, added, removed
}

func fieldChanges(base, compare normalizedSpan) []Change {
	changes := make([]Change, 0, 2)
	if base.snapshot.DurationMs != compare.snapshot.DurationMs {
		changes = append(changes, Change{
			Op:     OperationModify,
			Target: Target{Type: TargetField, Name: "duration_ms"},
			Before: base.snapshot.DurationMs,
			After:  compare.snapshot.DurationMs,
		})
	}
	if base.snapshot.Status != compare.snapshot.Status {
		changes = append(changes, Change{
			Op:     OperationModify,
			Target: Target{Type: TargetField, Name: "status"},
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
		target := Target{Type: TargetAttribute, Scope: "span", Key: key}

		switch {
		case !inBase:
			changes = append(changes, Change{Op: OperationAdd, Target: target, Before: nil, After: after})
		case !inCompare:
			changes = append(changes, Change{Op: OperationRemove, Target: target, Before: before, After: nil})
		case !valuesEqual(before, after):
			changes = append(changes, Change{Op: OperationModify, Target: target, Before: before, After: after})
		}
	}
	return changes
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

func indexByMatchKey(trace normalizedTrace) map[spanMatchKey]normalizedSpan {
	out := make(map[spanMatchKey]normalizedSpan, len(trace.spans))
	for _, span := range trace.spans {
		out[span.matchKey] = span
	}
	return out
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
