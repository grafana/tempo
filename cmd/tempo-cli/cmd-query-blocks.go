package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/google/uuid"

	"github.com/grafana/tempo/v2/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/v2/pkg/model/trace"
	"github.com/grafana/tempo/v2/pkg/tempopb"
	"github.com/grafana/tempo/v2/pkg/util"
	"github.com/grafana/tempo/v2/tempodb"
	"github.com/grafana/tempo/v2/tempodb/backend"
	"github.com/grafana/tempo/v2/tempodb/encoding"
	"github.com/grafana/tempo/v2/tempodb/encoding/common"
)

type queryResults struct {
	blockID uuid.UUID
	trace   *tempopb.Trace
}

type queryBlocksCmd struct {
	backendOptions

	TraceID    string  `arg:"" help:"trace ID to retrieve"`
	TenantID   string  `arg:"" help:"tenant ID to search"`
	Percentage float32 `help:"percentage of blocks to scan e.g..1 for 10%"`
}

func (cmd *queryBlocksCmd) Run(ctx *globalOptions) error {
	r, _, c, err := loadBackend(&cmd.backendOptions, ctx)
	if err != nil {
		return err
	}

	id, err := util.HexStringToTraceID(cmd.TraceID)
	if err != nil {
		return err
	}

	results, err := queryBucket(context.Background(), cmd.Percentage, r, c, cmd.TenantID, id)
	if err != nil {
		return err
	}

	var (
		combiner   = trace.NewCombiner(0)
		marshaller = new(jsonpb.Marshaler)
		jsonBytes  = bytes.Buffer{}
	)

	fmt.Println()
	for i, result := range results {
		fmt.Println(result.blockID, ":")

		err := marshaller.Marshal(&jsonBytes, result.trace)
		if err != nil {
			fmt.Println("failed to marshal to json: ", err)
			continue
		}

		fmt.Println(jsonBytes.String())
		jsonBytes.Reset()
		_, err = combiner.ConsumeWithFinal(result.trace, i == len(results)-1)
		if err != nil {
			return fmt.Errorf("error combining trace: %w", err)
		}
	}

	combinedTrace, _ := combiner.Result()
	fmt.Println("combined:")
	err = marshaller.Marshal(&jsonBytes, combinedTrace)
	if err != nil {
		fmt.Println("failed to marshal to json: ", err)
		return nil
	}
	fmt.Println(jsonBytes.String())
	return nil
}

func queryBucket(ctx context.Context, percentage float32, r backend.Reader, c backend.Compactor, tenantID string, traceID common.ID) ([]queryResults, error) {
	blockIDs, compactedBlockIDs, err := r.Blocks(context.Background(), tenantID)
	if err != nil {
		return nil, err
	}

	if percentage > 0 {
		// shuffle
		rand.Shuffle(len(blockIDs), func(i, j int) { blockIDs[i], blockIDs[j] = blockIDs[j], blockIDs[i] })

		// get the first n%
		total := len(blockIDs)
		total = int(float32(total) * percentage)
		blockIDs = blockIDs[:total]
	}
	fmt.Println("total blocks to search: ", len(blockIDs))

	blockIDs = append(blockIDs, compactedBlockIDs...)

	// Load in parallel
	wg := boundedwaitgroup.New(100)
	resultsCh := make(chan queryResults, len(blockIDs))

	for blockNum, id := range blockIDs {
		wg.Add(1)

		go func(blockNum2 int, id2 uuid.UUID) {
			defer wg.Done()

			// search here
			q, err := queryBlock(ctx, r, c, blockNum2, id2, tenantID, traceID)
			if err != nil {
				fmt.Println("Error querying block:", err)
				return
			}

			if q != nil {
				resultsCh <- *q
			}
		}(blockNum, id)
	}

	wg.Wait()
	close(resultsCh)

	results := make([]queryResults, 0)
	for q := range resultsCh {
		results = append(results, q)
	}

	return results, nil
}

func queryBlock(ctx context.Context, r backend.Reader, _ backend.Compactor, blockNum int, id uuid.UUID, tenantID string, traceID common.ID) (*queryResults, error) {
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

	trace, err := block.FindTraceByID(ctx, traceID, searchOpts)
	if err != nil {
		return nil, err
	}

	if trace == nil {
		return nil, nil
	}

	return &queryResults{
		blockID: id,
		trace:   trace,
	}, nil
}
