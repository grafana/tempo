package main

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/grafana/tempo/pkg/tempopb"
	v1common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
)

func getAttr(attrs []*v1common.KeyValue, key string) *v1common.AnyValue {
	for _, a := range attrs {
		if a.Key == key {
			return a.Value
		}
	}
	return nil
}

func attrValueEqual(a, b *v1common.AnyValue) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	switch av := a.Value.(type) {
	case *v1common.AnyValue_StringValue:
		if bv, ok := b.Value.(*v1common.AnyValue_StringValue); ok {
			return av.StringValue == bv.StringValue
		}
		return false
	case *v1common.AnyValue_IntValue:
		if bv, ok := b.Value.(*v1common.AnyValue_IntValue); ok {
			return av.IntValue == bv.IntValue
		}
		return false
	default:
		return util.StringifyAnyValue(a) == util.StringifyAnyValue(b)
	}
}

// compareAttrs returns an error if the two attribute slices do not have the same set of keys and values.
func compareAttrs(exp, ret []*v1common.KeyValue) error {
	for _, a := range exp {
		retVal := getAttr(ret, a.Key)
		if retVal == nil {
			return fmt.Errorf("missing attribute %s", a.Key)
		}
		if !attrValueEqual(a.Value, retVal) {
			return fmt.Errorf("attribute %s mismatch: expected %q, got %q", a.Key, util.StringifyAnyValue(a.Value), util.StringifyAnyValue(retVal))
		}
	}
	for _, a := range ret {
		expVal := getAttr(exp, a.Key)
		if expVal == nil {
			return fmt.Errorf("unexpected attribute %s", a.Key)
		}
	}
	return nil
}

// sortTraceForComparison normalizes trace order so that comparison is independent of backend
// reordering: ResourceSpans, ScopeSpans, Spans, and Events by span start time / event time;
// attributes by key at resource, span, and event levels.
func sortTraceForComparison(t *tempopb.Trace) {
	for _, b := range t.ResourceSpans {
		for _, ss := range b.ScopeSpans {
			sort.SliceStable(ss.Spans, func(i, j int) bool {
				return compareSpansForSort(ss.Spans[i], ss.Spans[j])
			})
		}
		sort.SliceStable(b.ScopeSpans, func(i, j int) bool {
			return compareScopeSpansForSort(b.ScopeSpans[i], b.ScopeSpans[j])
		})
	}
	sort.SliceStable(t.ResourceSpans, func(i, j int) bool {
		return compareBatchesForSort(t.ResourceSpans[i], t.ResourceSpans[j])
	})
	for _, b := range t.ResourceSpans {
		if b.Resource != nil {
			sort.Slice(b.Resource.Attributes, func(i, j int) bool {
				return b.Resource.Attributes[i].Key < b.Resource.Attributes[j].Key
			})
		}
		for _, ss := range b.ScopeSpans {
			for _, sp := range ss.Spans {
				sort.Slice(sp.Attributes, func(i, j int) bool {
					return sp.Attributes[i].Key < sp.Attributes[j].Key
				})
				sort.Slice(sp.Events, func(i, j int) bool {
					return sp.Events[i].TimeUnixNano < sp.Events[j].TimeUnixNano
				})
				for _, ev := range sp.Events {
					sort.Slice(ev.Attributes, func(i, j int) bool {
						return ev.Attributes[i].Key < ev.Attributes[j].Key
					})
				}
			}
		}
	}
}

func compareBatchesForSort(a, b *v1.ResourceSpans) bool {
	if len(a.ScopeSpans) > 0 && len(b.ScopeSpans) > 0 {
		return compareScopeSpansForSort(a.ScopeSpans[0], b.ScopeSpans[0])
	}
	return false
}

func compareScopeSpansForSort(a, b *v1.ScopeSpans) bool {
	if len(a.Spans) > 0 && len(b.Spans) > 0 {
		return compareSpansForSort(a.Spans[0], b.Spans[0])
	}
	return false
}

func compareSpansForSort(a, b *v1.Span) bool {
	if a.StartTimeUnixNano == b.StartTimeUnixNano {
		return bytes.Compare(a.SpanId, b.SpanId) == -1
	}
	return a.StartTimeUnixNano < b.StartTimeUnixNano
}

// cloneTrace returns a deep copy of the trace so we can sort without mutating the caller's data.
func cloneTrace(t *tempopb.Trace) *tempopb.Trace {
	data, err := t.Marshal()
	if err != nil {
		panic(err)
	}
	out := &tempopb.Trace{}
	if err := out.Unmarshal(data); err != nil {
		panic(err)
	}
	return out
}

// VerifyTraceContent verifies that all attributes at resource, span, and event levels in retrieved
// match the expected trace. Returns an error on first mismatch.
// Both traces are sorted (batches, spans, events, attributes) so that backend reordering does not cause false mismatches.
func VerifyTraceContent(want, got *tempopb.Trace) error {
	want = cloneTrace(want)
	got = cloneTrace(got)
	sortTraceForComparison(want)
	sortTraceForComparison(got)

	if len(want.ResourceSpans) != len(got.ResourceSpans) {
		return fmt.Errorf("resource span count mismatch: expected %d, got %d", len(want.ResourceSpans), len(got.ResourceSpans))
	}
	for rsIdx, wantRS := range want.ResourceSpans {
		gotRS := got.ResourceSpans[rsIdx]
		if wantRS.Resource != nil && gotRS.Resource != nil {
			if err := compareAttrs(wantRS.Resource.Attributes, gotRS.Resource.Attributes); err != nil {
				return fmt.Errorf("resource[%d]: %w", rsIdx, err)
			}
		}
		if len(wantRS.ScopeSpans) != len(gotRS.ScopeSpans) {
			return fmt.Errorf("resource[%d] scope span count mismatch", rsIdx)
		}
		for ssIdx, wantSS := range wantRS.ScopeSpans {
			gotSS := gotRS.ScopeSpans[ssIdx]
			if len(wantSS.Spans) != len(gotSS.Spans) {
				return fmt.Errorf("resource[%d] scope[%d] span count mismatch", rsIdx, ssIdx)
			}
			for spIdx, wantSp := range wantSS.Spans {
				gotSp := gotSS.Spans[spIdx]
				if err := compareAttrs(wantSp.Attributes, gotSp.Attributes); err != nil {
					return fmt.Errorf("resource[%d] scope[%d] span[%d]: %w", rsIdx, ssIdx, spIdx, err)
				}
				if len(wantSp.Events) != len(gotSp.Events) {
					return fmt.Errorf("resource[%d] scope[%d] span[%d] event count mismatch", rsIdx, ssIdx, spIdx)
				}
				for evIdx, wantEv := range wantSp.Events {
					gotEv := gotSp.Events[evIdx]
					if err := compareAttrs(wantEv.Attributes, gotEv.Attributes); err != nil {
						return fmt.Errorf("resource[%d] scope[%d] span[%d] event[%d]: %w", rsIdx, ssIdx, spIdx, evIdx, err)
					}
				}
			}
		}
	}
	return nil
}
