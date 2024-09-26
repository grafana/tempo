package main

import (
	"fmt"
	"os"

	"github.com/grafana/tempo/pkg/httpclient"
)

type queryTraceIDCmd struct {
	APIEndpoint string `arg:"" help:"tempo api endpoint"`
	TraceID     string `arg:"" help:"trace ID to retrieve"`

	V2    bool   `name:"v2" help:"use v2 API"`
	OrgID string `help:"optional orgID"`
}

func (cmd *queryTraceIDCmd) Run(_ *globalOptions) error {
	client := httpclient.New(cmd.APIEndpoint, cmd.OrgID)
	// util.QueryTrace will only add orgID header if len(orgID) > 0

	// use v2 API if specified
	if cmd.V2 {
		traceResp, err := client.QueryTraceV2(cmd.TraceID)
		if err != nil {
			return err
		}
		// log the Message and trace field
		if traceResp.Message != "" {
			// print message and status to stderr if there is one.
			// allows users to get a clean trace on the stdout, and pipe it to a file or another commands.
			_, _ = fmt.Fprintf(os.Stderr, "status: %s , message: %s\n", traceResp.Status, traceResp.Message)
		}
		// only print the trace field
		return printAsJSON(traceResp.Trace)
	}

	trace, err := client.QueryTrace(cmd.TraceID)
	if err != nil {
		return err
	}
	return printAsJSON(trace)
}
