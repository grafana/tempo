package encoding

import (
	"fmt"

	"github.com/grafana/tempo/tempodb/backend"
)

// BlockConfig holds configuration options for newly created blocks
type BlockConfig struct {
	IndexDownsample int              `yaml:"index_downsample"`
	BloomFP         float64          `yaml:"bloom_filter_false_positive"`
	Encoding        backend.Encoding `yaml:"encoding"`
}

// ValidateConfig returns true if the config is valid
func ValidateConfig(b *BlockConfig) error {
	if b.IndexDownsample == 0 {
		return fmt.Errorf("Non-zero index downsample required")
	}

	if b.BloomFP <= 0.0 {
		return fmt.Errorf("invalid bloom filter fp rate %v", b.BloomFP)
	}

	return nil
}
