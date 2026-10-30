package parquetquery

import (
	pq "github.com/parquet-go/parquet-go"
)

// PredicateStats counts the chunk/page/value decisions the iterator makes while
// applying its predicate. Read them with SyncIterator.Stats; Close exports them as
// metrics.
type PredicateStats struct {
	InspectedColumnChunks int64
	KeptColumnChunks      int64
	InspectedPages        int64
	KeptPages             int64
	InspectedValues       int64
	KeptValues            int64
}

// predicateNullValue returns the null pq.Value fed to KeepValue when deciding
// whether null rows in a chunk/page could match. It must be the same null
// representation the per-row value reader produces so the skip decision agrees
// with per-row evaluation.
func predicateNullValue() pq.Value { return pq.Value{} }

// keepColumnChunk reports whether cc may hold a value c.filter keeps, so a chunk that
// can't is skipped before any of its pages is read. Counts into c.stats.
//
// The footer's chunk statistics come first: they need no read, and since the writer
// builds them from the same page values as the column index, the index is only read
// for a chunk without them.
//
// The chunk is kept when:
//   - there is no filter;
//   - the footer has nulls the filter keeps, or bounds that pass KeepRange, and the
//     chunk is not dictionary-encoded;
//   - it is dictionary-encoded and a dictionary entry matches, or the filter keeps
//     nulls and the footer or column index shows the chunk has some;
//   - the filter doesn't support column index pruning, and its KeepRange keeps
//     regardless of bounds (always, except a CallbackPredicate whose callback is done);
//   - it has no column index;
//   - the column index shows a page with nulls the filter keeps, or a page whose
//     bounds pass KeepRange.
func (c *SyncIterator) keepColumnChunk(cc *ColumnChunkHelper) (keep bool) {
	c.dictPageReader.reset()
	if c.filter == nil {
		return true // no predicate: keep everything
	}
	c.stats.InspectedColumnChunks++
	defer func() {
		if keep {
			c.stats.KeptColumnChunks++
		}
	}()

	keepNull := c.filter.KeepValue(predicateNullValue())
	indexPrunes := c.filter.SupportsColumnIndexPruning()
	minV, maxV, nulls, hasFooterStats := cc.FooterStats()
	allNull := hasFooterStats && minV.IsNull()

	mayMatch := false // whether the footer stats allow a match
	if hasFooterStats {
		if keepNull && nulls > 0 {
			// The filter keeps nulls, and the chunk has some.
			mayMatch = true
		} else if !allNull {
			// The chunk has present values, so its bounds may contain a match. This is
			// skipped for an all-null chunk, which has no real bounds.
			mayMatch = c.filter.KeepRange(minV, maxV)
		}
	}

	if indexPrunes && hasFooterStats && !mayMatch {
		return false // ruled out before even the dictionary page is read
	}

	if dict := cc.Dictionary(); dict != nil {
		// Dict-encoded chunk: every present value is a dictionary entry, so answering
		// for each entry decides the chunk and, kept, its rows (dictionary pushdown).
		// Nulls aren't stored in the dictionary; keepNull decides them.
		if c.dictPageReader.matchDictionary(dict, cc.ColumnChunk, c.filter, keepNull) {
			return true
		}
		// No entry matches, so only nulls could. Check the footer, or else the index, for
		// them even if the filter doesn't otherwise use bounds: one small read at most
		// decides whether the whole chunk is read, and a filter rejecting nulls never
		// gets this far.
		if !keepNull {
			return false
		}
		if hasFooterStats {
			return nulls > 0
		}
		return chunkHasNulls(cc)
	}

	if !indexPrunes {
		// Without the index there are no bounds. Such a filter's KeepRange ignores them,
		// so this is true for all but a CallbackPredicate, which still stops reading the
		// chunk once its callback is done.
		return c.filter.KeepRange(predicateNullValue(), predicateNullValue())
	}

	if hasFooterStats {
		return true // the index could only prune on gaps between its pages' bounds
	}

	// Tempo's blocks always carry footer stats, so the rest only runs for chunks that
	// don't come from a file, and is kept for completeness.
	ci, err := cc.ColumnIndex()
	if err != nil || ci == nil {
		return true // no column index: cannot skip
	}
	for i := 0; i < ci.NumPages(); i++ {
		if keepNull && ci.NullCount(i) > 0 {
			return true
		}
		if ci.NullPage(i) {
			// All-null page: min/max are not recorded, so only the null decision (above) applies.
			continue
		}
		if c.filter.KeepRange(ci.MinValue(i), ci.MaxValue(i)) {
			return true
		}
	}
	return false
}

// keepPage reports whether any value in pg can match c.filter, combining a page-bounds
// range test via KeepRange with a null test via KeepValue(null). Counts into c.stats.
// The caller guards c.filter != nil, so it is always non-nil here.
func (c *SyncIterator) keepPage(pg pq.Page) (keep bool) {
	c.stats.InspectedPages++
	defer func() {
		if keep {
			c.stats.KeptPages++
		}
	}()

	keepNull := c.filter.KeepValue(predicateNullValue())
	if keepNull && pg.NumNulls() > 0 {
		return true
	}
	if pg.NumValues()-pg.NumNulls() <= 0 {
		// No present values on this page; only the null decision (handled above) applies.
		return false
	}
	if pg.Dictionary() != nil {
		// Dict-encoded page: pg.Bounds() has no stored min/max to hand back, so it would
		// rescan every value to recompute them - a second full pass over data the caller
		// is about to read anyway. Chunk-level dictionary pruning already covers this
		// column, so skip the range test rather than pay for it.
		return true
	}
	if minV, maxV, ok := pg.Bounds(); ok {
		return c.filter.KeepRange(minV, maxV)
	}
	return true // no bounds recorded: cannot skip
}

func chunkHasNulls(cc *ColumnChunkHelper) bool {
	ci, err := cc.ColumnIndex()
	if err != nil || ci == nil {
		return true // unknown: assume nulls may be present
	}
	for i := 0; i < ci.NumPages(); i++ {
		if ci.NullCount(i) > 0 {
			return true
		}
	}
	return false
}
