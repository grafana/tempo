package parquetquery

import (
	"errors"
	"io"
	"slices"

	pq "github.com/parquet-go/parquet-go"
)

// chunkReader scans a single column chunk page by page on behalf of SyncIterator. It
// resolves the dictionary keep bitmap once per chunk, tracks the current page's
// row-number bounds, and drives two page readers - choosing the dictionary fast path
// when a page supports it, else the buffered per-row path.
//
// SyncIterator keeps the row-group navigation and the shared row number (curr), which
// is passed in by pointer so the scan can advance it. The current page object lives in
// the active page reader, not here.
type chunkReader struct {
	// config, copied from the iterator
	filter       Predicate
	maxDef       int  // per-column max definition level, handed to readers at setup
	dictDisabled bool // never use the dictionary fast path (test-only)

	chunk *ColumnChunkHelper // current column chunk

	// Dictionary state, resolved once per chunk by admitChunk. keep[dictIndex] reports
	// whether that dictionary value matches; nil when the fast path does not apply.
	keep          []bool
	dictionary    pq.Dictionary
	dictPagesUsed int // pages served by the fast path (observability/tests)

	// row-number bounds of the current page; the page object itself lives in the reader
	havePage bool
	pageMin  RowNumber
	pageMax  RowNumber // exclusive: first row number of the next page

	// the two page readers; dictActive selects which one serves the current page
	bufferedReader bufferedPageReader
	dictReader     dictPageReader
	dictActive     bool
}

// dictFastPathPages reports how many pages the dictionary fast path served. Used by
// in-package tests to confirm it engaged.
func (c *SyncIterator) dictFastPathPages() int {
	return c.chunk.dictPagesUsed
}

func (cr *chunkReader) hasPage() bool { return cr.havePage }

// admitChunk reports whether cc must be scanned, resolving the dictionary keep bitmap
// along the way. When the predicate resolves against the dictionary, the keep bitmap
// also answers the chunk-skip question, so we avoid a second dictionary scan in
// KeepColumnChunk.
func (cr *chunkReader) admitChunk(cc *ColumnChunkHelper) bool {
	resolved := cr.resolveChunk(cc)
	switch {
	case cr.filter == nil:
		return true
	case resolved:
		return slices.Contains(cr.keep, true)
	default:
		return cr.filter.KeepColumnChunk(cc)
	}
}

// resolveChunk evaluates the predicate against the chunk's dictionary into keep,
// returning true when a usable bitmap was produced. It returns false (keep nil) when
// the fast path does not apply: disabled, the filter is not a DictionaryPredicate, the
// predicate declines (e.g. OR with a non-dictionary child), or the chunk is not
// dictionary-encoded.
func (cr *chunkReader) resolveChunk(cc *ColumnChunkHelper) bool {
	cr.keep = nil
	cr.dictionary = nil
	if cr.dictDisabled {
		return false
	}
	dp, ok := cr.filter.(DictionaryPredicate)
	if !ok {
		return false
	}
	dictionary := cc.Dictionary()
	if dictionary == nil {
		return false
	}
	cr.dictionary = dictionary
	cr.keep = dp.KeepIndexes(dictionary) // nil if the predicate declines
	return cr.keep != nil
}

func (cr *chunkReader) setChunk(cc *ColumnChunkHelper) {
	cr.chunk = cc
}

// closeChunk releases the current page and closes the chunk. keep/dictionary are left
// alone: admitChunk resolves them for the next chunk, and resolveChunk clears them at
// the start of each chunk.
func (cr *chunkReader) closeChunk(curr *RowNumber) {
	if cr.chunk != nil {
		cr.chunk.Close()
		cr.chunk = nil
	}
	cr.setPage(curr, nil)
}

// loadPage advances to the next page that must be scanned, skipping pages the page
// predicate rejects and (when seekTo is non-nil) pages that end before seekTo. curr is
// advanced over skipped pages. ok is false when the chunk is exhausted.
func (cr *chunkReader) loadPage(curr *RowNumber, seekTo *RowNumber, definitionLevel int) (ok bool, err error) {
	for {
		pg, err := cr.chunk.NextPage()
		if err != nil && !errors.Is(err, io.EOF) {
			return false, err
		}
		if pg == nil || errors.Is(err, io.EOF) {
			return false, nil // chunk exhausted
		}

		// Skip pages that end before the seek target.
		if seekTo != nil {
			endRN := *curr
			endRN.Skip(pg.NumRows() + 1)
			if CompareRowNumbers(definitionLevel, *seekTo, endRN) >= 0 {
				curr.Skip(pg.NumRows())
				pq.Release(pg)
				continue
			}
		}

		if cr.filter != nil && !cr.filter.KeepPage(pg) {
			curr.Skip(pg.NumRows())
			pq.Release(pg)
			continue
		}
		cr.setPage(curr, pg)
		return true, nil
	}
}

// setPage swaps in pg as the current page (nil to clear). The outgoing page is released
// (by the active reader) and curr repositioned to its end; the incoming page's bounds
// are computed and the matching page reader is set up.
func (cr *chunkReader) setPage(curr *RowNumber, pg pq.Page) {
	if cr.havePage {
		*curr = cr.pageMax.Preceding() // reposition to the end of the outgoing page
	}

	// Reset both readers - whichever held the page releases it.
	cr.bufferedReader.reset()
	cr.dictReader.reset()
	cr.dictActive = false
	cr.havePage = false
	cr.pageMin = EmptyRowNumber()
	cr.pageMax = EmptyRowNumber()

	if pg == nil {
		cr.bufferedReader.release() // no more pages: free the batch buffer
		return
	}

	end := *curr
	end.Skip(pg.NumRows() + 1) // exclusive upper bound: first row of the next page
	cr.pageMin = *curr
	cr.pageMax = end
	cr.havePage = true

	if cr.keep != nil && pg.Dictionary() != nil {
		cr.dictReader.setup(pg, cr.keep, cr.dictionary, cr.maxDef)
		cr.dictActive = true
		cr.dictPagesUsed++
	} else {
		cr.bufferedReader.setup(pg, cr.maxDef)
	}
}

// pastPage reports whether seekTo is at or beyond the end of the current page.
func (cr *chunkReader) pastPage(seekTo RowNumber, definitionLevel int) bool {
	return CompareRowNumbers(definitionLevel, seekTo, cr.pageMax) >= 0
}

// next returns the next matching value in the current page via the active reader.
func (cr *chunkReader) next(curr *RowNumber) (*pq.Value, bool, error) {
	if cr.dictActive {
		return cr.dictReader.next(curr)
	}
	return cr.bufferedReader.next(curr)
}

// seekWithinPage moves toward row number to. The buffered reader may reslice the page
// straight to the target for a large skip; the dictionary reader tracks its cursor
// against the whole page and cannot reslice, so it always advances via next (correct,
// just without the shortcut).
func (cr *chunkReader) seekWithinPage(curr *RowNumber, to RowNumber, definitionLevel int) {
	if cr.dictActive {
		return
	}
	if newMin, ok := cr.bufferedReader.reslice(curr, cr.pageMin, to, definitionLevel); ok {
		cr.pageMin = newMin
	}
}
