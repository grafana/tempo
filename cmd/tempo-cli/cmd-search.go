package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const (
	layoutString   = "2006-01-02T15:04:05"
	chunkSize      = 10 * 1024 * 1024
	iteratorBuffer = 10000
	limit          = 20
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
			if err == backend.ErrDoesNotExist {
				return
			}
			if err != nil {
				fmt.Println("Error querying block:", err)
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

	fmt.Println("Blocks In Range:", len(blockmetas))
	foundids := []common.ID{}
	for _, meta := range blockmetas {
		block, err := encoding.NewBackendBlock(meta, r)
		if err != nil {
			return err
		}

		// todo : graduated chunk sizes will increase throughput. i.e. first request should be small to feed the below parsing faster
		//  later queries should use larger chunk sizes to be more efficient
		iter, err := block.Iterator(chunkSize)
		if err != nil {
			return err
		}

		prefetchIter := encoding.NewPrefetchIterator(ctx, iter, iteratorBuffer)
		ids, err := searchIterator(prefetchIter, meta.DataEncoding, cmd.Name, cmd.Value, limit)
		prefetchIter.Close()
		if err != nil {
			return err
		}

		foundids = append(foundids, ids...)
		if len(foundids) >= limit {
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

		// todo : parrallelize unmarshal and search
		trace, err := model.MustNewDecoder(dataEncoding).PrepareForRead(obj)
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
	// todo : support other attribute types besides string
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
