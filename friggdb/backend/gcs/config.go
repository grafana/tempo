package gcs

type Config struct {
	BucketName      string `yaml:"bucket_name"`
	ChunkBufferSize int    `yaml:"chunk_buffer_size"`
}
