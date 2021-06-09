package main

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/tempodb/backend"
)

type unifiedBlockMeta struct {
	backend.BlockMeta
	backend.CompactedBlockMeta

	window    int64
	compacted bool
}

func getMeta(meta *backend.BlockMeta, compactedMeta *backend.CompactedBlockMeta, windowRange time.Duration) unifiedBlockMeta {
	if meta != nil {
		return unifiedBlockMeta{
			BlockMeta: *meta,
			window:    meta.EndTime.Unix() / int64(windowRange/time.Second),
			compacted: false,
		}
	}
	if compactedMeta != nil {
		return unifiedBlockMeta{
			BlockMeta:          compactedMeta.BlockMeta,
			CompactedBlockMeta: *compactedMeta,
			window:             compactedMeta.EndTime.Unix() / int64(windowRange/time.Second),
			compacted:          true,
		}
	}

	return unifiedBlockMeta{
		BlockMeta: backend.BlockMeta{
			BlockID:         uuid.UUID{},
			CompactionLevel: 0,
			TotalObjects:    -1,
		},
		window:    -1,
		compacted: false,
	}
}

type blockStats struct {
	unifiedBlockMeta
}

func loadBucket(r backend.Reader, c backend.Compactor, tenantID string, windowRange time.Duration, includeCompacted bool) ([]blockStats, error) {
	blockIDs, err := r.Blocks(context.Background(), tenantID)
	if err != nil {
		return nil, err
	}

	fmt.Println("total blocks: ", len(blockIDs))

	// Load in parallel
	wg := boundedwaitgroup.New(10)
	resultsCh := make(chan blockStats, len(blockIDs))

	for _, id := range blockIDs {
		wg.Add(1)

		go func(id2 uuid.UUID) {
			defer wg.Done()

			b, err := loadBlock(r, c, tenantID, id2, windowRange, includeCompacted)
			if err != nil {
				fmt.Println("Error loading block:", id2, err)
				return
			}

			if b != nil {
				resultsCh <- *b
			}
		}(id)
	}

	wg.Wait()
	close(resultsCh)

	results := make([]blockStats, 0)
	for b := range resultsCh {
		results = append(results, b)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].EndTime.Before(results[j].EndTime)
	})

	return results, nil
}

func loadBlock(r backend.Reader, c backend.Compactor, tenantID string, id uuid.UUID, windowRange time.Duration, includeCompacted bool) (*blockStats, error) {
	fmt.Print(".")

	meta, err := r.BlockMeta(context.Background(), id, tenantID)
	if err == backend.ErrMetaDoesNotExist && !includeCompacted {
		return nil, nil
	} else if err != nil && err != backend.ErrMetaDoesNotExist {
		return nil, err
	}

	compactedMeta, err := c.CompactedBlockMeta(id, tenantID)
	if err != nil && err != backend.ErrMetaDoesNotExist {
		return nil, err
	}

	return &blockStats{
		unifiedBlockMeta: getMeta(meta, compactedMeta, windowRange),
	}, nil
}
