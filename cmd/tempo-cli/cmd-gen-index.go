package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/google/uuid"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v2 "github.com/grafana/tempo/tempodb/encoding/v2"
)

type indexCmd struct {
	TenantID string `arg:"" help:"tenant-id within the bucket"`
	BlockID  string `arg:"" help:"block ID to list"`
	backendOptions
}

func ReplayBlockAndGetRecords(meta *backend.BlockMeta, filepath string) ([]v2.Record, error, error) {
	var replayError error
	// replay file to extract records
	f, err := os.OpenFile(filepath, os.O_RDONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}

	dataReader, err := v2.NewDataReader(backend.NewContextReaderWithAllReader(f), meta.Encoding)
	if err != nil {
		return nil, nil, err
	}
	defer dataReader.Close()

	var buffer []byte
	var records []v2.Record
	objectRW := v2.NewObjectReaderWriter()
	currentOffset := uint64(0)
	for {
		buffer, pageLen, err := dataReader.NextPage(buffer)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			replayError = err
			break
		}

		iter := v2.NewIterator(bytes.NewReader(buffer), objectRW)
		var lastID common.ID
		var iterErr error
		for {
			var id common.ID
			id, _, iterErr = iter.NextBytes(context.TODO())
			if iterErr != nil {
				break
			}
			lastID = id
		}

		if !errors.Is(iterErr, io.EOF) {
			replayError = iterErr
			break
		}

		// make a copy so we don't hold onto the iterator buffer
		recordID := append([]byte(nil), lastID...)
		records = append(records, v2.Record{
			ID:     recordID,
			Start:  currentOffset,
			Length: pageLen,
		})
		currentOffset += uint64(pageLen)
	}

	return records, replayError, nil
}

func VerifyIndex(indexReader v2.IndexReader, dataReader v2.DataReader) error {
	for i := 0; ; i++ {
		record, err := indexReader.At(context.TODO(), i)
		if err != nil {
			return err
		}

		if record == nil {
			break
		}

		// read data file at record position
		_, _, err = dataReader.Read(context.TODO(), []v2.Record{*record}, nil, nil)
		if err != nil {
			fmt.Println("index/data is corrupt, record/data mismatch")
			return err
		}
	}
	return nil
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

	if meta.Version != v2.VersionString {
		return fmt.Errorf("unsupported block version: %s", meta.Version)
	}

	// replay file to extract records
	records, replayError, err := ReplayBlockAndGetRecords(meta, cmd.backendOptions.Bucket+cmd.TenantID+"/"+cmd.BlockID+"/"+dataFilename)
	if replayError != nil {
		fmt.Println("error replaying block. data file likely corrupt", replayError)
		return replayError
	}
	if err != nil {
		fmt.Println("error accessing data/meta file")
		return err
	}

	// write using IndexWriter
	indexWriter := v2.NewIndexWriter(int(meta.IndexPageSize))
	indexBytes, err := indexWriter.Write(records)
	if err != nil {
		fmt.Println("error writing records to indexWriter", err)
		return err
	}

	// write to the local backend
	err = w.Write(context.TODO(), "index", blockID, cmd.TenantID, indexBytes, nil)
	if err != nil {
		fmt.Println("error writing index to backend", err)
		return err
	}

	fmt.Println("index written to backend successfully")

	// verify generated index

	// get index file with records
	indexFilePath := cmd.backendOptions.Bucket + cmd.TenantID + "/" + cmd.BlockID + "/" + indexFilename
	indexFile, err := os.OpenFile(indexFilePath, os.O_RDONLY, 0o644)
	if err != nil {
		fmt.Println("error opening index file")
		return err
	}

	indexReader, err := v2.NewIndexReader(backend.NewContextReaderWithAllReader(indexFile), int(meta.IndexPageSize), len(records))
	if err != nil {
		fmt.Println("error reading index file")
		return err
	}

	// data reader
	dataFilePath := cmd.backendOptions.Bucket + cmd.TenantID + "/" + cmd.BlockID + "/" + dataFilename
	dataFile, err := os.OpenFile(dataFilePath, os.O_RDONLY, 0o644)
	if err != nil {
		fmt.Println("error opening data file")
		return err
	}

	dataReader, err := v2.NewDataReader(backend.NewContextReaderWithAllReader(dataFile), meta.Encoding)
	if err != nil {
		fmt.Println("error reading data file")
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
