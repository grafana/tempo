package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dustin/go-humanize"

	"github.com/grafana/tempo/v3/pkg/benchmark"
)

type benchmarkRunCmd struct {
	Block   string `arg:"" help:"path to the block directory, laid out as <bucket>/<tenant-id>/<block-id>"`
	Profile string `short:"p" help:"profile of the block, from 'benchmark profile'" required:""`
	Out     string `short:"o" help:"file to write the result to, instead of stdout" default:""`

	Repeat     int `help:"passes over the query set" default:"1"`
	Warmup     int `help:"passes to run and discard first" default:"0"`
	MaxSamples int `help:"per-case latency samples to keep" default:"10000"`

	TargetBytesPerRequest string `help:"bytes per search shard, mirroring the query frontend option of the same name" default:"100MiB"`
	SearchLimit           int    `help:"traces a search returns per shard" default:"20"`

	ReadBufferSize     int    `help:"read buffer size in bytes, 0 for the default"`
	ReadBufferCount    int    `help:"number of read buffers, 0 for the default"`
	ChunkSizeBytes     uint32 `help:"chunk size in bytes, 0 for the default"`
	PrefetchTraceCount int    `help:"traces to prefetch, 0 for the default"`
}

func (cmd *benchmarkRunCmd) Run(_ *globalOptions) error {
	targetBytes, err := humanize.ParseBytes(cmd.TargetBytesPerRequest)
	if err != nil {
		return fmt.Errorf("invalid --target-bytes-per-request: %w", err)
	}

	f, err := os.Open(cmd.Profile)
	if err != nil {
		return err
	}
	defer f.Close()

	profile, err := benchmark.LoadProfile(f)
	if err != nil {
		return fmt.Errorf("reading profile %s: %w", cmd.Profile, err)
	}

	result, err := benchmark.Run(context.Background(), cmd.Block, profile, benchmark.RunOptions{
		Repeat:                cmd.Repeat,
		Warmup:                cmd.Warmup,
		MaxSamples:            cmd.MaxSamples,
		TargetBytesPerRequest: int(targetBytes),
		SearchLimit:           cmd.SearchLimit,
		ReadBufferSize:        cmd.ReadBufferSize,
		ReadBufferCount:       cmd.ReadBufferCount,
		ChunkSizeBytes:        cmd.ChunkSizeBytes,
		PrefetchTraceCount:    cmd.PrefetchTraceCount,
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
