package vparquet2

import "github.com/grafana/tempo/pkg/util"

// assignNestedSetModelBounds calculates and assigns the values Span.NestedSetLeft,
// Span.NestedSetRight, and Span.ParentID for all spans in a trace.
// The assignment is skipped when all spans have non-zero left and right bounds. If
// forceAssignment is true, the assignment is never skipped.
func assignNestedSetModelBounds(trace *Trace, forceAssignment bool) {
	var (
		assignmentNeeded bool
		rootSpans        []*wrappedSpan
		spanChildren     = map[[8]byte][]*Span{}
	)

	// Find root spans and map span IDs to child spans
	for _, rs := range trace.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			for i, s := range ss.Spans {
				if s.NestedSetLeft == 0 || s.NestedSetRight == 0 {
					assignmentNeeded = true
				}

				if s.IsRoot() {
					rootSpans = append(rootSpans, &wrappedSpan{span: &ss.Spans[i], id: util.SpanIDToArray(s.SpanID)})
				} else {
					parentID := util.SpanIDToArray(s.ParentSpanID)
					spanChildren[parentID] = append(spanChildren[parentID], &ss.Spans[i])
				}
			}
		}
	}

	if (!assignmentNeeded && !forceAssignment) || len(rootSpans) == 0 {
		return
	}

	// Traverse the tree depth first. When traversing down into the tree, assign NestedSetLeft
	// and assign NestedSetRight when going up.
	var (
		ancestors      util.Stack[*wrappedSpan]
		nestedSetBound int32 = 1
	)

	for _, root := range rootSpans {
		root.span.NestedSetLeft = nestedSetBound
		nestedSetBound++

		ancestors.Reset()
		ancestors.Push(root)

		for !ancestors.IsEmpty() {
			parent, _ := ancestors.Peek()
			children := spanChildren[parent.id]

			if parent.nextChild < len(children) {
				// The current node has children that were not visited: go down to next child

				child := children[parent.nextChild]
				child.ParentID = parent.span.NestedSetLeft // the left bound doubles as numeric span ID
				parent.nextChild++

				child.NestedSetLeft = nestedSetBound
				nestedSetBound++

				ancestors.Push(&wrappedSpan{span: child, id: util.SpanIDToArray(child.SpanID)})
			} else {
				// All children of the current node were visited: go up

				parent.span.NestedSetRight = nestedSetBound
				nestedSetBound++

				ancestors.Pop()
			}
		}
	}
}

// wrappedSpan is used to remember the converted span ID and position of the child that
// needs to be visited next
type wrappedSpan struct {
	span      *Span
	id        [8]byte
	nextChild int
}
