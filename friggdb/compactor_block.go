package friggdb

import (
	"encoding/json"
	"fmt"

	bloom "github.com/dgraph-io/ristretto/z"
	"github.com/dgryski/go-farm"
	"github.com/google/uuid"
	"github.com/grafana/frigg/friggdb/encoding"
)

// compactor block wraps a headblock to facilitate compaction.  it primarily needs the headblock
//   append code and then provides helper methods to massage the
// feel like this is kind of not worth it.  if tests start failing probably a good idea to just
//   split this functionality entirely
type compactorBlock struct {
	h     *headBlock
	metas []*encoding.BlockMeta

	cfg *walConfig // todo: these config elements are being used by more than the wal, change their location?
}

func newCompactorBlock(tenantID string, cfg *walConfig, metas []*encoding.BlockMeta) (*compactorBlock, error) {
	h, err := newBlock(uuid.New(), tenantID, cfg.workFilepath)
	if err != nil {
		return nil, err
	}

	if len(metas) == 0 {
		return nil, fmt.Errorf("empty block meta list")
	}

	return &compactorBlock{
		h:     h,
		cfg:   cfg,
		metas: metas,
	}, nil
}

func (c *compactorBlock) write(id encoding.ID, object []byte) error {
	return c.h.Write(id, object)
}

func (c *compactorBlock) id() uuid.UUID {
	return c.h.meta.BlockID
}

func (c *compactorBlock) meta() ([]byte, error) {
	meta := c.h.meta

	meta.StartTime = c.metas[0].StartTime
	meta.EndTime = c.metas[0].EndTime

	// everything should be correct here except the start/end times which we will get from the passed in metas
	for _, m := range c.metas[1:] {
		if m.StartTime.Before(meta.StartTime) {
			meta.StartTime = m.StartTime
		}
		if m.EndTime.After(meta.EndTime) {
			meta.EndTime = m.EndTime
		}
	}

	return json.Marshal(meta)
}

func (c *compactorBlock) bloom() ([]byte, error) {
	length := c.length()
	if length == 0 {
		return nil, fmt.Errorf("cannot create bloom without records")
	}

	b := bloom.NewBloomFilter(float64(length), c.cfg.bloomFP)

	// add all ids
	for _, r := range c.h.records {
		b.Add(farm.Fingerprint64(r.ID))
	}

	return b.JSONMarshal(), nil
}

func (c *compactorBlock) index() ([]byte, error) {
	numRecords := (len(c.h.records) / c.cfg.indexDownsample) + 1
	downsampledRecords := make([]*encoding.Record, 0, numRecords)
	// downsample index and then marshal
	var currentRecord *encoding.Record
	for i, r := range c.h.records {
		// start or continue working on a record
		if currentRecord == nil {
			currentRecord = &encoding.Record{
				ID:     r.ID,
				Start:  r.Start,
				Length: r.Length,
			}
		} else {
			currentRecord.Length += r.Length
		}

		// if this is the last record to be included by the downsample config OR is simply the last record
		if i%c.cfg.indexDownsample == c.cfg.indexDownsample-1 || i == len(c.h.records)-1 {
			currentRecord.ID = r.ID
			downsampledRecords = append(downsampledRecords, currentRecord)
			currentRecord = nil
		}
	}

	return encoding.MarshalRecords(downsampledRecords)
}

func (c *compactorBlock) length() int {
	return len(c.h.records)
}

func (c *compactorBlock) objectFilePath() string {
	return c.h.fullFilename()
}

func (c *compactorBlock) clear() error {
	return c.h.Clear()
}
