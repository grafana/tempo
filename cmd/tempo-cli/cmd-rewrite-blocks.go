package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type dropTraceCmd struct {
	backendOptions

	TraceID  string `arg:"" help:"trace ID to retrieve"`
	TenantID string `arg:"" help:"tenant ID to search"`
}

func (cmd *dropTraceCmd) Run(ctx *globalOptions) error {
	r, _, _, err := loadBackend(&cmd.backendOptions, ctx)
	if err != nil {
		return err
	}

	id, err := util.HexStringToTraceID(cmd.TraceID)
	if err != nil {
		return err
	}

	blocks, err := blocksWithTraceID(context.Background(), r, cmd.TenantID, id)
	if err != nil {
		return err
	}

	// print out blocks that have the trace id
	fmt.Println("found in:")
	for _, block := range blocks {
		fmt.Printf("  %v sz: %d traces: %d\n", block.BlockID, block.Size, block.TotalObjects)
	}

	return nil
}

func blocksWithTraceID(ctx context.Context, r backend.Reader, tenantID string, traceID common.ID) ([]*backend.BlockMeta, error) {
	blockIDs, compactedBlockIDs, err := r.Blocks(context.Background(), tenantID)
	if err != nil {
		return nil, err
	}

	blockIDs = append(blockIDs, compactedBlockIDs...)

	// Load in parallel
	wg := boundedwaitgroup.New(100)
	resultsCh := make(chan *backend.BlockMeta, len(blockIDs))

	for blockNum, id := range blockIDs {
		wg.Add(1)

		go func(blockNum2 int, id2 uuid.UUID) {
			defer wg.Done()

			// search here
			meta, err := isInBlock(ctx, r, blockNum2, id2, tenantID, traceID)
			if err != nil {
				fmt.Println("Error querying block:", err)
				return
			}

			if meta != nil {
				resultsCh <- meta
			}
		}(blockNum, id)
	}

	wg.Wait()
	close(resultsCh)

	results := make([]*backend.BlockMeta, 0)
	for q := range resultsCh {
		results = append(results, q)
	}

	return results, nil
}

func isInBlock(ctx context.Context, r backend.Reader, blockNum int, id uuid.UUID, tenantID string, traceID common.ID) (*backend.BlockMeta, error) {
	fmt.Print(".")
	if blockNum%100 == 0 {
		fmt.Print(strconv.Itoa(blockNum))
	}

	meta, err := r.BlockMeta(context.Background(), id, tenantID)
	if err != nil && !errors.Is(err, backend.ErrDoesNotExist) {
		return nil, err
	}

	if errors.Is(err, backend.ErrDoesNotExist) {
		// tempo proper searches compacted blocks, b/c each querier has a different view of the backend blocks.
		// however, with a single snaphot of the backend, we can only search the noncompacted blocks.
		return nil, nil
	}

	block, err := encoding.OpenBlock(meta, r)
	if err != nil {
		return nil, err
	}

	searchOpts := common.SearchOptions{}
	tempodb.SearchConfig{}.ApplyToOptions(&searchOpts)

	// technically we could do something even more efficient here by just testing to see if the trace id is in the block w/o
	// marshalling the whole thing. todo: do that.
	trace, err := block.FindTraceByID(ctx, traceID, searchOpts)
	if err != nil {
		return nil, err
	}

	if trace == nil {
		return nil, nil
	}

	return meta, nil
}
