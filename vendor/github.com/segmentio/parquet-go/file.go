package parquet

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"sort"

	"github.com/segmentio/encoding/thrift"
	"github.com/segmentio/parquet-go/format"
)

const (
	defaultDictBufferSize  = 8192
	defaultReadBufferSize  = 4096
	defaultLevelBufferSize = 1024
)

// File represents a parquet file. The layout of a Parquet file can be found
// here: https://github.com/apache/parquet-format#file-format
type File struct {
	metadata      format.FileMetaData
	protocol      thrift.CompactProtocol
	reader        io.ReaderAt
	size          int64
	schema        *Schema
	root          *Column
	columnIndexes []format.ColumnIndex
	offsetIndexes []format.OffsetIndex
	rowGroups     []RowGroup
}

// OpenFile opens a parquet file and reads the content between offset 0 and the given
// size in r.
//
// Only the parquet magic bytes and footer are read, column chunks and other
// parts of the file are left untouched; this means that successfully opening
// a file does not validate that the pages have valid checksums.
func OpenFile(r io.ReaderAt, size int64, options ...FileOption) (*File, error) {
	b := make([]byte, 8)
	f := &File{reader: r, size: size}
	c, err := NewFileConfig(options...)
	if err != nil {
		return nil, err
	}

	if _, err := r.ReadAt(b[:4], 0); err != nil {
		return nil, fmt.Errorf("reading magic header of parquet file: %w", err)
	}
	if string(b[:4]) != "PAR1" {
		return nil, fmt.Errorf("invalid magic header of parquet file: %q", b[:4])
	}

	if _, err := r.ReadAt(b[:8], size-8); err != nil {
		return nil, fmt.Errorf("reading magic footer of parquet file: %w", err)
	}
	if string(b[4:8]) != "PAR1" {
		return nil, fmt.Errorf("invalid magic footer of parquet file: %q", b[4:8])
	}

	footerSize := int64(binary.LittleEndian.Uint32(b[:4]))
	footerData := make([]byte, footerSize)

	if _, err := f.reader.ReadAt(footerData, size-(footerSize+8)); err != nil {
		return nil, fmt.Errorf("reading footer of parquet file: %w", err)
	}
	if err := thrift.Unmarshal(&f.protocol, footerData, &f.metadata); err != nil {
		return nil, fmt.Errorf("reading parquet file metadata: %w", err)
	}
	if len(f.metadata.Schema) == 0 {
		return nil, ErrMissingRootColumn
	}

	if !c.SkipPageIndex {
		if f.columnIndexes, f.offsetIndexes, err = f.ReadPageIndex(); err != nil {
			return nil, fmt.Errorf("reading page index of parquet file: %w", err)
		}
	}

	if f.root, err = openColumns(f); err != nil {
		return nil, fmt.Errorf("opening columns of parquet file: %w", err)
	}

	schema := NewSchema(f.root.Name(), f.root)
	columns := make([]*Column, 0, MaxColumnIndex+1)
	f.schema = schema
	f.root.forEachLeaf(func(c *Column) { columns = append(columns, c) })

	rowGroups := make([]fileRowGroup, len(f.metadata.RowGroups))
	for i := range rowGroups {
		rowGroups[i].init(f, schema, columns, &f.metadata.RowGroups[i])
	}
	f.rowGroups = make([]RowGroup, len(rowGroups))
	for i := range rowGroups {
		f.rowGroups[i] = &rowGroups[i]
	}

	if !c.SkipBloomFilters {
		h := format.BloomFilterHeader{}
		p := thrift.CompactProtocol{}
		s := io.NewSectionReader(r, 0, size)
		d := thrift.NewDecoder(p.NewReader(s))

		for i := range rowGroups {
			g := &rowGroups[i]

			for j := range g.columns {
				c := g.columns[j].(*fileColumnChunk)

				if offset := c.chunk.MetaData.BloomFilterOffset; offset > 0 {
					s.Seek(offset, io.SeekStart)
					h = format.BloomFilterHeader{}
					if err := d.Decode(&h); err != nil {
						return nil, err
					}
					offset, _ = s.Seek(0, io.SeekCurrent)
					c.bloomFilter = newBloomFilter(r, offset, &h)
				}
			}
		}
	}

	sortKeyValueMetadata(f.metadata.KeyValueMetadata)
	return f, nil
}

