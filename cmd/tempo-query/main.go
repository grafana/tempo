package main

import (
	"flag"
	"net"
	"os"
	"strings"

	"github.com/jaegertracing/jaeger/proto-gen/storage_v1"
	zaplogfmt "github.com/jsternberg/zap-logfmt"
	otgrpc "github.com/opentracing-contrib/go-grpc"
	"github.com/opentracing/opentracing-go"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	google_grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

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
		google_grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(opentracing.GlobalTracer())),
		google_grpc.StreamInterceptor(otgrpc.OpenTracingStreamServerInterceptor(opentracing.GlobalTracer())),
	}

	if cfg.TLSEnabled {
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

	lis, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		logger.Error("failed to listen", zap.Error(err))
	}

	logger.Info("Server starts serving", zap.String("address", cfg.Address))
	if err := srv.Serve(lis); err != nil {
		logger.Error("failed to serve", zap.Error(err))
	}
}
