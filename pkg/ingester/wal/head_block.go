package wal

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
)

type HeadBlock interface {
	Write(id ID, p proto.Message) error
	Find(id ID, out proto.Message) (bool, error)
	Clear() error // jpe ?
	Iterator(read proto.Message, fn IterFunc) error
	Identity() (uuid.UUID, string, []*Record, string) // jpe.  make cut block instead
	Length() int
}

type headblock struct {
	appendFile *os.File
	readFile   *os.File

	filepath   string
	blockID    uuid.UUID
	instanceID string
	records    []*Record
}

func (h *headblock) Identity() (uuid.UUID, string, []*Record, string) {
	return h.blockID, h.instanceID, h.records, fullFilename(h.filepath, h.blockID, h.instanceID)
}

func (h *headblock) Write(id ID, p proto.Message) error {
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

func (h *headblock) Find(id ID, out proto.Message) (bool, error) {

	i := sort.Search(len(h.records), func(idx int) bool {
		return bytes.Compare(h.records[idx].ID, id) >= 0
	})

	if i < 0 || i >= len(h.records) {
		return false, nil
	}

	rec := h.records[i]
	if bytes.Compare(rec.ID, id) != 0 {
		return false, nil
	}

	name := fullFilename(h.filepath, h.blockID, h.instanceID)
	if h.readFile == nil {
		f, err := os.OpenFile(name, os.O_RDONLY, 0644)
		if err != nil {
			return false, err
		}
		h.readFile = f
	}

	b := make([]byte, rec.Length)
	_, err := h.readFile.ReadAt(b, int64(rec.Start))
	if err != nil {
		return false, err
	}

	err = proto.Unmarshal(b, out)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (h *headblock) Clear() error { // jpe
	h.records = make([]*Record, 0) //todo : init this with some value?  max traces per block?

	var err error
	if h.appendFile != nil {
		err := h.appendFile.Close()
		if err != nil {
			return err
		}
	}
	if h.readFile != nil {
		err := h.readFile.Close()
		if err != nil {
			return err
		}
	}
	name := fullFilename(h.filepath, h.blockID, h.instanceID)
	err = os.Remove(name)
	return err
}

func (h *headblock) Iterator(read proto.Message, fn IterFunc) error {
	name := fullFilename(h.filepath, h.blockID, h.instanceID)
	f, err := os.OpenFile(name, os.O_RDONLY, 0644)
	defer f.Close()

	if err != nil {
		return err
	}

	for {
		var length uint32
		err := binary.Read(f, binary.LittleEndian, &length)
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		b := make([]byte, length)
		readLength, err := f.Read(b)
		if uint32(readLength) != length {
			return fmt.Errorf("read %d but expected %d", readLength, length)
		}

		err = proto.Unmarshal(b, read)
		if err != nil {
			return err
		}

		more, err := fn(read)
		if err != nil {
			return err
		}

		if !more {
			break
		}
	}

	return nil
}

func (h *headblock) Length() int {
	return len(h.records)
}
