package main

import (
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
)

type querySearchCmd struct {
	APIEndpoint string `arg:"" help:"tempo api endpoint"`
	Tags        string `arg:"" optional:"" help:"tags in logfmt format"`
	Start       string `arg:"" optional:"" help:"start time in ISO8601 format"`
	End         string `arg:"" optional:"" help:"end time in ISO8601 format"`

	OrgID string `help:"optional orgID"`
}

func (cmd *querySearchCmd) Run(_ *globalOptions) error {
	client := util.NewClient(cmd.APIEndpoint, cmd.OrgID)

	var start, end int64

	if cmd.Start != "" {
		startDate, err := time.Parse(time.RFC3339, cmd.Start)
		if err != nil {
			return err
		}
		start = startDate.Unix()
	}

	if cmd.End != "" {
		endDate, err := time.Parse(time.RFC3339, cmd.End)
		if err != nil {
			return err
		}
		end = endDate.Unix()
	}

	var tagValues *tempopb.SearchResponse
	var err error
	if start == 0 && end == 0 {
		tagValues, err = client.Search(cmd.Tags)
	} else {
		tagValues, err = client.SearchWithRange(cmd.Tags, start, end)
	}

	if err != nil {
		return err
	}

	return printAsJSON(tagValues)
}
