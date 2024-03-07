package main

import (
	"context"
	"errors"
	"io"
	"path"
	"time"

	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/pkg/tempopb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type querySearchTagValuesCmd struct {
	HostPort string `arg:"" help:"tempo host and port. scheme and path will be provided based on query type. e.g. localhost:3200"`
	Tag      string `arg:"" help:"tag name"`
	Start    string `arg:"" optional:"" help:"start time in ISO8601 format"`
	End      string `arg:"" optional:"" help:"end time in ISO8601 format"`

	Query      string `help:"TraceQL query to filter attribute results by (supported by GRPC only)"`
	OrgID      string `help:"optional orgID"`
	UseGRPC    bool   `help:"stream search results over GRPC"`
	PathPrefix string `help:"string to prefix all http paths with"`
}

func (cmd *querySearchTagValuesCmd) Run(_ *globalOptions) error {
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

	if cmd.UseGRPC {
		return cmd.searchGRPC(start, end)
	}
	return cmd.searchHTTP(start, end)
}

// nolint: goconst // goconst wants us to make http:// a const
func (cmd *querySearchTagValuesCmd) searchHTTP(start, end int64) error {
	if cmd.PathPrefix != "" {
		cmd.HostPort = path.Join(cmd.HostPort, cmd.PathPrefix)
	}
	client := httpclient.New("http://"+cmd.HostPort, cmd.OrgID)

	var tags *tempopb.SearchTagValuesV2Response
	var err error
	if start != 0 || end != 0 {
		tags, err = client.SearchTagValuesV2WithRange(cmd.Tag, start, end)
	} else {
		tags, err = client.SearchTagValuesV2(cmd.Tag, "")
	}

	if err != nil {
		return err
	}

	return printAsJSON(tags)
}

func (cmd *querySearchTagValuesCmd) searchGRPC(start, end int64) error {
	ctx := user.InjectOrgID(context.Background(), cmd.OrgID)
	ctx, err := user.InjectIntoGRPCRequest(ctx)
	if err != nil {
		return err
	}

	clientConn, err := grpc.DialContext(ctx, cmd.HostPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	client := tempopb.NewStreamingQuerierClient(clientConn)

	tagsRequest := &tempopb.SearchTagValuesRequest{
		TagName: cmd.Tag,
		Start:   uint32(start),
		End:     uint32(end),
		Query:   cmd.Query,
	}

	resp, err := client.SearchTagValuesV2(ctx, tagsRequest)
	if err != nil {
		return err
	}

	for {
		searchResp, err := resp.Recv()
		if searchResp != nil {
			err = printAsJSON(searchResp)
			if err != nil {
				return err
			}
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}
