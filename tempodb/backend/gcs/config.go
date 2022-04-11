package gcs

import (
	"time"
)

type Config struct {
	BucketName        string            `yaml:"bucket_name"`
	CacheControl      string            `yaml:"cache_control"`
	ChunkBufferSize   int               `yaml:"chunk_buffer_size"`
	Endpoint          string            `yaml:"endpoint"`
	HedgeRequestsAt   time.Duration     `yaml:"hedge_requests_at"`
	HedgeRequestsUpTo int               `yaml:"hedge_requests_up_to"`
	Insecure          bool              `yaml:"insecure"`
	Metadata          map[string]string `yaml:"metadata"`
}
