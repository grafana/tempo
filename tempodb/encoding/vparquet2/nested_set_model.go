package vparquet2

import "github.com/grafana/tempo/pkg/util"

// TODO should root spans and children be sorted before traversal to make the outcome more predictable
func assignNestedSetModelBounds(trace *Trace) {
	var (
		assignmentNeeded bool
		rootSpans        []wrappedSpan
		spanChildren     = map[uint64][]*Span{}
	)

	// find root spans and map span IDs to child spans
	for _, rs := range trace.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			for i, s := range ss.Spans {
				if s.NestedSetLeft == 0 || s.NestedSetRight == 0 {
					assignmentNeeded = true
				}

				if len(s.ParentSpanID) == 0 {
					rootSpans = append(rootSpans, wrappedSpan{span: &ss.Spans[i], id: util.SpanIDToUint64(s.SpanID)})
				} else {
					parentID := util.SpanIDToUint64(s.ParentSpanID)
					spanChildren[parentID] = append(spanChildren[parentID], &ss.Spans[i])
				}
			}
		}
	}

	if len(rootSpans) == 0 || !assignmentNeeded {
		return
	}

	// traverse the tree
	var (
		ancestors      stack[wrappedSpan]
		nestedSetBound int32 = 1
	)

	for i, root := range rootSpans {
		root.span.NestedSetLeft = nestedSetBound
		nestedSetBound++

		ancestors.reset()
		ancestors.push(&rootSpans[i])

		for !ancestors.isEmpty() {
			parent := ancestors.peek()
			children := spanChildren[parent.id]

			if parent.nextChild < len(children) {
				child := children[parent.nextChild]
				child.ParentID = parent.span.NestedSetLeft // the left bound doubles as numeric span ID
				parent.nextChild++

				child.NestedSetLeft = nestedSetBound
				nestedSetBound++

				ancestors.push(&wrappedSpan{span: child, id: util.SpanIDToUint64(child.SpanID)})
			} else {
				parent.span.NestedSetRight = nestedSetBound
				nestedSetBound++

				ancestors.pop()
			}
		}
	}
}

type wrappedSpan struct {
	span      *Span
	id        uint64
	nextChild int
}

type stack[T any] []*T

func (ss *stack[T]) push(element *T) {
	*ss = append(*ss, element)
}

func (ss *stack[T]) peek() *T {
	if len(*ss) == 0 {
		return nil
	}
	return (*ss)[len(*ss)-1]
}

func (ss *stack[T]) pop() *T {
	if len(*ss) == 0 {
		return nil
	}
	i := len(*ss) - 1
	s := (*ss)[i]
	*ss = (*ss)[:i]
	return s
}

func (ss *stack[T]) isEmpty() bool {
	return len(*ss) == 0
}

func (ss *stack[T]) reset() {
	*ss = (*ss)[:0]
}
