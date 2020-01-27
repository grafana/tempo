package wal

import (
	"bytes"
	"encoding/binary"
	"os"
	"sort"

	"github.com/golang/protobuf/proto"
)

type HeadBlock interface {
	CompleteBlock

	Write(id ID, p proto.Message) error
	Complete() (CompleteBlock, error)
}

type headBlock struct {
	completeBlock

	appendFile *os.File
}

func (h *headBlock) Write(id ID, p proto.Message) error {
	name := fullFilename(h.filepath, h.blockID, h.instanceID)
	var err error

	if h.appendFile == nil {
		f, err := os.OpenFile(name, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		h.appendFile = f
	}

	b, err := proto.Marshal(p)
	if err != nil {
		return err
	}

	err = binary.Write(h.appendFile, binary.LittleEndian, uint32(len(b)))
	if err != nil {
		return err
	}

	info, err := h.appendFile.Stat()
	if err != nil {
		return err
	}

	length, err := h.appendFile.Write(b)
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
		Start:  uint64(info.Size()),
		Length: uint32(length),
	}

	return nil
}

func (h *headBlock) Complete() (CompleteBlock, error) {
	if h.appendFile != nil {
		err := h.appendFile.Close()
		if err != nil {
			return nil, err
		}
	}
	// todo: any other book-keeping?  sort wal file in trace id order?

	return h, nil
}
