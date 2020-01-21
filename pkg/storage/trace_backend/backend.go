package trace_backend

import "github.com/google/uuid"

type BloomIter func(bytes []byte, blockID uuid.UUID) (bool, error)

type Writer interface {
	Write(path string, bBloom []byte, bIndex []byte, tracesFilePath string) error
}

type Reader interface {
	Bloom(instanceID string, fn BloomIter) error
	Index(path string) ([]byte, error)
	Trace(path string, start uint64, length uint32) ([]byte, error)
}
