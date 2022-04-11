package gcs

import (
	"time"

	"cloud.google.com/go/storage"
)

type Config struct {
	BucketName        string               `yaml:"bucket_name"`
	Endpoint          string               `yaml:"endpoint"`
	ChunkBufferSize   int                  `yaml:"chunk_buffer_size"`
	HedgeRequestsUpTo int                  `yaml:"hedge_requests_up_to"`
	Insecure          bool                 `yaml:"insecure"`
	HedgeRequestsAt   time.Duration        `yaml:"hedge_requests_at"`
	ObjAttrs          *storage.ObjectAttrs `yaml:"obj_attrs"`
}
