package v2

import (
	"context"
	"io"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/pkg/errors"
)

const DefaultFlushSizeBytes int = 30 * 1024 * 1024 // 30 MiB

func CreateBlock(ctx context.Context, cfg *common.BlockConfig, tenantID string, blockID uuid.UUID, encoding backend.Encoding,
	dataEncoding string, estimatedTotalObjects int, i common.TraceIterator, to backend.Writer) (*backend.BlockMeta, error) {
	defer i.Close()

	newMeta := backend.NewBlockMeta(tenantID, blockID, VersionString, encoding, dataEncoding)

	dec, err := model.NewSegmentDecoder(dataEncoding)
	if err != nil {
		return nil, err
	}

	newBlock, err := NewStreamingBlock(cfg, blockID, tenantID, []*backend.BlockMeta{newMeta}, estimatedTotalObjects)
	if err != nil {
		return nil, errors.Wrap(err, "error creating streaming block")
	}

	start := uint32(0)
	end := uint32(0)
	var tracker backend.AppendTracker
	for {
		id, tr, err := i.Next(ctx)
		if err != nil && err != io.EOF {
			return nil, errors.Wrap(err, "error iterating")
		}

		if id == nil {
			break
		}

		// Gather timestamps for individual traces
		// and also the whole block time range
		s, e := timestampsForTrace(tr)
		if s < start || start == 0 {
			start = s
		}
		if e > end {
			end = e
		}

		data, err := dec.PrepareForWrite(tr, s, e)
		if err != nil {
			return nil, errors.Wrap(err, "preparing for write")
		}

		data2, err := dec.ToObject([][]byte{data})
		if err != nil {
			return nil, errors.Wrap(err, "toobject")
		}

		err = newBlock.AddObject(id, data2)
		if err != nil {
			return nil, errors.Wrap(err, "error adding object to compactor block")
		}

		if newBlock.CurrentBufferLength() > DefaultFlushSizeBytes {
			tracker, _, err = newBlock.FlushBuffer(ctx, tracker, to)
			if err != nil {
				return nil, errors.Wrap(err, "error flushing compactor block")
			}
		}
	}

	newBlock.BlockMeta().StartTime = time.Unix(int64(start), 0)
	newBlock.BlockMeta().EndTime = time.Unix(int64(end), 0)

	_, err = newBlock.Complete(ctx, tracker, to)
	if err != nil {
		return nil, errors.Wrap(err, "error completing compactor block")
	}

	return newBlock.BlockMeta(), nil
}

func timestampsForTrace(t *tempopb.Trace) (uint32, uint32) {
	start := uint64(math.MaxUint64)
	end := uint64(0)

	for _, b := range t.Batches {
		for _, ils := range b.InstrumentationLibrarySpans {
			for _, s := range ils.Spans {
				if s.StartTimeUnixNano < start {
					start = s.StartTimeUnixNano
				}
				if s.EndTimeUnixNano > end {
					end = s.EndTimeUnixNano
				}
			}
		}
	}

	return uint32(start / uint64(time.Second)), uint32(end / uint64(time.Second))
}
