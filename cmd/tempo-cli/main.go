package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/grafana/dskit/flagext"

	"github.com/alecthomas/kong"
	"gopkg.in/yaml.v2"

	"github.com/grafana/tempo/cmd/tempo/app"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/azure"
	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/grafana/tempo/tempodb/backend/local"
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
	Backend string `help:"backend to connect to (s3/gcs/local/azure), optional, overrides backend in config file" enum:",s3,gcs,local,azure" default:""`
	Bucket  string `help:"bucket (or path on local backend) to scan, optional, overrides bucket in config file"`

	S3Endpoint         string `name:"s3-endpoint" help:"s3 endpoint (s3.dualstack.us-east-2.amazonaws.com), optional, overrides endpoint in config file"`
	S3User             string `name:"s3-user" help:"s3 username, optional, overrides username in config file"`
	S3Pass             string `name:"s3-pass" help:"s3 password, optional, overrides password in config file"`
	InsecureSkipVerify bool   `name:"insecure-skip-verify" help:"skip TLS verification, only applies to S3 and GCS" default:"false"`
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

	Analyse struct {
		Block  analyseBlockCmd  `cmd:"" help:"Analyse block in a bucket"`
		Blocks analyseBlocksCmd `cmd:"" help:"Analyse blocks in a bucket"`
	} `cmd:""`

	View struct {
		Index  viewIndexCmd  `cmd:"" help:"View contents of block index"`
		Schema viewSchemaCmd `cmd:"" help:"View parquet schema"`
	} `cmd:""`

	Gen struct {
		Index     indexCmd     `cmd:"" help:"Generate index for a block"`
		Bloom     bloomCmd     `cmd:"" help:"Generate bloom for a block"`
		AttrIndex attrIndexCmd `cmd:"" help:"Generate an attribute index for a parquet block (EXPERIMENTAL)"`
	} `cmd:""`

	Query struct {
		API struct {
			TraceID         queryTraceIDCmd         `cmd:"" help:"query Tempo by trace ID"`
			SearchTags      querySearchTagsCmd      `cmd:"" help:"query Tempo search tags"`
			SearchTagValues querySearchTagValuesCmd `cmd:"" help:"query Tempo search tag values"`
			Search          querySearchCmd          `cmd:"" help:"query Tempo search"`
			Metrics         metricsQueryCmd         `cmd:"" help:"query Tempo metrics query range"`
		} `cmd:""`
		TraceID      queryBlocksCmd       `cmd:"" help:"query for a traceid directly from backend blocks"`
		TraceSummary queryTraceSummaryCmd `cmd:"" help:"query summary for a traceid directly from backend blocks"`
		Search       searchBlocksCmd      `cmd:"" help:"search for a traceid directly from backend blocks"`
	} `cmd:""`

	RewriteBlocks struct {
		DropTraces dropTracesCmd `cmd:"" help:"rewrite blocks with given trace ids redacted"`
	} `cmd:""`

	Parquet struct {
		Convert2to3 convertParquet2to3 `cmd:"" help:"convert an existing vParquet2 file to vParquet3 block"`
		Convert3to4 convertParquet3to4 `cmd:"" help:"convert an existing vParquet3 file to vParquet4 block"`
	} `cmd:""`

	Migrate struct {
		Tenant          migrateTenantCmd          `cmd:"" help:"migrate tenant between two backends"`
		OverridesConfig migrateOverridesConfigCmd `cmd:"" help:"migrate overrides config"`
	} `cmd:""`
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

	cfg.StorageConfig.Trace.S3.InsecureSkipVerify = b.InsecureSkipVerify
	cfg.StorageConfig.Trace.GCS.Insecure = b.InsecureSkipVerify

	if b.S3User != "" {
		cfg.StorageConfig.Trace.S3.AccessKey = b.S3User
	}

	if b.S3Pass != "" {
		cfg.StorageConfig.Trace.S3.SecretKey = flagext.SecretWithValue(b.S3Pass)
	}

	if b.S3Endpoint != "" {
		cfg.StorageConfig.Trace.S3.Endpoint = b.S3Endpoint
	}

	var err error
	var r backend.RawReader
	var w backend.RawWriter
	var c backend.Compactor

	switch cfg.StorageConfig.Trace.Backend {
	case backend.Local:
		r, w, c, err = local.New(cfg.StorageConfig.Trace.Local)
	case backend.GCS:
		r, w, c, err = gcs.New(cfg.StorageConfig.Trace.GCS)
	case backend.S3:
		r, w, c, err = s3.New(cfg.StorageConfig.Trace.S3)
	case backend.Azure:
		r, w, c, err = azure.New(cfg.StorageConfig.Trace.Azure)
	default:
		err = fmt.Errorf("unknown backend %s", cfg.StorageConfig.Trace.Backend)
	}

	if err != nil {
		return nil, nil, nil, err
	}

	return backend.NewReader(r), backend.NewWriter(w), c, nil
}
