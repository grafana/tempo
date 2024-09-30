package main

import (
	"fmt"
	"os"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/pkg/tempopb"
)

type queryTraceIDCmd struct {
	APIEndpoint string `arg:"" help:"tempo api endpoint"`
	TraceID     string `arg:"" help:"trace ID to retrieve"`

	V1    bool   `name:"v1" help:"Use v1 API /api/traces endpoint"`
	OrgID string `help:"optional orgID"`
}

func (cmd *queryTraceIDCmd) Run(_ *globalOptions) error {
	client := httpclient.New(cmd.APIEndpoint, cmd.OrgID)
	// util.QueryTrace will only add orgID header if len(orgID) > 0

	// use v1 API if specified, we default to v2
	if cmd.V1 {
		trace, err := client.QueryTrace(cmd.TraceID)
		if err != nil {
			return err
		}
		return printTrace(trace)
	}

	traceResp, err := client.QueryTraceV2(cmd.TraceID)
	if err != nil {
		return err
	}
	if traceResp.Message != "" {
		// print message and status to stderr if there is one.
		// allows users to get a clean trace on the stdout, and pipe it to a file or another commands.
		_, _ = fmt.Fprintf(os.Stderr, "status: %s , message: %s\n", traceResp.Status, traceResp.Message)
	}
	return printTrace(traceResp.Trace)
}

func printTrace(trace *tempopb.Trace) error {
	// tracebyid endpoints are protobuf, we are using 'gogo/protobuf/jsonpb' to marshal the
	// trace to json because 'encoding/json' package can't handle +Inf, -Inf, NaN
	marshaller := &jsonpb.Marshaler{}
	err := marshaller.Marshal(os.Stdout, trace)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to marshal trace: %v\n", err)
	}
	return nil
}
