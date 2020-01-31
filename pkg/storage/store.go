package storage

import (
	"github.com/cortexproject/cortex/pkg/chunk/storage"
	"github.com/grafana/frigg/friggdb"
)

// Store is the Frigg chunk store to retrieve and save chunks.
type Store interface {
	friggdb.Reader
	friggdb.Writer
}

type store struct {
	cfg Config

	friggdb.Reader
	friggdb.Writer
}

// NewStore creates a new Frigg Store using configuration supplied.
func NewStore(cfg Config, limits storage.StoreLimits) (Store, error) {
	r, w, err := friggdb.New(&cfg.Trace)
	if err != nil {
		return nil, err
	}

	return &store{
		cfg:    cfg,
		Reader: r,
		Writer: w,
	}, nil
}
