package v2

import (
	"context"
	"fmt"
	"io"

	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/pkg/errors"
)

const DefaultFlushSizeBytes int = 30 * 1024 * 1024 // 30 MiB

func CreateBlock(ctx context.Context, cfg *common.BlockConfig, meta *backend.BlockMeta, i common.Iterator, to backend.Writer) (*backend.BlockMeta, error) {
	// Default data encoding if needed
	if meta.DataEncoding == "" {
		meta.DataEncoding = model.CurrentEncoding
	}

	newBlock, err := NewStreamingBlock(cfg, meta.BlockID, meta.TenantID, []*backend.BlockMeta{meta}, meta.TotalObjects)
	if err != nil {
		return nil, errors.Wrap(err, "error creating streaming block")
	}

	bytesIterator, isBytesIterator := i.(BytesIterator)

	dec, err := model.NewSegmentDecoder(meta.DataEncoding)
	if err != nil {
		return nil, fmt.Errorf("error creating segment decoder: %w", err)
	}

	var next func(ctx context.Context) (common.ID, []byte, error)
	if isBytesIterator {
		// if this is one of our iterators we are in luck. this is quite fast
		next = bytesIterator.NextBytes
	} else {
		// otherwise we need to marshal the object to bytes
		next = func(ctx context.Context) (common.ID, []byte, error) {
			id, tr, err := i.Next(ctx)
			if err != nil || tr == nil {
				return nil, nil, err
			}

			// copy the id to escape the iterator
			id = append([]byte(nil), id...)

			traceBytes, err := dec.PrepareForWrite(tr, 0, 0) // start/end of the blockmeta are used
			if err != nil {
				return nil, nil, err
			}

			obj, err := dec.ToObject([][]byte{traceBytes})
			return id, obj, err
		}
	}

	var tracker backend.AppendTracker
	for {
		id, trBytes, err := next(ctx)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, errors.Wrap(err, "error iterating")
		}

		if id == nil {
			break
		}

		// This assumes the incoming bytes are the same data encoding.
		err = newBlock.AddObject(id, trBytes)
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

	_, err = newBlock.Complete(ctx, tracker, to)
	if err != nil {
		return nil, errors.Wrap(err, "error completing compactor block")
	}

	return newBlock.BlockMeta(), nil
}
