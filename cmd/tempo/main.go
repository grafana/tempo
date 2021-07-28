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

	"github.com/drone/envsubst"
	ot "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"github.com/weaveworks/common/logging"
	octrace "go.opencensus.io/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/bridge/opencensus"
	"go.opentelemetry.io/otel/bridge/opentracing"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/cortexproject/cortex/pkg/util/log"
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
	mutexProfileFraction := flag.Int("mutex-profile-fraction", 0, "Enable mutex profiling.")

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
		level.Error(log.Logger).Log("msg", "invalid log level")
		os.Exit(1)
	}
	log.InitLogger(&config.Server)

	// Configure the OpenTelemetry Jaeger exporter
	jaegerCfg, err := jaegercfg.FromEnv()
	if err != nil {
		level.Error(log.Logger).Log("msg", "could not load jaeger tracer configuration", "err", err)
		os.Exit(1)
	}
	exp, err := jaeger.New(
		jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(jaegerCfg.Reporter.CollectorEndpoint),
		),
	)
	if err != nil {
		level.Error(log.Logger).Log("msg", "failed to create Jaeger exporter", "err", err)
		os.Exit(1)
	}
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("tempo"),
			// TODO add more resource attributes
		)),
	)
	otel.SetTracerProvider(tp)

	// Install the OpenTracing bridge
	tracer := tp.Tracer("")
	bridgeTracer, wrapperTracer := opentracing.NewTracerPair(tracer)
	bridgeTracer.SetWarningHandler(func(msg string) {
		level.Warn(log.Logger).Log("msg", msg, "source", "BridgeTracer.OnWarningHandler")
	})
	ot.SetGlobalTracer(bridgeTracer)

	// Install the OpenCensus bridge
	tracer = wrapperTracer.Tracer("")
	octrace.DefaultTracer = opencensus.NewTracer(tracer)

	if *mutexProfileFraction > 0 {
		runtime.SetMutexProfileFraction(*mutexProfileFraction)
	}

	// Allocate a block of memory to alter GC behaviour. See https://github.com/golang/go/issues/23044
	ballast := make([]byte, *ballastMBs*1024*1024)

	// Warn the user for suspect configurations
	config.CheckConfig()

	// Start Tempo
	t, err := app.New(*config)
	if err != nil {
		level.Error(log.Logger).Log("msg", "error initialising Tempo", "err", err)
		os.Exit(1)
	}

	level.Info(log.Logger).Log("msg", "Starting Tempo", "version", version.Info())

	if err := t.Run(); err != nil {
		level.Error(log.Logger).Log("msg", "error running Tempo", "err", err)
		os.Exit(1)
	}
	runtime.KeepAlive(ballast)

	level.Info(log.Logger).Log("msg", "Tempo running")
}

func loadConfig() (*app.Config, error) {
	const (
		configFileOption      = "config.file"
		configExpandEnvOption = "config.expand-env"
	)

	var (
		configFile      string
		configExpandEnv bool
	)

	args := os.Args[1:]
	config := &app.Config{}

	// first get the config file
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.SetOutput(ioutil.Discard)

	fs.StringVar(&configFile, configFileOption, "", "")
	fs.BoolVar(&configExpandEnv, configExpandEnvOption, false, "")

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
		buff, err := ioutil.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read configFile %s: %w", configFile, err)
		}

		if configExpandEnv {
			s, err := envsubst.EvalEnv(string(buff))
			if err != nil {
				return nil, fmt.Errorf("failed to expand env vars from configFile %s: %w", configFile, err)
			}
			buff = []byte(s)
		}

		err = yaml.UnmarshalStrict(buff, config)
		if err != nil {
			return nil, fmt.Errorf("failed to parse configFile %s: %w", configFile, err)
		}
	}

	// overlay with cli
	flagext.IgnoredFlag(flag.CommandLine, configFileOption, "Configuration file to load")
	flagext.IgnoredFlag(flag.CommandLine, configExpandEnvOption, "Whether to expand environment variables in config file")
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
