package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type dropTraceCmd struct {
	backendOptions

	TraceID   string `arg:"" help:"trace ID to retrieve"`
	TenantID  string `arg:"" help:"tenant ID to search"`
	DropTrace bool   `name:"drop-trace" help:"actually attempt to drop the trace" default:"false"`
}

func (cmd *dropTraceCmd) Run(ctx *globalOptions) error {
	fmt.Printf("beginning process to drop trace %v from tenant %v\n", cmd.TraceID, cmd.TenantID)
	fmt.Println("**warning**: compaction must be disabled or a compactor may duplicate a block as this process is rewriting it")
	fmt.Println("")
	if cmd.DropTrace {
		fmt.Println("************************************************************************")
		fmt.Println("**this is not a dry run. blocks will be rewritten and marked compacted**")
		fmt.Println("************************************************************************")
		fmt.Println("")
	}

	r, w, c, err := loadBackend(&cmd.backendOptions, ctx)
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

	if len(blocks) == 0 {
		fmt.Println("\ntrace not found in any block. aborting")
		return nil
	}

	// print out blocks that have the trace id
	fmt.Println("\n\ntrace found in:")
	for _, block := range blocks {
		fmt.Printf("  %v sz: %d traces: %d\n", block.BlockID, block.Size, block.TotalObjects)
	}

	if !cmd.DropTrace {
		fmt.Println("**not dropping trace, use --drop-trace to actually drop**")
		return nil
	}

	fmt.Println("rewriting blocks:")
	for _, block := range blocks {
		fmt.Printf("  rewriting %v\n", block.BlockID)
		newBlock, err := rewriteBlock(context.Background(), r, w, block, id)
		if err != nil {
			return err
		}
		fmt.Printf("  rewrote to new block: %v\n", newBlock.BlockID)
	}

	fmt.Println("marking old blocks compacted")
	for _, block := range blocks {
		fmt.Printf("  marking %v\n", block.BlockID)
		err = c.MarkBlockCompacted(block.BlockID, block.TenantID)
		if err != nil {
			return err
		}
	}

	fmt.Println("successfully rewrote blocks dropping requested trace")

	return nil
}

func rewriteBlock(ctx context.Context, r backend.Reader, w backend.Writer, meta *backend.BlockMeta, traceID common.ID) (*backend.BlockMeta, error) {
	enc, err := encoding.FromVersion(meta.Version)
	if err != nil {
		return nil, fmt.Errorf("error getting encoder: %w", err)
	}

	// todo: provide a way to pass a config in. this just uses defaults which is fine for now
	opts := common.CompactionOptions{
		BlockConfig: common.BlockConfig{
			// defaults should be fine for just recreating a few blocks
			BloomFP:             common.DefaultBloomFP,
			BloomShardSizeBytes: common.DefaultBloomShardSizeBytes,
			Version:             meta.Version,

			// these fields aren't in use anymore. we need to remove the old flatbuffer search. setting them for completeness
			SearchEncoding:      backend.EncSnappy,
			SearchPageSizeBytes: 1024 * 1024,

			// v2 fields
			IndexDownsampleBytes: common.DefaultIndexDownSampleBytes,
			IndexPageSizeBytes:   common.DefaultIndexPageSizeBytes,
			Encoding:             backend.EncZstd,

			// parquet fields
			RowGroupSizeBytes: 100_000_000, // default

			// vParquet3 fields
			DedicatedColumns: meta.DedicatedColumns,
		},
		ChunkSizeBytes:     tempodb.DefaultChunkSizeBytes,
		FlushSizeBytes:     tempodb.DefaultFlushSizeBytes,
		IteratorBufferSize: tempodb.DefaultIteratorBufferSize,
		OutputBlocks:       1,
		Combiner:           model.StaticCombiner, // this should never be necessary b/c we are only compacting one block
		MaxBytesPerTrace:   0,                    // disable for this process

		// hook to drop the trace
		DropObject: func(id common.ID) bool {
			return bytes.Equal(id, traceID)
		},

		// setting to prevent panics. should we track and report these?
		BytesWritten:      func(_, _ int) {},
		ObjectsCombined:   func(_, _ int) {},
		ObjectsWritten:    func(_, _ int) {},
		SpansDiscarded:    func(_, _, _ string, _ int) {},
		DisconnectedTrace: func() {},
		RootlessTrace:     func() {},
	}

	compactor := enc.NewCompactor(opts)
	fmt.Println("--beginning compaction logs--")
	out, err := compactor.Compact(ctx, log.NewLogfmtLogger(os.Stdout), r, w, []*backend.BlockMeta{meta})
	fmt.Println("--ending compaction logs--")
	if err != nil {
		return nil, err
	}

	if len(out) != 1 {
		return nil, fmt.Errorf("expected 1 block, got %d", len(out))
	}

	newMeta := out[0]

	if newMeta.TotalObjects != meta.TotalObjects-1 {
		return nil, fmt.Errorf("expected output to have one less object then in. out: %d in: %d", newMeta.TotalObjects, meta.TotalObjects)
	}

	return newMeta, nil
}

func blocksWithTraceID(ctx context.Context, r backend.Reader, tenantID string, traceID common.ID) ([]*backend.BlockMeta, error) {
	blockIDs, _, err := r.Blocks(context.Background(), tenantID)
	if err != nil {
		return nil, err
	}

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

	results := make([]*backend.BlockMeta, 0, len(resultsCh))
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
