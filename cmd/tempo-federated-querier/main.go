package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
)

// Build information - set via ldflags
var (
	Version   = "dev"
	Revision  = "unknown"
	Branch    = "unknown"
	BuildDate = "unknown"
	GoVersion = "unknown"
)

func main() {
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	logger = level.NewFilter(logger, level.AllowInfo())

	// Parse flags
	configFile := flag.String("config.file", "", "Path to the configuration file")
	printExampleConfig := flag.Bool("config.example", false, "Print example configuration and exit")
	printVersion := flag.Bool("version", false, "Print version and exit")

	var cfg Config
	cfg.RegisterFlagsAndApplyDefaults(flag.CommandLine)
	flag.Parse()

	if *printVersion {
		fmt.Printf("Tempo Federated Querier\n")
		fmt.Printf("  Version:    %s\n", Version)
		fmt.Printf("  Revision:   %s\n", Revision)
		fmt.Printf("  Branch:     %s\n", Branch)
		fmt.Printf("  Build Date: %s\n", BuildDate)
		fmt.Printf("  Go Version: %s\n", GoVersion)
		os.Exit(0)
	}

	if *printExampleConfig {
		fmt.Print(ExampleConfig())
		os.Exit(0)
	}

	// Load configuration from file
	if *configFile != "" {
		data, err := os.ReadFile(*configFile)
		if err != nil {
			level.Error(logger).Log("msg", "failed to read config file", "err", err)
			os.Exit(1)
		}

		if err := yaml.Unmarshal(data, &cfg); err != nil {
			level.Error(logger).Log("msg", "failed to parse config file", "err", err)
			os.Exit(1)
		}
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
	querier, err := NewFederatedQuerier(cfg, logger)
	if err != nil {
		level.Error(logger).Log("msg", "failed to create querier", "err", err)
		os.Exit(1)
	}

	// Create HTTP handler
	handler := NewHandler(querier, cfg, logger)

	// Setup router
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

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
