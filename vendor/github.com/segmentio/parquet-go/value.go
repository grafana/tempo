package parquet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"reflect"
	"strconv"
	"unsafe"

	"github.com/google/uuid"
	"github.com/segmentio/parquet-go/deprecated"
)

const (
	// 170 x sizeof(Value) = 4KB
	defaultValueBufferSize = 170
)

// The Value type is similar to the reflect.Value abstraction of Go values, but
// for parquet values. Value instances wrap underlying Go values mapped to one
// of the parquet physical types.
//
// Value instances are small, immutable objects, and usually passed by value
// between function calls.
//
// The zero-value of Value represents the null parquet value.
type Value struct {
	// data
	ptr *byte
	u64 uint64
	// type
	kind int8 // XOR(Kind) so the zero-value is <null>
	// levels
	definitionLevel int8
	repetitionLevel int8
	columnIndex     int16 // XOR so the zero-value is -1
}

// ValueReader is an interface implemented by types that support reading
// batches of values.
type ValueReader interface {
	// Read values into the buffer passed as argument and return the number of
	// values read. When all values have been read, the error will be io.EOF.
	ReadValues([]Value) (int, error)
}

type ValueReaderAt interface {
	ReadValuesAt([]Value, int64) (int, error)
}

// ValueReaderFrom is an interface implemented by value writers to read values
// from a reader.
type ValueReaderFrom interface {
	ReadValuesFrom(ValueReader) (int64, error)
}

// ValueWriter is an interface implemented by types that support reading
// batches of values.
type ValueWriter interface {
	// Write values from the buffer passed as argument and returns the number
	// of values written.
	WriteValues([]Value) (int, error)
}

// ValueWriterTo is an interface implemented by value readers to write values to
// a writer.
type ValueWriterTo interface {
	WriteValuesTo(ValueWriter) (int64, error)
}

// CopyValues copies values from src to dst, returning the number of values
// that were written.
//
// As an optimization, the reader and writer may choose to implement
// ValueReaderFrom and ValueWriterTo to provide their own copy logic.
//
// The function returns any error it encounters reading or writing pages, except
// for io.EOF from the reader which indicates that there were no more values to
// read.
func CopyValues(dst ValueWriter, src ValueReader) (int64, error) {
	return copyValues(dst, src, nil)
}

func copyValues(dst ValueWriter, src ValueReader, buf []Value) (written int64, err error) {
	if wt, ok := src.(ValueWriterTo); ok {
		return wt.WriteValuesTo(dst)
	}

	if rf, ok := dst.(ValueReaderFrom); ok {
		return rf.ReadValuesFrom(src)
	}

	if len(buf) == 0 {
		buf = make([]Value, defaultValueBufferSize)
	}

	defer clearValues(buf)

	for {
		n, err := src.ReadValues(buf)

		if n > 0 {
			wn, werr := dst.WriteValues(buf[:n])
			written += int64(wn)
			if werr != nil {
				return written, werr
			}
		}

		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return written, err
		}

		if n == 0 {
			return written, io.ErrNoProgress
		}
	}
}

// ValueOf constructs a parquet value from a Go value v.
//
// The physical type of the value is assumed from the Go type of v using the
// following conversion table:
//
//	Go type | Parquet physical type
//	------- | ---------------------
//	nil     | NULL
//	bool    | BOOLEAN
//	int8    | INT32
//	int16   | INT32
//	int32   | INT32
//	int64   | INT64
//	int     | INT64
//	uint8   | INT32
//	uint16  | INT32
//	uint32  | INT32
//	uint64  | INT64
//	uintptr | INT64
//	float32 | FLOAT
//	float64 | DOUBLE
//	string  | BYTE_ARRAY
//	[]byte  | BYTE_ARRAY
//	[*]byte | FIXED_LEN_BYTE_ARRAY
//
// When converting a []byte or [*]byte value, the underlying byte array is not
// copied; instead, the returned parquet value holds a reference to it.
//
// The repetition and definition levels of the returned value are both zero.
//
// The function panics if the Go value cannot be represented in parquet.
func ValueOf(v interface{}) Value {
	switch value := v.(type) {
	case nil:
		return Value{}
	case uuid.UUID:
		return makeValueBytes(FixedLenByteArray, value[:])
	case deprecated.Int96:
		return makeValueInt96(value)
	}

	k := Kind(-1)
	t := reflect.TypeOf(v)

	switch t.Kind() {
	case reflect.Bool:
		k = Boolean
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		k = Int32
	case reflect.Int64, reflect.Int, reflect.Uint64, reflect.Uint, reflect.Uintptr:
		k = Int64
	case reflect.Float32:
		k = Float
	case reflect.Float64:
		k = Double
	case reflect.String:
		k = ByteArray
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			k = ByteArray
		}
	case reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			k = FixedLenByteArray
		}
	}

	if k < 0 {
		panic("cannot create parquet value from go value of type " + t.String())
	}

	return makeValue(k, reflect.ValueOf(v))
}

