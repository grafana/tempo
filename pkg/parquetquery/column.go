package parquetquery

import "github.com/parquet-go/parquet-go"

type ColumnChunkHelper struct {
	parquet.ColumnChunk
	pages     parquet.Pages
	firstPage parquet.Page
	err       error
}

// Dictionary makes it easier to access the dictionary for this column chunk which
// is only accessible through the first page. Internally keeps some open buffers
// to reuse later which are accessed through the other methods. If there is no dictionary
// for this column chunk or an error occurs, return nil.
func (h *ColumnChunkHelper) Dictionary() parquet.Dictionary {
	if h.pages == nil {
		h.pages = h.Pages()
	}

	if h.firstPage == nil {
		h.firstPage, h.err = h.pages.ReadPage()
	}

	if h.firstPage == nil {
		// Maybe there was an error
		return nil
	}

	return h.firstPage.Dictionary()
}

// footerStats is implemented by file column chunks, whose statistics come with the
// file footer and need no backend request.
type footerStats interface {
	Bounds() (minV, maxV parquet.Value, ok bool)
	NullCount() int64
}

// FooterStats returns the chunk's min, max and null count from the file footer. minV and
// maxV are null for an all-null chunk, which has no bounds. ok is false when the chunk
// doesn't carry them, as for one not read from a file.
func (h *ColumnChunkHelper) FooterStats() (minV, maxV parquet.Value, nulls int64, ok bool) {
	s, isFile := h.ColumnChunk.(footerStats)
	if !isFile {
		return minV, maxV, 0, false
	}
	nulls = s.NullCount()
	if minV, maxV, ok = s.Bounds(); ok {
		return minV, maxV, nulls, true
	}
	if nulls == h.NumValues() {
		return parquet.Value{}, parquet.Value{}, nulls, true
	}
	return parquet.Value{}, parquet.Value{}, 0, false
}

// NextPage wraps pages.ReadPage and helps reuse already open buffers.
func (h *ColumnChunkHelper) NextPage() (parquet.Page, error) {
	if h.err != nil {
		return nil, h.err
	}

	if h.firstPage != nil {
		// Clear and return the already buffered first page.
		// Caller takes ownership of it.
		pg := h.firstPage
		h.firstPage = nil
		return pg, nil
	}

	if h.pages == nil {
		h.pages = h.Pages()
	}

	return h.pages.ReadPage()
}

func (h *ColumnChunkHelper) Close() error {
	if h.firstPage != nil {
		parquet.Release(h.firstPage)
		h.firstPage = nil
	}

	if h.pages != nil {
		err := h.pages.Close()
		h.pages = nil
		return err
	}

	return nil
}
