package parquet

import (
	"encoding/binary"
	"fmt"
	"io"
	"reflect"

	"github.com/segmentio/parquet-go/compress"
	"github.com/segmentio/parquet-go/deprecated"
	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/format"
)

// Column represents a column in a parquet file.
//
// Methods of Column values are safe to call concurrently from multiple
// goroutines.
//
// Column instances satisfy the Node interface.
type Column struct {
	typ         Type
	file        *File
	schema      *format.SchemaElement
	order       *format.ColumnOrder
	path        columnPath
	columns     []*Column
	chunks      []*format.ColumnChunk
	columnIndex []*format.ColumnIndex
	offsetIndex []*format.OffsetIndex
	encoding    encoding.Encoding
	compression compress.Codec

	depth              int8
	maxRepetitionLevel byte
	maxDefinitionLevel byte
	index              int16
}

// Type returns the type of the column.
//
// The returned value is unspecified if c is not a leaf column.
func (c *Column) Type() Type { return c.typ }

// Optional returns true if the column is optional.
func (c *Column) Optional() bool { return schemaRepetitionTypeOf(c.schema) == format.Optional }

// Repeated returns true if the column may repeat.
func (c *Column) Repeated() bool { return schemaRepetitionTypeOf(c.schema) == format.Repeated }

// Required returns true if the column is required.
func (c *Column) Required() bool { return schemaRepetitionTypeOf(c.schema) == format.Required }

// Leaf returns true if c is a leaf column.
func (c *Column) Leaf() bool { return c.index >= 0 }

// Fields returns the list of fields on the column.
func (c *Column) Fields() []Field {
	fields := make([]Field, len(c.columns))
	for i, column := range c.columns {
		fields[i] = column
	}
	return fields
}

// Encoding returns the encodings used by this column.
func (c *Column) Encoding() encoding.Encoding { return c.encoding }

// Compression returns the compression codecs used by this column.
func (c *Column) Compression() compress.Codec { return c.compression }

// Path of the column in the parquet schema.
func (c *Column) Path() []string { return c.path }

// Name returns the column name.
func (c *Column) Name() string { return c.schema.Name }

// Columns returns the list of child columns.
//
// The method returns the same slice across multiple calls, the program must
// treat it as a read-only value.
func (c *Column) Columns() []*Column { return c.columns }

// Column returns the child column matching the given name.
func (c *Column) Column(name string) *Column {
	for _, child := range c.columns {
		if child.Name() == name {
			return child
		}
	}
	return nil
}

// Pages returns a reader exposing all pages in this column, across row groups.
func (c *Column) Pages() Pages {
	if c.index < 0 {
		return emptyPages{}
	}
	r := &columnPages{
		pages: make([]filePages, len(c.file.rowGroups)),
	}
	for i := range r.pages {
		r.pages[i].init(c.file.rowGroups[i].(*fileRowGroup).columns[c.index].(*fileColumnChunk))
	}
	return r
}

type columnPages struct {
	pages []filePages
	index int
}

func (c *columnPages) ReadPage() (Page, error) {
	for {
		if c.index >= len(c.pages) {
			return nil, io.EOF
		}
		p, err := c.pages[c.index].ReadPage()
		if err == nil || err != io.EOF {
			return p, err
		}
		c.index++
	}
}

