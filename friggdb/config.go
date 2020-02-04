package friggdb

import (
	"time"

	"github.com/grafana/frigg/friggdb/backend/local"
)

type Config struct {
	Backend string       `yaml:"backend"`
	Local   local.Config `yaml:"local"`

	BlocklistRefreshRate     time.Duration `yaml:"blocklistRefreshRate"`
	WALFilepath              string        `yaml:"walpath"`
	IndexDownsample          int           `yaml:"index-downsample"`
	BloomFilterFalsePositive float64       `yaml:"bloom-filter-false-positive"`
}
