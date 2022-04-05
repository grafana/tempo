package s3

import (
	"time"

	"github.com/grafana/dskit/flagext"
)

type Config struct {
	Bucket            string         `yaml:"bucket"`
	Endpoint          string         `yaml:"endpoint"`
	Region            string         `yaml:"region"`
	AccessKey         string         `yaml:"access_key"`
	SecretKey         flagext.Secret `yaml:"secret_key"`
	Insecure          bool           `yaml:"insecure"`
	PartSize          uint64         `yaml:"part_size"`
	HedgeRequestsAt   time.Duration  `yaml:"hedge_requests_at"`
	HedgeRequestsUpTo int            `yaml:"hedge_requests_up_to"`
	// SignatureV2 configures the object storage to use V2 signing instead of V4
	SignatureV2    bool `yaml:"signature_v2"`
	ForcePathStyle bool `yaml:"forcepathstyle"`
}
