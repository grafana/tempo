package wal

import (
	"context"
	"os"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
)

// these values should never be used.  these dummy values can help detect
// if they leak elsewhere.  append blocks are not versioned and do not
// support different encodings
const appendBlockVersion = "append"
const appendBlockEncoding = backend.EncNone

// AppendBlock is a block that is actively used to append new objects to.  It stores all data in the appendFile
// in the order it was received and an in memory sorted index.
type AppendBlock struct {
	block

	appendFile *os.File
	appender   encoding.Appender
}

func newAppendBlock(id uuid.UUID, tenantID string, filepath string) (*AppendBlock, error) {
	h := &AppendBlock{
		block: block{
			meta:     backend.NewBlockMeta(tenantID, id, appendBlockVersion, appendBlockEncoding),
			filepath: filepath,
		},
	}

	name := h.fullFilename()

	f, err := os.OpenFile(name, os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	h.appendFile = f
	h.appender = encoding.NewAppender(v0.NewPageWriter(f))

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

// Complete should be called when you are done with the block.  This method will write and return a new CompleteBlock which
// includes an on disk file containing all objects in order.
// Note that calling this method leaves the original file on disk.  This file is still considered to be part of the WAL
// until Write() is successfully called on the CompleteBlock.
func (h *AppendBlock) Complete(cfg *encoding.BlockConfig, w *WAL, combiner common.ObjectCombiner) (*encoding.CompleteBlock, error) {
	if h.appendFile != nil {
		err := h.appendFile.Close()
		if err != nil {
			return nil, err
		}
	}

	records := h.appender.Records()
	readFile, err := h.file()
	if err != nil {
		return nil, err
	}

	iterator := encoding.NewRecordIterator(records, readFile)
	iterator, err = encoding.NewDedupingIterator(iterator, combiner)
	if err != nil {
		return nil, err
	}
	defer iterator.Close()

	orderedBlock, err := encoding.NewCompleteBlock(cfg, h.meta, iterator, len(records), w.c.CompletedFilepath)
	if err != nil {
		return nil, err
	}

	return orderedBlock, nil
}

func (h *AppendBlock) Find(id common.ID, combiner common.ObjectCombiner) ([]byte, error) {
	records := h.appender.Records()
	file, err := h.file()
	if err != nil {
		return nil, err
	}

	pageReader := v0.NewPageReader(backend.NewContextReaderWithAllReader(file))
	defer pageReader.Close()
	finder := encoding.NewPagedFinder(common.Records(records), pageReader, combiner)

	return finder.Find(context.Background(), id)
}

func (h *AppendBlock) Clear() error {
	if h.readFile != nil {
		_ = h.readFile.Close()
	}

	if h.appendFile != nil {
		_ = h.appendFile.Close()
	}

	name := h.fullFilename()
	return os.Remove(name)
}
