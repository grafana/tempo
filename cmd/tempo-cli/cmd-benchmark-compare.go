package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/render/html"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/render/markdown"
)

type benchmarkCompareCmd struct {
	Results []string `arg:"" help:"results from 'benchmark run', as name=path or a path. A path is named after the settings that set it apart, or its file. The first is the baseline"`

	Metric     []string `short:"m" help:"metrics to show, as glob patterns over keys like harness.wallNs" default:"harness.wallNs,harness.cpuNs,harness.allocBytes,backend.bytes,backend.reads"`
	Case       []string `short:"k" help:"cases to show, as glob patterns over IDs like traceid/*. Defaults to every case"`
	Percentile string   `help:"percentile the summaries show" enum:"min,p25,p50,p75,p90,p99,max" default:"p99"`
	HTTP       string   `name:"http" help:"serve the comparison as web pages on this address, like :8080, instead of writing markdown. An address without a host is served on localhost only"`
}

func (cmd *benchmarkCompareCmd) Run(_ *globalOptions) error {
	if len(cmd.Results) < 2 {
		return errors.New("at least two results are needed to compare")
	}

	runs := make([]compare.Run, 0, len(cmd.Results))
	for _, arg := range cmd.Results {
		run, err := compare.LoadRun(arg)
		if err != nil {
			return err
		}
		runs = append(runs, run)
	}

	c, err := compare.New(runs)
	if err != nil {
		return err
	}
	if err := c.FilterCases(cmd.Case); err != nil {
		return err
	}
	metrics, err := c.Metrics(cmd.Metric)
	if err != nil {
		return err
	}
	stat, err := compare.ParseStat(cmd.Percentile)
	if err != nil {
		return err
	}
	if cmd.HTTP == "" {
		return markdown.Write(os.Stdout, c, metrics, stat, c.Baseline)
	}
	h, err := html.Handler(c, metrics, stat)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return serveCompare(ctx, cmd.HTTP, h, os.Stderr)
}

// serveCompare serves h on addr until ctx is done, then shuts down, letting
// requests in flight finish. As with pprof's web view, an address without a
// host is served on localhost only. The address it serves on is written to
// out, so a port of 0 can be told.
func serveCompare(ctx context.Context, addr string, h http.Handler, out io.Writer) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid --http address %q: %w", addr, err)
	}
	if host == "" {
		host = "localhost"
	}
	ln, err := net.Listen("tcp", net.JoinHostPort(host, port))
	if err != nil {
		return fmt.Errorf("listening on %s: %w", addr, err)
	}

	srv := &http.Server{Handler: h, ReadHeaderTimeout: 10 * time.Second}
	served := make(chan error, 1)
	go func() { served <- srv.Serve(ln) }()

	if tcp, ok := ln.Addr().(*net.TCPAddr); ok {
		port = strconv.Itoa(tcp.Port)
	}
	fmt.Fprintf(out, "Serving the comparison on http://%s\n", net.JoinHostPort(host, port))

	select {
	case err := <-served:
		return fmt.Errorf("serving the comparison: %w", err)
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdown); err != nil {
		return fmt.Errorf("shutting down: %w", err)
	}
	return nil
}