func (c *columnPages) SeekToRow(rowIndex int64) error {
	c.index = 0

	for c.index < len(c.pages) && c.pages[c.index].chunk.rowGroup.NumRows >= rowIndex {
		rowIndex -= c.pages[c.index].chunk.rowGroup.NumRows
		c.index++
	}

	if c.index < len(c.pages) {
		if err := c.pages[c.index].SeekToRow(rowIndex); err != nil {
			return err
		}
		for i := range c.pages[c.index:] {
			p := &c.pages[c.index+i]
			if err := p.SeekToRow(0); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *columnPages) Close() error {
	var lastErr error

	for i := range c.pages {
		if err := c.pages[i].Close(); err != nil {
			lastErr = err
		}
	}

	c.pages = nil
	c.index = 0
	return lastErr
}

// Depth returns the position of the column relative to the root.
func (c *Column) Depth() int { return int(c.depth) }

// MaxRepetitionLevel returns the maximum value of repetition levels on this
// column.
func (c *Column) MaxRepetitionLevel() int { return int(c.maxRepetitionLevel) }

// MaxDefinitionLevel returns the maximum value of definition levels on this
// column.
func (c *Column) MaxDefinitionLevel() int { return int(c.maxDefinitionLevel) }

// Index returns the position of the column in a row. Only leaf columns have a
// column index, the method returns -1 when called on non-leaf columns.
func (c *Column) Index() int { return int(c.index) }

// GoType returns the Go type that best represents the parquet column.
func (c *Column) GoType() reflect.Type { return goTypeOf(c) }

// Value returns the sub-value in base for the child column at the given
// index.
func (c *Column) Value(base reflect.Value) reflect.Value {
	return base.MapIndex(reflect.ValueOf(&c.schema.Name).Elem())
}

// String returns a human-readable string representation of the column.
func (c *Column) String() string { return c.path.String() + ": " + sprint(c.Name(), c) }

func (c *Column) forEachLeaf(do func(*Column)) {
	if len(c.columns) == 0 {
		do(c)
	} else {
		for _, child := range c.columns {
			child.forEachLeaf(do)
		}
	}
}

func openColumns(file *File) (*Column, error) {
	cl := columnLoader{}

	c, err := cl.open(file, nil)
	if err != nil {
		return nil, err
	}

	// Validate that there aren't extra entries in the row group columns,
	// which would otherwise indicate that there are dangling data pages
	// in the file.
	for index, rowGroup := range file.metadata.RowGroups {
		if cl.rowGroupColumnIndex != len(rowGroup.Columns) {
			return nil, fmt.Errorf("row group at index %d contains %d columns but %d were referenced by the column schemas",
				index, len(rowGroup.Columns), cl.rowGroupColumnIndex)
		}
	}

	_, err = c.setLevels(0, 0, 0, 0)
	return c, err
}

func (c *Column) setLevels(depth, repetition, definition, index int) (int, error) {
	if depth > MaxColumnDepth {
		return -1, fmt.Errorf("cannot represent parquet columns with more than %d nested levels: %s", MaxColumnDepth, c.path)
	}
	if index > MaxColumnIndex {
		return -1, fmt.Errorf("cannot represent parquet rows with more than %d columns: %s", MaxColumnIndex, c.path)
	}
	if repetition > MaxRepetitionLevel {
		return -1, fmt.Errorf("cannot represent parquet columns with more than %d repetition levels: %s", MaxRepetitionLevel, c.path)
	}
	if definition > MaxDefinitionLevel {
		return -1, fmt.Errorf("cannot represent parquet columns with more than %d definition levels: %s", MaxDefinitionLevel, c.path)
	}

	switch schemaRepetitionTypeOf(c.schema) {
	case format.Optional:
		definition++
	case format.Repeated:
		repetition++
		definition++
	}

	c.depth = int8(depth)
	c.maxRepetitionLevel = byte(repetition)
	c.maxDefinitionLevel = byte(definition)
	depth++

	if len(c.columns) > 0 {
		c.index = -1
	} else {
		c.index = int16(index)
		index++
	}

	var err error
	for _, child := range c.columns {
		if index, err = child.setLevels(depth, repetition, definition, index); err != nil {
			return -1, err
		}
	}
	return index, nil
}

type columnLoader struct {
	schemaIndex         int
	columnOrderIndex    int
	rowGroupColumnIndex int
}

func (cl *columnLoader) open(file *File, path []string) (*Column, error) {
	c := &Column{
		file:   file,
		schema: &file.metadata.Schema[cl.schemaIndex],
	}
	c.path = c.path.append(c.schema.Name)

	cl.schemaIndex++
	numChildren := int(c.schema.NumChildren)

	if numChildren == 0 {
		c.typ = schemaElementTypeOf(c.schema)

		if cl.columnOrderIndex < len(file.metadata.ColumnOrders) {
			c.order = &file.metadata.ColumnOrders[cl.columnOrderIndex]
			cl.columnOrderIndex++
		}

		rowGroups := file.metadata.RowGroups
		rowGroupColumnIndex := cl.rowGroupColumnIndex
		cl.rowGroupColumnIndex++

		c.chunks = make([]*format.ColumnChunk, 0, len(rowGroups))
		c.columnIndex = make([]*format.ColumnIndex, 0, len(rowGroups))
		c.offsetIndex = make([]*format.OffsetIndex, 0, len(rowGroups))

		for i, rowGroup := range rowGroups {
			if rowGroupColumnIndex >= len(rowGroup.Columns) {
				return nil, fmt.Errorf("row group at index %d does not have enough columns", i)
			}
			c.chunks = append(c.chunks, &rowGroup.Columns[rowGroupColumnIndex])
		}

		if len(file.columnIndexes) > 0 {
			for i := range rowGroups {
				if rowGroupColumnIndex >= len(file.columnIndexes) {
					return nil, fmt.Errorf("row group at index %d does not have enough column index pages", i)
				}
				c.columnIndex = append(c.columnIndex, &file.columnIndexes[rowGroupColumnIndex])
			}
		}

		if len(file.offsetIndexes) > 0 {
			for i := range rowGroups {
				if rowGroupColumnIndex >= len(file.offsetIndexes) {
					return nil, fmt.Errorf("row group at index %d does not have enough offset index pages", i)
				}
				c.offsetIndex = append(c.offsetIndex, &file.offsetIndexes[rowGroupColumnIndex])
			}
		}

		if len(c.chunks) > 0 {
			// Pick the encoding and compression codec of the first chunk.
			//
			// Technically each column chunk may use a different compression
			// codec, and each page of the column chunk might have a different
			// encoding. Exposing these details does not provide a lot of value
			// to the end user.
			//
			// Programs that wish to determine the encoding and compression of
			// each page of the column should iterate through the pages and read
			// the page headers to determine which compression and encodings are
			// applied.
			for _, encoding := range c.chunks[0].MetaData.Encoding {
				c.encoding = LookupEncoding(encoding)
				break
			}
			c.compression = LookupCompressionCodec(c.chunks[0].MetaData.Codec)
		}

		return c, nil
	}

	c.typ = &groupType{}
	c.columns = make([]*Column, numChildren)

	for i := range c.columns {
		if cl.schemaIndex >= len(file.metadata.Schema) {
			return nil, fmt.Errorf("column %q has more children than there are schemas in the file: %d > %d",
				c.schema.Name, cl.schemaIndex+1, len(file.metadata.Schema))
		}

		var err error
		c.columns[i], err = cl.open(file, path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", c.schema.Name, err)
		}
	}

	return c, nil
}

func schemaElementTypeOf(s *format.SchemaElement) Type {
	if lt := s.LogicalType; lt != nil {
		// A logical type exists, the Type interface implementations in this
		// package are all based on the logical parquet types declared in the
		// format sub-package so we can return them directly via a pointer type
		// conversion.
		switch {
		case lt.UTF8 != nil:
			return (*stringType)(lt.UTF8)
		case lt.Map != nil:
			return (*mapType)(lt.Map)
		case lt.List != nil:
			return (*listType)(lt.List)
		case lt.Enum != nil:
			return (*enumType)(lt.Enum)
		case lt.Decimal != nil:
			// TODO:
			// return (*decimalType)(lt.Decimal)
		case lt.Date != nil:
			return (*dateType)(lt.Date)
		case lt.Time != nil:
			return (*timeType)(lt.Time)
		case lt.Timestamp != nil:
			return (*timestampType)(lt.Timestamp)
		case lt.Integer != nil:
			return (*intType)(lt.Integer)
		case lt.Unknown != nil:
			return (*nullType)(lt.Unknown)
		case lt.Json != nil:
			return (*jsonType)(lt.Json)
		case lt.Bson != nil:
			return (*bsonType)(lt.Bson)
		case lt.UUID != nil:
			return (*uuidType)(lt.UUID)
		}
	}

	if ct := s.ConvertedType; ct != nil {
		// This column contains no logical type but has a converted type, it
		// was likely created by an older parquet writer. Convert the legacy
		// type representation to the equivalent logical parquet type.
		switch *ct {
		case deprecated.UTF8:
			return &stringType{}
		case deprecated.Map:
			return &mapType{}
		case deprecated.MapKeyValue:
			return &groupType{}
		case deprecated.List:
			return &listType{}
		case deprecated.Enum:
			return &enumType{}
		case deprecated.Decimal:
			// TODO
		case deprecated.Date:
			return &dateType{}
		case deprecated.TimeMillis:
			return &timeType{IsAdjustedToUTC: true, Unit: Millisecond.TimeUnit()}
		case deprecated.TimeMicros:
			return &timeType{IsAdjustedToUTC: true, Unit: Microsecond.TimeUnit()}
		case deprecated.TimestampMillis:
			return &timestampType{IsAdjustedToUTC: true, Unit: Millisecond.TimeUnit()}
		case deprecated.TimestampMicros:
			return &timestampType{IsAdjustedToUTC: true, Unit: Microsecond.TimeUnit()}
		case deprecated.Uint8:
			return &unsignedIntTypes[0]
		case deprecated.Uint16:
			return &unsignedIntTypes[1]
		case deprecated.Uint32:
			return &unsignedIntTypes[2]
		case deprecated.Uint64:
			return &unsignedIntTypes[3]
		case deprecated.Int8:
			return &signedIntTypes[0]
		case deprecated.Int16:
			return &signedIntTypes[1]
		case deprecated.Int32:
			return &signedIntTypes[2]
		case deprecated.Int64:
			return &signedIntTypes[3]
		case deprecated.Json:
			return &jsonType{}
		case deprecated.Bson:
			return &bsonType{}
		case deprecated.Interval:
			// TODO
		}
	}

	if t := s.Type; t != nil {
		// The column only has a physical type, convert it to one of the
		// primitive types supported by this package.
		switch kind := Kind(*t); kind {
		case Boolean:
			return BooleanType
		case Int32:
			return Int32Type
		case Int64:
			return Int64Type
		case Int96:
			return Int96Type
		case Float:
			return FloatType
		case Double:
			return DoubleType
		case ByteArray:
			return ByteArrayType
		case FixedLenByteArray:
			if s.TypeLength != nil {
				return FixedLenByteArrayType(int(*s.TypeLength))
			}
		}
	}

	// If we reach this point, we are likely reading a parquet column that was
	// written with a non-standard type or is in a newer version of the format
	// than this package supports.
	return &nullType{}
}

func schemaRepetitionTypeOf(s *format.SchemaElement) format.FieldRepetitionType {
	if s.RepetitionType != nil {
		return *s.RepetitionType
	}
	return format.Required
}

type dictPage struct {
	values  []byte
	offsets []uint32
}

func (p *dictPage) reset() {
	p.values = p.values[:0]
	p.offsets = p.offsets[:0]
}

type dataPage struct {
	repetitionLevels []byte
	definitionLevels []byte
	data             []byte
	values           []byte
	offsets          []uint32
	dictionary       Dictionary
}

func (p *dataPage) reset() {
	p.repetitionLevels = p.repetitionLevels[:0]
	p.definitionLevels = p.definitionLevels[:0]
	p.data = p.data[:0]
	p.values = p.values[:0]
	p.offsets = p.offsets[:0]
	p.dictionary = nil
}

func (p *dataPage) decompress(codec compress.Codec, data []byte) (err error) {
	p.values, err = codec.Decode(p.values, data)
	p.data, p.values = p.values, p.data[:0]
	return err
}

// DecodeDataPageV1 decodes a data page from the header, compressed data, and
// optional dictionary passed as arguments.
func (c *Column) DecodeDataPageV1(header DataPageHeaderV1, data []byte, dict Dictionary) (Page, error) {
	return c.decodeDataPageV1(header, &dataPage{data: data, dictionary: dict})
}

func (c *Column) decodeDataPageV1(header DataPageHeaderV1, page *dataPage) (Page, error) {
	var err error

	if isCompressed(c.compression) {
		if err := page.decompress(c.compression, page.data); err != nil {
			return nil, fmt.Errorf("decompressing data page v1: %w", err)
		}
	}

	numValues := header.NumValues()
	data := page.data
	page.repetitionLevels = page.repetitionLevels[:0]
	page.definitionLevels = page.definitionLevels[:0]

	if c.maxRepetitionLevel > 0 {
		encoding := lookupLevelEncoding(header.RepetitionLevelEncoding(), c.maxRepetitionLevel)
		page.repetitionLevels, data, err = decodeLevelsV1(encoding, numValues, page.repetitionLevels, data)
		if err != nil {
			return nil, fmt.Errorf("decoding repetition levels of data page v1: %w", err)
		}
	}

	if c.maxDefinitionLevel > 0 {
		encoding := lookupLevelEncoding(header.DefinitionLevelEncoding(), c.maxDefinitionLevel)
		page.definitionLevels, data, err = decodeLevelsV1(encoding, numValues, page.definitionLevels, data)
		if err != nil {
			return nil, fmt.Errorf("decoding definition levels of data page v1: %w", err)
		}

		// Data pages v1 did not embed the number of null values,
		// so we have to compute it from the definition levels.
		numValues -= int64(countLevelsNotEqual(page.definitionLevels, c.maxDefinitionLevel))
	}

	return c.decodeDataPage(header, numValues, page, data)
}

// DecodeDataPageV2 decodes a data page from the header, compressed data, and
// optional dictionary passed as arguments.
func (c *Column) DecodeDataPageV2(header DataPageHeaderV2, data []byte, dict Dictionary) (Page, error) {
	return c.decodeDataPageV2(header, &dataPage{data: data, dictionary: dict})
}

func (c *Column) decodeDataPageV2(header DataPageHeaderV2, page *dataPage) (Page, error) {
	var numValues = header.NumValues()
	var err error
	var data = page.data
	page.repetitionLevels = page.repetitionLevels[:0]
	page.definitionLevels = page.definitionLevels[:0]

	if c.maxRepetitionLevel > 0 {
		encoding := lookupLevelEncoding(header.RepetitionLevelEncoding(), c.maxRepetitionLevel)
		length := header.RepetitionLevelsByteLength()
		page.repetitionLevels, data, err = decodeLevelsV2(encoding, numValues, page.repetitionLevels, data, length)
		if err != nil {
			return nil, fmt.Errorf("decoding repetition levels of data page v2: %w", io.ErrUnexpectedEOF)
		}
	}

	if c.maxDefinitionLevel > 0 {
		encoding := lookupLevelEncoding(header.DefinitionLevelEncoding(), c.maxDefinitionLevel)
		length := header.DefinitionLevelsByteLength()
		page.definitionLevels, data, err = decodeLevelsV2(encoding, numValues, page.definitionLevels, data, length)
		if err != nil {
			return nil, fmt.Errorf("decoding definition levels of data page v2: %w", io.ErrUnexpectedEOF)
		}
	}

	if isCompressed(c.compression) && header.IsCompressed() {
		if err := page.decompress(c.compression, data); err != nil {
			return nil, fmt.Errorf("decompressing data page v2: %w", err)
		}
		data = page.data
	}

	numValues -= header.NumNulls()
	return c.decodeDataPage(header, numValues, page, data)
}

func (c *Column) decodeDataPage(header DataPageHeader, numValues int64, page *dataPage, data []byte) (Page, error) {
	pageEncoding := LookupEncoding(header.Encoding())
	pageType := c.Type()

	if isDictionaryEncoding(pageEncoding) {
		// In some legacy configurations, the PLAIN_DICTIONARY encoding is used
		// on data page headers to indicate that the page contains indexes into
		// the dictionary page, but the page is still encoded using the RLE
		// encoding in this case, so we convert it to RLE_DICTIONARY.
		pageEncoding = &RLEDictionary
		pageType = indexedPageType{newIndexedType(pageType, page.dictionary)}
	}

	values := pageType.NewValues(page.values, page.offsets)
	values, err := pageType.Decode(values, data, pageEncoding)
	if err != nil {
		return nil, err
	}
	page.values, page.offsets = values.Data()

	newPage := pageType.NewPage(c.Index(), int(numValues), values)
	switch {
	case c.maxRepetitionLevel > 0:
		newPage = newRepeatedPage(newPage.Buffer(), c.maxRepetitionLevel, c.maxDefinitionLevel, page.repetitionLevels, page.definitionLevels)
	case c.maxDefinitionLevel > 0:
		newPage = newOptionalPage(newPage.Buffer(), c.maxDefinitionLevel, page.definitionLevels)
	}
	return newPage, nil
}

func decodeLevelsV1(enc encoding.Encoding, numValues int64, levels, data []byte) ([]byte, []byte, error) {
	if len(data) < 4 {
		return nil, data, io.ErrUnexpectedEOF
	}
	i := 4
	j := 4 + int(binary.LittleEndian.Uint32(data))
	if j > len(data) {
		return nil, data, io.ErrUnexpectedEOF
	}
	levels, err := decodeLevels(enc, numValues, levels, data[i:j])
	return levels, data[j:], err
}

func decodeLevelsV2(enc encoding.Encoding, numValues int64, levels, data []byte, length int64) ([]byte, []byte, error) {
	if length > int64(len(data)) {
		return nil, data, io.ErrUnexpectedEOF
	}
	levels, err := decodeLevels(enc, numValues, levels, data[:length])
	return levels, data[length:], err
}

func decodeLevels(enc encoding.Encoding, numValues int64, levels, data []byte) ([]byte, error) {
	if cap(levels) < int(numValues) {
		levels = make([]byte, numValues)
	}
	levels, err := enc.DecodeLevels(levels, data)
	if err == nil {
		switch {
		case len(levels) < int(numValues):
			err = fmt.Errorf("decoding level expected %d values but got only %d", numValues, len(levels))
		case len(levels) > int(numValues):
			levels = levels[:numValues]
		}
	}
	return levels, err
}

// DecodeDictionary decodes a data page from the header and compressed data
// passed as arguments.
func (c *Column) DecodeDictionary(header DictionaryPageHeader, data []byte) (Dictionary, error) {
	return c.decodeDictionary(header, &dataPage{data: data}, &dictPage{})
}

func (c *Column) decodeDictionary(header DictionaryPageHeader, page *dataPage, dict *dictPage) (Dictionary, error) {
	if isCompressed(c.compression) {
		if err := page.decompress(c.compression, page.data); err != nil {
			return nil, fmt.Errorf("decompressing dictionary page: %w", err)
		}
	}

	pageType := c.Type()
	pageEncoding := header.Encoding()
	if pageEncoding == format.PlainDictionary {
		pageEncoding = format.Plain
	}

	values := pageType.NewValues(dict.values, dict.offsets)
	values, err := pageType.Decode(values, page.data, LookupEncoding(pageEncoding))
	if err != nil {
		return nil, err
	}
	dict.values, dict.offsets = values.Data()
	values = pageType.NewValues(dict.values, dict.offsets)
	return pageType.NewDictionary(int(c.index), int(header.NumValues()), values), nil
}

var (
	_ Node = (*Column)(nil)
)
