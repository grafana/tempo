package main

import (
	"encoding/json"
	"fmt"

	"github.com/grafana/tempo/pkg/util"
)

type queryCmd struct {
	APIEndpoint string `arg:"" help:"tempo api endpoint"`
	TraceID     string `arg:"" help:"trace ID to retrieve"`

	OrgID string `help:"optional orgID"`
}

func (cmd *queryCmd) Run(_ *globalOptions) error {
	client := util.NewClient(cmd.APIEndpoint, cmd.OrgID, nil)

	// util.QueryTrace will only add orgID header if len(orgID) > 0
	trace, err := client.QueryTrace(cmd.TraceID)
	if err != nil {
		return err
	}

	traceJSON, err := json.Marshal(trace)
	if err != nil {
		return err
	}

	fmt.Println(string(traceJSON))
	return nil
}
