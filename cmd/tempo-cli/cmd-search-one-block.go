package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
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

	// blockID, err := uuid.Parse(cmd.BlockID)
	// if err != nil {
	// 	return err
	// }

	searchReq := &tempopb.SearchRequest{
		Tags:          map[string]string{cmd.Name: cmd.Value},
		MinDurationMs: 1000,
		Limit:         limit,
	}

	// find requested block
	wg := sync.WaitGroup{}
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go find(r, "a6cb3099-8763-4f43-bff2-01a3a5d420c9", searchReq, &wg)
		go find(r, "62153e5b-6c4e-442d-a05d-852e190cfc26", searchReq, &wg)
	}

	wg.Wait()

	return nil
}

func find(r backend.Reader, blockID string, searchReq *tempopb.SearchRequest, wg *sync.WaitGroup) {
	defer wg.Done()
	ctx := context.Background()

	meta, err := r.BlockMeta(ctx, uuid.MustParse(blockID), "1")
	if err != nil {
		panic(errors.Wrap(err, "failed to find block meta"))
	}

	fmt.Println("Searching:")
	spew.Dump(meta)

	searchOpts := common.SearchOptions{}
	tempodb.SearchConfig{}.ApplyToOptions(&searchOpts)
	searchOpts.PrefetchTraceCount = 1000
	searchOpts.ReadBufferSize = 1024 * 1024
	searchOpts.StartPage = 5
	searchOpts.TotalPages = rand.Intn(10) + 1

	block, err := encoding.OpenBlock(meta, r)
	if err != nil {
		panic(errors.Wrap(err, "failed to open block"))
	}

	resp, err := block.Search(ctx, searchReq, searchOpts)
	if err != nil {
		panic(errors.Wrap(err, "error searching block"))
	}

	fmt.Println(resp)
}
