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

// returns all blocks in the configured file path.  blocks fields are not complete though and
//   are useful for wal replay only
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
				meta:     newBlockMeta(tenantID, blockID),
				filepath: w.c.filepath,
			},
		})
	}

	return blocks, nil
}

func (w *wal) NewBlock(id uuid.UUID, tenantID string) (HeadBlock, error) {
	h := &headBlock{
		completeBlock: completeBlock{
			meta:     newBlockMeta(tenantID, id),
			filepath: w.c.filepath,
		},
	}

	name := h.fullFilename()
	_, err := os.Create(name)
	if err != nil {
		return nil, err
	}

	return h, nil
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
