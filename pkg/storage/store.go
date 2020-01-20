package storage

import (
	"github.com/cortexproject/cortex/pkg/chunk"
	"github.com/cortexproject/cortex/pkg/chunk/storage"
)

// Store is the Frigg chunk store to retrieve and save chunks.
type Store interface {
	chunk.Store
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
