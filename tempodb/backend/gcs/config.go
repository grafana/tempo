package gcs

type Config struct {
	BucketName      string `yaml:"bucket_name"`
	ChunkBufferSize int    `yaml:"chunk_buffer_size"`
	Endpoint        string `yaml:"endpoint"`
	Insecure        bool   `yaml:"insecure"`
	ProjectID       string `yaml:"project_id"`
}
