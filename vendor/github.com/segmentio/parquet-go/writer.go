package parquet

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"sort"

	"github.com/segmentio/encoding/thrift"
	"github.com/segmentio/parquet-go/compress"
	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/encoding/plain"
	"github.com/segmentio/parquet-go/format"
	"github.com/segmentio/parquet-go/internal/bits"
)

// A Writer uses a parquet schema and sequence of Go values to produce a parquet
// file to an io.Writer.
//
// This example showcases a typical use of parquet writers:
//
//	writer := parquet.NewWriter(output)
//
//	for _, row := range rows {
//		if err := writer.Write(row); err != nil {
//			...
//		}
//	}
//
//	if err := writer.Close(); err != nil {
//		...
//	}
//
// The Writer type optimizes for minimal memory usage, each page is written as
// soon as it has been filled so only a single page per column needs to be held
// in memory and as a result, there are no opportunities to sort rows within an
// entire row group. Programs that need to produce parquet files with sorted
// row groups should use the Buffer type to buffer and sort the rows prior to
// writing them to a Writer.
type Writer struct {
	output io.Writer
	config *WriterConfig
	schema *Schema
	writer *writer
	values []Value
}

// NewWriter constructs a parquet writer writing a file to the given io.Writer.
//
// The function panics if the writer configuration is invalid. Programs that
// cannot guarantee the validity of the options passed to NewWriter should
// construct the writer configuration independently prior to calling this
// function:
//
//	config, err := parquet.NewWriterConfig(options...)
//	if err != nil {
//		// handle the configuration error
//		...
//	} else {
//		// this call to create a writer is guaranteed not to panic
//		writer := parquet.NewWriter(output, config)
//		...
//	}
//
func NewWriter(output io.Writer, options ...WriterOption) *Writer {
	config, err := NewWriterConfig(options...)
	if err != nil {
		panic(err)
	}
	w := &Writer{
		output: output,
		config: config,
	}
	if config.Schema != nil {
		w.configure(config.Schema)
	}
	return w
}

func (w *Writer) configure(schema *Schema) {
	if schema != nil {
		w.config.Schema = schema
		w.schema = schema
		w.writer = newWriter(w.output, w.config)
	}
}

// Close must be called after all values were produced to the writer in order to
// flush all buffers and write the parquet footer.
func (w *Writer) Close() error {
	if w.writer != nil {
		return w.writer.close()
	}
	return nil
}

// Flush flushes all buffers into a row group to the underlying io.Writer.
//
// Flush is called automatically on Close, it is only useful to call explicitly
// if the application needs to limit the size of row groups or wants to produce
// multiple row groups per file.
func (w *Writer) Flush() error {
	if w.writer != nil {
		return w.writer.flush()
	}
	return nil
}

// Reset clears the state of the writer without flushing any of the buffers,
// and setting the output to the io.Writer passed as argument, allowing the
// writer to be reused to produce another parquet file.
//
// Reset may be called at any time, including after a writer was closed.
func (w *Writer) Reset(output io.Writer) {
	if w.output = output; w.writer != nil {
		w.writer.reset(w.output)
	}
}

// Write is called to write another row to the parquet file.
//
// The method uses the parquet schema configured on w to traverse the Go value
// and decompose it into a set of columns and values. If no schema were passed
// to NewWriter, it is deducted from the Go type of the row, which then have to
// be a struct or pointer to struct.
func (w *Writer) Write(row interface{}) error {
	if w.schema == nil {
		w.configure(SchemaOf(row))
	}
	defer func() {
		clearValues(w.values)
	}()
	w.values = w.schema.Deconstruct(w.values[:0], row)
	return w.WriteRow(w.values)
}

// WriteRow is called to write another row to the parquet file.
//
// The Writer must have been given a schema when NewWriter was called, otherwise
// the structure of the parquet file cannot be determined from the row only.
//
// The row is expected to contain values for each column of the writer's schema,
// in the order produced by the parquet.(*Schema).Deconstruct method.
func (w *Writer) WriteRow(row Row) error { return w.writer.WriteRow(row) }

// WriteRowGroup writes a row group to the parquet file.
//
// Buffered rows will be flushed prior to writing rows from the group, unless
// the row group was empty in which case nothing is written to the file.
//
// The content of the row group is flushed to the writer; after the method
// returns successfully, the row group will be empty and in ready to be reused.
func (w *Writer) WriteRowGroup(rowGroup RowGroup) (int64, error) {
	rowGroupSchema := rowGroup.Schema()
	switch {
	case rowGroupSchema == nil:
		return 0, ErrRowGroupSchemaMissing
	case w.schema == nil:
		w.configure(rowGroupSchema)
	case !nodesAreEqual(w.schema, rowGroupSchema):
		return 0, ErrRowGroupSchemaMismatch
	}
	if err := w.writer.flush(); err != nil {
		return 0, err
	}
	w.writer.configureBloomFilters(rowGroup.ColumnChunks())
	n, err := CopyRows(w.writer, rowGroup.Rows())
	if err != nil {
		return n, err
	}
	return w.writer.writeRowGroup(rowGroup.Schema(), rowGroup.SortingColumns())
}

// ReadRowsFrom reads rows from the reader passed as arguments and writes them
// to w.
//
// This is similar to calling WriteRow repeatedly, but will be more efficient
// if optimizations are supported by the reader.
func (w *Writer) ReadRowsFrom(rows RowReader) (written int64, err error) {
	if w.schema == nil {
		if r, ok := rows.(RowReaderWithSchema); ok {
			w.configure(r.Schema())
		}
	}
	written, w.values, err = copyRows(w.writer, rows, w.values[:0])
	return written, err
}

