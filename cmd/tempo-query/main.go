package main

import (
	"flag"
	"io"
	"net"
	"strings"

	"github.com/hashicorp/go-hclog"
	hcplugin "github.com/hashicorp/go-plugin"
	"github.com/jaegertracing/jaeger/proto-gen/storage_v1"
	otgrpc "github.com/opentracing-contrib/go-grpc"
	"github.com/opentracing/opentracing-go"
	"github.com/spf13/viper"
	jaeger_config "github.com/uber/jaeger-client-go/config"
	google_grpc "google.golang.org/grpc"

	"github.com/grafana/tempo/cmd/tempo-query/tempo"
)

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:       "jaeger-tempo",
		Level:      hclog.Error, // Jaeger only captures >= Warn, so don't bother logging below Warn
		JSONFormat: true,
	})

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
			logger.Error("failed to parse configuration file", "error", err)
		}
	}

	closer, err := initJaeger("tempo-grpc-plugin")
	if err != nil {
		logger.Error("failed to init tracer", "error", err)
	}
	defer closer.Close()

	cfg := &tempo.Config{}
	cfg.InitFromViper(v)

	backend, err := tempo.New(cfg)
	if err != nil {
		logger.Error("failed to init tracer backend", "error", err)
	}

	srv := hcplugin.DefaultGRPCServer([]google_grpc.ServerOption{
		google_grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(opentracing.GlobalTracer())),
		google_grpc.StreamInterceptor(otgrpc.OpenTracingStreamServerInterceptor(opentracing.GlobalTracer())),
	})

	storage_v1.RegisterSpanReaderPluginServer(srv, backend)
	storage_v1.RegisterDependenciesReaderPluginServer(srv, backend)
	storage_v1.RegisterSpanWriterPluginServer(srv, backend)

	const addr = "0.0.0.0:7777"
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("failed to listen", "error", err)
	}

	logger.Info("server is listening", "address", addr)
	if err := srv.Serve(lis); err != nil {
		logger.Error("failed to serve", "error", err)
	}
}

func initJaeger(service string) (io.Closer, error) {
	// .FromEnv() uses standard environment variables to allow for easy configuration
	cfg, err := jaeger_config.FromEnv()
	if err != nil {
		return nil, err
	}

	return cfg.InitGlobalTracer(service)
}
