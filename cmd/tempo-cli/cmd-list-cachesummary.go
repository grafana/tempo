package main

import (
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
)

type listCacheSummaryCmd struct {
	TenantID string `arg:"" help:"tenant-id within the bucket"`
	backendOptions
}

func (l *listCacheSummaryCmd) Run(ctx *globalOptions) error {
	r, _, c, err := loadBackend(&l.backendOptions, ctx)
	if err != nil {
		return err
	}

	windowDuration := time.Hour

	results, err := loadBucket(r, c, l.TenantID, windowDuration, false)
	if err != nil {
		return err
	}

	displayCacheSummary(results)

	return nil
}

func displayCacheSummary(results []blockStats) {
	fmt.Println()
	fmt.Println("Bloom filter shards by day and compaction level:")

	columns := []string{"bloom filter age"}
	out := make([][]string, 0)
	bloomTable := make([][]int, 0)

	for _, r := range results {
		row := r.CompactionLevel
		// extend rows
		for len(bloomTable)-1 < int(row) {
			bloomTable = append(bloomTable, make([]int, 0))
		}
		column := -1 * (int(time.Until(r.StartTime) / (time.Hour * 24)))
		// extend column of given row
		for len(bloomTable[row])-1 < column {
			bloomTable[row] = append(bloomTable[row], 0)
		}
		// extend columns (header of bloomTable)
		for i := len(columns) - 1; i <= column; i++ {
			columns = append(columns, fmt.Sprintf("%d days", i))
		}

		if int(row) < len(bloomTable) && column < len(bloomTable[row]) {
			bloomTable[row][column] += int(r.BloomShardCount)
		} else {
			fmt.Println("something wrong with row / column", row, column)
		}
	}

	fmt.Println()
	columnTotals := make([]int, len(columns)-1)
	for i, row := range bloomTable {
		line := make([]string, 0)
		line = append(line, fmt.Sprintf("compaction level %d", i))

		for j, column := range row {
			line = append(line, fmt.Sprintf("%d", column))
			columnTotals[j] += column
		}
		out = append(out, line)
	}

	columnTotalsRow := make([]string, 0, len(columns))
	columnTotalsRow = append(columnTotalsRow, "total")
	for _, total := range columnTotals {
		columnTotalsRow = append(columnTotalsRow, fmt.Sprintf("%d", total))
	}

	fmt.Println()
	w := tablewriter.NewWriter(os.Stdout)
	w.SetHeader(columns)
	w.AppendBulk(out)
	w.SetFooter(columnTotalsRow)
	w.Render()
}
