package s3

type Config struct {
	Bucket    string `yaml:"bucket"`
	Endpoint  string `yaml:"endpoint"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Insecure  bool   `yaml:"insecure"`
	PartSize  uint64 `yaml:"part_size"`
	// SignatureV2 configures the object storage to use V2 signing instead of V4
	SignatureV2 bool `yaml:"signature_v2"`
}
