package storage

import (
	"flag"
	"time"

	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/azure"
	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/backend/s3"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/pool"
	"github.com/grafana/tempo/tempodb/wal"
)

// Config is the Tempo storage configuration
type Config struct {
	Trace tempodb.Config `yaml:"trace"`
}

// RegisterFlagsAndApplyDefaults registers the flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.Trace.BlocklistPollFallback = true
	cfg.Trace.BlocklistPollConcurrency = tempodb.DefaultBlocklistPollConcurrency
	cfg.Trace.BlocklistPollTenantIndexBuilders = tempodb.DefaultTenantIndexBuilders
	cfg.Trace.BlocklistPollTolerateConsecutiveErrors = tempodb.DefaultTolerateConsecutiveErrors
	cfg.Trace.BlocklistPollTolerateTenantFailures = tempodb.DefaultTolerateTenantFailures

	f.StringVar(&cfg.Trace.Backend, util.PrefixConfig(prefix, "trace.backend"), "", "Trace backend (s3, azure, gcs, local)")
	f.DurationVar(&cfg.Trace.BlocklistPoll, util.PrefixConfig(prefix, "trace.blocklist_poll"), tempodb.DefaultBlocklistPoll, "Period at which to run the maintenance cycle.")

	cfg.Trace.WAL = &wal.Config{}
	f.StringVar(&cfg.Trace.WAL.Filepath, util.PrefixConfig(prefix, "trace.wal.path"), "/var/tempo/wal", "Path at which store WAL blocks.")
	cfg.Trace.WAL.Encoding = backend.EncSnappy
	cfg.Trace.WAL.SearchEncoding = backend.EncNone
	cfg.Trace.WAL.IngestionSlack = 2 * time.Minute

	cfg.Trace.Search = &tempodb.SearchConfig{}
	cfg.Trace.Search.RegisterFlagsAndApplyDefaults(prefix, f)

	cfg.Trace.Block = &common.BlockConfig{}
	cfg.Trace.Block.Version = encoding.DefaultEncoding().Version()
	cfg.Trace.Block.RegisterFlagsAndApplyDefaults(prefix, f)

	cfg.Trace.Azure = &azure.Config{}
	cfg.Trace.Azure.RegisterFlagsAndApplyDefaults(util.PrefixConfig(prefix, "trace"), f)

	cfg.Trace.S3 = &s3.Config{}
	cfg.Trace.S3.RegisterFlagsAndApplyDefaults(util.PrefixConfig(prefix, "trace"), f)

	cfg.Trace.GCS = &gcs.Config{}
	cfg.Trace.GCS.RegisterFlagsAndApplyDefaults(util.PrefixConfig(prefix, "trace"), f)

	cfg.Trace.Local = &local.Config{}
	cfg.Trace.Local.RegisterFlagsAndApplyDefaults(util.PrefixConfig(prefix, "trace"), f)

	cfg.Trace.BackgroundCache = &cache.BackgroundConfig{}
	cfg.Trace.BackgroundCache.WriteBackBuffer = 10000
	cfg.Trace.BackgroundCache.WriteBackGoroutines = 10

	cfg.Trace.Pool = &pool.Config{}
	f.IntVar(&cfg.Trace.Pool.MaxWorkers, util.PrefixConfig(prefix, "trace.pool.max-workers"), 400, "Workers in the worker pool.")
	f.IntVar(&cfg.Trace.Pool.QueueDepth, util.PrefixConfig(prefix, "trace.pool.queue-depth"), 20000, "Work item queue depth.")
}
