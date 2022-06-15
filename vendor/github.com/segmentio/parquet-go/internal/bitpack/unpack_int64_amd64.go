//go:build !purego

package bitpack

import "github.com/segmentio/parquet-go/internal/unsafecast"

//go:noescape
func unpackInt64Default(dst []int64, src []byte, bitWidth uint)

func unpackInt64(dst []int64, src []byte, bitWidth uint) {
	switch {
	case bitWidth == 64:
		copy(dst, unsafecast.BytesToInt64(src))
	default:
		unpackInt64Default(dst, src, bitWidth)
	}
}
