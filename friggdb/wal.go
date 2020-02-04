package friggdb

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/google/uuid"
)

const (
	workDir = "work"
)

type WAL interface {
	AllBlocks() ([]ReplayBlock, error)
	NewBlock(id uuid.UUID, tenantID string) (HeadBlock, error)
	config() *walConfig
}

type wal struct {
	c *walConfig
}

type walConfig struct {
	filepath        string
	workFilepath    string
	indexDownsample int
	bloomFP         float64
}

func newWAL(c *walConfig) (WAL, error) {
	if c.filepath == "" {
		return nil, fmt.Errorf("please provide a path for the WAL")
	}

	if c.indexDownsample == 0 {
		return nil, fmt.Errorf("Non-zero index downsample required")
	}

	// make folder
	err := os.MkdirAll(c.filepath, os.ModePerm)
	if err != nil {
		return nil, err
	}

	if c.workFilepath == "" {
		workFilepath := path.Join(c.filepath, workDir)
		err = os.RemoveAll(workFilepath)
		if err != nil {
			return nil, err
		}
		err = os.MkdirAll(workFilepath, os.ModePerm)
		if err != nil {
			return nil, err
		}

		c.workFilepath = workFilepath
	}

	return &wal{
		c: c,
	}, nil
}

func (w *wal) AllBlocks() ([]ReplayBlock, error) {
	files, err := ioutil.ReadDir(fmt.Sprintf("%s", w.c.filepath))
	if err != nil {
		return nil, err
	}

	blocks := make([]ReplayBlock, 0, len(files))
	for _, f := range files {
		if f.IsDir() {
			continue
		}

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

func (w *wal) config() *walConfig {
	return w.c
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