// ReadPageIndex reads the page index section of the parquet file f.
//
// If the file did not contain a page index, the method returns two empty slices
// and a nil error.
//
// Only leaf columns have indexes, the returned indexes are arranged using the
// following layout:
//
//	+ -------------- +
//	| col 0: chunk 0 |
//	+ -------------- +
//	| col 1: chunk 0 |
//	+ -------------- +
//	| ...            |
//	+ -------------- +
//	| col 0: chunk 1 |
//	+ -------------- +
//	| col 1: chunk 1 |
//	+ -------------- +
//	| ...            |
//	+ -------------- +
//
// This method is useful in combination with the SkipPageIndex option to delay
// reading the page index section until after the file was opened. Note that in
// this case the page index is not cached within the file, programs are expected
// to make use of independently from the parquet package.
func (f *File) ReadPageIndex() ([]format.ColumnIndex, []format.OffsetIndex, error) {
	columnIndexOffset := f.metadata.RowGroups[0].Columns[0].ColumnIndexOffset
	offsetIndexOffset := f.metadata.RowGroups[0].Columns[0].OffsetIndexOffset
	columnIndexLength := int64(0)
	offsetIndexLength := int64(0)

	if columnIndexOffset == 0 || offsetIndexOffset == 0 {
		return nil, nil, nil
	}

	forEachColumnChunk := func(do func(int, int, *format.ColumnChunk) error) error {
		for i := range f.metadata.RowGroups {
			for j := range f.metadata.RowGroups[i].Columns {
				c := &f.metadata.RowGroups[i].Columns[j]
				if err := do(i, j, c); err != nil {
					return err
				}
			}
		}
		return nil
	}

	forEachColumnChunk(func(_, _ int, c *format.ColumnChunk) error {
		columnIndexLength += int64(c.ColumnIndexLength)
		offsetIndexLength += int64(c.OffsetIndexLength)
		return nil
	})

	numRowGroups := len(f.metadata.RowGroups)
	numColumns := len(f.metadata.RowGroups[0].Columns)
	numColumnChunks := numRowGroups * numColumns

	columnIndexes := make([]format.ColumnIndex, numColumnChunks)
	offsetIndexes := make([]format.OffsetIndex, numColumnChunks)
	indexBuffer := make([]byte, max(int(columnIndexLength), int(offsetIndexLength)))

	if columnIndexOffset > 0 {
		columnIndexData := indexBuffer[:columnIndexLength]

		if _, err := f.reader.ReadAt(columnIndexData, columnIndexOffset); err != nil {
			return nil, nil, fmt.Errorf("reading %d bytes column index at offset %d: %w", columnIndexLength, columnIndexOffset, err)
		}

		err := forEachColumnChunk(func(i, j int, c *format.ColumnChunk) error {
			offset := c.ColumnIndexOffset - columnIndexOffset
			length := int64(c.ColumnIndexLength)
			buffer := columnIndexData[offset : offset+length]
			if err := thrift.Unmarshal(&f.protocol, buffer, &columnIndexes[(i*numColumns)+j]); err != nil {
				return fmt.Errorf("decoding column index: rowGroup=%d columnChunk=%d/%d: %w", i, j, numColumns, err)
			}
			return nil
		})
		if err != nil {
			return nil, nil, err
		}
	}

	if offsetIndexOffset > 0 {
		offsetIndexData := indexBuffer[:offsetIndexLength]

		if _, err := f.reader.ReadAt(offsetIndexData, offsetIndexOffset); err != nil {
			return nil, nil, fmt.Errorf("reading %d bytes offset index at offset %d: %w", offsetIndexLength, offsetIndexOffset, err)
		}

		err := forEachColumnChunk(func(i, j int, c *format.ColumnChunk) error {
			offset := c.OffsetIndexOffset - offsetIndexOffset
			length := int64(c.OffsetIndexLength)
			buffer := offsetIndexData[offset : offset+length]
			if err := thrift.Unmarshal(&f.protocol, buffer, &offsetIndexes[(i*numColumns)+j]); err != nil {
				return fmt.Errorf("decoding column index: rowGroup=%d columnChunk=%d/%d: %w", i, j, numColumns, err)
			}
			return nil
		})
		if err != nil {
			return nil, nil, err
		}
	}

	return columnIndexes, offsetIndexes, nil
}

