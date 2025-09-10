package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"

	"github.com/drone/envsubst"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/flagext"
	dslog "github.com/grafana/dskit/log"
	"github.com/grafana/tempo/pkg/tracing"
	"github.com/prometheus/client_golang/prometheus"
	ver "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/common/version"
	"google.golang.org/grpc/encoding"
	"gopkg.in/yaml.v2"

	"github.com/grafana/tempo/cmd/tempo/app"
	"github.com/grafana/tempo/pkg/gogocodec"
	"github.com/grafana/tempo/pkg/util/log"
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

	prometheus.MustRegister(ver.NewCollector(appName))

	// Register the gogocodec as early as possible.
	encoding.RegisterCodec(gogocodec.NewCodec())
}

func main() {
	printVersion := flag.Bool("version", false, "Print this builds version information")
	ballastMBs := flag.Int("mem-ballast-size-mbs", 0, "Size of memory ballast to allocate in MBs.")
	mutexProfileFraction := flag.Int("mutex-profile-fraction", 0, "Override default mutex profiling fraction.")
	blockProfileThreshold := flag.Int("block-profile-threshold", 0, "Override default block profiling threshold.")

	config, configVerify, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed parsing config: %v\n", err)
		os.Exit(1)
	}
	if *printVersion {
		fmt.Println(version.Print(appName))
		os.Exit(0)
	}

	// Init the logger which will honor the log level set in config.Server
	if reflect.DeepEqual(&config.Server.LogLevel, &dslog.Level{}) {
		level.Error(log.Logger).Log("msg", "invalid log level")
		os.Exit(1)
	}
	log.InitLogger(&config.Server)

	// Verifying the config's validity and log warnings now that the logger is initialized
	isValid := configIsValid(config)

	// Exit if config.verify flag is true
	if configVerify {
		if !isValid {
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Init tracer if OTEL_TRACES_EXPORTER, OTEL_EXPORTER_OTLP_ENDPOINT or OTEL_EXPORTER_OTLP_TRACES_ENDPOINT is set
	if os.Getenv("OTEL_TRACES_EXPORTER") != "" || os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" || os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") != "" {
		shutdownTracer, err := tracing.InstallOpenTelemetryTracer(appName, config.Target)
		if err != nil {
			level.Error(log.Logger).Log("msg", "error initialising tracer", "err", err)
			os.Exit(1)
		}
		defer shutdownTracer()
	}

	setMutexBlockProfiling(*mutexProfileFraction, *blockProfileThreshold)

	// Allocate a block of memory to alter GC behaviour. See https://github.com/golang/go/issues/23044
	ballast := make([]byte, *ballastMBs*1024*1024)

	// Start Tempo
	t, err := app.New(*config)
	if err != nil {
		level.Error(log.Logger).Log("msg", "error initialising Tempo", "err", err)
		os.Exit(1)
	}

	level.Info(log.Logger).Log(
		"msg", "Starting Tempo",
		"version", version.Info(),
		"target", config.Target,
	)

	if err := t.Run(); err != nil {
		level.Error(log.Logger).Log("msg", "error running Tempo", "err", err)
		os.Exit(1)
	}
	runtime.KeepAlive(ballast)
}

func configIsValid(config *app.Config) bool {
	// Warn the user for suspect configurations
	if warnings := config.CheckConfig(); len(warnings) != 0 {
		level.Warn(log.Logger).Log("msg", "-- CONFIGURATION WARNINGS --")
		for _, w := range warnings {
			output := []any{"msg", w.Message}
			if w.Explain != "" {
				output = append(output, "explain", w.Explain)
			}
			level.Warn(log.Logger).Log(output...)
		}
		return false
	}
	return true
}

func loadConfig() (*app.Config, bool, error) {
	const (
		configFileOption      = "config.file"
		configExpandEnvOption = "config.expand-env"
		configVerifyOption    = "config.verify"
	)

	var (
		configFile      string
		configExpandEnv bool
		configVerify    bool
	)

	args := os.Args[1:]
	config := &app.Config{}

	// first get the config file
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&configFile, configFileOption, "", "")
	fs.BoolVar(&configExpandEnv, configExpandEnvOption, false, "")
	fs.BoolVar(&configVerify, configVerifyOption, false, "")

	// Try to find -config.file & -config.expand-env flags. As Parsing stops on the first error, eg. unknown flag,
	// we simply try remaining parameters until we find config flag, or there are no params left.
	// (ContinueOnError just means that flag.Parse doesn't call panic or os.Exit, but it returns error, which we ignore)
	for len(args) > 0 {
		_ = fs.Parse(args)
		args = args[1:]
	}

	// load config defaults and register flags
	config.RegisterFlagsAndApplyDefaults("", flag.CommandLine)

	// overlay with config file if provided
	if configFile != "" {
		buff, err := os.ReadFile(configFile)
		if err != nil {
			return nil, false, fmt.Errorf("failed to read configFile %s: %w", configFile, err)
		}

		if configExpandEnv {
			s, err := envsubst.EvalEnv(string(buff))
			if err != nil {
				return nil, false, fmt.Errorf("failed to expand env vars from configFile %s: %w", configFile, err)
			}
			buff = []byte(s)
		}

		err = yaml.UnmarshalStrict(buff, config)
		if err != nil {
			return nil, false, fmt.Errorf("failed to parse configFile %s: %w", configFile, err)
		}

	}

	// Pass --config.expand-env flag to overrides module
	config.Overrides.ExpandEnv = configExpandEnv

	// overlay with cli
	flagext.IgnoredFlag(flag.CommandLine, configFileOption, "Configuration file to load")
	flagext.IgnoredFlag(flag.CommandLine, configExpandEnvOption, "Whether to expand environment variables in config file")
	flagext.IgnoredFlag(flag.CommandLine, configVerifyOption, "Verify configuration and exit")
	flag.Parse()

	// after loading config, let's force some values if in single binary mode
	// if we're in single binary mode we're going to force some settings b/c nothing else makes sense
	if config.Target == app.SingleBinary {
		config.Ingester.LifecyclerConfig.RingConfig.KVStore.Store = "inmemory"
		config.Ingester.LifecyclerConfig.RingConfig.ReplicationFactor = 1
		config.Ingester.LifecyclerConfig.Addr = "127.0.0.1"

		// Generator's ring
		config.Generator.Ring.KVStore.Store = "inmemory"
		config.Generator.Ring.InstanceAddr = "127.0.0.1"
	}

	return config, configVerify, nil
}

func setMutexBlockProfiling(mutexFraction int, blockThreshold int) {
	if mutexFraction > 0 {
		// The this is evaluated as 1/mutexFraction sampling, so 1 is 100%.
		runtime.SetMutexProfileFraction(mutexFraction)
	} else {
		// Why 1000 because that is what istio defaults to and that seemed reasonable to start with.
		runtime.SetMutexProfileFraction(1000)
	}
	if blockThreshold > 0 {
		runtime.SetBlockProfileRate(blockThreshold)
	} else {
		// This should have a negligible impact. This will track anything over 10_000ns, and will randomly sample shorter durations.
		runtime.SetBlockProfileRate(10_000)
	}
}
