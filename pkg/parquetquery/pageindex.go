package parquetquery

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"

	pq "github.com/parquet-go/parquet-go"
)

// FindRowByColumnIndex returns the row number, relative to the row group, of
// the first row containing value in the given column. The column index
// (per-page min/max bounds) is binary searched for the page that can contain
// the value, the offset index maps the page to its first row, and the page is
// scanned for the exact row.
// ok is false when the column index can't be used: missing, unordered, or
// lacking bounds. The caller must fall back to scanning the row group.
// When ok is true and the returned row is -1, the row group does not contain
// the value.
func FindRowByColumnIndex(rg pq.RowGroup, column int, value []byte) (int64, bool, error) {
	cc := rg.ColumnChunks()[column]
	fcc, ok := cc.(*pq.FileColumnChunk)
	if !ok {
		return -1, false, nil
	}

	ci, err := fcc.ColumnIndex()
	if errors.Is(err, pq.ErrMissingColumnIndex) {
		return -1, false, nil
	}
	if err != nil {
		return -1, false, fmt.Errorf("error loading column index: %w", err)
	}

	page, ok := findPageByColumnIndex(ci, value)
	if !ok {
		return -1, false, nil
	}
	if page == -1 {
		// Ruled out by the page bounds: the row group does not contain the value.
		return -1, true, nil
	}

	oi, err := fcc.OffsetIndex()
	if errors.Is(err, pq.ErrMissingOffsetIndex) {
		return -1, false, nil
	}
	if err != nil {
		return -1, false, fmt.Errorf("error loading offset index: %w", err)
	}
	if page >= oi.NumPages() {
		// The column and offset indexes disagree on the page count.
		return -1, false, nil
	}

	// First row of the page, relative to the row group.
	firstRow := oi.FirstRowIndex(page)

	// Scan the page for the exact row. With the offset index loaded, SeekToRow
	// jumps directly to the page. Values are sorted ascending, so the scan
	// stops at the first value greater than the searched value.
	pages := fcc.Pages()
	defer pages.Close()

	if err := pages.SeekToRow(firstRow); err != nil {
		return -1, false, fmt.Errorf("seek to row: %w", err)
	}

	pg, err := pages.ReadPage()
	if err != nil {
		return -1, false, fmt.Errorf("error reading page: %w", err)
	}
	defer pq.Release(pg)

	var (
		values = pg.Values()
		buf    = make([]pq.Value, 256)
		row    = int64(0)
	)
	for row < pg.NumRows() {
		n, err := values.ReadValues(buf)
		for i := 0; i < n; i++ {
			cmp := bytes.Compare(buf[i].ByteArray(), value)
			if cmp == 0 {
				return firstRow + row + int64(i), true, nil
			}
			if cmp > 0 {
				// The page didn't contain the value.
				return -1, true, nil
			}
		}
		row += int64(n)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return -1, false, fmt.Errorf("error reading page values: %w", err)
		}
	}

	// The page didn't contain the value.
	return -1, true, nil
}

// findPageByColumnIndex returns the index of the page whose min/max bounds
// contain value, or -1 if no page can contain it. ok is false when the page
// bounds can't rule pages in or out: a page has no bounds, or the bounds are
// not in ascending order.
func findPageByColumnIndex(ci pq.ColumnIndex, value []byte) (int, bool) {
	// The binary search is only valid if every page has bounds and the min/max
	// values are in ascending order. Bounds are skipped at write time for some
	// columns, e.g. blobs.
	//
	// The order is verified here instead of relying on IsAscending: the writer
	// reports an unordered index for row groups with a single page.
	var prevMin, prevMax []byte
	for i := 0; i < ci.NumPages(); i++ {
		if ci.NullPage(i) {
			return -1, false
		}
		minVal := ci.MinValue(i).ByteArray()
		maxVal := ci.MaxValue(i).ByteArray()
		if len(minVal) == 0 || len(maxVal) == 0 {
			return -1, false
		}
		if bytes.Compare(minVal, prevMin) < 0 || bytes.Compare(maxVal, prevMax) < 0 {
			return -1, false
		}
		prevMin, prevMax = minVal, maxVal
	}

	i := sort.Search(ci.NumPages(), func(i int) bool {
		return bytes.Compare(ci.MaxValue(i).ByteArray(), value) >= 0
	})
	if i == ci.NumPages() || bytes.Compare(value, ci.MinValue(i).ByteArray()) < 0 {
		return -1, true
	}
	return i, true
}
