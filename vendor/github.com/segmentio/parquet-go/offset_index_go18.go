//go:build go1.18

package parquet

type offsetIndex[T primitive] struct{ page *page[T] }

func (i offsetIndex[T]) NumPages() int                { return 1 }
func (i offsetIndex[T]) Offset(int) int64             { return 0 }
func (i offsetIndex[T]) CompressedPageSize(int) int64 { return i.page.Size() }
func (i offsetIndex[T]) FirstRowIndex(int) int64      { return 0 }
