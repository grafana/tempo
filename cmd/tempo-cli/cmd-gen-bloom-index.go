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

const (
	dataFilename = "data"
	indexFilename = "index"
)

type indexCmd struct {
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
	objectRW := v.NewObjectReaderWriter()
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

		iter := encoding.NewIterator(bytes.NewReader(buffer), objectRW)
		var lastID common.ID
		var iterErr error
		for {
			var id common.ID
			id, _, iterErr = iter.Next(context.TODO())
			if iterErr != nil {
				break
			}
			lastID = id
		}

		if iterErr != io.EOF {
			warning = iterErr
			break
		}

		// make a copy so we don't hold onto the iterator buffer
		recordID := append([]byte(nil), lastID...)
		records = append(records, common.Record{
			ID:     recordID,
			Start:  currentOffset,
			Length: pageLen,
		})
		currentOffset += uint64(pageLen)
	}

	return records, warning, nil
}

func VerifyIndex(indexReader common.IndexReader, dataReader common.DataReader) error {
	for i := 0; ; i++ {
		record, err := indexReader.At(context.TODO(), i)
		if err != nil {
			return err
		}

		if record == nil {
			break
		}

		// read data file at record position
		_, _, err = dataReader.Read(context.TODO(), []common.Record{*record}, nil)
		if err != nil {
			fmt.Println("index/data is corrupt, record/data mismatch")
			return err
		}
	}
}

func (cmd *indexCmd) Run(ctx *globalOptions) error {
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
	records, warning, err := ReplayBlockAndGetRecords(meta, cmd.backendOptions.Bucket+cmd.TenantID+"/"+cmd.BlockID+"/"+dataFilename)
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

	// verify generated index

	// get index file with records
	indexFilePath := cmd.backendOptions.Bucket+cmd.TenantID+"/"+cmd.BlockID+"/"+indexFilename
	indexFile, err := os.OpenFile(indexFilePath, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}

	indexReader, err := v.NewIndexReader(backend.NewContextReaderWithAllReader(indexFile), int(meta.IndexPageSize), len(records))
	if err != nil {
		return err
	}

	// data reader
	dataFilePath := cmd.backendOptions.Bucket+cmd.TenantID+"/"+cmd.BlockID+"/"+dataFilename
	dataFile, err := os.OpenFile(dataFilePath, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}

	dataReader, err := v.NewDataReader(backend.NewContextReaderWithAllReader(dataFile), meta.Encoding)
	if err != nil {
		return err
	}
	defer dataReader.Close()

	err = VerifyIndex(indexReader, dataReader)
	if err != nil {
		return err
	}

	fmt.Println("index verified!")
	return nil
}
