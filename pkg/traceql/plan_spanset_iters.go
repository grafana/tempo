package traceql

import (
	"bytes"
	"context"
	"fmt"
)

// FilterSpansetIter returns a SpansetIterator that applies filter to every
// spanset produced by child, forwarding only non-empty results.
// It reuses the existing SpansetFilter.evaluate logic.
func FilterSpansetIter(filter *SpansetFilter, child SpansetIterator) SpansetIterator {
	return &filterSpansetIter{filter: filter, child: child}
}

type filterSpansetIter struct {
	filter *SpansetFilter
	child  SpansetIterator
}

func (f *filterSpansetIter) Next(ctx context.Context) (*Spanset, error) {
	for {
		ss, err := f.child.Next(ctx)
		if err != nil {
			return nil, err
		}
		if ss == nil {
			return nil, nil
		}
		result, err := f.filter.evaluate([]*Spanset{ss})
		if err != nil {
			return nil, err
		}
		if len(result) > 0 {
			return result[0], nil
		}
		ss.Release()
	}
}

func (f *filterSpansetIter) Close() { f.child.Close() }

// GroupBySpansetIter returns a SpansetIterator that groups spans from each
// child spanset by the given FieldExpression, yielding one spanset per group.
// It reuses GroupOperation.evaluate logic.
func GroupBySpansetIter(by FieldExpression, child SpansetIterator) SpansetIterator {
	return &groupBySpansetIter{
		op:    newGroupOperation(by),
		child: child,
	}
}

type groupBySpansetIter struct {
	op    GroupOperation
	child SpansetIterator
	buf   []*Spanset
	idx   int
}

func (g *groupBySpansetIter) Next(ctx context.Context) (*Spanset, error) {
	for {
		// Drain the local buffer first.
		if g.idx < len(g.buf) {
			ss := g.buf[g.idx]
			g.idx++
			return ss, nil
		}
		// Fetch next raw spanset and group it.
		ss, err := g.child.Next(ctx)
		if err != nil {
			return nil, err
		}
		if ss == nil {
			return nil, nil
		}
		groups, err := g.op.evaluate([]*Spanset{ss})
		if err != nil {
			return nil, err
		}
		g.buf = groups
		g.idx = 0
	}
}

func (g *groupBySpansetIter) Close() { g.child.Close() }

// CoalesceSpansetIter returns a SpansetIterator that merges all spans from
// consecutive child spansets for the same trace into one spanset.
// It reuses CoalesceOperation.evaluate logic.
func CoalesceSpansetIter(child SpansetIterator) SpansetIterator {
	return &coalesceSpansetIter{child: child}
}

type coalesceSpansetIter struct {
	child   SpansetIterator
	pending *Spanset
}

func (c *coalesceSpansetIter) Next(ctx context.Context) (*Spanset, error) {
	// Collect all spansets for one trace (they share the same TraceID).
	var batch []*Spanset
	// Consume any spanset buffered from the previous call.
	if c.pending != nil {
		batch = append(batch, c.pending)
		c.pending = nil
	}
	for {
		ss, err := c.child.Next(ctx)
		if err != nil {
			return nil, err
		}
		if ss == nil {
			break
		}
		if len(batch) > 0 && !bytes.Equal(batch[0].TraceID, ss.TraceID) {
			// Different trace — buffer for the next call.
			c.pending = ss
			break
		}
		batch = append(batch, ss)
	}
	if len(batch) == 0 {
		return nil, nil
	}
	result, err := CoalesceOperation{}.evaluate(batch)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result[0], nil
}

func (c *coalesceSpansetIter) Close() { c.child.Close() }

// StructuralOpSpansetIter returns a SpansetIterator that merge-joins two
// independently-scanned SpansetIterators by trace ID and evaluates the given
// structural relationship for each matching pair.
// Reuses the Span.DescendantOf / ChildOf / SiblingOf interface methods.
func StructuralOpSpansetIter(op StructuralOp, left, right SpansetIterator) SpansetIterator {
	return &structuralOpSpansetIter{op: op, left: left, right: right}
}

type structuralOpSpansetIter struct {
	op          StructuralOp
	left, right SpansetIterator
	// pending: one buffered spanset from the side that is ahead after an
	// out-of-sync advance.
	leftPending  *Spanset
	rightPending *Spanset
	// reusable buffer for relationship functions
	matchBuf []Span
}

func (s *structuralOpSpansetIter) next(ctx context.Context, iter SpansetIterator, pending **Spanset) (*Spanset, error) {
	if *pending != nil {
		ss := *pending
		*pending = nil
		return ss, nil
	}
	return iter.Next(ctx)
}

func (s *structuralOpSpansetIter) Next(ctx context.Context) (*Spanset, error) {
	for {
		lss, err := s.next(ctx, s.left, &s.leftPending)
		if err != nil {
			return nil, err
		}
		if lss == nil {
			return nil, nil // left exhausted → done
		}

		rss, err := s.next(ctx, s.right, &s.rightPending)
		if err != nil {
			return nil, err
		}
		if rss == nil {
			return nil, nil // right exhausted → done
		}

		cmp := bytes.Compare(lss.TraceID, rss.TraceID)
		switch {
		case cmp < 0:
			// Left is behind — discard left, keep right pending.
			lss.Release()
			s.rightPending = rss
			continue
		case cmp > 0:
			// Right is behind — discard right, keep left pending.
			rss.Release()
			s.leftPending = lss
			continue
		}

		// Same trace — evaluate structural relationship.
		matching, err := s.evalRelationship(lss, rss)
		if err != nil {
			return nil, err
		}
		lss.Release()
		if len(matching) == 0 {
			rss.Release()
			continue
		}
		out := rss.clone()
		out.Spans = append([]Span(nil), matching...)
		return out, nil
	}
}

// evalRelationship applies the StructuralOp relationship function.
// It mirrors the logic in SpansetOperation.evaluate / joinSpansets, calling
// the same exported Span interface methods.
func (s *structuralOpSpansetIter) evalRelationship(lss, rss *Spanset) ([]Span, error) {
	if len(rss.Spans) == 0 {
		return nil, nil
	}
	lspans := lss.Spans
	rspans := rss.Spans
	s.matchBuf = s.matchBuf[:0]

	switch s.op {
	case StructuralOpDescendant: // >>
		return rspans[0].DescendantOf(lspans, rspans, false, false, false, s.matchBuf), nil
	case StructuralOpAncestor: // <<
		return rspans[0].DescendantOf(lspans, rspans, false, true, false, s.matchBuf), nil
	case StructuralOpChild: // >
		return rspans[0].ChildOf(lspans, rspans, false, false, false, s.matchBuf), nil
	case StructuralOpParent: // <
		return rspans[0].ChildOf(lspans, rspans, false, true, false, s.matchBuf), nil
	case StructuralOpSibling: // ~
		return rspans[0].SiblingOf(lspans, rspans, false, false, s.matchBuf), nil
	default:
		return nil, fmt.Errorf("plan_spanset_iters: unsupported structural op %d", s.op)
	}
}

func (s *structuralOpSpansetIter) Close() {
	s.left.Close()
	s.right.Close()
}
