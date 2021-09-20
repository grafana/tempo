package main

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type searchBlocksCmd struct {
	backendOptions

	Name     string `arg:"" help:"attribute name to search for"`
	Value    string `arg:"" help:"attribute value to search for"`
	Start    int64  `arg:"" help:"start of time range to search (unix epoch)"`
	End      int64  `arg:"" help:"end of time range to search (unix epoch)"`
	Limit    int    `arg:"" help:"maximum number of results to return"`
	TenantID string `arg:"" help:"tenant ID to search"`
}

func (cmd *searchBlocksCmd) Run(opts *globalOptions) error {
	r, _, _, err := loadBackend(&cmd.backendOptions, opts)
	if err != nil {
		return err
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
	for blockNum, id := range blockIDs {
		wg.Add(1)

		go func(blockNum2 int, id2 uuid.UUID) {
			defer wg.Done()

			// search here
			meta, err := r.BlockMeta(ctx, id, cmd.TenantID)
			if err == backend.ErrDoesNotExist {
				return
			}
			if err != nil {
				fmt.Println("Error querying block:", err)
				return
			}
			if meta.StartTime.Unix() <= cmd.End &&
				meta.EndTime.Unix() >= cmd.Start {
				resultsCh <- meta
			}
		}(blockNum, id)
	}

	wg.Wait()
	close(resultsCh)

	blockmetas := []*backend.BlockMeta{}
	for q := range resultsCh {
		blockmetas = append(blockmetas, q)
	}

	fmt.Println("Blocks In Range:", len(blockmetas))
	foundids := []common.ID{}
	for _, meta := range blockmetas {
		block, err := encoding.NewBackendBlock(meta, r)
		if err != nil {
			return err
		}

		// todo : graduated chunk sizes will increase throughput
		iter, err := block.Iterator(10 * 1024 * 1024) // jpe: param chunk size
		if err != nil {
			return err
		}

		prefetchIter := encoding.NewPrefetchIterator(ctx, iter, 1000) // jpe : param iterator buffer
		ids, err := searchIterator(iter, meta.DataEncoding, cmd.Name, cmd.Value, cmd.Limit)
		prefetchIter.Close()
		if err != nil {
			return err
		}

		foundids = append(foundids, ids...)
		if len(foundids) >= cmd.Limit {
			break
		}
	}

	fmt.Println("Matching Traces:", len(foundids))
	for _, id := range foundids {
		fmt.Println("  ", util.TraceIDToHexString(id))
	}

	return nil
}

func searchIterator(iter encoding.Iterator, dataEncoding string, name string, value string, limit int) ([]common.ID, error) {
	ctx := context.Background()
	found := []common.ID{}

	for {
		id, obj, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		trace, err := model.Unmarshal(obj, dataEncoding)
		if err != nil {
			return nil, err
		}

		if traceContainsKeyValue(trace, name, value) {
			found = append(found, id)
		}

		if len(found) >= limit {
			break
		}
	}

	return found, nil
}

func traceContainsKeyValue(trace *tempopb.Trace, name string, value string) bool {
	// jpe : only works for string values
	for _, b := range trace.Batches {
		for _, a := range b.Resource.Attributes {
			if a.Key == name && a.Value.GetStringValue() == value {
				return true
			}
		}

		for _, ils := range b.InstrumentationLibrarySpans {
			for _, s := range ils.Spans {
				for _, a := range s.Attributes {
					if a.Key == name && a.Value.GetStringValue() == value {
						return true
					}
				}
			}
		}
	}

	return false
}
