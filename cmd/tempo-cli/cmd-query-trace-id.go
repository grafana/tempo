package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/pkg/tempopb"
)

type queryTraceIDCmd struct {
	APIEndpoint string `arg:"" help:"tempo api endpoint"`
	TraceID     string `arg:"" help:"trace ID to retrieve"`

	V1            bool     `name:"v1" help:"Use v1 API /api/traces endpoint"`
	Q             string   `name:"q" help:"V2-only TraceQL spanset filter; only matching spans are returned"`
	KeepHierarchy bool     `name:"keep-hierarchy" default:"true" help:"include ancestor path to the root for each matched span (V2 only)"`
	OrgID         string   `help:"optional orgID"`
	Headers       []string `help:"extra HTTP header in key=value format" name:"header"`
}

func (cmd *queryTraceIDCmd) Run(_ *globalOptions) error {
	client := httpclient.New(cmd.APIEndpoint, cmd.OrgID)
	applyHeaders(client, cmd.Headers)
	// util.QueryTrace will only add orgID header if len(orgID) > 0

	// the v1 endpoint does not support spanset filtering
	if cmd.Q != "" && cmd.V1 {
		return fmt.Errorf("--q filtering is only supported on the v2 API, remove --v1")
	}

	// use v1 API if specified, we default to v2
	if cmd.V1 {
		trace, err := client.QueryTrace(cmd.TraceID)
		if err != nil {
			return err
		}
		return printTrace(trace)
	}

	var traceResp *tempopb.TraceByIDResponse
	var err error
	if cmd.Q != "" {
		params := map[string]string{"q": cmd.Q, "keep_hierarchy": strconv.FormatBool(cmd.KeepHierarchy)}
		traceResp, err = client.QueryTraceV2WithQueryParams(cmd.TraceID, params)
	} else {
		traceResp, err = client.QueryTraceV2(cmd.TraceID)
	}
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
