package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	tempodb_backend "github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/olekukonko/tablewriter"
)

type listBlocksCmd struct {
	TenantID  string `arg:"" help:"tenant-id within the bucket"`
	LoadIndex bool   `help:"load block indexes and display additional information"`

	backendOptions
}

func (l *listBlocksCmd) Run(ctx *globalOptions) error {
	r, _, c, err := loadBackend(&l.backendOptions, ctx)
	if err != nil {
		return err
	}

	windowDuration := time.Hour

	results, err := loadBucket(r, c, l.TenantID, windowDuration, l.LoadIndex)
	if err != nil {
		return err
	}

	displayResults(results, windowDuration, l.LoadIndex)
	return nil
}

type bucketStats struct {
	id              uuid.UUID
	compactionLevel uint8
	objects         int
	window          int64
	start           time.Time
	end             time.Time

	totalIDs     int
	duplicateIDs int
}

func loadBucket(r tempodb_backend.Reader, c tempodb_backend.Compactor, tenantID string, windowRange time.Duration, loadIndex bool) ([]bucketStats, error) {
	blockIDs, err := r.Blocks(context.Background(), tenantID)
	if err != nil {
		return nil, err
	}

	fmt.Println("total blocks: ", len(blockIDs))
	results := make([]bucketStats, 0)

	for _, id := range blockIDs {
		fmt.Print(".")

		meta, err := r.BlockMeta(context.Background(), id, tenantID)
		if err != nil && err != tempodb_backend.ErrMetaDoesNotExist {
			return nil, err
		}

		compactedMeta, err := c.CompactedBlockMeta(id, tenantID)
		if err != nil && err != tempodb_backend.ErrMetaDoesNotExist {
			return nil, err
		}

		totalIDs := -1
		duplicateIDs := -1

		if loadIndex {
			indexBytes, err := r.Index(context.Background(), id, tenantID)
			if err == nil {
				records, err := encoding.UnmarshalRecords(indexBytes)
				if err != nil {
					return nil, err
				}
				duplicateIDs = 0
				totalIDs = len(records)
				for i := 1; i < len(records); i++ {
					if bytes.Equal(records[i-1].ID, records[i].ID) {
						duplicateIDs++
					}
				}
			}
		}

		objects, lvl, window, start, end := blockStats(meta, compactedMeta, windowRange)

		results = append(results, bucketStats{
			id:              id,
			compactionLevel: lvl,
			objects:         objects,
			window:          window,
			start:           start,
			end:             end,

			totalIDs:     totalIDs,
			duplicateIDs: duplicateIDs,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		bI := results[i]
		bJ := results[j]

		if bI.window == bJ.window {
			return bI.compactionLevel < bJ.compactionLevel
		}

		return bI.window < bJ.window
	})

	return results, nil
}

func displayResults(results []bucketStats, windowDuration time.Duration, includeIndexInfo bool) {

	columns := []string{"id", "lvl", "count", "window", "start", "end", "duration"}
	if includeIndexInfo {
		columns = append(columns, "idx", "dupe")
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
			case "idx":
				// Number of entries in the index (may not be the same as the block when index downsampling enabled)
				s = strconv.Itoa(r.totalIDs)
			case "dupe":
				// Number of duplicate IDs found in the index
				s = strconv.Itoa(r.duplicateIDs)
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
