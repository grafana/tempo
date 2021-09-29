package wal

import (
	"bytes"
	"io"
	"os"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// ReplayWALAndGetRecords replays a WAL file that could contain either traces or searchdata
//nolint:interfacer
func ReplayWALAndGetRecords(file *os.File, v encoding.VersionedEncoding, enc backend.Encoding) ([]common.Record, error, error) {
	dataReader, err := v.NewDataReader(backend.NewContextReaderWithAllReader(file), enc)
	if err != nil {
		return nil, nil, err
	}

	var buffer []byte
	var records []common.Record
	var warning error
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
		// wal should only ever have one object per page, test that here
		_, _, err = objectReader.UnmarshalObjectFromReader(reader)
		if err != io.EOF {
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
