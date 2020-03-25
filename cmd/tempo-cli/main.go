package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/grafana/tempo/tempodb/backend"

	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/olekukonko/tablewriter"
)

var (
	gcsBucket string
	tenantID  string
)

func init() {
	flag.StringVar(&gcsBucket, "gcs-bucket", "", "bucket to scan")
	flag.StringVar(&tenantID, "tenant-id", "", "tenant-id that contains the bucket")
}

func main() {
	flag.Parse()

	if len(gcsBucket) == 0 {
		fmt.Println("-gcs-bucket is required")
		return
	}

	if len(tenantID) == 0 {
		fmt.Println("-tenant-id is required")
		return
	}

	err := dumpBucket(gcsBucket, tenantID)
	if err != nil {
		fmt.Printf("%v", err)
	}
}

func dumpBucket(bucketName string, tenantID string) error {
	r, _, c, err := gcs.New(&gcs.Config{
		BucketName:      bucketName,
		ChunkBufferSize: 10 * 1024 * 1024,
	})
	if err != nil {
		return err
	}

	blockIDs, err := r.Blocks(tenantID)
	if err != nil {
		return err
	}

	fmt.Println("total blocks: ", len(blockIDs))

	totalObjects := 0
	out := make([][]string, 0)
	for _, id := range blockIDs {
		meta, err := r.BlockMeta(id, tenantID)
		if err != nil && err != backend.ErrMetaDoesNotExist {
			return err
		}

		compactedMeta, err := c.CompactedBlockMeta(id, tenantID)
		if err != nil && err != backend.ErrMetaDoesNotExist {
			return err
		}

		indexBytes, err := r.Index(id, tenantID)
		totalIDs := -1
		duplicateIDs := -1
		if err == nil {
			records, err := backend.UnmarshalRecords(indexBytes)
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

		objects, lvl, start, end := blockStats(meta, compactedMeta)
		out = append(out, []string{
			id.String(),
			strconv.Itoa(int(lvl)),
			strconv.Itoa(totalIDs),
			strconv.Itoa(objects),
			strconv.Itoa(duplicateIDs),
			start.Format(time.RFC3339),
			end.Format(time.RFC3339),
		})
		totalObjects += objects
	}

	w := tablewriter.NewWriter(os.Stdout)
	w.SetHeader([]string{"id", "lvl", "idx", "count", "dupe", "start", "end"})
	w.SetFooter([]string{"", "", "", strconv.Itoa(totalObjects), "", "", ""})
	w.AppendBulk(out)
	w.Render()

	return nil
}

func blockStats(meta *backend.BlockMeta, compactedMeta *backend.CompactedBlockMeta) (int, uint8, time.Time, time.Time) {
	if meta != nil {
		return meta.TotalObjects, meta.CompactionLevel, meta.StartTime, meta.EndTime
	} else if compactedMeta != nil {
		return compactedMeta.TotalObjects, compactedMeta.CompactionLevel, compactedMeta.StartTime, compactedMeta.EndTime
	}

	return -1, 0, time.Unix(0, 0), time.Unix(0, 0)
}
