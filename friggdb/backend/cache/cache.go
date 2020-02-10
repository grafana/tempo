package cache

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/grafana/frigg/friggdb/backend"
)

type reader struct {
	cfg  *Config
	next backend.Reader

	stopCh chan struct{}
}

// jpe: add shutdown method?  stop stopCh?

func New(next backend.Reader, cfg *Config) (backend.Reader, error) {
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
		stopCh: make(chan struct{}, 0),
	}

	go r.startJanitor()

	return r, nil
}

func (r *reader) Tenants() ([]string, error) {
	return r.next.Tenants()
}

func (r *reader) Blocklist(tenantID string) ([][]byte, error) {
	return r.next.Blocklist(tenantID)
}

// jpe: how to force cache all blooms at the start
func (r *reader) Bloom(blockID uuid.UUID, tenantID string) ([]byte, error) {
	return r.readOrCacheKeyToDisk(blockID, tenantID, "bloom", r.next.Bloom)
}

func (r *reader) Index(blockID uuid.UUID, tenantID string) ([]byte, error) {
	return r.readOrCacheKeyToDisk(blockID, tenantID, "index", r.next.Index)
}

func (r *reader) Object(blockID uuid.UUID, tenantID string, start uint64, length uint32) ([]byte, error) {
	// not attempting to cache these...yet...
	return r.next.Object(blockID, tenantID, start, length)
}

func key(blockID uuid.UUID, tenantID string, t string) string {
	return blockID.String() + ":" + tenantID + ":" + t
}
