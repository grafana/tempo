package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/google/uuid"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type genIndexCmd struct {
	TenantID string `arg:"" help:"tenant-id within the bucket"`
	BlockID  string `arg:"" help:"block ID to list"`
	backendOptions
}

func ReplayBlockAndGetRecords(meta *backend.BlockMeta, filepath string) ([]common.Record, error, error) {
	v, err := encoding.FromVersion(meta.Version)
	if err != nil {
		return nil, nil, err
	}

	var warning error
	// replay file to extract records
	f, err := os.OpenFile(filepath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, nil, err
	}

	dataReader, err := v.NewDataReader(backend.NewContextReaderWithAllReader(f), meta.Encoding)
	if err != nil {
		return nil, nil, err
	}
	defer dataReader.Close()

	var buffer []byte
	var records []common.Record
	objectReader := v.NewObjectReaderWriter()
	currentOffset := uint64(0)
	for {
		buffer, pageLen, err := dataReader.NextPage(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			warning = err
			break
		}

		reader := bytes.NewReader(buffer)
		id, _, err := objectReader.UnmarshalObjectFromReader(reader)
		if err != nil {
			warning = err
			break
		}

		// make a copy so we don't hold onto the iterator buffer
		recordID := append([]byte(nil), id...)
		records = append(records, common.Record{
			ID:     recordID,
			Start:  currentOffset,
			Length: pageLen,
		})
		currentOffset += uint64(pageLen)
	}

	common.SortRecords(records)

	return records, warning, nil
}

func (cmd *genIndexCmd) Run(ctx *globalOptions) error {
	blockID, err := uuid.Parse(cmd.BlockID)
	if err != nil {
		return err
	}

	r, w, _, err := loadBackend(&cmd.backendOptions, ctx)
	if err != nil {
		return err
	}

	meta, err := r.BlockMeta(context.TODO(), blockID, cmd.TenantID)
	if err != nil {
		return err
	}

	// replay file to extract records
	records, warning, err := ReplayBlockAndGetRecords(meta, "./cmd/tempo-cli/test-data/"+cmd.TenantID+"/"+cmd.BlockID+"/data")
	if warning != nil || err != nil {
		fmt.Println("error replaying block", warning, err)
		return nil
	}

	// write using IndexWriter
	v, err := encoding.FromVersion(meta.Version)
	if err != nil {
		fmt.Println("error creating versioned encoding", err)
	}

	indexWriter := v.NewIndexWriter(int(meta.IndexPageSize))
	indexBytes, err := indexWriter.Write(records)
	if err != nil {
		fmt.Println("error writing records to indexWriter", err)
	}

	// write to the local backend
	err = w.Write(context.TODO(), "index", blockID, cmd.TenantID, indexBytes, false)
	if err != nil {
		fmt.Println("error writing index to backend", err)
	}

	fmt.Println("index written to backend successfully")

	return nil
}