// Schema returns the schema of rows written by w.
//
// The returned value will be nil if no schema has yet been configured on w.
func (w *Writer) Schema() *Schema { return w.schema }

type writer struct {
	buffer *bufio.Writer
	writer offsetTrackingWriter

	createdBy string
	metadata  []format.KeyValue

	columns       []*writerColumn
	columnChunk   []format.ColumnChunk
	columnIndex   []format.ColumnIndex
	offsetIndex   []format.OffsetIndex
	encodingStats [][]format.PageEncodingStats

	columnOrders   []format.ColumnOrder
	schemaElements []format.SchemaElement
	rowGroups      []format.RowGroup
	columnIndexes  [][]format.ColumnIndex
	offsetIndexes  [][]format.OffsetIndex
	sortingColumns []format.SortingColumn
}

type writerBuffers struct {
	compressed []byte
	header     bytes.Buffer
	page       bytes.Buffer
	reader     bytes.Reader
}

func newWriter(output io.Writer, config *WriterConfig) *writer {
	w := new(writer)
	if config.WriteBufferSize <= 0 {
		w.writer.Reset(output)
	} else {
		w.buffer = bufio.NewWriterSize(output, config.WriteBufferSize)
		w.writer.Reset(w.buffer)
	}
	w.createdBy = config.CreatedBy
	w.metadata = make([]format.KeyValue, 0, len(config.KeyValueMetadata))
	for k, v := range config.KeyValueMetadata {
		w.metadata = append(w.metadata, format.KeyValue{Key: k, Value: v})
	}
	sortKeyValueMetadata(w.metadata)
	w.sortingColumns = make([]format.SortingColumn, len(config.SortingColumns))

	config.Schema.forEachNode(func(name string, node Node) {
		nodeType := node.Type()

		repetitionType := (*format.FieldRepetitionType)(nil)
		if node != config.Schema { // the root has no repetition type
			repetitionType = fieldRepetitionTypePtrOf(node)
		}

		// For backward compatibility with older readers, the parquet specification
		// recommends to set the scale and precision on schema elements when the
		// column is of logical type decimal.
		logicalType := nodeType.LogicalType()
		scale, precision := (*int32)(nil), (*int32)(nil)
		if logicalType != nil && logicalType.Decimal != nil {
			scale = &logicalType.Decimal.Scale
			precision = &logicalType.Decimal.Precision
		}

		typeLength := (*int32)(nil)
		if n := int32(nodeType.Length()); n > 0 {
			typeLength = &n
		}

		w.schemaElements = append(w.schemaElements, format.SchemaElement{
			Type:           nodeType.PhysicalType(),
			TypeLength:     typeLength,
			RepetitionType: repetitionType,
			Name:           name,
			NumChildren:    int32(len(node.Fields())),
			ConvertedType:  nodeType.ConvertedType(),
			Scale:          scale,
			Precision:      precision,
			LogicalType:    logicalType,
		})
	})

	dataPageType := format.DataPage
	if config.DataPageVersion == 2 {
		dataPageType = format.DataPageV2
	}

	defaultCompression := config.Compression
	if defaultCompression == nil {
		defaultCompression = &Uncompressed
	}

	// Those buffers are scratch space used to generate the page header and
	// content, they are shared by all column chunks because they are only
	// used during calls to writeDictionaryPage or writeDataPage, which are
	// not done concurrently.
	buffers := new(writerBuffers)

	forEachLeafColumnOf(config.Schema, func(leaf leafColumn) {
		encoding := encodingOf(leaf.node)
		dictionary := Dictionary(nil)
		columnType := leaf.node.Type()
		columnIndex := int(leaf.columnIndex)
		compression := leaf.node.Compression()

		if compression == nil {
			compression = defaultCompression
		}

		if isDictionaryEncoding(encoding) {
			dictionary = columnType.NewDictionary(columnIndex, defaultDictBufferSize)
			columnType = dictionary.Type()
		}

		c := &writerColumn{
			buffers:            buffers,
			pool:               config.ColumnPageBuffers,
			columnPath:         leaf.path,
			columnType:         columnType,
			columnIndex:        columnType.NewColumnIndexer(config.ColumnIndexSizeLimit),
			columnFilter:       searchBloomFilterColumn(config.BloomFilters, leaf.path),
			compression:        compression,
			dictionary:         dictionary,
			dataPageType:       dataPageType,
			maxRepetitionLevel: leaf.maxRepetitionLevel,
			maxDefinitionLevel: leaf.maxDefinitionLevel,
			bufferIndex:        int32(leaf.columnIndex),
			bufferSize:         int32(config.PageBufferSize),
			writePageStats:     config.DataPageStatistics,
			encodings:          make([]format.Encoding, 0, 3),
			// Data pages in version 2 can omit compression when dictionary
			// encoding is employed; only the dictionary page needs to be
			// compressed, the data pages are encoded with the hybrid
			// RLE/Bit-Pack encoding which doesn't benefit from an extra
			// compression layer.
			isCompressed: compression.CompressionCodec() != format.Uncompressed && (dataPageType != format.DataPageV2 || dictionary == nil),
		}

		c.header.encoder.Reset(c.header.protocol.NewWriter(&buffers.header))

		if leaf.maxRepetitionLevel > 0 {
			c.insert = (*writerColumn).insertRepeated
			c.commit = (*writerColumn).commitRepeated
			c.values = make([]Value, 0, 10)
		} else {
			c.insert = (*writerColumn).writeRow
			c.commit = func(*writerColumn) error { return nil }
		}

		if leaf.maxDefinitionLevel > 0 {
			c.levels.encoder = RLE.NewEncoder(nil)
			c.encodings = addEncoding(c.encodings, format.RLE)
		}

		if isDictionaryEncoding(encoding) {
			c.encodings = addEncoding(c.encodings, format.Plain)
		}

		c.page.encoder = encoding.NewEncoder(nil)
		c.page.encoding = encoding.Encoding()
		c.encodings = addEncoding(c.encodings, c.page.encoding)
		sortPageEncodings(c.encodings)

		w.columns = append(w.columns, c)

		if sortingIndex := searchSortingColumn(config.SortingColumns, leaf.path); sortingIndex < len(w.sortingColumns) {
			w.sortingColumns[sortingIndex] = format.SortingColumn{
				ColumnIdx:  int32(leaf.columnIndex),
				Descending: config.SortingColumns[sortingIndex].Descending(),
				NullsFirst: config.SortingColumns[sortingIndex].NullsFirst(),
			}
		}
	})

	w.columnChunk = make([]format.ColumnChunk, len(w.columns))
	w.columnIndex = make([]format.ColumnIndex, len(w.columns))
	w.offsetIndex = make([]format.OffsetIndex, len(w.columns))
	w.columnOrders = make([]format.ColumnOrder, len(w.columns))

	for i, c := range w.columns {
		w.columnChunk[i] = format.ColumnChunk{
			MetaData: format.ColumnMetaData{
				Type:             format.Type(c.columnType.Kind()),
				Encoding:         c.encodings,
				PathInSchema:     c.columnPath,
				Codec:            c.compression.CompressionCodec(),
				KeyValueMetadata: nil, // TODO
			},
		}
	}

	for i, c := range w.columns {
		c.columnChunk = &w.columnChunk[i]
		c.offsetIndex = &w.offsetIndex[i]
	}

	for i, c := range w.columns {
		w.columnOrders[i] = *c.columnType.ColumnOrder()
	}

	return w
}

