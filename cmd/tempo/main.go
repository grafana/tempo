package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"runtime"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/cmd/tempo/app"
	"github.com/grafana/tempo/cmd/tempo/build"
	_ "github.com/grafana/tempo/cmd/tempo/build"
	"gopkg.in/yaml.v2"

	"github.com/go-kit/kit/log/level"

	"github.com/drone/envsubst"
	ot "github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
	"github.com/weaveworks/common/logging"
	"github.com/weaveworks/common/tracing"
	oc "go.opencensus.io/trace"
	"go.opentelemetry.io/otel"
	oc_bridge "go.opentelemetry.io/otel/bridge/opencensus"
	ot_bridge "go.opentelemetry.io/otel/bridge/opentracing"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/grafana/dskit/flagext"
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
	blockID := flag.String("uuid", "", "")
	chunkSizeBytes := flag.Int("chunk", 10485760, "")
	ballastMBs := flag.Int("mem-ballast-size-mbs", 0, "Size of memory ballast to allocate in MBs.")
	slurpBuffer := flag.Int("slurp-buffer", 1000, "")
	parse := flag.Bool("parse", true, "")

	config, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed parsing config: %v\n", err)
		os.Exit(1)
	}
	if *printVersion {
		fmt.Println(version.Print(appName))
		os.Exit(0)
	}

	ballast := make([]byte, *ballastMBs*1024*1024)

	// Init the logger which will honor the log level set in config.Server
	if reflect.DeepEqual(&config.Server.LogLevel, &logging.Level{}) {
		level.Error(log.Logger).Log("msg", "invalid log level")
		os.Exit(1)
	}
	log.InitLogger(&config.Server)

	blockSlurp("jpe-test", uuid.MustParse(*blockID), "3446", uint32(*chunkSizeBytes), *slurpBuffer, *parse)
	runtime.KeepAlive(ballast)

	// blockDeslurp("jpe-test", uuid.MustParse(*blockID), "1", uint32(*chunkSizeBytes))
	os.Exit(0)
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

func installOpenTracingTracer(config *app.Config) (func(), error) {
	level.Info(log.Logger).Log("msg", "initialising OpenTracing tracer")

	// Setting the environment variable JAEGER_AGENT_HOST enables tracing
	trace, err := tracing.NewFromEnv(fmt.Sprintf("%s-%s", appName, config.Target))
	if err != nil {
		return nil, errors.Wrap(err, "error initialising tracer")
	}
	return func() {
		if err := trace.Close(); err != nil {
			level.Error(log.Logger).Log("msg", "error closing tracing", "err", err)
			os.Exit(1)
		}
	}, nil
}

func installOpenTelemetryTracer(config *app.Config) (func(), error) {
	level.Info(log.Logger).Log("msg", "initialising OpenTelemetry tracer")

	// for now, migrate OpenTracing Jaeger environment variables
	migrateJaegerEnvironmentVariables()

	exp, err := jaeger.New(jaeger.WithCollectorEndpoint())
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Jaeger exporter")
	}

	resources, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(fmt.Sprintf("%s-%s", appName, config.Target)),
			semconv.ServiceVersionKey.String(build.Version),
		),
		resource.WithHost(),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialise trace resuorces")
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(resources),
	)
	otel.SetTracerProvider(tp)

	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			level.Error(log.Logger).Log("msg", "OpenTelemetry trace provider failed to shutdown", "err", err)
			os.Exit(1)
		}
	}

	propagator := propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{})
	otel.SetTextMapPropagator(propagator)

	otel.SetErrorHandler(otelErrorHandlerFunc(func(err error) {
		level.Error(log.Logger).Log("msg", "OpenTelemetry.ErrorHandler", "err", err)
	}))

	// Install the OpenTracing bridge
	// TODO the bridge emits warnings because the Jaeger exporter does not defer context setup
	bridgeTracer, _ := ot_bridge.NewTracerPair(tp.Tracer("OpenTracing"))
	bridgeTracer.SetWarningHandler(func(msg string) {
		level.Warn(log.Logger).Log("msg", msg, "source", "BridgeTracer.OnWarningHandler")
	})
	ot.SetGlobalTracer(bridgeTracer)

	// Install the OpenCensus bridge
	oc.DefaultTracer = oc_bridge.NewTracer(tp.Tracer("OpenCensus"))

	return shutdown, nil
}

func migrateJaegerEnvironmentVariables() {
	// jaeger-tracing-go: https://github.com/jaegertracing/jaeger-client-go#environment-variables
	// opentelemetry-go: https://github.com/open-telemetry/opentelemetry-go/tree/main/exporters/jaeger#environment-variables
	jaegerToOtel := map[string]string{
		"JAEGER_AGENT_HOST": "OTEL_EXPORTER_JAEGER_AGENT_HOST",
		"JAEGER_AGENT_PORT": "OTEL_EXPORTER_JAEGER_AGENT_PORT",
		"JAEGER_ENDPOINT":   "OTEL_EXPORTER_JAEGER_ENDPOINT",
		"JAEGER_USER":       "OTEL_EXPORTER_JAEGER_USER",
		"JAEGER_PASSWORD":   "OTEL_EXPORTER_JAEGER_PASSWORD",
		"JAEGER_TAGS":       "OTEL_RESOURCE_ATTRIBUTES",
	}
	for jaegerKey, otelKey := range jaegerToOtel {
		value, jaegerOk := os.LookupEnv(jaegerKey)
		_, otelOk := os.LookupEnv(otelKey)

		if jaegerOk && !otelOk {
			level.Warn(log.Logger).Log("msg", "migrating Jaeger environment variable, consider using native OpenTelemetry variables", "jaeger", jaegerKey, "otel", otelKey)
			_ = os.Setenv(otelKey, value)
		}
	}

	if _, ok := os.LookupEnv("JAEGER_SAMPLER_TYPE"); ok {
		level.Warn(log.Logger).Log("msg", "JAEGER_SAMPLER_TYPE is not supported with the OpenTelemetry tracer, no sampling will be performed")
	}
}

type otelErrorHandlerFunc func(error)

// Handle implements otel.ErrorHandler
func (f otelErrorHandlerFunc) Handle(err error) {
	f(err)
}
