package ingester

import (
	"flag"
	"time"

	"github.com/cortexproject/cortex/pkg/ring"
	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/grafana/tempo/modules/storage"
)

// Config for an ingester.
type Config struct {
	LifecyclerConfig ring.LifecyclerConfig `yaml:"lifecycler,omitempty"`

	ConcurrentFlushes    int           `yaml:"concurrent_flushes"`
	FlushCheckPeriod     time.Duration `yaml:"flush_check_period"`
	FlushOpTimeout       time.Duration `yaml:"flush_op_timeout"`
	MaxTraceIdle         time.Duration `yaml:"trace_idle_period"`
	MaxBlockDuration     time.Duration `yaml:"max_block_duration"`
	MaxBlockBytes        uint64        `yaml:"max_block_bytes"`
	CompleteBlockTimeout time.Duration `yaml:"complete_block_timeout"`
	OverrideRingKey      string        `yaml:"override_ring_key"`
}

// RegisterFlagsAndApplyDefaults registers the flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	// apply generic defaults and then overlay tempo default
	flagext.DefaultValues(&cfg.LifecyclerConfig)
	cfg.LifecyclerConfig.RingConfig.KVStore.Store = "memberlist"
	cfg.LifecyclerConfig.RingConfig.ReplicationFactor = 1
	cfg.LifecyclerConfig.RingConfig.HeartbeatTimeout = 5 * time.Minute

	cfg.ConcurrentFlushes = 16
	cfg.FlushCheckPeriod = 30 * time.Second
	cfg.FlushOpTimeout = 5 * time.Minute

	f.DurationVar(&cfg.MaxTraceIdle, "ingester.trace-idle-period", 30*time.Second, "Duration after which to consider a trace complete if no spans have been received")
	f.DurationVar(&cfg.MaxBlockDuration, "ingester.max-block-duration", time.Hour, "Maximum duration which the head block can be appended to before cutting it.")
	f.Uint64Var(&cfg.MaxBlockBytes, "ingester.max-block-bytes", 1024*1024*1024, "Maximum size of the head block before cutting it.")
	f.DurationVar(&cfg.CompleteBlockTimeout, "ingester.complete-block-timeout", time.Minute+storage.DefaultBlocklistPoll, "Duration to keep head blocks in the ingester after they have been cut.")
	cfg.OverrideRingKey = ring.IngesterRingKey
}
