package compactor

import (
	"github.com/grafana/frigg/pkg/storage"
)

type Compactor struct {
	cfg   *Config
	store storage.Store
}

// New makes a new Querier.
func New(cfg Config, store storage.Store) (*Compactor, error) {
	return &Compactor{
		cfg:   &cfg,
		store: store,
	}, nil
}