// NumRows returns the number of rows in the file.
func (f *File) NumRows() int64 { return f.metadata.NumRows }

// RowGroups returns the list of row group in the file.
func (f *File) RowGroups() []RowGroup { return f.rowGroups }

// Root returns the root column of f.
func (f *File) Root() *Column { return f.root }

// Schema returns the schema of f.
func (f *File) Schema() *Schema { return f.schema }

// Size returns the size of f (in bytes).
func (f *File) Size() int64 { return f.size }

// ReadAt reads bytes into b from f at the given offset.
//
// The method satisfies the io.ReaderAt interface.
func (f *File) ReadAt(b []byte, off int64) (int, error) {
	if off < 0 || off >= f.size {
		return 0, io.EOF
	}

	if limit := f.size - off; limit < int64(len(b)) {
		n, err := f.reader.ReadAt(b[:limit], off)
		if err == nil {
			err = io.EOF
		}
		return n, err
	}

	return f.reader.ReadAt(b, off)
}

// ColumnIndexes returns the page index of the parquet file f.
//
// If the file did not contain a column index, the method returns an empty slice
// and nil error.
func (f *File) ColumnIndexes() []format.ColumnIndex { return f.columnIndexes }

// OffsetIndexes returns the page index of the parquet file f.
//
// If the file did not contain an offset index, the method returns an empty
// slice and nil error.
func (f *File) OffsetIndexes() []format.OffsetIndex { return f.offsetIndexes }

// Lookup returns the value associated with the given key in the file key/value
// metadata.
//
// The ok boolean will be true if the key was found, false otherwise.
func (f *File) Lookup(key string) (value string, ok bool) {
	return lookupKeyValueMetadata(f.metadata.KeyValueMetadata, key)
}

func (f *File) hasIndexes() bool {
	return f.columnIndexes != nil && f.offsetIndexes != nil
}

var (
	_ io.ReaderAt = (*File)(nil)
)

func sortKeyValueMetadata(keyValueMetadata []format.KeyValue) {
	sort.Slice(keyValueMetadata, func(i, j int) bool {
		switch {
		case keyValueMetadata[i].Key < keyValueMetadata[j].Key:
			return true
		case keyValueMetadata[i].Key > keyValueMetadata[j].Key:
			return false
		default:
			return keyValueMetadata[i].Value < keyValueMetadata[j].Value
		}
	})
}

func lookupKeyValueMetadata(keyValueMetadata []format.KeyValue, key string) (value string, ok bool) {
	i := sort.Search(len(keyValueMetadata), func(i int) bool {
		return keyValueMetadata[i].Key >= key
	})
	if i == len(keyValueMetadata) || keyValueMetadata[i].Key != key {
		return "", false
	}
	return keyValueMetadata[i].Value, true
}

type fileRowGroup struct {
	schema   *Schema
	rowGroup *format.RowGroup
	columns  []ColumnChunk
	sorting  []SortingColumn
}

func (g *fileRowGroup) init(file *File, schema *Schema, columns []*Column, rowGroup *format.RowGroup) {
	g.schema = schema
	g.rowGroup = rowGroup
	g.columns = make([]ColumnChunk, len(rowGroup.Columns))
	g.sorting = make([]SortingColumn, len(rowGroup.SortingColumns))
	fileColumnChunks := make([]fileColumnChunk, len(rowGroup.Columns))

	for i := range g.columns {
		fileColumnChunks[i] = fileColumnChunk{
			file:     file,
			column:   columns[i],
			rowGroup: rowGroup,
			chunk:    &rowGroup.Columns[i],
		}

		if file.hasIndexes() {
			j := (int(rowGroup.Ordinal) * len(columns)) + i
			fileColumnChunks[i].columnIndex = &file.columnIndexes[j]
			fileColumnChunks[i].offsetIndex = &file.offsetIndexes[j]
		}

		g.columns[i] = &fileColumnChunks[i]
	}

	for i := range g.sorting {
		g.sorting[i] = &fileSortingColumn{
			column:     columns[rowGroup.SortingColumns[i].ColumnIdx],
			descending: rowGroup.SortingColumns[i].Descending,
			nullsFirst: rowGroup.SortingColumns[i].NullsFirst,
		}
	}
}

