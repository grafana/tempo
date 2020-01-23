package storage

import (
	"fmt"

	"github.com/cortexproject/cortex/pkg/chunk"
	"github.com/cortexproject/cortex/pkg/chunk/storage"
	"github.com/joe-elliott/frigg/pkg/storage/trace_backend"
	"github.com/joe-elliott/frigg/pkg/storage/trace_backend/local"
)

// Store is the Frigg chunk store to retrieve and save chunks.
type Store interface {
	chunk.Store
	TraceReader
	TraceWriter
}

type store struct {
	cfg Config

	chunk.Store
	TraceReader
	TraceWriter
}

// NewStore creates a new Loki Store using configuration supplied.
func NewStore(cfg Config, storeCfg chunk.StoreConfig, schemaCfg chunk.SchemaConfig, limits storage.StoreLimits) (Store, error) {
	s, err := storage.NewStore(cfg.Columnar, storeCfg, schemaCfg, limits)
	if err != nil {
		return nil, err
	}
	r, w, err := newTraceStore(cfg.Trace)
	if err != nil {
		return nil, err
	}

	return &store{
		Store:       s,
		cfg:         cfg,
		TraceReader: r,
		TraceWriter: w,
	}, nil
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

	return rw, rw, nil
}
