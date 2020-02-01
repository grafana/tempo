package friggdb

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
	name := h.fullFilename()
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

	start, length, err := appendBinary(h.appendFile, b)
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

func (h *headBlock) Complete() (CompleteBlock, error) {
	if h.appendFile != nil {
		err := h.appendFile.Close()
		if err != nil {
			return nil, err
		}
	}

	// create a new block and write all objects to it in sorted order
	// todo: if the app crashes here then we'd have to wals with duplicate info and
	//   we'd replay both.  add a crc to the end of the file?

	// for each

	return h, nil
}

func appendBinary(f *os.File, b []byte) (uint64, uint32, error) {
	err := binary.Write(f, binary.LittleEndian, uint32(len(b)))
	if err != nil {
		return 0, 0, err
	}

	info, err := f.Stat()
	if err != nil {
		return 0, 0, err
	}

	length, err := f.Write(b)
	if err != nil {
		return 0, 0, err
	}

	return uint64(info.Size()), uint32(length), nil
}
