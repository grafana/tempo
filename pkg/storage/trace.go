package storage

import (
	"io"

	"github.com/google/uuid"

	"github.com/joe-elliott/frigg/pkg/friggpb"
)

type TraceID []byte

type TraceRecord struct {
	TraceID []byte
	start   int64
	length  int
}

type TraceWriter interface {
	WriteBlock(records []TraceRecord, block io.Reader, blockID uuid.UUID, tenantID string)
}

type TraceReader interface {
	FindTrace(id TraceID) *friggpb.Trace
}
