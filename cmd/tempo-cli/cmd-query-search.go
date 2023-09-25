package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gorilla/websocket"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/pkg/tempopb"
)

type querySearchCmd struct {
	APIEndpoint string `arg:"" help:"tempo api endpoint"` // jpe - change name
	TraceQL     string `arg:"" optional:"" help:"traceql query"`
	Start       string `arg:"" optional:"" help:"start time in ISO8601 format"`
	End         string `arg:"" optional:"" help:"end time in ISO8601 format"`

	OrgID   string `help:"optional orgID"`
	UseGRPC bool   `help:"stream search results over GRPC"`
	UseWS   bool   `help:"stream search results over websocket"`
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
		Query: cmd.TraceQL,
		Start: uint32(start),
		End:   uint32(end),
	}

	if cmd.UseGRPC {
		return cmd.searchGRPC(req)
	} else if cmd.UseWS {
		return cmd.searchWS(req)
	}

	return cmd.searchHTTP(req)
}

// jpe - not working?
func (cmd *querySearchCmd) searchGRPC(req *tempopb.SearchRequest) error {
	ctx := user.InjectOrgID(context.Background(), cmd.OrgID)
	ctx, err := user.InjectIntoGRPCRequest(ctx)
	if err != nil {
		return err
	}

	clientConn, err := grpc.DialContext(ctx, cmd.APIEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
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

func (cmd *querySearchCmd) searchWS(req *tempopb.SearchRequest) error {
	httpReq, err := api.BuildSearchRequest(nil, req)
	if err != nil {
		return err
	}

	// steal http request url and replace with websocket path/scheme
	u := httpReq.URL
	u.Scheme = "ws"
	u.Host = cmd.APIEndpoint
	u.Path = api.PathWSSearch

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				panic("failed to read msg: " + err.Error())
			}
			resp := &tempopb.SearchResponse{}
			err = jsonpb.Unmarshal(bytes.NewReader(message), resp)
			if err != nil {
				panic("failed to parse resp: " + err.Error())
			}
			printAsJSON(resp)
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	for {
		select {
		case <-done:
			return nil
		case <-interrupt:
			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				return err
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return nil
		}
	}
}

func (cmd *querySearchCmd) searchHTTP(req *tempopb.SearchRequest) error {
	client := httpclient.New("http://"+cmd.APIEndpoint, cmd.OrgID)
	resp, err := client.SearchTraceQLWithRange(req.Query, int64(req.Start), int64(req.End))
	if err != nil {
		return err
	}
	err = printAsJSON(resp)
	if err != nil {
		return err
	}

	return nil
}
