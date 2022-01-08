package gcs

import "time"

type Config struct {
	BucketName        string        `yaml:"bucket_name"`
	ChunkBufferSize   int           `yaml:"chunk_buffer_size"`
	Endpoint          string        `yaml:"endpoint"`
	Insecure          bool          `yaml:"insecure"`
	HedgeRequestsAt   time.Duration `yaml:"hedge_requests_at"`
	HedgeRequestsUpTo int           `yaml:"hedge_requests_up_to"`
}
