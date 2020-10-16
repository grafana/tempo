package wal

import (
	"os"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/bloom"
)

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
			meta:     encoding.NewBlockMeta(tenantID, id),
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
	h.appender = encoding.NewAppender(f)

	return h, nil
}

func (h *AppendBlock) Write(id encoding.ID, b []byte) error {
	err := h.appender.Append(id, b)
	if err != nil {
		return err
	}
	h.meta.ObjectAdded(id)
	return nil
}

func (h *AppendBlock) Length() int {
	return h.appender.Length()
}

// Complete should be called when you are done with the block.  This method will write and return a new CompleteBlock which
// includes an on disk file containing all objects in order.
// Note that calling this method leaves the original file on disk.  This file is still considered to be part of the WAL
// until Flush() is successfully called on the CompleteBlock.
func (h *AppendBlock) Complete(w *WAL, combiner encoding.ObjectCombiner) (*CompleteBlock, error) {
	if h.appendFile != nil {
		err := h.appendFile.Close()
		if err != nil {
			return nil, err
		}
	}

	walConfig := w.config()

	// 1) create a new block in work dir
	// 2) append all objects from this block in order
	// 3) move from completeddir -> realdir
	// 4) remove old
	records := h.appender.Records()
	orderedBlock := &CompleteBlock{
		block: block{
			meta:     encoding.NewBlockMeta(h.meta.TenantID, uuid.New()),
			filepath: walConfig.CompletedFilepath,
		},
		bloom: bloom.NewWithEstimates(uint(len(records)), walConfig.BloomFP),
	}
	orderedBlock.meta.StartTime = h.meta.StartTime
	orderedBlock.meta.EndTime = h.meta.EndTime
	orderedBlock.meta.MinID = h.meta.MinID
	orderedBlock.meta.MaxID = h.meta.MaxID
	orderedBlock.meta.TotalObjects = h.meta.TotalObjects

	_, err := os.Create(orderedBlock.fullFilename())
	if err != nil {
		return nil, err
	}

	// records are already sorted
	appendFile, err := os.OpenFile(orderedBlock.fullFilename(), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	readFile, err := h.file()
	if err != nil {
		_ = appendFile.Close()
		_ = os.Remove(orderedBlock.fullFilename())
		return nil, err
	}

	iterator := encoding.NewRecordIterator(records, readFile)
	iterator, err = encoding.NewDedupingIterator(iterator, combiner)
	if err != nil {
		_ = appendFile.Close()
		_ = os.Remove(orderedBlock.fullFilename())
		return nil, err
	}
	appender := encoding.NewBufferedAppender(appendFile, walConfig.IndexDownsample, len(records))
	for {
		bytesID, bytesObject, err := iterator.Next()
		if bytesID == nil {
			break
		}
		if err != nil {
			_ = appendFile.Close()
			_ = os.Remove(orderedBlock.fullFilename())
			return nil, err
		}

		orderedBlock.bloom.Add(bytesID)
		// obj gets written to disk immediately but the id escapes the iterator and needs to be copied
		writeID := append([]byte(nil), bytesID...)
		err = appender.Append(writeID, bytesObject)
		if err != nil {
			_ = appendFile.Close()
			_ = os.Remove(orderedBlock.fullFilename())
			return nil, err
		}
	}
	appender.Complete()
	appendFile.Close()
	orderedBlock.records = appender.Records()
	orderedBlock.walFilename = h.fullFilename() // pass the filename to the complete block for cleanup when it's flusehd

	return orderedBlock, nil
}

func (h *AppendBlock) Find(id encoding.ID, combiner encoding.ObjectCombiner) ([]byte, error) {
	records := h.appender.Records()
	file, err := h.file()
	if err != nil {
		return nil, err
	}

	finder := encoding.NewDedupingFinder(records, file, combiner)

	return finder.Find(id)
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
