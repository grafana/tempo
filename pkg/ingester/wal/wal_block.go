package wal

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
)

type WALBlock interface {
	Write(p proto.Message) (uint64, uint32, error)
	Read(start uint64, offset uint32, out proto.Message) error
	Clear() error
	Iterator(read proto.Message, fn IterFunc) error
	Identity() (uuid.UUID, string)
}

type walblock struct {
	appendFile *os.File
	readFile   *os.File

	filepath   string
	blockID    uuid.UUID
	instanceID string
}

func (w *walblock) Identity() (uuid.UUID, string) {
	return w.blockID, fullFilename(w.filepath, w.blockID, w.instanceID)
}

func (w *walblock) Write(p proto.Message) (uint64, uint32, error) {
	name := fullFilename(w.filepath, w.blockID, w.instanceID)
	var err error

	if w.appendFile == nil {
		f, err := os.OpenFile(name, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return 0, 0, err
		}
		w.appendFile = f
	}

	b, err := proto.Marshal(p)
	if err != nil {
		return 0, 0, err
	}

	err = binary.Write(w.appendFile, binary.LittleEndian, uint32(len(b)))
	if err != nil {
		return 0, 0, err
	}

	info, err := w.appendFile.Stat()
	if err != nil {
		return 0, 0, err
	}

	length, err := w.appendFile.Write(b)
	if err != nil {
		return 0, 0, err
	}

	return uint64(info.Size()), uint32(length), nil
}

func (w *walblock) Read(start uint64, length uint32, out proto.Message) error {
	name := fullFilename(w.filepath, w.blockID, w.instanceID)
	var err error

	if w.readFile == nil {
		f, err := os.OpenFile(name, os.O_RDONLY, 0644)
		if err != nil {
			return err
		}
		w.readFile = f
	}

	b := make([]byte, length)
	_, err = w.readFile.ReadAt(b, int64(start))
	if err != nil {
		return err
	}

	err = proto.Unmarshal(b, out)
	if err != nil {
		return err
	}

	return nil
}

func (w *walblock) Clear() error {
	var err error
	if w.appendFile != nil {
		err := w.appendFile.Close()
		if err != nil {
			return err
		}
	}
	if w.readFile != nil {
		err := w.readFile.Close()
		if err != nil {
			return err
		}
	}
	name := fullFilename(w.filepath, w.blockID, w.instanceID)
	err = os.Remove(name)
	return err
}

func (w *walblock) Iterator(read proto.Message, fn IterFunc) error {
	name := fullFilename(w.filepath, w.blockID, w.instanceID)
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