func (w *writer) reset(writer io.Writer) {
	if w.buffer == nil {
		w.writer.Reset(writer)
	} else {
		w.buffer.Reset(writer)
		w.writer.Reset(w.buffer)
	}
	for _, c := range w.columns {
		c.reset()
	}
	for i := range w.rowGroups {
		w.rowGroups[i] = format.RowGroup{}
	}
	for i := range w.columnIndexes {
		w.columnIndexes[i] = nil
	}
	for i := range w.offsetIndexes {
		w.offsetIndexes[i] = nil
	}
	w.rowGroups = w.rowGroups[:0]
	w.columnIndexes = w.columnIndexes[:0]
	w.offsetIndexes = w.offsetIndexes[:0]
}

func (w *writer) close() error {
	if err := w.writeFileHeader(); err != nil {
		return err
	}
	if err := w.flush(); err != nil {
		return err
	}
	if err := w.writeFileFooter(); err != nil {
		return err
	}
	if w.buffer != nil {
		return w.buffer.Flush()
	}
	return nil
}

func (w *writer) flush() error {
	_, err := w.writeRowGroup(nil, nil)
	return err
}

func (w *writer) writeFileHeader() error {
	if w.writer.writer == nil {
		return io.ErrClosedPipe
	}
	if w.writer.offset == 0 {
		_, err := w.writer.WriteString("PAR1")
		return err
	}
	return nil
}

func (w *writer) configureBloomFilters(columnChunks []ColumnChunk) {
	for i, c := range w.columns {
		if c.columnFilter != nil {
			c.page.filter = c.newBloomFilterEncoder(columnChunks[i].NumValues())
		}
	}
}

func (w *writer) writeFileFooter() error {
	// The page index is composed of two sections: column and offset indexes.
	// They are written after the row groups, right before the footer (which
	// is written by the parent Writer.Close call).
	//
	// This section both writes the page index and generates the values of
	// ColumnIndexOffset, ColumnIndexLength, OffsetIndexOffset, and
	// OffsetIndexLength in the corresponding columns of the file metadata.
	//
	// Note: the page index is always written, even if we created data pages v1
	// because the parquet format is backward compatible in this case. Older
	// readers will simply ignore this section since they do not know how to
	// decode its content, nor have loaded any metadata to reference it.
	protocol := new(thrift.CompactProtocol)
	encoder := thrift.NewEncoder(protocol.NewWriter(&w.writer))

	for i, columnIndexes := range w.columnIndexes {
		rowGroup := &w.rowGroups[i]
		for j := range columnIndexes {
			column := &rowGroup.Columns[j]
			column.ColumnIndexOffset = w.writer.offset
			if err := encoder.Encode(&columnIndexes[j]); err != nil {
				return err
			}
			column.ColumnIndexLength = int32(w.writer.offset - column.ColumnIndexOffset)
		}
	}

	for i, offsetIndexes := range w.offsetIndexes {
		rowGroup := &w.rowGroups[i]
		for j := range offsetIndexes {
			column := &rowGroup.Columns[j]
			column.OffsetIndexOffset = w.writer.offset
			if err := encoder.Encode(&offsetIndexes[j]); err != nil {
				return err
			}
			column.OffsetIndexLength = int32(w.writer.offset - column.OffsetIndexOffset)
		}
	}

	numRows := int64(0)
	for rowGroupIndex := range w.rowGroups {
		numRows += w.rowGroups[rowGroupIndex].NumRows
	}

	footer, err := thrift.Marshal(new(thrift.CompactProtocol), &format.FileMetaData{
		Version:          1,
		Schema:           w.schemaElements,
		NumRows:          numRows,
		RowGroups:        w.rowGroups,
		KeyValueMetadata: w.metadata,
		CreatedBy:        w.createdBy,
		ColumnOrders:     w.columnOrders,
	})
	if err != nil {
		return err
	}

	length := len(footer)
	footer = append(footer, 0, 0, 0, 0)
	footer = append(footer, "PAR1"...)
	binary.LittleEndian.PutUint32(footer[length:], uint32(length))

	_, err = w.writer.Write(footer)
	return err
}

