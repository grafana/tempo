package compactor

import (
	"github.com/grafana/tempo/pkg/storage"
)

type Compactor struct {
	cfg   *Config
	store storage.Store
}

// New makes a new Querier.
func New(cfg Config, store storage.Store) (*Compactor, error) {
	store.EnableCompaction(cfg.Compactor)

	return &Compactor{
		cfg:   &cfg,
		store: store,
	}, nil
}
