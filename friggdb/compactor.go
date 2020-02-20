package friggdb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"time"

	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/frigg/friggdb/backend"
	"github.com/grafana/frigg/friggdb/encoding"
	"github.com/grafana/frigg/friggdb/pool"
)

type compactorConfig struct {
	ChunkSizeBytes uint32 `yaml:"chunkSizeBytes"`
}

/*
metrics:
 - time to stop jobs
 - compaction time
 - compaction block count
 - blocks compacted due to retention
 - blocks deleted
*/

type compactor struct {
	c backend.Compactor

	jobStopper *pool.Stopper

	compactedBlockLists map[string][]*encoding.CompactedBlockMeta
}

const (
	inputBlocks  = 4
	outputBlocks = 2

	maxCompactionRange = 1 * time.Hour

	cursorDone = -1
)

func (rw *readerWriter) doCompaction() {
	// stop any existing compaction jobs
	if rw.jobStopper != nil {
		err := rw.jobStopper.Stop()
		if err != nil {
			level.Warn(rw.logger).Log("msg", "error during compaction cycle", "err", err)
		}
	}

	// start crazy jobs to do compaction with new list
	tenants := rw.blocklistTenants()

	var err error
	rw.jobStopper, err = rw.pool.RunStoppableJobs(tenants, func(payload interface{}, stopCh <-chan struct{}) error {
		var warning error
		tenantID := payload.(string)

		cursor := 0
	L:
		for {
			select {
			case <-stopCh:
				return warning
			default:
				var blocks []*encoding.BlockMeta
				blocks, cursor = rw.blocksToCompact(tenantID, cursor)
				if cursor == cursorDone {
					break L
				}
				if blocks == nil {
					continue
				}
				err := rw.compact(blocks, tenantID)
				if err != nil {
					warning = err
				}
			}
		}

		return warning
	})

	if err != nil {
		level.Error(rw.logger).Log("msg", "failed to start compaction.  compaction broken until next polling cycle.", "err", err)
	}
}

func (rw *readerWriter) blocksToCompact(tenantID string, cursor int) ([]*encoding.BlockMeta, int) {
	// loop through blocks starting at cursor for the given tenant, blocks are sorted by start date so candidates for compaction should be near each other
	//   - consider candidateBlocks at a time.
	//   - find the blocks with the fewest records that are within the compaction range
	rw.blockListsMtx.Lock() // jpe - lot of contention on this list.  think about it
	defer rw.blockListsMtx.Unlock()

	blocklist := rw.blockLists[tenantID]
	if inputBlocks > len(blocklist) {
		return nil, cursorDone
	}

	if cursor < 0 {
		return nil, cursorDone
	}

	cursorEnd := cursor + inputBlocks
	for {
		if cursorEnd >= len(blocklist) {
			break
		}

		blockStart := blocklist[cursor]
		blockEnd := blocklist[cursorEnd]

		if blockEnd.EndTime.Sub(blockStart.StartTime) < maxCompactionRange {
			return blocklist[cursor:cursorEnd], cursorEnd + 1
		}

		cursor++
		cursorEnd = cursor + inputBlocks
	}

	return nil, cursorDone
}