func (w *writer) writeRowGroup(rowGroupSchema *Schema, rowGroupSortingColumns []SortingColumn) (int64, error) {
	numRows := w.columns[0].totalRowCount()
	if numRows == 0 {
		return 0, nil
	}

	defer func() {
		for _, c := range w.columns {
			c.reset()
		}
		for i := range w.columnIndex {
			w.columnIndex[i] = format.ColumnIndex{}
		}
	}()

	for _, c := range w.columns {
		if err := c.flush(); err != nil {
			return 0, err
		}
		if err := c.flushFilterPages(); err != nil {
			return 0, err
		}
	}

	if err := w.writeFileHeader(); err != nil {
		return 0, err
	}
	fileOffset := w.writer.offset

	for _, c := range w.columns {
		if c.page.filter != nil {
			c.columnChunk.MetaData.BloomFilterOffset = w.writer.offset
			if err := c.writeBloomFilter(&w.writer); err != nil {
				return 0, err
			}
		}
	}

	for i, c := range w.columns {
		w.columnIndex[i] = format.ColumnIndex(c.columnIndex.ColumnIndex())

		if c.dictionary != nil {
			c.columnChunk.MetaData.DictionaryPageOffset = w.writer.offset
			if err := c.writeDictionaryPage(&w.writer, c.dictionary); err != nil {
				return 0, fmt.Errorf("writing dictionary page of row group colum %d: %w", i, err)
			}
		}

		dataPageOffset := w.writer.offset
		c.columnChunk.MetaData.DataPageOffset = dataPageOffset
		for j := range c.offsetIndex.PageLocations {
			c.offsetIndex.PageLocations[j].Offset += dataPageOffset
		}

		for _, page := range c.pages {
			if _, err := io.Copy(&w.writer, page); err != nil {
				return 0, fmt.Errorf("writing buffered pages of row group column %d: %w", i, err)
			}
		}
	}

	totalByteSize := int64(0)
	totalCompressedSize := int64(0)

	for i := range w.columnChunk {
		c := &w.columnChunk[i].MetaData
		sortPageEncodingStats(c.EncodingStats)
		totalByteSize += int64(c.TotalUncompressedSize)
		totalCompressedSize += int64(c.TotalCompressedSize)
	}

	sortingColumns := w.sortingColumns
	if len(sortingColumns) == 0 && len(rowGroupSortingColumns) > 0 {
		sortingColumns = make([]format.SortingColumn, 0, len(rowGroupSortingColumns))
		forEachLeafColumnOf(rowGroupSchema, func(leaf leafColumn) {
			if sortingIndex := searchSortingColumn(rowGroupSortingColumns, leaf.path); sortingIndex < len(sortingColumns) {
				sortingColumns[sortingIndex] = format.SortingColumn{
					ColumnIdx:  int32(leaf.columnIndex),
					Descending: rowGroupSortingColumns[sortingIndex].Descending(),
					NullsFirst: rowGroupSortingColumns[sortingIndex].NullsFirst(),
				}
			}
		})
	}

	columns := make([]format.ColumnChunk, len(w.columnChunk))
	copy(columns, w.columnChunk)

	columnIndex := make([]format.ColumnIndex, len(w.columnIndex))
	copy(columnIndex, w.columnIndex)

	offsetIndex := make([]format.OffsetIndex, len(w.offsetIndex))
	copy(offsetIndex, w.offsetIndex)

	w.rowGroups = append(w.rowGroups, format.RowGroup{
		Columns:             columns,
		TotalByteSize:       totalByteSize,
		NumRows:             numRows,
		SortingColumns:      sortingColumns,
		FileOffset:          fileOffset,
		TotalCompressedSize: totalCompressedSize,
		Ordinal:             int16(len(w.rowGroups)),
	})

	w.columnIndexes = append(w.columnIndexes, columnIndex)
	w.offsetIndexes = append(w.offsetIndexes, offsetIndex)
	return numRows, nil
}

func (w *writer) WriteRow(row Row) error {
	for i := range row {
		c := w.columns[row[i].Column()]
		if err := c.insert(c, row[i:i+1]); err != nil {
			return err
		}
	}
	for _, c := range w.columns {
		if err := c.commit(c); err != nil {
			return err
		}
	}
	return nil
}

// The WriteValues method is intended to work in pair with WritePage to allow
// programs to target writing values to specific columns of of the writer.
func (w *writer) WriteValues(values []Value) (numValues int, err error) {
	return w.columns[values[0].Column()].WriteValues(values)
}

// This WritePage method satisfies the PageWriter interface as a mechanism to
// allow writing whole pages of values instead of individual rows. It is called
// indirectly by readers that implement WriteRowsTo and are able to leverage
// the method to optimize writes.
func (w *writer) WritePage(page Page) (int64, error) {
	return w.columns[page.Column()].WritePage(page)
}