func makeValue(k Kind, v reflect.Value) Value {
	switch k {
	case Boolean:
		return makeValueBoolean(v.Bool())

	case Int32:
		switch v.Kind() {
		case reflect.Int8, reflect.Int16, reflect.Int32:
			return makeValueInt32(int32(v.Int()))
		case reflect.Uint8, reflect.Uint16, reflect.Uint32:
			return makeValueInt32(int32(v.Uint()))
		}

	case Int64:
		switch v.Kind() {
		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
			return makeValueInt64(v.Int())
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint, reflect.Uintptr:
			return makeValueInt64(int64(v.Uint()))
		}

	case Int96:
		switch v.Type() {
		case reflect.TypeOf(deprecated.Int96{}):
			return makeValueInt96(v.Interface().(deprecated.Int96))
		}

	case Float:
		switch v.Kind() {
		case reflect.Float32:
			return makeValueFloat(float32(v.Float()))
		}

	case Double:
		switch v.Kind() {
		case reflect.Float32, reflect.Float64:
			return makeValueDouble(v.Float())
		}

	case ByteArray:
		switch v.Kind() {
		case reflect.String:
			return makeValueString(k, v.String())
		case reflect.Slice:
			if v.Type().Elem().Kind() == reflect.Uint8 {
				return makeValueBytes(k, v.Bytes())
			}
		}

	case FixedLenByteArray:
		switch v.Kind() {
		case reflect.String: // uuid
			return makeValueString(k, v.String())
		case reflect.Array:
			if v.Type().Elem().Kind() == reflect.Uint8 {
				return makeValueFixedLenByteArray(v)
			}
		}
	}

	panic("cannot create parquet value of type " + k.String() + " from go value of type " + v.Type().String())
}

func makeValueBoolean(value bool) Value {
	v := Value{kind: ^int8(Boolean)}
	if value {
		v.u64 = 1
	}
	return v
}

func makeValueInt32(value int32) Value {
	return Value{
		kind: ^int8(Int32),
		u64:  uint64(value),
	}
}

func makeValueInt64(value int64) Value {
	return Value{
		kind: ^int8(Int64),
		u64:  uint64(value),
	}
}

func makeValueInt96(value deprecated.Int96) Value {
	// TODO: this is highly inefficient because we need a heap allocation to
	// store the value; we don't expect INT96 to be used frequently since it
	// is a deprecated feature of parquet, and it helps keep the Value type
	// compact for all the other more common cases.
	bits := [12]byte{}
	binary.LittleEndian.PutUint32(bits[0:4], value[0])
	binary.LittleEndian.PutUint32(bits[4:8], value[1])
	binary.LittleEndian.PutUint32(bits[8:12], value[2])
	return Value{
		kind: ^int8(Int96),
		ptr:  &bits[0],
		u64:  12, // set the length so we can use the ByteArray method
	}
}

func makeValueFloat(value float32) Value {
	return Value{
		kind: ^int8(Float),
		u64:  uint64(math.Float32bits(value)),
	}
}

func makeValueDouble(value float64) Value {
	return Value{
		kind: ^int8(Double),
		u64:  math.Float64bits(value),
	}
}

func makeValueBytes(kind Kind, value []byte) Value {
	return makeValueByteArray(kind, *(**byte)(unsafe.Pointer(&value)), len(value))
}

func makeValueString(kind Kind, value string) Value {
	return makeValueByteArray(kind, *(**byte)(unsafe.Pointer(&value)), len(value))
}

func makeValueFixedLenByteArray(v reflect.Value) Value {
	t := v.Type()
	// When the array is addressable, we take advantage of this
	// condition to avoid the heap allocation otherwise needed
	// to pack the reference into an interface{} value.
	if v.CanAddr() {
		v = v.Addr()
	} else {
		u := reflect.New(t)
		u.Elem().Set(v)
		v = u
	}
	// TODO: unclear if the conversion to unsafe.Pointer from
	// reflect.Value.Pointer is safe here.
	return makeValueByteArray(FixedLenByteArray, (*byte)(unsafe.Pointer(v.Pointer())), t.Len())
}

