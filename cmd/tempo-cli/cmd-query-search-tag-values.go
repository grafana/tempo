package main

import (
	"github.com/grafana/tempo/pkg/httpclient"
)

type querySearchTagValuesCmd struct {
	APIEndpoint string `arg:"" help:"tempo api endpoint"`
	Tag         string `arg:"" help:"tag name"`

	OrgID string `help:"optional orgID"`
}

func (cmd *querySearchTagValuesCmd) Run(_ *globalOptions) error {
	client := httpclient.New(cmd.APIEndpoint, cmd.OrgID)

	tagValues, err := client.SearchTagValues(cmd.Tag)
	if err != nil {
		return err
	}

	return printAsJSON(tagValues)
}
