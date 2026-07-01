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
	Format string `help:"Output format" default:"trace-patch-v0" enum:"trace-patch-v0,trace-summary-v0-aggregate,trace-summary-v0-ranked,trace-summary-v0-grouped"`
	Top    int    `help:"Maximum entries per summary top-change section or service group" default:"10"`
	Out    string `short:"o" type:"path" help:"File to write output to, instead of stdout" default:""`
	Pretty bool   `help:"Pretty-print JSON output"`
}

func (cmd *experimentalTraceDiffCmd) Run(_ *globalOptions) error {
	base, warningsA, err := readTraceJSONFileWithWarnings(cmd.TraceA, "trace-a")
	if err != nil {
		return fmt.Errorf("read trace-a: %w", err)
	}
	compare, warningsB, err := readTraceJSONFileWithWarnings(cmd.TraceB, "trace-b")
	if err != nil {
		return fmt.Errorf("read trace-b: %w", err)
	}
	warnings := warningsA
	warnings = append(warnings, warningsB...)

	result, err := buildTraceDiffOutput(base, compare, cmd.Format, cmd.Top, warnings)
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

func buildTraceDiffOutput(base, compare *tempopb.Trace, format string, topN int, warnings []tracediff.Warning) (any, error) {
	switch format {
	case tracediff.VersionTracePatchV0:
		result, err := tracediff.Diff(base, compare, tracediff.FormatTracePatchV0)
		if err != nil {
			return nil, err
		}
		result.Warnings = append(result.Warnings, warnings...)
		return result, nil
	case tracediff.VersionTraceSummaryV0Aggregate, tracediff.VersionTraceSummaryV0Ranked, tracediff.VersionTraceSummaryV0Grouped:
		result, err := tracediff.Summarize(base, compare, tracediff.SummaryOptions{Format: tracediff.SummaryFormat(format), TopN: topN})
		if err != nil {
			return nil, err
		}
		result.Warnings = append(result.Warnings, warnings...)
		return result, nil
	default:
		return nil, fmt.Errorf("%q: %w", format, tracediff.ErrUnsupportedFormat)
	}
}

func readTraceJSONFileWithWarnings(path string, label string) (*tempopb.Trace, []tracediff.Warning, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	if resp, ok := unmarshalTraceByIDResponseFull(bytes); ok {
		var warnings []tracediff.Warning
		if resp.GetStatus() == tempopb.PartialStatus_PARTIAL {
			warnings = append(warnings, partialTraceWarning(label, path, resp.GetMessage()))
		}
		return resp.Trace, warnings, nil
	}

	trace := &tempopb.Trace{}
	if err := tempopb.UnmarshalFromJSONV1(bytes, trace); err != nil {
		return nil, nil, err
	}
	return trace, nil, nil
}

// unmarshalTraceByIDResponseFull parses a v2 TraceByIDResponse wrapper, returning
// the full response so callers can inspect Status/Message. It reports false when
// the bytes are not a wrapper with a populated Trace.
func unmarshalTraceByIDResponseFull(bytes []byte) (*tempopb.TraceByIDResponse, bool) {
	resp := &tempopb.TraceByIDResponse{}
	if err := jsonpb.UnmarshalString(string(bytes), resp); err != nil {
		return nil, false
	}
	if resp.Trace == nil {
		return nil, false
	}
	return resp, true
}

// partialTraceWarning builds the warning for a partial trace. source is a trace
// file path; detail is the optional server explanation.
func partialTraceWarning(label, source, detail string) tracediff.Warning {
	message := fmt.Sprintf("%s (%s) was returned as a partial trace and may be incomplete; the diff could be inaccurate", label, source)
	if detail != "" {
		message = fmt.Sprintf("%s: %s", message, detail)
	}
	return tracediff.Warning{Code: tracediff.WarningPartialTrace, Message: message}
}
