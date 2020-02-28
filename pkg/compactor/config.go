package compactor

import (
	"flag"
	"time"
)

type Config struct {
	ChunkSizeBytes          uint32        `yaml:"chunkSizeBytes"`
	MaxCompactionRange      time.Duration `yaml:"maxCompactionRange"`
	BlockRetention          time.Duration `yaml:"blockRetention"`
	CompactedBlockRetention time.Duration `yaml:"compactedBlockRetention"`
	MaintenanceCycle        time.Duration `yaml:"maintenanceCycle"`
}

// RegisterFlags registers the flags.
func (cfg *Config) RegisterFlags(f *flag.FlagSet) {

}
