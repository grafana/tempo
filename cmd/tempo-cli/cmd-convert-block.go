package main

import (
	"fmt"
	"os"

	"github.com/grafana/tempo/tempodb/encoding/vparquet"
	"github.com/parquet-go/parquet-go"
)

type convertParquet struct {
	In  string `arg:"" help:"The input parquet file to read from"`
	Out string `arg:"" help:"The output parquet file to write to"`
}

func (cmd *convertParquet) Run() error {
	// open In
	fIn, err := os.Open(cmd.In)
	if err != nil {
		return err
	}
	s, err := fIn.Stat()
	if err != nil {
		return err
	}
	pf, err := parquet.OpenFile(fIn, s.Size())
	if err != nil {
		return err
	}

	// open Out
	fOut, err := os.Create(cmd.Out)
	if err != nil {
		return err
	}
	sch := parquet.SchemaOf(new(vparquet.Trace))
	writer := parquet.NewWriter(fOut, sch)

	conversion, err := parquet.Convert(sch, pf.Schema())
	if err != nil {
		return err
	}

	// copy a rowgroup at a time
	rgs := pf.RowGroups()
	fmt.Println("Total Rowgroups: ", len(rgs))
	for i, rg := range rgs {
		fmt.Println("Converting ", i+1)
		rg = parquet.ConvertRowGroup(rg, conversion)

		_, err = writer.WriteRowGroup(rg)
		if err != nil {
			return err
		}
		err = writer.Flush()
		if err != nil {
			return err
		}
	}

	return writer.Close()
}
