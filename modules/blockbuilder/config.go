package blockbuilder

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type BlockConfig struct {
	MaxBlockBytes uint64 `yaml:"max_block_bytes" doc:"Maximum size of a block."`

	BlockCfg common.BlockConfig `yaml:"-,inline"`
}

func (c *BlockConfig) RegisterFlags(f *flag.FlagSet) {
	c.RegisterFlagsAndApplyDefaults("", f)
}

func (c *BlockConfig) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.Uint64Var(&c.MaxBlockBytes, prefix+".max-block-bytes", 20*1024*1024, "Maximum size of a block.") // TODO - Review default

	c.BlockCfg.Version = encoding.DefaultEncoding().Version()
	c.BlockCfg.RegisterFlagsAndApplyDefaults(prefix, f)
}

type Config struct {
	InstanceID           string             `yaml:"instance_id" doc:"Instance id."`
	AssignedPartitions   map[string][]int32 `yaml:"assigned_partitions" doc:"List of partitions assigned to this block builder."`
	ConsumeCycleDuration time.Duration      `yaml:"consume_cycle_duration" doc:"Interval between consumption cycles."`

	LookbackOnNoCommit time.Duration `yaml:"lookback_on_no_commit" category:"advanced"`

	blockConfig BlockConfig `yaml:"block" doc:"Configuration for the block builder."`

	// This config is dynamically injected because defined outside the ingester config.
	IngestStorageConfig ingest.Config `yaml:"-"`
}

func (c *Config) Validate() error {
	return nil
}

func (c *Config) RegisterFlags(f *flag.FlagSet) {
	c.RegisterFlagsAndApplyDefaults("", f)
}

func (c *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	hostname, err := os.Hostname()
	if err != nil {
		level.Error(log.Logger).Log("msg", "failed to get hostname", "err", err)
		os.Exit(1)
	}

	f.StringVar(&c.InstanceID, "block-builder.instance-id", hostname, "Instance id.")
	f.Var(newPartitionAssignmentVar(&c.AssignedPartitions), prefix+".assigned-partitions", "List of partitions assigned to this block builder.")
	f.DurationVar(&c.ConsumeCycleDuration, prefix+".consume-cycle-duration", 5*time.Minute, "Interval between consumption cycles.")
	// TODO - Review default
	f.DurationVar(&c.LookbackOnNoCommit, prefix+".lookback-on-no-commit", 12*time.Hour, "How much of the historical records to look back when there is no kafka commit for a partition.")

	c.blockConfig.RegisterFlagsAndApplyDefaults(prefix+".block", f)
}

type partitionAssignmentVar struct {
	p *map[string][]int32
}

func newPartitionAssignmentVar(p *map[string][]int32) *partitionAssignmentVar {
	return &partitionAssignmentVar{p}
}

func (p *partitionAssignmentVar) Set(s string) error {
	if s == "" {
		return nil
	}

	val := make(map[string][]int32)
	if err := json.Unmarshal([]byte(s), &val); err != nil {
		return err
	}
	*p.p = val

	return nil
}

func (p *partitionAssignmentVar) String() string {
	return fmt.Sprintf("%v", *p.p)
}
