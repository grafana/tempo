//go:build !tinygo.wasm && !re2_cgo && !wasm

package internal

const defaultMaxPages = uint32(65536)
