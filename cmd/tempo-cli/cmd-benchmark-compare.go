package main

import (
	"errors"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/render/markdown"
)

type benchmarkCompareCmd struct {
	Results []string `arg:"" help:"results from 'benchmark run', as name=path or a path. A path is named after the settings that set it apart, or its file. The first is the baseline"`

	Metric     []string `short:"m" help:"metrics to show, as glob patterns over keys like harness.wallNs" default:"harness.wallNs,harness.cpuNs,harness.allocBytes,backend.bytes,backend.reads"`
	Case       []string `short:"k" help:"cases to show, as glob patterns over IDs like traceid/*. Defaults to every case"`
	Percentile string   `help:"percentile the summaries show" enum:"p50,p90,p99" default:"p50"`
	Format     string   `help:"'interactive' opens a view to explore the results in a terminal; 'markdown' writes the summaries for a pull request" enum:"interactive,markdown" default:"interactive"`
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

	if cmd.Format == "markdown" {
		return markdown.Write(os.Stdout, c, metrics, stat, c.Baseline)
	}
	if !isTerminal(os.Stdout) {
		return errors.New("the interactive view needs a terminal, use --format=markdown to write the comparison out")
	}
	if _, err := tea.NewProgram(newBenchmarkCompareModel(c, metrics, stat)).Run(); err != nil {
		return fmt.Errorf("running interactive view: %w", err)
	}
	return nil
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
