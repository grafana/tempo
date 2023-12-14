package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/parquet-go/parquet-go"

	pq "github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/tempodb/encoding/vparquet"
)

type listColumnCmd struct {
	backendOptions

	TenantID string `arg:"" help:"tenant-id within the bucket"`
	BlockID  string `arg:"" help:"block ID to list"`
	Column   string `arg:"TraceID" help:"column name to list values of"`
}

func (cmd *listColumnCmd) Run(ctx *globalOptions) error {
	r, _, _, err := loadBackend(&cmd.backendOptions, ctx)
	if err != nil {
		return err
	}

	meta, err := r.BlockMeta(context.TODO(), uuid.MustParse(cmd.BlockID), cmd.TenantID)
	if err != nil {
		return err
	}

	rr := vparquet.NewBackendReaderAt(context.Background(), r, vparquet.DataFileName, meta)
	pf, err := parquet.OpenFile(rr, int64(meta.Size))
	if err != nil {
		return err
	}

	colIndex, _ := pq.GetColumnIndexByPath(pf, cmd.Column)

	for i, rg := range pf.RowGroups() {

		// choose the column mentioned in the cli param
		cc := rg.ColumnChunks()[colIndex]

		fmt.Printf("\n***************       rowgroup %d      ********************\n\n\n", i)

		pages := cc.Pages()
		idx, err := cc.ColumnIndex()
		if err != nil {
			return err
		}
		numPages := idx.NumPages()
		fmt.Println("Min Value of rowgroup", idx.MinValue(0).Bytes())
		fmt.Println("Max Value of rowgroup", idx.MaxValue(numPages-1).Bytes())

		buffer := make([]parquet.Value, 10000)
		for {
			pg, err := pages.ReadPage()
			if pg == nil || errors.Is(err, io.EOF) {
				break
			}

			vr := pg.Values()
			for {
				x, err := vr.ReadValues(buffer)
				for y := 0; y < x; y++ {
					fmt.Println(buffer[y].Bytes())
				}

				// check for EOF after processing any returned data
				if errors.Is(err, io.EOF) {
					break
				}
				// todo: better error handling
				if err != nil {
					break
				}
			}
		}
	}

	return nil
}
