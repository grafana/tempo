//go:build !go1.18

package parquet

import (
	"reflect"
	"unsafe"
)

func unsafePointer(v reflect.Value) unsafe.Pointer {
	// This may not have been a safe conversion but there were no better way
	// prior to Go 1.18 and the introduction of reflect.Value.UnsafePointer.
	return unsafe.Pointer(v.Pointer())
}