func makeValueByteArray(kind Kind, data *byte, size int) Value {
	return Value{
		kind: ^int8(kind),
		ptr:  data,
		u64:  uint64(size),
	}
}

// Kind returns the kind of v, which represents its parquet physical type.
func (v Value) Kind() Kind { return ^Kind(v.kind) }

// IsNull returns true if v is the null value.
func (v Value) IsNull() bool { return v.kind == 0 }

// Boolean returns v as a bool, assuming the underlying type is BOOLEAN.
func (v Value) Boolean() bool { return v.u64 != 0 }

// Int32 returns v as a int32, assuming the underlying type is INT32.
func (v Value) Int32() int32 { return int32(v.u64) }

// Int64 returns v as a int64, assuming the underlying type is INT64.
func (v Value) Int64() int64 { return int64(v.u64) }

// Int96 returns v as a int96, assuming the underlying type is INT96.
func (v Value) Int96() deprecated.Int96 { return makeInt96(v.ByteArray()) }

// Float returns v as a float32, assuming the underlying type is FLOAT.
func (v Value) Float() float32 { return math.Float32frombits(uint32(v.u64)) }

// Double returns v as a float64, assuming the underlying type is DOUBLE.
func (v Value) Double() float64 { return math.Float64frombits(v.u64) }

// ByteArray returns v as a []byte, assuming the underlying type is either
// BYTE_ARRAY or FIXED_LEN_BYTE_ARRAY.
//
// The application must treat the returned byte slice as a read-only value,
// mutating the content will result in undefined behaviors.
func (v Value) ByteArray() []byte { return unsafe.Slice(v.ptr, int(v.u64)) }

// RepetitionLevel returns the repetition level of v.
func (v Value) RepetitionLevel() int { return int(v.repetitionLevel) }

// DefinitionLevel returns the definition level of v.
func (v Value) DefinitionLevel() int { return int(v.definitionLevel) }

// Column returns the column index within the row that v was created from.
//
// Returns -1 if the value does not carry a column index.
func (v Value) Column() int { return int(^v.columnIndex) }

// Bytes returns the binary representation of v.
//
// If v is the null value, an nil byte slice is returned.
func (v Value) Bytes() []byte { return v.AppendBytes(nil) }

// AppendBytes appends the binary representation of v to b.
//
// If v is the null value, b is returned unchanged.
func (v Value) AppendBytes(b []byte) []byte {
	buf := [8]byte{}
	switch v.Kind() {
	case Boolean:
		binary.LittleEndian.PutUint32(buf[:4], uint32(v.u64))
		return append(b, buf[0])
	case Int32, Float:
		binary.LittleEndian.PutUint32(buf[:4], uint32(v.u64))
		return append(b, buf[:4]...)
	case Int64, Double:
		binary.LittleEndian.PutUint64(buf[:8], v.u64)
		return append(b, buf[:8]...)
	case ByteArray, FixedLenByteArray, Int96:
		return append(b, v.ByteArray()...)
	default:
		return b
	}
}

// Format outputs a human-readable representation of v to w, using r as the
// formatting verb to describe how the value should be printed.
//
// The following formatting options are supported:
//
//		%c	prints the column index
//		%+c	prints the column index, prefixed with "C:"
//		%d	prints the definition level
//		%+d	prints the definition level, prefixed with "D:"
//		%r	prints the repetition level
//		%+r	prints the repetition level, prefixed with "R:"
//		%q	prints the quoted representation of v
//		%+q	prints the quoted representation of v, prefixed with "V:"
//		%s	prints the string representation of v
//		%+s	prints the string representation of v, prefixed with "V:"
//		%v	same as %s
//		%+v	prints a verbose representation of v
//		%#v	prints a Go value representation of v
//
// Format satisfies the fmt.Formatter interface.
func (v Value) Format(w fmt.State, r rune) {
	switch r {
	case 'c':
		if w.Flag('+') {
			io.WriteString(w, "C:")
		}
		fmt.Fprint(w, v.Column())

	case 'd':
		if w.Flag('+') {
			io.WriteString(w, "D:")
		}
		fmt.Fprint(w, v.DefinitionLevel())

	case 'r':
		if w.Flag('+') {
			io.WriteString(w, "R:")
		}
		fmt.Fprint(w, v.RepetitionLevel())

	case 'q':
		if w.Flag('+') {
			io.WriteString(w, "V:")
		}
		switch v.Kind() {
		case ByteArray, FixedLenByteArray:
			fmt.Fprintf(w, "%q", v.ByteArray())
		default:
			fmt.Fprintf(w, `"%s"`, v)
		}

	case 's':
		if w.Flag('+') {
			io.WriteString(w, "V:")
		}
		switch v.Kind() {
		case Boolean:
			fmt.Fprint(w, v.Boolean())
		case Int32:
			fmt.Fprint(w, v.Int32())
		case Int64:
			fmt.Fprint(w, v.Int64())
		case Int96:
			fmt.Fprint(w, v.Int96())
		case Float:
			fmt.Fprint(w, v.Float())
		case Double:
			fmt.Fprint(w, v.Double())
		case ByteArray, FixedLenByteArray:
			w.Write(v.ByteArray())
		default:
			io.WriteString(w, "<null>")
		}

	case 'v':
		switch {
		case w.Flag('+'):
			fmt.Fprintf(w, "%+[1]c %+[1]d %+[1]r %+[1]s", v)
		case w.Flag('#'):
			fmt.Fprintf(w, "parquet.Value{%+[1]c, %+[1]d, %+[1]r, %+[1]s}", v)
		default:
			v.Format(w, 's')
		}
	}
}

