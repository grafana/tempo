package gcs

import (
	"flag"
	"time"

	"github.com/grafana/tempo/v2/pkg/util"
)

type Config struct {
	BucketName            string            `yaml:"bucket_name"`
	Prefix                string            `yaml:"prefix"`
	ChunkBufferSize       int               `yaml:"chunk_buffer_size"`
	Endpoint              string            `yaml:"endpoint"`
	HedgeRequestsAt       time.Duration     `yaml:"hedge_requests_at"`
	HedgeRequestsUpTo     int               `yaml:"hedge_requests_up_to"`
	Insecure              bool              `yaml:"insecure"`
	ObjectCacheControl    string            `yaml:"object_cache_control"`
	ObjectMetadata        map[string]string `yaml:"object_metadata"`
	ListBlocksConcurrency int               `yaml:"list_blocks_concurrency"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.StringVar(&cfg.BucketName, util.PrefixConfig(prefix, "gcs.bucket"), "", "gcs bucket to store traces in.")
	f.StringVar(&cfg.Prefix, util.PrefixConfig(prefix, "gcs.prefix"), "", "gcs bucket prefix to store traces in.")
	f.IntVar(&cfg.ListBlocksConcurrency, util.PrefixConfig(prefix, "gcs.list_blocks_concurrency"), 3, "number of concurrent list calls to make to backend")
	cfg.ChunkBufferSize = 10 * 1024 * 1024
	cfg.HedgeRequestsUpTo = 2
}

func (cfg *Config) PathMatches(other *Config) bool {
	// GCS bucket names are globally unique
	return cfg.BucketName == other.BucketName && cfg.Prefix == other.Prefix
}
