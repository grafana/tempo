package s3express

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/grafana/tempo/v3/tempodb/backend/s3"
)

// Config is the configuration of the s3express backend. It takes the options of the s3 backend.
type Config struct {
	s3.Config `yaml:",inline"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, _ *flag.FlagSet) {
	// The options are the s3 backend's, which registers their flags. Apply their defaults without
	// registering them again.
	cfg.Config.RegisterFlagsAndApplyDefaults(prefix, &flag.FlagSet{})

	// directory buckets are listed with a single listing, see readerWriter.ListBlocks
	cfg.ListBlocksConcurrency = 1
}

func (cfg *Config) PathMatches(other *Config) bool {
	return cfg.Config.PathMatches(&other.Config)
}

// Validate returns an error when the backend can't run with cfg, and warnings for the options it
// ignores.
func (cfg *Config) Validate() (warnings []error, err error) {
	if !IsDirectoryBucket(cfg.Bucket) {
		return nil, fmt.Errorf("bucket %q isn't an S3 directory bucket, the s3express backend only supports buckets whose names end in --x-s3", cfg.Bucket)
	}
	if cfg.Region == "" {
		return nil, fmt.Errorf("region is required for S3 directory bucket %s", cfg.Bucket)
	}
	if cfg.Endpoint == "" && zonalEndpoint(cfg.Bucket, cfg.Region) == "" {
		return nil, fmt.Errorf("no zone ID in the name of S3 directory bucket %s, set endpoint to its Zonal endpoint", cfg.Bucket)
	}
	if cfg.Insecure {
		return nil, errors.New("insecure isn't supported: Tempo can only sign uploads to S3 directory buckets over https")
	}

	if cfg.ListBlocksConcurrency > 1 {
		warnings = append(warnings, fmt.Errorf("list_blocks_concurrency is %d, but S3 directory buckets don't list keys in order, so their blocks are listed with a single listing", cfg.ListBlocksConcurrency))
	}
	return warnings, nil
}

// IsDirectoryBucket reports whether bucket names an S3 directory bucket, in an Availability Zone
// (S3 Express One Zone) or a Local Zone, or an access point for one. These are identified by the
// same name suffixes the AWS SDKs use.
func IsDirectoryBucket(bucket string) bool {
	return strings.HasSuffix(bucket, "--x-s3") || strings.HasSuffix(bucket, "--xa-s3")
}

// zonalEndpoint returns the Zonal endpoint of a directory bucket, built from the zone ID in its
// name (<base>--<zone-id>--x-s3) like the AWS SDKs do.
func zonalEndpoint(bucket, region string) string {
	parts := strings.Split(bucket, "--")
	if len(parts) < 3 {
		return ""
	}
	return "s3express-" + parts[len(parts)-2] + "." + region + ".amazonaws.com"
}
