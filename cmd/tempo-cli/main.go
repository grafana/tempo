package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/grafana/tempo/cmd/tempo/app"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"gopkg.in/yaml.v2"

	"github.com/alecthomas/kong"
	"github.com/grafana/tempo/tempodb/backend/azure"
	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/grafana/tempo/tempodb/backend/s3"
)

const (
	dataFilename    = "data"
	indexFilename   = "index"
	bloomFilePrefix = "bloom-"
)

type globalOptions struct {
	ConfigFile string `type:"path" short:"c" help:"Path to tempo config file"`
}

type backendOptions struct {
	Backend string `help:"backend to connect to (s3/gcs/local/azure), optional, overrides backend in config file" enum:",s3,gcs,local,azure"`
	Bucket  string `help:"bucket (or path on local backend) to scan, optional, overrides bucket in config file"`

	S3Endpoint string `name:"s3-endpoint" help:"s3 endpoint (s3.dualstack.us-east-2.amazonaws.com), optional, overrides endpoint in config file"`
	S3User     string `name:"s3-user" help:"s3 username, optional, overrides username in config file"`
	S3Pass     string `name:"s3-pass" help:"s3 password, optional, overrides password in config file"`
}

var cli struct {
	globalOptions

	List struct {
		Block             listBlockCmd             `cmd:"" help:"List information about a block"`
		Blocks            listBlocksCmd            `cmd:"" help:"List information about all blocks in a bucket"`
		CompactionSummary listCompactionSummaryCmd `cmd:"" help:"List summary of data by compaction level"`
		CacheSummary      listCacheSummaryCmd      `cmd:"" help:"List summary of bloom sizes per day per compaction level"`
		Index             listIndexCmd             `cmd:"" help:"List information about a block index"`
		Column            listColumnCmd            `cmd:"" help:"List values in a given column"`
	} `cmd:""`

	View struct {
		Index  viewIndexCmd  `cmd:"" help:"View contents of block index"`
		Schema viewSchemaCmd `cmd:"" help:"View parquet schema"`
	} `cmd:""`

	Gen struct {
		Index indexCmd `cmd:"" help:"Generate index for a block"`
		Bloom bloomCmd `cmd:"" help:"Generate bloom for a block"`
	} `cmd:""`

	Query struct {
		API struct {
			TraceID         queryTraceIDCmd         `cmd:"" help:"query Tempo by trace ID"`
			SearchTags      querySearchTagsCmd      `cmd:"" help:"query Tempo search tags"`
			SearchTagValues querySearchTagValuesCmd `cmd:"" help:"query Tempo search tag values"`
			Search          querySearchCmd          `cmd:"" help:"query Tempo search"`
		} `cmd:""`
		Blocks queryBlocksCmd `cmd:"" help:"query for a traceid directly from backend blocks"`
	} `cmd:""`

	Search struct {
		Blocks   searchBlocksCmd   `cmd:"" help:"search for a key value pair directly from backend blocks"`
		OneBlock searchOneBlockCmd `cmd:"" help:"search for a key value pair from exactly one backend block"`
	} `cmd:""`

	Parquet struct {
		Convert convertParquet `cmd:"" help:"convert from an existing file to tempodb parquet schema"`
	} `cmd:""`
}

func main() {
	ctx := kong.Parse(&cli,
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			//Compact: true,
		}),
	)
	err := ctx.Run(&cli.globalOptions)
	ctx.FatalIfErrorf(err)
}

func loadBackend(b *backendOptions, g *globalOptions) (backend.Reader, backend.Writer, backend.Compactor, error) {
	// Defaults
	cfg := app.Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})

	// Existing config
	if g.ConfigFile != "" {
		buff, err := os.ReadFile(g.ConfigFile)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to read configFile %s: %w", g.ConfigFile, err)
		}

		err = yaml.UnmarshalStrict(buff, &cfg)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to parse configFile %s: %w", g.ConfigFile, err)
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
		cfg.StorageConfig.Trace.Azure.ContainerName = b.Bucket
	}

	if b.S3Endpoint != "" {
		cfg.StorageConfig.Trace.S3.Endpoint = b.S3Endpoint
	}

	var err error
	var r backend.RawReader
	var w backend.RawWriter
	var c backend.Compactor

	switch cfg.StorageConfig.Trace.Backend {
	case "local":
		r, w, c, err = local.New(cfg.StorageConfig.Trace.Local)
	case "gcs":
		r, w, c, err = gcs.New(cfg.StorageConfig.Trace.GCS)
	case "s3":
		r, w, c, err = s3.New(cfg.StorageConfig.Trace.S3)
	case "azure":
		r, w, c, err = azure.New(cfg.StorageConfig.Trace.Azure)
	default:
		err = fmt.Errorf("unknown backend %s", cfg.StorageConfig.Trace.Backend)
	}

	if err != nil {
		return nil, nil, nil, err
	}

	return backend.NewReader(r), backend.NewWriter(w), c, nil
}
