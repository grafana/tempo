package main

import (
	"flag"
	"fmt"
	"os"
	"reflect"

	"github.com/grafana/frigg/cmd/frigg/app"
	_ "github.com/grafana/frigg/cmd/frigg/build"

	"github.com/go-kit/kit/log/level"

	"github.com/grafana/frigg/cmd/frigg/cfg"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
	"github.com/weaveworks/common/logging"
	"github.com/weaveworks/common/tracing"

	"github.com/cortexproject/cortex/pkg/util"
)

const appName = "frigg"

func init() {
	prometheus.MustRegister(version.NewCollector(appName))
}

func main() {
	printVersion := flag.Bool("version", false, "Print this builds version information")

	var config app.Config
	if err := cfg.Parse(&config); err != nil {
		fmt.Fprintf(os.Stderr, "failed parsing config: %v\n", err)
		os.Exit(1)
	}
	if *printVersion {
		fmt.Println(version.Print(appName))
		os.Exit(0)
	}

	// Init the logger which will honor the log level set in config.Server
	if reflect.DeepEqual(&config.Server.LogLevel, &logging.Level{}) {
		level.Error(util.Logger).Log("msg", "invalid log level")
		os.Exit(1)
	}
	util.InitLogger(&config.Server)

	// Setting the environment variable JAEGER_AGENT_HOST enables tracing
	trace := tracing.NewFromEnv(fmt.Sprintf("%s-%s", appName, config.Target))
	defer func() {
		if err := trace.Close(); err != nil {
			level.Error(util.Logger).Log("msg", "error closing tracing", "err", err)
			os.Exit(1)
		}
	}()

	// Start Loki
	t, err := app.New(config)
	if err != nil {
		level.Error(util.Logger).Log("msg", "error initialising loki", "err", err)
		os.Exit(1)
	}

	level.Info(util.Logger).Log("msg", "Starting Loki", "version", version.Info())

	if err := t.Run(); err != nil {
		level.Error(util.Logger).Log("msg", "error running loki", "err", err)
	}

	if err := t.Stop(); err != nil {
		level.Error(util.Logger).Log("msg", "error stopping loki", "err", err)
		os.Exit(1)
	}
}
