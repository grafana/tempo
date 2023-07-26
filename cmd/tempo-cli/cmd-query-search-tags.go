package main

import (
	"github.com/grafana/tempo/pkg/httpclient"
)

type querySearchTagsCmd struct {
	APIEndpoint string `arg:"" help:"tempo api endpoint"`

	OrgID string `help:"optional orgID"`
}

func (cmd *querySearchTagsCmd) Run(_ *globalOptions) error {
	client := httpclient.New(cmd.APIEndpoint, cmd.OrgID)

	tags, err := client.SearchTags()
	if err != nil {
		return err
	}

	return printAsJSON(tags)
}