// String returns a string representation of v.
func (v Value) String() string {
	switch v.Kind() {
	case Boolean:
		return strconv.FormatBool(v.Boolean())
	case Int32:
		return strconv.FormatInt(int64(v.Int32()), 10)
	case Int64:
		return strconv.FormatInt(v.Int64(), 10)
	case Int96:
		return v.Int96().String()
	case Float:
		return strconv.FormatFloat(float64(v.Float()), 'g', -1, 32)
	case Double:
		return strconv.FormatFloat(v.Double(), 'g', -1, 32)
	case ByteArray, FixedLenByteArray:
		// As an optimizations for the common case of using String on UTF8
		// columns we convert the byte array to a string without copying the
		// underlying data to a new memory location. This is safe as long as the
		// application respects the requirement to not mutate the byte slices
		// returned when calling ByteArray.
		return unsafeBytesToString(v.ByteArray())
	default:
		return "<null>"
	}
}

// GoString returns a Go value string representation of v.
func (v Value) GoString() string {
	return fmt.Sprintf("%#v", v)
}

// Level returns v with the repetition level, definition level, and column index
// set to the values passed as arguments.
//
// The method panics if either argument is negative.
func (v Value) Level(repetitionLevel, definitionLevel, columnIndex int) Value {
	v.repetitionLevel = makeRepetitionLevel(repetitionLevel)
	v.definitionLevel = makeDefinitionLevel(definitionLevel)
	v.columnIndex = ^makeColumnIndex(columnIndex)
	return v
}

// Clone returns a copy of v which does not share any pointers with it.
func (v Value) Clone() Value {
	switch k := v.Kind(); k {
	case ByteArray, FixedLenByteArray:
		b := copyBytes(v.ByteArray())
		v.ptr = *(**byte)(unsafe.Pointer(&b))
	}
	return v
}

func makeInt96(bits []byte) (i96 deprecated.Int96) {
	return deprecated.Int96{
		2: binary.LittleEndian.Uint32(bits[8:12]),
		1: binary.LittleEndian.Uint32(bits[4:8]),
		0: binary.LittleEndian.Uint32(bits[0:4]),
	}
}