func (g *fileRowGroup) Schema() *Schema                 { return g.schema }
func (g *fileRowGroup) NumRows() int64                  { return g.rowGroup.NumRows }
func (g *fileRowGroup) ColumnChunks() []ColumnChunk     { return g.columns }
func (g *fileRowGroup) SortingColumns() []SortingColumn { return g.sorting }
func (g *fileRowGroup) Rows() Rows                      { return &rowGroupRowReader{rowGroup: g} }

type fileSortingColumn struct {
	column     *Column
	descending bool
	nullsFirst bool
}

func (s *fileSortingColumn) Path() []string   { return s.column.Path() }
func (s *fileSortingColumn) Descending() bool { return s.descending }
func (s *fileSortingColumn) NullsFirst() bool { return s.nullsFirst }

type fileColumnChunk struct {
	file        *File
	column      *Column
	bloomFilter *bloomFilter
	rowGroup    *format.RowGroup
	columnIndex *format.ColumnIndex
	offsetIndex *format.OffsetIndex
	chunk       *format.ColumnChunk
}

func (c *fileColumnChunk) Type() Type {
	return c.column.Type()
}

func (c *fileColumnChunk) Column() int {
	return int(c.column.Index())
}

func (c *fileColumnChunk) Pages() Pages {
	r := new(filePages)
	c.setPagesOn(r)
	return r
}

func (c *fileColumnChunk) setPagesOn(r *filePages) {
	r.column = c
	r.page = filePage{
		column:     c.column,
		columnType: c.column.Type(),
		codec:      c.chunk.MetaData.Codec,
	}
	r.baseOffset = c.chunk.MetaData.DataPageOffset
	r.dataOffset = r.baseOffset
	if c.chunk.MetaData.DictionaryPageOffset != 0 {
		r.baseOffset = c.chunk.MetaData.DictionaryPageOffset
		r.dictOffset = r.baseOffset
	}
	r.section = io.NewSectionReader(c.file, r.baseOffset, c.chunk.MetaData.TotalCompressedSize)
	r.rbuf = bufio.NewReaderSize(r.section, defaultReadBufferSize)
	r.section.Seek(r.dataOffset-r.baseOffset, io.SeekStart)
	r.decoder.Reset(r.protocol.NewReader(r.rbuf))
}

func (c *fileColumnChunk) ColumnIndex() ColumnIndex {
	if c.columnIndex == nil {
		return nil
	}
	return fileColumnIndex{c}
}

func (c *fileColumnChunk) OffsetIndex() OffsetIndex {
	if c.offsetIndex == nil {
		return nil
	}
	return (*fileOffsetIndex)(c.offsetIndex)
}

func (c *fileColumnChunk) BloomFilter() BloomFilter {
	if c.bloomFilter == nil {
		return nil
	}
	return c.bloomFilter
}

func (c *fileColumnChunk) NumValues() int64 {
	return c.chunk.MetaData.NumValues
}

type filePages struct {
	column     *fileColumnChunk
	protocol   thrift.CompactProtocol
	decoder    thrift.Decoder
	baseOffset int64
	dictOffset int64
	dataOffset int64

	section *io.SectionReader
	rbuf    *bufio.Reader

	page filePage
	skip int64
}

