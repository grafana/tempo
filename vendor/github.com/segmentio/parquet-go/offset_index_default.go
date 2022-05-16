//go:build !go1.18

package parquet

type booleanOffsetIndex struct{ page *booleanPage }

func (i booleanOffsetIndex) NumPages() int                { return 1 }
func (i booleanOffsetIndex) Offset(int) int64             { return 0 }
func (i booleanOffsetIndex) CompressedPageSize(int) int64 { return i.page.Size() }
func (i booleanOffsetIndex) FirstRowIndex(int) int64      { return 0 }

type int32OffsetIndex struct{ page *int32Page }

func (i int32OffsetIndex) NumPages() int                { return 1 }
func (i int32OffsetIndex) Offset(int) int64             { return 0 }
func (i int32OffsetIndex) CompressedPageSize(int) int64 { return i.page.Size() }
func (i int32OffsetIndex) FirstRowIndex(int) int64      { return 0 }

type int64OffsetIndex struct{ page *int64Page }

func (i int64OffsetIndex) NumPages() int                { return 1 }
func (i int64OffsetIndex) Offset(int) int64             { return 0 }
func (i int64OffsetIndex) CompressedPageSize(int) int64 { return i.page.Size() }
func (i int64OffsetIndex) FirstRowIndex(int) int64      { return 0 }

type int96OffsetIndex struct{ page *int96Page }

func (i int96OffsetIndex) NumPages() int                { return 1 }
func (i int96OffsetIndex) Offset(int) int64             { return 0 }
func (i int96OffsetIndex) CompressedPageSize(int) int64 { return i.page.Size() }
func (i int96OffsetIndex) FirstRowIndex(int) int64      { return 0 }

type floatOffsetIndex struct{ page *floatPage }

func (i floatOffsetIndex) NumPages() int                { return 1 }
func (i floatOffsetIndex) Offset(int) int64             { return 0 }
func (i floatOffsetIndex) CompressedPageSize(int) int64 { return i.page.Size() }
func (i floatOffsetIndex) FirstRowIndex(int) int64      { return 0 }

type doubleOffsetIndex struct{ page *doublePage }

func (i doubleOffsetIndex) NumPages() int                { return 1 }
func (i doubleOffsetIndex) Offset(int) int64             { return 0 }
func (i doubleOffsetIndex) CompressedPageSize(int) int64 { return i.page.Size() }
func (i doubleOffsetIndex) FirstRowIndex(int) int64      { return 0 }

type uint32OffsetIndex struct{ page uint32Page }

func (i uint32OffsetIndex) NumPages() int                { return 1 }
func (i uint32OffsetIndex) Offset(int) int64             { return 0 }
func (i uint32OffsetIndex) CompressedPageSize(int) int64 { return i.page.Size() }
func (i uint32OffsetIndex) FirstRowIndex(int) int64      { return 0 }

type uint64OffsetIndex struct{ page uint64Page }

func (i uint64OffsetIndex) NumPages() int                { return 1 }
func (i uint64OffsetIndex) Offset(int) int64             { return 0 }
func (i uint64OffsetIndex) CompressedPageSize(int) int64 { return i.page.Size() }
func (i uint64OffsetIndex) FirstRowIndex(int) int64      { return 0 }
