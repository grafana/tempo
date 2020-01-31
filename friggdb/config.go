package friggdb

import "github.com/grafana/frigg/friggdb/backend/local"

type Config struct {
	Backend string       `yaml:"backend"`
	Local   local.Config `yaml:"local"`

	WALFilepath              string  `yaml:"walpath"`
	BloomFilterFalsePositive float64 `yaml:"bloom-filter-false-positive"`
}
