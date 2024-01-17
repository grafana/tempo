package localblocks

import (
	"flag"
	"time"

	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
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
	ConcurrentBlocks     uint                  `yaml:"concurrent_blocks"`
	FilterServerSpans    bool                  `yaml:"filter_server_spans"`
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
	cfg.ConcurrentBlocks = 10
}
