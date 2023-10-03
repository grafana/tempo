package main

import (
	"context"
	"errors"
	"io"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/pkg/tempopb"
)

type querySearchCmd struct {
	APIEndpoint string `arg:"" help:"tempo api endpoint"`
	TraceQL     string `arg:"" optional:"" help:"traceql query"`
	Start       string `arg:"" optional:"" help:"start time in ISO8601 format"`
	End         string `arg:"" optional:"" help:"end time in ISO8601 format"`

	OrgID string `help:"optional orgID"`
}

func (cmd *querySearchCmd) Run(_ *globalOptions) error {
	startDate, err := time.Parse(time.RFC3339, cmd.Start)
	if err != nil {
		return err
	}
	start := startDate.Unix()

	endDate, err := time.Parse(time.RFC3339, cmd.End)
	if err != nil {
		return err
	}
	end := endDate.Unix()

	ctx := user.InjectOrgID(context.Background(), cmd.OrgID)
	ctx, err = user.InjectIntoGRPCRequest(ctx)
	if err != nil {
		return err
	}
	clientConn, err := grpc.DialContext(ctx, cmd.APIEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	client := tempopb.NewStreamingQuerierClient(clientConn)

	resp, err := client.Search(ctx, &tempopb.SearchRequest{
		Query: cmd.TraceQL,
		Start: uint32(start),
		End:   uint32(end),
	})
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
