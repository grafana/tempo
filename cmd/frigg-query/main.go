package main

import (
	"flag"
	"path"
	"strings"

	"github.com/spf13/viper"

	"github.com/grafana/frigg/cmd/frigg-query/frigg"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc"
	"github.com/jaegertracing/jaeger/storage/dependencystore"
	"github.com/jaegertracing/jaeger/storage/spanstore"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "A path to the plugin's configuration file")
	flag.Parse()

	if configPath != "" {
		viper.SetConfigFile(path.Base(configPath))
		viper.AddConfigPath(path.Dir(configPath))
	}

	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	cfg := &frigg.Config{}
	cfg.InitFromViper(v)

	backend := frigg.New(cfg)
	grpc.Serve(&plugin{backend: backend})
}

type plugin struct {
	backend *frigg.Backend
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
