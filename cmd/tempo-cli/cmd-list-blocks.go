package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/olekukonko/tablewriter"
)

type listBlocksCmd struct {
	TenantID         string `arg:"" help:"tenant-id within the bucket"`
	IncludeCompacted bool   `help:"include compacted blocks"`
	backendOptions
}

func (l *listBlocksCmd) Run(ctx *globalOptions) error {
	r, _, c, err := loadBackend(&l.backendOptions, ctx)
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

func displayResults(results []blockStats, windowDuration time.Duration, includeCompacted bool) {
	columns := []string{"id", "lvl", "objects", "size", "encoding", "vers", "window", "start", "end", "duration", "age"}
	if includeCompacted {
		columns = append(columns, "cmp")
	}

	totalObjects := 0
	totalBytes := uint64(0)

	out := make([][]string, 0)
	for _, r := range results {

		line := make([]string, 0)

		for _, c := range columns {
			s := ""
			switch c {
			case "id":
				s = r.BlockID.String()
			case "lvl":
				s = strconv.Itoa(int(r.CompactionLevel))
			case "objects":
				s = strconv.Itoa(int(r.TotalObjects))
			case "size":
				s = fmt.Sprintf("%v", humanize.Bytes(r.Size_))
			case "encoding":
				s = r.Encoding.String()
			case "vers":
				s = r.Version
			case "window":
				// Display compaction window in human-readable format
				window := time.Unix(r.window*int64(windowDuration.Seconds()), 0).UTC()
				s = window.Format(time.RFC3339)
			case "start":
				s = r.StartTime.Format(time.RFC3339)
			case "end":
				s = r.EndTime.Format(time.RFC3339)
			case "duration":
				// Time range included in bucket
				s = fmt.Sprint(r.EndTime.Sub(r.StartTime).Round(time.Second))
			case "age":
				s = fmt.Sprint(time.Since(r.EndTime).Round(time.Second))
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
		totalObjects += int(r.TotalObjects)
		totalBytes += r.Size_
	}

	footer := make([]string, 0)
	for _, c := range columns {
		switch c {
		case "objects":
			footer = append(footer, strconv.Itoa(totalObjects))
		case "size":
			footer = append(footer, fmt.Sprintf("%v", humanize.Bytes(totalBytes)))
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
