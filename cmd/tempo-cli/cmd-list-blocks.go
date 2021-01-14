package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	tempodb_backend "github.com/grafana/tempo/tempodb/backend"
	"github.com/olekukonko/tablewriter"
)

type listBlocksCmd struct {
	TenantID         string `arg:"" help:"tenant-id within the bucket"`
	IncludeCompacted bool   `help:"include compacted blocks"`

	backendOptions
}

func (l *listBlocksCmd) Run(ctx *globalOptions) error {
	r, c, err := loadBackend(&l.backendOptions, ctx)
	if err != nil {
		return err
	}

	windowDuration := time.Hour

	results, err := loadBucket(r, c, l.TenantID, windowDuration, l.IncludeCompacted)
	if err != nil {
		return err
	}

	displayResults(results, windowDuration, l.IncludeCompacted)
	return nil
}

type blockStats struct {
	unifiedBlockMeta
}

func loadBucket(r tempodb_backend.Reader, c tempodb_backend.Compactor, tenantID string, windowRange time.Duration, includeCompacted bool) ([]blockStats, error) {
	blockIDs, err := r.Blocks(context.Background(), tenantID)
	if err != nil {
		return nil, err
	}

	fmt.Println("total blocks: ", len(blockIDs))

	// Load in parallel
	wg := newBoundedWaitGroup(10)
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
		return results[i].end.Before(results[j].end)
	})

	return results, nil
}

func loadBlock(r tempodb_backend.Reader, c tempodb_backend.Compactor, tenantID string, id uuid.UUID, windowRange time.Duration, includeCompacted bool) (*blockStats, error) {
	fmt.Print(".")

	meta, err := r.BlockMeta(context.Background(), id, tenantID)
	if err == tempodb_backend.ErrMetaDoesNotExist && !includeCompacted {
		return nil, nil
	} else if err != nil && err != tempodb_backend.ErrMetaDoesNotExist {
		return nil, err
	}

	compactedMeta, err := c.CompactedBlockMeta(id, tenantID)
	if err != nil && err != tempodb_backend.ErrMetaDoesNotExist {
		return nil, err
	}

	return &blockStats{
		unifiedBlockMeta: getMeta(meta, compactedMeta, windowRange),
	}, nil
}

func displayResults(results []blockStats, windowDuration time.Duration, includeCompacted bool) {

	columns := []string{"id", "lvl", "count", "window", "start", "end", "duration", "age"}
	if includeCompacted {
		columns = append(columns, "cmp")
	}

	totalObjects := 0
	out := make([][]string, 0)
	for _, r := range results {

		line := make([]string, 0)

		for _, c := range columns {
			s := ""
			switch c {
			case "id":
				s = r.id.String()
			case "lvl":
				s = strconv.Itoa(int(r.compactionLevel))
			case "count":
				s = strconv.Itoa(r.objects)
			case "window":
				// Display compaction window in human-readable format
				window := time.Unix(r.window*int64(windowDuration.Seconds()), 0).UTC()
				s = window.Format(time.RFC3339)
			case "start":
				s = r.start.Format(time.RFC3339)
			case "end":
				s = r.end.Format(time.RFC3339)
			case "duration":
				// Time range included in bucket
				s = fmt.Sprint(r.end.Sub(r.start).Round(time.Second))
			case "age":
				s = fmt.Sprint(time.Since(r.end).Round(time.Second))
			case "cmp":
				// Compacted?
				if r.compacted {
					s = "Y"
				} else {
					s = " "
				}
			}

			line = append(line, s)
		}

		out = append(out, line)
		totalObjects += r.objects
	}

	footer := make([]string, 0)
	for _, c := range columns {
		switch c {
		case "count":
			footer = append(footer, strconv.Itoa(totalObjects))
		default:
			footer = append(footer, "")
		}
	}

	fmt.Println()
	w := tablewriter.NewWriter(os.Stdout)
	w.SetHeader(columns)
	w.SetFooter(footer)
	w.AppendBulk(out)
	w.Render()
}
