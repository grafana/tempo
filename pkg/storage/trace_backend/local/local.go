package local

import (
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

func (w *readerWriter) Write(name string, bBloom []byte, bIndex []byte, tracesFilePath string) error {
	return nil
}

func (w *readerWriter) Bloom(instanceID string, fn trace_backend.BloomIter) error {
	return nil
}

func (w *readerWriter) Index(name string) ([]byte, error) {
	return nil, nil
}

func (w *readerWriter) Trace(name string, start uint64, length uint32) ([]byte, error) {
	return nil, nil
}
