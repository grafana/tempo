package s3

type Config struct {
	Bucket          string            `yaml:"bucket"`
	Endpoint        string            `yaml:"endpoint"`
	Region          string            `yaml:"region"`
	AccessKey       string            `yaml:"access_key"`
	SecretKey       string            `yaml:"secret_key"`
	Insecure        bool              `yaml:"insecure"`
	PartSize uint64 `yaml:"part_size"`
}
