package storage

import (
	"io"

	"github.com/google/uuid"

	"github.com/joe-elliott/frigg/pkg/friggpb"
	"github.com/joe-elliott/frigg/pkg/storage/trace_backend"
)

type TraceID []byte

type TraceRecord struct {
	TraceID []byte
	start   uint32
	length  uint32
}

type TraceWriter interface {
	WriteBlock(blockID uuid.UUID, tenantID string, records []TraceRecord, block io.Reader) error
}

type TraceReader interface {
	FindTrace(id TraceID) (*friggpb.Trace, error)
}

type readerWriter struct {
	r trace_backend.Reader
	w trace_backend.Writer
}

func (rw *readerWriter) WriteBlock(blockID uuid.UUID, tenantID string, records []TraceRecord, block io.Reader) error {
	return nil
}

func (rw *readerWriter) FindTrace(id TraceID) (*friggpb.Trace, error) {
	return nil, nil
}
