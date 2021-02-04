package storage

import (
	"flag"
	"time"

	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/azure"
	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/backend/s3"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/pool"
	"github.com/grafana/tempo/tempodb/wal"
)

// Config is the Tempo storage configuration
type Config struct {
	Trace tempodb.Config `yaml:"trace"`
}

const DefaultBlocklistPoll = 5 * time.Minute

// RegisterFlagsAndApplyDefaults registers the flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {

	cfg.Trace.BlocklistPollConcurrency = tempodb.DefaultBlocklistPollConcurrency

	f.StringVar(&cfg.Trace.Backend, util.PrefixConfig(prefix, "trace.backend"), "", "Trace backend (s3, azure, gcs, local)")
	f.DurationVar(&cfg.Trace.BlocklistPoll, util.PrefixConfig(prefix, "trace.maintenance-cycle"), DefaultBlocklistPoll, "Period at which to run the maintenance cycle.")

	cfg.Trace.WAL = &wal.Config{}
	f.StringVar(&cfg.Trace.WAL.Filepath, util.PrefixConfig(prefix, "trace.wal.path"), "/var/tempo/wal", "Path at which store WAL blocks.")

	cfg.Trace.Block = &encoding.BlockConfig{}
	f.Float64Var(&cfg.Trace.Block.BloomFP, util.PrefixConfig(prefix, "trace.block.bloom-filter-false-positive"), .05, "Bloom False Positive.")
	f.IntVar(&cfg.Trace.Block.IndexDownsample, util.PrefixConfig(prefix, "trace.block.index-downsample"), 100, "Number of traces per index record.")
	cfg.Trace.Block.Encoding = backend.EncLZ4_256k

	cfg.Trace.Azure = &azure.Config{}
	f.StringVar(&cfg.Trace.Azure.StorageAccountName.Value, util.PrefixConfig(prefix, "trace.azure.storage-account-name"), "", "Azure storage account name.")
	f.StringVar(&cfg.Trace.Azure.StorageAccountKey.Value, util.PrefixConfig(prefix, "trace.azure.storage-account-key"), "", "Azure storage access key.")
	f.StringVar(&cfg.Trace.Azure.ContainerName, util.PrefixConfig(prefix, "trace.azure.container-name"), "", "Azure container name to store blocks in.")
	f.StringVar(&cfg.Trace.Azure.Endpoint, util.PrefixConfig(prefix, "trace.azure.endpoint"), "blob.core.windows.net", "Azure endpoint to push blocks to.")
	f.IntVar(&cfg.Trace.Azure.MaxBuffers, util.PrefixConfig(prefix, "trace.azure.max-buffers"), 4, "Number of simultaneous uploads.")
	cfg.Trace.Azure.BufferSize = 3 * 1024 * 1024

	cfg.Trace.S3 = &s3.Config{}
	f.StringVar(&cfg.Trace.S3.Bucket, util.PrefixConfig(prefix, "trace.s3.bucket"), "", "s3 bucket to store blocks in.")
	f.StringVar(&cfg.Trace.S3.Endpoint, util.PrefixConfig(prefix, "trace.s3.endpoint"), "", "s3 endpoint to push blocks to.")
	f.StringVar(&cfg.Trace.S3.AccessKey.Value, util.PrefixConfig(prefix, "trace.s3.access_key"), "", "s3 access key.")
	f.StringVar(&cfg.Trace.S3.SecretKey.Value, util.PrefixConfig(prefix, "trace.s3.secret_key"), "", "s3 secret key.")

	cfg.Trace.GCS = &gcs.Config{}
	f.StringVar(&cfg.Trace.GCS.BucketName, util.PrefixConfig(prefix, "trace.gcs.bucket"), "", "gcs bucket to store traces in.")
	cfg.Trace.GCS.ChunkBufferSize = 10 * 1024 * 1024

	cfg.Trace.Local = &local.Config{}
	f.StringVar(&cfg.Trace.Local.Path, util.PrefixConfig(prefix, "trace.local.path"), "", "path to store traces at.")

	cfg.Trace.Pool = &pool.Config{}
	f.IntVar(&cfg.Trace.Pool.MaxWorkers, util.PrefixConfig(prefix, "trace.pool.max-workers"), 50, "Workers in the worker pool.")
	f.IntVar(&cfg.Trace.Pool.QueueDepth, util.PrefixConfig(prefix, "trace.pool.queue-depth"), 200, "Work item queue depth.")
}
