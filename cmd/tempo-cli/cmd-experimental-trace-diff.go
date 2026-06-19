package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/gogo/protobuf/jsonpb"

	"github.com/grafana/tempo/pkg/model/tracediff"
	"github.com/grafana/tempo/pkg/tempopb"
)

type experimentalTraceDiffCmd struct {
	TraceA string `name:"trace-a" type:"path" required:"" help:"Baseline trace JSON file"`
	TraceB string `name:"trace-b" type:"path" required:"" help:"Comparison trace JSON file"`
	Format string `help:"Output format" default:"trace-patch-v0" enum:"trace-patch-v0"`
	Out    string `short:"o" type:"path" help:"File to write output to, instead of stdout" default:""`
	Pretty bool   `help:"Pretty-print JSON output"`
}

func (cmd *experimentalTraceDiffCmd) Run(_ *globalOptions) error {
	base, err := readTraceJSONFile(cmd.TraceA)
	if err != nil {
		return fmt.Errorf("read trace-a: %w", err)
	}
	compare, err := readTraceJSONFile(cmd.TraceB)
	if err != nil {
		return fmt.Errorf("read trace-b: %w", err)
	}

	result, err := tracediff.Diff(base, compare, tracediff.Format(cmd.Format))
	if err != nil {
		return err
	}

	out := io.Writer(os.Stdout)
	var file *os.File
	if cmd.Out != "" {
		file, err = os.Create(cmd.Out)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer file.Close()
		out = file
	}

	enc := json.NewEncoder(out)
	if cmd.Pretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(result)
}

func readTraceJSONFile(path string) (*tempopb.Trace, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if trace, ok := unmarshalTraceByIDResponse(bytes); ok {
		return trace, nil
	}

	trace := &tempopb.Trace{}
	if err := tempopb.UnmarshalFromJSONV1(bytes, trace); err != nil {
		return nil, err
	}
	return trace, nil
}

func unmarshalTraceByIDResponse(bytes []byte) (*tempopb.Trace, bool) {
	resp := &tempopb.TraceByIDResponse{}
	if err := jsonpb.UnmarshalString(string(bytes), resp); err != nil {
		return nil, false
	}
	if resp.Trace == nil {
		return nil, false
	}
	return resp.Trace, true
}
