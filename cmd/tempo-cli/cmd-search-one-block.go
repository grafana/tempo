package main

import (
	"context"
	"fmt"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend/instrumentation"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/pkg/errors"
)

type searchOneBlockCmd struct {
	backendOptions

	Name     string `arg:"" help:"attribute name to search for"`
	Value    string `arg:"" help:"attribute value to search for"`
	BlockID  string `arg:"" help:"guid of block to search"`
	TenantID string `arg:"" help:"tenant ID to search"`
}

func (cmd *searchOneBlockCmd) Run(opts *globalOptions) error {
	r, _, _, err := loadBackend(&cmd.backendOptions, opts)
	if err != nil {
		return err
	}

	blockID, err := uuid.Parse(cmd.BlockID)
	if err != nil {
		return err
	}

	searchReq := &tempopb.SearchRequest{
		Tags:  map[string]string{cmd.Name: cmd.Value},
		Limit: limit,
	}

	ctx := context.Background()

	// find requested block
	meta, err := r.BlockMeta(ctx, blockID, cmd.TenantID)
	if err != nil {
		return errors.Wrap(err, "failed to find block meta")
	}

	fmt.Println("Searching:")
	spew.Dump(meta)

	searchOpts := common.DefaultSearchOptions()
	searchOpts.ChunkSizeBytes = chunkSize
	searchOpts.PrefetchTraceCount = iteratorBuffer
	// jpe make these default?
	searchOpts.ReadBufferCount = 8
	searchOpts.ReadBufferSize = 1 * 1024 * 1024

	start := time.Now()
	block, err := encoding.OpenBlock(meta, r)
	if err != nil {
		return errors.Wrap(err, "failed to open block")
	}

	resp, err := block.Search(ctx, searchReq, searchOpts)
	if err != nil {
		return errors.Wrap(err, "error searching block:")
	}

	fmt.Println("Duration:", time.Since(start))
	fmt.Println("Response:")
	spew.Dump(resp)

	vals, err := test.GetHistogramValue(instrumentation.HistogramVec(), "GET", "206")
	if err != nil {
		fmt.Println("Error retrieving histogram: ", err)
		return nil
	}
	spew.Dump(vals)

	return nil
}
