package gcs

import (
	"time"
)

type Config struct {
	BucketName         string            `yaml:"bucket_name"`
	ChunkBufferSize    int               `yaml:"chunk_buffer_size"`
	Endpoint           string            `yaml:"endpoint"`
	HedgeRequestsAt    time.Duration     `yaml:"hedge_requests_at"`
	HedgeRequestsUpTo  int               `yaml:"hedge_requests_up_to"`
	Insecure           bool              `yaml:"insecure"`
	ObjectCacheControl string            `yaml:"object_cache_control"`
	ObjectMetadata     map[string]string `yaml:"object_metadata"`
}
