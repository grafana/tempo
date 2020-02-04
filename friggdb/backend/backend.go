package backend

import (
	"github.com/google/uuid"
)

type Writer interface {
	Write(blockID uuid.UUID, tenantID string, bMeta []byte, bBloom []byte, bIndex []byte, tracesFilePath string) error
}

type Reader interface {
	Tenants() ([]string, error)
	Blocklist(tenantID string) ([][]byte, error)
	Bloom(blockID uuid.UUID, tenantID string) ([]byte, error)
	Index(blockID uuid.UUID, tenantID string) ([]byte, error)
	Object(blockID uuid.UUID, tenantID string, start uint64, length uint32) ([]byte, error)
}
