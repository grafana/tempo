package storage

import (
	"flag"
	"time"

	prometheus_config "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/tsdb/agent"
)

type Config struct {
	// Path to store the WAL. Each tenant will be stored in its own subdirectory.
	Path string `yaml:"path"`

	Wal agentOptions `yaml:"wal"`

	// How long to wait when flushing sample on shutdown
	RemoteWriteFlushDeadline time.Duration `yaml:"remote_write_flush_deadline"`

	// Add X-Scope-OrgID header in remote write requests
	RemoteWriteRemoveOrgIDHeader bool `yaml:"remote_write_remove_org_id_header,omitempty"`

	// Prometheus remote write config
	// https://prometheus.io/docs/prometheus/latest/configuration/configuration/#remote_write
	RemoteWrite []prometheus_config.RemoteWriteConfig `yaml:"remote_write,omitempty"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(string, *flag.FlagSet) {
	cfg.Wal = agentDefaultOptions()

	cfg.RemoteWriteFlushDeadline = time.Minute
}

// agentOptions is a copy of agent.Options but with yaml struct tags. Refer to agent.Options for
// documentation.
type agentOptions struct {
	WALSegmentSize    int           `yaml:"wal_segment_size"`
	WALCompression    bool          `yaml:"wal_compression"`
	StripeSize        int           `yaml:"stripe_size"`
	TruncateFrequency time.Duration `yaml:"truncate_frequency"`
	MinWALTime        int64         `yaml:"min_wal_time"`
	MaxWALTime        int64         `yaml:"max_wal_time"`
	NoLockfile        bool          `yaml:"no_lockfile"`
}

func agentDefaultOptions() agentOptions {
	defaultOptions := agent.DefaultOptions()
	return agentOptions{
		WALSegmentSize:    defaultOptions.WALSegmentSize,
		WALCompression:    defaultOptions.WALCompression,
		StripeSize:        defaultOptions.StripeSize,
		TruncateFrequency: defaultOptions.TruncateFrequency,
		MinWALTime:        defaultOptions.MinWALTime,
		MaxWALTime:        defaultOptions.MaxWALTime,
		NoLockfile:        defaultOptions.NoLockfile,
	}
}

func (a *agentOptions) toPrometheusAgentOptions() *agent.Options {
	return &agent.Options{
		WALSegmentSize:    a.WALSegmentSize,
		WALCompression:    a.WALCompression,
		StripeSize:        a.StripeSize,
		TruncateFrequency: a.TruncateFrequency,
		MinWALTime:        a.MinWALTime,
		MaxWALTime:        a.MaxWALTime,
		NoLockfile:        a.NoLockfile,
	}
}
