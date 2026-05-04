package cos

import (
	"flag"
	"time"

	"github.com/grafana/dskit/flagext"

	"github.com/grafana/tempo/pkg/util"
)

type Config struct {
	Bucket           string         `yaml:"bucket"`
	Prefix           string         `yaml:"prefix"`
	Region           string         `yaml:"region"`
	AppID            string         `yaml:"app_id"`
	SecretID         string         `yaml:"secret_id"`
	SecretKey        flagext.Secret `yaml:"secret_key"`
	Endpoint         string         `yaml:"endpoint"`
	Insecure         bool           `yaml:"insecure"`
	HedgeRequestsAt  time.Duration  `yaml:"hedge_requests_at"`
	HedgeRequestsUpTo int           `yaml:"hedge_requests_up_to"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.StringVar(&cfg.Bucket, util.PrefixConfig(prefix, "cos.bucket"), "", "COS bucket to store blocks in.")
	f.StringVar(&cfg.Prefix, util.PrefixConfig(prefix, "cos.prefix"), "", "COS root directory to store blocks in.")
	f.StringVar(&cfg.Region, util.PrefixConfig(prefix, "cos.region"), "", "COS region.")
	f.StringVar(&cfg.AppID, util.PrefixConfig(prefix, "cos.app_id"), "", "COS app id.")
	f.StringVar(&cfg.SecretID, util.PrefixConfig(prefix, "cos.secret_id"), "", "COS secret id.")
	f.Var(&cfg.SecretKey, util.PrefixConfig(prefix, "cos.secret_key"), "COS secret key.")
	f.StringVar(&cfg.Endpoint, util.PrefixConfig(prefix, "cos.endpoint"), "", "COS endpoint override.")
	cfg.HedgeRequestsUpTo = 2
}

func (cfg *Config) PathMatches(other *Config) bool {
	return cfg.Bucket == other.Bucket && cfg.Prefix == other.Prefix && cfg.Region == other.Region && cfg.AppID == other.AppID
}
