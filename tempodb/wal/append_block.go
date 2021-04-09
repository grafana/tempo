package wal

import (
	"context"
	"os"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// AppendBlock is a block that is actively used to append new objects to.  It stores all data in the appendFile
// in the order it was received and an in memory sorted index.
type AppendBlock struct {
	block
	encoding encoding.VersionedEncoding

	appendFile *os.File
	appender   encoding.Appender
}

func newAppendBlock(id uuid.UUID, tenantID string, filepath string, e backend.Encoding) (*AppendBlock, error) {
	v, err := encoding.EncodingByVersion("v2") // let's pin wal files instead of tracking latest for safety
	if err != nil {
		return nil, err
	}

	h := &AppendBlock{
		encoding: v,
		block: block{
			meta:     backend.NewBlockMeta(tenantID, id, v.Version(), e),
			filepath: filepath,
		},
	}

	name := h.fullFilename()

	f, err := os.OpenFile(name, os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	h.appendFile = f

	dataWriter, err := h.encoding.NewDataWriter(f, e)
	if err != nil {
		return nil, err
	}

	h.appender = encoding.NewAppender(dataWriter)

	return h, nil
}

func (h *AppendBlock) Write(id common.ID, b []byte) error {
	err := h.appender.Append(id, b)
	if err != nil {
		return err
	}
	h.meta.ObjectAdded(id)
	return nil
}

func (h *AppendBlock) BlockID() uuid.UUID {
	return h.block.meta.BlockID
}

func (h *AppendBlock) DataLength() uint64 {
	return h.appender.DataLength()
}

func (h *AppendBlock) Meta() *backend.BlockMeta {
	return h.meta
}

func (h *AppendBlock) GetIterator(combiner common.ObjectCombiner) (encoding.Iterator, error) {
	if h.appendFile != nil {
		err := h.appendFile.Close()
		if err != nil {
			return nil, err
		}
		h.appendFile = nil
	}

	records := h.appender.Records()
	readFile, err := h.file()
	if err != nil {
		return nil, err
	}

	dataReader, err := h.encoding.NewDataReader(backend.NewContextReaderWithAllReader(readFile), h.meta.Encoding)
	if err != nil {
		return nil, err
	}

	iterator := encoding.NewRecordIterator(records, dataReader, h.encoding.NewObjectReaderWriter())
	iterator, err = encoding.NewDedupingIterator(iterator, combiner)
	if err != nil {
		return nil, err
	}

	return iterator, nil
}

func (h *AppendBlock) Find(id common.ID, combiner common.ObjectCombiner) ([]byte, error) {
	records := h.appender.Records()
	file, err := h.file()
	if err != nil {
		return nil, err
	}

	dataReader, err := h.encoding.NewDataReader(backend.NewContextReaderWithAllReader(file), h.meta.Encoding)
	if err != nil {
		return nil, err
	}
	defer dataReader.Close()
	finder := encoding.NewPagedFinder(common.Records(records), dataReader, combiner, h.encoding.NewObjectReaderWriter())

	return finder.Find(context.Background(), id)
}

func (h *AppendBlock) Clear() error {
	if h.readFile != nil {
		_ = h.readFile.Close()
		h.readFile = nil
	}

	if h.appendFile != nil {
		_ = h.appendFile.Close()
		h.appendFile = nil
	}

	name := h.fullFilename()
	return os.Remove(name)
}
