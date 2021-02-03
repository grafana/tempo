package s3

import "github.com/cortexproject/cortex/pkg/util/flagext"

type Config struct {
	Bucket    string         `yaml:"bucket"`
	Endpoint  string         `yaml:"endpoint"`
	Region    string         `yaml:"region"`
	AccessKey flagext.Secret `yaml:"access_key"`
	SecretKey flagext.Secret `yaml:"secret_key"`
	Insecure  bool           `yaml:"insecure"`
	PartSize  uint64         `yaml:"part_size"`
	// SignatureV2 configures the object storage to use V2 signing instead of V4
	SignatureV2    bool `yaml:"signature_v2"`
	ForcePathStyle bool `yaml:"forcepathstyle"`
}
