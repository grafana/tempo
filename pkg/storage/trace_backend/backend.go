package trace_backend

import (
	"io"
)

type BloomIter func(bytes []byte, blockID string) bool

type Writer interface {
	Write(bBloom []byte, bIndex []byte, bTraces io.Reader) error
}

type Reader interface {
	Bloom(fn BloomIter) error
	Index(name string) ([]byte, error)
	Trace(name string, start uint32, length uint32) ([]byte, error)
}
