package wal

import (
	"os"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
)

// AppendBlock is a block that is actively used to append new objects to.  It stores all data in the appendFile
// in the order it was received and an in memory sorted index.
type AppendBlock struct {
	block

	appendFile *os.File
	appender   common.Appender
}

func newAppendBlock(id uuid.UUID, tenantID string, filepath string) (*AppendBlock, error) {
	h := &AppendBlock{
		block: block{
			meta:     backend.NewBlockMeta(tenantID, id),
			filepath: filepath,
		},
	}

	name := h.fullFilename()
	_, err := os.Create(name)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(name, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	h.appendFile = f
	h.appender = v0.NewAppender(f)

	return h, nil
}

// NewAppendBlockFromWal is a method of creating an append block from a wal file.  It
//  is intended to be used on startup by the ingester to rapidly "replay" the wal.
func NewAppendBlockFromWal(walfile string) (*AppendBlock, error) {
	blockID, tenantID, err := parseFilename(walfile)
	if err != nil {
		return nil, err
	}

	// load the walfile and rebuild the index
	readFile, err := os.OpenFile(walfile, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	iterator := encoding.NewIterator(readFile)

	for {
		id, obj, err := iterator.Next()
		if id == nil {
			break
		}
		if err != nil {
			return nil, err
		}

		// obj gets written to disk immediately but the id escapes the iterator and needs to be copied
		writeID := append([]byte(nil), id...)
		// jpe - write to an appender?
		// err = instance.PushBytes(context.Background(), writeID, obj)
		if err != nil {
			return nil, err
		}
	}
	readFile.Close()

	// actually build the block
	b := &AppendBlock{
		block: block{
			meta:     backend.NewBlockMeta(tenantID, blockID),
			filepath: walfile,
		},
	}
	appendFile, err := os.OpenFile(walfile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	b.appendFile = appendFile
	b.appender = encoding.NewAppender(appendFile)
}

// Write adds an id and object to this AppendBlock
func (h *AppendBlock) Write(id common.ID, b []byte) error {
	err := h.appender.Append(id, b)
	if err != nil {
		return err
	}
	h.meta.ObjectAdded(id)
	return nil
}

// Length() indicates how many objects are currently stored by the block
func (h *AppendBlock) Length() int {
	return h.appender.Length()
}

// Complete should be called when you are done with the block.  This method will write and return a new CompleteBlock which
// includes an on disk file containing all objects in order.
// Note that calling this method leaves the original file on disk.  This file is still considered to be part of the WAL
// until Write() is successfully called on the CompleteBlock.
func (h *AppendBlock) Complete(w *WAL, combiner common.ObjectCombiner) (*encoding.CompleteBlock, error) {
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

	iterator := v0.NewRecordIterator(records, readFile)
	iterator, err = v0.NewDedupingIterator(iterator, combiner)
	if err != nil {
		return nil, err
	}

	orderedBlock, err := encoding.NewCompleteBlock(h.meta, iterator, w.c.BloomFP, len(records), w.c.IndexDownsample, w.c.CompletedFilepath, h.fullFilename())
	if err != nil {
		return nil, err
	}

	return orderedBlock, nil
}

// Find searches for a given id in this AppendBlock
func (h *AppendBlock) Find(id encoding.ID, combiner encoding.ObjectCombiner) ([]byte, error) {
	records := h.appender.Records()
	file, err := h.file()
	if err != nil {
		return nil, err
	}

	finder := v0.NewDedupingFinder(records, file, combiner)

	return finder.Find(id)
}

// Clear removes the backing wal file for this AppendBlock
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
