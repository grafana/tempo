package v2

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/grafana/tempo/tempodb/backend"
)

// ReplayWALAndGetRecords replays a WAL file that could contain either traces or searchdata
func ReplayWALAndGetRecords(file *os.File, enc backend.Encoding, handleObj func([]byte) error) ([]Record, error, error) {
	dataReader, err := NewDataReader(backend.NewContextReaderWithAllReader(file), enc)
	if err != nil {
		return nil, nil, err
	}

	var buffer []byte
	var records []Record
	var warning error
	var pageLen uint32
	var id []byte
	objectReader := NewObjectReaderWriter()
	currentOffset := uint64(0)
	for {
		buffer, pageLen, err = dataReader.NextPage(buffer)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			warning = fmt.Errorf("accessing NextPage while replaying wal: %w", err)
			break
		}

		reader := bytes.NewReader(buffer)
		id, buffer, err = objectReader.UnmarshalObjectFromReader(reader)
		if err != nil {
			warning = fmt.Errorf("unmarshalling object while replaying wal: %w", err)
			break
		}
		// wal should only ever have one object per page, test that here
		_, _, err = objectReader.UnmarshalObjectFromReader(reader)
		if !errors.Is(err, io.EOF) {
			warning = fmt.Errorf("expected EOF while replaying wal: %w", err)
			break
		}

		// handleObj is primarily used by search replay to record search data in block header
		err = handleObj(buffer)
		if err != nil {
			warning = fmt.Errorf("custom obj handler while replaying wal: %w", err)
			break
		}

		// make a copy so we don't hold onto the iterator buffer
		recordID := append([]byte(nil), id...)
		records = append(records, Record{
			ID:     recordID,
			Start:  currentOffset,
			Length: pageLen,
		})
		currentOffset += uint64(pageLen)
	}

	SortRecords(records)

	return records, warning, nil
}
