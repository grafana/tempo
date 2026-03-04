package traceql

import (
	"bytes"
	"context"
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

// RelationSpansetIter evaluates a SpansetOperation in-memory for each spanset
// produced by child, forwarding non-empty results.
// It reuses SpansetOperation.evaluate, matching the current engine strategy.
func RelationSpansetIter(expr SpansetOperation, child SpansetIterator) SpansetIterator {
	return &relationSpansetIter{expr: expr, child: child}
}

type relationSpansetIter struct {
	expr  SpansetOperation
	child SpansetIterator
}

func (e *relationSpansetIter) Next(ctx context.Context) (*Spanset, error) {
	for {
		ss, err := e.child.Next(ctx)
		if err != nil || ss == nil {
			return nil, err
		}
		result, err := e.expr.evaluate([]*Spanset{ss})
		if err != nil {
			return nil, err
		}
		if len(result) > 0 {
			return result[0], nil
		}
		ss.Release()
	}
}

func (e *relationSpansetIter) Close() { e.child.Close() }

