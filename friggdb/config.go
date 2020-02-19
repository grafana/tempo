package friggdb

import (
	"time"

	"github.com/grafana/frigg/friggdb/backend/cache"
	"github.com/grafana/frigg/friggdb/backend/gcs"
	"github.com/grafana/frigg/friggdb/backend/local"
	"github.com/grafana/frigg/friggdb/pool"
	"github.com/grafana/frigg/friggdb/wal"
)

type Config struct {
	Backend   string           `yaml:"backend"`
	Local     *local.Config    `yaml:"local"`
	GCS       *gcs.Config      `yaml:"gcs"`
	Cache     *cache.Config    `yaml:"cache"`
	Pool      *pool.Config     `yaml:"query_pool,omitempty"`
	WAL       *wal.Config      `yaml:"wal"`
	Compactor *compactorConfig `yaml:"compactor"`

	BlocklistRefreshRate time.Duration `yaml:"blocklistRefreshRate"`
}