func (r *filePages) readPage() (*filePage, error) {
	r.page.header = format.PageHeader{}

	/*
		h := &r.page.header
			h.Type = 0
			h.UncompressedPageSize = 0
			h.CompressedPageSize = 0
			h.CRC = 0

			if h.DataPageHeader != nil {
				*h.DataPageHeader = format.DataPageHeader{}
			}
			if h.IndexPageHeader != nil {
				h.IndexPageHeader = nil
			}
			if h.DictionaryPageHeader != nil {
				h.DictionaryPageHeader = nil
			}
			if h.DataPageHeaderV2 != nil {
				*h.DataPageHeaderV2 = format.DataPageHeaderV2{}
			}
	*/

	if err := r.decoder.Decode(&r.page.header); err != nil {
		if err != io.EOF {
			err = fmt.Errorf("decoding page header: %w", err)
		}
		return nil, err
	}

	compressedPageSize := int(r.page.header.CompressedPageSize)
	if cap(r.page.data) < compressedPageSize {
		r.page.data = make([]byte, compressedPageSize)
	} else {
		r.page.data = r.page.data[:compressedPageSize]
	}

	_, err := io.ReadFull(r.rbuf, r.page.data)
	if err != nil {
		return nil, fmt.Errorf("reading page %d of column %q", r.page.index, r.page.columnPath())
	}

	if r.page.header.CRC != 0 {
		headerChecksum := uint32(r.page.header.CRC)
		bufferChecksum := crc32.ChecksumIEEE(r.page.data)

		if headerChecksum != bufferChecksum {
			// The parquet specs indicate that corruption errors could be
			// handled gracefully by skipping pages, tho this may not always
			// be practical. Depending on how the pages are consumed,
			// missing rows may cause unpredictable behaviors in algorithms.
			//
			// For now, we assume these errors to be fatal, but we may
			// revisit later and improve error handling to be more resilient
			// to data corruption.
			return nil, fmt.Errorf("crc32 checksum mismatch in page %d of column %q: 0x%08X != 0x%08X: %w",
				r.page.index,
				r.page.columnPath(),
				headerChecksum,
				bufferChecksum,
				ErrCorrupted,
			)
		}
	}

	if r.column.columnIndex != nil {
		err = r.page.parseColumnIndex(r.column.columnIndex)
	} else {
		err = r.page.parseStatistics()
	}
	return &r.page, err
}

func (r *filePages) readDictionary() error {
	if _, err := r.section.Seek(r.dictOffset-r.baseOffset, io.SeekStart); err != nil {
		return fmt.Errorf("seeking to dictionary page offset: %w", err)
	}
	r.rbuf.Reset(r.section)
	p, err := r.readPage()
	if err != nil {
		return err
	}
	return r.readDictionaryPage(p)
}

func (r *filePages) readDictionaryPage(p *filePage) error {
	pageData, err := p.decompress(p.data)
	if err != nil {
		return fmt.Errorf("decompressing dictionary page of column %q: %w", p.columnPath(), err)
	}

	enc := p.header.DictionaryPageHeader.Encoding
	dec := LookupEncoding(enc).NewDecoder(bytes.NewReader(pageData))

	columnIndex := r.column.Column()
	numValues := int(p.NumValues())
	dict, err := p.columnType.ReadDictionary(columnIndex, numValues, dec)

	if err != nil {
		return fmt.Errorf("reading dictionary of column %q: %w", p.columnPath(), err)
	}

	r.page.dictionary = dict
	r.page.columnType = dict.Type()
	return nil
}

func (r *filePages) ReadPage() (Page, error) {
	if r.page.dictionary == nil && r.dictOffset > 0 {
		if err := r.readDictionary(); err != nil {
			return nil, err
		}
	}

	for {
		p, err := r.readPage()
		if err != nil {
			return nil, err
		}

		// Sometimes parquet files do not have the dictionary page offset
		// recorded in the column metadata. We account for this by lazily
		// checking whether the first page is a dictionary page.
		if p.index == 0 && p.header.Type == format.DictionaryPage && r.page.dictionary == nil {
			offset, err := r.section.Seek(0, io.SeekCurrent)
			if err != nil {
				return nil, err
			}
			r.dictOffset = r.baseOffset
			r.dataOffset = r.baseOffset + offset
			if err := r.readDictionaryPage(p); err != nil {
				return nil, err
			}
			continue
		}

		p.index++
		if r.skip == 0 {
			return p, nil
		}

		numRows := p.NumRows()
		if numRows > r.skip {
			seek := r.skip
			r.skip = 0
			if seek > 0 {
				return p.Buffer().Slice(seek, numRows), nil
			}
			return p, nil
		}

		r.skip -= numRows
	}
}

