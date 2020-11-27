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

type lsBucketCmd struct {
	backendOptions

	TenantID  string `arg:"" help:"tenant-id within the bucket"`
	LoadIndex bool   `help:"load block indexes and display additional information"`
}

func (l *lsBucketCmd) Run(ctx *globalOptions) error {
	r, _, c, err := loadBackend(&l.backendOptions)
	if err != nil {
		return err
	}

	return dumpBucket(r, c, l.TenantID, time.Hour, l.LoadIndex)
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

func dumpBucket(r tempodb_backend.Reader, c tempodb_backend.Compactor, tenantID string, windowRange time.Duration, loadIndex bool) error {
	blockIDs, err := r.Blocks(context.Background(), tenantID)
	if err != nil {
		return err
	}

	fmt.Println("total blocks: ", len(blockIDs))
	results := make([]bucketStats, 0)

	for _, id := range blockIDs {
		fmt.Print(".")

		meta, err := r.BlockMeta(context.Background(), id, tenantID)
		if err != nil && err != tempodb_backend.ErrMetaDoesNotExist {
			return err
		}

		compactedMeta, err := c.CompactedBlockMeta(id, tenantID)
		if err != nil && err != tempodb_backend.ErrMetaDoesNotExist {
			return err
		}

		totalIDs := -1
		duplicateIDs := -1

		if loadIndex {
			indexBytes, err := r.Index(context.Background(), id, tenantID)
			if err == nil {
				records, err := encoding.UnmarshalRecords(indexBytes)
				if err != nil {
					return err
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

	totalObjects := 0
	out := make([][]string, 0)
	for _, r := range results {

		out = append(out, []string{
			r.id.String(),
			strconv.Itoa(int(r.compactionLevel)),
			strconv.Itoa(r.totalIDs),
			strconv.Itoa(r.objects),
			strconv.Itoa(int(r.window)),
			strconv.Itoa(r.duplicateIDs),
			r.start.Format(time.RFC3339),
			r.end.Format(time.RFC3339),
		})
		totalObjects += r.objects
	}

	fmt.Println()
	w := tablewriter.NewWriter(os.Stdout)
	w.SetHeader([]string{"id", "lvl", "idx", "count", "window", "dupe", "start", "end"})
	w.SetFooter([]string{"", "", "", strconv.Itoa(totalObjects), "", "", "", ""})
	w.AppendBulk(out)
	w.Render()

	return nil
}
