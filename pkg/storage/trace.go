package storage

import (
	"fmt"
	"io"

	"github.com/google/uuid"

	"github.com/joe-elliott/frigg/pkg/friggpb"
	"github.com/joe-elliott/frigg/pkg/storage/trace_backend"
	"github.com/joe-elliott/frigg/pkg/storage/trace_backend/local"
)

type TraceConfig struct {
	Engine string       `yaml:"engine"`
	Local  local.Config `yaml:"local"`
}

type TraceID []byte

type TraceRecord struct {
	TraceID []byte
	start   uint32
	length  uint32
}

type TraceWriter interface {
	WriteBlock(records []TraceRecord, block io.Reader, blockID uuid.UUID, tenantID string) error
}

type TraceReader interface {
	FindTrace(id TraceID) (*friggpb.Trace, error)
}

type readerWriter struct {
	r trace_backend.Reader
	w trace_backend.Writer
}

func newTraceStore(cfg TraceConfig) (TraceReader, TraceWriter, error) {
	var r trace_backend.Reader
	var w trace_backend.Writer

	switch cfg.Engine {
	case "local":
		r, w = local.New(cfg.Local)
	default:
		return nil, nil, fmt.Errorf("unknown engine %s", cfg.Engine)
	}

	rw := &readerWriter{
		r: r,
		w: w,
	}

	return rw, rw, fmt.Errorf("unknown engine %s", cfg.Engine)
}

func (rw *readerWriter) WriteBlock(records []TraceRecord, block io.Reader, blockID uuid.UUID, tenantID string) error {
	return nil
}

func (rw *readerWriter) FindTrace(id TraceID) (*friggpb.Trace, error) {
	return nil, nil
}
