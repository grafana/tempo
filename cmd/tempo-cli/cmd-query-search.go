package main

import (
	"github.com/grafana/tempo/pkg/util"
)

type querySearchCmd struct {
	APIEndpoint string `arg:"" help:"tempo api endpoint"`
	Tags        string `arg:"" optional:"" help:"tags in logfmt format"`

	OrgID string `help:"optional orgID"`
}

func (cmd *querySearchCmd) Run(_ *globalOptions) error {
	client := util.NewClient(cmd.APIEndpoint, cmd.OrgID)

	tagValues, err := client.Search(cmd.Tags)
	if err != nil {
		return err
	}

	return printAsJson(tagValues)
}
