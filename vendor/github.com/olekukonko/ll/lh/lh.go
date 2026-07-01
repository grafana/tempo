package lh

import (
	"fmt"
	"strconv"
	"strings"
)

// rightPad pads a string with spaces on the right to reach the specified length.
// Returns the original string if it's already at or exceeds the target length.
// Uses strings.Builder for efficient memory allocation.
func rightPad(str string, length int) string {
	if len(str) >= length {
		return str
	}
	var sb strings.Builder
	sb.Grow(length)
	sb.WriteString(str)
	sb.WriteString(strings.Repeat(" ", length-len(str)))
	return sb.String()
}

// stringWriter is the interface for types that can write strings and bytes.
// Both *strings.Builder and *bytes.Buffer implement this.
type stringWriter interface {
	WriteString(s string) (int, error)
	Write(p []byte) (n int, err error)
}

// writeFieldValue writes a field value to the builder using type switches
// to avoid reflection and allocations associated with fmt.Fprint.
func writeFieldValue(b stringWriter, v interface{}) {
	switch val := v.(type) {
	case string:
		b.WriteString(val)
	case int:
		b.WriteString(strconv.Itoa(val))
	case int8:
		b.WriteString(strconv.FormatInt(int64(val), 10))
	case int16:
		b.WriteString(strconv.FormatInt(int64(val), 10))
	case int32:
		b.WriteString(strconv.FormatInt(int64(val), 10))
	case int64:
		b.WriteString(strconv.FormatInt(val, 10))
	case uint:
		b.WriteString(strconv.FormatUint(uint64(val), 10))
	case uint8:
		b.WriteString(strconv.FormatUint(uint64(val), 10))
	case uint16:
		b.WriteString(strconv.FormatUint(uint64(val), 10))
	case uint32:
		b.WriteString(strconv.FormatUint(uint64(val), 10))
	case uint64:
		b.WriteString(strconv.FormatUint(val, 10))
	case float32:
		b.WriteString(strconv.FormatFloat(float64(val), 'g', -1, 32))
	case float64:
		b.WriteString(strconv.FormatFloat(val, 'g', -1, 64))
	case bool:
		if val {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case nil:
		b.WriteString("nil")
	case error:
		b.WriteString(val.Error())
	case fmt.Stringer:
		b.WriteString(val.String())
	default:
		fmt.Fprint(b, val)
	}
}
