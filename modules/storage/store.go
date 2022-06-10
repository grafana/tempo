package storage

import (
	"context"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/services"

	"github.com/grafana/tempo/pkg/usagestats"
	"github.com/grafana/tempo/tempodb"
)

var (
	cacheStats               = usagestats.NewString("storage_cache")
	backendStats             = usagestats.NewString("storage_backend")
	walEncodingStats         = usagestats.NewString("storage_wal_encoding")
	walSearchEncodingStats   = usagestats.NewString("storage_wal_search_encoding")
	blockEncodingStats       = usagestats.NewString("storage_block_encoding")
	blockSearchEncodingStats = usagestats.NewString("storage_block_search_encoding")
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

	cacheStats.Set(cfg.Trace.Cache)
	backendStats.Set(cfg.Trace.Backend)
	walEncodingStats.Set(cfg.Trace.WAL.Encoding.String())
	walSearchEncodingStats.Set(cfg.Trace.WAL.SearchEncoding.String())
	blockEncodingStats.Set(cfg.Trace.Block.Encoding.String())
	blockSearchEncodingStats.Set(cfg.Trace.Block.SearchEncoding.String())

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

	s.Service = services.NewIdleService(s.starting, s.stopping)
	return s, nil
}

func (s *store) starting(_ context.Context) error {
	return nil
}

func (s *store) stopping(_ error) error {
	s.Reader.Shutdown()

	return nil
}
