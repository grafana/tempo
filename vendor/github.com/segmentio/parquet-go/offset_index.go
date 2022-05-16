package parquet

import (
	"github.com/segmentio/parquet-go/format"
)

type OffsetIndex interface {
	// NumPages returns the number of pages in the offset index.
	NumPages() int

	// Offset returns the offset starting from the beginning of the file for the
	// page at the given index.
	Offset(int) int64

	// CompressedPageSize returns the size of the page at the given index
	// (in bytes).
	CompressedPageSize(int) int64

	// FirstRowIndex returns the the first row in the page at the given index.
	//
	// The returned row index is based on the row group that the page belongs
	// to, the first row has index zero.
	FirstRowIndex(int) int64
}

type emptyOffsetIndex struct{}

func (emptyOffsetIndex) NumPages() int                { return 0 }
func (emptyOffsetIndex) Offset(int) int64             { return 0 }
func (emptyOffsetIndex) CompressedPageSize(int) int64 { return 0 }
func (emptyOffsetIndex) FirstRowIndex(int) int64      { return 0 }

type fileOffsetIndex format.OffsetIndex

func (i *fileOffsetIndex) NumPages() int {
	return len(i.PageLocations)
}

func (i *fileOffsetIndex) Offset(j int) int64 {
	return i.PageLocations[j].Offset
}

func (i *fileOffsetIndex) CompressedPageSize(j int) int64 {
	return int64(i.PageLocations[j].CompressedPageSize)
}

func (i *fileOffsetIndex) FirstRowIndex(j int) int64 {
	return i.PageLocations[j].FirstRowIndex
}

type byteArrayOffsetIndex struct{ page *byteArrayPage }

func (i byteArrayOffsetIndex) NumPages() int                { return 1 }
func (i byteArrayOffsetIndex) Offset(int) int64             { return 0 }
func (i byteArrayOffsetIndex) CompressedPageSize(int) int64 { return i.page.Size() }
func (i byteArrayOffsetIndex) FirstRowIndex(int) int64      { return 0 }

type fixedLenByteArrayOffsetIndex struct{ page *fixedLenByteArrayPage }

func (i fixedLenByteArrayOffsetIndex) NumPages() int                { return 1 }
func (i fixedLenByteArrayOffsetIndex) Offset(int) int64             { return 0 }
func (i fixedLenByteArrayOffsetIndex) CompressedPageSize(int) int64 { return i.page.Size() }
func (i fixedLenByteArrayOffsetIndex) FirstRowIndex(int) int64      { return 0 }
