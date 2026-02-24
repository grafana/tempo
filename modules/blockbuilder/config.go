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
	"github.com/grafana/tempo/tempodb/wal"
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

	c.BlockCfg.RegisterFlagsAndApplyDefaults(prefix, f)
}

type Config struct {
	InstanceID            string             `yaml:"instance_id" doc:"Instance id."`
	AssignedPartitionsMap map[string][]int32 `yaml:"assigned_partitions" doc:"List of partitions assigned to this block builder."`
	PartitionsPerInstance int                `yaml:"partitions_per_instance" doc:"Number of partitions assigned to this block builder."`
	ConsumeCycleDuration  time.Duration      `yaml:"consume_cycle_duration" doc:"Interval between consumption cycles."`
	MaxBytesPerCycle      uint64             `yaml:"max_consuming_bytes" doc:"Maximum number of bytes that can be consumed in a single cycle.  0 to disable"`

	BlockConfig BlockConfig `yaml:"block" doc:"Configuration for the block builder."`
	WAL         wal.Config  `yaml:"wal" doc:"Configuration for the write ahead log."`

	// GlobalBlockConfig is the main storage trace block config (storage.trace.block). Used as fallback
	// when block.version is not set. This config is injected by the application when creating the BlockBuilder.
	GlobalBlockConfig *common.BlockConfig `yaml:"-"`

	// This config is dynamically injected because defined outside the ingester config.
	IngestStorageConfig ingest.Config `yaml:"-"`
}

func (c *Config) AssignedPartitions() []int32 {
	if len(c.AssignedPartitionsMap) > 0 {
		return c.AssignedPartitionsMap[c.InstanceID]
	}

	id, err := ingest.IngesterPartitionID(c.InstanceID)
	if err != nil {
		return c.AssignedPartitionsMap[c.InstanceID]
	}

	assignedPartitions := make([]int32, 0, c.PartitionsPerInstance)
	for i := 0; i < c.PartitionsPerInstance; i++ {
		assignedPartitions = append(assignedPartitions, id*int32(c.PartitionsPerInstance)+int32(i))
	}
	return assignedPartitions
}

func (c *Config) Validate() error {
	if _, err := coalesceBlockVersion(c); err != nil {
		return err
	}

	if err := common.ValidateConfig(&c.BlockConfig.BlockCfg); err != nil {
		return fmt.Errorf("block config validation failed: %w", err)
	}

	if err := c.WAL.Validate(); err != nil {
		return fmt.Errorf("wal config validation failed: %w", err)
	}

	if len(c.AssignedPartitionsMap) == 0 && c.PartitionsPerInstance <= 0 {
		return fmt.Errorf("at least one of AssignedPartitionsMap or PartitionsPerInstance must be set")
	}

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
	f.Var(newPartitionAssignmentVar(&c.AssignedPartitionsMap), prefix+".assigned-partitions", "List of partitions assigned to this block builder.")
	f.IntVar(&c.PartitionsPerInstance, prefix+".partitions-per-instance", 0, "Number of partitions assigned to this block builder.")
	f.DurationVar(&c.ConsumeCycleDuration, prefix+".consume-cycle-duration", 5*time.Minute, "Interval between consumption cycles.")
	f.Uint64Var(&c.MaxBytesPerCycle, prefix+".max-bytes-per-cycle", 5e9, "Maximum number of bytes that can be consumed in a single cycle. 0 to disable") // 5 Gb

	c.BlockConfig.RegisterFlagsAndApplyDefaults(prefix+".block", f)
	c.WAL.RegisterFlags(f)
	f.StringVar(&c.WAL.Filepath, prefix+".wal.path", "/var/tempo/block-builder/traces", "Path at which store WAL blocks.")
}

// coalesceBlockVersion resolves the block encoding version using the shared
// encoding.CoalesceVersion helper. Priority: default < storage.trace.block < block_builder.block.
// The WAL version always follows the resolved block version.
func coalesceBlockVersion(cfg *Config) (encoding.VersionedEncoding, error) {
	globalVer := ""
	if cfg.GlobalBlockConfig != nil {
		globalVer = cfg.GlobalBlockConfig.Version
	}

	enc, err := encoding.CoalesceVersion(globalVer, cfg.BlockConfig.BlockCfg.Version)
	if err != nil {
		return nil, fmt.Errorf("block version validation failed: %w", err)
	}

	cfg.BlockConfig.BlockCfg.Version = enc.Version()
	cfg.WAL.Version = enc.Version()

	return enc, nil
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
	if p.p == nil {
		return "map[]"
	}
	return fmt.Sprintf("%v", *p.p)
}
