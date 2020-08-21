package wal

import (
	"os"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/willf/bloom"
)

type HeadBlock struct {
	block

	appendFile *os.File
	appender   backend.Appender
}

func newHeadBlock(id uuid.UUID, tenantID string, filepath string) (*HeadBlock, error) {
	h := &HeadBlock{
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

func (h *HeadBlock) Write(id encoding.ID, b []byte) error {
	err := h.appender.Append(id, b)
	if err != nil {
		return err
	}
	h.meta.ObjectAdded(id)
	return nil
}

func (h *HeadBlock) Length() int {
	return h.appender.Length()
}

func (h *HeadBlock) Complete(w *WAL, combiner encoding.ObjectCombiner) (*CompleteBlock, error) {
	if h.appendFile != nil {
		err := h.appendFile.Close()
		if err != nil {
			return nil, err
		}
	}

	walConfig := w.config()

	// 1) create a new block in work dir
	// 2) append all objects from this block in order
	// 3) move from workdir -> realdir
	// 4) remove old
	records := h.appender.Records()
	orderedBlock := &CompleteBlock{
		block: block{
			meta:     encoding.NewBlockMeta(h.meta.TenantID, uuid.New()),
			filepath: walConfig.WorkFilepath,
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
		return nil, err
	}

	iterator := encoding.NewRecordIterator(records, readFile)
	iterator, err = encoding.NewDedupingIterator(iterator, combiner)
	if err != nil {
		return nil, err
	}
	appender := encoding.NewBufferedAppender(appendFile, walConfig.IndexDownsample, len(records))
	for {
		bytesID, bytesObject, err := iterator.Next()
		if bytesID == nil {
			break
		}
		if err != nil {
			return nil, err
		}

		orderedBlock.bloom.Add(bytesID)
		// obj gets written to disk immediately but the id escapes the iterator and needs to be copied
		writeID := append([]byte(nil), bytesID...)
		err = appender.Append(writeID, bytesObject)
		if err != nil {
			return nil, err
		}
	}
	appender.Complete()
	appendFile.Close()
	orderedBlock.records = appender.Records()

	workFilename := orderedBlock.fullFilename()
	orderedBlock.filepath = h.filepath
	completeFilename := orderedBlock.fullFilename()

	err = os.Rename(workFilename, completeFilename)
	if err != nil {
		return nil, err
	}

	os.Remove(h.fullFilename())
	if err != nil {
		return nil, err
	}

	return orderedBlock, nil
}

func (h *HeadBlock) Find(id encoding.ID, combiner encoding.ObjectCombiner) ([]byte, error) {
	records := h.appender.Records()
	file, err := h.file()
	if err != nil {
		return nil, err
	}

	finder := encoding.NewDedupingFinder(records, file, combiner)

	return finder.Find(id)
}
