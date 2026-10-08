package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/grafana/tempo/v3/pkg/benchmark"
	"github.com/grafana/tempo/v3/pkg/benchmark/profile"
)

type benchmarkRunCmd struct {
	Block   string `arg:"" help:"path to the block directory, laid out as <bucket>/<tenant-id>/<block-id>"`
	Profile string `short:"p" help:"profile of the block, from 'benchmark profile'" required:""`
	Out     string `short:"o" help:"file to write the result to, instead of stdout" default:""`

	Repeat int `help:"passes over the query set" default:"1"`
	Warmup int `help:"passes to run and discard first, paying the cold-read cost outside the measurement" default:"1"`

	TargetBytesPerRequest string `help:"bytes per search shard, mirroring the query frontend option of the same name" default:"100MiB"`
	SearchLimit           int    `help:"traces a search returns per shard" default:"20"`
	MaxSeries             int    `help:"series a metrics query returns" default:"1000"`
	Exemplars             int    `help:"exemplars a metrics query collects" default:"0"`

	ReadBufferSize string `help:"read buffer size, e.g. 8MiB, 0 for the default" default:"0"`

	BackendLatency   time.Duration `help:"latency added to every backend request, simulating an object store"`
	BackendBandwidth string        `help:"per-request backend bandwidth per second, e.g. 100MiB, simulating an object store; 0 for unlimited" default:"0"`
}

func (cmd *benchmarkRunCmd) Run(_ *globalOptions) error {
	targetBytes, err := humanize.ParseBytes(cmd.TargetBytesPerRequest)
	if err != nil {
		return fmt.Errorf("invalid --target-bytes-per-request: %w", err)
	}
	readBufferSize, err := humanize.ParseBytes(cmd.ReadBufferSize)
	if err != nil {
		return fmt.Errorf("invalid --read-buffer-size: %w", err)
	}
	bandwidth, err := humanize.ParseBytes(cmd.BackendBandwidth)
	if err != nil {
		return fmt.Errorf("invalid --backend-bandwidth: %w", err)
	}

	f, err := os.Open(cmd.Profile)
	if err != nil {
		return err
	}
	defer f.Close()

	prof, err := profile.Load(f)
	if err != nil {
		return fmt.Errorf("reading profile %s: %w", cmd.Profile, err)
	}

	result, err := benchmark.Run(context.Background(), cmd.Block, prof, benchmark.RunOptions{
		Repeat:                cmd.Repeat,
		Warmup:                cmd.Warmup,
		TargetBytesPerRequest: int(targetBytes),
		SearchLimit:           cmd.SearchLimit,
		MaxSeries:             cmd.MaxSeries,
		Exemplars:             cmd.Exemplars,
		ReadBufferSize:        int(readBufferSize),
		BackendLatency:        cmd.BackendLatency,
		BackendBandwidth:      int64(bandwidth),
	})
	if err != nil {
		return err
	}

	out := os.Stdout
	if cmd.Out != "" {
		file, err := os.Create(cmd.Out)
		if err != nil {
			return err
		}
		defer file.Close()
		out = file
	}
	return result.Write(out)
}
