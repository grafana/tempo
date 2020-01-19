package wal

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
)

type WAL interface {
	AllBlocks() ([]WALBlock, error)
	NewBlock(id uuid.UUID, instanceID string) (WALBlock, error)
}

type WALBlock interface {
	Write(p proto.Message) (int64, int32, error)
	Read(start int64, offset int32, out proto.Message) error
	Clear() error
}

type wal struct {
	c *Config
}

type walblock struct {
	filepath   string
	blockID    uuid.UUID
	instanceID string
}

func New(c *Config) WAL {
	return &wal{
		c: c,
	}
}

func (w *wal) AllBlocks() ([]WALBlock, error) {
	files, err := ioutil.ReadDir(fmt.Sprintf("%s", w.c.filepath))
	if err != nil {
		return nil, err
	}

	blocks := make([]WALBlock, 0, len(files))
	for _, f := range files {
		name := f.Name()
		blockID, instanceID, err := parseFilename(name)
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, &walblock{
			filepath:   w.c.filepath,
			blockID:    blockID,
			instanceID: instanceID,
		})
	}

	return blocks, nil
}

func (w *wal) NewBlock(id uuid.UUID, instanceID string) (WALBlock, error) {
	name := fullFilename(w.c.filepath, id, instanceID)

	_, err := os.Create(name)
	if err != nil {
		return nil, err
	}

	return &walblock{
		filepath:   w.c.filepath,
		blockID:    id,
		instanceID: instanceID,
	}, nil
}

func (w *walblock) Write(p proto.Message) (int64, int32, error) {
	name := fullFilename(w.filepath, w.blockID, w.instanceID)

	f, err := os.OpenFile(name, os.O_APPEND|os.O_WRONLY, 0644) // todo:  evaluate reopening on each write?
	defer f.Close()
	if err != nil {
		return 0, 0, err
	}

	b, err := proto.Marshal(p)
	if err != nil {
		return 0, 0, err
	}

	lengthBuffer := make([]byte, 4)
	binary.LittleEndian.PutUint32(lengthBuffer, uint32(len(b)))
	_, err = f.Write(lengthBuffer)
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

	return info.Size(), int32(length), nil
}

func (w *walblock) Read(start int64, length int32, out proto.Message) error {
	name := fullFilename(w.filepath, w.blockID, w.instanceID)

	f, err := os.OpenFile(name, os.O_RDONLY, 0644) // todo:  evaluate reopening on each write?
	defer f.Close()
	if err != nil {
		return err
	}

	b := make([]byte, length)
	_, err = f.ReadAt(b, start)
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
	name := fullFilename(w.filepath, w.blockID, w.instanceID)
	err := os.Remove(name)
	return err
}

func parseFilename(name string) (uuid.UUID, string, error) {
	i := strings.Index(name, ":")

	if i < 0 {
		return uuid.UUID{}, "", fmt.Errorf("unable to parse %s", name)
	}

	blockIDString := name[:i]
	instanceID := name[i+1:]

	blockID, err := uuid.Parse(blockIDString)
	if err != nil {
		return uuid.UUID{}, "", err
	}

	return blockID, instanceID, nil
}

func fullFilename(filepath string, blockID uuid.UUID, instanceID string) string {
	return fmt.Sprintf("%s/%v:%v", filepath, blockID, instanceID)
}
