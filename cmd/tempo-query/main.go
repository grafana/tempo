package main

import (
	"flag"
	"io"
	"strings"

	"github.com/hashicorp/go-hclog"
	hcplugin "github.com/hashicorp/go-plugin"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc/shared"
	"github.com/jaegertracing/jaeger/storage/dependencystore"
	"github.com/jaegertracing/jaeger/storage/spanstore"
	otgrpc "github.com/opentracing-contrib/go-grpc"
	"github.com/opentracing/opentracing-go"
	"github.com/spf13/viper"
	jaeger_config "github.com/uber/jaeger-client-go/config"
	google_grpc "google.golang.org/grpc"

	"github.com/grafana/tempo/v2/cmd/tempo-query/tempo"
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
	plugin := &plugin{backend: backend}
	grpc.ServeWithGRPCServer(&shared.PluginServices{
		Store: plugin,
	}, func(options []google_grpc.ServerOption) *google_grpc.Server {
		return hcplugin.DefaultGRPCServer([]google_grpc.ServerOption{
			google_grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(opentracing.GlobalTracer())),
			google_grpc.StreamInterceptor(otgrpc.OpenTracingStreamServerInterceptor(opentracing.GlobalTracer())),
		})
	})
}

type plugin struct {
	backend *tempo.Backend
}

func (p *plugin) DependencyReader() dependencystore.Reader {
	return p.backend
}

func (p *plugin) SpanReader() spanstore.Reader {
	return p.backend
}

func (p *plugin) SpanWriter() spanstore.Writer {
	return p.backend
}

func initJaeger(service string) (io.Closer, error) {
	// .FromEnv() uses standard environment variables to allow for easy configuration
	cfg, err := jaeger_config.FromEnv()
	if err != nil {
		return nil, err
	}

	return cfg.InitGlobalTracer(service)
}
