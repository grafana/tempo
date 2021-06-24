package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/olekukonko/tablewriter"
	"github.com/bsipos/thist"
)

type listCompactionSummaryCmd struct {
	TenantID string `arg:"" help:"tenant-id within the bucket"`
	backendOptions
}

func (l *listCompactionSummaryCmd) Run(ctx *globalOptions) error {
	r, c, err := loadBackend(&l.backendOptions, ctx)
	if err != nil {
		return err
	}

	windowDuration := time.Hour

	results, err := loadBucket(r, c, l.TenantID, windowDuration, false)
	if err != nil {
		return err
	}

	displayCompactionSummary(results)

	return nil
}

func displayCompactionSummary(results []blockStats) {
	fmt.Println()
	fmt.Println("Stats by compaction level:")
	resultsByLevel := make(map[int][]blockStats)
	var levels []int
	for _, r := range results {
		l := int(r.CompactionLevel)

		s, ok := resultsByLevel[l]
		if !ok {
			s = make([]blockStats, 0)
			levels = append(levels, l)
		}

		s = append(s, r)
		resultsByLevel[l] = s
	}

	sort.Ints(levels)

	columns := []string{"lvl", "blocks", "total", "smallest block", "largest block", "earliest", "latest", "bloom shard count"}

	out := make([][]string, 0)

	for _, l := range levels {
		sizeSum := uint64(0)
		sizeMin := uint64(0)
		sizeMax := uint64(0)
		countSum := 0
		countMin := 0
		countMax := 0
		countBloomShards := 0
		hist := thist.NewHist(nil, "level " + strconv.Itoa(l), "auto", -1, true)

		var newest time.Time
		var oldest time.Time
		for _, r := range resultsByLevel[l] {
			sizeSum += r.Size
			countSum += r.TotalObjects
			countBloomShards += int(r.BloomShardCount)
			hist.Update(float64(r.Size))

			if r.Size < sizeMin || sizeMin == 0 {
				sizeMin = r.Size
			}
			if r.Size > sizeMax {
				sizeMax = r.Size
			}
			if r.TotalObjects < countMin || countMin == 0 {
				countMin = r.TotalObjects
			}
			if r.TotalObjects > countMax {
				countMax = r.TotalObjects
			}
			if r.StartTime.Before(oldest) || oldest.IsZero() {
				oldest = r.StartTime
			}
			if r.EndTime.After(newest) {
				newest = r.EndTime
			}
		}

		fmt.Println(hist.Draw())

		line := make([]string, 0)

		for _, c := range columns {
			s := ""
			switch c {
			case "lvl":
				s = strconv.Itoa(l)
			case "blocks":
				s = fmt.Sprintf("%d (%d %%)", len(resultsByLevel[l]), len(resultsByLevel[l])*100/len(results))
			case "total":
				s = fmt.Sprintf("%s objects (%s)", humanize.Comma(int64(countSum)), humanize.Bytes(sizeSum))
			case "smallest block":
				s = fmt.Sprintf("%s objects (%s)", humanize.Comma(int64(countMin)), humanize.Bytes(sizeMin))
			case "largest block":
				s = fmt.Sprintf("%s objects (%s)", humanize.Comma(int64(countMax)), humanize.Bytes(sizeMax))
			case "earliest":
				s = fmt.Sprint(time.Since(oldest).Round(time.Second), " ago")
			case "latest":
				s = fmt.Sprint(time.Since(newest).Round(time.Second), " ago")
			case "bloom shard count":
				s = fmt.Sprint(countBloomShards)
			}
			line = append(line, s)
		}
		out = append(out, line)
	}

	fmt.Println()
	w := tablewriter.NewWriter(os.Stdout)
	w.SetHeader(columns)
	w.AppendBulk(out)
	w.Render()
}
