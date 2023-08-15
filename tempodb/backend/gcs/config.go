package gcs

import (
	"flag"
	"time"

	"github.com/grafana/tempo/pkg/util"
)

type Config struct {
	BucketName         string            `yaml:"bucket_name"`
	Prefix             string            `yaml:"prefix"`
	ChunkBufferSize    int               `yaml:"chunk_buffer_size"`
	Endpoint           string            `yaml:"endpoint"`
	HedgeRequestsAt    time.Duration     `yaml:"hedge_requests_at"`
	HedgeRequestsUpTo  int               `yaml:"hedge_requests_up_to"`
	Insecure           bool              `yaml:"insecure"`
	ObjectCacheControl string            `yaml:"object_cache_control"`
	ObjectMetadata     map[string]string `yaml:"object_metadata"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.StringVar(&cfg.BucketName, util.PrefixConfig(prefix, "gcs.bucket"), "", "gcs bucket to store traces in.")
	f.StringVar(&cfg.Prefix, util.PrefixConfig(prefix, "gcs.prefix"), "", "gcs bucket prefix to store traces in.")
	cfg.ChunkBufferSize = 10 * 1024 * 1024
	cfg.HedgeRequestsUpTo = 2
}

func (c *Config) PathMatches(other *Config) bool {
	// GCS bucket names are globally unique
	return c.BucketName == other.BucketName && c.Prefix == other.Prefix
}