func (r *filePages) SeekToRow(rowIndex int64) (err error) {
	if r.column.offsetIndex == nil {
		_, err = r.section.Seek(r.dataOffset-r.baseOffset, io.SeekStart)
		r.skip = rowIndex
		r.page.index = 0
	} else {
		pages := r.column.offsetIndex.PageLocations
		index := sort.Search(len(pages), func(i int) bool {
			return pages[i].FirstRowIndex > rowIndex
		}) - 1
		if index < 0 {
			return ErrSeekOutOfRange
		}
		_, err = r.section.Seek(pages[index].Offset-r.baseOffset, io.SeekStart)
		r.skip = rowIndex - pages[index].FirstRowIndex
		r.page.index = index
	}
	r.rbuf.Reset(r.section)
	return err
}

type filePage struct {
	column     *Column
	columnType Type
	dictionary Dictionary

	codec  format.CompressionCodec
	header format.PageHeader
	data   []byte
	buffer []byte

	index     int
	minValue  Value
	maxValue  Value
	hasBounds bool
}

var (
	errPageIndexExceedsColumnIndexNullPages  = errors.New("page index exceeds column index null pages")
	errPageIndexExceedsColumnIndexMinValues  = errors.New("page index exceeds column index min values")
	errPageIndexExceedsColumnIndexMaxValues  = errors.New("page index exceeds column index max values")
	errPageIndexExceedsColumnIndexNullCounts = errors.New("page index exceeds column index null counts")
)

func (p *filePage) decompress(pageData []byte) ([]byte, error) {
	if p.codec != format.Uncompressed {
		var err error
		p.buffer, err = LookupCompressionCodec(p.codec).Decode(p.buffer[:0], pageData)
		if err != nil {
			return nil, err
		}
		pageData = p.buffer
	}
	return pageData, nil
}

func (p *filePage) statistics() *format.Statistics {
	switch p.header.Type {
	case format.DataPageV2:
		return &p.header.DataPageHeaderV2.Statistics
	case format.DataPage:
		return &p.header.DataPageHeader.Statistics
	default:
		return nil
	}
}

func (p *filePage) parseColumnIndex(columnIndex *format.ColumnIndex) (err error) {
	if p.index >= len(columnIndex.NullPages) {
		return p.errColumnIndex(errPageIndexExceedsColumnIndexNullPages)
	}
	if p.index >= len(columnIndex.MinValues) {
		return p.errColumnIndex(errPageIndexExceedsColumnIndexMinValues)
	}
	if p.index >= len(columnIndex.MaxValues) {
		return p.errColumnIndex(errPageIndexExceedsColumnIndexMaxValues)
	}
	if p.index >= len(columnIndex.NullCounts) {
		return p.errColumnIndex(errPageIndexExceedsColumnIndexNullCounts)
	}

	minValue := columnIndex.MinValues[p.index]
	maxValue := columnIndex.MaxValues[p.index]

	if stats := p.statistics(); stats != nil {
		if stats.MinValue == nil {
			stats.MinValue = minValue
		}
		if stats.MaxValue == nil {
			stats.MaxValue = maxValue
		}
		if stats.NullCount == 0 {
			stats.NullCount = columnIndex.NullCounts[p.index]
		}
	}

	if columnIndex.NullPages[p.index] {
		p.minValue = Value{}
		p.maxValue = Value{}
		p.hasBounds = false
	} else {
		kind := p.columnType.Kind()
		p.minValue, err = parseValue(kind, minValue)
		if err != nil {
			return p.errColumnIndex(err)
		}
		p.maxValue, err = parseValue(kind, maxValue)
		if err != nil {
			return p.errColumnIndex(err)
		}
		p.hasBounds = true
	}

	return nil
}

func (p *filePage) parseStatistics() (err error) {
	kind := p.columnType.Kind()
	stats := p.statistics()

	if stats == nil {
		// The column has no index and page has no statistics,
		// default to reporting that the min and max are both null.
		p.minValue = Value{}
		p.maxValue = Value{}
		p.hasBounds = false
		return nil
	}

	if stats.MinValue == nil {
		p.minValue = Value{}
	} else {
		p.minValue, err = parseValue(kind, stats.MinValue)
		if err != nil {
			return p.errStatistics(err)
		}
	}

	if stats.MaxValue == nil {
		p.maxValue = Value{}
	} else {
		p.maxValue, err = parseValue(kind, stats.MaxValue)
		if err != nil {
			return p.errStatistics(err)
		}
	}

	p.hasBounds = true
	return nil
}

