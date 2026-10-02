package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/grafana/tempo/v3/pkg/benchmark"
	"github.com/grafana/tempo/v3/pkg/benchmark/profile"
)

type benchmarkProfileCmd struct {
	Block string `arg:"" help:"path to the block directory, laid out as <bucket>/<tenant-id>/<block-id>"`

	TraceIDs   string `name:"trace-ids" help:"number of present trace IDs to sample, or 'all' to enumerate at run time; one absent ID is derived per present ID. 0 skips the scan they need" default:"10000"`
	Attributes uint32 `name:"attributes" help:"number of attributes to profile in each scope and kind of value, ranked by total bytes. 0 skips the scan they need" default:"100"`
	Out        string `short:"o" help:"file to write the profile to, instead of stdout" default:""`
}

func (cmd *benchmarkProfileCmd) Run(_ *globalOptions) error {
	numTraceIDs, err := parseTraceIDCount(cmd.TraceIDs)
	if err != nil {
		return err
	}

	ctx := context.Background()
	meta, r, err := benchmark.LoadLocalBlock(ctx, cmd.Block)
	if err != nil {
		return err
	}

	prof, err := profile.Build(ctx, meta, r, profile.Options{
		NumTraceIDs:   numTraceIDs,
		NumAttributes: int(cmd.Attributes),
	})
	if err != nil {
		return err
	}

	out := os.Stdout
	if cmd.Out != "" {
		f, err := os.Create(cmd.Out)
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
	}
	return prof.Write(out)
}

func parseTraceIDCount(s string) (int, error) {
	if strings.EqualFold(strings.TrimSpace(s), "all") {
		return profile.TraceIDsAll, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("--trace-ids must be a number or 'all': %w", err)
	}
	if n < 0 {
		return 0, errors.New("--trace-ids must not be negative")
	}
	return n, nil
}
