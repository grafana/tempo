//go:build go1.18

package parquet

import (
	"reflect"
	"unsafe"
)

// This function exists for backward compatiblity with the Go 1.17 build which
// has a different implementation.
//
// TODO: remove when we drop support for Go versions prior to 1.18.
func unsafePointer(v reflect.Value) unsafe.Pointer { return v.UnsafePointer() }
