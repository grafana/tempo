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

	f.StringVar(&cfg.Trace.Backend, util.PrefixConfig(prefix, "trace.backend"), "", "Trace backend (s3, azure, gcs, local)")
	f.DurationVar(&cfg.Trace.BlocklistPoll, util.PrefixConfig(prefix, "trace.blocklist_poll"), tempodb.DefaultBlocklistPoll, "Period at which to run the maintenance cycle.")

	cfg.Trace.WAL = &wal.Config{}
	f.StringVar(&cfg.Trace.WAL.Filepath, util.PrefixConfig(prefix, "trace.wal.path"), "/var/tempo/wal", "Path at which store WAL blocks.")
	cfg.Trace.WAL.Encoding = backend.EncSnappy
	cfg.Trace.WAL.SearchEncoding = backend.EncNone
	cfg.Trace.WAL.IngestionSlack = 2 * time.Minute

	cfg.Trace.Search = &tempodb.SearchConfig{}
	cfg.Trace.Search.ChunkSizeBytes = tempodb.DefaultSearchChunkSizeBytes
	cfg.Trace.Search.PrefetchTraceCount = tempodb.DefaultPrefetchTraceCount

	cfg.Trace.Block = &common.BlockConfig{}
	f.Float64Var(&cfg.Trace.Block.BloomFP, util.PrefixConfig(prefix, "trace.block.bloom-filter-false-positive"), .01, "Bloom Filter False Positive.")
	f.IntVar(&cfg.Trace.Block.BloomShardSizeBytes, util.PrefixConfig(prefix, "trace.block.bloom-filter-shard-size-bytes"), 100*1024, "Bloom Filter Shard Size in bytes.")
	f.IntVar(&cfg.Trace.Block.IndexDownsampleBytes, util.PrefixConfig(prefix, "trace.block.index-downsample-bytes"), 1024*1024, "Number of bytes (before compression) per index record.")
	f.IntVar(&cfg.Trace.Block.IndexPageSizeBytes, util.PrefixConfig(prefix, "trace.block.index-page-size-bytes"), 250*1024, "Number of bytes per index page.")
	cfg.Trace.Block.Encoding = backend.EncZstd
	cfg.Trace.Block.SearchEncoding = backend.EncSnappy
	cfg.Trace.Block.SearchPageSizeBytes = 1024 * 1024 // 1 MB

	cfg.Trace.Azure = &azure.Config{}
	f.StringVar(&cfg.Trace.Azure.StorageAccountName, util.PrefixConfig(prefix, "trace.azure.storage-account-name"), "", "Azure storage account name.")
	f.Var(&cfg.Trace.Azure.StorageAccountKey, util.PrefixConfig(prefix, "trace.azure.storage-account-key"), "Azure storage access key.")
	f.StringVar(&cfg.Trace.Azure.ContainerName, util.PrefixConfig(prefix, "trace.azure.container-name"), "", "Azure container name to store blocks in.")
	f.StringVar(&cfg.Trace.Azure.Endpoint, util.PrefixConfig(prefix, "trace.azure.endpoint"), "blob.core.windows.net", "Azure endpoint to push blocks to.")
	f.IntVar(&cfg.Trace.Azure.MaxBuffers, util.PrefixConfig(prefix, "trace.azure.max-buffers"), 4, "Number of simultaneous uploads.")
	cfg.Trace.Azure.BufferSize = 3 * 1024 * 1024
	cfg.Trace.Azure.HedgeRequestsUpTo = 2

	cfg.Trace.S3 = &s3.Config{}
	f.StringVar(&cfg.Trace.S3.Bucket, util.PrefixConfig(prefix, "trace.s3.bucket"), "", "s3 bucket to store blocks in.")
	f.StringVar(&cfg.Trace.S3.Endpoint, util.PrefixConfig(prefix, "trace.s3.endpoint"), "", "s3 endpoint to push blocks to.")
	f.StringVar(&cfg.Trace.S3.AccessKey, util.PrefixConfig(prefix, "trace.s3.access_key"), "", "s3 access key.")
	f.Var(&cfg.Trace.S3.SecretKey, util.PrefixConfig(prefix, "trace.s3.secret_key"), "s3 secret key.")
	cfg.Trace.S3.HedgeRequestsUpTo = 2

	cfg.Trace.GCS = &gcs.Config{}
	f.StringVar(&cfg.Trace.GCS.BucketName, util.PrefixConfig(prefix, "trace.gcs.bucket"), "", "gcs bucket to store traces in.")
	cfg.Trace.GCS.ChunkBufferSize = 10 * 1024 * 1024
	cfg.Trace.GCS.HedgeRequestsUpTo = 2

	cfg.Trace.Local = &local.Config{}
	f.StringVar(&cfg.Trace.Local.Path, util.PrefixConfig(prefix, "trace.local.path"), "", "path to store traces at.")

	cfg.Trace.BackgroundCache = &cache.BackgroundConfig{}
	cfg.Trace.BackgroundCache.WriteBackBuffer = 10000
	cfg.Trace.BackgroundCache.WriteBackGoroutines = 10

	cfg.Trace.Pool = &pool.Config{}
	f.IntVar(&cfg.Trace.Pool.MaxWorkers, util.PrefixConfig(prefix, "trace.pool.max-workers"), 50, "Workers in the worker pool.")
	f.IntVar(&cfg.Trace.Pool.QueueDepth, util.PrefixConfig(prefix, "trace.pool.queue-depth"), 10000, "Work item queue depth.")
}
