package storage

import (
	"fmt"

	"github.com/cortexproject/cortex/pkg/chunk"
	"github.com/cortexproject/cortex/pkg/chunk/storage"
	"github.com/grafana/frigg/pkg/storage/trace_backend"
	"github.com/grafana/frigg/pkg/storage/trace_backend/local"
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
	var err error
	var r trace_backend.Reader
	var w trace_backend.Writer

	switch cfg.Backend {
	case "local":
		r, w, err = local.New(cfg.Local)
	default:
		err = fmt.Errorf("unknown local %s", cfg.Backend)
	}

	if err != nil {
		return nil, nil, err
	}

	if cfg.BloomFilterFalsePositive <= 0.0 {
		return nil, nil, fmt.Errorf("invalid bloom filter fp rate %v", cfg.BloomFilterFalsePositive)
	}

	rw := &readerWriter{
		r:       r,
		w:       w,
		bloomFP: cfg.BloomFilterFalsePositive,
	}

	return rw, rw, nil
}
