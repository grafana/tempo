package wal

import (
	"os"

	bloom "github.com/dgraph-io/ristretto/z"
	"github.com/dgryski/go-farm"
	"github.com/google/uuid"
	"github.com/grafana/frigg/friggdb/backend"
)

// Appendlock
//  wal.CompleteBlock(AppendBlock)
//    creates AppendBlock with Buffered Appender and iterates through "HeadBlock" and appends
//    returns as CompleteBlock
// CompactorBlock is append block with bufferedappender?  how is bloom handled?

// jpe: get rid of unnecessary block interfaces
type HeadBlock interface { // jpe HeadBlock => AppendBlock.  takes appender factory?  CompactorBlock becomes AppendBlock with different buffered appender?
	Write(id backend.ID, b []byte) error
	Complete(w *WAL) (CompleteBlock, error)
	Length() int
	Find(id backend.ID) ([]byte, error)
}

type headBlock struct {
	block

	appendFile *os.File
	appender   backend.Appender
}

func newHeadBlock(id uuid.UUID, tenantID string, filepath string) (*headBlock, error) {
	h := &headBlock{
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
	h.appender = backend.NewAppender(f)

	return h, nil
}

func (h *headBlock) Write(id backend.ID, b []byte) error {
	err := h.appender.Append(id, b)
	if err != nil {
		return err
	}
	h.meta.ObjectAdded(id)
	return nil
}

func (h *headBlock) Length() int {
	return h.appender.Length()
}

func (h *headBlock) Complete(w *WAL) (CompleteBlock, error) {
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
	orderedBlock := &completeBlock{
		block: block{
			meta:     backend.NewBlockMeta(h.meta.TenantID, uuid.New()),
			filepath: walConfig.WorkFilepath,
		},
		bloom: bloom.NewBloomFilter(float64(len(records)), walConfig.BloomFP),
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

	iterator := backend.NewRecordIterator(records, readFile)
	appender := backend.NewBufferedAppender(appendFile, walConfig.IndexDownsample, len(records))
	for {
		bytesID, bytesObject, err := iterator.Next()
		if err != nil {
			return nil, err
		}
		if bytesID == nil {
			break
		}

		orderedBlock.bloom.Add(farm.Fingerprint64(bytesID))
		err = appender.Append(bytesID, bytesObject)
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

func (h *headBlock) Find(id backend.ID) ([]byte, error) {
	records := h.appender.Records()
	file, err := h.file()
	if err != nil {
		return nil, err
	}

	finder := backend.NewFinder(records, file)

	return finder.Find(id)
}
