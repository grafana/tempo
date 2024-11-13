package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

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

type dropTracesCmd struct {
	backendOptions

	TenantID  string `arg:"" help:"tenant ID to search"`
	TraceIDs  string `arg:"" help:"Trace IDs to drop"`
	DropTrace bool   `name:"drop-trace" help:"actually attempt to drop the trace" default:"false"`
}

func (cmd *dropTracesCmd) Run(opts *globalOptions) error {
	fmt.Printf("beginning process to drop traces %v from tenant %v\n", cmd.TraceIDs, cmd.TenantID)
	fmt.Println("**warning**: compaction must be disabled or a compactor may duplicate a block as this process is rewriting it")
	fmt.Println("")
	if cmd.DropTrace {
		fmt.Println("************************************************************************")
		fmt.Println("**this is not a dry run. blocks will be rewritten and marked compacted**")
		fmt.Println("************************************************************************")
		fmt.Println("")
	}

	ctx := context.Background()

	r, w, c, err := loadBackend(&cmd.backendOptions, opts)
	if err != nil {
		return err
	}

	type pair struct {
		traceIDs  []common.ID
		blockMeta *backend.BlockMeta
	}
	tracesByBlock := map[backend.UUID]pair{}

	// Group trace IDs by blocks
	ids := strings.Split(cmd.TraceIDs, ",")
	for _, id := range ids {
		traceID, err := util.HexStringToTraceID(id)
		if err != nil {
			return err
		}

		// It might be significantly improved if common.BackendBlock supported bulk searches.
		blocks, err := blocksWithTraceID(ctx, r, cmd.TenantID, traceID)
		if err != nil {
			return err
		}

		if len(blocks) == 0 {
			fmt.Printf("\ntrace %s not found in any block. skipping\n", util.TraceIDToHexString(traceID))
		}
		for _, block := range blocks {
			p, ok := tracesByBlock[block.BlockID]
			if !ok {
				p = pair{blockMeta: block}
			}
			p.traceIDs = append(p.traceIDs, traceID)
			tracesByBlock[block.BlockID] = p
		}
	}

	// Remove traces from blocks
	for _, p := range tracesByBlock {
		// print out trace IDs to be removed in the block
		strTraceIDs := make([]string, len(p.traceIDs))
		for i, tid := range p.traceIDs {
			strTraceIDs[i] = util.TraceIDToHexString(tid)
		}
		fmt.Printf("\nFound %d traces: %v in block: %v\n", len(strTraceIDs), strTraceIDs, p.blockMeta.BlockID)
		fmt.Printf("blockInfo: ID: %v, Size: %d Total Traces: %d\n", p.blockMeta.BlockID, p.blockMeta.Size_, p.blockMeta.TotalObjects)

		if !cmd.DropTrace {
			fmt.Println("**not dropping trace, use --drop-trace to actually drop**")
			continue
		}

		fmt.Printf("  rewriting %v\n", p.blockMeta.BlockID)
		newMeta, err := rewriteBlock(ctx, r, w, p.blockMeta, p.traceIDs)
		if err != nil {
			return err
		}
		if newMeta == nil {
			fmt.Printf("  block %v was removed\n", p.blockMeta.BlockID)
		} else {
			fmt.Printf("  rewrote to new block: %v\n", newMeta.BlockID)
		}

		fmt.Printf("  marking %v compacted\n", p.blockMeta.BlockID)
		err = c.MarkBlockCompacted((uuid.UUID)(p.blockMeta.BlockID), p.blockMeta.TenantID)
		if err != nil {
			return err
		}
	}
	if cmd.DropTrace {
		fmt.Printf("successfully rewrote blocks dropping requested traces: %v from tenant: %v\n", cmd.TraceIDs, cmd.TenantID)
	}

	return nil
}

func rewriteBlock(ctx context.Context, r backend.Reader, w backend.Writer, meta *backend.BlockMeta, traceIDs []common.ID) (*backend.BlockMeta, error) {
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
			for _, tid := range traceIDs {
				if bytes.Equal(id, tid) {
					return true
				}
			}
			return false
		},

		// setting to prevent panics. should we track and report these?
		BytesWritten:      func(_, _ int) {},
		ObjectsCombined:   func(_, _ int) {},
		ObjectsWritten:    func(_, _ int) {},
		SpansDiscarded:    func(_, _, _ string, _ int) {},
		DisconnectedTrace: func() {},
		RootlessTrace:     func() {},
		DedupedSpans:      func(_, _ int) {},
	}

	compactor := enc.NewCompactor(opts)
	fmt.Println("--beginning compaction logs--")
	out, err := compactor.Compact(ctx, log.NewLogfmtLogger(os.Stdout), r, w, []*backend.BlockMeta{meta})
	fmt.Println("--ending compaction logs--")
	if err != nil {
		return nil, err
	}

	if len(out) != 1 {
		if meta.TotalObjects == int64(len(traceIDs)) {
			// we removed all traces from the block
			return nil, nil
		}
		return nil, fmt.Errorf("expected 1 block, got %d", len(out))
	}

	newMeta := out[0]

	if newMeta.TotalObjects != meta.TotalObjects-int64(len(traceIDs)) {
		return nil, fmt.Errorf("expected output to have one less object then in. out: %d in: %d", newMeta.TotalObjects, meta.TotalObjects)
	}

	return newMeta, nil
}

func blocksWithTraceID(ctx context.Context, r backend.Reader, tenantID string, traceID common.ID) ([]*backend.BlockMeta, error) {
	blockIDs, _, err := r.Blocks(ctx, tenantID)
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
				fmt.Println("\nError querying block:", err)
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

	meta, err := r.BlockMeta(ctx, id, tenantID)
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
