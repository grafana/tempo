package main

import (
	"github.com/grafana/tempo/v2/pkg/httpclient"
)

type queryTraceIDCmd struct {
	APIEndpoint string `arg:"" help:"tempo api endpoint"`
	TraceID     string `arg:"" help:"trace ID to retrieve"`

	OrgID string `help:"optional orgID"`
}

func (cmd *queryTraceIDCmd) Run(_ *globalOptions) error {
	client := httpclient.New(cmd.APIEndpoint, cmd.OrgID)

	// util.QueryTrace will only add orgID header if len(orgID) > 0
	trace, err := client.QueryTrace(cmd.TraceID)
	if err != nil {
		return err
	}

	return printAsJSON(trace)
}
