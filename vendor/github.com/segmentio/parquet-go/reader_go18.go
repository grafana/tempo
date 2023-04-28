//go:build go1.18

package parquet

import (
	"io"
	"reflect"
)

// GenericReader is similar to a Reader but uses a type parameter to define the
// Go type representing the schema of rows being read.
//
// See GenericWriter for details about the benefits over the classic Reader API.
type GenericReader[T any] struct {
	base Reader
	read readFunc[T]
}

// NewGenericReader is like NewReader but returns GenericReader[T] suited to write
// rows of Go type T.
//
// The type parameter T should be a map, struct, or any. Any other types will
// cause a panic at runtime. Type checking is a lot more effective when the
// generic parameter is a struct type, using map and interface types is somewhat
// similar to using a Writer.
//
// If the option list may explicitly declare a schema, it must be compatible
// with the schema generated from T.
func NewGenericReader[T any](input io.ReaderAt, options ...ReaderOption) *GenericReader[T] {
	c, err := NewReaderConfig(options...)
	if err != nil {
		panic(err)
	}

	f, err := openFile(input)
	if err != nil {
		panic(err)
	}

	rowGroup := fileRowGroupOf(f)

	t := typeOf[T]()
	if c.Schema == nil {
		if t == nil {
			c.Schema = rowGroup.Schema()
		} else {
			c.Schema = schemaOf(dereference(t))
		}
	}

	r := &GenericReader[T]{
		base: Reader{
			file: reader{
				schema:   c.Schema,
				rowGroup: rowGroup,
			},
		},
	}

	if !nodesAreEqual(c.Schema, f.schema) {
		r.base.file.rowGroup = convertRowGroupTo(r.base.file.rowGroup, c.Schema)
	}

	r.base.read.init(r.base.file.schema, r.base.file.rowGroup)
	r.read = readFuncOf[T](t, r.base.file.schema)
	return r
}

func NewGenericRowGroupReader[T any](rowGroup RowGroup, options ...ReaderOption) *GenericReader[T] {
	c, err := NewReaderConfig(options...)
	if err != nil {
		panic(err)
	}

	t := typeOf[T]()
	if c.Schema == nil {
		if t == nil {
			c.Schema = rowGroup.Schema()
		} else {
			c.Schema = schemaOf(dereference(t))
		}
	}

	r := &GenericReader[T]{
		base: Reader{
			file: reader{
				schema:   c.Schema,
				rowGroup: rowGroup,
			},
		},
	}

	if !nodesAreEqual(c.Schema, rowGroup.Schema()) {
		r.base.file.rowGroup = convertRowGroupTo(r.base.file.rowGroup, c.Schema)
	}

	r.base.read.init(r.base.file.schema, r.base.file.rowGroup)
	r.read = readFuncOf[T](t, r.base.file.schema)
	return r
}

func (r *GenericReader[T]) Reset() {
	r.base.Reset()
}

// Read reads the next rows from the reader into the given rows slice up to len(rows).
//
// The returned values are safe to reuse across Read calls and do not share
// memory with the reader's underlying page buffers.
//
// The method returns the number of rows read and io.EOF when no more rows
// can be read from the reader.
func (r *GenericReader[T]) Read(rows []T) (int, error) {
	return r.read(r, rows)
}

func (r *GenericReader[T]) ReadRows(rows []Row) (int, error) {
	return r.base.ReadRows(rows)
}

func (r *GenericReader[T]) Schema() *Schema {
	return r.base.Schema()
}

func (r *GenericReader[T]) NumRows() int64 {
	return r.base.NumRows()
}

func (r *GenericReader[T]) SeekToRow(rowIndex int64) error {
	return r.base.SeekToRow(rowIndex)
}

func (r *GenericReader[T]) Close() error {
	return r.base.Close()
}

// readRows reads the next rows from the reader into the given rows slice up to len(rows).
//
// The returned values are safe to reuse across readRows calls and do not share
// memory with the reader's underlying page buffers.
//
// The method returns the number of rows read and io.EOF when no more rows
// can be read from the reader.
func (r *GenericReader[T]) readRows(rows []T) (int, error) {
	nRequest := len(rows)
	if cap(r.base.rowbuf) < nRequest {
		r.base.rowbuf = make([]Row, nRequest)
	} else {
		r.base.rowbuf = r.base.rowbuf[:nRequest]
	}

	var n, nTotal int
	var err error
	for {
		// ReadRows reads the minimum remaining rows in a column page across all columns
		// of the underlying reader, unless the length of the slice passed to it is smaller.
		// In that case, ReadRows will read the number of rows equal to the length of the
		// given slice argument. We limit that length to never be more than requested
		// because sequential reads can cross page boundaries.
		n, err = r.base.ReadRows(r.base.rowbuf[:nRequest-nTotal])
		if n > 0 {
			schema := r.base.Schema()

			for i, row := range r.base.rowbuf[:n] {
				if err2 := schema.Reconstruct(&rows[nTotal+i], row); err2 != nil {
					return nTotal + i, err2
				}
			}
		}
		nTotal += n
		if n == 0 || nTotal == nRequest || err != nil {
			break
		}
	}

	return nTotal, err
}

var (
	_ Rows                = (*GenericReader[any])(nil)
	_ RowReaderWithSchema = (*Reader)(nil)

	_ Rows                = (*GenericReader[struct{}])(nil)
	_ RowReaderWithSchema = (*GenericReader[struct{}])(nil)

	_ Rows                = (*GenericReader[map[struct{}]struct{}])(nil)
	_ RowReaderWithSchema = (*GenericReader[map[struct{}]struct{}])(nil)
)

type readFunc[T any] func(*GenericReader[T], []T) (int, error)

func readFuncOf[T any](t reflect.Type, schema *Schema) readFunc[T] {
	if t == nil {
		return (*GenericReader[T]).readRows
	}
	switch t.Kind() {
	case reflect.Interface, reflect.Map:
		return (*GenericReader[T]).readRows

	case reflect.Struct:
		return (*GenericReader[T]).readRows

	case reflect.Pointer:
		if e := t.Elem(); e.Kind() == reflect.Struct {
			return (*GenericReader[T]).readRows
		}
	}
	panic("cannot create reader for values of type " + t.String())
}
