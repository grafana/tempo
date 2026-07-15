package parquetquery

import (
	pq "github.com/parquet-go/parquet-go"
)

// predicateStats counts the chunk/page pruning decisions made by keepColumnChunk /
// keepPage. It is optional (nil-safe) and, when supplied via an InstrumentedPredicate,
// lets callers report inspected/kept counts without the Predicate interface needing
// chunk/page methods.
type predicateStats struct {
	InspectedColumnChunks int64
	KeptColumnChunks      int64
	InspectedPages        int64
	KeptPages             int64
}

// predicateNullValue returns the null pq.Value fed to KeepValue when deciding
// whether null rows in a chunk/page could match. It must be the same null
// representation the per-row value reader produces so the skip decision agrees
// with per-row evaluation.
func predicateNullValue() pq.Value { return pq.Value{} }

// keepColumnChunk reports whether any value in cc can match c.pred. The decision
// combines a dictionary match for dict-encoded (string) columns, a column-index
// range test via KeepRange, and a null-count test via KeepValue(null). Counts into
// c.stats when set.
func (c *SyncIterator) keepColumnChunk(cc *ColumnChunkHelper) (keep bool) {
	if c.filter == nil {
		return true // no predicate: keep everything
	}
	if c.stats != nil {
		c.stats.InspectedColumnChunks++
		defer func() {
			if keep {
				c.stats.KeptColumnChunks++
			}
		}()
	}

	keepNull := c.pred.KeepValue(predicateNullValue())

	if dict := cc.Dictionary(); dict != nil {
		// Dict-encoded (string) chunk: any present value that matches lives in the
		// dictionary; null rows are decided by keepNull since nulls are not stored there.
		if keepDictionary(dict, c.pred.KeepValue) {
			return true
		}
		return keepNull && chunkHasNulls(cc)
	}

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
		if c.pred.KeepRange(ci.MinValue(i), ci.MaxValue(i)) {
			return true
		}
	}
	return false
}

// keepPage reports whether any value in pg can match c.pred, combining a page-bounds
// range test via KeepRange with a null test via KeepValue(null). Counts into c.stats
// when set. The caller guards c.filter != nil, so c.pred is always non-nil here.
func (c *SyncIterator) keepPage(pg pq.Page) (keep bool) {
	if c.stats != nil {
		c.stats.InspectedPages++
		defer func() {
			if keep {
				c.stats.KeptPages++
			}
		}()
	}

	keepNull := c.pred.KeepValue(predicateNullValue())
	if keepNull && pg.NumNulls() > 0 {
		return true
	}
	if pg.NumValues()-pg.NumNulls() <= 0 {
		// No present values on this page; only the null decision (handled above) applies.
		return false
	}
	if min, max, ok := pg.Bounds(); ok {
		return c.pred.KeepRange(min, max)
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

// keepDictionary reports whether any dictionary entry matches keepValue.
func keepDictionary(dict pq.Dictionary, keepValue func(pq.Value) bool) bool {
	for i, l := 0, dict.Len(); i < l; i++ {
		if keepValue(dict.Index(int32(i))) {
			return true
		}
	}
	return false
}
