package friggdb

import (
	"encoding/json"
	"fmt"

	bloom "github.com/dgraph-io/ristretto/z"
	"github.com/dgryski/go-farm"
	"github.com/google/uuid"
	"github.com/grafana/frigg/friggdb/encoding"
	"github.com/grafana/frigg/friggdb/wal"
)

// compactor block wraps a headblock to facilitate compaction.  it primarily needs the headblock
//   append code and then provides helper methods to massage the
// feel like this is kind of not worth it.  if tests start failing probably a good idea to just
//   split this functionality entirely
type compactorBlock struct {
	h     wal.HeadBlock
	metas []*encoding.BlockMeta

	bloomFP         float64
	indexDownsample int
}

func newCompactorBlock(h wal.HeadBlock, bloomFP float64, indexDownsample int, metas []*encoding.BlockMeta) (*compactorBlock, error) {
	if h == nil {
		return nil, fmt.Errorf("headblock should not be nil")
	}

	if len(metas) == 0 {
		return nil, fmt.Errorf("empty block meta list")
	}

	if bloomFP <= 0.0 {
		return nil, fmt.Errorf("invalid bloomFP rate")
	}

	if indexDownsample <= 0 {
		return nil, fmt.Errorf("invalid index downsample rate")
	}

	return &compactorBlock{
		h:               h,
		bloomFP:         bloomFP,
		indexDownsample: indexDownsample,
		metas:           metas,
	}, nil
}

func (c *compactorBlock) write(id encoding.ID, object []byte) error {
	return c.h.Write(id, object)
}

func (c *compactorBlock) id() uuid.UUID {
	return c.h.BlockMeta().BlockID
}

func (c *compactorBlock) meta() ([]byte, error) {
	meta := c.h.BlockMeta()

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

	b := bloom.NewBloomFilter(float64(length), c.bloomFP)

	_, _, records, _ := c.h.WriteInfo()

	// add all ids
	for _, r := range records {
		b.Add(farm.Fingerprint64(r.ID))
	}

	return b.JSONMarshal(), nil
}

func (c *compactorBlock) index() ([]byte, error) {
	_, _, records, _ := c.h.WriteInfo()

	numRecords := (len(records) / c.indexDownsample) + 1
	downsampledRecords := make([]*encoding.Record, 0, numRecords)
	// downsample index and then marshal
	var currentRecord *encoding.Record
	for i, r := range records {
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
		if i%c.indexDownsample == c.indexDownsample-1 || i == len(records)-1 {
			currentRecord.ID = r.ID
			downsampledRecords = append(downsampledRecords, currentRecord)
			currentRecord = nil
		}
	}

	return encoding.MarshalRecords(downsampledRecords)
}

func (c *compactorBlock) length() int {
	return c.h.Length()
}

func (c *compactorBlock) objectFilePath() string {
	_, _, _, filepath := c.h.WriteInfo()
	return filepath
}

func (c *compactorBlock) clear() error {
	return c.h.Clear()
}
