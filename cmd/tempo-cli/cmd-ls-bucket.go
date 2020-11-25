package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	tempodb_backend "github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/olekukonko/tablewriter"
)

type lsBucketCmd struct {
	backendOptions

	TenantID string `arg:"" help:"tenant-id within the bucket"`
}

func (l *lsBucketCmd) Run(ctx *globalOptions) error {
	r, _, c, err := loadBackend(&l.backendOptions)
	if err != nil {
		return err
	}

	return dumpBucket(r, c, l.TenantID, time.Hour)
}

func dumpBucket(r tempodb_backend.Reader, c tempodb_backend.Compactor, tenantID string, windowRange time.Duration) error {
	blockIDs, err := r.Blocks(context.Background(), tenantID)
	if err != nil {
		return err
	}

	fmt.Println("total blocks: ", len(blockIDs))

	totalObjects := 0
	out := make([][]string, 0)
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

		indexBytes, err := r.Index(context.Background(), id, tenantID)
		totalIDs := -1
		duplicateIDs := -1
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

		objects, lvl, window, start, end := blockStats(meta, compactedMeta, windowRange)
		out = append(out, []string{
			id.String(),
			strconv.Itoa(int(lvl)),
			strconv.Itoa(totalIDs),
			strconv.Itoa(objects),
			strconv.Itoa(int(window)),
			strconv.Itoa(duplicateIDs),
			start.Format(time.RFC3339),
			end.Format(time.RFC3339),
		})
		totalObjects += objects
	}

	fmt.Println()

	sort.Slice(out, func(i, j int) bool {
		lineI := out[i]
		lineJ := out[j]

		if lineI[4] == lineJ[4] {
			return lineI[1] < lineJ[1]
		}

		return lineI[4] < lineJ[4]
	})

	w := tablewriter.NewWriter(os.Stdout)
	w.SetHeader([]string{"id", "lvl", "idx", "count", "window", "dupe", "start", "end"})
	w.SetFooter([]string{"", "", "", strconv.Itoa(totalObjects), "", "", "", ""})
	w.AppendBulk(out)
	w.Render()

	return nil
}
