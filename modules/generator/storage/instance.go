package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	prometheus_config "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/scrape"
	"github.com/prometheus/prometheus/storage"
	"github.com/prometheus/prometheus/storage/remote"
	"github.com/prometheus/prometheus/tsdb/agent"
	tsdb_errors "github.com/prometheus/prometheus/tsdb/errors"
)

type Storage interface {
	storage.Appendable

	// Close closes the storage and all its underlying resources.
	Close() error
}

type storageImpl struct {
	walDir  string
	storage storage.Storage

	logger log.Logger
}

var _ Storage = (*storageImpl)(nil)

// New creates a metrics WAL that remote writes its data.
func New(cfg *Config, tenant string, reg prometheus.Registerer, logger log.Logger) (Storage, error) {
	logger = log.With(logger, "tenant", tenant)
	reg = prometheus.WrapRegistererWith(prometheus.Labels{"tenant": tenant}, reg)

	walDir := filepath.Join(cfg.Path, tenant)

	level.Info(logger).Log("msg", "creating WAL", "dir", walDir)

	// Create WAL directory with necessary permissions
	// This creates both <walDir>/<tenant>/ and <walDir>/<tenant>/wal/. If we don't create the wal
	// subdirectory remote storage logs a scary error.
	err := os.MkdirAll(filepath.Join(walDir, "wal"), 0o755)
	if err != nil {
		return nil, fmt.Errorf("could not create directory for metrics WAL: %w", err)
	}

	// Set up remote storage writer
	startTimeCallback := func() (int64, error) {
		return int64(model.Latest), nil
	}
	remoteStorage := remote.NewStorage(log.With(logger, "component", "remote"), reg, startTimeCallback, walDir, cfg.RemoteWriteFlushDeadline, &noopScrapeManager{})

	remoteStorageConfig := &prometheus_config.Config{
		RemoteWriteConfigs: generateTenantRemoteWriteConfigs(cfg.RemoteWrite, tenant, cfg.RemoteWriteRemoveOrgIDHeader, logger),
	}

	err = remoteStorage.ApplyConfig(remoteStorageConfig)
	if err != nil {
		return nil, err
	}

	// Set up WAL
	wal, err := agent.Open(log.With(logger, "component", "wal"), reg, remoteStorage, walDir, cfg.Wal.toPrometheusAgentOptions())
	if err != nil {
		return nil, err
	}

	return &storageImpl{
		walDir:  walDir,
		storage: storage.NewFanout(logger, wal, remoteStorage),

		logger: logger,
	}, nil
}

func (s *storageImpl) Appender(ctx context.Context) storage.Appender {
	return s.storage.Appender(ctx)
}

func (s *storageImpl) Close() error {
	level.Info(s.logger).Log("msg", "closing WAL", "dir", s.walDir)

	return tsdb_errors.NewMulti(
		s.storage.Close(),
		func() error {
			// remove the WAL at shutdown since remote write starts at the end of the WAL anyways
			// https://github.com/prometheus/prometheus/issues/8809
			return os.RemoveAll(s.walDir)
		}(),
	).Err()
}

type noopScrapeManager struct{}

func (noop *noopScrapeManager) Get() (*scrape.Manager, error) {
	return nil, errors.New("scrape manager not implemented")
}
