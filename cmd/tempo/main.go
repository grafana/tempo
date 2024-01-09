package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"
	"time"

	"github.com/drone/envsubst"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/flagext"
	dslog "github.com/grafana/dskit/log"
	"github.com/grafana/dskit/tracing"
	ot "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
	"go.opentelemetry.io/otel"
	oc_bridge "go.opentelemetry.io/otel/bridge/opencensus"
	ot_bridge "go.opentelemetry.io/otel/bridge/opentracing"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.18.0"
	"google.golang.org/grpc/encoding"
	"gopkg.in/yaml.v2"

	"github.com/grafana/tempo/cmd/tempo/app"
	"github.com/grafana/tempo/cmd/tempo/build"
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
	prometheus.MustRegister(version.NewCollector(appName))

	// Register the gogocodec as early as possible.
	encoding.RegisterCodec(gogocodec.NewCodec())
}

func main() {
	printVersion := flag.Bool("version", false, "Print this builds version information")
	ballastMBs := flag.Int("mem-ballast-size-mbs", 0, "Size of memory ballast to allocate in MBs.")
	mutexProfileFraction := flag.Int("mutex-profile-fraction", 0, "Enable mutex profiling.")

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

	// Init tracer
	var shutdownTracer func()
	if config.UseOTelTracer {
		shutdownTracer, err = installOpenTelemetryTracer(config)
	} else {
		shutdownTracer, err = installOpenTracingTracer(config)
	}
	if err != nil {
		level.Error(log.Logger).Log("msg", "error initialising tracer", "err", err)
		os.Exit(1)
	}
	defer shutdownTracer()

	if *mutexProfileFraction > 0 {
		runtime.SetMutexProfileFraction(*mutexProfileFraction)
	}

	// Allocate a block of memory to alter GC behaviour. See https://github.com/golang/go/issues/23044
	ballast := make([]byte, *ballastMBs*1024*1024)

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

func installOpenTracingTracer(config *app.Config) (func(), error) {
	level.Info(log.Logger).Log("msg", "initialising OpenTracing tracer")

	// Setting the environment variable JAEGER_AGENT_HOST enables tracing
	trace, err := tracing.NewFromEnv(fmt.Sprintf("%s-%s", appName, config.Target))
	if err != nil {
		return nil, fmt.Errorf("error initialising tracer: %w", err)
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
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	resources, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(fmt.Sprintf("%s-%s", appName, config.Target)),
			semconv.ServiceVersionKey.String(build.Version),
		),
		resource.WithHost(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialise trace resources: %w", err)
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
	oc_bridge.InstallTraceBridge(oc_bridge.WithTracerProvider(tp))

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
