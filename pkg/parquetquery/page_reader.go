package parquetquery

import (
	"errors"
	"io"

	pq "github.com/parquet-go/parquet-go"
)

// pageReader scans one parquet page, returning matching values and advancing the
// shared row number. chunkReader owns one of each implementation and picks per page:
//   - dictPageReader     when the page is dictionary-encoded and the predicate was
//     resolved into a keep bitmap; matches rows by dictionary index.
//   - bufferedPageReader otherwise; materializes values and runs the predicate.
//
// Both advance curr across every value - including filtered and null ones - so row
// numbers stay correct no matter which reader runs. maxDef is configured at setup
// (an immutable per-column constant), so only curr is passed per call.
type pageReader interface {
	// next returns the next matching value, advancing curr. ok is false at end of page.
	next(curr *RowNumber) (*pq.Value, bool, error)
	// reset releases the page and drops per-page state so the reader can be reused.
	reset()
}

var (
	_ pageReader = (*bufferedPageReader)(nil)
	_ pageReader = (*dictPageReader)(nil)
)

// bufferedPageReader materializes the page's values in batches and applies the value
// predicate to each one. It owns its page (and reslicing it on seek) and a pooled
// batch buffer reused across the chunk's pages.
type bufferedPageReader struct {
	filter   Predicate // value predicate (shared with chunkReader)
	readSize int       // values per ReadValues batch
	maxDef   int       // row-number tracking cap

	page    pq.Page
	values  pq.ValueReader
	buf     []pq.Value // current batch; retained across pages, freed by release
	bufPos  int        // next value to read from buf
	pagePos int        // values consumed so far in the page (used by reslice)
}

func (b *bufferedPageReader) setup(pg pq.Page, maxDef int) {
	b.page = pg
	b.maxDef = maxDef
	b.values = pg.Values()
	b.bufPos = 0
	b.pagePos = 0
}

// reset releases the page and clears the per-page cursors; the batch buffer is kept
// for the next page and freed separately by release.
func (b *bufferedPageReader) reset() {
	if b.page != nil {
		pq.Release(b.page)
		b.page = nil
	}
	b.values = nil
	b.bufPos = 0
	b.pagePos = 0
}

// release returns the batch buffer to the pool. Called when the chunk has no more pages.
func (b *bufferedPageReader) release() {
	if b.buf != nil {
		syncIteratorPoolPut(b.buf)
		b.buf = nil
	}
}

func (b *bufferedPageReader) next(curr *RowNumber) (*pq.Value, bool, error) {
	if b.buf == nil {
		b.buf = syncIteratorPoolGet(b.readSize, 0)
	}
	for {
		// Refill the batch once the current one is drained.
		if b.bufPos >= len(b.buf) || len(b.buf) == 0 {
			b.buf = b.buf[:cap(b.buf)]
			n, err := b.values.ReadValues(b.buf)
			if err != nil && !errors.Is(err, io.EOF) {
				return nil, false, err
			}
			b.buf = b.buf[:n]
			b.bufPos = 0
			if n == 0 {
				return nil, false, nil // end of page
			}
		}

		for b.bufPos < len(b.buf) {
			v := &b.buf[b.bufPos]

			// Advance the row number for every value, so it stays accurate even when
			// the value is filtered out below.
			curr.Next(v.RepetitionLevel(), v.DefinitionLevel(), b.maxDef)
			b.bufPos++
			b.pagePos++

			if b.filter != nil && !b.filter.KeepValue(*v) {
				continue
			}
			return v, true, nil
		}
	}
}

// peek returns the current value on the page without consuming it; ok is false at end
// of page. Used by NilSyncIterator, which inspects a value to decide scope membership
// and may return (emitting a synthetic nil) before consuming it. The pointer is valid
// until the next consume (which may refill the batch).
func (b *bufferedPageReader) peek() (*pq.Value, bool, error) {
	if b.buf == nil {
		b.buf = syncIteratorPoolGet(b.readSize, 0)
	}
	if b.bufPos >= len(b.buf) || len(b.buf) == 0 {
		b.buf = b.buf[:cap(b.buf)]
		n, err := b.values.ReadValues(b.buf)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, false, err
		}
		b.buf = b.buf[:n]
		b.bufPos = 0
		if n == 0 {
			return nil, false, nil // end of page
		}
	}
	return &b.buf[b.bufPos], true, nil
}

// consume advances past the value most recently returned by peek.
func (b *bufferedPageReader) consume() {
	b.bufPos++
	b.pagePos++
}

