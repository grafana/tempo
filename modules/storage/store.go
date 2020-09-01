package storage

import (
	"github.com/cortexproject/cortex/pkg/util/services"
	"github.com/go-kit/kit/log"
	"github.com/grafana/tempo/tempodb"
)

// Store wraps the tempodb storage layer
type Store interface {
	services.Service

	tempodb.Reader
	tempodb.Writer
	tempodb.Compactor
}

type store struct {
	services.Service

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

	s := &store{
		cfg:       cfg,
		Reader:    r,
		Writer:    w,
		Compactor: c,
	}

	s.Service = services.NewBasicService(nil, nil, s.stopping)

	return s, nil
}

func (s *store) stopping(_ error) error {
	s.Reader.Shutdown()

	return nil
}
