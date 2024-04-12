package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"path"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
)

// jpe cli command and docs
type querySearchCmd struct {
	HostPort string `arg:"" help:"tempo host and port. scheme and path will be provided based on query type. e.g. localhost:3200"`
	TraceQL  string `arg:"" optional:"" help:"traceql query"`
	Start    string `arg:"" optional:"" help:"start time in ISO8601 format"`
	End      string `arg:"" optional:"" help:"end time in ISO8601 format"`

	OrgID      string `help:"optional orgID"`
	UseGRPC    bool   `help:"stream search results over GRPC"`
	SPSS       int    `help:"spans per spanset" default:"0"`
	Limit      int    `help:"limit number of results" default:"0"`
	PathPrefix string `help:"string to prefix all http paths with"`
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

	req := &tempopb.SearchRequest{
		Query:           cmd.TraceQL,
		Start:           uint32(start),
		End:             uint32(end),
		SpansPerSpanSet: uint32(cmd.SPSS),
		Limit:           uint32(cmd.Limit),
	}

	if cmd.UseGRPC {
		return cmd.searchGRPC(req)
	}

	return cmd.searchHTTP(req)
}

func (cmd *querySearchCmd) searchGRPC(req *tempopb.SearchRequest) error {
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

	resp, err := client.Search(ctx, req)
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

// nolint: goconst // goconst wants us to make http:// a const
func (cmd *querySearchCmd) searchHTTP(req *tempopb.SearchRequest) error {
	httpReq, err := http.NewRequest("GET", "http://"+path.Join(cmd.HostPort, cmd.PathPrefix, api.PathSearch), nil)
	if err != nil {
		return err
	}

	httpReq, err = api.BuildSearchRequest(httpReq, req)
	if err != nil {
		return err
	}

	httpReq.Header = http.Header{}
	err = user.InjectOrgIDIntoHTTPRequest(user.InjectOrgID(context.Background(), cmd.OrgID), httpReq)
	if err != nil {
		return err
	}

	// fmt.Println(httpReq)
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return err
	}

	if httpResp.StatusCode != http.StatusOK {
		return errors.New("failed to query. body: " + string(body) + " status: " + httpResp.Status)
	}

	resp := &tempopb.SearchResponse{}
	err = jsonpb.Unmarshal(bytes.NewReader(body), resp)
	if err != nil {
		panic("failed to parse resp: " + err.Error())
	}
	err = printAsJSON(resp)
	if err != nil {
		return err
	}

	return nil
}