// reslice jumps straight to row number to by replacing the page with a slice starting
// there, when the skip is large enough to be worth it (magicThreshold Next()s). It
// returns the new page-min row number and true when it resliced; otherwise pageMin and
// false (the caller advances via next instead). Buffered-only: the dictionary reader
// cannot reslice.
func (b *bufferedPageReader) reslice(curr *RowNumber, pageMin, to RowNumber, definitionLevel int) (RowNumber, bool) {
	rowSkipRelative := int(to[0] - curr[0])
	if rowSkipRelative == 0 {
		return pageMin, false
	}

	const magicThreshold = 1000
	shouldSkip := false
	if definitionLevel == 0 {
		// One Next() per row at definition level 0.
		shouldSkip = rowSkipRelative > magicThreshold
	} else {
		// Nested column: count the Next()s needed to reach the target row.
		replvls := b.page.RepetitionLevels()
		nextsRequired := 0
		for i := b.pagePos; i < len(replvls); i++ {
			nextsRequired++
			if nextsRequired > magicThreshold {
				shouldSkip = true
				break
			}
			if replvls[i] == 0 { // a 0 repetition level starts a new row
				rowSkipRelative--
				if rowSkipRelative <= 0 {
					break
				}
			}
		}
	}
	if !shouldSkip {
		return pageMin, false
	}

	// Reslice from the start of the page to the target row.
	rowSkip := to[0] - pageMin[0]
	if rowSkip < 1 || rowSkip > int32(b.page.NumRows()) {
		return pageMin, false
	}
	pg := b.page.Slice(int64(rowSkip-1), b.page.NumRows())
	*curr = TruncateRowNumber(0, to).Preceding() // drop detail below the row number

	pq.Release(b.page)
	b.release() // discard the stale batch buffer
	b.setup(pg, b.maxDef)
	return *curr, true
}

// dictPageReader matches rows by their dictionary index against the chunk's keep
// bitmap, materializing only the values that match.
//
// A dictionary-encoded page has one entry per row position ("slot"). Present
// (non-null) slots carry a dictionary index in dictIndexes; null slots do not. A slot
// is present when its definition level equals presentDefLevel.
type dictPageReader struct {
	// (keep, dictionary): the chunk-scope decision, resolved once by chunkReader and
	// lent to the reader for the page. keep is indexed by the same dictionary indexes
	// dictionary resolves.
	keep       []bool        // keep[dictIndex]: does that dictionary value match?
	dictionary pq.Dictionary // resolves a dictionary index back to its value
	maxDef     int           // row-number tracking cap

	page        pq.Page // owned; released on reset
	column      int     // column index, for stamping a matched value
	dictIndexes []int32 // dictionary index of each present value, in page order
	defLevels   []byte  // per slot; nil for required columns (every slot present)
	repLevels   []byte  // per slot; nil for non-repeated columns

	presentDefLevel byte // definition level that marks a present (non-null) leaf
	slotPos         int  // next slot to read (present or null)
	valuePos        int  // next present value (index into dictIndexes)

	value pq.Value // reused buffer for the returned value
}

func (d *dictPageReader) setup(pg pq.Page, keep []bool, dictionary pq.Dictionary, maxDef int) {
	d.page = pg
	d.keep = keep
	d.dictionary = dictionary
	d.maxDef = maxDef
	d.column = pg.Column()
	d.defLevels = pg.DefinitionLevels()
	d.repLevels = pg.RepetitionLevels()
	data := pg.Data()
	d.dictIndexes = data.Int32()
	// The highest definition level on the page marks a present leaf; required columns
	// have no definition levels, in which case every slot is present.
	d.presentDefLevel = maxByte(d.defLevels)
	d.slotPos = 0
	d.valuePos = 0
}

// reset releases the page and drops all per-page state.
func (d *dictPageReader) reset() {
	if d.page != nil {
		pq.Release(d.page)
	}
	*d = dictPageReader{}
}

func (d *dictPageReader) next(curr *RowNumber) (*pq.Value, bool, error) {
	numSlots := len(d.dictIndexes)
	if d.defLevels != nil {
		numSlots = len(d.defLevels)
	}

	for d.slotPos < numSlots {
		repLvl := 0
		if d.repLevels != nil {
			repLvl = int(d.repLevels[d.slotPos])
		}
		defLvl := int(d.presentDefLevel)
		if d.defLevels != nil {
			defLvl = int(d.defLevels[d.slotPos])
		}

		// Advance the row number for every slot, present or null.
		curr.Next(repLvl, defLvl, d.maxDef)
		d.slotPos++

		// Skip null/empty slots: a lower definition level, or no dictionary index left
		// to consume. The valuePos bound also covers pages with no present values at all
		// (all null, or only empty lists), where presentDefLevel collapses to the null
		// level. The gated predicates never keep nulls, matching the per-row path.
		if (d.defLevels != nil && byte(defLvl) != d.presentDefLevel) || d.valuePos >= len(d.dictIndexes) {
			continue
		}

		idx := d.dictIndexes[d.valuePos]
		d.valuePos++
		if !d.keep[idx] {
			continue
		}

		// Materialize only matches, stamping the page levels and column so the result is
		// indistinguishable from the per-row path.
		d.value = d.dictionary.Index(idx).Level(repLvl, defLvl, d.column)
		return &d.value, true, nil
	}
	return nil, false, nil
}

func maxByte(b []byte) byte {
	var m byte
	for _, v := range b {
		m = max(m, v)
	}
	return m
}
