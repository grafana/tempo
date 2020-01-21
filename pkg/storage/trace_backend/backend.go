package trace_backend

import "github.com/google/uuid"

type BloomIter func(bytes []byte, blockID uuid.UUID) (bool, error)

type Writer interface {
	Write(blockID uuid.UUID, tenantID string, bBloom []byte, bIndex []byte, tracesFilePath string) error
}

type Reader interface {
	Bloom(tenantID string, fn BloomIter) error
	Index(blockID uuid.UUID, tenantID string) ([]byte, error)
	Trace(blockID uuid.UUID, tenantID string, start uint64, length uint32) ([]byte, error)
}
