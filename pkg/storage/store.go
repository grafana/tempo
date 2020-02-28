package storage

import (
	"github.com/cortexproject/cortex/pkg/chunk/storage"
	"github.com/go-kit/kit/log"
	"github.com/grafana/frigg/friggdb"
	"github.com/grafana/frigg/friggdb/backend"
)

// Store is the Frigg chunk store to retrieve and save chunks.
type Store interface {
	friggdb.BlockStore
	GetBackendConfig() friggdb.Config
}

type store struct {
	cfg Config

	friggdb.BlockStore
}

// NewStore creates a new Frigg Store using configuration supplied.
func NewStore(cfg Config, limits storage.StoreLimits, logger log.Logger) (Store, error) {
	b, err := friggdb.New(&cfg.Trace, logger)
	if err != nil {
		return nil, err
	}

	return &store{
		cfg:        cfg,
		BlockStore: b,
	}, nil
}

func (s *store) GetBackendConfig() friggdb.Config {
	return s.cfg.Trace
}

func (s *store) GetBackendReader() backend.Reader {
	return s.BlockStore.GetBackendReader()
}
