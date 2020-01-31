package storage

import (
	"fmt"

	"github.com/cortexproject/cortex/pkg/chunk/storage"
	"github.com/grafana/frigg/pkg/storage/trace_backend"
	"github.com/grafana/frigg/pkg/storage/trace_backend/local"
)

// Store is the Frigg chunk store to retrieve and save chunks.
type Store interface {
	TraceReader
	TraceWriter
}

type store struct {
	cfg Config

	TraceReader
	TraceWriter
}

// NewStore creates a new Frigg Store using configuration supplied.
func NewStore(cfg Config, limits storage.StoreLimits) (Store, error) {
	r, w, err := NewTraceStore(cfg.Trace)
	if err != nil {
		return nil, err
	}

	return &store{
		cfg:         cfg,
		TraceReader: r,
		TraceWriter: w,
	}, nil
}

func NewTraceStore(cfg TraceConfig) (TraceReader, TraceWriter, error) {
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
