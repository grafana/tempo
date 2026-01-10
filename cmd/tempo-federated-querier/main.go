package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/drone/envsubst"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gorilla/mux"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/tempo/cmd/tempo-federated-querier/combiner"
	"github.com/grafana/tempo/cmd/tempo-federated-querier/handler"
	"github.com/prometheus/client_golang/prometheus"
	ver "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/common/version"
	"gopkg.in/yaml.v2"
)

const appName = "tempo-federated-querier"

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
}

func main() {
	printVersion := flag.Bool("version", false, "Print version and exit")
	printExampleConfig := flag.Bool("config.example", false, "Print example configuration and exit")

	// Handle example config before loading config
	for _, arg := range os.Args[1:] {
		if arg == "-config.example" || arg == "--config.example" {
			fmt.Print(ExampleConfig())
			os.Exit(0)
		}
	}

	// Load configuration using Tempo-style config loading
	cfg, configVerify, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed parsing config: %v\n", err)
		os.Exit(1)
	}
	if *printVersion {
		fmt.Println(version.Print(appName))
		os.Exit(0)
	}

	// These flags are registered but ignored (handled above or by LoadConfig)
	_ = printVersion
	_ = printExampleConfig

	// Initialize logger
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	logger = level.NewFilter(logger, level.AllowInfo())

	// Check config and log warnings
	configValid := true
	if warnings := cfg.CheckConfig(); len(warnings) != 0 {
		level.Warn(logger).Log("msg", "-- CONFIGURATION WARNINGS --")
		for _, w := range warnings {
			output := []any{"msg", w.Message}
			if w.Explain != "" {
				output = append(output, "explain", w.Explain)
			}
			level.Warn(logger).Log(output...)
		}
		configValid = false
	}

	// Exit if config.verify flag is true
	if configVerify {
		if err := cfg.Validate(); err != nil {
			level.Error(logger).Log("msg", "invalid configuration", "err", err)
			os.Exit(1)
		}
		if !configValid {
			os.Exit(1)
		}
		level.Info(logger).Log("msg", "configuration is valid")
		os.Exit(0)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		level.Error(logger).Log("msg", "invalid configuration", "err", err)
		os.Exit(1)
	}

	level.Info(logger).Log(
		"msg", "starting Tempo Federated Querier",
		"version", Version,
		"instances", len(cfg.Instances),
	)

	// Log configured instances
	for i, inst := range cfg.Instances {
		level.Info(logger).Log(
			"msg", "configured instance",
			"index", i,
			"name", inst.Name,
			"endpoint", inst.Endpoint,
		)
	}

	// Create federated querier
	querier, err := NewFederatedQuerier(*cfg, logger)
	if err != nil {
		level.Error(logger).Log("msg", "failed to create querier", "err", err)
		os.Exit(1)
	}

	// Create combiner
	comb := combiner.New(cfg.MaxBytesPerTrace, logger)

	// Create handler config
	handlerCfg := handler.Config{
		QueryTimeout: cfg.QueryTimeout,
		Instances:    make([]handler.InstanceInfo, len(cfg.Instances)),
	}
	for i, inst := range cfg.Instances {
		handlerCfg.Instances[i] = handler.InstanceInfo{
			Name:     inst.Name,
			Endpoint: inst.Endpoint,
		}
	}

	// Create default handler config for status diff
	defaultCfg := NewDefaultConfig()
	defaultHandlerCfg := handler.Config{
		QueryTimeout: defaultCfg.QueryTimeout,
		Instances:    []handler.InstanceInfo{},
	}

	// Create HTTP handler
	h := handler.NewHandler(querier, comb, handlerCfg, defaultHandlerCfg, logger)

	// Setup router
	router := mux.NewRouter()
	h.RegisterRoutes(router)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.HTTPListenAddress, cfg.HTTPListenPort)
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Handle graceful shutdown
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		level.Info(logger).Log("msg", "shutting down server...")
		if err := server.Close(); err != nil {
			level.Error(logger).Log("msg", "error during shutdown", "err", err)
		}
		done <- true
	}()

	// Start server
	level.Info(logger).Log("msg", "server listening", "addr", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		level.Error(logger).Log("msg", "server error", "err", err)
		os.Exit(1)
	}

	<-done
	level.Info(logger).Log("msg", "server stopped")
}

func loadConfig() (*Config, bool, error) {
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
	config := &Config{}

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

	return config, configVerify, nil
}
