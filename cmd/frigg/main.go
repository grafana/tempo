package main

import (
	"flag"
	"fmt"
	"os"
	"reflect"
	"runtime"

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

var (
	ballastMBs int
)

func init() {
	prometheus.MustRegister(version.NewCollector(appName))
	flag.IntVar(&ballastMBs, "mem-ballast-size-mbs", 0, "Size of memory ballast to allocate in MBs.")
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

	// Allocate a block of memory to alter GC behaviour. See https://github.com/golang/go/issues/23044
	ballast := make([]byte, ballastMBs*1024*1024)

	// Start Frigg
	t, err := app.New(config)
	if err != nil {
		level.Error(util.Logger).Log("msg", "error initialising Frigg", "err", err)
		os.Exit(1)
	}

	level.Info(util.Logger).Log("msg", "Starting Frigg", "version", version.Info())

	if err := t.Run(); err != nil {
		level.Error(util.Logger).Log("msg", "error running Frigg", "err", err)
	}

	runtime.KeepAlive(ballast)
	if err := t.Stop(); err != nil {
		level.Error(util.Logger).Log("msg", "error stopping Frigg", "err", err)
		os.Exit(1)
	}
}
