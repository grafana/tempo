package storage

import (
	"context"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/services"

	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/pkg/usagestats"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/blocklist"
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

	cfg           Config
	pollingCancel context.CancelFunc

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

// EnablePolling overrides the embedded method to capture a cancellable context
// so that the polling goroutine exits promptly when the store service stops.
func (s *store) EnablePolling(ctx context.Context, sharder blocklist.JobSharder, skipNoCompact bool) {
	pollingCtx, cancel := context.WithCancel(context.Background())
	s.pollingCancel = cancel
	s.Reader.EnablePolling(pollingCtx, sharder, skipNoCompact)
}

func (s *store) stopping(_ error) error {
	if s.pollingCancel != nil {
		s.pollingCancel()
	}
	s.Reader.Shutdown()

	return nil
}
