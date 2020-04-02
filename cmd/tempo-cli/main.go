package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"

	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/olekukonko/tablewriter"
)

var (
	gcsBucket   string
	tenantID    string
	windowRange time.Duration
	blockID     string
)

func init() {
	flag.StringVar(&gcsBucket, "gcs-bucket", "", "bucket to scan")
	flag.StringVar(&tenantID, "tenant-id", "", "tenant-id that contains the bucket")
	flag.StringVar(&blockID, "block-id", "", "block-id to dump (optional)")
	flag.DurationVar(&windowRange, "window-range", 4*time.Hour, "block time window range for compaction")
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

	var err error
	if len(blockID) > 0 {
		err = dumpBlock(gcsBucket, tenantID, windowRange, blockID)
	} else {
		err = dumpBucket(gcsBucket, tenantID, windowRange)
	}

	if err != nil {
		fmt.Printf("%v", err)
	}
}

func dumpBlock(bucketName string, tenantID string, windowRange time.Duration, blockID string) error {
	r, _, c, err := gcs.New(&gcs.Config{
		BucketName:      bucketName,
		ChunkBufferSize: 10 * 1024 * 1024,
	})
	if err != nil {
		return err
	}

	id := uuid.MustParse(blockID)

	meta, err := r.BlockMeta(id, tenantID)
	if err != nil {
		return err
	}

	compactedMeta, err := c.CompactedBlockMeta(id, tenantID)
	if err != nil && err != backend.ErrMetaDoesNotExist {
		return err
	}

	objects, lvl, window, start, end := blockStats(meta, compactedMeta, windowRange)

	fmt.Println("ID            : ", id)
	fmt.Println("Total Objects : ", objects)
	fmt.Println("Level         : ", lvl)
	fmt.Println("Window        : ", window)
	fmt.Println("Start         : ", start)
	fmt.Println("End           : ", end)

	fmt.Println("Searching for dupes ...")

	iter, err := backend.NewBackendIterator(tenantID, id, 10*1024*1024, r)
	if err != nil {
		return err
	}

	i := 0
	dupe := 0
	prevID := make([]byte, 16)
	for {
		objID, _, err := iter.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if bytes.Equal(objID, prevID) {
			dupe++
		}

		copy(prevID, objID)
		i++
		if i%100000 == 0 {
			fmt.Println("Record: ", i)
		}
	}

	fmt.Println("total: ", i)
	fmt.Println("dupes: ", dupe)

	return nil
}

func dumpBucket(bucketName string, tenantID string, windowRange time.Duration) error {
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

func blockStats(meta *backend.BlockMeta, compactedMeta *backend.CompactedBlockMeta, windowRange time.Duration) (int, uint8, int64, time.Time, time.Time) {
	if meta != nil {
		return meta.TotalObjects, meta.CompactionLevel, meta.EndTime.Unix() / int64(windowRange/time.Second), meta.StartTime, meta.EndTime
	} else if compactedMeta != nil {
		return compactedMeta.TotalObjects, compactedMeta.CompactionLevel, compactedMeta.EndTime.Unix() / int64(windowRange/time.Second), compactedMeta.StartTime, compactedMeta.EndTime
	}

	return -1, 0, -1, time.Unix(0, 0), time.Unix(0, 0)
}
