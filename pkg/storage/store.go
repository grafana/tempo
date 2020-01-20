package storage

import (
	"github.com/cortexproject/cortex/pkg/chunk"
	"github.com/cortexproject/cortex/pkg/chunk/storage"
)

// Store is the Loki chunk store to retrieve and save chunks.
type Store interface {
	chunk.Store
}

type store struct {
	chunk.Store
	cfg Config
}

// NewStore creates a new Loki Store using configuration supplied.
func NewStore(cfg Config, storeCfg chunk.StoreConfig, schemaCfg chunk.SchemaConfig, limits storage.StoreLimits) (Store, error) {
	s, err := storage.NewStore(cfg.Config, storeCfg, schemaCfg, limits)
	if err != nil {
		return nil, err
	}
	return &store{
		Store: s,
		cfg:   cfg,
	}, nil
}