type writerColumn struct {
	insert func(*writerColumn, []Value) error
	commit func(*writerColumn) error
	values []Value
	filter []BufferedPage

	pool  PageBufferPool
	pages []io.ReadWriter

	columnPath   columnPath
	columnType   Type
	columnIndex  ColumnIndexer
	columnBuffer ColumnBuffer
	columnFilter BloomFilterColumn
	compression  compress.Codec
	dictionary   Dictionary

	dataPageType       format.PageType
	maxRepetitionLevel int8
	maxDefinitionLevel int8

	buffers *writerBuffers

	levels struct {
		encoder encoding.Encoder
	}

	header struct {
		protocol thrift.CompactProtocol
		encoder  thrift.Encoder
	}

	page struct {
		filter   *bloomFilterEncoder
		encoding format.Encoding
		encoder  encoding.Encoder
	}

	dict struct {
		encoder plain.Encoder
	}

	numRows        int64
	maxValues      int32
	numValues      int32
	bufferIndex    int32
	bufferSize     int32
	writePageStats bool
	isCompressed   bool
	encodings      []format.Encoding

	columnChunk *format.ColumnChunk
	offsetIndex *format.OffsetIndex
}

func (c *writerColumn) reset() {
	if c.columnBuffer != nil {
		c.columnBuffer.Reset()
	}
	if c.columnIndex != nil {
		c.columnIndex.Reset()
	}
	if c.dictionary != nil {
		c.dictionary.Reset()
	}
	for _, page := range c.pages {
		c.pool.PutPageBuffer(page)
	}
	for i := range c.filter {
		c.filter[i] = nil
	}
	for i := range c.pages {
		c.pages[i] = nil
	}
	c.filter = c.filter[:0]
	c.pages = c.pages[:0]
	c.numRows = 0
	c.numValues = 0
	// Reset the fields of column chunks that change between row groups,
	// but keep the ones that remain unchanged.
	c.columnChunk.MetaData.NumValues = 0
	c.columnChunk.MetaData.TotalUncompressedSize = 0
	c.columnChunk.MetaData.TotalCompressedSize = 0
	c.columnChunk.MetaData.DataPageOffset = 0
	c.columnChunk.MetaData.DictionaryPageOffset = 0
	c.columnChunk.MetaData.Statistics = format.Statistics{}
	c.columnChunk.MetaData.EncodingStats = make([]format.PageEncodingStats, 0, cap(c.columnChunk.MetaData.EncodingStats))
	c.columnChunk.MetaData.BloomFilterOffset = 0
	// Retain the previous capacity in the new page locations array, assuming
	// the number of pages should be roughly the same between row groups written
	// by the writer.
	c.offsetIndex.PageLocations = make([]format.PageLocation, 0, cap(c.offsetIndex.PageLocations))
	// Bloom filters may change in size between row groups; we may want to
	// optimize this by retaining the filter and reusing it if needed, but
	// for now we take the simpler approach of freeing it and having the
	// write path lazily reallocate it if the writer is reused.
	c.page.filter = nil
}

func (c *writerColumn) totalRowCount() int64 {
	n := c.numRows
	if c.columnBuffer != nil {
		n += int64(c.columnBuffer.Len())
	}
	return n
}

func (c *writerColumn) canFlush() bool {
	return c.columnBuffer.Size() >= int64(c.bufferSize/2)
}

func (c *writerColumn) flush() (err error) {
	if c.numValues != 0 {
		c.numValues = 0
		defer c.columnBuffer.Reset()
		_, err = c.writeBufferedPage(c.columnBuffer.Page())
	}
	return err
}

