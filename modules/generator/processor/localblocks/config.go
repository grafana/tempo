package localblocks

import (
	"flag"
	"time"

	"github.com/grafana/tempo/v2/tempodb"
	"github.com/grafana/tempo/v2/tempodb/encoding"
	"github.com/grafana/tempo/v2/tempodb/encoding/common"
)

const (
	Name = "local-blocks"
)

type Config struct {
	Block                *common.BlockConfig   `yaml:"block"`
	Search               *tempodb.SearchConfig `yaml:"search"`
	FlushCheckPeriod     time.Duration         `yaml:"flush_check_period"`
	TraceIdlePeriod      time.Duration         `yaml:"trace_idle_period"`
	MaxBlockDuration     time.Duration         `yaml:"max_block_duration"`
	MaxBlockBytes        uint64                `yaml:"max_block_bytes"`
	CompleteBlockTimeout time.Duration         `yaml:"complete_block_timeout"`
	MaxLiveTraces        uint64                `yaml:"max_live_traces"`
	FilterServerSpans    bool                  `yaml:"filter_server_spans"`
	FlushToStorage       bool                  `yaml:"flush_to_storage"`
	Metrics              MetricsConfig         `yaml:",inline"`
}

type MetricsConfig struct {
	ConcurrentBlocks uint `yaml:"concurrent_blocks"`
	// TimeOverlapCutoff is a tuning factor that controls whether the trace-level
	// timestamp columns are used in a metrics query.  Loading these columns has a cost,
	// so in some cases it faster to skip these columns entirely, reducing I/O but
	// increasing the number of spans evalulated and thrown away. The value is a ratio
	// between 0.0 and 1.0.  If a block overlaps the time window by less than this value,
	// then we skip the columns. A value of 1.0 will always load the columns, and 0.0 never.
	TimeOverlapCutoff float64 `yaml:"time_overlap_cutoff,omitempty"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.Block = &common.BlockConfig{}
	cfg.Block.Version = encoding.DefaultEncoding().Version()
	cfg.Block.RegisterFlagsAndApplyDefaults(prefix, f)

	cfg.Search = &tempodb.SearchConfig{}
	cfg.Search.RegisterFlagsAndApplyDefaults(prefix, f)

	cfg.FlushCheckPeriod = 10 * time.Second
	cfg.TraceIdlePeriod = 10 * time.Second
	cfg.MaxBlockDuration = 1 * time.Minute
	cfg.MaxBlockBytes = 500_000_000
	cfg.CompleteBlockTimeout = time.Hour
	cfg.FilterServerSpans = true
	cfg.Metrics = MetricsConfig{
		ConcurrentBlocks:  10,
		TimeOverlapCutoff: 0.2,
	}
}
