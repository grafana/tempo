package local

import (
	"github.com/google/uuid"
	"github.com/joe-elliott/frigg/pkg/storage/trace_backend"
)

type readerWriter struct {
	cfg Config
}

func New(cfg Config) (trace_backend.Reader, trace_backend.Writer) {
	rw := &readerWriter{
		cfg: cfg,
	}

	return rw, rw
}

func (rw *readerWriter) Write(blockID uuid.UUID, tenantID string, bBloom []byte, bIndex []byte, tracesFilePath string) error {

	return nil
}

func (rw *readerWriter) Bloom(tenantID string, fn trace_backend.BloomIter) error {
	return nil
}

func (rw *readerWriter) Index(blockID uuid.UUID, tenantID string) ([]byte, error) {
	return nil, nil
}

func (rw *readerWriter) Trace(blockID uuid.UUID, tenantID string, start uint64, length uint32) ([]byte, error) {
	return nil, nil
}

func (rw *readerWriter) bloomFileName(blockID uuid.UUID, tenantID string) string {
	return ""
}
