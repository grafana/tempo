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
	c := &Compactor{
		cfg:   &cfg,
		store: store,
	}

	store.EnableCompaction(cfg.Compactor, c)

	return c, nil
}

func (c *Compactor) Owns(hash string) bool {
	return true
}

func (c *Compactor) Combine(objA []byte, objB []byte) []byte {
	if len(objA) > len(objB) {
		return objA
	}

	return objB
}
