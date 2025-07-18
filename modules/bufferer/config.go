package bufferer

import (
	"flag"

	"github.com/grafana/dskit/ring"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
)

type Config struct {
	LifecyclerConfig ring.LifecyclerConfig        `yaml:"lifecycler,omitempty"`
	PartitionRing    ingester.PartitionRingConfig `yaml:"partition_ring" category:"experimental"`

	// This config is dynamically injected because defined outside the ingester config.
	IngestConfig ingest.Config `yaml:"-"`

	// WAL is non-configurable and only uses defaults
	WAL wal.Config `yaml:"-"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.LifecyclerConfig.RegisterFlagsWithPrefix(prefix, f, log.Logger)
	cfg.PartitionRing.RegisterFlags(prefix, f)

	cfg.WAL.RegisterFlags(f) // WAL config has no flags, only defaults
	cfg.WAL.Version = encoding.DefaultEncoding().Version()
	f.StringVar(&cfg.WAL.Filepath, prefix+".wal.path", "/var/tempo/bufferer/traces", "Path at which store WAL blocks.")
}

func (cfg *Config) Validate() error {
	err := cfg.LifecyclerConfig.Validate()
	if err != nil {
		return err
	}

	err = cfg.WAL.Validate()
	if err != nil {
		return err
	}

	return nil
}