func (p *filePage) errColumnIndex(err error) error {
	return fmt.Errorf("reading bounds of page %d from index of column %q: %w", p.index, p.columnPath(), err)
}

func (p *filePage) errStatistics(err error) error {
	return fmt.Errorf("reading bounds of page %d from statistics in column %q: %w", p.index, p.columnPath(), err)
}

func (p *filePage) columnPath() columnPath {
	return columnPath(p.column.Path())
}

func (p *filePage) Column() int {
	return int(p.column.Index())
}

func (p *filePage) Dictionary() Dictionary {
	return p.dictionary
}

func (p *filePage) NumRows() int64 {
	switch p.header.Type {
	case format.DataPageV2:
		return int64(p.header.DataPageHeaderV2.NumRows)
	default:
		return 0
	}
}

func (p *filePage) NumValues() int64 {
	switch p.header.Type {
	case format.DataPageV2:
		return int64(p.header.DataPageHeaderV2.NumValues)
	case format.DataPage:
		return int64(p.header.DataPageHeader.NumValues)
	case format.DictionaryPage:
		return int64(p.header.DictionaryPageHeader.NumValues)
	default:
		return 0
	}
}

func (p *filePage) NumNulls() int64 {
	switch p.header.Type {
	case format.DataPageV2:
		return int64(p.header.DataPageHeaderV2.NumNulls)
	case format.DataPage:
		return p.header.DataPageHeader.Statistics.NullCount
	default:
		return 0
	}
}

func (p *filePage) Bounds() (min, max Value, ok bool) {
	return p.minValue, p.maxValue, p.hasBounds
}

func (p *filePage) Size() int64 {
	return int64(p.header.UncompressedPageSize)
}

func (p *filePage) Values() ValueReader {
	v, err := p.values()
	if err != nil {
		v = &errorValueReader{err}
	}
	return v
}

func (p *filePage) values() (ValueReader, error) {
	var repetitionLevels []byte
	var definitionLevels []byte
	var pageEncoding format.Encoding
	var pageData = p.data
	var numValues int
	var err error

	maxRepetitionLevel := p.column.maxRepetitionLevel
	maxDefinitionLevel := p.column.maxDefinitionLevel

	switch p.header.Type {
	case format.DataPageV2:
		header := p.header.DataPageHeaderV2
		repetitionLevels, definitionLevels, pageData, err = readDataPageV2(header, pageData)
		if err != nil {
			return nil, fmt.Errorf("initializing v2 reader for page of column %q: %w", p.columnPath(), err)
		}
		if p.codec != format.Uncompressed && (header.IsCompressed == nil || *header.IsCompressed) {
			if pageData, err = p.decompress(pageData); err != nil {
				return nil, fmt.Errorf("decompressing data page v2 of column %q: %w", p.columnPath(), err)
			}
		}
		pageEncoding = header.Encoding
		numValues = int(header.NumValues)

	case format.DataPage:
		if pageData, err = p.decompress(pageData); err != nil {
			return nil, fmt.Errorf("decompressing data page v1 of column %q: %w", p.columnPath(), err)
		}
		repetitionLevels, definitionLevels, pageData, err = readDataPageV1(maxRepetitionLevel, maxDefinitionLevel, pageData)
		if err != nil {
			return nil, fmt.Errorf("initializing v1 reader for page of column %q: %w", p.columnPath(), err)
		}
		header := p.header.DataPageHeader
		pageEncoding = header.Encoding
		numValues = int(header.NumValues)

	default:
		return nil, fmt.Errorf("cannot read values of type %s from page of column %q", p.header.Type, p.columnPath())
	}

	// In some legacy configurations, the PLAIN_DICTIONARY encoding is used on
	// data page headers to indicate that the page contains indexes into the
	// dictionary page, tho it is still encoded using the RLE encoding in this
	// case, so we convert the encoding to RLE_DICTIONARY to simplify.
	switch pageEncoding {
	case format.PlainDictionary:
		pageEncoding = format.RLEDictionary
	}

	pageDecoder := LookupEncoding(pageEncoding).NewDecoder(bytes.NewReader(pageData))
	reader := p.columnType.NewColumnReader(int(p.column.index), defaultReadBufferSize)
	reader.Reset(numValues, pageDecoder)

	hasLevels := maxRepetitionLevel > 0 || maxDefinitionLevel > 0
	if hasLevels {
		repetitions := RLE.NewDecoder(bytes.NewReader(repetitionLevels))
		definitions := RLE.NewDecoder(bytes.NewReader(definitionLevels))
		fileReader := newFileColumnReader(reader, maxRepetitionLevel, maxDefinitionLevel, defaultReadBufferSize)
		fileReader.reset(numValues, repetitions, definitions, pageDecoder)
		reader = fileReader
	}

	return reader, nil
}

