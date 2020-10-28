package diskcache

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type bloomMissFunc func(ctx context.Context, blockID uuid.UUID, tenantID string, bloomShard int) ([]byte, error)
type indexMissFunc func(ctx context.Context, blockID uuid.UUID, tenantID string) ([]byte, error)

const (
	typeBloom = "bloom"
	typeIndex = "index"
)

var (
	metricDiskCacheMiss = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "disk_cache_miss_total",
		Help:      "Total number of times the disk cache missed.",
	}, []string{"type"})
	metricDiskCache = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "disk_cache_total",
		Help:      "Total number of times the disk cache was queried.",
	}, []string{"type", "status"})
	metricDiskCacheClean = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "disk_cache_clean_total",
		Help:      "Total number of times a disk clean has occurred.",
	}, []string{"status"})
)

type reader struct {
	cfg  *Config
	next backend.Reader

	logger log.Logger
	stopCh chan struct{}
}

func New(next backend.Reader, cfg *Config, logger log.Logger) (backend.Reader, error) {
	// cleanup disk cache dir
	err := os.RemoveAll(cfg.Path)
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(cfg.Path, os.ModePerm)
	if err != nil {
		return nil, err
	}

	if cfg.DiskPruneCount == 0 {
		return nil, fmt.Errorf("must specify disk prune count")
	}

	if cfg.DiskCleanRate == 0 {
		return nil, fmt.Errorf("must specify a clean rate")
	}

	if cfg.MaxDiskMBs == 0 {
		return nil, fmt.Errorf("must specify a maximum number of MBs to save")
	}

	r := &reader{
		cfg:    cfg,
		next:   next,
		stopCh: make(chan struct{}),
		logger: logger,
	}

	r.startJanitor()

	return r, nil
}

func (r *reader) Tenants(ctx context.Context) ([]string, error) {
	return r.next.Tenants(ctx)
}

func (r *reader) Blocks(ctx context.Context, tenantID string) ([]uuid.UUID, error) {
	return r.next.Blocks(ctx, tenantID)
}

func (r *reader) BlockMeta(ctx context.Context, blockID uuid.UUID, tenantID string) (*encoding.BlockMeta, error) {
	return r.next.BlockMeta(ctx, blockID, tenantID)
}

func (r *reader) Bloom(ctx context.Context, blockID uuid.UUID, tenantID string, bloomShard int) ([]byte, error) {
	b, skippableErr, err := r.readOrCacheBloom(ctx, blockID, tenantID, bloomShard, r.next.Bloom)

	if skippableErr != nil {
		metricDiskCache.WithLabelValues(typeBloom, "error").Inc()
		level.Error(r.logger).Log("err", skippableErr)
	} else {
		metricDiskCache.WithLabelValues(typeBloom, "success").Inc()
	}

	return b, err
}

func (r *reader) Index(ctx context.Context, blockID uuid.UUID, tenantID string) ([]byte, error) {
	b, skippableErr, err := r.readOrCacheIndex(ctx, blockID, tenantID, r.next.Index)

	if skippableErr != nil {
		metricDiskCache.WithLabelValues(typeIndex, "error").Inc()
		level.Error(r.logger).Log("err", skippableErr)
	} else {
		metricDiskCache.WithLabelValues(typeIndex, "success").Inc()
	}

	return b, err
}

func (r *reader) Object(ctx context.Context, blockID uuid.UUID, tenantID string, start uint64, buffer []byte) error {
	// not attempting to cache these...yet...
	return r.next.Object(ctx, blockID, tenantID, start, buffer)
}

func (r *reader) Shutdown() {
	r.stopCh <- struct{}{}
	r.next.Shutdown()
}

func key(blockID uuid.UUID, tenantID string, t string) string {
	return blockID.String() + ":" + tenantID + ":" + t
}

func bloomKey(blockID uuid.UUID, tenantID string, t string, shardNum int) string {
	return blockID.String() + ":" + tenantID + ":" + t + ":" + strconv.Itoa(shardNum)
}
