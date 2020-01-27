package wal

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
)

type HeadBlock interface {
	Write(p proto.Message) (uint64, uint32, error)
	Read(start uint64, offset uint32, out proto.Message) error
	Clear() error
	Iterator(read proto.Message, fn IterFunc) error
	Identity() (uuid.UUID, string)
}

type headblock struct {
	appendFile *os.File
	readFile   *os.File

	filepath   string
	blockID    uuid.UUID
	instanceID string
}

func (h *headblock) Identity() (uuid.UUID, string) {
	return h.blockID, fullFilename(h.filepath, h.blockID, h.instanceID)
}

func (h *headblock) Write(p proto.Message) (uint64, uint32, error) {
	name := fullFilename(h.filepath, h.blockID, h.instanceID)
	var err error

	if h.appendFile == nil {
		f, err := os.OpenFile(name, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return 0, 0, err
		}
		h.appendFile = f
	}

	b, err := proto.Marshal(p)
	if err != nil {
		return 0, 0, err
	}

	err = binary.Write(h.appendFile, binary.LittleEndian, uint32(len(b)))
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

func (h *headblock) Read(start uint64, length uint32, out proto.Message) error {
	name := fullFilename(h.filepath, h.blockID, h.instanceID)
	var err error

	if h.readFile == nil {
		f, err := os.OpenFile(name, os.O_RDONLY, 0644)
		if err != nil {
			return err
		}
		h.readFile = f
	}

	b := make([]byte, length)
	_, err = h.readFile.ReadAt(b, int64(start))
	if err != nil {
		return err
	}

	err = proto.Unmarshal(b, out)
	if err != nil {
		return err
	}

	return nil
}

func (h *headblock) Clear() error {
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
