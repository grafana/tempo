package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	tempodb_backend "github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"

	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/grafana/tempo/tempodb/backend/s3"
	"github.com/olekukonko/tablewriter"
)

var (
	bucket      string
	s3Endpoint  string
	s3User      string
	s3Pass      string
	backend     string
	tenantID    string
	windowRange time.Duration
	blockID     string

	queryEndpoint string
	traceID       string
	orgID         string
)

func init() {
	flag.StringVar(&backend, "backend", "", "backend to connect to (s3/gcs)")
	flag.StringVar(&bucket, "bucket", "", "bucket to scan")
	flag.StringVar(&s3Endpoint, "s3-endpoint", "", "s3 endpoint")
	flag.StringVar(&s3User, "s3-user", "", "s3 username")
	flag.StringVar(&s3Pass, "s3-pass", "", "s3 password")
	flag.StringVar(&tenantID, "tenant-id", "", "tenant-id that contains the bucket")
	flag.StringVar(&blockID, "block-id", "", "block-id to dump (optional)")
	flag.DurationVar(&windowRange, "window-range", 4*time.Hour, "block time window range for compaction")

	flag.StringVar(&queryEndpoint, "query-endpoint", "", "tempo query endpoint")
	flag.StringVar(&traceID, "traceID", "", "traceID to query")
	flag.StringVar(&orgID, "orgID", "", "orgID to query")
}

func main() {
	flag.Parse()

	if len(queryEndpoint) > 0 && len(traceID) > 0 {
		// util.QueryTrace will only add orgID header if len(orgID) > 0
		trace, err := util.QueryTrace(queryEndpoint, traceID, orgID)
		if err != nil {
			fmt.Println("error querying tempo, err:", err)
			return
		}

		traceJSON, _ := json.Marshal(trace)
		fmt.Println(string(traceJSON))
		fmt.Println("------------------------------------")
		return
	}

	if len(backend) == 0 {
		fmt.Println("-backend is required")
		return
	}

	if len(bucket) == 0 {
		fmt.Println("-bucket is required")
		return
	}

	if len(tenantID) == 0 {
		fmt.Println("-tenant-id is required")
		return
	}

	r, _, c, err := getBackendUtils(backend, bucket, s3Endpoint, s3User, s3Pass)
	if err != nil {
		fmt.Printf("error creating backend utils, please check config")
	}

	if len(blockID) > 0 {
		err = dumpBlock(r, c, tenantID, windowRange, blockID)
	} else {
		err = dumpBucket(r, c, tenantID, windowRange)
	}

	if err != nil {
		fmt.Printf("%v", err)
	}
}

func getBackendUtils(backend, bucket, s3Endpoint, s3User, s3Pass string) (tempodb_backend.Reader, tempodb_backend.Writer, tempodb_backend.Compactor, error) {
	switch backend {
	case "s3":
		return s3.New(&s3.Config{
			Bucket:    bucket,
			Endpoint:  s3Endpoint,
			AccessKey: s3User,
			SecretKey: s3Pass,
			Insecure:  true,
		})
	case "gcs":
		return gcs.New(&gcs.Config{
			BucketName:      bucket,
			ChunkBufferSize: 10 * 1024 * 1024,
		})
	case "local":
		return local.New(&local.Config{
			Path: bucket,
		})
	default:
		return nil, nil, nil, fmt.Errorf("unknown backend %s", backend)
	}
}

func dumpBlock(r tempodb_backend.Reader, c tempodb_backend.Compactor, tenantID string, windowRange time.Duration, blockID string) error {
	id := uuid.MustParse(blockID)

	meta, err := r.BlockMeta(context.TODO(), id, tenantID)
	if err != nil {
		return err
	}

	compactedMeta, err := c.CompactedBlockMeta(id, tenantID)
	if err != nil && err != tempodb_backend.ErrMetaDoesNotExist {
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

	iter, err := encoding.NewBackendIterator(tenantID, id, 10*1024*1024, r)
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

func dumpBucket(r tempodb_backend.Reader, c tempodb_backend.Compactor, tenantID string, windowRange time.Duration) error {
	blockIDs, err := r.Blocks(context.Background(), tenantID)
	if err != nil {
		return err
	}

	fmt.Println("total blocks: ", len(blockIDs))

	totalObjects := 0
	out := make([][]string, 0)
	for _, id := range blockIDs {
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

func blockStats(meta *encoding.BlockMeta, compactedMeta *encoding.CompactedBlockMeta, windowRange time.Duration) (int, uint8, int64, time.Time, time.Time) {
	if meta != nil {
		return meta.TotalObjects, meta.CompactionLevel, meta.EndTime.Unix() / int64(windowRange/time.Second), meta.StartTime, meta.EndTime
	} else if compactedMeta != nil {
		return compactedMeta.TotalObjects, compactedMeta.CompactionLevel, compactedMeta.EndTime.Unix() / int64(windowRange/time.Second), compactedMeta.StartTime, compactedMeta.EndTime
	}

	return -1, 0, -1, time.Unix(0, 0), time.Unix(0, 0)
}
