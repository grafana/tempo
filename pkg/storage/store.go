package storage

import (
	"github.com/cortexproject/cortex/pkg/chunk/storage"
	"github.com/go-kit/kit/log"
	"github.com/grafana/frigg/friggdb"
)

// Store is the Frigg chunk store to retrieve and save chunks.
type Store interface {
	friggdb.Reader
	friggdb.Writer
	friggdb.Compactor
}

type store struct {
	cfg Config

	friggdb.Reader
	friggdb.Writer
	friggdb.Compactor
}

// NewStore creates a new Frigg Store using configuration supplied.
func NewStore(cfg Config, limits storage.StoreLimits, logger log.Logger) (Store, error) {
	r, w, c, err := friggdb.New(&cfg.Trace, logger)
	if err != nil {
		return nil, err
	}

	return &store{
		cfg:       cfg,
		Reader:    r,
		Writer:    w,
		Compactor: c,
	}, nil
}
