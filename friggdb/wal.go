package friggdb

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/google/uuid"
)

type WAL interface {
	AllBlocks() ([]HeadBlock, error)
	NewBlock(id uuid.UUID, tenantID string) (HeadBlock, error)
}

type wal struct {
	c *walConfig
}

type walConfig struct {
	filepath string
}

func newWAL(c *walConfig) (WAL, error) {
	if c.filepath == "" {
		return nil, fmt.Errorf("please provide a path for the WAL")
	}

	// make folder
	err := os.MkdirAll(c.filepath, os.ModePerm)
	if err != nil {
		return nil, err
	}

	return &wal{
		c: c,
	}, nil
}

func (w *wal) AllBlocks() ([]HeadBlock, error) {
	files, err := ioutil.ReadDir(fmt.Sprintf("%s", w.c.filepath))
	if err != nil {
		return nil, err
	}

	blocks := make([]HeadBlock, 0, len(files))
	for _, f := range files {
		name := f.Name()
		blockID, tenantID, err := parseFilename(name)
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, &headBlock{
			completeBlock: completeBlock{
				filepath:   w.c.filepath,
				blockID:    blockID,
				tenantID: tenantID,
			},
		})
	}

	return blocks, nil
}

func (w *wal) NewBlock(id uuid.UUID, tenantID string) (HeadBlock, error) {
	name := fullFilename(w.c.filepath, id, tenantID)

	_, err := os.Create(name)
	if err != nil {
		return nil, err
	}

	return &headBlock{
		completeBlock: completeBlock{
			filepath:   w.c.filepath,
			blockID:    id,
			tenantID: tenantID,
		},
	}, nil
}

func parseFilename(name string) (uuid.UUID, string, error) {
	i := strings.Index(name, ":")

	if i < 0 {
		return uuid.UUID{}, "", fmt.Errorf("unable to parse %s", name)
	}

	blockIDString := name[:i]
	tenantID := name[i+1:]

	blockID, err := uuid.Parse(blockIDString)
	if err != nil {
		return uuid.UUID{}, "", err
	}

	return blockID, tenantID, nil
}

func fullFilename(filepath string, blockID uuid.UUID, tenantID string) string {
	return fmt.Sprintf("%s/%v:%v", filepath, blockID, tenantID)
}
