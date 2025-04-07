package main

import (
	"flag"
	"net"
	"os"
	"strings"

	"github.com/jaegertracing/jaeger/proto-gen/storage_v1"
	zaplogfmt "github.com/jsternberg/zap-logfmt"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	google_grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/grafana/tempo/cmd/tempo-query/tempo"
)

func main() {
	config := zap.NewProductionEncoderConfig()
	logger := zap.New(zapcore.NewCore(
		zaplogfmt.NewEncoder(config),
		os.Stdout,
		zapcore.InfoLevel,
	))

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

	backend, err := tempo.New(logger, cfg)
	if err != nil {
		logger.Error("failed to init tracer backend", zap.Error(err))
	}

	grpcOpts := []google_grpc.ServerOption{
		google_grpc.StatsHandler(otelgrpc.NewServerHandler()),
	}

	if cfg.TLSServerEnabeld {
		creds, err := credentials.NewClientTLSFromFile(cfg.TLS.CertPath, cfg.TLS.ServerName)
		if err != nil {
			logger.Error("failed to load TLS credentials", zap.Error(err))
		} else {
			grpcOpts = append(grpcOpts, google_grpc.Creds(creds))
		}
	}

	srv := google_grpc.NewServer(grpcOpts...)

	storage_v1.RegisterSpanReaderPluginServer(srv, backend)
	storage_v1.RegisterDependenciesReaderPluginServer(srv, backend)
	storage_v1.RegisterSpanWriterPluginServer(srv, backend)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	lis, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		logger.Error("failed to listen", zap.Error(err))
	}

	logger.Info("Server starts serving", zap.String("address", cfg.Address))
	if err := srv.Serve(lis); err != nil {
		logger.Error("failed to serve", zap.Error(err))
	}
}
