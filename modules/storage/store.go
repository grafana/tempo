package storage

import (
	"github.com/go-kit/kit/log"
	"github.com/grafana/tempo/tempodb"
)

// Store is the Tempo chunk store to retrieve and save chunks.
type Store interface {
	tempodb.Reader
	tempodb.Writer
	tempodb.Compactor
}

type store struct {
	cfg Config

	tempodb.Reader
	tempodb.Writer
	tempodb.Compactor
}

// NewStore creates a new Tempo Store using configuration supplied.
func NewStore(cfg Config, logger log.Logger) (Store, error) {
	r, w, c, err := tempodb.New(&cfg.Trace, logger)
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