// jpe : this method is brittle and has weird failure conditions.  if at any point it fails it can't clean up the old blocks and just leaves them around
func (rw *readerWriter) compact(blockMetas []*encoding.BlockMeta, tenantID string) error {
	var err error
	bookmarks := make([]*bookmark, 0, len(blockMetas))

	var totalRecords uint32
	for _, blockMeta := range blockMetas {
		totalRecords += blockMeta.TotalObjects

		index, err := rw.r.Index(blockMeta.BlockID, tenantID)
		if err != nil {
			return err
		}

		bookmarks = append(bookmarks, &bookmark{
			id:    blockMeta.BlockID,
			index: index,
		})

		_, err = rw.r.BlockMeta(blockMeta.BlockID, tenantID)
		if os.IsNotExist(err) {
			// if meta doesn't exist right now it probably means this block was compacted.  warn and bail
			level.Warn(rw.logger).Log("msg", "unable to find meta during compaction", "blockID", blockMeta.BlockID, "tenantID", tenantID, "err", err)
			return nil
		} else if err != nil {
			return err
		}
	}

	recordsPerBlock := (totalRecords / outputBlocks) + 1
	var currentBlock *compactorBlock

	for !allDone(bookmarks) {
		var lowestID []byte
		var lowestObject []byte
		var lowestBookmark *bookmark

		// find lowest ID of the new object
		for _, b := range bookmarks {
			currentID, currentObject, err := currentObject(b, tenantID, rw.cfg.Compactor.ChunkSizeBytes, rw.r)
			if err == io.EOF {
				continue
			} else if err != nil {
				return err
			}

			// todo:  right now if we run into equal ids we take the larger object in the hopes that it's a more complete trace.
			//   in the future add a callback or something that allows the owning application to make a more intelligent choice
			//   such as combining traces if they're both incomplete
			if bytes.Equal(currentID, lowestID) {
				if len(currentObject) > len(lowestObject) {
					lowestID = currentID
					lowestObject = currentObject
					lowestBookmark = b
				}
			} else if len(lowestID) == 0 || bytes.Compare(currentID, lowestID) == -1 {
				lowestID = currentID
				lowestObject = currentObject
				lowestBookmark = b
			}
		}

		if len(lowestID) == 0 || len(lowestObject) == 0 || lowestBookmark == nil {
			return fmt.Errorf("failed to find a lowest object in compaction")
		}

		// make a new block if necessary
		if currentBlock == nil {
			h, err := rw.wal.NewWorkingBlock(uuid.New(), tenantID)
			if err != nil {
				return err
			}

			currentBlock, err = newCompactorBlock(h, rw.cfg.WAL.BloomFP, rw.cfg.WAL.IndexDownsample, blockMetas)
			if err != nil {
				return err
			}
		}

		// write to new block
		err = currentBlock.write(lowestID, lowestObject)
		if err != nil {
			return err
		}
		lowestBookmark.clearObject()

		// ship block to backend if done
		if uint32(currentBlock.length()) >= recordsPerBlock {
			currentMeta := currentBlock.meta()
			currentIndex, err := currentBlock.index()
			if err != nil {
				return err
			}

			currentBloom, err := currentBlock.bloom()
			if err != nil {
				return err
			}

			err = rw.w.Write(context.TODO(), currentBlock.id(), tenantID, currentMeta, currentBloom, currentIndex, currentBlock.objectFilePath())
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

	// mark old blocks compacted so they don't show up in polling
	for _, meta := range blockMetas {
		if err := rw.c.MarkBlockCompacted(meta.BlockID, tenantID); err != nil {
			// jpe: log
		}
	}

	return nil
}

func currentObject(b *bookmark, tenantID string, chunkSizeBytes uint32, r backend.Reader) ([]byte, []byte, error) {
	if len(b.currentID) != 0 && len(b.currentObject) != 0 {
		return b.currentID, b.currentObject, nil
	}

	var err error
	b.currentID, b.currentObject, err = nextObject(b, tenantID, chunkSizeBytes, r)
	if err != nil {
		return nil, nil, err
	}

	return b.currentID, b.currentObject, nil
}

func nextObject(b *bookmark, tenantID string, chunkSizeBytes uint32, r backend.Reader) ([]byte, []byte, error) {
	var err error

	// if no objects, pull objects

	if len(b.objects) == 0 {
		// if no index left, EOF
		if len(b.index) == 0 {
			return nil, nil, io.EOF
		}

		// pull next n bytes into objects
		var start uint64
		var length uint32

		start = math.MaxUint64
		for length < chunkSizeBytes && len(b.index) > 0 {
			var rec *encoding.Record
			rec, b.index = encoding.UnmarshalRecordAndAdvance(b.index)

			if start == math.MaxUint64 {
				start = rec.Start
			}
			length += rec.Length
		}

		b.objects, err = r.Object(b.id, tenantID, start, length)
		if err != nil {
			return nil, nil, err
		}
	}

	// attempt to get next object from objects
	objectReader := bytes.NewReader(b.objects)
	id, object, err := encoding.UnmarshalObjectFromReader(objectReader)
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