func assignValue(dst reflect.Value, src Value) error {
	if src.IsNull() {
		dst.Set(reflect.Zero(dst.Type()))
		return nil
	}

	dstKind := dst.Kind()
	srcKind := src.Kind()

	var val reflect.Value
	switch srcKind {
	case Boolean:
		v := src.Boolean()
		switch dstKind {
		case reflect.Bool:
			dst.SetBool(v)
			return nil
		default:
			val = reflect.ValueOf(v)
		}

	case Int32:
		v := int64(src.Int32())
		switch dstKind {
		case reflect.Int8, reflect.Int16, reflect.Int32:
			dst.SetInt(int64(v))
			return nil
		case reflect.Uint8, reflect.Uint16, reflect.Uint32:
			dst.SetUint(uint64(v))
			return nil
		default:
			val = reflect.ValueOf(v)
		}

	case Int64:
		v := src.Int64()
		switch dstKind {
		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
			dst.SetInt(v)
			return nil
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint, reflect.Uintptr:
			dst.SetUint(uint64(v))
			return nil
		default:
			val = reflect.ValueOf(v)
		}

	case Int96:
		val = reflect.ValueOf(src.Int96())

	case Float:
		v := src.Float()
		switch dstKind {
		case reflect.Float32, reflect.Float64:
			dst.SetFloat(float64(v))
			return nil
		default:
			val = reflect.ValueOf(v)
		}

	case Double:
		v := src.Double()
		switch dstKind {
		case reflect.Float32, reflect.Float64:
			dst.SetFloat(v)
			return nil
		default:
			val = reflect.ValueOf(v)
		}

	case ByteArray:
		v := src.ByteArray()
		switch dstKind {
		case reflect.String:
			dst.SetString(string(v))
			return nil
		case reflect.Slice:
			if dst.Type().Elem().Kind() == reflect.Uint8 {
				dst.SetBytes(copyBytes(v))
				return nil
			}
		default:
			val = reflect.ValueOf(v)
		}

	case FixedLenByteArray:
		v := src.ByteArray()
		switch dstKind {
		case reflect.Array:
			if dst.Type().Elem().Kind() == reflect.Uint8 && dst.Len() == len(v) {
				// This code could be implemented as a call to reflect.Copy but
				// it would require creating a reflect.Value from v which causes
				// the heap allocation to pack the []byte value. To avoid this
				// overhead we instead convert the reflect.Value holding the
				// destination array into a byte slice which allows us to use
				// a more efficient call to copy.
				copy(unsafeByteArray(dst, len(v)), v)
				return nil
			}
		case reflect.Slice:
			if dst.Type().Elem().Kind() == reflect.Uint8 {
				dst.SetBytes(copyBytes(v))
				return nil
			}
		default:
			val = reflect.ValueOf(v)
		}
	}

	if val.IsValid() && val.Type().AssignableTo(dst.Type()) {
		dst.Set(val)
		return nil
	}

	return fmt.Errorf("cannot assign parquet value of type %s to go value of type %s", srcKind.String(), dst.Type())
}

func parseValue(kind Kind, data []byte) (val Value, err error) {
	switch kind {
	case Boolean:
		if len(data) == 1 {
			val = makeValueBoolean(data[0] != 0)
		}
	case Int32:
		if len(data) == 4 {
			val = makeValueInt32(int32(binary.LittleEndian.Uint32(data)))
		}
	case Int64:
		if len(data) == 8 {
			val = makeValueInt64(int64(binary.LittleEndian.Uint64(data)))
		}
	case Int96:
		if len(data) == 12 {
			val = makeValueInt96(makeInt96(data))
		}
	case Float:
		if len(data) == 4 {
			val = makeValueFloat(float32(math.Float32frombits(binary.LittleEndian.Uint32(data))))
		}
	case Double:
		if len(data) == 8 {
			val = makeValueDouble(float64(math.Float64frombits(binary.LittleEndian.Uint64(data))))
		}
	case ByteArray, FixedLenByteArray:
		val = makeValueBytes(kind, data)
	}
	if val.IsNull() {
		err = fmt.Errorf("cannot decode %s value from input of length %d", kind, len(data))
	}
	return val, err
}

func copyBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

func unsafeBytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func unsafePointerOf(v reflect.Value) unsafe.Pointer {
	return (*[2]unsafe.Pointer)(unsafe.Pointer(&v))[1]
}

func unsafeByteArray(v reflect.Value, n int) []byte {
	return unsafe.Slice((*byte)(unsafePointerOf(v)), n)
}

// Equal returns true if v1 and v2 are equal.
//
// Values are considered equal if they are of the same physical type and hold
// the same Go values. For BYTE_ARRAY and FIXED_LEN_BYTE_ARRAY, the content of
// the underlying byte arrays are tested for equality.
//
// Note that the repetition levels, definition levels, and column indexes are
// not compared by this function.
func Equal(v1, v2 Value) bool {
	if v1.kind != v2.kind {
		return false
	}
	switch v1.Kind() {
	case Boolean:
		return v1.Boolean() == v2.Boolean()
	case Int32:
		return v1.Int32() == v2.Int32()
	case Int64:
		return v1.Int64() == v2.Int64()
	case Int96:
		return v1.Int96() == v2.Int96()
	case Float:
		return v1.Float() == v2.Float()
	case Double:
		return v1.Double() == v2.Double()
	case ByteArray, FixedLenByteArray:
		return bytes.Equal(v1.ByteArray(), v2.ByteArray())
	case -1: // null
		return true
	default:
		return false
	}
}

var (
	_ fmt.Formatter = Value{}
	_ fmt.Stringer  = Value{}
)

func clearValues(values []Value) {
	for i := range values {
		values[i] = Value{}
	}
}

type errorValueReader struct{ err error }

func (r *errorValueReader) ReadValues([]Value) (int, error) { return 0, r.err }
