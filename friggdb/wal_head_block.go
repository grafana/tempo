package friggdb

import (
	"bytes"
	"encoding/binary"
	"os"
	"sort"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
)

type HeadBlock interface {
	CompleteBlock

	Write(id ID, p proto.Message) error
	Complete(w WAL) (CompleteBlock, error)
}

type headBlock struct {
	completeBlock

	appendFile *os.File
}

func (h *headBlock) Write(id ID, p proto.Message) error {
	b, err := proto.Marshal(p)
	if err != nil {
		return err
	}

	start, length, err := h.appendObject(b)
	if err != nil {
		return err
	}

	// insert sorted to records
	i := sort.Search(len(h.records), func(idx int) bool {
		return bytes.Compare(h.records[idx].ID, id) == 1
	})
	h.records = append(h.records, nil)
	copy(h.records[i+1:], h.records[i:])
	h.records[i] = &Record{
		ID:     id,
		Start:  start,
		Length: length,
	}

	return nil
}

// jpe: add downsample granularity of records file here
func (h *headBlock) Complete(w WAL) (CompleteBlock, error) {
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
	orderedBlock := &headBlock{
		completeBlock: completeBlock{
			meta:     newBlockMeta(h.meta.TenantID, uuid.New()),
			filepath: walConfig.workFilepath,
			records:  make([]*Record, 0, len(h.records)),
		},
	}

	_, err := os.Create(orderedBlock.fullFilename())
	if err != nil {
		return nil, err
	}

	// records are already sorted
	for _, r := range h.records {
		b, err := h.readObject(r)
		if err != nil {
			return nil, err
		}

		start, length, err := orderedBlock.appendObject(b)
		if err != nil {
			return nil, err
		}

		orderedBlock.records = append(orderedBlock.records, &Record{
			ID:     r.ID,
			Start:  start,
			Length: length,
		})
	}

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

func (h *headBlock) appendObject(b []byte) (uint64, uint32, error) {
	if h.appendFile == nil {
		name := h.fullFilename()

		f, err := os.OpenFile(name, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return 0, 0, err
		}
		h.appendFile = f
	}

	info, err := h.appendFile.Stat()
	if err != nil {
		return 0, 0, err
	}

	err = binary.Write(h.appendFile, binary.LittleEndian, uint32(len(b)))
	if err != nil {
		return 0, 0, err
	}

	length, err := h.appendFile.Write(b)
	if err != nil {
		return 0, 0, err
	}

	return uint64(info.Size()), uint32(length) + 4, nil // 4 => uint32
}
