package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const (
	layoutString = "2006-01-02T15:04:05"
	limit        = 20
)

type searchBlocksCmd struct {
	backendOptions

	Name     string `arg:"" help:"attribute name to search for"`
	Value    string `arg:"" help:"attribute value to search for"`
	Start    string `arg:"" help:"start of time range to search (YYYY-MM-DDThh:mm:ss)"`
	End      string `arg:"" help:"end of time range to search (YYYY-MM-DDThh:mm:ss)"`
	TenantID string `arg:"" help:"tenant ID to search"`
}

func (cmd *searchBlocksCmd) Run(opts *globalOptions) error {
	r, _, _, err := loadBackend(&cmd.backendOptions, opts)
	if err != nil {
		return err
	}

	startTime, err := time.Parse(layoutString, cmd.Start)
	if err != nil {
		return err
	}
	endTime, err := time.Parse(layoutString, cmd.End)
	if err != nil {
		return err
	}

	searchReq := &tempopb.SearchRequest{
		Tags:  map[string]string{cmd.Name: cmd.Value},
		Limit: limit,
	}

	ctx := context.Background()

	blockIDs, err := r.Blocks(ctx, cmd.TenantID)
	if err != nil {
		return err
	}

	fmt.Println("Total Blocks:", len(blockIDs))

	// Load in parallel
	wg := boundedwaitgroup.New(20)
	resultsCh := make(chan *backend.BlockMeta, len(blockIDs))
	for _, id := range blockIDs {
		wg.Add(1)

		go func(id2 uuid.UUID) {
			defer wg.Done()

			// search here
			meta, err := r.BlockMeta(ctx, id2, cmd.TenantID)
			if errors.Is(err, backend.ErrDoesNotExist) {
				return
			}
			if err != nil {
				fmt.Println("Error reading block meta:", err)
				return
			}
			if meta.StartTime.Unix() <= endTime.Unix() &&
				meta.EndTime.Unix() >= startTime.Unix() {
				resultsCh <- meta
			}
		}(id)
	}

	wg.Wait()
	close(resultsCh)

	blockmetas := []*backend.BlockMeta{}
	for q := range resultsCh {
		blockmetas = append(blockmetas, q)
	}

	searchOpts := common.SearchOptions{}
	tempodb.SearchConfig{}.ApplyToOptions(&searchOpts)

	fmt.Println("Blocks In Range:", len(blockmetas))
	foundids := []string{}
	for _, meta := range blockmetas {
		block, err := encoding.OpenBlock(meta, r)
		if err != nil {
			return err
		}

		resp, err := block.Search(ctx, searchReq, searchOpts)
		if err != nil {
			fmt.Println("Error searching block:", err)
			return nil
		}

		if resp != nil {
			for _, r := range resp.Traces {
				foundids = append(foundids, r.TraceID)
			}
		}

		if len(foundids) >= limit {
			break
		}
	}

	fmt.Println("Matching Traces:", len(foundids))
	for _, id := range foundids {
		fmt.Println("  ", id)
	}

	return nil
}
