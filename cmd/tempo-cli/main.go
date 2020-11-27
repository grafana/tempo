package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/grafana/tempo/cmd/tempo/app"
	tempodb_backend "github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"gopkg.in/yaml.v2"

	"github.com/alecthomas/kong"
	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/grafana/tempo/tempodb/backend/s3"
)

type globalOptions struct {
}

type backendOptions struct {
	ConfigFile string `type:"path" help:"Path to tempo config file"`
	Backend    string `help:"backend to connect to (s3/gcs)"`
	Bucket     string `help:"bucket to scan"`

	S3Endpoint string `name:"s3-endpoint" help:"s3 endpoint"`
	S3User     string `name:"s3-user" help:"s3 username override"`
	S3Pass     string `name:"s3-pass" help:"s3 password override"`
}

var cli struct {
	globalOptions

	List struct {
		Block  lsBlockCmd  `cmd:"" help:"List information about a block"`
		Bucket lsBucketCmd `cmd:"" help:"List contents of bucket."`
	} `cmd:""`

	Query queryCmd `cmd:"" help:"query tempo api"`
}

func main() {
	ctx := kong.Parse(&cli,
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)
	err := ctx.Run(&cli.globalOptions)
	ctx.FatalIfErrorf(err)
}

func loadBackend(b *backendOptions) (tempodb_backend.Reader, tempodb_backend.Writer, tempodb_backend.Compactor, error) {
	// Defaults
	cfg := app.Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})

	// Existing config
	if b.ConfigFile != "" {
		buff, err := ioutil.ReadFile(b.ConfigFile)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to read configFile %s: %w", b.ConfigFile, err)
		}

		err = yaml.UnmarshalStrict(buff, &cfg)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to parse configFile %s: %w", b.ConfigFile, err)
		}
	}

	// cli overrides
	if b.Backend != "" {
		cfg.StorageConfig.Trace.Backend = b.Backend
	}

	if b.Bucket != "" {
		cfg.StorageConfig.Trace.Local.Path = b.Bucket
		cfg.StorageConfig.Trace.GCS.BucketName = b.Bucket
		cfg.StorageConfig.Trace.S3.Bucket = b.Bucket
	}

	if b.S3Endpoint != "" {
		cfg.StorageConfig.Trace.S3.Endpoint = b.S3Endpoint
	}

	var err error
	var r tempodb_backend.Reader
	var w tempodb_backend.Writer
	var c tempodb_backend.Compactor

	switch cfg.StorageConfig.Trace.Backend {
	case "local":
		r, w, c, err = local.New(cfg.StorageConfig.Trace.Local)
	case "gcs":
		r, w, c, err = gcs.New(cfg.StorageConfig.Trace.GCS)
	case "s3":
		r, w, c, err = s3.New(cfg.StorageConfig.Trace.S3)
	default:
		err = fmt.Errorf("unknown backend %s", cfg.StorageConfig.Trace.Backend)
	}

	if err != nil {
		return nil, nil, nil, err
	}

	return r, w, c, nil
}

func blockStats(meta *encoding.BlockMeta, compactedMeta *encoding.CompactedBlockMeta, windowRange time.Duration) (int, uint8, int64, time.Time, time.Time) {
	if meta != nil {
		return meta.TotalObjects, meta.CompactionLevel, meta.EndTime.Unix() / int64(windowRange/time.Second), meta.StartTime, meta.EndTime
	} else if compactedMeta != nil {
		return compactedMeta.TotalObjects, compactedMeta.CompactionLevel, compactedMeta.EndTime.Unix() / int64(windowRange/time.Second), compactedMeta.StartTime, compactedMeta.EndTime
	}

	return -1, 0, -1, time.Unix(0, 0), time.Unix(0, 0)
}