func (c *writerColumn) flushFilterPages() error {
	if c.columnFilter != nil {
		numValues := int64(0)
		for _, page := range c.filter {
			numValues += page.NumValues()
		}
		if c.page.filter == nil {
			c.page.filter = c.newBloomFilterEncoder(numValues)
		}

		// If there is a dictionary, we need to only write the dictionary.
		if dict := c.dictionary; dict != nil {
			return dict.Page().WriteTo(c.page.filter)
		}

		for _, page := range c.filter {
			if err := page.WriteTo(c.page.filter); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *writerColumn) insertRepeated(row []Value) error {
	c.values = append(c.values, row...)
	return nil
}

func (c *writerColumn) commitRepeated() error {
	defer func() {
		clearValues(c.values)
		c.values = c.values[:0]
	}()
	return c.writeRow(c.values)
}

func (c *writerColumn) newColumnBuffer() ColumnBuffer {
	column := c.columnType.NewColumnBuffer(int(c.bufferIndex), int(c.bufferSize))
	switch {
	case c.maxRepetitionLevel > 0:
		column = newRepeatedColumnBuffer(column, c.maxRepetitionLevel, c.maxDefinitionLevel, nullsGoLast)
	case c.maxDefinitionLevel > 0:
		column = newOptionalColumnBuffer(column, c.maxDefinitionLevel, nullsGoLast)
	}
	return column
}

func (c *writerColumn) newBloomFilterEncoder(numRows int64) *bloomFilterEncoder {
	const bitsPerValue = 10 // TODO: make this configurable
	return newBloomFilterEncoder(
		c.columnFilter.NewFilter(numRows, bitsPerValue),
		c.columnFilter.Hash(),
	)
}

func (c *writerColumn) writeRow(row []Value) error {
	if c.columnBuffer == nil {
		// Lazily create the row group column so we don't need to allocate it if
		// rows are not written individually to the column.
		c.columnBuffer = c.newColumnBuffer()
		c.maxValues = int32(c.columnBuffer.Cap())
	}

	if c.numValues > 0 && c.numValues > (c.maxValues-int32(len(row))) {
		if err := c.flush(); err != nil {
			return err
		}
	}

	if _, err := c.columnBuffer.WriteValues(row); err != nil {
		return err
	}
	c.numValues += int32(len(row))
	return nil
}

func (c *writerColumn) WriteValues(values []Value) (numValues int, err error) {
	if c.columnBuffer == nil {
		c.columnBuffer = c.newColumnBuffer()
		c.maxValues = int32(c.columnBuffer.Cap())
	}
	numValues, err = c.columnBuffer.WriteValues(values)
	c.numValues += int32(numValues)
	return numValues, err
}

func (c *writerColumn) WritePage(page Page) (numValues int64, err error) {
	// Page write optimizations are only available the column is not reindexing
	// the values. If a dictionary is present, the column needs to see each
	// individual value in order to re-index them in the dictionary.
	if c.dictionary == nil || c.dictionary == page.Dictionary() {
		// If the column had buffered values, we continue writing values from
		// the page into the column buffer if it would have caused producing a
		// page less than half the size of the target; if there were enough
		// buffered values already, we have to flush the buffered page instead
		// otherwise values would get reordered.
		if c.numValues > 0 && c.canFlush() {
			if err := c.flush(); err != nil {
				return 0, err
			}
		}

		// If we were successful at flushing buffered values, we attempt to
		// optimize the write path by copying whole pages without decoding them
		// into a sequence of values.
		if c.numValues == 0 {
			switch p := page.(type) {
			case BufferedPage:
				// Buffered pages may be larger than the target page size on the
				// column, in which case multiple pages get written by slicing
				// the original page into sub-pages.
				err = forEachPageSlice(p, int64(c.bufferSize), func(p BufferedPage) error {
					n, err := c.writeBufferedPage(p)
					numValues += n
					return err
				})
				return numValues, err

			case CompressedPage:
				// Compressed pages are written as-is to the compressed page
				// buffers; those pages should be coming from parquet files that
				// are being copied into a new file, they are simply copied to
				// amortize the cost of decoding and re-encoding the pages, which
				// often includes costly compression steps.
				return c.writeCompressedPage(p)
			}
		}
	}

	// Pages that implement neither of those interfaces can still be
	// written by copying their values into the column buffer and flush
	// them to compressed page buffers as if the program had written
	// rows individually.
	return c.writePageValues(page.Values())
}

func (c *writerColumn) writePageValues(page ValueReader) (numValues int64, err error) {
	numValues, err = CopyValues(c, page)
	if err == nil && c.canFlush() {
		// Always attempt to flush after writing a full page if we have enough
		// buffered values; the intent is to leave the column clean so that
		// subsequent calls to the WritePage method can use optimized write path
		// to bypass buffering.
		err = c.flush()
	}
	return numValues, err
}

func (c *writerColumn) writeBloomFilter(w io.Writer) error {
	e := thrift.NewEncoder(c.header.protocol.NewWriter(w))
	h := bloomFilterHeader(c.columnFilter)
	b := c.page.filter.Bytes()
	h.NumBytes = int32(len(b))
	if err := e.Encode(&h); err != nil {
		return err
	}
	_, err := w.Write(b)
	return err
}

func (c *writerColumn) writeBufferedPage(page BufferedPage) (int64, error) {
	numValues := page.NumValues()
	if numValues == 0 {
		return 0, nil
	}

	buffer := &c.buffers.page
	buffer.Reset()
	repetitionLevelsByteLength := 0
	definitionLevelsByteLength := 0

	switch c.dataPageType {
	case format.DataPageV2:
		if c.maxRepetitionLevel > 0 {
			c.levels.encoder.Reset(buffer)
			c.levels.encoder.SetBitWidth(bits.Len8(c.maxRepetitionLevel))
			c.levels.encoder.EncodeInt8(page.RepetitionLevels())
			repetitionLevelsByteLength = buffer.Len()
		}
		if c.maxDefinitionLevel > 0 {
			c.levels.encoder.Reset(buffer)
			c.levels.encoder.SetBitWidth(bits.Len8(c.maxDefinitionLevel))
			c.levels.encoder.EncodeInt8(page.DefinitionLevels())
			definitionLevelsByteLength = buffer.Len() - repetitionLevelsByteLength
		}

	case format.DataPage:
		// In data pages v1, the repetition and definition levels are prefixed
		// with the 4 bytes length of the sections. While the parquet-format
		// documentation indicates that the length prefix is part of the hybrid
		// RLE/Bit-Pack encoding, this is the only condition where it is used
		// so we treat it as a special case rather than implementing it in the
		// encoding.
		//
		// Reference https://github.com/apache/parquet-format/blob/master/Encodings.md#run-length-encoding--bit-packing-hybrid-rle--3
		lengthPlaceholder := make([]byte, 4)
		if c.maxRepetitionLevel > 0 {
			buffer.Write(lengthPlaceholder)
			offset := buffer.Len()
			c.levels.encoder.Reset(buffer)
			c.levels.encoder.SetBitWidth(bits.Len8(c.maxRepetitionLevel))
			c.levels.encoder.EncodeInt8(page.RepetitionLevels())
			binary.LittleEndian.PutUint32(buffer.Bytes()[offset-4:], uint32(buffer.Len()-offset))
		}
		if c.maxDefinitionLevel > 0 {
			buffer.Write(lengthPlaceholder)
			offset := buffer.Len()
			c.levels.encoder.Reset(buffer)
			c.levels.encoder.SetBitWidth(bits.Len8(c.maxDefinitionLevel))
			c.levels.encoder.EncodeInt8(page.DefinitionLevels())
			binary.LittleEndian.PutUint32(buffer.Bytes()[offset-4:], uint32(buffer.Len()-offset))
		}
	}

	switch {
	case c.page.filter != nil:
		if err := page.WriteTo(c.page.filter); err != nil {
			return 0, err
		}
	case c.columnFilter != nil:
		c.filter = append(c.filter, page.Clone())
	}

	statistics := format.Statistics{}
	if c.writePageStats {
		statistics = c.makePageStatistics(page)
	}

	c.page.encoder.Reset(buffer)
	if err := page.WriteTo(c.page.encoder); err != nil {
		return 0, err
	}

	uncompressedPageSize := buffer.Len()
	pageData := buffer.Bytes()
	if c.isCompressed {
		offset := repetitionLevelsByteLength + definitionLevelsByteLength
		b, err := c.compress(pageData[offset:])
		if err != nil {
			return 0, fmt.Errorf("compressing parquet data page: %w", err)
		}
		if offset == 0 {
			pageData = b
		} else {
			// TODO: can this copy be optimized away?
			buffer.Truncate(offset)
			buffer.Write(b)
			pageData = buffer.Bytes()
		}
	}

	pageHeader := &format.PageHeader{
		Type:                 c.dataPageType,
		UncompressedPageSize: int32(uncompressedPageSize),
		CompressedPageSize:   int32(len(pageData)),
		CRC:                  int32(crc32.ChecksumIEEE(pageData)),
	}

	numRows := page.NumRows()
	numNulls := page.NumNulls()
	switch c.dataPageType {
	case format.DataPage:
		pageHeader.DataPageHeader = &format.DataPageHeader{
			NumValues:               int32(numValues),
			Encoding:                c.page.encoding,
			DefinitionLevelEncoding: format.RLE,
			RepetitionLevelEncoding: format.RLE,
			Statistics:              statistics,
		}
	case format.DataPageV2:
		pageHeader.DataPageHeaderV2 = &format.DataPageHeaderV2{
			NumValues:                  int32(numValues),
			NumNulls:                   int32(numNulls),
			NumRows:                    int32(numRows),
			Encoding:                   c.page.encoding,
			DefinitionLevelsByteLength: int32(definitionLevelsByteLength),
			RepetitionLevelsByteLength: int32(repetitionLevelsByteLength),
			IsCompressed:               &c.isCompressed,
			Statistics:                 statistics,
		}
	}

	header := &c.buffers.header
	header.Reset()
	if err := c.header.encoder.Encode(pageHeader); err != nil {
		return 0, err
	}
	headerSize := int32(header.Len())
	compressedSize := int64(headerSize) + int64(len(pageData))

	reader := &c.buffers.reader
	reader.Reset(pageData)

	if err := c.writePage(compressedSize, header, reader); err != nil {
		return 0, err
	}

	c.recordPageStats(headerSize, pageHeader, page)
	return numValues, nil
}

func (c *writerColumn) writeCompressedPage(page CompressedPage) (int64, error) {
	switch {
	case c.page.filter != nil:
		// TODO: modify the Buffer method to accept some kind of buffer pool as
		// argument so we can use a pre-allocated page buffer to load the page
		// and reduce the memory footprint.
		bufferedPage := page.Buffer()
		// The compressed page must be decompressed here in order to generate
		// the bloom filter. Note that we don't re-compress it which still saves
		// most of the compute cost (compression algorithms are usually designed
		// to make decompressing much cheaper than compressing since it happens
		// more often).
		if err := bufferedPage.WriteTo(c.page.filter); err != nil {
			return 0, err
		}
	case c.columnFilter != nil:
		// When a column filter is configured but no page filter was allocated,
		// we need to buffer the page in order to have access to the number of
		// values and properly size the bloom filter when writing the row group.
		c.filter = append(c.filter, page.Buffer())
	}

	pageHeader := &format.PageHeader{
		UncompressedPageSize: int32(page.Size()),
		CompressedPageSize:   int32(page.PageSize()),
		CRC:                  int32(page.CRC()),
	}

	switch h := page.PageHeader().(type) {
	case DataPageHeaderV1:
		pageHeader.DataPageHeader = h.header
	case DataPageHeaderV2:
		pageHeader.DataPageHeaderV2 = h.header
	default:
		return 0, fmt.Errorf("writing compressed page type of unknown type: %s", h.PageType())
	}

	header := &c.buffers.header
	header.Reset()
	if err := c.header.encoder.Encode(pageHeader); err != nil {
		return 0, err
	}
	headerSize := int32(header.Len())
	compressedSize := int64(headerSize + pageHeader.CompressedPageSize)

	if err := c.writePage(compressedSize, header, page.PageData()); err != nil {
		return 0, err
	}

	c.recordPageStats(headerSize, pageHeader, page)
	return page.NumValues(), nil
}

func (c *writerColumn) writePage(size int64, header, data io.Reader) error {
	buffer := c.pool.GetPageBuffer()
	defer func() {
		if buffer != nil {
			c.pool.PutPageBuffer(buffer)
		}
	}()
	headerSize, err := io.Copy(buffer, header)
	if err != nil {
		return err
	}
	dataSize, err := io.Copy(buffer, data)
	if err != nil {
		return err
	}
	written := headerSize + dataSize
	if size != written {
		return fmt.Errorf("writing parquet colum page expected %dB but got %dB: %w", size, written, io.ErrShortWrite)
	}
	c.pages = append(c.pages, buffer)
	buffer = nil
	return nil
}

func (c *writerColumn) writeDictionaryPage(output io.Writer, dict Dictionary) error {
	buffer := &c.buffers.page
	buffer.Reset()
	c.dict.encoder.Reset(buffer)

	if err := dict.Page().WriteTo(&c.dict.encoder); err != nil {
		return fmt.Errorf("writing parquet dictionary page: %w", err)
	}

	uncompressedPageSize := buffer.Len()
	pageData, err := c.compress(buffer.Bytes())
	if err != nil {
		return fmt.Errorf("compressing parquet dictionary page: %w", err)
	}

	pageHeader := &format.PageHeader{
		Type:                 format.DictionaryPage,
		UncompressedPageSize: int32(uncompressedPageSize),
		CompressedPageSize:   int32(len(pageData)),
		CRC:                  int32(crc32.ChecksumIEEE(pageData)),
		DictionaryPageHeader: &format.DictionaryPageHeader{
			NumValues: int32(dict.Len()),
			Encoding:  format.Plain,
			IsSorted:  false,
		},
	}

	header := &c.buffers.header
	header.Reset()
	if err := c.header.encoder.Encode(pageHeader); err != nil {
		return err
	}
	if _, err := output.Write(header.Bytes()); err != nil {
		return err
	}
	if _, err := output.Write(pageData); err != nil {
		return err
	}
	c.recordPageStats(int32(header.Len()), pageHeader, nil)
	return nil
}

func (c *writerColumn) compress(pageData []byte) ([]byte, error) {
	if c.compression.CompressionCodec() != format.Uncompressed {
		b, err := c.compression.Encode(c.buffers.compressed[:0], pageData)
		c.buffers.compressed = b
		if err != nil {
			return nil, err
		}
		pageData = b
	}
	return pageData, nil
}

func (c *writerColumn) makePageStatistics(page Page) format.Statistics {
	numNulls := page.NumNulls()
	minValue, maxValue, _ := page.Bounds()
	minValueBytes := minValue.Bytes()
	maxValueBytes := maxValue.Bytes()
	return format.Statistics{
		Min:       minValueBytes, // deprecated
		Max:       maxValueBytes, // deprecated
		NullCount: numNulls,
		MinValue:  minValueBytes,
		MaxValue:  maxValueBytes,
	}
}

func (c *writerColumn) recordPageStats(headerSize int32, header *format.PageHeader, page Page) {
	uncompressedSize := headerSize + header.UncompressedPageSize
	compressedSize := headerSize + header.CompressedPageSize

	if page != nil {
		numNulls := page.NumNulls()
		numValues := page.NumValues()
		minValue, maxValue, _ := page.Bounds()
		c.columnIndex.IndexPage(numValues, numNulls, minValue, maxValue)
		c.columnChunk.MetaData.NumValues += numValues

		c.offsetIndex.PageLocations = append(c.offsetIndex.PageLocations, format.PageLocation{
			Offset:             c.columnChunk.MetaData.TotalCompressedSize,
			CompressedPageSize: compressedSize,
			FirstRowIndex:      c.numRows,
		})

		c.numRows += page.NumRows()
	}

	pageType := header.Type
	encoding := format.Encoding(-1)
	switch pageType {
	case format.DataPageV2:
		encoding = header.DataPageHeaderV2.Encoding
	case format.DataPage:
		encoding = header.DataPageHeader.Encoding
	case format.DictionaryPage:
		encoding = header.DictionaryPageHeader.Encoding
	}

	c.columnChunk.MetaData.TotalUncompressedSize += int64(uncompressedSize)
	c.columnChunk.MetaData.TotalCompressedSize += int64(compressedSize)
	c.columnChunk.MetaData.EncodingStats = addPageEncodingStats(c.columnChunk.MetaData.EncodingStats, format.PageEncodingStats{
		PageType: pageType,
		Encoding: encoding,
		Count:    1,
	})
}

func addEncoding(encodings []format.Encoding, add format.Encoding) []format.Encoding {
	for _, enc := range encodings {
		if enc == add {
			return encodings
		}
	}
	return append(encodings, add)
}

func addPageEncodingStats(stats []format.PageEncodingStats, pages ...format.PageEncodingStats) []format.PageEncodingStats {
addPages:
	for _, add := range pages {
		for i, st := range stats {
			if st.PageType == add.PageType && st.Encoding == add.Encoding {
				stats[i].Count += add.Count
				continue addPages
			}
		}
		stats = append(stats, add)
	}
	return stats
}

func sortPageEncodings(encodings []format.Encoding) {
	sort.Slice(encodings, func(i, j int) bool {
		return encodings[i] < encodings[j]
	})
}

func sortPageEncodingStats(stats []format.PageEncodingStats) {
	sort.Slice(stats, func(i, j int) bool {
		s1 := &stats[i]
		s2 := &stats[j]
		if s1.PageType != s2.PageType {
			return s1.PageType < s2.PageType
		}
		return s1.Encoding < s2.Encoding
	})
}

type offsetTrackingWriter struct {
	writer io.Writer
	offset int64
}

func (w *offsetTrackingWriter) Reset(writer io.Writer) {
	w.writer = writer
	w.offset = 0
}

func (w *offsetTrackingWriter) Write(b []byte) (int, error) {
	n, err := w.writer.Write(b)
	w.offset += int64(n)
	return n, err
}

func (w *offsetTrackingWriter) WriteString(s string) (int, error) {
	n, err := io.WriteString(w.writer, s)
	w.offset += int64(n)
	return n, err
}

var (
	_ RowWriterWithSchema = (*Writer)(nil)
	_ RowReaderFrom       = (*Writer)(nil)
	_ RowGroupWriter      = (*Writer)(nil)

	_ RowWriter   = (*writer)(nil)
	_ PageWriter  = (*writer)(nil)
	_ ValueWriter = (*writer)(nil)

	_ PageWriter  = (*writerColumn)(nil)
	_ ValueWriter = (*writerColumn)(nil)
)
