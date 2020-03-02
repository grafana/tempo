package wal

import (
	"fmt"
	"os"

	"github.com/grafana/frigg/friggdb/backend"
)

type block struct {
	meta     *backend.BlockMeta
	filepath string
	readFile *os.File
}

func (b *block) fullFilename() string {
	return fmt.Sprintf("%s/%v:%v", b.filepath, b.meta.BlockID, b.meta.TenantID)
}

func (b *block) readRecordBytes(r *backend.Record) ([]byte, error) { // jpe?  belongs in backend? goes away when find is done
	file, err := b.file()
	if err != nil {
		return nil, err
	}

	buff := make([]byte, r.Length)
	_, err = file.ReadAt(buff, int64(r.Start))
	if err != nil {
		return nil, err
	}

	return buff, nil
}

func (b *block) file() (*os.File, error) {
	if b.readFile == nil {
		name := b.fullFilename()

		f, err := os.OpenFile(name, os.O_RDONLY, 0644)
		if err != nil {
			return nil, err
		}
		b.readFile = f
	}

	return b.readFile, nil
}
