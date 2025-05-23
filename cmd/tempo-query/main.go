package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	zaplogfmt "github.com/jsternberg/zap-logfmt"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/grafana/tempo/cmd/tempo-query/tempo"
	"github.com/grafana/tempo/cmd/tempo/build"
	storagev2 "github.com/grafana/tempo/pkg/jaegerpb/storage/v2"
)

const (
	appName = "tempo-query"
)

var logger *zap.Logger

func main() {
	config := zap.NewProductionEncoderConfig()
	logger = zap.New(zapcore.NewCore(
		zaplogfmt.NewEncoder(config),
		os.Stdout,
		zapcore.InfoLevel,
	))

	otelShutdown, err := setupOtel()
	if err != nil {
		logger.Error("failed to setup OpenTelemetry", zap.Error(err))
	}

	var configPath string
	flag.StringVar(&configPath, "config", "", "A path to the plugin's configuration file")
	flag.Parse()

	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	if configPath != "" {
		v.SetConfigFile(configPath)

		err := v.ReadInConfig()
		if err != nil {
			logger.Error("failed to parse configuration file", zap.Error(err))
		}
	}

	cfg := &tempo.Config{}
	cfg.InitFromViper(v)
	if cfg.LogLevel != "" {
		lvl, err := zapcore.ParseLevel(cfg.LogLevel)
		if err != nil {
			logger.Error("failed to parse log level", zap.Error(err))
		} else {
			logger = zap.New(zapcore.NewCore(
				zaplogfmt.NewEncoder(config),
				os.Stdout,
				lvl,
			))
		}
	}

	backend, err := tempo.New(logger, cfg)
	if err != nil {
		logger.Error("failed to init tracer backend", zap.Error(err))
	}

	grpcOpts := []grpc.ServerOption{
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	}

	if cfg.TLSServerEnabeld {
		creds, err := credentials.NewClientTLSFromFile(cfg.TLS.CertPath, cfg.TLS.ServerName)
		if err != nil {
			logger.Error("failed to load TLS credentials", zap.Error(err))
		} else {
			grpcOpts = append(grpcOpts, grpc.Creds(creds))
		}
	}

	srv := grpc.NewServer(grpcOpts...)
	reflection.Register(srv)

	storagev2.RegisterDependencyReaderServer(srv, backend)
	storagev2.RegisterTraceReaderServer(srv, backend)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	lis, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		logger.Error("failed to listen", zap.Error(err))
	}

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.Info("Received signal, shutting down...", zap.String("signal", sig.String()))
		otelShutdown()
		srv.GracefulStop()
		logger.Info("Server stopped.")
		os.Exit(0)
	}()

	logger.Info("Server starts serving", zap.String("address", cfg.Address))
	if err := srv.Serve(lis); err != nil {
		logger.Error("failed to serve", zap.Error(err))
	}
}

func setupOtel() (func(), error) {
	ctx := context.Background()
	resources, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(appName),
			semconv.ServiceVersionKey.String(build.Version),
		),
		resource.WithHost(),
		resource.WithProcess(),
		resource.WithOSDescription(),
	)
	if err != nil {
		return func() {}, fmt.Errorf("failed to initialise trace resources: %w", err)
	}
	spanExporter, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		return func() {}, fmt.Errorf("create span exporter: %w", err)
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(spanExporter),
		tracesdk.WithResource(resources),
	)
	otel.SetTracerProvider(tp)
	return func() {
		logger.Info("Shutting down OpenTelemetry...")
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error("failed to shutdown tracer provider", zap.Error(err))
		}
	}, nil
}
