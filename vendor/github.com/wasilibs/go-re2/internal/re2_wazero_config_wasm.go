//go:build !tinygo.wasm && !re2_cgo && wasm

package internal

// While we normally use pointer size to determine a 32-bit system, Go
// uses 64-bit pointers even with a 32-bit address space. So we go ahead
// and recognize this special case and lower our max pages.
const defaultMaxPages = uint32(65536 / 4)
