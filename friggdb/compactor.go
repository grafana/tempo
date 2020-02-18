package friggdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"

	"github.com/google/uuid"
	"github.com/grafana/frigg/friggdb/backend"
)

type compactorConfig struct {
	BlocksAtOnce int    `yaml:"blocks-per"`
	BytesAtOnce  uint32 `yaml:"bytes-per"`
	BlocksOut    int    `yaml:"blocks-out"`
}

type compactor struct {
	cfg    *compactorConfig
	walCfg *walConfig

	r backend.Reader
	w backend.Writer
}

func newCompactor(cfg *compactorConfig, walCfg *walConfig, r backend.Reader, w backend.Writer) *compactor {
	return &compactor{
		cfg:    cfg,
		r:      r,
		w:      w,
		walCfg: walCfg,
	}
}

func (c *compactor) blocksToCompact(tenantID string) []uuid.UUID {
	return nil
}

func (c *compactor) compact(ids []uuid.UUID, tenantID string) error {
	var err error
	bookmarks := make([]*bookmark, 0, len(ids))
	blockMetas := make([]*blockMeta, 0, len(ids))

	totalRecords := 0
	for _, id := range ids {
		index, err := c.r.Index(id, tenantID)
		if err != nil {
			return err
		}

		totalRecords += recordCount(index)
		bookmarks = append(bookmarks, &bookmark{
			id:    id,
			index: index,
		})

		metaBytes, err := c.r.BlockMeta(id, tenantID)
		if err != nil {
			return err
		}

		meta := &blockMeta{}
		err = json.Unmarshal(metaBytes, meta)
		if err != nil {
			return err
		}
		blockMetas = append(blockMetas, meta)
	}

	recordsPerBlock := (totalRecords / c.cfg.BlocksOut) + 1
	var currentBlock *compactorBlock

	for !allDone(bookmarks) {
		var lowestID []byte
		var lowestObject []byte

		// find lowest ID of the new object
		for _, b := range bookmarks {
			currentID, currentObject, err := c.currentObject(b, tenantID)
			if err == io.EOF {
				continue
			} else if err != nil {
				return err
			}

			// todo:  right now if we run into equal ids we take the larger object in the hopes that it's a more complete trace or something.
			//   in the future add a callback or something that allows the owning application to make a choice or attempt to combine the traces or something
			if bytes.Equal(currentID, lowestID) {
				if len(currentObject) > len(lowestObject) {
					lowestID = currentID
					lowestObject = currentObject
				}
			} else if len(lowestID) == 0 || bytes.Compare(currentID, lowestID) == -1 {
				lowestID = currentID
				lowestObject = currentObject
			}
		}

		if len(lowestID) == 0 || len(lowestObject) == 0 {
			return fmt.Errorf("failed to find a lowest object in compaction")
		}

		// make a new block if necessary
		if currentBlock == nil {
			currentBlock, err = newCompactorBlock(tenantID, c.walCfg, blockMetas)
			if err != nil {
				return err
			}
		}

		// write to new block
		err = currentBlock.write(lowestID, lowestObject)
		if err != nil {
			return err
		}

		// ship block to backend if done
		if currentBlock.length() >= recordsPerBlock {
			currentMeta, err := currentBlock.meta()
			if err != nil {
				return err
			}

			currentIndex, err := currentBlock.index()
			if err != nil {
				return err
			}

			currentBloom, err := currentBlock.bloom()
			if err != nil {
				return err
			}

			err = c.w.Write(context.TODO(), currentBlock.id(), tenantID, currentMeta, currentBloom, currentIndex, currentBlock.objectFilePath())
			if err != nil {
				return err
			}

			currentBlock.clear()
			if err != nil {
				// jpe: log?  return warning?
			}
			currentBlock = nil
		}
	}

	return nil
}

func (c *compactor) currentObject(b *bookmark, tenantID string) ([]byte, []byte, error) {
	if len(b.currentID) != 0 && len(b.currentObject) != 0 {
		return b.currentID, b.currentObject, nil
	}

	var err error
	b.currentID, b.currentObject, err = c.nextObject(b, tenantID)
	if err != nil {
		return nil, nil, err
	}

	return b.currentID, b.currentObject, nil
}

func (c *compactor) nextObject(b *bookmark, tenantID string) ([]byte, []byte, error) {
	var err error

	// if no objects, pull objects

	if len(b.objects) == 0 {
		// if no index left, EOF
		if len(b.index) == 0 {
			return nil, nil, io.EOF
		}

		// pull next n bytes into objects
		rec := &Record{}

		var start uint64
		var length uint32

		start = math.MaxUint64
		for length < c.cfg.BytesAtOnce {
			buff := b.index[:recordLength]
			marshalRecord(rec, buff)

			if start == math.MaxUint64 {
				start = rec.Start
			}
			length += rec.Length

			b.index = b.index[recordLength:]
		}

		b.objects, err = c.r.Object(b.id, tenantID, start, length)
		if err != nil {
			return nil, nil, err
		}
	}

	// attempt to get next object from objects
	objectReader := bytes.NewReader(b.objects)
	id, object, err := unmarshalObjectFromReader(objectReader)
	if err != nil {
		return nil, nil, err
	}

	// advance the objects buffer
	bytesRead := objectReader.Size() - int64(objectReader.Len())
	if bytesRead < 0 || bytesRead > int64(len(b.objects)) {
		return nil, nil, fmt.Errorf("bad object read during compaction")
	}
	b.objects = b.objects[bytesRead:]

	return id, object, nil
}

func allDone(bookmarks []*bookmark) bool {
	for _, b := range bookmarks {
		if !b.done() {
			return false
		}
	}

	return true
}
