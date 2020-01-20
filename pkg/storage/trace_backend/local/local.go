package local

import (
	"io"

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

func (w *readerWriter) Write(bBloom []byte, bIndex []byte, bTraces io.Reader) error {
	return nil
}

func (w *readerWriter) Bloom(fn trace_backend.BloomIter) error {
	return nil
}

func (w *readerWriter) Index(name string) ([]byte, error) {
	return nil, nil
}

func (w *readerWriter) Trace(name string, start uint32, length uint32) ([]byte, error) {
	return nil, nil
}
