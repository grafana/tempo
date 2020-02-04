package friggdb

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"

	bloom "github.com/dgraph-io/ristretto/z"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
)

const (
	uint32Size = 4
)

type IterFunc func(id ID, msg proto.Message) (bool, error)

// complete block has all of the fields
type completeBlock struct {
	meta     *blockMeta
	bloom    *bloom.Bloom
	filepath string
	records  []*Record

	readFile *os.File
}

type ReplayBlock interface {
	Iterator(read proto.Message, fn IterFunc) error
	Identity() (blockID uuid.UUID, tenantID string, records []*Record, filepath string) // jpe : No more identity!
	Clear() error
}

type CompleteBlock interface {
	ReplayBlock

	Find(id ID, out proto.Message) (bool, error)
	blockMeta() *blockMeta
	bloomFilter() *bloom.Bloom
}

// todo:  I hate this method.  Make it not exist
func (c *completeBlock) Identity() (uuid.UUID, string, []*Record, string) {
	return c.meta.BlockID, c.meta.TenantID, c.records, c.fullFilename()
}

func (c *completeBlock) Find(id ID, out proto.Message) (bool, error) {

	i := sort.Search(len(c.records), func(idx int) bool {
		return bytes.Compare(c.records[idx].ID, id) >= 0
	})

	if i < 0 || i >= len(c.records) {
		return false, nil
	}

	rec := c.records[i]

	b, err := c.readRecordBytes(rec)
	if err != nil {
		return false, err
	}

	found := false
	err = iterateObjects(bytes.NewReader(b), out, func(foundID ID, msg proto.Message) (bool, error) {
		if bytes.Equal(foundID, id) {
			found = true
			return false, nil
		}

		return true, nil

	})
	if err != nil {
		return false, err
	}

	return found, nil
}

func (c *completeBlock) Iterator(read proto.Message, fn IterFunc) error {
	name := c.fullFilename()
	f, err := os.OpenFile(name, os.O_RDONLY, 0644)
	defer f.Close()

	if err != nil {
		return err
	}

	return iterateObjects(f, read, fn)
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

func (c *completeBlock) blockMeta() *blockMeta {
	return c.meta
}

func (c *completeBlock) bloomFilter() *bloom.Bloom {
	return c.bloom
}

func (c *completeBlock) fullFilename() string {
	return fmt.Sprintf("%s/%v:%v", c.filepath, c.meta.BlockID, c.meta.TenantID)
}

func (c *completeBlock) readObject(r *Record, out proto.Message) (ID, error) {
	b, err := c.readRecordBytes(r)
	if err != nil {
		return nil, err
	}

	// only reads and returns the first object
	id, _, err := unmarshalObjectFromReader(out, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	return id, nil
}

func (c *completeBlock) readRecordBytes(r *Record) ([]byte, error) {
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

func iterateObjects(reader io.Reader, read proto.Message, fn IterFunc) error {
	for {
		id, more, err := unmarshalObjectFromReader(read, reader)
		if err != nil {
			return err
		}
		if !more {
			// there are no more objects in the reader
			break
		}

		more, err = fn(id, read)
		if err != nil {
			return err
		}
		if !more {
			// the calling code doesn't need any more objects
			break
		}
	}

	return nil
}
