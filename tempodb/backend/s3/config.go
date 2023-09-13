package s3

import (
	"time"

	"github.com/grafana/dskit/crypto/tls"
	"github.com/grafana/dskit/flagext"
)

type Config struct {
	tls.ClientConfig `yaml:",inline"`

	Bucket            string         `yaml:"bucket"`
	Prefix            string         `yaml:"prefix"`
	Endpoint          string         `yaml:"endpoint"`
	Region            string         `yaml:"region"`
	AccessKey         string         `yaml:"access_key"`
	SecretKey         flagext.Secret `yaml:"secret_key"`
	SessionToken      flagext.Secret `yaml:"session_token"`
	Insecure          bool           `yaml:"insecure"`
	PartSize          uint64         `yaml:"part_size"`
	HedgeRequestsAt   time.Duration  `yaml:"hedge_requests_at"`
	HedgeRequestsUpTo int            `yaml:"hedge_requests_up_to"`
	// SignatureV2 configures the object storage to use V2 signing instead of V4
	SignatureV2          bool              `yaml:"signature_v2"`
	ForcePathStyle       bool              `yaml:"forcepathstyle"`
	BucketLookupType     int               `yaml:"bucket_lookup_type"`
	Tags                 map[string]string `yaml:"tags"`
	StorageClass         string            `yaml:"storage_class"`
	Metadata             map[string]string `yaml:"metadata"`
	NativeAWSAuthEnabled bool              `yaml:"native_aws_auth_enabled"`
}

func (c *Config) PathMatches(other *Config) bool {
	// S3 bucket names are globally unique
	return c.Bucket == other.Bucket && c.Prefix == other.Prefix
}