func (p *filePage) Buffer() BufferedPage {
	bufferedPage := p.column.Type().NewColumnBuffer(p.Column(), int(p.Size()))
	_, err := CopyValues(bufferedPage, p.Values())
	if err != nil {
		return &errorPage{err: err, columnIndex: p.Column()}
	}
	return bufferedPage.Page()
}

func (p *filePage) PageHeader() PageHeader {
	switch p.header.Type {
	case format.DataPageV2:
		return DataPageHeaderV2{p.header.DataPageHeaderV2}
	case format.DataPage:
		return DataPageHeaderV1{p.header.DataPageHeader}
	case format.DictionaryPage:
		return DictionaryPageHeader{p.header.DictionaryPageHeader}
	default:
		return unknownPageHeader{&p.header}
	}
}

func (p *filePage) PageData() io.Reader { return bytes.NewReader(p.data) }

func (p *filePage) PageSize() int64 { return int64(p.header.CompressedPageSize) }

func (p *filePage) CRC() uint32 { return uint32(p.header.CRC) }

func readDataPageV1(maxRepetitionLevel, maxDefinitionLevel int8, page []byte) (repetitionLevels, definitionLevels, data []byte, err error) {
	data = page
	if maxRepetitionLevel > 0 {
		repetitionLevels, data, err = readDataPageV1Level(data, "repetition")
		if err != nil {
			return nil, nil, page, err
		}
	}
	if maxDefinitionLevel > 0 {
		definitionLevels, data, err = readDataPageV1Level(data, "definition")
		if err != nil {
			return nil, nil, page, err
		}
	}
	return repetitionLevels, definitionLevels, data, nil
}

func readDataPageV1Level(page []byte, typ string) (level, data []byte, err error) {
	size, page, err := read(page, 4)
	if err != nil {
		return nil, page, fmt.Errorf("reading %s level: %w", typ, err)
	}
	return read(page, int(binary.LittleEndian.Uint32(size)))
}

func readDataPageV2(header *format.DataPageHeaderV2, page []byte) (repetitionLevels, definitionLevels, data []byte, err error) {
	repetitionLevelsByteLength := header.RepetitionLevelsByteLength
	definitionLevelsByteLength := header.DefinitionLevelsByteLength
	data = page
	if repetitionLevelsByteLength > 0 {
		repetitionLevels, data, err = readDataPageV2Level(data, repetitionLevelsByteLength, "repetition")
		if err != nil {
			return nil, nil, page, err
		}
	}
	if definitionLevelsByteLength > 0 {
		definitionLevels, data, err = readDataPageV2Level(data, definitionLevelsByteLength, "definition")
		if err != nil {
			return nil, nil, page, err
		}
	}
	return repetitionLevels, definitionLevels, data, nil
}

func readDataPageV2Level(page []byte, size int32, typ string) (level, data []byte, err error) {
	level, data, err = read(page, int(size))
	if err != nil {
		err = fmt.Errorf("reading %s level: %w", typ, err)
	}
	return level, data, err
}

func read(data []byte, size int) (head, tail []byte, err error) {
	if len(data) < size {
		return nil, data, io.ErrUnexpectedEOF
	}
	return data[:size], data[size:], nil
}

var (
	_ CompressedPage = (*filePage)(nil)
)
