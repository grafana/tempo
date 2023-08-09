package main

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type queryResults struct {
	blockID uuid.UUID
	trace   *tempopb.Trace
}

type queryBlocksCmd struct {
	backendOptions

	TraceID  string `arg:"" help:"trace ID to retrieve"`
	TenantID string `arg:"" help:"tenant ID to search"`
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

	results, err := queryBucket(context.Background(), r, c, cmd.TenantID, id)
	if err != nil {
		return err
	}

	var (
		combiner   = trace.NewCombiner()
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
		combiner.ConsumeWithFinal(result.trace, i == len(results)-1)
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

func queryBucket(ctx context.Context, r backend.Reader, c backend.Compactor, tenantID string, traceID common.ID) ([]queryResults, error) {
	blockIDs, err := r.Blocks(context.Background(), tenantID)
	if err != nil {
		return nil, err
	}

	fmt.Println("total blocks: ", len(blockIDs))

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

func queryBlock(ctx context.Context, r backend.Reader, c backend.Compactor, blockNum int, id uuid.UUID, tenantID string, traceID common.ID) (*queryResults, error) {
	fmt.Print(".")
	if blockNum%100 == 0 {
		fmt.Print(strconv.Itoa(blockNum))
	}

	meta, err := r.BlockMeta(context.Background(), id, tenantID)
	if err != nil && err != backend.ErrDoesNotExist {
		return nil, err
	}

	if err == backend.ErrDoesNotExist {
		compactedMeta, err := c.CompactedBlockMeta(id, tenantID)
		if err != nil && err != backend.ErrDoesNotExist {
			return nil, err
		}

		if compactedMeta == nil {
			return nil, fmt.Errorf("compacted meta nil?")
		}

		meta = &compactedMeta.BlockMeta
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
