package storage

import (
	"context"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/services"

	"github.com/grafana/tempo/v3/pkg/cache"
	"github.com/grafana/tempo/v3/pkg/usagestats"
	"github.com/grafana/tempo/v3/tempodb"
	"github.com/grafana/tempo/v3/tempodb/blocklist"
)

var (
	statCache   = usagestats.NewString("storage_cache")
	statBackend = usagestats.NewString("storage_backend")
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

	// set by EnablePolling, so stopping can end the polling loop
	stopPoller context.CancelFunc

	tempodb.Reader
	tempodb.Writer
	tempodb.Compactor
}

// NewStore creates a new Tempo Store using configuration supplied.
func NewStore(cfg Config, cacheProvider cache.Provider, logger log.Logger) (Store, error) {
	statCache.Set(cfg.Trace.Cache)
	statBackend.Set(cfg.Trace.Backend)

	r, w, c, err := tempodb.New(&cfg.Trace, cacheProvider, logger)
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

// EnablePolling owns the poller's cancel because Reader.Shutdown waits for the loop to exit.
func (s *store) EnablePolling(ctx context.Context, sharder blocklist.JobSharder, skipNoCompactBlocks bool) {
	ctx, s.stopPoller = context.WithCancel(ctx)

	s.Reader.EnablePolling(ctx, sharder, skipNoCompactBlocks)
}

func (s *store) stopping(_ error) error {
	if s.stopPoller != nil {
		s.stopPoller()
	}

	s.Reader.Shutdown()

	return nil
}
