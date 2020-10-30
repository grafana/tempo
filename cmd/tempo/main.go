package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"runtime"

	"github.com/grafana/tempo/cmd/tempo/app"
	_ "github.com/grafana/tempo/cmd/tempo/build"
	"gopkg.in/yaml.v2"

	"github.com/go-kit/kit/log/level"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
	"github.com/weaveworks/common/logging"
	"github.com/weaveworks/common/tracing"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/flagext"
)

const appName = "tempo"

// Version is set via build flag -ldflags -X main.Version
var (
	Version  string
	Branch   string
	Revision string
)

func init() {
	version.Version = Version
	version.Branch = Branch
	version.Revision = Revision
	prometheus.MustRegister(version.NewCollector(appName))
}

func main() {
	printVersion := flag.Bool("version", false, "Print this builds version information")
	ballastMBs := flag.Int("mem-ballast-size-mbs", 0, "Size of memory ballast to allocate in MBs.")

	config, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed parsing config: %v\n", err)
		os.Exit(1)
	}
	if *printVersion {
		fmt.Println(version.Print(appName))
		os.Exit(0)
	}

	// Init the logger which will honor the log level set in config.Server
	if reflect.DeepEqual(&config.Server.LogLevel, &logging.Level{}) {
		level.Error(util.Logger).Log("msg", "invalid log level")
		os.Exit(1)
	}
	util.InitLogger(&config.Server)

	// Setting the environment variable JAEGER_AGENT_HOST enables tracing
	trace, err := tracing.NewFromEnv(fmt.Sprintf("%s-%s", appName, config.Target))
	if err != nil {
		level.Error(util.Logger).Log("msg", "error initialising tracer", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := trace.Close(); err != nil {
			level.Error(util.Logger).Log("msg", "error closing tracing", "err", err)
			os.Exit(1)
		}
	}()

	// Allocate a block of memory to alter GC behaviour. See https://github.com/golang/go/issues/23044
	ballast := make([]byte, *ballastMBs*1024*1024)

	// Warn the user for suspect configurations
	if config.Ingester.CompleteBlockTimeout < config.StorageConfig.Trace.MaintenanceCycle {
		level.Warn(util.Logger).Log("msg", "ingester.CompleteBlockTimeout < storage.trace.maintenance-cycle",
			"explan", "You may receive 404s between the time the ingesters have flushed a trace and the querier is aware of the new block")
	}

	if config.Compactor.Compactor.BlockRetention < config.StorageConfig.Trace.MaintenanceCycle {
		level.Warn(util.Logger).Log("msg", "compactor.CompactedBlockRetention < storage.trace.maintenance-cycle",
			"explan", "Queriers and Compactors may attempt to read a block that no longer exists")
	}

	// Start Tempo
	t, err := app.New(*config)
	if err != nil {
		level.Error(util.Logger).Log("msg", "error initialising Tempo", "err", err)
		os.Exit(1)
	}

	level.Info(util.Logger).Log("msg", "Starting Tempo", "version", version.Info())

	if err := t.Run(); err != nil {
		level.Error(util.Logger).Log("msg", "error running Tempo", "err", err)
		os.Exit(1)
	}
	runtime.KeepAlive(ballast)

	level.Info(util.Logger).Log("msg", "Tempo running")
}

func loadConfig() (*app.Config, error) {
	const configFileOption = "config.file"

	var configFile string

	args := os.Args[1:]
	config := &app.Config{}

	// first get the config file
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.SetOutput(ioutil.Discard)

	fs.StringVar(&configFile, configFileOption, "", "")

	// Try to find -config.file flags. As Parsing stops on the first error, eg. unknown flag, we simply
	// try remaining parameters until we find config flag, or there are no params left.
	// (ContinueOnError just means that flag.Parse doesn't call panic or os.Exit, but it returns error, which we ignore)
	for len(args) > 0 {
		_ = fs.Parse(args)
		args = args[1:]
	}

	// load config defaults and register flags
	config.RegisterFlagsAndApplyDefaults("", flag.CommandLine)

	// overlay with config file if provided
	if configFile != "" {
		buff, err := ioutil.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read configFile %s: %w", configFile, err)
		}

		err = yaml.UnmarshalStrict(buff, config)
		if err != nil {
			return nil, fmt.Errorf("failed to parse configFile %s: %w", configFile, err)
		}
	}

	// overlay with cli
	flagext.IgnoredFlag(flag.CommandLine, configFileOption, "Configuration file to load")
	flag.Parse()

	// after loading config, let's force some values if in single binary mode
	// if we're in single binary mode we're going to force some settings b/c nothing else makes sense
	if config.Target == app.All {
		config.Ingester.LifecyclerConfig.RingConfig.KVStore.Store = "inmemory"
		config.Ingester.LifecyclerConfig.RingConfig.ReplicationFactor = 1
		config.Ingester.LifecyclerConfig.Addr = "127.0.0.1"
	}

	return config, nil
}
