package wal

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/grafana/frigg/friggdb/encoding"

	bloom "github.com/dgraph-io/ristretto/z"
	"github.com/google/uuid"
)

// complete block has all of the fields
type completeBlock struct {
	meta        *encoding.BlockMeta
	bloom       *bloom.Bloom
	filepath    string
	records     []*encoding.Record
	timeWritten time.Time

	readFile *os.File
}

type ReplayBlock interface {
	Iterator(fn encoding.IterFunc) error
	TenantID() string
	Clear() error
}

type CompleteBlock interface {
	ReplayBlock

	Find(id encoding.ID) ([]byte, error)
	TimeWritten() time.Time

	BlockMeta() *encoding.BlockMeta
	BloomFilter() *bloom.Bloom
	BlockWroteSuccessfully(t time.Time)
	WriteInfo() (blockID uuid.UUID, tenantID string, records []*encoding.Record, filepath string) // todo:  i hate this method.  do something better.
}

func (c *completeBlock) TenantID() string {
	return c.meta.TenantID
}

func (c *completeBlock) WriteInfo() (uuid.UUID, string, []*encoding.Record, string) {
	return c.meta.BlockID, c.meta.TenantID, c.records, c.fullFilename()
}

func (c *completeBlock) Find(id encoding.ID) ([]byte, error) {

	i := sort.Search(len(c.records), func(idx int) bool {
		return bytes.Compare(c.records[idx].ID, id) >= 0
	})

	if i < 0 || i >= len(c.records) {
		return nil, nil
	}

	rec := c.records[i]

	b, err := c.readRecordBytes(rec)
	if err != nil {
		return nil, err
	}

	var foundObject []byte
	err = encoding.IterateObjects(bytes.NewReader(b), func(foundID encoding.ID, b []byte) (bool, error) {
		if bytes.Equal(foundID, id) {
			foundObject = b
			return false, nil
		}

		return true, nil

	})
	if err != nil {
		return nil, err
	}

	return foundObject, nil
}

func (c *completeBlock) Iterator(fn encoding.IterFunc) error {
	name := c.fullFilename()
	f, err := os.OpenFile(name, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	return encoding.IterateObjects(f, fn)
}

func (c *completeBlock) Clear() error {
	if c.readFile != nil {
		err := c.readFile.Close()
		if err != nil {
			return err
		}
	}

	name := c.fullFilename()
	return os.Remove(name)
}

func (c *completeBlock) TimeWritten() time.Time {
	return c.timeWritten
}

func (c *completeBlock) BlockWroteSuccessfully(t time.Time) {
	c.timeWritten = t
}

func (c *completeBlock) BlockMeta() *encoding.BlockMeta {
	return c.meta
}

func (c *completeBlock) BloomFilter() *bloom.Bloom {
	return c.bloom
}

func (c *completeBlock) fullFilename() string {
	return fmt.Sprintf("%s/%v:%v", c.filepath, c.meta.BlockID, c.meta.TenantID)
}

func (c *completeBlock) readRecordBytes(r *encoding.Record) ([]byte, error) {
	if c.readFile == nil {
		name := c.fullFilename()

		f, err := os.OpenFile(name, os.O_RDONLY, 0644)
		if err != nil {
			return nil, err
		}
		c.readFile = f
	}

	b := make([]byte, r.Length)
	_, err := c.readFile.ReadAt(b, int64(r.Start))
	if err != nil {
		return nil, err
	}

	return b, nil
}
