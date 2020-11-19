package main

import (
	"flag"
	"strings"

	"github.com/spf13/viper"

	"github.com/grafana/tempo/cmd/tempo-query/tempo"
	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc"
	"github.com/jaegertracing/jaeger/storage/dependencystore"
	"github.com/jaegertracing/jaeger/storage/spanstore"
	jaeger_config "github.com/uber/jaeger-client-go/config"
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

	logger.Error(configPath)

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

	cfg := &tempo.Config{}
	cfg.InitFromViper(v)

	backend := tempo.New(cfg)
	initJaeger("tempo-grpc-plugin")
	grpc.Serve(&plugin{backend: backend})
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

func initJaeger(service string) {
	// .FromEnv() uses standard environment variables to allow for easy configuration
	cfg, err := jaeger_config.FromEnv()
	if err != nil {
		panic(err)
	}

	cfg.InitGlobalTracer(service)
}
