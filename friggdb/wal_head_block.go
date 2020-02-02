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
	b, err := proto.Marshal(p)
	if err != nil {
		return err
	}

	start, length, err := h.appendBytes(b)
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

// jpe:  move CompleteBlock method to the wal so we can access the folder and other config from the wal
//  add wal init method that creates "work" folder beneath wal folder.  use work folder for replay and this logic
//  also downsample granularity of records based on configurable value
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

	// jpe if about to return error clean up new file!

	return h, nil
}

func (h *headBlock) appendBytes(b []byte) (uint64, uint32, error) {
	if h.appendFile == nil {
		name := h.fullFilename()

		f, err := os.OpenFile(name, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return 0, 0, err
		}
		h.appendFile = f
	}

	err := binary.Write(h.appendFile, binary.LittleEndian, uint32(len(b)))
	if err != nil {
		return 0, 0, err
	}

	info, err := h.appendFile.Stat()
	if err != nil {
		return 0, 0, err
	}

	length, err := h.appendFile.Write(b)
	if err != nil {
		return 0, 0, err
	}

	return uint64(info.Size()), uint32(length), nil
}
